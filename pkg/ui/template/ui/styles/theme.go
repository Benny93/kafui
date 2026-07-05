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

// NewLightTheme returns the light-background palette. It mirrors NewDarkTheme's
// shape so the whole template renders consistently in light mode (UI-3).
func NewLightTheme() *Theme {
	return &Theme{
		Name:   "light",
		IsDark: false,

		Primary:   lipgloss.Color("#5B3FC4"), // Slate Blue (darker for light bg)
		Secondary: lipgloss.Color("#B8860B"), // Dark Goldenrod
		Tertiary:  lipgloss.Color("#2C6FA6"), // Steel Blue (darker)
		Accent:    lipgloss.Color("#C2410C"), // Burnt Orange

		// Backgrounds - light
		BgBase:        lipgloss.Color("#FFFFFF"),
		BgBaseLighter: lipgloss.Color("#F3F4F6"),
		BgSubtle:      lipgloss.Color("#E5E7EB"),
		BgOverlay:     lipgloss.Color("#D1D5DB"),

		// Foregrounds - dark for contrast on light bg
		FgBase:      lipgloss.Color("#1F2937"),
		FgMuted:     lipgloss.Color("#6B7280"),
		FgHalfMuted: lipgloss.Color("#4B5563"),
		FgSubtle:    lipgloss.Color("#9CA3AF"),
		FgSelected:  lipgloss.Color("#FFFFFF"),

		// Borders
		Border:      lipgloss.Color("#D1D5DB"),
		BorderFocus: lipgloss.Color("#5B3FC4"),

		// Status
		Success: lipgloss.Color("#059669"),
		Error:   lipgloss.Color("#DC2626"),
		Warning: lipgloss.Color("#D97706"),
		Info:    lipgloss.Color("#2563EB"),

		White: lipgloss.Color("#FFFFFF"),
	}
}

var defaultTheme *Theme

func CurrentTheme() *Theme {
	if defaultTheme == nil {
		defaultTheme = NewDarkTheme()
	}
	return defaultTheme
}

// ponytail: UI-3 residual — this template Theme and pkg/ui/styles.Theme remain
// two separate types (approach (b): the shell keeps them in sync via SetTheme on
// toggle rather than merging them into one). Full unification into a single Theme
// type + removal of pkg/ui/styles.go's package-level color vars is deferred.
//
// SetTheme swaps the active template theme so page chrome follows the user's
// selection (UI-3). Passing isDark chooses the dark or light palette; the style
// cache is rebuilt lazily on the next S() call.
func SetTheme(isDark bool) {
	if isDark {
		defaultTheme = NewDarkTheme()
	} else {
		defaultTheme = NewLightTheme()
	}
}

// IsDark reports whether the active template theme is the dark palette.
func IsDark() bool {
	return CurrentTheme().IsDark
}