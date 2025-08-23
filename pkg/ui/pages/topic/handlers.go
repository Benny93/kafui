package topic

import (
	"fmt"
	"time"

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

	// Add type information to debug logging
	switch msg.(type) {
	case spinner.TickMsg:
		shared.DebugLog("Topic Update - spinner.TickMsg")
	case tea.KeyMsg:
		shared.DebugLog("Topic Update - tea.KeyMsg: %s", msg.(tea.KeyMsg).String())
	case MessageConsumedMsg:
		shared.DebugLog("Topic Update - MessageConsumedMsg")
	case StartConsumingMsg:
		shared.DebugLog("Topic Update - StartConsumingMsg")
	case ContinuousListenMsg:
		shared.DebugLog("Topic Update - ContinuousListenMsg")
	case ContinuousErrorListenMsg:
		shared.DebugLog("Topic Update - ContinuousErrorListenMsg")
	default:
		shared.DebugLog("Topic Update - %T: %v", msg, msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return h.handleWindowSize(model, msg)

	case tea.KeyMsg:
		return h.handleKeyMsg(model, msg)

	case MessageConsumedMsg:
		return h.handleMessageConsumed(model, msg)

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

	case TimerTickMsg:
		return h.handleTimerTick(model, msg)

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
	// Add the consumed message
	model.AddMessage(msg.Message)
	
	// Continue listening for more messages if we're still consuming
	var cmds []tea.Cmd
	if model.consuming && model.msgChan != nil {
		cmds = append(cmds, model.consumption.ListenForMessages(model.msgChan))
	}
	
	return model, tea.Batch(cmds...)
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

func (h *Handlers) handleTimerTick(model *Model, msg TimerTickMsg) (tea.Model, tea.Cmd) {
	// Update last update time
	model.lastUpdate = time.Time(msg)
	
	// Schedule next timer tick
	return model, tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TimerTickMsg(t)
	})
}

func (h *Handlers) handleError(model *Model, msg ErrorMsg) (tea.Model, tea.Cmd) {
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

// Helper methods

// scheduleMessagePolling creates a command to poll for new messages periodically
func (h *Handlers) scheduleMessagePolling() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return ContinuousListenMsg{}
	})
}

// scheduleErrorPolling creates a command to poll for errors periodically
func (h *Handlers) scheduleErrorPolling() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return ContinuousErrorListenMsg{}
	})
}

// validateModel performs basic validation on the model state
func (h *Handlers) validateModel(model *Model) error {
	if model.topicName == "" {
		return ErrorMsg(fmt.Errorf("topic name cannot be empty"))
	}
	
	if model.dataSource == nil {
		return ErrorMsg(fmt.Errorf("data source cannot be nil"))
	}
	
	return nil
}

