package broker

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/authz"
	"github.com/Benny93/kafui/pkg/ui/components/menu"
	"github.com/Benny93/kafui/pkg/ui/keys"
)

// KeyScope implements core.KeyScoper.
func (m *Model) KeyScope() keys.Scope { return keys.ScopeListContent }

// Unwind implements core.Unwinder: esc leaves the config search or collapses an
// expanded log directory before the shell navigates back.
func (m *Model) Unwind() (tea.Cmd, bool) {
	switch {
	case m.searching:
		m.searching = false
		m.searchInput.Blur()
	case m.expanded >= 0:
		m.expanded = -1
	default:
		return nil, false
	}
	return nil, true
}

// ContextActions implements core.ActionProvider. Moving a replica used to sit
// on a bare `m`, which is the metadata-pane key everywhere else.
func (m *Model) ContextActions() []menu.Entry {
	keyOf := func(a keys.Action) string { return keys.Default.KeyFor(a) }

	gated := func(e menu.Entry, action authz.Action) menu.Entry {
		if m.common != nil && !m.common.Can(action, authz.ResourceClusterConfig, "") {
			e.Disabled = true
			e.Reason = "not permitted by the active profile or the cluster is read-only"
		}
		return e
	}

	entries := []menu.Entry{
		{Label: "Log directories", Key: "1", Run: func() tea.Cmd { return m.switchTab(tabLogDirs) }},
		{Label: "Configuration", Key: "2", Run: func() tea.Cmd { return m.switchTab(tabConfigs) }},
		{Label: "Metrics", Key: "3", Run: func() tea.Cmd { return m.switchTab(tabMetrics) }},
		gated(menu.Entry{Label: "Edit the selected configuration entry", Key: keyOf(keys.ActionEdit),
			Run: func() tea.Cmd { return m.beginEdit() }}, authz.ActionEdit),
	}
	if m.expanded >= 0 {
		entries = append(entries, gated(menu.Entry{
			Label: "Move the selected replica to another log directory…",
			Run:   func() tea.Cmd { return m.openMoveForm() },
		}, authz.ActionEdit))
	}
	return append(entries,
		menu.Entry{Label: "Search configuration", Key: keyOf(keys.ActionSearch), Run: func() tea.Cmd {
			m.searching = true
			m.searchInput.SetValue(m.cfgFilter)
			return m.searchInput.Focus()
		}},
		menu.Entry{Label: "Refresh", Key: keyOf(keys.ActionRefresh), Run: func() tea.Cmd { return m.retry() }},
	)
}
