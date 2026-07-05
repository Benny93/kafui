package topic

import (
	"fmt"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components"
	formpkg "github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

// Handlers manages event handling for the topic page
type Handlers struct {
	model *Model
}

// NewHandlers creates a new Handlers instance
func NewHandlers(model *Model) *Handlers {
	return &Handlers{
		model: model,
	}
}

// Handle routes messages to appropriate handlers
func (h *Handlers) Handle(model *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	h.model = model // Update model reference
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return h.handleWindowSize(model, msg)

	case tea.KeyMsg:
		return h.handleKeyMsg(model, msg)

	case tea.MouseMsg:
		return h.handleMouseMsg(model, msg)

	case MessageConsumedMsg:
		return h.handleMessageConsumed(model, msg)

	case MessagesFetchedMsg:
		return h.handleMessagesFetched(model, msg)

	case VisibleMessagesDecodedMsg:
		return h.handleVisibleMessagesDecoded(model, msg)

	case StartFetchMsg:
		return h.handleStartFetch(model, msg)

	case components.ProgressMsg:
		var cmd tea.Cmd
		model.fetchProgressBar, cmd = model.fetchProgressBar.Update(msg)
		model.markRenderDirty()
		return model, cmd

	// Animation frames for the progress bar spring animation.
	case components.ProgressBarFrameMsg:
		var cmd tea.Cmd
		model.fetchProgressBar, cmd = model.fetchProgressBar.Update(msg)
		return model, cmd

	case StartConsumingMsg:
		return h.handleStartConsuming(model, msg)

	case StopConsumingMsg:
		return h.handleStopConsuming(model, msg)

	case ContinuousListenMsg:
		return h.handleContinuousListen(model, msg)

	case ContinuousErrorListenMsg:
		return h.handleContinuousErrorListen(model, msg)

	case ConnectionStatusMsg:
		return h.handleConnectionStatus(model, msg)

	case RetryConsumptionMsg:
		return h.handleRetryConsumption(model, msg)

	case ConnectionFailedMsg:
		return h.handleConnectionFailed(model, msg)

	case SearchMessagesMsg:
		return h.handleSearchMessages(model, msg)

	case ClearSearchMsg:
		return h.handleClearSearch(model, msg)

	case MessageSelectedMsg:
		return h.handleMessageSelected(model, msg)

	case TopicGroupsLoadedMsg:
		return h.handleTopicGroupsLoaded(model, msg)

	case TopicDetailsLoadedMsg:
		return h.handleTopicDetailsLoaded(model, msg)

	case TopicConfigLoadedMsg:
		return h.handleTopicConfigLoaded(model, msg)

	case EditConfigLoadedMsg:
		return h.handleEditConfigLoaded(model, msg)

	case settingsUpdatedMsg:
		return h.handleSettingsUpdated(model, msg)

	case topicMutationMsg:
		return h.handleTopicMutation(model, msg)

	case AnalysisLoadedMsg:
		return h.handleAnalysisLoaded(model, msg)

	case analysisTickMsg:
		return h.handleAnalysisTick(model, msg)

	case formpkg.FormSubmitMsg:
		// Route the submission to whichever form overlay is open.
		if model.showMutationForm {
			return h.handleMutationFormSubmit(model, msg.Values)
		}
		if model.showSettingsEdit {
			return h.handleSettingsFormSubmit(model, msg.Values)
		}
		if model.showSeek {
			return h.handleSeekFormSubmit(model, msg.Values)
		}
		if model.showPartitions {
			return h.handlePartitionFormSubmit(model, msg.Values)
		}
		if model.showProduce {
			return h.handleProduceFormSubmit(model, msg.Values)
		}
		if model.showProjections {
			return h.handleProjectionsSubmit(model, msg.Values)
		}
		return model, nil

	case formpkg.FormCancelMsg:
		model.showSettingsEdit = false
		model.showMutationForm = false
		model.showSeek = false
		model.showPartitions = false
		model.showProduce = false
		model.showProjections = false
		model.settingsForm = nil
		model.mutationForm = nil
		model.seekForm = nil
		model.partitionForm = nil
		model.produceForm = nil
		model.markRenderDirty()
		return model, nil

	case ErrorMsg:
		return h.handleError(model, msg)

	case spinner.TickMsg:
		return h.handleSpinnerTick(model, msg)

	default:
		// Handle any unrecognized messages
		return model, tea.Batch(cmds...)
	}
}

