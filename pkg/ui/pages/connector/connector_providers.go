package connector

import (
	tea "github.com/charmbracelet/bubbletea"
)

// contentProvider bridges the template content area to the page model.
type contentProvider struct{ model *Model }

func (p *contentProvider) RenderContent(width, height int) string {
	return p.model.render(width, height)
}
func (p *contentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd { return p.model.handle(msg) }
func (p *contentProvider) InitContent() tea.Cmd                    { return nil }

// IsInputMode delegates to the page model, the single source of truth.
func (p *contentProvider) IsInputMode() bool { return p.model.IsInputMode() }

func (p *contentProvider) GetContentSize(width int) int {
	return len(p.model.details.Tasks) + len(p.model.details.Config) + 10
}
