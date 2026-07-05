package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDS is a minimal KafkaDataSource exposing only what the collector uses.
// It embeds api.KafkaDataSource so unused methods are nil (never called here).
type fakeDS struct {
	api.KafkaDataSource
	mu       sync.Mutex
	ctx      string
	topics   map[string]api.Topic
	counts   map[string]int64
	brokers  map[int32]api.BrokerStats
	topicErr error
}

func (f *fakeDS) GetContext() string { return f.ctx }
func (f *fakeDS) GetTopics() (map[string]api.Topic, error) {
	if f.topicErr != nil {
		return nil, f.topicErr
	}
	return f.topics, nil
}
func (f *fakeDS) GetTopicMessageCounts(_ map[string]int32) (map[string]int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]int64, len(f.counts))
	for k, v := range f.counts {
		out[k] = v
	}
	return out, nil
}
func (f *fakeDS) GetBrokerStats() (map[int32]api.BrokerStats, api.BrokerSummary, error) {
	return f.brokers, api.BrokerSummary{}, nil
}

func (f *fakeDS) setCount(topic string, n int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.counts[topic] = n
}

func newFake() *fakeDS {
	return &fakeDS{
		ctx:    "c1",
		topics: map[string]api.Topic{"orders": {NumPartitions: 3}, "events": {NumPartitions: 2}},
		counts: map[string]int64{"orders": 1000, "events": 500},
		brokers: map[int32]api.BrokerStats{
			1: {LeaderCount: 10, ReplicaCount: 30, SegmentSize: 1 << 20},
			2: {LeaderCount: 9, ReplicaCount: 28, SegmentSize: 2 << 20},
		},
	}
}

func TestDeriveRate(t *testing.T) {
	tests := []struct {
		name    string
		prev    int64
		cur     int64
		elapsed time.Duration
		hasPrev bool
		want    float64
	}{
		{"no prior sample", 0, 100, time.Second, false, api.RateUnknown},
		{"zero elapsed", 100, 200, 0, true, api.RateUnknown},
		{"negative delta (reset)", 200, 100, time.Second, true, api.RateUnknown},
		{"steady 100/s", 1000, 1100, time.Second, true, 100},
		{"half rate over 2s", 1000, 1100, 2 * time.Second, true, 50},
		{"flat", 500, 500, 5 * time.Second, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, deriveRate(tt.prev, tt.cur, tt.elapsed, tt.hasPrev))
		})
	}
}

func TestFirstCycleRatesUnknown(t *testing.T) {
	c := New(newFake(), time.Second, nil)
	c.CollectAll(context.Background())
	cm, ok := c.Active()
	require.True(t, ok)
	assert.Equal(t, api.RateUnknown, cm.MessagesInPerSec, "no rate on the first cycle")
	assert.Equal(t, int64(1500), cm.MessageCount)
	assert.Equal(t, 2, cm.TopicCount)
	assert.Equal(t, 5, cm.PartitionCount)
	assert.Equal(t, 2, cm.BrokerCount)
	// Byte rates are unknown without a configured endpoint.
	assert.Equal(t, api.RateUnknown, cm.BytesInPerSec)
	assert.Equal(t, api.RateUnknown, cm.BytesOutPerSec)
}

func TestRateFromDeltaAcrossCycles(t *testing.T) {
	f := newFake()
	c := New(f, time.Second, nil)

	now := time.Unix(0, 0)
	c.now = func() time.Time { return now }
	c.CollectAll(context.Background()) // seed baseline: total 1500

	// Advance 10s and add 1000 messages total (600 orders + 400 events).
	now = now.Add(10 * time.Second)
	f.setCount("orders", 1600)
	f.setCount("events", 900)
	c.CollectAll(context.Background())

	cm, _ := c.Active()
	assert.InDelta(t, 100.0, cm.MessagesInPerSec, 1e-9, "1000 msgs / 10s")
	// Per-topic rates.
	byTopic := map[string]float64{}
	for _, tm := range cm.Topics {
		byTopic[tm.Name] = tm.MessagesInPerSec
	}
	assert.InDelta(t, 60.0, byTopic["orders"], 1e-9)
	assert.InDelta(t, 40.0, byTopic["events"], 1e-9)
}

