// Package metrics provides a UI-independent background collector that samples
// the active cluster's message counts and broker stats on an interval, derives
// message-in rates from the count deltas between cycles, and caches the results
// plus a rolling time-series history so the metrics page can read them without
// blocking. It mirrors the design of pkg/cluster/collector.go.
//
// Collection is driven purely through the api.KafkaDataSource interface
// (GetTopics, GetTopicMessageCounts, GetBrokerStats), which operate on the
// active context — so the collector samples whichever cluster is currently
// selected. Byte rates require a configured metrics endpoint; that path is a
// documented stub and byte rates are otherwise api.RateUnknown.
package metrics

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/metrics/jolokia"
	"github.com/Benny93/kafui/pkg/ui/shared"
)

// perCollectTimeout bounds a single collection cycle so a slow/unreachable
// cluster can't stall the loop indefinitely.
const perCollectTimeout = 10 * time.Second

// Collector samples the active cluster and caches metrics + history. Safe for
// concurrent use: the background loop writes while the UI reads.
type Collector struct {
	ds         api.KafkaDataSource
	interval   time.Duration
	historyCap int
	// now is the clock seam (defaults to time.Now); tests inject a fake clock.
	now func() time.Time
	// endpointFor resolves a cluster's optional metrics endpoint for byte-rate
	// scraping (may be nil ⇒ no endpoint anywhere, byte rates unknown).
	endpointFor func(cluster string) string
	// settingsFor resolves a cluster's full metrics settings, enabling the JMX
	// (Jolokia bridge) scrape path (MM-17). Optional; nil ⇒ Prometheus-only.
	settingsFor func(cluster string) appconfig.MetricsSettings

	mu        sync.RWMutex
	cache     map[string]api.ClusterMetrics
	prev      map[string]sample
	history   map[string]*clusterHistory
	order     []string
	warnedJMX map[string]bool
}

// SetSettingsResolver installs a resolver for per-cluster metrics settings so
// the collector can honor Type "JMX" via the Jolokia bridge (MM-17). It is
// wired from the UI boot path; without it collection stays Prometheus-only.
func (c *Collector) SetSettingsResolver(fn func(cluster string) appconfig.MetricsSettings) {
	c.mu.Lock()
	c.settingsFor = fn
	c.mu.Unlock()
}

// sample is the previous cycle's raw counts, used to derive rates.
type sample struct {
	at       time.Time
	total    int64
	perTopic map[string]int64
}

// New builds a Collector. interval <= 0 falls back to the config default.
// endpointFor may be nil (no metrics endpoints configured anywhere).
func New(ds api.KafkaDataSource, interval time.Duration, endpointFor func(string) string) *Collector {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	c := &Collector{
		ds:          ds,
		interval:    interval,
		historyCap:  defaultHistoryCap,
		now:         time.Now,
		endpointFor: endpointFor,
		cache:       map[string]api.ClusterMetrics{},
		prev:        map[string]sample{},
		history:     map[string]*clusterHistory{},
		warnedJMX:   map[string]bool{},
	}
	c.seed()
	return c
}

// seed records the active cluster with an uncollected placeholder so the page
// can show a "collecting…" state before the first cycle completes.
func (c *Collector) seed() {
	if c.ds == nil {
		return
	}
	name := c.ds.GetContext()
	if name == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensureLocked(name)
}

// ensureLocked registers a cluster (order + empty cache/history) if unseen.
// Caller must hold c.mu.
func (c *Collector) ensureLocked(name string) {
	if _, ok := c.cache[name]; ok {
		return
	}
	c.cache[name] = api.ClusterMetrics{
		Cluster:          name,
		MessagesInPerSec: api.RateUnknown,
		BytesInPerSec:    api.RateUnknown,
		BytesOutPerSec:   api.RateUnknown,
	}
	c.history[name] = newClusterHistory(c.historyCap)
	c.order = append(c.order, name)
	sort.Strings(c.order)
}

// Interval returns the configured collection cadence.
func (c *Collector) Interval() time.Duration { return c.interval }

// CollectAll runs one collection cycle against the active cluster and updates
// the cache atomically. Fetch failures leave the previous snapshot in place.
func (c *Collector) CollectAll(ctx context.Context) {
	if c.ds == nil {
		return
	}
	cctx, cancel := context.WithTimeout(ctx, perCollectTimeout)
	defer cancel()
	c.collectActive(cctx)
}

