package clusters

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/ui/components/menu"
	"github.com/Benny93/kafui/pkg/ui/keys"
)

// KeyScope implements core.KeyScoper.
func (m *Model) KeyScope() keys.Scope { return pageScope() }

// ContextActions implements core.ActionProvider. Validating a cluster used to
// sit on a bare `v` that appeared in no shared help.
func (m *Model) ContextActions() []menu.Entry {
	keyOf := func(a keys.Action) string { return keys.Default.KeyFor(a) }
	return []menu.Entry{
		{Label: "Switch to this cluster", Key: keyOf(keys.ActionActivate),
			Run: func() tea.Cmd { return m.openSelected() }},
		{Label: "Validate connectivity", Run: func() tea.Cmd { return m.validateSelected() }},
		{Label: "Show only offline clusters", Key: keyOf(keys.ActionToggleInternal), Run: func() tea.Cmd {
			m.offlineOnly = !m.offlineOnly
			m.table.SetCursor(0)
			m.rebuildRows()
			return nil
		}},
		{Label: "Refresh", Key: keyOf(keys.ActionRefresh), Run: func() tea.Cmd { return m.refreshSelected() }},
	}
}
