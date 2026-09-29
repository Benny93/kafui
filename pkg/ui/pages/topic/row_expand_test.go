package topic

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Benny93/kafui/pkg/api"
)

// expandModel returns a model holding n messages whose values are JSON
// objects with fields fields each.
func expandModel(t *testing.T, n, fields int) *Model {
	t.Helper()
	zone.NewGlobal() // RenderContent zone-marks the table
	m := newLifecycleModel()
	for i := 0; i < n; i++ {
		var b strings.Builder
		b.WriteString("{")
		for f := 0; f < fields; f++ {
			if f > 0 {
				b.WriteString(",")
			}
			fmt.Fprintf(&b, `"field%d":"value %d/%d"`, f, i, f)
		}
		b.WriteString("}")
		m.AddMessage(api.Message{Partition: 0, Offset: int64(i), Key: fmt.Sprintf("k%d", i), Value: b.String()})
	}
	m.FilterMessages()
	return m
}

func render(m *Model, w, h int) string {
	return NewTopicContentProvider(m).RenderContent(w, h)
}

func TestExpandShowsFullValueInline(t *testing.T) {
	m := expandModel(t, 5, 4)
	before := render(m, 120, 40)
	sel := m.GetSelectedMessage()
	require.NotNil(t, sel)
	assert.NotContains(t, before, `"field3": "value`, "collapsed rows only show a truncated value")

	m.toggleExpand()
	out := render(m, 120, 40)
	// Pretty-printed, one field per line, right under the highlighted row.
	assert.Contains(t, out, `"field3": "value`)
	lines := strings.Split(out, "\n")
	row := -1
	for i, l := range lines {
		if strings.Contains(l, " "+sel.Key+" ") {
			row = i
			break
		}
	}
	require.GreaterOrEqual(t, row, 0)
	assert.Contains(t, lines[row+1], "╭", "the panel starts on the line after the row")

	m.toggleExpand()
	assert.NotContains(t, render(m, 120, 40), "╭")
}

// The panel follows the cursor and never pushes the layout past its height.
func TestExpandedLayoutFitsAndFollowsCursor(t *testing.T) {
	for _, size := range [][2]int{{80, 20}, {120, 40}, {160, 60}} {
		m := expandModel(t, 60, 80) // values far taller than any panel
		m.toggleExpand()
		collapsedHeight := lipgloss.Height(render(newLifecycleModelWith(m), size[0], size[1]))
		for _, cursor := range []int{0, 3, 10} {
			t.Run(fmt.Sprintf("%dx%d/cursor%d", size[0], size[1], cursor), func(t *testing.T) {
				page := len(m.pagination.GetVisibleMessages(m.filteredMessages))
				if cursor >= page {
					t.Skip("page smaller than cursor")
				}
				m.cursorRow = cursor
				out := render(m, size[0], size[1])
				assert.LessOrEqual(t, lipgloss.Height(out), collapsedHeight, "expanding must not grow the table")
				assert.LessOrEqual(t, lipgloss.Width(out), size[0])
				sel := m.GetSelectedMessage()
				require.NotNil(t, sel)
				assert.Contains(t, out, fmt.Sprintf("p0 @ %d", sel.Offset), "panel shows the highlighted message")
				assert.GreaterOrEqual(t, cursor, m.tableLayout.start)
				assert.Less(t, cursor, m.tableLayout.start+m.tableLayout.shown, "cursor row stays drawn")
			})
		}
	}
}

// newLifecycleModelWith copies m's messages into a collapsed model, to measure
// the height of the same table without a panel.
func newLifecycleModelWith(m *Model) *Model {
	c := newLifecycleModel()
	for _, msg := range m.messages {
		c.AddMessage(msg)
	}
	c.FilterMessages()
	return c
}

func TestExpandedContentScrolls(t *testing.T) {
	m := expandModel(t, 3, 200)
	m.toggleExpand()
	// Pretty-printed fields ("key": value) appear only in the panel; the
	// collapsed rows show the compact JSON.
	out := render(m, 120, 30)
	sel := m.GetSelectedMessage() // newest first: the highest offset
	require.NotNil(t, sel)
	assert.Contains(t, out, fmt.Sprintf(`"field0": "value %d/0"`, sel.Offset))
	assert.Contains(t, out, "lines 1-")
	assert.NotContains(t, out, `"field199": `)

	for i := 0; i < 300; i++ {
		m.scrollExpanded(1)
	}
	out = render(m, 120, 30)
	assert.Contains(t, out, `"field199": `, "scrolled to the end")
	assert.NotContains(t, out, `"field0": `)

	// Moving to another message starts it at the top.
	m.cursorRow = 1
	next := m.GetSelectedMessage()
	require.NotNil(t, next)
	assert.Contains(t, render(m, 120, 30), fmt.Sprintf(`"field0": "value %d/0"`, next.Offset))
}

