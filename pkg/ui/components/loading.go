package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Spinner is a thin, reusable wrapper around bubbles/spinner that gives pages a
// single shared loading-indicator mechanism instead of each page hand-rolling
// its own spinner state. Use NewSpinner + Tick to start it, forward
// spinner.TickMsg to Update, and CenteredLoading (or Frame) to render.
type Spinner struct {
	model spinner.Model
}

// NewSpinner returns a Spinner using the standard dot animation.
func NewSpinner() Spinner {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return Spinner{model: sp}
}

// Tick returns the command that starts (and keeps) the spinner animating.
func (s Spinner) Tick() tea.Cmd { return s.model.Tick }

// Update advances the animation on a spinner.TickMsg.
func (s Spinner) Update(msg tea.Msg) (Spinner, tea.Cmd) {
	var cmd tea.Cmd
	s.model, cmd = s.model.Update(msg)
	return s, cmd
}

// Frame returns the current animation frame.
func (s Spinner) Frame() string { return s.model.View() }

// CenteredLoading renders a spinner frame and label centered in the given box.
// When width/height are unknown (<=0) it falls back to a simple left-aligned line.
func CenteredLoading(frame, label string, width, height int) string {
	line := frame
	if label != "" {
		line = frame + " " + label
	}
	if width <= 0 || height <= 0 {
		return lipgloss.NewStyle().Padding(1).Render(line)
	}
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(line)
}
