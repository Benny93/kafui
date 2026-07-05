package appconfig_view

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// contentProvider renders the pre-built config document inside a scrollable
// viewport. The document is static (config does not change while viewing), so it
// is built once and only re-sized/scrolled here.
type contentProvider struct {
	document string
	viewport viewport.Model
	ready    bool
}

func newContentProvider(document string) *contentProvider {
	return &contentProvider{document: document}
}

func (p *contentProvider) RenderContent(width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	if !p.ready {
		p.viewport = viewport.New(width, height)
		p.viewport.SetContent(p.document)
		p.ready = true
	} else {
		p.viewport.Width = width
		p.viewport.Height = height
	}
	return p.viewport.View()
}

func (p *contentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd {
	if !p.ready {
		return nil
	}
	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return cmd
}

func (p *contentProvider) InitContent() tea.Cmd { return nil }

// GetContentSize returns the viewport height so the template does not draw its
// own scrollbar — scrolling is handled by the viewport itself.
func (p *contentProvider) GetContentSize(width int) int {
	if p.ready {
		return p.viewport.Height
	}
	return 0
}

func (p *contentProvider) IsInputMode() bool { return false }
