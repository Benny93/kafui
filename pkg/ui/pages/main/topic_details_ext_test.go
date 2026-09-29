package mainpage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingDS counts the calls the topics list makes, so the test can assert
// that a page costs a fixed number of requests rather than one per topic.
type countingDS struct {
	api.KafkaDataSource
	sizes  map[string]int64
	health map[string]api.TopicHealth

	sizeCalls   int
	healthCalls int
	detailCalls int
}

func (c *countingDS) GetTopicSizes(names []string) (map[string]int64, error) {
	c.sizeCalls++
	return c.sizes, nil
}

func (c *countingDS) GetTopicHealth(names []string) (map[string]api.TopicHealth, error) {
	c.healthCalls++
	return c.health, nil
}

func (c *countingDS) GetTopicDetails(name string) (api.TopicDetails, error) {
	c.detailCalls++
	return api.TopicDetails{}, nil
}

// seedTopics puts unloaded topic rows on the current page.
func seedTopics(k *KafuiContentProvider, names ...string) {
	items := make([]interface{}, 0, len(names))
	for _, n := range names {
		items = append(items, shared.ResourceListItem{ResourceItem: &TopicResourceItem{
			id: n, outOfSync: -1, size: -1,
		}})
	}
	k.allItems = items
	k.pagination.SetTotalItems(len(items))
}

// Regression for the N+1 that made the topics list appear to hang: the page
// used to call GetTopicDetails once per topic, each opening its own cluster
// admin and its own client and then fetching offsets per partition. Two batched
// calls now cover the whole page.
func TestTopicDetailsExtUsesTwoBatchedCalls(t *testing.T) {
	ds := &countingDS{
		sizes:  map[string]int64{"a": 100, "b": 200},
		health: map[string]api.TopicHealth{"a": {OutOfSyncReplicas: 2}, "b": {}},
	}
	k := NewKafuiContentProvider(ds)
	seedTopics(k, "a", "b", "c")

	cmd := k.loadTopicDetailsExt()
	if cmd == nil {
		t.Fatal("expected a load command")
	}
	msg, ok := cmd().(TopicDetailsExtLoadedMsg)
	if !ok {
		t.Fatalf("unexpected message %T", cmd())
	}

	if ds.sizeCalls != 1 || ds.healthCalls != 1 {
		t.Errorf("expected one batched call each, got sizes=%d health=%d", ds.sizeCalls, ds.healthCalls)
	}
	if ds.detailCalls != 0 {
		t.Errorf("the list must not call GetTopicDetails per topic; got %d calls", ds.detailCalls)
	}

	if got := msg["a"].outOfSync; got != 2 {
		t.Errorf("a: OSR = %d, want 2", got)
	}
	if got := msg["b"].outOfSync; got != 0 {
		t.Errorf("b: a healthy topic must report 0, not unknown; got %d", got)
	}
	// "c" is absent from both responses: unknown, which renders as N/A rather
	// than a confident zero.
	if got := msg["c"].outOfSync; got != -1 {
		t.Errorf("c: unknown OSR must stay -1, got %d", got)
	}
	if got := msg["c"].size; got != -1 {
		t.Errorf("c: unknown size must stay -1 (N/A), got %d", got)
	}
}

// exportDS extends countingDS with the calls the CSV export makes.
type exportDS struct {
	countingDS
	countCalls  int
	topicsCalls int
}

func (e *exportDS) GetContext() string { return "test" }

func (e *exportDS) GetTopicMessageCounts(topics map[string]int32) (map[string]int64, error) {
	e.countCalls++
	out := make(map[string]int64, len(topics))
	for n, p := range topics {
		out[n] = int64(p) * 10
	}
	return out, nil
}

func (e *exportDS) GetTopics() (map[string]api.Topic, error) {
	e.topicsCalls++
	return map[string]api.Topic{"c": {NumPartitions: 4, ReplicationFactor: 3}}, nil
}

// PERF-10: the CSV export costs a fixed number of batched calls, not one
// GetTopicDetails (admin + client + per-partition offsets) per topic.
func TestExportTopicsCSVUsesBatchedCalls(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(wd) })

	ds := &exportDS{countingDS: countingDS{
		sizes:  map[string]int64{"a": 100},
		health: map[string]api.TopicHealth{"a": {UnderReplicatedPartitions: 1}},
	}}
	k := NewKafuiContentProvider(ds)
	k.allItems = []interface{}{
		shared.ResourceListItem{ResourceItem: &TopicResourceItem{id: "a", partitions: 2, replicationFactor: 1, messageCount: -1}},
		shared.ResourceListItem{ResourceItem: &TopicResourceItem{id: "b", partitions: 1, replicationFactor: 1, messageCount: -1}},
		// Not loaded yet: filled from the one GetTopics call.
		shared.ResourceListItem{ResourceItem: &TopicResourceItem{id: "c", partitions: -1, replicationFactor: -1, messageCount: -1}},
	}

	cmd := k.exportTopicsCSV()
	require.NotNil(t, cmd)
	_, ok := cmd().(core.NotificationMsg)
	require.True(t, ok)

	assert.Equal(t, 0, ds.detailCalls, "no per-topic GetTopicDetails")
	assert.Equal(t, 1, ds.sizeCalls)
	assert.Equal(t, 1, ds.healthCalls)
	assert.Equal(t, 1, ds.countCalls)
	assert.Equal(t, 1, ds.topicsCalls)

	files, err := filepath.Glob(filepath.Join(dir, "kafui-topics-*.csv"))
	require.NoError(t, err)
	require.Len(t, files, 1)
	data, err := os.ReadFile(files[0])
	require.NoError(t, err)
	assert.Contains(t, string(data), "a,2,1,20,1,")
	assert.Contains(t, string(data), "c,4,3,40,0,N/A")
}
