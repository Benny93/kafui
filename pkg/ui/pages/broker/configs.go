// Package broker implements the broker detail page (dynamic page ID
// "broker:<id>"). It renders a summary strip plus three tabs — Log Dirs,
// Configs and Metrics — over the shared template shell. The page is created by
// the router; see NewModelWithCommon / NewModelWithInfo for the constructors the
// router wires to the "broker:<id>" dynamic ID.
package broker

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

func configColumns() []table.Column {
	return []table.Column{
		{Title: "Key", Width: 34},
		{Title: "Value", Width: 26},
		{Title: "Source", Width: 26},
	}
}

// --- core.Page ---

func (m *Model) handleConfigsKey(msg tea.KeyMsg) tea.Cmd {
	// Search and edit are registry actions handled in handleKey; this tab has
	// nothing of its own left.
	return m.forwardToActive(msg)
}

func (m *Model) handleSearchKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		m.searching = false
		m.cfgFilter = m.searchInput.Value()
		m.searchInput.Blur()
		m.rebuildConfigTable()
		return nil
	case "esc":
		m.searching = false
		m.searchInput.Blur()
		return nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return cmd
}

func (m *Model) selectedConfig() (api.BrokerConfigEntry, bool) {
	i := m.cfgTable.Cursor()
	if i < 0 || i >= len(m.cfgVisible) {
		return api.BrokerConfigEntry{}, false
	}
	return m.cfgVisible[i], true
}

func (m *Model) beginEdit() tea.Cmd {
	entry, ok := m.selectedConfig()
	if !ok {
		return nil
	}
	if entry.ReadOnly {
		return core.NewNotification(core.StatusWarning, "Config", "Property is read-only")
	}
	m.editing = true
	m.editKey = entry.Name
	m.editOld = entry.Value
	m.editInput.SetValue(entry.Value)
	return m.editInput.Focus()
}

func (m *Model) handleEditKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.cancelEdit()
		return nil
	case "enter":
		return m.commitEdit()
	}
	var cmd tea.Cmd
	m.editInput, cmd = m.editInput.Update(msg)
	return cmd
}

func (m *Model) cancelEdit() {
	m.editing = false
	m.editInput.Blur()
	m.editKey = ""
}

// commitEdit implements the save state machine: unchanged value is a no-op;
// a changed value asks for confirmation before calling AlterBrokerConfig.
func (m *Model) commitEdit() tea.Cmd {
	newVal := m.editInput.Value()
	if newVal == m.editOld {
		m.cancelEdit()
		return nil
	}
	key := m.editKey
	id := m.brokerID
	ds := m.common.DataSource
	// Keep edit mode open until the change is confirmed + applied.
	m.editInput.Blur()
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Change config",
			Message:      "Are you sure you want to change the value?",
			Danger:       true,
			ConfirmLabel: "Change",
			OnConfirm: func() tea.Msg {
				err := ds.AlterBrokerConfig(id, key, newVal)
				return configAlteredMsg{brokerID: id, key: key, value: newVal, err: err}
			},
		}
	}
}

func (m *Model) handleConfigAltered(v configAlteredMsg) tea.Cmd {
	if v.brokerID != m.brokerID {
		return nil
	}
	if v.err != nil {
		// Stay in edit mode and surface the cluster's rejection message.
		m.editing = true
		m.editInput.SetValue(v.value)
		var invalid api.InvalidConfigError
		msg := v.err.Error()
		if ok := asInvalidConfig(v.err, &invalid); ok {
			msg = invalid.Error()
		}
		return tea.Batch(
			func() tea.Msg { return shared.NewUIError("config", msg, nil) },
			m.editInput.Focus(),
		)
	}
	m.cancelEdit()
	m.configsLoaded = false
	return tea.Batch(core.NewNotification(core.StatusSuccess, "Config updated", v.key), m.loadConfigs())
}

func (m *Model) rebuildConfigTable() {
	entries := sortedFilteredConfigs(m.configs, m.cfgFilter)
	m.cfgVisible = entries
	rows := make([]table.Row, 0, len(entries))
	for _, e := range entries {
		val := shared.FormatConfigValue(e.Name, e.Value, e.Sensitive)
		rows = append(rows, table.Row{e.Name, val, e.Source})
	}
	m.cfgTable.SetRows(rows)
	if m.cfgTable.Cursor() >= len(rows) {
		m.cfgTable.SetCursor(0)
	}
}

