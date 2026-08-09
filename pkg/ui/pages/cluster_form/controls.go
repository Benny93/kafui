package cluster_form

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/ui/components/menu"
	"github.com/Benny93/kafui/pkg/ui/keys"
)

// KeyScope implements core.KeyScoper.
func (m *Model) KeyScope() keys.Scope { return keys.ScopeContent }

// IsInputMode implements core.InputModeReporter. The wizard is a form, so every
// printable key is typed and no Normal-mode binding may fire.
func (m *Model) IsInputMode() bool { return !m.disabled && m.form != nil }

// ContextActions implements core.ActionProvider.
func (m *Model) ContextActions() []menu.Entry {
	entries := []menu.Entry{
		{Label: "Validate connectivity", Key: keys.Default.KeyFor(keys.ActionRefresh),
			Run: func() tea.Cmd { return m.runValidate() }},
	}
	if m.originalName != "" {
		entries = append(entries, menu.Entry{
			Label:       "Delete this cluster",
			Destructive: true,
			Run:         func() tea.Cmd { return m.confirmDelete() },
		})
	}
	return entries
}
