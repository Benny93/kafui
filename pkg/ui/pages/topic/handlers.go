package topic

import (
	"github.com/Benny93/kafui/pkg/ui/components"
	formpkg "github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
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
		// Progress from a fetch the bar no longer tracks: the bar is inactive
		// after its Done, the zero value comes from a closed channel, and a
		// superseded fetch reports on a channel other than the current one.
		// Forwarding any of them would end or re-arm the current fetch's bar.
		if !model.fetchProgressBar.IsActive() || !model.fetchProgressBar.Tracks(msg) || (msg.Total == 0 && !msg.Done) {
			return model, nil
		}
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

	case streamClosedMsg:
		return h.handleStreamClosed(model, msg)

	case liveErrorMsg:
		return h.handleLiveError(model, msg)

	case topicMetaLoadedMsg:
		return h.handleTopicMetaLoaded(model, msg)

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
	model.markRenderDirty()
	return model, nil
}

func (h *Handlers) handleSpinnerTick(model *Model, msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	model.spinner, cmd = model.spinner.Update(msg)
	return model, cmd
}