// collectActive samples the active cluster. The context bounds any endpoint
// scraping; the current data-source calls are synchronous and best-effort.
func (c *Collector) collectActive(ctx context.Context) {
	name := c.ds.GetContext()
	if name == "" {
		return
	}

	topics, err := c.ds.GetTopics()
	if err != nil {
		return // keep the previous snapshot; try again next cycle
	}

	numParts := make(map[string]int32, len(topics))
	for tn, t := range topics {
		numParts[tn] = t.NumPartitions
	}
	counts, _ := c.ds.GetTopicMessageCounts(numParts) // best-effort; missing topics omitted
	brokerStats, _, _ := c.ds.GetBrokerStats()        // best-effort

	// Byte-rate scrape runs before the cache lock: it may do network I/O
	// (Jolokia bridge) and must not stall UI reads of the previous snapshot.
	bytesIn, bytesOut := c.scrapeByteRates(ctx, name)

	at := c.now()

	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensureLocked(name)

	prev, hasPrev := c.prev[name]

	var total, partitions int64
	topicMetrics := make([]api.TopicMetrics, 0, len(topics))
	curPerTopic := make(map[string]int64, len(topics))
	for tn, t := range topics {
		cnt := counts[tn]
		curPerTopic[tn] = cnt
		total += cnt
		partitions += int64(t.NumPartitions)
		rate := api.RateUnknown
		if hasPrev {
			rate = deriveRate(prev.perTopic[tn], cnt, at.Sub(prev.at), true)
		}
		topicMetrics = append(topicMetrics, api.TopicMetrics{
			Name:             tn,
			PartitionCount:   t.NumPartitions,
			MessageCount:     cnt,
			MessagesInPerSec: rate,
		})
	}
	sort.Slice(topicMetrics, func(i, j int) bool { return topicMetrics[i].Name < topicMetrics[j].Name })

	msgRate := deriveRate(prev.total, total, at.Sub(prev.at), hasPrev)

	brokers := make([]api.BrokerMetrics, 0, len(brokerStats))
	for id, bs := range brokerStats {
		brokers = append(brokers, api.BrokerMetrics{
			ID:             id,
			LeaderCount:    bs.LeaderCount,
			ReplicaCount:   bs.ReplicaCount,
			SegmentSize:    bs.SegmentSize,
			BytesInPerSec:  api.RateUnknown,
			BytesOutPerSec: api.RateUnknown,
		})
	}
	sort.Slice(brokers, func(i, j int) bool { return brokers[i].ID < brokers[j].ID })

	cm := api.ClusterMetrics{
		Cluster:          name,
		CollectedAt:      at,
		BrokerCount:      len(brokerStats),
		TopicCount:       len(topics),
		PartitionCount:   int(partitions),
		MessageCount:     total,
		MessagesInPerSec: msgRate,
		BytesInPerSec:    bytesIn,
		BytesOutPerSec:   bytesOut,
		Topics:           topicMetrics,
		Brokers:          brokers,
	}
	c.cache[name] = cm
	c.prev[name] = sample{at: at, total: total, perTopic: curPerTopic}

	h := c.history[name]
	if msgRate >= 0 {
		h.messagesIn.append(at, msgRate)
	}
	if bytesIn >= 0 {
		h.bytesIn.append(at, bytesIn)
	}
	if bytesOut >= 0 {
		h.bytesOut.append(at, bytesOut)
	}
}

func (c *Collector) endpoint(cluster string) string {
	if c.endpointFor == nil {
		return ""
	}
	return c.endpointFor(cluster)
}

