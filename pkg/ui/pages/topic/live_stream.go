package topic

import (
	"context"
	"fmt"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	tea "github.com/charmbracelet/bubbletea"
)

// consumeFlagsForMode returns the appropriate ConsumeFlags for the given mode.
func consumeFlagsForMode(mode ConsumeMode) api.ConsumeFlags {
	switch mode {
	case ModeNewest:
		return api.ConsumeFlags{
			Follow:     false,
			Tail:       60,
			OffsetFlag: "latest",
		}
	case ModeOldest:
		return api.ConsumeFlags{
			Follow:     false,
			Tail:       0,
			OffsetFlag: "oldest",
		}
	case ModeLive:
		return api.ConsumeFlags{
			Follow:     true,
			Tail:       0,
			OffsetFlag: "latest",
		}
	}
	return api.DefaultConsumeFlags()
}

const batchSize int64 = 60 // messages per fetch batch

// minLoadedOffset returns the lowest offset across all loaded messages, or -1 if none.
func (m *Model) minLoadedOffset() int64 {
	if len(m.messages) == 0 {
		return -1
	}
	min := m.messages[0].Offset
	for _, msg := range m.messages[1:] {
		if msg.Offset < min {
			min = msg.Offset
		}
	}
	return min
}

// maxLoadedOffset returns the highest offset across all loaded messages, or -1 if none.
func (m *Model) maxLoadedOffset() int64 {
	if len(m.messages) == 0 {
		return -1
	}
	max := m.messages[0].Offset
	for _, msg := range m.messages[1:] {
		if msg.Offset > max {
			max = msg.Offset
		}
	}
	return max
}

// nextBatchFlags returns ConsumeFlags that fetch the next batch beyond the
// currently loaded messages, in the direction appropriate for the current mode.
// Returns nil when there is nowhere to go (already at the start of the topic).
func (m *Model) nextBatchFlags() *api.ConsumeFlags {
	switch m.consumeMode {
	case ModeNewest:
		// Go to older messages: start batchSize offsets before the current minimum.
		min := m.minLoadedOffset()
		if min <= 0 {
			return nil // already at the beginning
		}
		start := min - batchSize
		if start < 0 {
			start = 0
		}
		f := api.ConsumeFlags{
			Follow:        false,
			Tail:          0, // must be 0 so OffsetFlag numeric value is used
			OffsetFlag:    fmt.Sprintf("%d", start),
			LimitMessages: batchSize,
		}
		return &f
	case ModeOldest:
		// Go to newer messages: start right after the current maximum.
		max := m.maxLoadedOffset()
		if max < 0 {
			return nil
		}
		f := api.ConsumeFlags{
			Follow:        false,
			Tail:          0,
			OffsetFlag:    fmt.Sprintf("%d", max+1),
			LimitMessages: batchSize,
		}
		return &f
	}
	return nil
}

// stopLive cancels the running live stream, if any, and forgets its channels.
func (m *Model) stopLive() {
	if m.cancelConsumption != nil {
		m.cancelConsumption()
		m.cancelConsumption = nil
	}
	m.consuming = false
	m.msgChan = nil
	m.errChan = nil
}

// beginGeneration ends the current fetch generation: it stops the live
// stream, cancels in-flight fetches and decodes, and makes every result still
// on its way stale. It also clears the error shown in place of the table, so
// a new attempt starts clean.
func (m *Model) beginGeneration() {
	m.stopLive()
	if m.cancelFetch != nil {
		m.cancelFetch()
	}
	m.fetchCtx, m.cancelFetch = context.WithCancel(context.Background())
	m.fetchGen++
	m.appendNextFetch = 0
	m.error = nil
}

// isCurrent reports whether a result tagged with topic and gen belongs to this
// page's current generation.
func (m *Model) isCurrent(topic string, gen uint64) bool {
	return topic == m.topicName && gen == m.fetchGen
}

// generationContext is the context of the current generation's work.
func (m *Model) generationContext() context.Context {
	if m.fetchCtx == nil {
		return context.Background()
	}
	return m.fetchCtx
}

// startLive (re)starts the live stream in a new generation, keeping the
// messages already loaded.
func (m *Model) startLive() tea.Cmd {
	m.beginGeneration()
	return m.connectLive()
}

// connectLive starts the live stream for the current generation.
func (m *Model) connectLive() tea.Cmd {
	m.loading = false
	m.SetConnectionStatus(StatusConnecting)
	return m.consumption.StartConsuming()
}

// clearMessages drops every loaded and buffered message.
func (m *Model) clearMessages() {
	m.mu.Lock()
	m.messages = []api.Message{}
	m.consumedMessages = make(map[string]api.Message)
	m.filteredMessages = []api.Message{}
	m.mu.Unlock()
	m.pendingWhilePaused = nil
}

// startForMode clears buffered messages and starts consumption for the
// current consumeMode. Returns the initial command(s) to run.
func (m *Model) startForMode() tea.Cmd {
	m.beginGeneration()
	m.retryCount = 0
	m.clearMessages()
	m.paused = false // a fresh start is never frozen by a pause left over from before
	m.pagination.SetTotalMessages(0)
	m.pagination.FirstPage()
	m.pendingReset = true
	m.markRenderDirty()

	m.consumeFlags = consumeFlagsForMode(m.consumeMode)

	// Sort order: Oldest mode shows lowest offset first; all others show newest first.
	if m.consumeMode == ModeOldest {
		m.pagination.SortOrder = "oldest_first"
	} else {
		m.pagination.SortOrder = "newest_first"
	}

	switch m.consumeMode {
	case ModeNewest, ModeOldest:
		m.loading = true
		return m.consumption.FetchLatestMessages(60)
	case ModeLive:
		return m.connectLive()
	}
	return nil
}

// startForFlags clears buffered messages and (re)starts browsing with an
// explicit set of ConsumeFlags chosen by the seek/partition dialogs (MSG-21/22).
func (m *Model) startForFlags(flags api.ConsumeFlags) tea.Cmd {
	m.beginGeneration()
	m.retryCount = 0
	m.clearMessages()
	m.paused = false // a fresh start is never frozen by a pause left over from before
	m.pagination.SetTotalMessages(0)
	m.pagination.FirstPage()
	m.pendingReset = true
	m.rowStringsDirty = true
	m.markRenderDirty()

	m.consumeFlags = flags
	m.browseStart = time.Now()
	m.browseStats = api.BrowseStats{}

	if flags.Seek.Backward() {
		m.pagination.SortOrder = "newest_first"
	} else {
		m.pagination.SortOrder = "oldest_first"
	}

	// Live seek tails in real time; everything else is a bounded fetch.
	if flags.Seek == api.SeekLive {
		m.consumeMode = ModeLive
		return m.connectLive()
	}

	m.loading = true
	count := int(flags.LimitMessages)
	if count <= 0 {
		count = seekPageSize
	}
	return m.consumption.FetchWithFlags(flags, count)
}
