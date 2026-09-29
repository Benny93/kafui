package mainpage

import (
	"time"

	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
)

// topicCountsMaxAge is how long a known topic message count is shown before
// the list refresh fetches it again.
const topicCountsMaxAge = 30 * time.Second

// loadTopicMessageCounts returns a Cmd that fetches message counts for the
// topics currently visible on screen (current pagination page only).
// Fetching all topics at once would be extremely slow on large clusters
// (2000+ topics × N partitions = thousands of broker round-trips).
//
// Topics whose count is already known are skipped unless the known counts are
// older than topicCountsMaxAge, so the 5s list refresh does not re-query
// offsets for the whole page every tick. Only one request is in flight at a
// time: when it returns, applyTopicMessageCounts follows up with whatever the
// page still lacks.
func (k *KafuiContentProvider) loadTopicMessageCounts() tea.Cmd {
	return k.loadTopicMessageCountsExcept(nil)
}

// loadTopicMessageCountsExcept is loadTopicMessageCounts, leaving out the
// topics in skip (those the request that just finished already asked for, so
// a topic whose count failed to load is not retried in a tight loop).
func (k *KafuiContentProvider) loadTopicMessageCountsExcept(skip map[string]bool) tea.Cmd {
	k.syncCountsContext()
	if k.countsInFlight != nil {
		return nil
	}
	pageItems := k.pagination.GetCurrentPageItems(k.activeItems())
	refreshKnown := time.Since(k.countsFetchedAt) >= topicCountsMaxAge

	input := make(map[string]int32, len(pageItems))
	for _, item := range pageItems {
		// topicItemFrom unwraps the highlighted items a search produces too.
		tri, _, ok := topicItemFrom(item)
		if !ok || tri.partitions <= 0 || skip[tri.id] || (tri.messageCount >= 0 && !refreshKnown) {
			continue
		}
		input[tri.id] = tri.partitions
	}
	if len(input) == 0 {
		return nil
	}
	if refreshKnown {
		k.countsFetchedAt = time.Now()
	}

	shared.Log.Info("loadTopicMessageCounts: starting", "topics", len(input))
	k.countsInFlight = make(map[string]bool, len(input))
	for name := range input {
		k.countsInFlight[name] = true
	}
	k.countLoading = true
	ds := k.dataSource
	gen := k.countsGen
	return tea.Batch(k.countSpinner.Tick, func() tea.Msg {
		counts, err := ds.GetTopicMessageCounts(input)
		if err != nil {
			shared.Log.Error("loadTopicMessageCounts: GetTopicMessageCounts failed", "err", err)
			return TopicCountsLoadedMsg{Counts: map[string]int64{}, gen: gen}
		}
		shared.Log.Info("loadTopicMessageCounts: got counts", "topics", len(counts))
		return TopicCountsLoadedMsg{Counts: counts, gen: gen}
	})
}

// applyTopicMessageCounts updates messageCount on all TopicResourceItem entries
// and rebuilds the display rows so the Details column refreshes. It returns a
// follow-up fetch for topics that came onto the page while the request ran.
func (k *KafuiContentProvider) applyTopicMessageCounts(msg TopicCountsLoadedMsg) tea.Cmd {
	k.syncCountsContext()
	if msg.gen != k.countsGen {
		// Requested against a context that is no longer active: its counts
		// belong to another cluster's topics, and the in-flight marker has
		// already been cleared by the switch.
		shared.Log.Info("applyTopicMessageCounts: dropping stale result", "counts", len(msg.Counts))
		return nil
	}
	counts := msg.Counts
	shared.Log.Info("applyTopicMessageCounts", "counts", len(counts))
	updated := 0
	for _, item := range k.allItems {
		if tri, _, ok := topicItemFrom(item); ok {
			if count, found := counts[tri.id]; found {
				tri.messageCount = count
				updated++
			}
		}
	}
	shared.Log.Info("applyTopicMessageCounts: done", "updated", updated)
	requested := k.countsInFlight
	k.countsInFlight = nil
	k.countLoading = false
	k.countSpinnerFrame = ""
	// Rebuild rows so the spinner placeholder is replaced by the real count.
	if k.isFiltered {
		k.reapplyFilter()
	} else {
		k.updateTableForCurrentPage()
	}
	if !k.isTopicResource() {
		return nil
	}
	return k.loadTopicMessageCountsExcept(requested)
}

// syncCountsContext forgets the count state when the active context changed
// since it was recorded. Switches happen here and on the Clusters page, so the
// check reads the datasource instead of relying on one switch path: an old
// cluster's request must not block the new cluster's counts, and its result is
// dropped (gen mismatch) when it arrives.
func (k *KafuiContentProvider) syncCountsContext() {
	if k.dataSource == nil {
		return
	}
	if cur := k.dataSource.GetContext(); cur != k.countsCtx {
		k.resetCounts(cur)
	}
}

// resetCounts starts a fresh count state for context ctx.
func (k *KafuiContentProvider) resetCounts(ctx string) {
	k.countsCtx = ctx
	k.countsGen++
	k.countsInFlight = nil
	k.countsFetchedAt = time.Time{}
	k.countLoading = false
	k.countSpinnerFrame = ""
}
