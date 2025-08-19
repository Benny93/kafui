package ui

import "github.com/charmbracelet/lipgloss"

// Shared styles across pages
var (
	// Colors
	Subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	Highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	Special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	Info      = lipgloss.AdaptiveColor{Light: "#4A90E2", Dark: "#4A90E2"}
	Warning   = lipgloss.AdaptiveColor{Light: "#F5A623", Dark: "#F5A623"}

	// Border styles
	RoundedBorder = lipgloss.Border{
		Top:         "",
		Bottom:      "",
		Left:        "",
		Right:       "",
		TopLeft:     "",
		TopRight:    "",
		BottomLeft:  "",
		BottomRight: "",
	}

	// Header styles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(Highlight).
			Padding(0, 1).
			MarginBottom(1)

	// Main layout styles
	LayoutStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Content panel styles
	MainPanelStyle = lipgloss.NewStyle().
			BorderStyle(RoundedBorder).
			BorderForeground(Subtle).
			Padding(1, 1)

	SidebarPanelStyle = lipgloss.NewStyle().
				BorderStyle(RoundedBorder).
				BorderForeground(Subtle).
				Padding(1, 2)

	// Footer styles
	FooterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(Subtle).
			Padding(0, 1)

	// Text styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#1a1a1a")).
			Padding(0, 1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Special).
			Bold(true).
			MarginBottom(1)

	InfoStyle = lipgloss.NewStyle().
			Foreground(Subtle).
			Italic(true)

	// Resource type indicator
	ResourceTypeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(Info).
				Bold(true).
				Padding(0, 1).
				MarginRight(1)

	// Common styles
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#3c3c3c")).
			Padding(0, 1)

	docStyle = lipgloss.NewStyle().
			Margin(1, 2)

	// Shared component styles
	tableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	sharedSidebarStyle = lipgloss.NewStyle().
				Width(30).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(1, 2)

	sidebarTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true).
				Align(lipgloss.Center)

	sidebarContentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("250")).
				PaddingTop(1)

	searchBarStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	// Custom colors
	highlightColor = lipgloss.Color("205")
)
