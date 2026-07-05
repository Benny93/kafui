package ksql

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// queryContentProvider bridges the template content area to the query model.
type queryContentProvider struct{ model *QueryModel }

func (p *queryContentProvider) RenderContent(width, height int) string {
	return p.model.renderContent(width, height)
}
func (p *queryContentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd {
	return p.model.handle(msg)
}
func (p *queryContentProvider) InitContent() tea.Cmd { return nil }

// IsInputMode is true while editing so the shell lets every keystroke (letters
// like 'q', digits) reach the SQL editor unmodified.
func (p *queryContentProvider) IsInputMode() bool { return p.model.IsInputMode() }

func (p *queryContentProvider) GetContentSize(width int) int {
	return len(p.model.resRows) + len(p.model.props)*2 + 16
}

// queryHelpKeyMap adapts the query bindings to the footer help.KeyMap.
type queryHelpKeyMap struct{ keys queryKeys }

func (h queryHelpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{h.keys.Execute, h.keys.Clear, h.keys.ClearRes, h.keys.AddProp, h.keys.FocusNext, h.keys.Back}
}
func (h queryHelpKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{h.ShortHelp()} }
