package styles

import "github.com/charmbracelet/lipgloss"

// ThemeType represents the type of theme
type ThemeType string

const (
	// DarkTheme is the default dark theme
	DarkTheme ThemeType = "dark"

	// LightTheme is the light theme option
	LightTheme ThemeType = "light"
)

// Theme defines a complete color theme
type Theme struct {
	// Theme name
	Name ThemeType

	// Primary colors
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color
	Error     lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Info      lipgloss.Color

	// Backgrounds
	BgBase    lipgloss.Color
	BgSubtle  lipgloss.Color
	BgOverlay lipgloss.Color

	// Foregrounds
	FgBase   lipgloss.Color
	FgMuted  lipgloss.Color
	FgSubtle lipgloss.Color
}

// DarkThemeColors returns the dark theme color palette
func DarkThemeColors() Theme {
	return Theme{
		Name: DarkTheme,

		// Primary colors - vibrant for dark background
		Primary:   lipgloss.Color("#7D56F4"),
		Secondary: lipgloss.Color("#383838"),
		Accent:    lipgloss.Color("#73F59F"),
		Error:     lipgloss.Color("#F25D94"),
		Success:   lipgloss.Color("#10B981"),
		Warning:   lipgloss.Color("#F59E0B"),
		Info:      lipgloss.Color("#3B82F6"),

		// Backgrounds - dark
		BgBase:    lipgloss.Color("#1A1A2E"),
		BgSubtle:  lipgloss.Color("#16213E"),
		BgOverlay: lipgloss.Color("#0F3460"),

		// Foregrounds - light for contrast
		FgBase:   lipgloss.Color("#EAEAEA"),
		FgMuted:  lipgloss.Color("#A0A0A0"),
		FgSubtle: lipgloss.Color("#666666"),
	}
}

// LightThemeColors returns the light theme color palette
func LightThemeColors() Theme {
	return Theme{
		Name: LightTheme,

		// Primary colors - slightly darker for light background
		Primary:   lipgloss.Color("#5B3FC4"),
		Secondary: lipgloss.Color("#E0E0E0"),
		Accent:    lipgloss.Color("#059669"),
		Error:     lipgloss.Color("#DC2626"),
		Success:   lipgloss.Color("#059669"),
		Warning:   lipgloss.Color("#D97706"),
		Info:      lipgloss.Color("#2563EB"),

		// Backgrounds - light
		BgBase:    lipgloss.Color("#FFFFFF"),
		BgSubtle:  lipgloss.Color("#F3F4F6"),
		BgOverlay: lipgloss.Color("#E5E7EB"),

		// Foregrounds - dark for contrast
		FgBase:   lipgloss.Color("#1F2937"),
		FgMuted:  lipgloss.Color("#6B7280"),
		FgSubtle: lipgloss.Color("#9CA3AF"),
	}
}

// DefaultTheme returns the default theme (dark)
func DefaultTheme() Theme {
	return DarkThemeColors()
}

// ApplyTheme applies a theme to the Styles struct
func (s *Styles) ApplyTheme(theme Theme) {
	// Update color variables (these are package-level)
	Primary = theme.Primary
	Secondary = theme.Secondary
	Accent = theme.Accent
	Error = theme.Error
	Success = theme.Success
	Warning = theme.Warning
	Info = theme.Info

	BgBase = theme.BgBase
	BgSubtle = theme.BgSubtle
	BgOverlay = theme.BgOverlay

	FgBase = theme.FgBase
	FgMuted = theme.FgMuted
	FgSubtle = theme.FgSubtle

	// Re-initialize styles with new colors. DefaultStyles() hardcodes the dark
	// theme type, so restore the applied theme's name afterwards (UI-3) — without
	// this the toggle silently reverted to dark and persistence recorded "dark".
	*s = *DefaultStyles()
	s.CurrentTheme = theme.Name
}

// GetTheme returns the theme for a given theme type
func GetTheme(themeType ThemeType) Theme {
	switch themeType {
	case LightTheme:
		return LightThemeColors()
	default:
		return DarkThemeColors()
	}
}
