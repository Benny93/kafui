package core

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func at(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease}
}

func TestDoubleClickNeedsSameSpotAndTime(t *testing.T) {
	now := time.Unix(0, 0)
	c := &ClickTracker{now: func() time.Time { return now }}

	assert.False(t, c.Click(at(5, 5)), "the first click is never a double")
	assert.True(t, c.Click(at(5, 5)), "a second click in place completes a double")

	// A third rapid click starts over rather than reading as another double.
	assert.False(t, c.Click(at(5, 5)))
}

func TestDoubleClickRejectsDifferentSpot(t *testing.T) {
	now := time.Unix(0, 0)
	c := &ClickTracker{now: func() time.Time { return now }}
	c.Click(at(5, 5))
	assert.False(t, c.Click(at(5, 6)), "a click on another row is a fresh selection")
}

func TestDoubleClickRejectsSlowSecondClick(t *testing.T) {
	now := time.Unix(0, 0)
	c := &ClickTracker{now: func() time.Time { return now }}
	c.Click(at(5, 5))
	now = now.Add(DoubleClickWindow + time.Millisecond)
	assert.False(t, c.Click(at(5, 5)), "past the window it is two single clicks")
}

func TestOnlyReleaseActs(t *testing.T) {
	assert.True(t, IsLeftRelease(at(1, 1)))
	assert.False(t, IsLeftRelease(tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}),
		"acting on press would make a drag fire the row under its start")
	assert.False(t, IsLeftRelease(tea.MouseMsg{Button: tea.MouseButtonRight, Action: tea.MouseActionRelease}))
}

func TestHoverIsMotionWithNoButton(t *testing.T) {
	assert.True(t, IsHover(tea.MouseMsg{Button: tea.MouseButtonNone, Action: tea.MouseActionMotion}))
	assert.False(t, IsHover(tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion}),
		"a drag is not hover")
}
