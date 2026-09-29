// Package connector implements the connector detail page (dynamic page ID
// "connector:<connect>:<name>"). It renders a summary strip plus four tabs —
// Overview, Tasks, Config and Topics — over the shared template shell. The page
// is created by the router; see NewModelWithCommon for the constructor the
// router wires to the "connector:<connect>:<name>" dynamic ID.
package connector

import (
	"encoding/json"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) beginConfigEdit() tea.Cmd {
	m.configText = m.configJSON()
	m.configEditor.SetValue(m.configText)
	m.editing = true
	return m.configEditor.Focus()
}

// commitConfigEdit validates the edited JSON, requires a change from the loaded
// value, then confirms before calling UpdateConnectorConfig.
func (m *Model) commitConfigEdit() tea.Cmd {
	newText := m.configEditor.Value()
	if newText == m.configText {
		return core.NewNotification(core.StatusWarning, "Config", "no changes to save")
	}
	var parsed map[string]string
	if err := json.Unmarshal([]byte(newText), &parsed); err != nil {
		return core.NotifyError("Invalid JSON config", err)
	}
	connect, name := m.connect, m.name
	ds := m.common.DataSource
	m.configEditor.Blur()
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Update config",
			Message:      "Save connector configuration? Masked secrets (********) will overwrite the stored values as-is.",
			Danger:       true,
			ConfirmLabel: "Save",
			OnConfirm: func() tea.Msg {
				_, err := ds.UpdateConnectorConfig(connect, name, parsed)
				return configUpdatedMsg{err: err}
			},
		}
	}
}

func (m *Model) handleConfigUpdated(v configUpdatedMsg) tea.Cmd {
	if v.err != nil {
		m.editing = true
		return tea.Batch(
			func() tea.Msg { return shared.NewUIError("connector", "Config update failed", v.err) },
			m.configEditor.Focus(),
		)
	}
	m.editing = false
	m.detailsLoaded = false
	return tea.Batch(core.NewNotification(core.StatusSuccess, "Config updated", m.name), m.loadDetails())
}

// configJSON renders the (masked) config map as indented JSON.
func (m *Model) configJSON() string {
	if len(m.details.Config) == 0 {
		return "{}"
	}
	b, err := json.MarshalIndent(m.details.Config, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (m *Model) configHasMasked() bool {
	for _, v := range m.details.Config {
		if v == api.ConnectorSecretPlaceholder {
			return true
		}
	}
	return false
}

// --- rendering ---

func (m *Model) renderConfig() string {
	var b strings.Builder
	if m.editing {
		if m.configHasMasked() {
			b.WriteString(m.common.Styles.StatusStyle.Warning.Render("⚠ Masked secrets (********) must be replaced with real values before saving.") + "\n\n")
		}
		b.WriteString(m.configEditor.View())
		b.WriteString("\n")
		b.WriteString(m.common.Styles.Muted.Render(keys.Hint(keys.ScopeTextEntry, keys.ActionCommitSave, "save", keys.ActionCancel, "cancel")))
		return b.String()
	}
	if m.configHasMasked() {
		b.WriteString(m.common.Styles.StatusStyle.Warning.Render("⚠ Secret values are masked (********).") + "\n\n")
	}
	b.WriteString(m.configJSON())
	b.WriteString("\n\n")
	b.WriteString(m.common.Styles.Muted.Render(keys.Default.KeyFor(keys.ActionEdit) + ": edit config"))
	return b.String()
}
