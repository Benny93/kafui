package topic

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/ui/core"
)

func (h *Handlers) handleMessageConsumed(model *Model, msg MessageConsumedMsg) (tea.Model, tea.Cmd) {
	// A message from a stream that was since replaced or stopped: drop it and
	// let that listener chain end.
	if !model.isCurrent(msg.Topic, msg.Gen) {
		return model, nil
	}

	// The stream delivers again, so a later failure gets a fresh set of retries.
	model.retryCount = 0

	// Paused: keep draining the stream, but hold the message until resume.
	if model.paused {
		model.bufferWhilePaused(msg.Message)
		if model.consuming && model.msgChan != nil {
			return model, model.consumption.ListenForMessages(model.msgChan)
		}
		return model, nil
	}

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

func (h *Handlers) handleStartConsuming(model *Model, msg StartConsumingMsg) (tea.Model, tea.Cmd) {
	// A stream started for an older generation (or another page): stop it now,
	// nothing will ever read it.
	if !model.isCurrent(msg.Topic, msg.Gen) {
		if msg.Cancel != nil {
			msg.Cancel()
		}
		return model, nil
	}
	// Never lose the cancel func of a stream that is still running.
	if model.cancelConsumption != nil {
		model.cancelConsumption()
	}

	// Set consumption state
	model.consuming = true
	model.loading = false
	model.error = nil
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
	model.stopLive()
	model.paused = false
	model.SetConnectionStatus(StatusDisconnected)

	return model, nil
}

func (h *Handlers) handleContinuousListen(model *Model, msg ContinuousListenMsg) (tea.Model, tea.Cmd) {
	// Continue listening for messages if we're still consuming this stream
	if model.isCurrent(msg.Topic, msg.Gen) && model.consuming && model.msgChan != nil {
		return model, model.consumption.ListenForMessages(model.msgChan)
	}
	return model, nil
}

func (h *Handlers) handleContinuousErrorListen(model *Model, msg ContinuousErrorListenMsg) (tea.Model, tea.Cmd) {
	// Continue listening for errors if we're still consuming
	// Use a reasonable interval to prevent UI freezing
	if model.isCurrent(msg.Topic, msg.Gen) && model.consuming && model.errChan != nil {
		return model, model.consumption.ListenForErrors(model.errChan)
	}
	return model, nil
}

func (h *Handlers) handleConnectionStatus(model *Model, msg ConnectionStatusMsg) (tea.Model, tea.Cmd) {
	model.SetConnectionStatus(string(msg))
	return model, nil
}

func (h *Handlers) handleRetryConsumption(model *Model, msg RetryConsumptionMsg) (tea.Model, tea.Cmd) {
	// The user refreshed, seeked or left the stream since this was scheduled.
	if !model.isCurrent(msg.Topic, msg.Gen) || model.consumption == nil {
		return model, nil
	}
	model.retryCount = msg.Attempt
	if msg.LastError != nil {
		shared.Log.Warn("retrying consumption", "topic", model.topicName, "attempt", msg.Attempt, "err", msg.LastError)
	}

	// Restart in a new generation, which cancels whatever is left of the old
	// consumer before the new one starts.
	cmd := model.startLive()
	model.SetConnectionStatus(StatusRetrying)
	model.statusMessage = fmt.Sprintf("Retrying (attempt %d/%d)…", msg.Attempt, model.maxRetries)
	return model, cmd
}

// handleStreamClosed handles the live stream's channels closing. After a
// deliberate stop consuming is already false; otherwise the stream died.
func (h *Handlers) handleStreamClosed(model *Model, msg streamClosedMsg) (tea.Model, tea.Cmd) {
	if !model.isCurrent(msg.Topic, msg.Gen) || !model.consuming {
		return model, nil
	}
	return h.failLive(model, shared.NewUIError(
		shared.ErrorTypeDataLoad,
		"Message stream closed unexpectedly",
		nil,
	))
}

// handleLiveError handles an error reported by the live stream.
func (h *Handlers) handleLiveError(model *Model, msg liveErrorMsg) (tea.Model, tea.Cmd) {
	if !model.isCurrent(msg.Topic, msg.Gen) || !model.consuming {
		return model, nil
	}
	return h.failLive(model, msg.Err)
}

// failLive stops the failed live stream, shows the error and schedules a
// retry while retries remain. Stopping first means the stream's remaining
// errors and its channel closing are ignored instead of each scheduling
// another retry.
func (h *Handlers) failLive(model *Model, err error) (tea.Model, tea.Cmd) {
	shared.Log.Error("live consumption failed", "topic", model.topicName, "err", err)
	model.stopLive()
	model.SetError(err)
	model.loading = false
	model.markRenderDirty()

	if model.retryCount < model.maxRetries {
		model.retryCount++
		return model, model.consumption.ScheduleRetry(err)
	}
	model.SetConnectionStatus(StatusFailed)
	return model, core.NotifyError("Live consumption failed", err)
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
