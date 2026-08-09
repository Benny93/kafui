package messagedetail

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/ui/components/menu"
	"github.com/Benny93/kafui/pkg/ui/keys"
)

// The router holds MessageDetailPageModel, not the content provider, so the
// wrapper forwards the controls-spec interfaces.

// KeyScope implements core.KeyScoper.
func (m *MessageDetailPageModel) KeyScope() keys.Scope { return keys.ScopeListContent }

// ContextActions implements core.ActionProvider. Export, and copying the key or
// the value separately, have no direct keys — `c` copies the focused pane.
func (m *MessageDetailPageModel) ContextActions() []menu.Entry {
	p := m.contentProvider
	if p == nil {
		return nil
	}
	keyOf := func(a keys.Action) string { return keys.Default.KeyFor(a) }
	return []menu.Entry{
		{Label: "Copy the focused pane", Key: keyOf(keys.ActionCopy), Run: func() tea.Cmd {
			p.model.CopyContentWithFeedback()
			return nil
		}},
		{Label: "Copy headers as CSV", Run: func() tea.Cmd { p.copyHeadersAsCSV(); return nil }},
		{Label: "Copy metadata as CSV", Run: func() tea.Cmd { p.copyMetadataAsCSV(); return nil }},
		{Label: "Export the message to a file", Run: func() tea.Cmd { p.exportMessageToFile(); return nil }},
		{Label: "Cycle the payload format", Key: keyOf(keys.ActionFormat), Run: func() tea.Cmd {
			p.model.ToggleDisplayFormat()
			return nil
		}},
		{Label: "Toggle soft wrap", Key: keyOf(keys.ActionWrap)},
		{Label: "Reload schema info", Key: keyOf(keys.ActionRefresh), Run: func() tea.Cmd {
			return p.model.LoadSchemaInfoAsync()
		}},
	}
}

// IsInputMode implements core.InputModeReporter.
func (m *MessageDetailPageModel) IsInputMode() bool {
	return m.contentProvider != nil && m.contentProvider.IsInputMode()
}