func TestHistoryAccumulatesAndSummary(t *testing.T) {
	f := newFake()
	c := New(f, time.Second, nil)
	now := time.Unix(0, 0)
	c.now = func() time.Time { return now }

	c.CollectAll(context.Background()) // baseline (no rate → not appended)
	for i := 1; i <= 3; i++ {
		now = now.Add(time.Second)
		f.setCount("orders", 1000+int64(100*i)) // +100/s
		c.CollectAll(context.Background())
	}
	ts := c.ActiveMessagesInHistory()
	require.Len(t, ts.Points, 3, "one point per cycle that produced a rate")
	s := ts.Summary()
	assert.True(t, s.OK)
	assert.InDelta(t, 100.0, s.Avg, 1e-9)
	assert.InDelta(t, 100.0, s.Min, 1e-9)
	assert.InDelta(t, 100.0, s.Max, 1e-9)
	// Ordering preserved (oldest first).
	for i := 1; i < len(ts.Points); i++ {
		assert.False(t, ts.Points[i].Time.Before(ts.Points[i-1].Time))
	}
}

func TestTopicErrorKeepsPreviousSnapshot(t *testing.T) {
	f := newFake()
	c := New(f, time.Second, nil)
	c.CollectAll(context.Background())
	before, _ := c.Active()

	f.topicErr = assertAnErr{}
	c.CollectAll(context.Background())
	after, _ := c.Active()
	assert.Equal(t, before.MessageCount, after.MessageCount, "failed fetch must not clobber cache")
}

type assertAnErr struct{}

func (assertAnErr) Error() string { return "boom" }

func TestReadsAreCacheOnlyAndRaceFree(t *testing.T) {
	f := newFake()
	c := New(f, time.Second, nil)
	c.CollectAll(context.Background())

	var wg sync.WaitGroup
	stop := make(chan struct{})
	// Concurrent collectors.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					c.CollectAll(context.Background())
				}
			}
		}()
	}
	// Concurrent readers.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_, _ = c.Active()
					_ = c.List()
					_ = c.ActiveMessagesInHistory()
				}
			}
		}()
	}
	time.Sleep(50 * time.Millisecond)
	close(stop)
	wg.Wait()
}

func TestByteRatesWithConfiguredEndpointStillUnknown(t *testing.T) {
	// Documented stub: even with an endpoint configured, byte-rate scraping is
	// not implemented, so rates remain unknown (never fabricated).
	c := New(newFake(), time.Second, func(string) string { return "http://exporter:9404/metrics" })
	c.CollectAll(context.Background())
	cm, _ := c.Active()
	assert.Equal(t, api.RateUnknown, cm.BytesInPerSec)
	assert.Equal(t, api.RateUnknown, cm.BytesOutPerSec)
}

func TestJMXWithoutBridgeDegradesGracefully(t *testing.T) {
	// Type JMX without a Jolokia URL: no error, byte rates unknown (empty sets).
	c := New(newFake(), time.Second, nil)
	c.SetSettingsResolver(func(string) appconfig.MetricsSettings {
		return appconfig.MetricsSettings{Type: appconfig.MetricsTypeJMX}
	})
	c.CollectAll(context.Background())
	cm, ok := c.Active()
	require.True(t, ok)
	assert.Equal(t, api.RateUnknown, cm.BytesInPerSec)
	assert.Equal(t, api.RateUnknown, cm.BytesOutPerSec)
}

func TestJMXViaJolokiaBridgeExtractsByteRates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":200,"value":{
			"kafka.server:name=BytesInPerSec,type=BrokerTopicMetrics":{"OneMinuteRate":100.0},
			"kafka.server:name=BytesOutPerSec,type=BrokerTopicMetrics":{"OneMinuteRate":50.0}
		}}`))
	}))
	defer srv.Close()

	c := New(newFake(), time.Second, nil)
	c.SetSettingsResolver(func(string) appconfig.MetricsSettings {
		return appconfig.MetricsSettings{Type: appconfig.MetricsTypeJMX, JolokiaURL: srv.URL}
	})
	c.CollectAll(context.Background())
	cm, _ := c.Active()
	assert.Equal(t, 100.0, cm.BytesInPerSec)
	assert.Equal(t, 50.0, cm.BytesOutPerSec)
}

func TestRingWrapAround(t *testing.T) {
	r := newRing(3)
	base := time.Unix(0, 0)
	for i := 0; i < 5; i++ {
		r.append(base.Add(time.Duration(i)*time.Second), float64(i))
	}
	ts := r.series()
	require.Len(t, ts.Points, 3)
	// Oldest retained is i=2, newest i=4.
	assert.Equal(t, 2.0, ts.Points[0].Value)
	assert.Equal(t, 3.0, ts.Points[1].Value)
	assert.Equal(t, 4.0, ts.Points[2].Value)
}

func TestSnapshotUnknownCluster(t *testing.T) {
	c := New(newFake(), time.Second, nil)
	_, ok := c.Snapshot("does-not-exist")
	assert.False(t, ok)
}
