package core

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// DoubleClickWindow is how close together two clicks on the same spot must be
// to count as a double click. 400ms is the common desktop default; shorter and
// deliberate double clicks get missed on a slow terminal link.
const DoubleClickWindow = 400 * time.Millisecond

// ClickTracker classifies left clicks as single or double. The controls spec
// wants double click to activate, but a bare "click the already-selected row"
// rule cannot distinguish a user re-selecting a row from one opening it, so
// both gestures exist and this is what tells them apart.
//
// The zero value is ready to use. A tracker is per-pane: two panes tracking
// clicks independently is what stops a click in one and a click in the other
// from reading as a double click.
type ClickTracker struct {
	lastX, lastY int
	lastAt       time.Time
	// now is swappable so tests do not have to sleep.
	now func() time.Time
}

func (c *ClickTracker) clock() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

// Click records a left-click release and reports whether it completes a double
// click. A double click resets the tracker, so three rapid clicks are one
// single and one double rather than two doubles.
func (c *ClickTracker) Click(msg tea.MouseMsg) (double bool) {
	at := c.clock()
	sameSpot := msg.X == c.lastX && msg.Y == c.lastY
	inTime := !c.lastAt.IsZero() && at.Sub(c.lastAt) <= DoubleClickWindow

	if sameSpot && inTime {
		c.lastAt = time.Time{}
		return true
	}
	c.lastX, c.lastY, c.lastAt = msg.X, msg.Y, at
	return false
}

// IsLeftRelease reports whether msg is the release half of a left click, which
// is the only mouse event that should ever act. Acting on press makes a drag
// that starts on a row fire the row's action.
func IsLeftRelease(msg tea.MouseMsg) bool {
	return msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionRelease
}

// IsHover reports whether msg is pointer motion with no button held, which is
// what drives hover highlighting.
func IsHover(msg tea.MouseMsg) bool {
	return msg.Button == tea.MouseButtonNone && msg.Action == tea.MouseActionMotion
}
