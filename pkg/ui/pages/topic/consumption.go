package topic

import (
	"fmt"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
)

// ConsumptionController handles message consumption logic and error recovery
type ConsumptionController struct {
	model       *Model
	retryPolicy RetryPolicy
}

// NewConsumptionController creates a new consumption controller
func NewConsumptionController(model *Model) *ConsumptionController {
	return &ConsumptionController{
		model:       model,
		retryPolicy: DefaultRetryPolicy(),
	}
}

// StartConsuming initiates message consumption for the topic
func (cc *ConsumptionController) StartConsuming() tea.Cmd {
	return func() tea.Msg {
		shared.DebugLog("Starting consumption for topic: %s", cc.model.topicName)

		// Set consuming state
		cc.model.loading = true
		cc.model.SetConnectionStatus(StatusConnecting)

		// Start consuming messages using the correct ConsumeTopic method
		// This method blocks, so we'll need to handle it differently
		// For now, just return a start consuming message without channels
		shared.DebugLog("Successfully started consumption setup")
		return StartConsumingMsg{
			MsgChan: nil, // Will be set up in handlers
			ErrChan: nil, // Will be set up in handlers
			Cancel:  nil, // Will be set up in handlers
		}
	}
}

// StopConsuming stops message consumption
func (cc *ConsumptionController) StopConsuming() tea.Cmd {
	return func() tea.Msg {
		shared.DebugLog("Stopping consumption for topic: %s", cc.model.topicName)
		return StopConsumingMsg{}
	}
}

// ListenForMessages creates a command to listen for incoming messages
func (cc *ConsumptionController) ListenForMessages(msgChan <-chan api.Message) tea.Cmd {
	return func() tea.Msg {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				shared.DebugLog("Message channel closed")
				return ErrorMsg(fmt.Errorf("message channel was closed"))
			}

			shared.DebugLog("Received message: partition=%d, offset=%d", msg.Partition, msg.Offset)

			// Return the message via the command
			return MessageConsumedMsg{Message: msg}

		case <-time.After(time.Millisecond * 100): // Timeout to prevent blocking
			// No message received, continue listening
			return ContinuousListenMsg{}
		}
	}
}

// ListenForErrors creates a command to listen for consumption errors
func (cc *ConsumptionController) ListenForErrors(errChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case err, ok := <-errChan:
			if !ok {
				shared.DebugLog("Error channel closed")
				return ErrorMsg(fmt.Errorf("error channel was closed"))
			}

			shared.DebugLog("Received consumption error: %v", err)

			// Return the error via the command
			return ErrorMsg(err)

		case <-time.After(time.Millisecond * 100): // Timeout to prevent blocking
			// No error received, continue listening
			return ContinuousErrorListenMsg{}
		}
	}
}

// RetryConnection attempts to retry the connection after a failure
func (cc *ConsumptionController) RetryConnection() tea.Cmd {
	return func() tea.Msg {
		cc.model.retryCount++

		if cc.model.retryCount > cc.retryPolicy.MaxRetries {
			shared.DebugLog("Max retries exceeded (%d), giving up", cc.retryPolicy.MaxRetries)
			return ConnectionFailedMsg{
				Attempts:  cc.model.retryCount,
				LastError: cc.model.lastError,
			}
		}

		shared.DebugLog("Retrying connection, attempt %d/%d", cc.model.retryCount, cc.retryPolicy.MaxRetries)

		// Schedule retry after delay
		return cc.ScheduleRetry(cc.model.lastError)
	}
}

// ScheduleRetry schedules a retry attempt after a delay
func (cc *ConsumptionController) ScheduleRetry(err error) tea.Cmd {
	delay := cc.calculateRetryDelay(cc.model.retryCount)

	return tea.Tick(delay, func(t time.Time) tea.Msg {
		shared.DebugLog("Executing scheduled retry")
		return RetryConsumptionMsg{
			Attempt:   cc.model.retryCount,
			LastError: err,
		}
	})
}

// calculateRetryDelay calculates the delay for the next retry attempt
func (cc *ConsumptionController) calculateRetryDelay(attempt int) time.Duration {
	if !cc.retryPolicy.EnableExponential {
		return cc.retryPolicy.InitialDelay
	}

	// Exponential backoff: delay = initial * (backoffFactor ^ (attempt - 1))
	delay := cc.retryPolicy.InitialDelay
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * cc.retryPolicy.BackoffFactor)
	}

	// Cap at maximum delay
	if delay > cc.retryPolicy.MaxDelay {
		delay = cc.retryPolicy.MaxDelay
	}

	return delay
}

