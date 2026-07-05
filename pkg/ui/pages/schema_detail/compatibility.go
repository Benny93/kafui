package schemadetail

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// enterPicker opens the compatibility-level selector, highlighting the current
// effective level (SR-19).
func (m *Model) enterPicker() {
	m.mode = modePicker
	m.pickerCursor = 0
	for i, level := range api.CompatibilityLevels() {
		if level == m.compat {
			m.pickerCursor = i
			break
		}
	}
}

// SelectedLevel returns the compatibility level under the picker cursor.
func (m *Model) SelectedLevel() api.CompatibilityLevel {
	levels := api.CompatibilityLevels()
	if m.pickerCursor < 0 || m.pickerCursor >= len(levels) {
		return ""
	}
	return levels[m.pickerCursor]
}

// confirmSetCompatCmd asks for confirmation, then sets the subject's level
// (SR-19). The datasource call runs only after confirmation.
func (m *Model) confirmSetCompatCmd(level api.CompatibilityLevel) tea.Cmd {
	subject, ds := m.subject, m.dataSource
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Set compatibility",
			Message:      fmt.Sprintf("Set compatibility of %q to %s?", subject, level),
			ConfirmLabel: "Set",
			OnConfirm: func() tea.Msg {
				err := ds.SetSubjectCompatibility(subject, level)
				return SchemaCompatSetResultMsg{Level: level, Err: err}
			},
		}
	}
}

func handlePickerKey(m *Model, msg tea.KeyMsg) tea.Cmd {
	levels := api.CompatibilityLevels()
	switch msg.String() {
	case "up", "k":
		if m.pickerCursor > 0 {
			m.pickerCursor--
		}
	case "down", "j":
		if m.pickerCursor < len(levels)-1 {
			m.pickerCursor++
		}
	case "enter":
		if level := m.SelectedLevel(); level != "" {
			m.mode = modeContent
			return m.confirmSetCompatCmd(level)
		}
	case "esc", "backspace":
		m.mode = modeContent
	}
	return nil
}

func renderPicker(m *Model, width, height int) string {
	titleStyle := lipgloss.NewStyle().Foreground(stylesPkg.Primary).Bold(true)
	cursorStyle := lipgloss.NewStyle().Foreground(stylesPkg.BgBase).Background(stylesPkg.Primary)
	rowStyle := lipgloss.NewStyle().Foreground(stylesPkg.FgBase)
	mutedStyle := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)

	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Compatibility level for %s", m.subject)))
	b.WriteString("\n\n")
	for i, level := range api.CompatibilityLevels() {
		marker := ""
		if level == m.compat {
			marker = "  (current)"
		}
		line := string(level) + marker
		if i == m.pickerCursor {
			b.WriteString(cursorStyle.Render("▸ " + line))
		} else {
			b.WriteString(rowStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("↑/↓ select · enter set (confirm) · esc cancel"))
	return b.String()
}