func TestExpandKeys(t *testing.T) {
	m := expandModel(t, 3, 200)
	k := NewKeys()
	k.HandleKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	require.True(t, m.expanded)
	render(m, 120, 30)
	k.HandleKey(m, tea.KeyMsg{Type: tea.KeyShiftDown})
	top, _, _ := m.expandViewer.ScrollInfo()
	assert.Equal(t, 1, top)
	k.HandleKey(m, tea.KeyMsg{Type: tea.KeyShiftUp})
	top, _, _ = m.expandViewer.ScrollInfo()
	assert.Equal(t, 0, top)
	k.HandleKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	assert.False(t, m.expanded)
}

func TestLineTarget(t *testing.T) {
	l := tableLayout{firstRow: 4, start: 2, shown: 3, panelAfter: 3, panelLines: 5}
	for line, want := range map[int][2]int{ // line -> {row, inPanel}
		0: {-1, 0}, 3: {-1, 0},
		4: {2, 0}, 5: {3, 0},
		6: {-1, 1}, 10: {-1, 1},
		11: {4, 0}, 12: {-1, 0},
	} {
		row, in := l.lineTarget(line)
		assert.Equal(t, want[0], row, "line %d", line)
		assert.Equal(t, want[1] == 1, in, "line %d", line)
	}
}

func TestWindowStart(t *testing.T) {
	assert.Equal(t, 0, windowStart(0, 5, 10, 10), "everything fits")
	assert.Equal(t, 0, windowStart(0, 2, 3, 10))
	assert.Equal(t, 3, windowStart(0, 5, 3, 10), "scrolls down to keep the cursor")
	assert.Equal(t, 3, windowStart(3, 4, 3, 10), "stays put while the cursor is inside")
	assert.Equal(t, 1, windowStart(3, 1, 3, 10), "scrolls up")
	assert.Equal(t, 7, windowStart(9, 9, 3, 10), "clamped to the end")
}

// A click lands on the row drawn under the pointer, collapsed and expanded,
// through a real zone scan.
func TestClickSelectsRowUnderPointer(t *testing.T) {
	for _, expanded := range []bool{false, true} {
		m := expandModel(t, 10, 3)
		m.expanded = expanded
		out := zone.Scan(render(m, 120, 40))
		lines := strings.Split(out, "\n")
		target := -1
		for i, l := range lines {
			if strings.Contains(lipglossStrip(l), " k4 ") {
				target = i
			}
		}
		require.GreaterOrEqual(t, target, 0)
		require.Eventually(t, func() bool { return zone.Get("message-table").InBounds(click(5, target)) }, time.Second, 10*time.Millisecond)

		h := NewHandlers(m)
		h.handleMouseMsg(m, click(5, target))
		sel := m.GetSelectedMessage()
		require.NotNil(t, sel)
		assert.Equal(t, "k4", sel.Key, "expanded=%v", expanded)
	}
}

func click(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease}
}

func lipglossStrip(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEsc = true
		case inEsc && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'):
			inEsc = false
		case !inEsc:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// The table uses the height it is given, with and without the search bar,
// collapsed and expanded: nothing overflows and no rows are left blank.
func TestTableFillsHeight(t *testing.T) {
	for _, height := range []int{20, 36, 56} {
		for _, search := range []bool{false, true} {
			for _, expanded := range []bool{false, true} {
				t.Run(fmt.Sprintf("h%d/search=%v/expanded=%v", height, search, expanded), func(t *testing.T) {
					m := expandModel(t, 200, 30)
					m.searchMode = search
					m.expanded = expanded
					got := lipgloss.Height(render(m, 140, height))
					assert.LessOrEqual(t, got, height)
					assert.GreaterOrEqual(t, got, height-1)
				})
			}
		}
	}
}