// ResetRetryCount resets the retry counter
func (cc *ConsumptionController) ResetRetryCount() {
	cc.model.retryCount = 0
}

// SetRetryPolicy updates the retry policy
func (cc *ConsumptionController) SetRetryPolicy(policy RetryPolicy) {
	cc.retryPolicy = policy
}

// GetRetryPolicy returns the current retry policy
func (cc *ConsumptionController) GetRetryPolicy() RetryPolicy {
	return cc.retryPolicy
}

// IsRetrying returns whether the controller is currently retrying
func (cc *ConsumptionController) IsRetrying() bool {
	return cc.model.retryCount > 0 && cc.model.retryCount <= cc.retryPolicy.MaxRetries
}

// GetRetryStatus returns information about retry status
func (cc *ConsumptionController) GetRetryStatus() (current, max int, nextDelay time.Duration) {
	current = cc.model.retryCount
	max = cc.retryPolicy.MaxRetries

	if current < max {
		nextDelay = cc.calculateRetryDelay(current + 1)
	}

	return current, max, nextDelay
}

// HandlePanicRecovery handles panics in consumption goroutines
func (cc *ConsumptionController) HandlePanicRecovery() tea.Cmd {
	return func() tea.Msg {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in consumption: %v", r)
			shared.DebugLog("Recovered from panic: %v", err)
			return ErrorMsg(err)
		}
		return nil
	}
}

// ValidateConsumptionFlags validates the consumption configuration
func (cc *ConsumptionController) ValidateConsumptionFlags(flags api.ConsumeFlags) error {
	if flags.OffsetFlag == "" {
		return fmt.Errorf("offset flag cannot be empty")
	}

	if flags.Tail < 0 {
		return fmt.Errorf("invalid tail: %d", flags.Tail)
	}

	return nil
}

// GetConsumptionStats returns statistics about message consumption
func (cc *ConsumptionController) GetConsumptionStats() ConsumptionStats {
	return ConsumptionStats{
		MessagesConsumed: len(cc.model.messages),
		ErrorCount:       len(cc.model.errorHistory),
		RetryCount:       cc.model.retryCount,
		IsConsuming:      cc.model.consuming,
		IsPaused:         cc.model.paused,
		ConnectionStatus: cc.model.connectionStatus,
		LastError:        cc.model.lastError,
		StartTime:        time.Time{},             // TODO: Track start time
		Duration:         time.Since(time.Time{}), // TODO: Calculate duration
	}
}

// ConsumptionStats contains statistics about message consumption
type ConsumptionStats struct {
	MessagesConsumed int
	ErrorCount       int
	RetryCount       int
	IsConsuming      bool
	IsPaused         bool
	ConnectionStatus string
	LastError        error
	StartTime        time.Time
	Duration         time.Duration
}

// String returns a string representation of the consumption stats
func (cs ConsumptionStats) String() string {
	return fmt.Sprintf(
		"Messages: %d, Errors: %d, Retries: %d, Status: %s, Consuming: %t, Paused: %t",
		cs.MessagesConsumed,
		cs.ErrorCount,
		cs.RetryCount,
		cs.ConnectionStatus,
		cs.IsConsuming,
		cs.IsPaused,
	)
}

// UpdateConsumptionFlags updates the consumption flags and restarts consumption if needed
func (cc *ConsumptionController) UpdateConsumptionFlags(flags api.ConsumeFlags) tea.Cmd {
	if err := cc.ValidateConsumptionFlags(flags); err != nil {
		return func() tea.Msg {
			return ErrorMsg(err)
		}
	}

	cc.model.consumeFlags = flags

	// If currently consuming, restart with new flags
	if cc.model.consuming {
		return tea.Batch(
			cc.StopConsuming(),
			cc.StartConsuming(),
		)
	}

	return nil
}

// GetHealthStatus returns the health status of the consumption controller
func (cc *ConsumptionController) GetHealthStatus() string {
	if cc.model.error != nil {
		return "unhealthy"
	}

	if cc.model.consuming && !cc.model.paused {
		return "healthy"
	}

	if cc.model.paused {
		return "paused"
	}

	return "idle"
}
