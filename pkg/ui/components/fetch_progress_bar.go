package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ProgressMsg reports progress of a counted operation.
// When Done is true the operation has finished; the caller may deliver the
// final result payload through a separate channel/message.
type ProgressMsg struct {
	Current int
	Total   int
	Done    bool
}

// ProgressBarFrameMsg is a re-export of the underlying animation frame message.
// Callers should forward this type to FetchProgressBar.Update so the spring
// animation keeps running. This avoids callers importing bubbles/progress directly.
type ProgressBarFrameMsg = progress.FrameMsg

// NewProgressChannel creates a properly buffered channel for progress updates.
func NewProgressChannel(total int) chan ProgressMsg {
	buf := total + 10
	if buf < 10 {
		buf = 10
	}
	return make(chan ProgressMsg, buf)
}

// ListenForProgress returns a Cmd that blocks until the next ProgressMsg
// arrives on ch, then delivers it to the Bubble Tea Update loop.
// Chain this inside your Update handler to receive a continuous stream.
func ListenForProgress(ch <-chan ProgressMsg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// FetchProgressBar is a reusable, self-contained animated progress bar
// component for tracking counted fetch / load operations.
//
// Typical usage:
//
//	// 1. Embed in your model:
//	type Model struct { bar components.FetchProgressBar }
//
//	// 2. Start when the operation begins:
//	ch := components.NewProgressChannel(total)
//	cmds = append(cmds, model.bar.StartListening(ch, total))
//
//	// 3. Forward messages in Update:
//	model.bar, cmd = model.bar.Update(msg)
//
//	// 4. Render in View:
//	if model.bar.IsActive() { content += model.bar.View(width) }
type FetchProgressBar struct {
	bar     progress.Model
	current int
	total   int
	active  bool
	ch      <-chan ProgressMsg
}

// NewFetchProgressBar creates a FetchProgressBar with a default gradient style.
func NewFetchProgressBar() FetchProgressBar {
	pb := progress.New(
		progress.WithDefaultGradient(),
		progress.WithoutPercentage(),
	)
	return FetchProgressBar{bar: pb}
}

// StartListening initialises the component for a new operation, stores the
// progress channel and returns the initial Cmds (reset animation + first
// listener). Call this once when the fetch begins.
func (f *FetchProgressBar) StartListening(ch <-chan ProgressMsg, total int) tea.Cmd {
	f.ch = ch
	f.total = total
	f.current = 0
	f.active = true
	return tea.Batch(
		f.bar.SetPercent(0),
		ListenForProgress(ch),
	)
}

// Update handles ProgressMsg (progress updates and completion) and
// progress.FrameMsg (spring animation ticks). Pass all incoming messages
// to this method from your parent component's Update.
func (f FetchProgressBar) Update(msg tea.Msg) (FetchProgressBar, tea.Cmd) {
	switch msg := msg.(type) {
	case ProgressMsg:
		if msg.Done {
			f.current = f.total
			f.active = false
			f.ch = nil
			return f, f.bar.SetPercent(1)
		}
		f.current = msg.Current
		var pct float64
		if f.total > 0 {
			pct = float64(msg.Current) / float64(f.total)
		}
		return f, tea.Batch(f.bar.SetPercent(pct), ListenForProgress(f.ch))

	case progress.FrameMsg:
		updatedBar, cmd := f.bar.Update(msg)
		f.bar = updatedBar.(progress.Model)
		return f, cmd
	}
	return f, nil
}

// View renders the progress bar at the given width.
// Returns an empty string when the component is not active and has never run.
func (f FetchProgressBar) View(width int) string {
	if !f.active && f.total == 0 {
		return ""
	}
	barWidth := width - 4
	if barWidth < 10 {
		barWidth = 10
	}
	f.bar.Width = barWidth

	pct := 0
	if f.total > 0 {
		pct = int(float64(f.current) / float64(f.total) * 100)
	}
	label := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Render(fmt.Sprintf("%d%%", pct))

	return label + "\n" + f.bar.View()
}

// IsActive returns true while an operation is in progress.
func (f FetchProgressBar) IsActive() bool { return f.active }

// Current returns the number of items received so far.
func (f FetchProgressBar) Current() int { return f.current }

// Total returns the target item count.
func (f FetchProgressBar) Total() int { return f.total }
