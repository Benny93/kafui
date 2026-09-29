package messagedetail

import (
	"fmt"
	"strings"
	"time"

	"github.com/evertras/bubble-table/table"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components/editor"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

// RenderContent renders the tab bar and the active tab within exactly
// width x height, so the template's frame is never pushed out of view.
func (m *MessageDetailContentProvider) RenderContent(width, height int) string {
	if m.model == nil {
		return "No message data available"
	}

	m.width = width
	m.height = height
	m.updateEditorContent()
	m.sizeEditors()

	// Clear expired status messages
	if m.model.statusMsg != "" && time.Since(m.model.statusTime) > 3*time.Second {
		m.model.statusMsg = ""
	}

	// Tab bar (1) + blank (1), then the tab body in the remaining height.
	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderTabBar(width),
		"",
		m.renderActiveTabContent(),
	)
}

// renderTabBar renders the shared tab strip, with a transient status message
// (e.g. "copied") right-aligned on the same line so it costs no height.
func (m *MessageDetailContentProvider) renderTabBar(width int) string {
	m.tabStrip.SetActive(m.activeTab)
	bar := m.tabStrip.View()
	if m.model.statusMsg == "" {
		return bar
	}
	status := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted).Render(m.model.statusMsg)
	gap := width - lipgloss.Width(bar) - lipgloss.Width(status)
	if gap < 2 {
		return bar
	}
	return bar + strings.Repeat(" ", gap) + status
}

// renderActiveTabContent renders the content for the currently active tab.
func (m *MessageDetailContentProvider) renderActiveTabContent() string {
	switch m.activeTab {
	case 0:
		return m.renderContentTab()
	case 1:
		return lipgloss.JoinVertical(lipgloss.Left,
			m.renderSelectedKeyLabel(m.headersTable, colHdrKey),
			m.headersTable.View(),
		)
	case 2:
		return lipgloss.JoinVertical(lipgloss.Left,
			m.renderSelectedKeyLabel(m.metadataTable, colMdName),
			m.metadataTable.View(),
		)
	}
	return "Unknown tab"
}

// renderSelectedKeyLabel returns a styled one-line label showing the full key
// of the currently highlighted row in the given table. If no row is selected
// or the row has no value for keyCol, an empty string is returned so the
// layout doesn't shift.
func (m *MessageDetailContentProvider) renderSelectedKeyLabel(t table.Model, keyCol string) string {
	rows := t.GetVisibleRows()
	idx := t.GetHighlightedRowIndex()
	if idx < 0 || idx >= len(rows) {
		return ""
	}
	raw := rows[idx].Data[keyCol]
	if raw == nil {
		return ""
	}
	key := fmt.Sprintf("%v", raw)
	if key == "" {
		return ""
	}
	label := lipgloss.NewStyle().
		Foreground(stylesPkg.Primary).
		Bold(true).
		Padding(0, 1).
		Render("▶ " + key)
	return label
}

// renderContentTab stacks the key pane (sized to the key, usually one line)
// above the value pane, which gets the full width and the remaining height.
// Each pane is labelled with its schema, and the focused pane's border is
// highlighted.
func (m *MessageDetailContentProvider) renderContentTab() string {
	keyBorder, valueBorder := blurredBorderStyle, focusedBorderStyle
	if m.focusedEditor == 0 {
		keyBorder, valueBorder = focusedBorderStyle, blurredBorderStyle
	}
	var keySchema, valueSchema *api.SchemaInfo
	if info := m.model.GetSchemaInfo(); info != nil {
		keySchema, valueSchema = info.KeySchema, info.ValueSchema
	}
	maxSubject := m.width - 20
	return lipgloss.JoinVertical(lipgloss.Left,
		paneLabel("Key", keySchema, maxSubject, m.focusedEditor == 0),
		keyBorder.Render(m.keyEditor.View()),
		paneLabel("Value", valueSchema, maxSubject, m.focusedEditor == 1),
		valueBorder.Render(m.valueEditor.View()),
	)
}

