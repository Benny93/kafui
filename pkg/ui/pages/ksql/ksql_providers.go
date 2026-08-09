package ksql

import (
	tea "github.com/charmbracelet/bubbletea"
)

// overviewContentProvider bridges the template content area to the overview model.
type overviewContentProvider struct{ model *Model }

func (p *overviewContentProvider) RenderContent(width, height int) string {
	return p.model.renderContent(width, height)
}
func (p *overviewContentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd {
	return p.model.handle(msg)
}
func (p *overviewContentProvider) InitContent() tea.Cmd { return nil }
func (p *overviewContentProvider) IsInputMode() bool    { return false }
func (p *overviewContentProvider) GetContentSize(width int) int {
	return len(p.model.streams) + len(p.model.tables) + 8
}
