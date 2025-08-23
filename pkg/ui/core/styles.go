package core

import (
	"github.com/charmbracelet/lipgloss"
)

// DefaultTheme returns the default application theme
func DefaultTheme() Theme {
	return Theme{
		Primary:   "#7D56F4",
		Secondary: "#383838",
		Accent:    "#73F59F",
		Error:     "#F25D94",
		Success:   "#73F59F",
		Warning:   "#F9F295",
		Info:      "#7D56F4",
	}
}

// DarkTheme returns a dark theme variant
func DarkTheme() Theme {
	return Theme{
		Primary:   "#8B5CF6",
		Secondary: "#1F2937",
		Accent:    "#10B981",
		Error:     "#EF4444",
		Success:   "#10B981",
		Warning:   "#F59E0B",
		Info:      "#3B82F6",
	}
}

// GlobalStyles contains commonly used styles across the application
type GlobalStyles struct {
	Theme Theme
}

// NewGlobalStyles creates a new GlobalStyles instance with the given theme
func NewGlobalStyles(theme Theme) *GlobalStyles {
	return &GlobalStyles{
		Theme: theme,
	}
}

// BorderStyles for different use cases
func (gs *GlobalStyles) PrimaryBorder() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(gs.Theme.Primary))
}

func (gs *GlobalStyles) SecondaryBorder() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(gs.Theme.Secondary))
}

func (gs *GlobalStyles) AccentBorder() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(gs.Theme.Accent))
}

// Text styles for different purposes
func (gs *GlobalStyles) HeaderText() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(gs.Theme.Primary)).
		Bold(true)
}

func (gs *GlobalStyles) ErrorText() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(gs.Theme.Error)).
		Bold(true)
}

func (gs *GlobalStyles) SuccessText() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(gs.Theme.Success)).
		Bold(true)
}

func (gs *GlobalStyles) WarningText() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(gs.Theme.Warning)).
		Bold(true)
}

func (gs *GlobalStyles) InfoText() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(gs.Theme.Info))
}

func (gs *GlobalStyles) SecondaryText() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(gs.Theme.Secondary))
}

// Background styles
func (gs *GlobalStyles) PrimaryBackground() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(gs.Theme.Primary))
}

func (gs *GlobalStyles) AccentBackground() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(gs.Theme.Accent))
}

func (gs *GlobalStyles) ErrorBackground() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(gs.Theme.Error))
}

// Layout styles
func (gs *GlobalStyles) Box() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(gs.Theme.Secondary)).
		Padding(1)
}

func (gs *GlobalStyles) Container() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(gs.Theme.Secondary)).
		Padding(0, 1)
}

func (gs *GlobalStyles) Panel() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(gs.Theme.Primary)).
		Padding(1).
		MarginBottom(1)
}

// Button styles
func (gs *GlobalStyles) PrimaryButton() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(gs.Theme.Primary)).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 2).
		Bold(true)
}

func (gs *GlobalStyles) SecondaryButton() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(gs.Theme.Secondary)).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 2)
}

func (gs *GlobalStyles) AccentButton() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(gs.Theme.Accent)).
		Foreground(lipgloss.Color("#000000")).
		Padding(0, 2).
		Bold(true)
}

// Table styles
func (gs *GlobalStyles) TableHeader() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
}

func (gs *GlobalStyles) TableSelected() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	}

func (gs *GlobalStyles) TableHighlight() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(gs.Theme.Accent)).
		Foreground(lipgloss.Color("#000000")).
		Bold(true)
}

// Status styles based on status type
func (gs *GlobalStyles) StatusStyle(statusType StatusType) lipgloss.Style {
	switch statusType {
	case StatusError:
		return gs.ErrorText()
	case StatusSuccess:
		return gs.SuccessText()
	case StatusWarning:
		return gs.WarningText()
	case StatusInfo:
		return gs.InfoText()
	default:
		return gs.SecondaryText()
	}
}

// Common dimensions and spacing
const (
	DefaultPadding = 1
	DefaultMargin  = 1
	HeaderHeight   = 3
	FooterHeight   = 3
	SidebarWidth   = 35
)

// Helper functions for common styling operations
func WithTheme(style lipgloss.Style, theme Theme) lipgloss.Style {
	return style.
		BorderForeground(lipgloss.Color(theme.Primary)).
		Foreground(lipgloss.Color(theme.Primary))
}

func CenterAlign(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center)
}

func RightAlign(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Right)
}

func LeftAlign(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Left)
}