// paneLabel renders "Key · schema <name>" above a pane, bold when focused.
func paneLabel(name string, schema *api.SchemaInfo, maxSubject int, focused bool) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(stylesPkg.FgMuted)
	if focused {
		title = title.Foreground(stylesPkg.Primary)
	}
	detail := "no schema"
	if d := schemaDisplayName(schema, max(maxSubject, 10)); d != "" {
		detail = "schema " + d
	}
	return " " + title.Render(name) + lipgloss.NewStyle().Foreground(stylesPkg.FgSubtle).Italic(true).Render(" · "+detail)
}

// schemaDisplayName returns the best human-readable label for a SchemaInfo.
// Priority: Avro record name (from the schema "name" field) > subject name.
// Long strings are truncated from the left so the most specific suffix stays visible.
func schemaDisplayName(s *api.SchemaInfo, maxLen int) string {
	if s == nil {
		return ""
	}
	if s.RecordName != "" && s.RecordName != "Unknown" {
		return truncateSubjectLeft(s.RecordName, maxLen)
	}
	if s.Subject != "" {
		return truncateSubjectLeft(s.Subject, maxLen)
	}
	return ""
}

// truncateSubjectLeft truncates s from the beginning when it exceeds maxLen,
// replacing the removed prefix with "...". This keeps the most specific
// (rightmost) part of a dotted schema subject name visible.
func truncateSubjectLeft(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	const ellipsis = "..."
	if maxLen <= len(ellipsis) {
		return ellipsis
	}
	return ellipsis + s[len(s)-(maxLen-len(ellipsis)):]
}

// sizeEditors fits the active tab into m.width x m.height, less the tab bar
// and blank line above it.
func (m *MessageDetailContentProvider) sizeEditors() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	bodyHeight := m.height - 2

	switch m.activeTab {
	case 0: // Content tab - key pane stacked above the value pane
		// Each pane costs a label line and a top and bottom border.
		const paneChrome = 3
		inner := bodyHeight - 2*paneChrome
		paneWidth := max(m.width-2, 10)

		// The key pane shows the whole key when it is short, and at most a
		// quarter of the space otherwise; the value pane takes the rest.
		keyLines := strings.Count(m.model.GetFormattedKey(), "\n") + 1
		if m.keyEditor.HasStatusLine() {
			keyLines++ // grow the pane for the search prompt instead of hiding the key
		}
		keyH := min(keyLines, max(inner/4, 2))
		valueH := max(inner-keyH, 3)
		m.keyEditor.SetDimensions(paneWidth, viewportHeight(m.keyEditor, keyH))
		m.valueEditor.SetDimensions(paneWidth, viewportHeight(m.valueEditor, valueH))

	case 1: // Headers tab — resize the value column to fill available width
		const hdrKeyWidth = 30
		hdrValWidth := m.width - 4 - hdrKeyWidth // 3 column rules + 1 slack
		if hdrValWidth < 20 {
			hdrValWidth = 20
		}
		m.headersTable = m.headersTable.WithColumns([]table.Column{
			table.NewColumn(colHdrKey, "Key", hdrKeyWidth),
			table.NewColumn(colHdrVal, "Value", hdrValWidth),
		})

	case 2: // Metadata tab — resize the value column to fill available width
		const nameColWidth = 20
		valueColWidth := m.width - 4 - nameColWidth // 3 column rules + 1 slack
		if valueColWidth < 20 {
			valueColWidth = 20
		}
		m.metadataTable = m.metadataTable.WithColumns([]table.Column{
			table.NewColumn(colMdName, "Name", nameColWidth),
			table.NewColumn(colMdValue, "Value", valueColWidth),
		})
	}
}

// viewportHeight is the viewport height that makes v render exactly h lines,
// leaving room for its search status line when one is showing.
func viewportHeight(v *editor.Viewer, h int) int {
	if v.HasStatusLine() && h > 1 {
		return h - 1
	}
	return h
}

var (
	highlightColor = stylesPkg.Primary

	// Pane border styles
	focusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(highlightColor)
	blurredBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(stylesPkg.FgSubtle)
)
