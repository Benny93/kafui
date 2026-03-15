package core

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// StatusMessage represents a status message to be displayed
type StatusMessage struct {
	// Type of status message
	Type StatusType

	// Message to display
	Message string

	// TTL is the time-to-live for the message (0 = no auto-dismiss)
	TTL time.Duration

	// Timestamp when the message was created
	Timestamp time.Time
}

// NewStatusMessage creates a new status message
func NewStatusMessage(statusType StatusType, message string, ttl time.Duration) StatusMessage {
	return StatusMessage{
		Type:      statusType,
		Message:   message,
		TTL:       ttl,
		Timestamp: time.Now(),
	}
}

// IsExpired returns true if the message has expired based on its TTL
func (sm StatusMessage) IsExpired() bool {
	if sm.TTL == 0 {
		return false
	}
	return time.Since(sm.Timestamp) > sm.TTL
}

// Status messages for common operations

// NewInfoMsg creates an informational status message
func NewInfoMsg(message string) tea.Cmd {
	return func() tea.Msg {
		return NewStatusMessage(StatusInfo, message, 5*time.Second)
	}
}

// NewSuccessMsg creates a success status message
func NewSuccessMsg(message string) tea.Cmd {
	return func() tea.Msg {
		return NewStatusMessage(StatusSuccess, message, 5*time.Second)
	}
}

// NewWarningMsg creates a warning status message
func NewWarningMsg(message string) tea.Cmd {
	return func() tea.Msg {
		return NewStatusMessage(StatusWarning, message, 7*time.Second)
	}
}

// NewErrorMsg creates an error status message
func NewErrorMsg(message string) tea.Cmd {
	return func() tea.Msg {
		return NewStatusMessage(StatusError, message, 10*time.Second)
	}
}

// NewPermanentErrorMsg creates a permanent error status message (no auto-dismiss)
func NewPermanentErrorMsg(message string) tea.Cmd {
	return func() tea.Msg {
		return NewStatusMessage(StatusError, message, 0)
	}
}

// Status bar configuration

// StatusBarConfig holds configuration for the status bar
type StatusBarConfig struct {
	// ShowByDefault indicates whether status bar should be visible
	ShowByDefault bool

	// DefaultTTL is the default time-to-live for messages
	DefaultTTL time.Duration

	// MaxMessageLength is the maximum length of a message before truncation
	MaxMessageLength int
}

// DefaultStatusBarConfig returns the default status bar configuration
func DefaultStatusBarConfig() StatusBarConfig {
	return StatusBarConfig{
		ShowByDefault:    true,
		DefaultTTL:       5 * time.Second,
		MaxMessageLength: 100,
	}
}
