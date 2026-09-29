package mainpage

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// main-1: a reload after a change to the topics (purge, recreate, batch
// action, returning to the page) re-fetches the counts already known, instead
// of keeping the old ones for up to topicCountsMaxAge.
func TestTopicMutationReload_RefetchesKnownCounts(t *testing.T) {
	tests := []struct {
		name   string
		reload func(k *KafuiContentProvider) tea.Cmd
	}{
		{"purge", func(k *KafuiContentProvider) tea.Cmd { return k.HandleContentUpdate(topicPurgedMsg{name: "a"}) }},
		{"recreate", func(k *KafuiContentProvider) tea.Cmd { return k.HandleContentUpdate(topicRecreatedMsg{name: "a"}) }},
		{"batch purge", func(k *KafuiContentProvider) tea.Cmd {
			return k.HandleContentUpdate(topicBatchResultMsg{action: "purge", total: 2})
		}},
		{"batch purge with failures", func(k *KafuiContentProvider) tea.Cmd {
			return k.HandleContentUpdate(topicBatchResultMsg{action: "purge", total: 2, failures: []string{"b: boom"}})
		}},
		{"return to page", func(k *KafuiContentProvider) tea.Cmd { return k.InitContent() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, spy := newCountProvider("a", "b")
			counts := runCountCmd(t, k.loadTopicMessageCounts())
			require.Nil(t, k.applyTopicMessageCounts(counts))
			require.Nil(t, k.loadTopicMessageCounts(), "known counts are fresh")

			require.NotNil(t, tt.reload(k))

			assert.True(t, k.countsFetchedAt.IsZero(), "reload must mark the known counts stale")
			runCountCmd(t, k.loadTopicMessageCounts())
			require.Len(t, spy.countCalls, 2)
			assert.Equal(t, []string{"a", "b"}, spy.countCalls[1])
		})
	}
}

// main-1: the plain timer refresh still leaves fresh known counts alone.
func TestTimerTick_KeepsFreshKnownCounts(t *testing.T) {
	k, _ := newCountProvider("a")
	require.Nil(t, k.applyTopicMessageCounts(runCountCmd(t, k.loadTopicMessageCounts())))
	fetchedAt := k.countsFetchedAt

	k.HandleContentUpdate(TimerTickMsg(time.Now()))

	assert.Equal(t, fetchedAt, k.countsFetchedAt)
}

// main-2: a count request still running against the previous cluster neither
// blocks the new cluster's counts nor writes its results onto them.
func TestSelectContext_DropsInFlightCountsOfOldCluster(t *testing.T) {
	k, spy := newCountProvider("orders", "payments")
	oldCmd := k.loadTopicMessageCounts()
	require.NotNil(t, oldCmd)

	require.NotNil(t, k.HandleContentUpdate(SelectContextMsg{ContextName: "no-such-context"}))
	assert.Nil(t, k.countsInFlight, "switch clears the old in-flight marker")
	assert.True(t, k.countsFetchedAt.IsZero())

	// The new cluster's topic list arrives; one topic shares a name.
	k.allItems = []interface{}{
		topicRLI(&TopicResourceItem{id: "orders", partitions: 1, messageCount: -1}),
		topicRLI(&TopicResourceItem{id: "audit", partitions: 1, messageCount: -1}),
	}
	k.reapplyFilter()
	newCmd := k.loadTopicMessageCounts()
	require.NotNil(t, newCmd, "old request must not block the new cluster's counts")

	// The old cluster's result arrives late: it is dropped.
	stale := runCountCmd(t, oldCmd)
	stale.Counts["orders"] = 999
	assert.Nil(t, k.applyTopicMessageCounts(stale))
	tri, _, ok := topicItemFrom(k.allItems[0])
	require.True(t, ok)
	assert.Equal(t, int64(-1), tri.messageCount, "old cluster's count must not be applied")
	assert.NotNil(t, k.countsInFlight, "new cluster's request is still in flight")

	// The new cluster's own result is applied.
	assert.Nil(t, k.applyTopicMessageCounts(runCountCmd(t, newCmd)))
	assert.Equal(t, int64(7), tri.messageCount)
	require.Len(t, spy.countCalls, 2)
	assert.Equal(t, []string{"audit", "orders"}, spy.countCalls[1])
}

// A failed context switch keeps the current count state untouched.
func TestSelectContext_FailureKeepsCountState(t *testing.T) {
	ds := &mockKafkaDataSource{}
	ds.On("SetContext", "bad").Return(errors.New("unreachable"))
	k := newTestContentProvider(ds)
	k.countsInFlight = map[string]bool{"a": true}
	gen := k.countsGen

	require.NotNil(t, k.HandleContentUpdate(SelectContextMsg{ContextName: "bad"}))
	assert.Equal(t, gen, k.countsGen)
	assert.NotNil(t, k.countsInFlight)
}

// A switch made outside the main page (the Clusters page calls SetContext
// directly) must also drop the old cluster's in-flight count request.
func TestCountsDroppedWhenContextChangesElsewhere(t *testing.T) {
	k, spy := newCountProvider("orders")
	orig := spy.GetContext()
	defer func() { _ = spy.SetContext(orig) }()

	oldCmd := k.loadTopicMessageCounts()
	require.NotNil(t, oldCmd)

	other := "kafka-prod"
	if orig == other {
		other = "kafka-dev"
	}
	require.NoError(t, spy.SetContext(other))
	require.Equal(t, other, spy.GetContext())

	newCmd := k.loadTopicMessageCounts()
	require.NotNil(t, newCmd, "old cluster's request must not block the new one")

	stale := runCountCmd(t, oldCmd)
	stale.Counts["orders"] = 999
	k.applyTopicMessageCounts(stale)
	tri, _, ok := topicItemFrom(k.allItems[0])
	require.True(t, ok)
	assert.Equal(t, int64(-1), tri.messageCount, "old cluster's count must not be applied")
}
