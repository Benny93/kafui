package ksql

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/ui/keys"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
)

func (m *QueryModel) renderContent(width, height int) string {
	// ponytail: keyword syntax highlighting inside the textarea is deferred —
	// bubbles/textarea renders its own buffer, so lipgloss token styling would
	// require reimplementing its view; the spec marks highlighting best-effort.
	// ponytail: statement history (up/down recall) deferred — not required by
	// KS-12 and cheap to add later on top of the existing editor.
	m.resTable.SetWidth(width - 2) // -2 leaves room for the FrameTable border
	var b strings.Builder
	b.WriteString(m.common.Styles.Header.Render("Statement"))
	if m.running {
		b.WriteString("  " + m.common.Styles.StatusStyle.Info.Render("● streaming… (esc: abort)"))
	}
	b.WriteString("\n")
	b.WriteString(m.editor.View())
	b.WriteString("\n\n")

	b.WriteString(m.renderProps())
	b.WriteString("\n")

	b.WriteString(m.renderResults())
	b.WriteString("\n")
	b.WriteString(m.common.Styles.Muted.Render(
		keys.Hint(keys.ScopeTextEntry, keys.ActionRefresh, "run", keys.ActionFocusNext, "focus",
			keys.ActionClearField, "clear editor", keys.ActionCancel, "back") +
			"  (properties and clearing results are in the actions menu)"))
	return b.String()
}

func (m *QueryModel) renderResults() string {
	var b strings.Builder
	title := m.resTitle
	if title == "" {
		title = "Results"
	}
	b.WriteString(m.common.Styles.Header.Render(title))
	b.WriteString("\n")

	if m.errPanel != "" {
		b.WriteString(m.common.Styles.Error.Render("Error: " + m.errPanel))
		return b.String()
	}
	if m.placeholder {
		b.WriteString(m.common.Styles.Muted.Render("(no results)"))
		return b.String()
	}
	if !m.hasResult {
		b.WriteString(m.common.Styles.Muted.Render("Run a statement to see results."))
		return b.String()
	}
	b.WriteString(stylesPkg.FrameTable(m.resTable.View()))
	if m.truncated {
		b.WriteString("\n" + m.common.Styles.StatusStyle.Warning.Render(
			fmt.Sprintf("… showing last %d rows (older rows dropped)", maxResultRows)))
	}
	return b.String()
}

// --- helpers ---
