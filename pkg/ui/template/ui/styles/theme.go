package styles

import (
	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Name   string
	IsDark bool

	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Tertiary  lipgloss.Color
	Accent    lipgloss.Color

	BgBase        lipgloss.Color
	BgBaseLighter lipgloss.Color
	BgSubtle      lipgloss.Color
	BgOverlay     lipgloss.Color

	FgBase      lipgloss.Color
	FgMuted     lipgloss.Color
	FgHalfMuted lipgloss.Color
	FgSubtle    lipgloss.Color
	FgSelected  lipgloss.Color

	Border      lipgloss.Color
	BorderFocus lipgloss.Color

	Success lipgloss.Color
	Error   lipgloss.Color
	Warning lipgloss.Color
	Info    lipgloss.Color

	White lipgloss.Color

	styles *Styles
}

type Styles struct {
	Base         lipgloss.Style
	SelectedBase lipgloss.Style

	Title        lipgloss.Style
	Subtitle     lipgloss.Style
	Text         lipgloss.Style
	TextSelected lipgloss.Style
	Muted        lipgloss.Style
	Subtle       lipgloss.Style

	Success lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style
}

func (t *Theme) S() *Styles {
	if t.styles == nil {
		t.styles = t.buildStyles()
	}
	return t.styles
}

func (t *Theme) buildStyles() *Styles {
	base := lipgloss.NewStyle().Foreground(t.FgBase)
	
	return &Styles{
		Base:         base,
		SelectedBase: base.Background(t.Primary),
		
		Title:    base.Foreground(t.Accent).Bold(true),
		Subtitle: base.Foreground(t.Secondary).Bold(true),
		Text:     base,
		TextSelected: base.Background(t.Primary).Foreground(t.FgSelected),
		Muted:    base.Foreground(t.FgMuted),
		Subtle:   base.Foreground(t.FgSubtle),
		
		Success: base.Foreground(t.Success),
		Error:   base.Foreground(t.Error),
		Warning: base.Foreground(t.Warning),
		Info:    base.Foreground(t.Info),
	}
}

func NewDarkTheme() *Theme {
	return &Theme{
		Name:   "dark",
		IsDark: true,

		Primary:   lipgloss.Color("#9370DB"), // Medium Slate Blue
		Secondary: lipgloss.Color("#FFD700"), // Gold
		Tertiary:  lipgloss.Color("#4682B4"), // Steel Blue
		Accent:    lipgloss.Color("#FF8C00"), // Dark Orange

		// Backgrounds
		BgBase:        lipgloss.Color("#282C34"), // Dark Gray
		BgBaseLighter: lipgloss.Color("#32363E"), // Lighter Dark Gray
		BgSubtle:      lipgloss.Color("#3C4048"), // Subtle Gray
		BgOverlay:     lipgloss.Color("#50545C"), // Overlay Gray

		// Foregrounds
		FgBase:      lipgloss.Color("#DCDCDC"), // Light Gray
		FgMuted:     lipgloss.Color("#969696"), // Muted Gray
		FgHalfMuted: lipgloss.Color("#B4B4B4"), // Half Muted Gray
		FgSubtle:    lipgloss.Color("#787878"), // Subtle Gray
		FgSelected:  lipgloss.Color("#FFFFFF"), // White

		// Borders
		Border:      lipgloss.Color("#50545C"), // Border Gray
		BorderFocus: lipgloss.Color("#9370DB"), // Primary for focus

		// Status
		Success: lipgloss.Color("#2ECC71"), // Green
		Error:   lipgloss.Color("#E74C3C"), // Red
		Warning: lipgloss.Color("#F1C40F"), // Yellow
		Info:    lipgloss.Color("#3498DB"), // Blue

		// Colors
		White: lipgloss.Color("#FFFFFF"), // Pure White
	}
}

var defaultTheme *Theme

func CurrentTheme() *Theme {
	if defaultTheme == nil {
		defaultTheme = NewDarkTheme()
	}
	return defaultTheme
}