func (h *Handlers) handleWindowSize(model *Model, msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	model.SetDimensions(msg.Width, msg.Height)
	return model, nil
}

func (h *Handlers) handleKeyMsg(model *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Delegate to the keys handler
	cmd := model.keys.HandleKey(model, msg)
	return model, cmd
}

func (h *Handlers) handleMessageConsumed(model *Model, msg MessageConsumedMsg) (tea.Model, tea.Cmd) {
	// Add the consumed message to internal storage (doesn't trigger view update)
	model.addMessageInternal(msg.Message)
	
	// Ensure messages are sorted for pagination
	model.sortMessages()
	model.updateMessageTable()
	
	// Update total messages for pagination
	model.pagination.SetTotalMessages(len(model.filteredMessages))
	
	model.markRenderDirty()

	// Continue listening for more messages if we're still consuming
	if model.consuming && model.msgChan != nil {
		return model, model.consumption.ListenForMessages(model.msgChan)
	}

	return model, nil
}

func (h *Handlers) handleMessagesFetched(model *Model, msg MessagesFetchedMsg) (tea.Model, tea.Cmd) {
	shared.Log.Info("messages fetched", "topic", model.topicName, "count", len(msg.Messages), "pendingAppend", model.appendNextFetch)

	// Treat as append if there are outstanding append-batch fetches.
	appending := model.appendNextFetch > 0
	if appending {
		model.appendNextFetch--
	}
	// Keep loading indicator active only while further batches are still in-flight.
	model.loading = model.appendNextFetch > 0

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

	if len(model.messages) == 0 {
		model.statusMessage = "Topic is empty — no messages found"
	} else if appending && len(msg.Messages) == 0 {
		model.statusMessage = "No more messages to load"
	} else {
		model.statusMessage = fmt.Sprintf("Loaded %d messages", len(model.messages))
	}

	// Decode ALL fetched messages in one background pass so that every page is
	// pre-decoded before the user scrolls. A shared schema registry client
	// (cachedSchemaCache) means only one HTTP round-trip per unique schema ID.
	return model, model.consumption.DecodeVisibleMessages(model.messages)
}

