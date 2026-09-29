package topic

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// RetryConnection counts a retry attempt and schedules it, or reports that
// the retries are exhausted. Call it from Update.
func (cc *ConsumptionController) RetryConnection() tea.Cmd {
	cc.model.retryCount++

	if cc.model.retryCount > cc.retryPolicy.MaxRetries {
		failed := ConnectionFailedMsg{
			Attempts:  cc.model.retryCount,
			LastError: cc.model.lastError,
		}
		return func() tea.Msg { return failed }
	}

	// Schedule retry after delay
	return cc.ScheduleRetry(cc.model.lastError)
}

// ScheduleRetry schedules a retry of the current generation's live stream
// after a backoff delay. Call it from Update after counting the attempt.
func (cc *ConsumptionController) ScheduleRetry(err error) tea.Cmd {
	attempt, topic, gen := cc.model.retryCount, cc.model.topicName, cc.model.fetchGen
	delay := cc.calculateRetryDelay(attempt)

	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return RetryConsumptionMsg{
			Attempt:   attempt,
			LastError: err,
			Topic:     topic,
			Gen:       gen,
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
