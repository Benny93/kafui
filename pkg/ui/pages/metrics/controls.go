package metrics

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/ui/components/menu"
	"github.com/Benny93/kafui/pkg/ui/keys"
)

// KeyScope implements core.KeyScoper.
func (m *Model) KeyScope() keys.Scope { return pageScope() }

// Unwind implements core.Unwinder: esc closes the graph picker before the shell
// navigates back.
func (m *Model) Unwind() (tea.Cmd, bool) {
	if m.picker.visible {
		m.picker.visible = false
		return nil, true
	}
	return nil, false
}

// ContextActions implements core.ActionProvider. The graph picker used to be a
// bare `g` — which is the go-to-first alias — and its parameter prompt a bare
// `p`, which means pause everywhere else.
func (m *Model) ContextActions() []menu.Entry {
	entries := []menu.Entry{
		{Label: "Refresh metrics", Key: keys.Default.KeyFor(keys.ActionRefresh), Run: func() tea.Cmd {
			if m.common != nil && m.common.MetricsCollector != nil {
				return m.common.MetricsCollector.CollectCmd()
			}
			return nil
		}},
	}
	graphs := menu.Entry{Label: "Show the graph picker", Run: func() tea.Cmd {
		m.picker.visible = !m.picker.visible
		return nil
	}}
	if !m.picker.hasGraphs() {
		graphs.Disabled = true
		graphs.Reason = "no graphs are configured for this cluster"
	}
	return append(entries, graphs)
}
