package mainpage

import (
	"sort"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countSpyDS records every GetTopicMessageCounts request.
type countSpyDS struct {
	*mock.KafkaDataSourceMock
	countCalls [][]string
}

func (s *countSpyDS) GetTopicMessageCounts(topics map[string]int32) (map[string]int64, error) {
	names := make([]string, 0, len(topics))
	for n := range topics {
		names = append(names, n)
	}
	sort.Strings(names)
	s.countCalls = append(s.countCalls, names)
	out := make(map[string]int64, len(topics))
	for n := range topics {
		out[n] = 7
	}
	return out, nil
}

// runCountCmd runs cmd, descending into batches, and returns the counts
// message the fetch produced. Timer commands (the spinner tick) are abandoned
// after a short wait rather than awaited.
func runCountCmd(t *testing.T, cmd tea.Cmd) TopicCountsLoadedMsg {
	t.Helper()
	require.NotNil(t, cmd)
	if counts, ok := findCounts(cmd); ok {
		return counts
	}
	t.Fatal("no TopicCountsLoadedMsg produced")
	return TopicCountsLoadedMsg{}
}

func findCounts(cmd tea.Cmd) (TopicCountsLoadedMsg, bool) {
	if cmd == nil {
		return TopicCountsLoadedMsg{}, false
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-done:
	case <-time.After(100 * time.Millisecond):
		return TopicCountsLoadedMsg{}, false
	}
	switch m := msg.(type) {
	case TopicCountsLoadedMsg:
		return m, true
	case tea.BatchMsg:
		for _, c := range m {
			if counts, ok := findCounts(c); ok {
				return counts, true
			}
		}
	}
	return TopicCountsLoadedMsg{}, false
}

func newCountProvider(names ...string) (*KafuiContentProvider, *countSpyDS) {
	spy := &countSpyDS{KafkaDataSourceMock: newMockDS()}
	k := NewKafuiContentProvider(spy)
	for _, n := range names {
		k.allItems = append(k.allItems, topicRLI(&TopicResourceItem{id: n, partitions: 2, messageCount: -1, outOfSync: -1, size: -1}))
	}
	k.pagination.SetTotalItems(len(k.allItems))
	return k, spy
}

// PERF-4: a count request in flight blocks a second one, and a count that is
// already known is not fetched again on the next refresh.
func TestLoadTopicMessageCounts_InFlightGuardAndKnownCounts(t *testing.T) {
	k, spy := newCountProvider("a", "b")

	cmd := k.loadTopicMessageCounts()
	require.NotNil(t, cmd)
	assert.Nil(t, k.loadTopicMessageCounts(), "no second request while one is in flight")

	counts := runCountCmd(t, cmd)
	assert.Nil(t, k.applyTopicMessageCounts(counts), "nothing left to fetch")
	require.Len(t, spy.countCalls, 1)
	assert.Equal(t, []string{"a", "b"}, spy.countCalls[0])

	// A timer refresh inside topicCountsMaxAge leaves known counts alone.
	assert.Nil(t, k.loadTopicMessageCounts())

	// Once they are stale (or after an explicit refresh) they are re-fetched.
	k.countsFetchedAt = time.Now().Add(-topicCountsMaxAge)
	assert.NotNil(t, k.loadTopicMessageCounts())
}

// PERF-4: topics that came onto the page while a request ran are fetched as
// soon as it returns, but the ones it asked for are not retried in a loop.
func TestApplyTopicMessageCounts_FollowsUpForNewPageItems(t *testing.T) {
	k, spy := newCountProvider("a", "b")
	cmd := k.loadTopicMessageCounts()

	// A search narrows the page to "b" and a topic without a count yet.
	k.allItems = append(k.allItems, topicRLI(&TopicResourceItem{id: "c", partitions: 1, messageCount: -1}))
	k.pagination.SetTotalItems(len(k.allItems))

	counts := runCountCmd(t, cmd)
	delete(counts.Counts, "b") // "b" failed to load; it must not be retried at once
	follow := k.applyTopicMessageCounts(counts)
	runCountCmd(t, follow)
	require.Len(t, spy.countCalls, 2)
	assert.Equal(t, []string{"c"}, spy.countCalls[1])
}

// PERF-7: search results are highlighted wrappers; they still get counts, and
// typing schedules the page enrichment once the query settles.
func TestSearch_LoadsCountsForMatches(t *testing.T) {
	k, spy := newCountProvider("orders", "payments")
	k.searchMode = true

	cmd := k.HandleContentUpdate(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ord")})
	require.NotNil(t, cmd, "typing schedules the page enrichment")
	require.True(t, k.isFiltered)

	assert.Nil(t, k.HandleContentUpdate(searchSettledMsg{query: "stale"}), "a stale tick does nothing")

	counts := runCountCmd(t, k.HandleContentUpdate(searchSettledMsg{query: "ord"}))
	assert.Contains(t, counts.Counts, "orders")
	assert.Equal(t, []string{"orders"}, spy.countCalls[len(spy.countCalls)-1])
}

// PERF-8: the timer refresh keeps the user's topic sort instead of snapping
// back to name order.
func TestHandleResourceList_KeepsTopicSort(t *testing.T) {
	k := NewKafuiContentProvider(newMockDS())
	k.topicSortCol = 1 // partitions
	k.topicSortDesc = true

	list := CurrentResourceListMsg{ResourceType: TopicResourceType, Items: []interface{}{
		topicRLI(&TopicResourceItem{id: "a", partitions: 1}),
		topicRLI(&TopicResourceItem{id: "b", partitions: 9}),
		topicRLI(&TopicResourceItem{id: "c", partitions: 5}),
	}}
	k.handleResourceList(list)

	var got []string
	for _, it := range k.allItems {
		got = append(got, k.getItemID(it))
	}
	assert.Equal(t, []string{"b", "c", "a"}, got)
}
