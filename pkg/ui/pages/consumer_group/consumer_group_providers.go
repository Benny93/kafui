package consumergroup

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// contentProvider bridges the template content area to the page model.
type contentProvider struct{ model *Model }

func (p *contentProvider) RenderContent(width, height int) string {
	return p.model.render(width, height)
}
func (p *contentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd { return p.model.handle(msg) }
func (p *contentProvider) InitContent() tea.Cmd                    { return nil }

// IsInputMode reports an active text-input sub-state (topic filter or the reset
// form) so the shell stops intercepting hotkeys.
func (p *contentProvider) IsInputMode() bool {
	return p.model.searching || p.model.resetForm != nil
}

func (p *contentProvider) GetContentSize(width int) int {
	return len(p.model.topicRows) + 10
}

// helpKeyMap adapts the page bindings to the footer help.KeyMap interface.
type helpKeyMap struct{ keys pageKeys }

func (h helpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{h.keys.Expand, h.keys.Filter, h.keys.Refresh, h.keys.Reset, h.keys.Delete, h.keys.Back}
}
func (h helpKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{h.ShortHelp()} }
