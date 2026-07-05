// Package errorpage provides a full-content error view used as the router's
// fallback for unknown/uncreatable routes (UI-10). Variants: not-found,
// access-denied, and a generic unexpected error.
package errorpage

import (
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Variant selects the icon/title/tone of the error page.
type Variant int

const (
	NotFound Variant = iota
	AccessDenied
	Generic
)

// Model is a minimal core.Page rendering a centered error message with a
// recovery hint. It holds no data source and never mutates anything.
type Model struct {
	common        *core.Common
	variant       Variant
	title         string
	detail        string
	width, height int
}

// New builds an error page. title/detail may be empty to use variant defaults.
func New(common *core.Common, variant Variant, title, detail string) *Model {
	return &Model{common: common, variant: variant, title: title, detail: detail}
}

func (m *Model) Init() tea.Cmd                            { return nil }
func (m *Model) Update(tea.Msg) (tea.Model, tea.Cmd)      { return m, nil }
func (m *Model) SetDimensions(w, h int)                   { m.width, m.height = w, h }
func (m *Model) GetID() string                            { return "error" }
func (m *Model) GetTitle() string                         { return "Error" }
func (m *Model) GetHelp() []key.Binding                   { return nil }
func (m *Model) HandleNavigation(tea.Msg) (core.Page, tea.Cmd) { return m, nil }
func (m *Model) OnFocus() tea.Cmd                         { return nil }
func (m *Model) OnBlur() tea.Cmd                          { return nil }

func (m *Model) icon() string {
	switch m.variant {
	case NotFound:
		return "🔍"
	case AccessDenied:
		return "🔒"
	default:
		return "⚠"
	}
}

func (m *Model) titleText() string {
	if m.title != "" {
		return m.title
	}
	switch m.variant {
	case NotFound:
		return "Not found"
	case AccessDenied:
		return "Access denied"
	default:
		return "Something went wrong"
	}
}

func (m *Model) View() string {
	accent := lipgloss.Color("#F25D94")
	title := lipgloss.NewStyle().Foreground(accent).Bold(true).Render(m.icon() + "  " + m.titleText())
	body := m.detail
	hint := lipgloss.NewStyle().Faint(true).Render("press esc to go back")
	content := lipgloss.JoinVertical(lipgloss.Center, title, "", body, "", hint)
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}