// --- rendering ---

func (m *Model) renderConfigs() string {
	if !m.configsLoaded {
		return m.common.Styles.Muted.Render("Loading configs…")
	}
	if m.configsErr != nil {
		return m.common.Styles.Error.Render("Error: "+m.configsErr.Error()) + "\n" + m.common.Styles.Muted.Render("Press r to retry.")
	}
	var b strings.Builder
	b.WriteString(stylesPkg.FrameTable(m.cfgTable.View()))
	b.WriteString("\n")
	if m.searching {
		b.WriteString(m.searchInput.View())
		return b.String()
	}
	if m.editing {
		b.WriteString(m.common.Styles.Header.Render("Edit " + m.editKey + ": "))
		b.WriteString(m.editInput.View())
		b.WriteString("\n")
		b.WriteString(m.common.Styles.Muted.Render(keys.Hint(keys.ScopeListContent, keys.ActionActivate, "save", keys.ActionCancel, "cancel")))
		return b.String()
	}
	b.WriteString(m.configFooter())
	return b.String()
}

// configFooter shows the source-category hint and sensitive/exact-byte hints for
// the selected row (the TUI adaptation of the hover tooltips).
func (m *Model) configFooter() string {
	entry, ok := m.selectedConfig()
	if !ok {
		return m.common.Styles.Muted.Render(keys.Default.KeyFor(keys.ActionEdit) + ": edit • " + keys.Default.KeyFor(keys.ActionSearch) + ": search")
	}
	hint := m.common.Styles.Muted.Render(sourceExplanation(entry.Source))
	extra := ""
	if entry.Sensitive {
		extra = "  •  Sensitive Value"
	} else if n, err := strconv.ParseInt(entry.Value, 10, 64); err == nil && n > 0 && strings.HasSuffix(entry.Name, ".bytes") {
		extra = fmt.Sprintf("  •  %d bytes", n)
	}
	return hint + m.common.Styles.Muted.Render(extra) + "\n" + m.common.Styles.Muted.Render(keys.Default.KeyFor(keys.ActionEdit)+": edit • "+keys.Default.KeyFor(keys.ActionSearch)+": search")
}

// sourceRank orders config sources: dynamic* first, then static broker, default,
// then unknown/other.
func sourceRank(source string) int {
	switch source {
	case "Dynamic broker config":
		return 0
	case "Dynamic default broker config":
		return 1
	case "Static broker config":
		return 2
	case "Default config":
		return 3
	case "Unknown":
		return 4
	default:
		return 5
	}
}

func sourceExplanation(source string) string {
	switch source {
	case "Dynamic broker config":
		return "Dynamic broker config: set per-broker at runtime"
	case "Dynamic default broker config":
		return "Dynamic default broker config: cluster-wide runtime default"
	case "Static broker config":
		return "Static broker config: from server.properties (needs restart)"
	case "Default config":
		return "Default config: Kafka built-in default"
	default:
		return "Unknown config source"
	}
}

// sortedFilteredConfigs returns entries filtered by a case-insensitive substring
// match on key OR value, ordered by source priority (stable within groups).
func sortedFilteredConfigs(entries []api.BrokerConfigEntry, filter string) []api.BrokerConfigEntry {
	out := make([]api.BrokerConfigEntry, 0, len(entries))
	q := strings.ToLower(filter)
	for _, e := range entries {
		if q == "" || strings.Contains(strings.ToLower(e.Name), q) || strings.Contains(strings.ToLower(e.Value), q) {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return sourceRank(out[i].Source) < sourceRank(out[j].Source)
	})
	return out
}

// asInvalidConfig reports whether err is (or wraps) an api.InvalidConfigError,
// copying it into dst when so.
func asInvalidConfig(err error, dst *api.InvalidConfigError) bool {
	if ic, ok := err.(api.InvalidConfigError); ok {
		*dst = ic
		return true
	}
	return false
}