// scrapeByteRates returns the cluster-wide bytes-in/out rates for a cluster,
// honoring the configured collection mechanism (MM-17):
//
//   - Type "JMX" with a Jolokia bridge URL: read kafka.server via the bridge and
//     derive byte rates from BrokerTopicMetrics {Bytes,BytesOut}PerSec.
//   - Type "JMX" without a bridge URL: native JMX/RMI is unsupported from Go, so
//     log one warning per cluster and report unknown rates (graceful degradation).
//   - Otherwise (Prometheus / unset): unknown rates — exposition-format byte-rate
//     scraping is not implemented; offset-delta message rates remain the primary
//     signal.
func (c *Collector) scrapeByteRates(ctx context.Context, cluster string) (bytesIn, bytesOut float64) {
	c.mu.RLock()
	resolver := c.settingsFor
	c.mu.RUnlock()
	if resolver == nil {
		return api.RateUnknown, api.RateUnknown
	}
	s := resolver(cluster)
	if s.Type != appconfig.MetricsTypeJMX {
		return api.RateUnknown, api.RateUnknown
	}
	if s.JolokiaURL == "" {
		c.warnJMXOnce(cluster)
		return api.RateUnknown, api.RateUnknown
	}
	samples, err := jolokia.New(s.JolokiaURL, s.Username, s.Password).Collect(ctx)
	if err != nil {
		shared.Log.Warn("jolokia metrics scrape failed", "cluster", cluster, "err", err)
		return api.RateUnknown, api.RateUnknown
	}
	return byteRateFromSamples(samples)
}

// warnJMXOnce logs a single degradation warning per cluster.
func (c *Collector) warnJMXOnce(cluster string) {
	c.mu.Lock()
	warned := c.warnedJMX[cluster]
	c.warnedJMX[cluster] = true
	c.mu.Unlock()
	if !warned {
		shared.Log.Warn("metrics Type=JMX without a Jolokia bridge URL: native JMX is unsupported; broker metrics will be empty for this cluster", "cluster", cluster)
	}
}

// byteRateFromSamples sums the one-minute BytesInPerSec/BytesOutPerSec rates
// across Jolokia samples. Absent metrics leave the rate unknown.
func byteRateFromSamples(samples []jolokia.Sample) (bytesIn, bytesOut float64) {
	bytesIn, bytesOut = api.RateUnknown, api.RateUnknown
	for _, s := range samples {
		switch {
		case strings.Contains(s.Name, "BytesInPerSec") && strings.HasSuffix(s.Name, "OneMinuteRate"):
			if bytesIn < 0 {
				bytesIn = 0
			}
			bytesIn += s.Value
		case strings.Contains(s.Name, "BytesOutPerSec") && strings.HasSuffix(s.Name, "OneMinuteRate"):
			if bytesOut < 0 {
				bytesOut = 0
			}
			bytesOut += s.Value
		}
	}
	return bytesIn, bytesOut
}

// deriveRate returns messages-per-second between two count samples. It yields
// api.RateUnknown when there is no prior sample, elapsed time is non-positive,
// or the count moved backwards (e.g. after retention/compaction reset).
func deriveRate(prevCount, curCount int64, elapsed time.Duration, hasPrev bool) float64 {
	if !hasPrev || elapsed <= 0 {
		return api.RateUnknown
	}
	d := curCount - prevCount
	if d < 0 {
		return api.RateUnknown
	}
	return float64(d) / elapsed.Seconds()
}

// --- cache-only readers (never trigger collection) ---

// Snapshot returns the cached metrics for a cluster. ok is false for an unknown
// cluster.
func (c *Collector) Snapshot(cluster string) (api.ClusterMetrics, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cm, ok := c.cache[cluster]
	return cm, ok
}

// Active returns the cached metrics for the currently-selected cluster.
func (c *Collector) Active() (api.ClusterMetrics, bool) {
	if c.ds == nil {
		return api.ClusterMetrics{}, false
	}
	return c.Snapshot(c.ds.GetContext())
}

// List returns cached metrics for all seen clusters in stable order.
func (c *Collector) List() []api.ClusterMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]api.ClusterMetrics, 0, len(c.order))
	for _, n := range c.order {
		if cm, ok := c.cache[n]; ok {
			out = append(out, cm)
		}
	}
	return out
}

// MessagesInHistory returns the retained message-in-rate series for a cluster.
func (c *Collector) MessagesInHistory(cluster string) api.TimeSeries {
	c.mu.RLock()
	h := c.history[cluster]
	c.mu.RUnlock()
	if h == nil {
		return api.TimeSeries{}
	}
	return h.messagesIn.series()
}

// ActiveMessagesInHistory returns the message-in-rate series for the active cluster.
func (c *Collector) ActiveMessagesInHistory() api.TimeSeries {
	if c.ds == nil {
		return api.TimeSeries{}
	}
	return c.MessagesInHistory(c.ds.GetContext())
}
