package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

// The bug this replaced: the overlay wrote its line over the WHOLE terminal
// row, so the sidebar and the table's right-hand columns disappeared on every
// row the box covered.
func TestCompositeKeepsTheBaseOutsideTheOverlay(t *testing.T) {
	row := "LEFT......................RIGHT"
	base := strings.Join([]string{row, row, row, row, row, row, row}, "\n")
	over := "XXXXX\nXXXXX"

	out := composite(base, over, 31, 7)
	lines := strings.Split(out, "\n")

	covered := 0
	for _, l := range lines {
		if !strings.Contains(l, "XXXXX") {
			continue
		}
		covered++
		assert.True(t, strings.HasPrefix(l, "LEFT"), "the base left of the overlay must survive: %q", l)
		assert.True(t, strings.HasSuffix(l, "RIGHT"), "the base right of the overlay must survive: %q", l)
	}
	assert.Equal(t, 2, covered, "both overlay rows must be drawn")

	for _, l := range lines {
		assert.Equal(t, 31, lipgloss.Width(l), "compositing must not change the row width: %q", l)
	}
}

func TestCompositeCentresTheOverlay(t *testing.T) {
	base := strings.TrimRight(strings.Repeat("...........\n", 5), "\n")
	out := composite(base, "XXX", 11, 5)
	lines := strings.Split(out, "\n")
	// The overlay carries a one-cell gutter, so a 3-wide box occupies 5 cells
	// centred on 11: columns 3..7.
	assert.Equal(t, "... XXX ...", lines[2], "the box, with its gutter, must be centred")
	assert.Equal(t, "...     ...", lines[1], "the gutter blanks the row above")
}

// Styled base content either side of the overlay must keep its colours. The
// escapes are written literally rather than through lipgloss, because lipgloss
// strips colour when it detects no TTY — as it does under `go test`.
func TestCompositePreservesStylingOutsideTheOverlay(t *testing.T) {
	const red, reset = "\x1b[31m", "\x1b[0m"
	base := red + "AAAA" + reset + "....." + red + "BBBB" + reset
	out := composite(base, "XX", lipgloss.Width(base), 1)

	assert.Contains(t, out, "XX")
	assert.Contains(t, out, "AAAA")
	assert.Contains(t, out, "BBBB")
	assert.Contains(t, out, red, "the base's colour either side of the overlay must survive")
	assert.Equal(t, lipgloss.Width(base), lipgloss.Width(out),
		"compositing must not change the visible width")
}

// An overlay taller than the base must not be silently clipped.
func TestCompositeGrowsForATallOverlay(t *testing.T) {
	out := composite("......", "XX\nXX\nXX", 6, 1)
	// Three rows plus the gutter above and below.
	assert.Equal(t, 5, len(strings.Split(out, "\n")))
}

func TestCompositeIgnoresAnEmptyOverlay(t *testing.T) {
	assert.Equal(t, "abc", composite("abc", "", 3, 1))
}

// A ragged overlay — lines of differing width — must still consume exactly the
// same span on every row, or the base loses cells on the short ones. That is
// what made the sidebar lose its border and first character.
func TestCompositeHandlesARaggedOverlay(t *testing.T) {
	row := "LEFT................RIGHT"
	base := strings.Join([]string{row, row, row, row, row, row}, "\n")
	// Second line is shorter than the first.
	over := "XXXXX\nXX"

	out := composite(base, over, 25, 6)
	for i, l := range strings.Split(out, "\n") {
		assert.Equal(t, 25, lipgloss.Width(l), "row %d width changed: %q", i, l)
		assert.True(t, strings.HasSuffix(l, "RIGHT"),
			"the base right of the overlay must survive on row %d: %q", i, l)
		assert.True(t, strings.HasPrefix(l, "LEFT"),
			"the base left of the overlay must survive on row %d: %q", i, l)
	}
}
