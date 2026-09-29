package topic

import (
	"fmt"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/ui/core"
)

func (h *Handlers) handleMessagesFetched(model *Model, msg MessagesFetchedMsg) (tea.Model, tea.Cmd) {
	// A result for another topic's page, or for a fetch a refresh, seek or
	// mode switch has since replaced.
	if !model.isCurrent(msg.Topic, msg.Gen) {
		return model, nil
	}
	shared.Log.Info("messages fetched", "topic", model.topicName, "count", len(msg.Messages), "append", msg.Append, "err", msg.Err)

	appending := msg.Append
	if appending && model.appendNextFetch > 0 {
		model.appendNextFetch--
	}
	// Keep loading indicator active only while further batches are still in-flight.
	model.loading = model.appendNextFetch > 0

	// A failed fetch that returned nothing is an error, not an empty topic.
	if msg.Err != nil && len(msg.Messages) == 0 {
		if appending {
			model.statusMessage = fmt.Sprintf("Loading more messages failed: %v", msg.Err)
			model.markRenderDirty()
			return model, core.NotifyError("Loading more messages failed", msg.Err)
		}
		model.mu.Lock()
		model.messages = []api.Message{}
		model.consumedMessages = make(map[string]api.Message)
		model.filteredMessages = []api.Message{}
		model.mu.Unlock()
		model.pagination.SetTotalMessages(0)
		model.SetError(msg.Err)
		model.markRenderDirty()
		return model, core.NotifyError("Fetching messages failed", msg.Err)
	}

	if !appending {
		// Fresh fetch — clear existing messages.
		model.mu.Lock()
		model.messages = []api.Message{}
		model.consumedMessages = make(map[string]api.Message)
		model.mu.Unlock()
	}

	// Add all fetched messages
	for _, m := range msg.Messages {
		model.addMessageInternal(m)
	}

	// Ensure messages are sorted for pagination
	model.sortMessages()

	// Recompute browse statistics from the full loaded set (MSG-27).
	model.browseStats = api.BrowseStats{}
	for _, msg := range model.messages {
		model.browseStats.AddMessage(msg)
	}
	if !model.browseStart.IsZero() {
		model.browseStats.ElapsedMs = time.Since(model.browseStart).Milliseconds()
	}

	// Re-apply the active filter (substring or smart) to the new data (MSG-23/24).
	model.applyFilter()

	// Update pagination
	model.pagination.SetTotalMessages(len(model.filteredMessages))

	if appending {
		// Navigate to the last page so the user sees the newly added messages.
		model.pagination.LastPage()
	} else {
		// Reset to first page to see the newest messages.
		model.pagination.FirstPage()
	}
	// Do NOT set pendingReset — preserve cursor position if still in bounds.
	// updateMessageTable() already clamps cursorRow when it exceeds visibleCount.

	// Rebuild table rows with sorted data before the first render.
	model.updateMessageTable()

	// Mark render as dirty to show the messages
	model.markRenderDirty()

	var notify tea.Cmd
	switch {
	case msg.Err != nil:
		// Partial result: keep what arrived, but say the fetch stopped early.
		model.statusMessage = fmt.Sprintf("Loaded %d messages, then the fetch failed: %v", len(model.messages), msg.Err)
		notify = core.NotifyError("Fetch stopped early", msg.Err)
	case len(model.messages) == 0:
		model.statusMessage = "Topic is empty — no messages found"
	case appending && len(msg.Messages) == 0:
		model.statusMessage = "No more messages to load"
	default:
		model.statusMessage = fmt.Sprintf("Loaded %d messages", len(model.messages))
	}
	if msg.Err == nil && model.connectionStatus != StatusConnected {
		model.connectionStatus = StatusConnected
	}

	// Decode ALL fetched messages in one background pass so that every page is
	// pre-decoded before the user scrolls. A shared schema registry client
	// (cachedSchemaCache) means only one HTTP round-trip per unique schema ID.
	return model, tea.Batch(notify, model.consumption.DecodeVisibleMessages(model.messages))
}

func (h *Handlers) handleVisibleMessagesDecoded(model *Model, msg VisibleMessagesDecodedMsg) (tea.Model, tea.Cmd) {
	if len(msg.Messages) == 0 || !model.isCurrent(msg.Topic, msg.Gen) {
		return model, nil
	}
	// Build a lookup map for fast update
	decodedByKey := make(map[string]api.Message, len(msg.Messages))
	for _, d := range msg.Messages {
		decodedByKey[fmt.Sprintf("%d-%d", d.Partition, d.Offset)] = d
	}

	model.mu.Lock()
	for key, decoded := range decodedByKey {
		if _, exists := model.consumedMessages[key]; exists {
			model.consumedMessages[key] = decoded
		}
	}
	for i, m := range model.messages {
		key := fmt.Sprintf("%d-%d", m.Partition, m.Offset)
		if decoded, exists := decodedByKey[key]; exists {
			model.messages[i] = decoded
		}
	}
	for i, m := range model.filteredMessages {
		key := fmt.Sprintf("%d-%d", m.Partition, m.Offset)
		if decoded, exists := decodedByKey[key]; exists {
			model.filteredMessages[i] = decoded
		}
	}
	model.mu.Unlock()

	model.updateMessageTable()
	model.markRenderDirty()
	return model, nil
}

func (h *Handlers) handleStartFetch(model *Model, msg StartFetchMsg) (tea.Model, tea.Cmd) {
	// A fetch that was superseded before it even started reporting: its
	// context is already cancelled and its result will be dropped.
	if !model.isCurrent(msg.Topic, msg.Gen) {
		return model, nil
	}
	model.loading = true
	if msg.Append {
		model.appendNextFetch++
	} else if model.browseStart.IsZero() {
		// Track elapsed for browse statistics (MSG-27). startForFlags sets this
		// already; cover the normal mode fetches here.
		model.browseStart = time.Now()
	}
	model.SetConnectionStatus(StatusConnecting)
	model.markRenderDirty()

	// Delegate listening to the encapsulated component; also listen for the result.
	progressCmd := model.fetchProgressBar.StartListening(msg.ProgressCh, msg.Total)
	return model, tea.Batch(progressCmd, listenForResult(msg.ResultCh))
}
