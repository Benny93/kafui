package ui

import "github.com/charmbracelet/lipgloss"

// Shared styles across pages
var (
	// Common styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#1a1a1a")).
			Padding(0, 1)

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
