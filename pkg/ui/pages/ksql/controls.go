package ksql

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/ui/components/menu"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
)

// KeyScope implements core.KeyScoper for the overview page.
func (m *Model) KeyScope() keys.Scope { return overviewScope() }

// ContextActions implements core.ActionProvider for the overview page.
func (m *Model) ContextActions() []menu.Entry {
	keyOf := func(a keys.Action) string { return keys.Default.KeyFor(a) }
	return []menu.Entry{
		{Label: "Open the query editor", Key: keyOf(keys.ActionEdit),
			Run: func() tea.Cmd { return core.NewPageChangeMsg("ksql_query", nil) }},
		{Label: "Query the selected stream or table", Key: keyOf(keys.ActionActivate),
			Run: func() tea.Cmd { return m.seedQuery() }},
		{Label: "Sort by the next column", Key: keyOf(keys.ActionSort)},
		{Label: "Refresh", Key: keyOf(keys.ActionRefresh), Run: func() tea.Cmd {
			m.streamsLoaded, m.tablesLoaded = false, false
			return tea.Batch(m.loadStreams(), m.loadTables())
		}},
	}
}

// KeyScope implements core.KeyScoper for the query editor.
func (m *QueryModel) KeyScope() keys.Scope { return queryScope() }

// ContextActions implements core.ActionProvider for the query editor. Clearing
// the editor and results, and the property rows, used to be ctrl+l / ctrl+r /
// ctrl+n / ctrl+d — two of which collided with global chords and one of which
// the spec forbids outright.
func (m *QueryModel) ContextActions() []menu.Entry {
	return []menu.Entry{
		{Label: "Run the statement", Key: keys.Default.KeyFor(keys.ActionRefresh),
			Run: func() tea.Cmd { return m.execute() }},
		{Label: "Clear the editor", Run: func() tea.Cmd {
			if !m.running {
				m.editor.SetValue("")
			}
			return nil
		}},
		{Label: "Clear the results", Run: func() tea.Cmd { return m.clearResults() }},
		{Label: "Add a property row", Run: func() tea.Cmd { m.addProp(); return nil }},
		{Label: "Remove the property row", Run: func() tea.Cmd { m.delProp(); return nil }},
	}
}
