package appconfig_view

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/ui/components/menu"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
)

// KeyScope implements core.KeyScoper.
func (m *Model) KeyScope() keys.Scope { return keys.ScopeContent }

// ContextActions implements core.ActionProvider. This screen is read-only, so
// the menu is short — but `a` must never open an empty menu.
func (m *Model) ContextActions() []menu.Entry {
	keyOf := func(a keys.Action) string { return keys.Default.KeyFor(a) }
	return []menu.Entry{
		{Label: "Copy the configuration", Key: keyOf(keys.ActionCopy)},
		{Label: "Search the configuration", Key: keyOf(keys.ActionSearch)},
		{Label: "Toggle soft wrap", Key: keyOf(keys.ActionWrap)},
		{Label: "Open the cluster setup wizard", Run: func() tea.Cmd {
			return core.NewPageChangeMsg("cluster_form", nil)
		}},
	}
}
