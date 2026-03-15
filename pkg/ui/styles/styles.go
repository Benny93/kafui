package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Semantic color palette
var (
	Primary   = lipgloss.Color("#7D56F4")
	Secondary = lipgloss.Color("#383838")
	Accent    = lipgloss.Color("#73F59F")
	Error     = lipgloss.Color("#F25D94")
	Success   = lipgloss.Color("#10B981")
	Warning   = lipgloss.Color("#F59E0B")
	Info      = lipgloss.Color("#3B82F6")

	// Backgrounds
	BgBase    = lipgloss.Color("#1A1A2E")
	BgSubtle  = lipgloss.Color("#16213E")
	BgOverlay = lipgloss.Color("#0F3460")

	// Foregrounds
	FgBase   = lipgloss.Color("#EAEAEA")
	FgMuted  = lipgloss.Color("#A0A0A0")
	FgSubtle = lipgloss.Color("#666666")
)

// Styles contains all application-wide styles
type Styles struct {
	// Text styles
	Base   lipgloss.Style
	Muted  lipgloss.Style
	Header lipgloss.Style
	Error  lipgloss.Style

	// Component styles
	HeaderStyle     HeaderStyles
	SidebarStyle    SidebarStyles
	FooterStyle     FooterStyles
	TableStyle      TableStyles
	SearchStyle     SearchStyles
	ModalStyle      ModalStyles
	StatusStyle     StatusStyles
	HelpStyle       HelpStyles
	NavigationStyle NavigationStyles
}

type HeaderStyles struct {
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Resource lipgloss.Style
}

type SidebarStyles struct {
	Title    lipgloss.Style
	Section  lipgloss.Style
	Item     lipgloss.Style
	Selected lipgloss.Style
}

type FooterStyles struct {
	Base lipgloss.Style
	Key  lipgloss.Style
	Help lipgloss.Style
}

type TableStyles struct {
	Header   lipgloss.Style
	Row      lipgloss.Style
	Selected lipgloss.Style
}

type SearchStyles struct {
	Prompt lipgloss.Style
	Input  lipgloss.Style
	Help   lipgloss.Style
}

type ModalStyles struct {
	Box     lipgloss.Style
	Title   lipgloss.Style
	Content lipgloss.Style
}

type StatusStyles struct {
	Info    lipgloss.Style
	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style
}

type HelpStyles struct {
	Title   lipgloss.Style
	Key     lipgloss.Style
	Desc    lipgloss.Style
	Section lipgloss.Style
}

type NavigationStyles struct {
	Breadcrumb lipgloss.Style
	Separator  lipgloss.Style
	Active     lipgloss.Style
}

// DefaultStyles returns the default application styles
func DefaultStyles() *Styles {
	s := &Styles{}

	// Text styles
	s.Base = lipgloss.NewStyle().Foreground(FgBase)
	s.Muted = lipgloss.NewStyle().Foreground(FgMuted)
	s.Header = lipgloss.NewStyle().Foreground(Primary).Bold(true)
	s.Error = lipgloss.NewStyle().Foreground(Error)

	// Component styles
	s.HeaderStyle = HeaderStyles{
		Title:    lipgloss.NewStyle().Foreground(FgBase).Background(Primary).Padding(0, 1).Bold(true),
		Subtitle: lipgloss.NewStyle().Foreground(Accent).Italic(true),
		Resource: lipgloss.NewStyle().Foreground(FgBase).Background(Info).Padding(0, 1).Bold(true),
	}

	s.SidebarStyle = SidebarStyles{
		Title:    lipgloss.NewStyle().Foreground(Accent).Bold(true).MarginBottom(1),
		Section:  lipgloss.NewStyle().Foreground(FgMuted).Bold(true).MarginTop(1),
		Item:     lipgloss.NewStyle().Foreground(FgBase).PaddingLeft(2),
		Selected: lipgloss.NewStyle().Foreground(Primary).Background(Secondary).PaddingLeft(2).Bold(true),
	}

	s.FooterStyle = FooterStyles{
		Base: lipgloss.NewStyle().Foreground(FgMuted).Background(BgSubtle).Padding(0, 1),
		Key:  lipgloss.NewStyle().Foreground(Accent).Bold(true),
		Help: lipgloss.NewStyle().Foreground(FgSubtle),
	}

	s.TableStyle = TableStyles{
		Header:   lipgloss.NewStyle().Foreground(Primary).Bold(true).BorderStyle(lipgloss.NormalBorder()).BorderBottom(true),
		Row:      lipgloss.NewStyle().Foreground(FgBase),
		Selected: lipgloss.NewStyle().Foreground(FgBase).Background(Secondary).Bold(true),
	}

	s.SearchStyle = SearchStyles{
		Prompt: lipgloss.NewStyle().Foreground(Accent).MarginRight(1),
		Input:  lipgloss.NewStyle().Foreground(FgBase),
		Help:   lipgloss.NewStyle().Foreground(FgSubtle).Italic(true).MarginTop(1),
	}

	s.ModalStyle = ModalStyles{
		Box:     lipgloss.NewStyle().Padding(1, 2).BorderStyle(lipgloss.RoundedBorder()).BorderForeground(Primary),
		Title:   lipgloss.NewStyle().Foreground(Primary).Bold(true).MarginBottom(1),
		Content: lipgloss.NewStyle().Foreground(FgBase),
	}

	s.StatusStyle = StatusStyles{
		Info:    lipgloss.NewStyle().Foreground(Info),
		Success: lipgloss.NewStyle().Foreground(Success),
		Warning: lipgloss.NewStyle().Foreground(Warning),
		Error:   lipgloss.NewStyle().Foreground(Error),
	}

	s.HelpStyle = HelpStyles{
		Title:   lipgloss.NewStyle().Foreground(Primary).Bold(true).MarginBottom(1),
		Key:     lipgloss.NewStyle().Foreground(Accent).Bold(true),
		Desc:    lipgloss.NewStyle().Foreground(FgMuted),
		Section: lipgloss.NewStyle().Foreground(Accent).Bold(true).MarginTop(1),
	}

	s.NavigationStyle = NavigationStyles{
		Breadcrumb: lipgloss.NewStyle().Foreground(FgMuted),
		Separator:  lipgloss.NewStyle().Foreground(FgSubtle).Margin(0, 1),
		Active:     lipgloss.NewStyle().Foreground(Primary).Bold(true),
	}

	return s
}