func (h *Handlers) handleVisibleMessagesDecoded(model *Model, msg VisibleMessagesDecodedMsg) (tea.Model, tea.Cmd) {
	if len(msg.Messages) == 0 {
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

func (h *Handlers) handleStartConsuming(model *Model, msg StartConsumingMsg) (tea.Model, tea.Cmd) {
	// Set consumption state
	model.consuming = true
	model.loading = false
	model.msgChan = msg.MsgChan
	model.errChan = msg.ErrChan
	model.cancelConsumption = msg.Cancel
	model.SetConnectionStatus(StatusConnected)

	// Start listening for messages and errors
	var cmds []tea.Cmd
	if msg.MsgChan != nil {
		cmds = append(cmds, model.consumption.ListenForMessages(msg.MsgChan))
	}
	if msg.ErrChan != nil {
		cmds = append(cmds, model.consumption.ListenForErrors(msg.ErrChan))
	}

	return model, tea.Batch(cmds...)
}

func (h *Handlers) handleStopConsuming(model *Model, msg StopConsumingMsg) (tea.Model, tea.Cmd) {
	// Stop consumption
	model.consuming = false
	model.paused = false
	if model.cancelConsumption != nil {
		model.cancelConsumption()
		model.cancelConsumption = nil
	}
	model.SetConnectionStatus(StatusDisconnected)

	return model, nil
}

func (h *Handlers) handleContinuousListen(model *Model, msg ContinuousListenMsg) (tea.Model, tea.Cmd) {
	// Continue listening for messages if we're still consuming
	if model.consuming && model.msgChan != nil {
		return model, model.consumption.ListenForMessages(model.msgChan)
	}
	return model, nil
}

func (h *Handlers) handleContinuousErrorListen(model *Model, msg ContinuousErrorListenMsg) (tea.Model, tea.Cmd) {
	// Continue listening for errors if we're still consuming
	// Use a reasonable interval to prevent UI freezing
	if model.consuming && model.errChan != nil {
		return model, model.consumption.ListenForErrors(model.errChan)
	}
	return model, nil
}

func (h *Handlers) handleConnectionStatus(model *Model, msg ConnectionStatusMsg) (tea.Model, tea.Cmd) {
	model.SetConnectionStatus(string(msg))
	return model, nil
}

func (h *Handlers) handleRetryConsumption(model *Model, msg RetryConsumptionMsg) (tea.Model, tea.Cmd) {
	model.retryCount = msg.Attempt
	model.SetConnectionStatus(StatusRetrying)

	if msg.LastError != nil {
		shared.Log.Warn("retrying consumption", "topic", model.topicName, "attempt", msg.Attempt, "err", msg.LastError)
		model.SetError(msg.LastError)
	}

	// Try to restart consumption
	if model.consumption != nil {
		return model, model.consumption.StartConsuming()
	}

	return model, nil
}

func (h *Handlers) handleConnectionFailed(model *Model, msg ConnectionFailedMsg) (tea.Model, tea.Cmd) {
	model.retryCount = msg.Attempts
	model.consuming = false
	model.loading = false

	if msg.LastError != nil {
		shared.Log.Error("connection failed", "topic", model.topicName, "attempts", msg.Attempts, "err", msg.LastError)
		model.SetError(msg.LastError)
	}

	model.SetConnectionStatus(StatusFailed)
	return model, nil
}

func (h *Handlers) handleSearchMessages(model *Model, msg SearchMessagesMsg) (tea.Model, tea.Cmd) {
	// Update search input and filter messages
	model.searchInput.SetValue(string(msg))
	model.FilterMessages()
	return model, nil
}

func (h *Handlers) handleClearSearch(model *Model, msg ClearSearchMsg) (tea.Model, tea.Cmd) {
	// Clear search and show all messages
	model.searchInput.SetValue("")
	model.searchMode = false
	model.searchInput.Blur()
	model.FilterMessages()
	return model, nil
}

func (h *Handlers) handleMessageSelected(model *Model, msg MessageSelectedMsg) (tea.Model, tea.Cmd) {
	// Set the selected message
	model.selectedMessage = &msg.Message
	model.statusMessage = "Message selected"
	return model, nil
}

func (h *Handlers) handleError(model *Model, msg ErrorMsg) (tea.Model, tea.Cmd) {
	shared.Log.Error("topic page error", "topic", model.topicName, "err", error(msg))
	model.SetError(error(msg))
	model.loading = false

	// If we're supposed to be consuming, try to retry
	if model.consuming && model.retryCount < model.maxRetries {
		return model, model.consumption.ScheduleRetry(error(msg))
	}

	return model, nil
}

func (h *Handlers) handleSpinnerTick(model *Model, msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	model.spinner, cmd = model.spinner.Update(msg)
	return model, cmd
}

func (h *Handlers) handleStartFetch(model *Model, msg StartFetchMsg) (tea.Model, tea.Cmd) {
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

const (
	// tableHeaderLines is the number of non-data lines rendered above the first
	// data row: top border + header row + separator line.
	tableHeaderLines = 3
)

func (h *Handlers) handleMouseMsg(model *Model, msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if model.cursorRow > 0 {
			model.cursorRow--
			model.markRenderDirty()
		}

	case tea.MouseButtonWheelDown:
		visibleCount := len(model.pagination.GetVisibleMessages(model.filteredMessages))
		if model.cursorRow < visibleCount-1 {
			model.cursorRow++
			model.markRenderDirty()
		}

	case tea.MouseButtonLeft:
		z := zone.Get("message-table")
		if !z.InBounds(msg) {
			break
		}
		_, relY := z.Pos(msg)
		row := relY - tableHeaderLines
		if row < 0 {
			break
		}
		visibleCount := len(model.pagination.GetVisibleMessages(model.filteredMessages))
		if row >= visibleCount {
			break
		}

		model.cursorRow = row
		model.markRenderDirty()
		return model, model.keys.handleSelect(model)
	}

	return model, nil
}
