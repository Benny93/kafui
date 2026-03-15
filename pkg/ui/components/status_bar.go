package components

import (
	"time"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StatusBar represents a status bar component for displaying messages
type StatusBar struct {
	// Current status message
	message core.StatusMessage

	// Whether the status bar is visible
	visible bool

	// Spinner for loading states
	spinner spinner.Model

	// Styles
	styles StatusBarStyles

	// Configuration
	config core.StatusBarConfig
}

// StatusBarStyles contains styles for the status bar
type StatusBarStyles struct {
	Container lipgloss.Style
	Info      lipgloss.Style
	Success   lipgloss.Style
	Warning   lipgloss.Style
	Error     lipgloss.Style
	Spinner   lipgloss.Style
}

// DefaultStatusBarStyles returns the default status bar styles
func DefaultStatusBarStyles() StatusBarStyles {
	return StatusBarStyles{
		Container: lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1),
		Info: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")),
		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color("34")),
		Warning: lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
		Spinner: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")),
	}
}

// NewStatusBar creates a new status bar component
func NewStatusBar(config core.StatusBarConfig) *StatusBar {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &StatusBar{
		visible: config.ShowByDefault,
		spinner: sp,
		styles:  DefaultStatusBarStyles(),
		config:  config,
	}
}

// Init initializes the status bar
func (sb *StatusBar) Init() tea.Cmd {
	return nil
}

// Update handles messages for the status bar
func (sb *StatusBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case core.StatusMessage:
		// Set new status message
		sb.message = msg
		sb.visible = true
		return sb, nil

	case StatusBarClearMsg:
		// Clear the status message
		sb.message = core.StatusMessage{}
		sb.visible = false
		return sb, nil

	case spinner.TickMsg:
		// Update spinner
		var cmd tea.Cmd
		sb.spinner, cmd = sb.spinner.Update(msg)
		return sb, cmd
	}

	return sb, nil
}

// View renders the status bar
func (sb *StatusBar) View() string {
	if !sb.visible {
		return ""
	}

	if sb.message.Message == "" {
		return ""
	}

	// Check if message has expired
	if sb.message.IsExpired() {
		return ""
	}

	// Get style based on message type
	var style lipgloss.Style
	var prefix string

	switch sb.message.Type {
	case core.StatusInfo:
		style = sb.styles.Info
		prefix = "ℹ "
	case core.StatusSuccess:
		style = sb.styles.Success
		prefix = "✓ "
	case core.StatusWarning:
		style = sb.styles.Warning
		prefix = "⚠ "
	case core.StatusError:
		style = sb.styles.Error
		prefix = "✗ "
	}

	// Truncate message if needed
	message := sb.message.Message
	maxLen := sb.config.MaxMessageLength
	if maxLen > 0 && len(message) > maxLen {
		message = message[:maxLen-3] + "..."
	}

	// Render the status message
	content := prefix + style.Render(message)

	return sb.styles.Container.Render(content)
}

// SetMessage sets a new status message
func (sb *StatusBar) SetMessage(msg core.StatusMessage) {
	sb.message = msg
	sb.visible = true
}

// Clear clears the status message
func (sb *StatusBar) Clear() {
	sb.message = core.StatusMessage{}
	sb.visible = false
}

// SetVisible sets the visibility of the status bar
func (sb *StatusBar) SetVisible(visible bool) {
	sb.visible = visible
}

// IsVisible returns whether the status bar is visible
func (sb *StatusBar) IsVisible() bool {
	return sb.visible && !sb.message.IsExpired()
}

// GetWidth returns the width of the status bar
func (sb *StatusBar) GetWidth() int {
	return 0 // Full width
}

// SetWidth sets the width of the status bar
func (sb *StatusBar) SetWidth(width int) {
	// Status bar is full width, width parameter ignored
}

// SetStyles sets the status bar styles
func (sb *StatusBar) SetStyles(styles StatusBarStyles) {
	sb.styles = styles
}

// StatusBarClearMsg is a message to clear the status bar
type StatusBarClearMsg struct{}

// ClearStatusBar creates a command to clear the status bar
func ClearStatusBar() tea.Cmd {
	return func() tea.Msg {
		return StatusBarClearMsg{}
	}
}

// AutoClearStatus creates a command that clears the status bar after a delay
func AutoClearStatus(delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return StatusBarClearMsg{}
	})
}
