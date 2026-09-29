package topic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components/editor"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
)

// Inline row expansion: `x` toggles a panel under the highlighted row that
// shows the message's full, pretty-printed value, so its content can be read
// without opening the detail page. The panel follows the cursor; content
// taller than the panel scrolls (shift+↑/↓, or the wheel over the panel).

const (
	// panelChrome is the panel's border (2) plus its header line (1).
	panelChrome = 3
	// minRowsWithPanel is how many table rows stay visible beside the panel.
	minRowsWithPanel = 3
)

// tableLayout records where the last render placed things, in lines relative
// to the table's first line, so mouse events can be mapped back to rows.
type tableLayout struct {
	firstRow   int // line of the first drawn row
	start      int // page index of the first drawn row
	shown      int // rows drawn
	panelAfter int // page index the panel follows, or -1
	panelLines int // lines the panel occupies
}

// lineTarget maps a line of the rendered table to the page row drawn there,
// or reports that the line is inside the expanded panel. row is -1 for
// header, footer and panel lines.
func (l tableLayout) lineTarget(line int) (row int, inPanel bool) {
	rel := line - l.firstRow
	if rel < 0 {
		return -1, false
	}
	for idx := l.start; idx < l.start+l.shown; idx++ {
		if rel == 0 {
			return idx, false
		}
		rel--
		if idx == l.panelAfter {
			if rel < l.panelLines {
				return -1, true
			}
			rel -= l.panelLines
		}
	}
	return -1, false
}

// toggleExpand shows or hides the inline panel.
func (m *Model) toggleExpand() {
	m.expanded = !m.expanded
	m.markRenderDirty()
}

// scrollExpanded scrolls the inline panel by n lines.
func (m *Model) scrollExpanded(n int) {
	if m.expanded && m.expandViewer != nil {
		m.expandViewer.ScrollBy(n)
		m.markRenderDirty()
	}
}

// loadExpanded points the panel's viewer at msg, keeping the scroll position
// while the same message stays highlighted.
func (m *Model) loadExpanded(msg api.Message) {
	if m.expandViewer == nil {
		m.expandViewer = editor.NewViewer("")
		m.expandViewer.SetWrap(true) // nothing scrolls horizontally
	}
	id := fmt.Sprintf("%d/%d/%s", msg.Partition, msg.Offset, m.displayFormat())
	if id == m.expandFor {
		return
	}
	body, isJSON := expandedBody(m.displayValue(msg))
	m.expandViewer.SetHighlight(isJSON)
	m.expandViewer.SetContent(body)
	m.expandViewer.ScrollToTop()
	m.expandFor = id
}

// displayFormat identifies the current value rendering, so a format change
// reloads the panel.
func (m *Model) displayFormat() string {
	return fmt.Sprintf("%v", m.valueSerde)
}

// expandedBody returns value ready for the panel: pretty-printed when it is
// JSON, with control characters other than newlines removed.
func expandedBody(value string) (body string, isJSON bool) {
	value = sanitizeKeepNewlines(value)
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "(empty value)", false
	}
	if json.Valid([]byte(trimmed)) && (trimmed[0] == '{' || trimmed[0] == '[') {
		var buf bytes.Buffer
		if json.Indent(&buf, []byte(trimmed), "", "  ") == nil {
			return buf.String(), true
		}
	}
	return value, false
}

// sanitizeKeepNewlines is sanitizeForDisplay for multi-line content: newlines
// survive, tabs become two spaces, and other control characters are dropped.
func sanitizeKeepNewlines(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n':
			b.WriteRune(r)
		case r == '\t':
			b.WriteString("  ")
		case unicode.IsControl(r):
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// panelInnerHeight sizes the viewer for msg within a table that has budget
// row lines: as tall as the content, but leaving minRowsWithPanel rows. It
// returns 0 when there is no room for a panel.
func (m *Model) panelInnerHeight(width, budget int) int {
	maxInner := budget - minRowsWithPanel - panelChrome
	if maxInner < 1 {
		return 0
	}
	// Measure at the panel's width first; wrapping depends on it.
	m.expandViewer.SetDimensions(panelWidth(width), maxInner)
	inner := min(max(m.expandViewer.LineCount(), 1), maxInner)
	m.expandViewer.SetDimensions(panelWidth(width), inner)
	return inner
}

// panelWidth is the viewer width inside the panel's border.
func panelWidth(tableWidth int) int { return max(tableWidth-4, 10) }

// renderExpandPanel draws the bordered panel: a header naming the message and
// the scroll position, then the viewer.
func (m *Model) renderExpandPanel(msg api.Message, width int) string {
	muted := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)
	header := fmt.Sprintf("p%d @ %d · %s", msg.Partition, msg.Offset,
		shared.FormatBytes2dp(int64(len(msg.Value))))
	if len(msg.Headers) > 0 {
		header += fmt.Sprintf(" · %d headers", len(msg.Headers))
	}
	top, visible, total := m.expandViewer.ScrollInfo()
	if total > visible {
		header += fmt.Sprintf(" · lines %d-%d of %d · shift+↑/↓ scroll", top+1, min(top+visible, total), total)
	}
	header += " · c copy · enter details · x close"
	header = truncateString(header, panelWidth(width))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(stylesPkg.Primary).
		Padding(0, 1).
		Width(width - 2).
		Render(muted.Render(header) + "\n" + m.expandViewer.View())
}
