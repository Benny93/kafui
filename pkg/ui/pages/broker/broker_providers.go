package broker

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

// IsInputMode reports an active text-input sub-state so the shell stops
// intercepting hotkeys (search, inline config edit, or the reassignment form).
func (p *contentProvider) IsInputMode() bool {
	return p.model.searching || p.model.editing || p.model.moveForm != nil
}

func (p *contentProvider) GetContentSize(width int) int {
	switch p.model.active {
	case tabConfigs:
		return len(p.model.configs) + 8
	default:
		return len(p.model.logDirs) + 8
	}
}

// helpKeyMap adapts the page bindings to the footer help.KeyMap interface.
type helpKeyMap struct{ keys pageKeys }

func (h helpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{h.keys.NextTab, h.keys.Expand, h.keys.Edit, h.keys.Move, h.keys.Search, h.keys.Back}
}
func (h helpKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{h.ShortHelp()} }
