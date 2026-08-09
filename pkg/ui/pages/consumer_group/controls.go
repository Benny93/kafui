package consumergroup

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/authz"
	"github.com/Benny93/kafui/pkg/ui/components/menu"
	"github.com/Benny93/kafui/pkg/ui/keys"
)

// KeyScope implements core.KeyScoper.
func (m *Model) KeyScope() keys.Scope { return keys.ScopeListContent }

// Unwind implements core.Unwinder: esc leaves the topic filter before the shell
// navigates back to the group list.
func (m *Model) Unwind() (tea.Cmd, bool) {
	if m.searching {
		m.searching = false
		m.searchInput.Blur()
		return nil, true
	}
	return nil, false
}

// ContextActions implements core.ActionProvider. Offset resets are the reason
// this menu matters: they are irreversible and were previously reachable only
// by knowing an undocumented key.
func (m *Model) ContextActions() []menu.Entry {
	keyOf := func(a keys.Action) string { return keys.Default.KeyFor(a) }
	group := m.groupID

	gated := func(e menu.Entry, action authz.Action) menu.Entry {
		if m.common != nil && !m.common.Can(action, authz.ResourceConsumerGroup, group) {
			e.Disabled = true
			e.Reason = "not permitted by the active profile or the cluster is read-only"
		}
		return e
	}

	return []menu.Entry{
		gated(menu.Entry{Label: "Reset offsets…", Destructive: true,
			Run: func() tea.Cmd { return m.openResetForm() }}, authz.ActionResetOffsets),
		gated(menu.Entry{Label: "Delete group", Key: keyOf(keys.ActionDelete), Destructive: true,
			Run: func() tea.Cmd { return m.deleteGroup() }}, authz.ActionDelete),
		{Label: "Filter topics", Key: keyOf(keys.ActionSearch), Run: func() tea.Cmd {
			m.searching = true
			return m.searchInput.Focus()
		}},
		gated(menu.Entry{Label: "Delete offsets for the selected topic", Destructive: true,
			Run: func() tea.Cmd { return m.deleteSelectedTopicOffsets() }}, authz.ActionResetOffsets),
		{Label: "Cycle auto-refresh interval", Run: func() tea.Cmd { return m.cycleAutoRefresh() }},
		{Label: "Refresh", Key: keyOf(keys.ActionRefresh), Run: func() tea.Cmd { return m.loadDetail() }},
	}
}
