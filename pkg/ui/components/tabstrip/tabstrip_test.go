package tabstrip

import (
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initZones makes sure the whole package run shares a single bubblezone global
// manager. View() marks each tab with a zone id; clicks and hovers only mean
// something once a manager exists to answer InBounds().
var initZones sync.Once

func ensureZones(t *testing.T) {
	t.Helper()
	initZones.Do(func() { zone.NewGlobal() })
}

// waitZone polls until the global manager has registered the zone for strip
// tab i. The manager records scanned zones on a worker goroutine, so
// registration is asynchronous with respect to Scan.
func waitZone(t *testing.T, s *Model, i int) *zone.ZoneInfo {
	t.Helper()
	id := s.zoneID(i)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if z := zone.Get(id); z != nil {
			return z
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("zone %q was never registered", id)
	return nil
}

func TestNewSetsDefaults(t *testing.T) {
	s := New("tabs", []string{"One", "Two"})
	assert.Equal(t, "tabs", s.id)
	assert.Equal(t, []string{"One", "Two"}, s.titles)
	assert.Equal(t, 0, s.Active())
	assert.Equal(t, -1, s.hovered, "nothing is hovered until the mouse moves over a tab")
}

func TestActiveAndSetActive(t *testing.T) {
	s := New("tabs", []string{"One", "Two", "Three"})

	assert.Equal(t, 0, s.Active())
	s.SetActive(1)
	assert.Equal(t, 1, s.Active())

	// Out-of-range indices are ignored rather than clamping or panicking.
	s.SetActive(-1)
	assert.Equal(t, 1, s.Active(), "a negative index must be ignored")
	s.SetActive(3)
	assert.Equal(t, 1, s.Active(), "an index past the end must be ignored")
}

func TestSetTitlesReplacesTitlesAndClampsActive(t *testing.T) {
	s := New("tabs", []string{"One", "Two", "Three"})
	s.SetActive(2)

	s.SetTitles([]string{"Only"})
	assert.Equal(t, []string{"Only"}, s.titles)
	assert.Equal(t, 0, s.Active(), "the active index is clamped back into range")

	// A larger replacement keeps the active index untouched.
	s.SetActive(0)
	s.SetTitles([]string{"A", "B", "C", "D"})
	assert.Equal(t, 0, s.Active())
	assert.Len(t, s.titles, 4)
}

func TestZoneIDIncludesTheStripID(t *testing.T) {
	s := New("broker", nil)
	assert.Equal(t, "broker-tab-0", s.zoneID(0))
	assert.Equal(t, "broker-tab-1", s.zoneID(1))
}

func TestViewRendersOrdinalLabels(t *testing.T) {
	ensureZones(t)
	s := New("view", []string{"Overview", "Log dirs"})

	out := s.View()
	assert.Contains(t, out, "1 Overview")
	assert.Contains(t, out, "2 Log dirs")
	assert.Equal(t, 1, strings.Count(out, "1 Overview"), "each tab appears exactly once")
	assert.NotContains(t, out, "\n", "the strip renders on a single line")
}

func TestActiveTabStillRendersAfterSelection(t *testing.T) {
	ensureZones(t)
	s := New("view", []string{"Overview", "Log dirs"})
	s.SetActive(1)

	out := s.View()
	assert.Contains(t, out, "2 Log dirs")
	assert.Equal(t, 1, s.Active())
}

func TestViewIsSafeWithoutAZoneManager(t *testing.T) {
	saved := zone.DefaultManager
	zone.DefaultManager = nil
	defer func() { zone.DefaultManager = saved }()

	s := New("safe", []string{"One", "Two"})
	assert.NotPanics(t, func() {
		out := s.View()
		assert.Contains(t, out, "1 One")
		assert.Contains(t, out, "2 Two")
	}, "View must not panic when no zone manager has been set up")
}

func TestClickOnTabActivatesIt(t *testing.T) {
	ensureZones(t)
	s := New("click", []string{"One", "Two", "Three"})
	zone.Scan(s.View())

	z := waitZone(t, s, 1)
	msg := tea.MouseMsg{X: z.StartX + 1, Y: z.StartY, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease}

	clicked, hit := s.HandleMouse(msg)
	require.True(t, hit, "a left click inside a tab must be consumed")
	assert.Equal(t, 1, clicked)
	assert.Equal(t, 1, s.Active(), "clicking a tab makes it active")
}

func TestHoverAdvancesHighlightWithoutActivating(t *testing.T) {
	ensureZones(t)
	s := New("hover", []string{"One", "Two", "Three"})
	zone.Scan(s.View())

	z := waitZone(t, s, 1)
	msg := tea.MouseMsg{X: z.StartX + 1, Y: z.StartY, Button: tea.MouseButtonNone, Action: tea.MouseActionMotion}

	clicked, hit := s.HandleMouse(msg)
	assert.False(t, hit, "hovering highlights rather than selects")
	assert.Equal(t, 0, clicked)
	assert.Equal(t, 1, s.hovered)
	assert.Equal(t, 0, s.Active(), "hovering must not change the active tab")

	// Rendering after a hover still shows every tab, with the hovered one styled.
	assert.Contains(t, s.View(), "2 Two")
}

func TestClickOutsideAnyTabIsIgnored(t *testing.T) {
	ensureZones(t)
	s := New("out", []string{"One"})
	zone.Scan(s.View())

	z := waitZone(t, s, 0)
	msg := tea.MouseMsg{
		X:      z.EndX + 40,
		Y:      z.EndY + 5,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	}

	clicked, hit := s.HandleMouse(msg)
	assert.False(t, hit)
	assert.Equal(t, 0, clicked)
	assert.Equal(t, 0, s.Active())
}

func TestNonLeftButtonReleaseIsNotAClick(t *testing.T) {
	ensureZones(t)
	s := New("right", []string{"One"})
	zone.Scan(s.View())

	z := waitZone(t, s, 0)
	msg := tea.MouseMsg{X: z.StartX + 1, Y: z.StartY, Button: tea.MouseButtonRight, Action: tea.MouseActionRelease}

	clicked, hit := s.HandleMouse(msg)
	assert.False(t, hit, "a right-button release inside a tab must not activate it")
	assert.Equal(t, 0, clicked)
}
