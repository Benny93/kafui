package core

import (
	"math"
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// RetryConfig holds configuration for retry operations
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// InitialDelay is the delay before the first retry
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration

	// Multiplier is the factor by which the delay increases
	Multiplier float64

	// Jitter adds randomness to prevent thundering herd
	Jitter float64
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	}
}

// RetryState tracks the state of a retry operation
type RetryState struct {
	// Current attempt number (0-based)
	Attempt int

	// Last error encountered
	LastError error

	// Next retry time
	NextRetry time.Time

	// Config for this retry operation
	Config RetryConfig
}

// NewRetryState creates a new retry state
func NewRetryState(config RetryConfig) *RetryState {
	return &RetryState{
		Attempt: 0,
		Config:  config,
	}
}

// CanRetry returns true if another retry attempt is allowed
func (rs *RetryState) CanRetry() bool {
	return rs.Attempt < rs.Config.MaxRetries
}

// GetDelay calculates the delay for the current attempt
func (rs *RetryState) GetDelay() time.Duration {
	// Calculate exponential backoff
	delay := float64(rs.Config.InitialDelay) * math.Pow(rs.Config.Multiplier, float64(rs.Attempt))

	// Cap at max delay
	if delay > float64(rs.Config.MaxDelay) {
		delay = float64(rs.Config.MaxDelay)
	}

	// Add jitter if configured
	if rs.Config.Jitter > 0 {
		jitter := delay * rs.Config.Jitter * (0.5 - rand.Float64())
		delay += jitter
	}

	return time.Duration(delay)
}

// RecordAttempt records a failed attempt and returns the delay before next retry
func (rs *RetryState) RecordAttempt(err error) time.Duration {
	rs.Attempt++
	rs.LastError = err
	rs.NextRetry = time.Now().Add(rs.GetDelay())
	return rs.GetDelay()
}

// Reset resets the retry state
func (rs *RetryState) Reset() {
	rs.Attempt = 0
	rs.LastError = nil
	rs.NextRetry = time.Time{}
}

// RetryError wraps an error with retry information
type RetryError struct {
	// Original error
	Err error

	// Attempt number when this error occurred
	Attempt int

	// Max retries configured
	MaxRetries int
}

// Error implements the error interface
func (re *RetryError) Error() string {
	if re.Err == nil {
		return "retry error"
	}
	return re.Err.Error()
}

// Unwrap returns the wrapped error
func (re *RetryError) Unwrap() error {
	return re.Err
}

// RetryCommand wraps a command with retry logic
type RetryCommand struct {
	// Command to execute
	Cmd tea.Cmd

	// Retry configuration
	Config RetryConfig

	// Current retry state
	State *RetryState

	// Error handler called on each retry
	OnError func(err error, attempt int) tea.Cmd

	// Success handler called on success
	OnSuccess func(result tea.Msg) tea.Cmd
}

// NewRetryCommand creates a new retry command
func NewRetryCommand(cmd tea.Cmd, config RetryConfig) *RetryCommand {
	return &RetryCommand{
		Cmd:   cmd,
		Config: config,
		State: NewRetryState(config),
	}
}

// Execute executes the retry command
func (rc *RetryCommand) Execute() tea.Cmd {
	return func() tea.Msg {
		// Execute the wrapped command
		result := rc.Cmd()

		// Check if it's an error
		if err, ok := result.(error); ok {
			// Record the failed attempt
			delay := rc.State.RecordAttempt(err)

			// Check if we can retry
			if rc.State.CanRetry() {
				// Call error handler if provided
				if rc.OnError != nil {
					if cmd := rc.OnError(err, rc.State.Attempt); cmd != nil {
						return cmd()
					}
				}

				// Wait for the delay and retry
				time.Sleep(delay)
				return rc.Execute()()
			}

			// Max retries exceeded, return the error
			return &RetryError{
				Err:        err,
				Attempt:    rc.State.Attempt,
				MaxRetries: rc.Config.MaxRetries,
			}
		}

		// Success - call success handler if provided
		if rc.OnSuccess != nil {
			if cmd := rc.OnSuccess(result); cmd != nil {
				return cmd()
			}
		}

		return result
	}
}

// RetryWithBackoff creates a command that executes another command with exponential backoff
func RetryWithBackoff(cmd tea.Cmd, config RetryConfig) tea.Cmd {
	return func() tea.Msg {
		rc := NewRetryCommand(cmd, config)
		return rc.Execute()()
	}
}

// RetryableOperation represents an operation that can be retried
type RetryableOperation[T any] struct {
	// Operation to execute
	Operation func() (T, error)

	// Retry configuration
	Config RetryConfig
}

// Execute executes the retryable operation
func (ro *RetryableOperation[T]) Execute() (T, error) {
	var result T
	var err error

	state := NewRetryState(ro.Config)

	for state.CanRetry() {
		result, err = ro.Operation()
		if err == nil {
			return result, nil
		}

		if !state.CanRetry() {
			break
		}

		// Wait before retrying
		delay := state.RecordAttempt(err)
		time.Sleep(delay)
	}

	return result, &RetryError{
		Err:        err,
		Attempt:    state.Attempt,
		MaxRetries: ro.Config.MaxRetries,
	}
}

// NewRetryableOperation creates a new retryable operation
func NewRetryableOperation[T any](operation func() (T, error), config RetryConfig) *RetryableOperation[T] {
	return &RetryableOperation[T]{
		Operation: operation,
		Config:    config,
	}
}
