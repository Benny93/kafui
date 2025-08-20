package components

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// MainContentConfig holds configuration for the main content area
type MainContentConfig struct {
	Width       int
	Height      int
	ShowSearch  bool
	SearchBar   SearchBarModel
	List        list.Model
	CustomTitle string
}

// MainContent represents a reusable main content component
type MainContent struct {
	config MainContentConfig
}

// NewMainContent creates a new main content component
func NewMainContent(config MainContentConfig) *MainContent {
	return &MainContent{config: config}
}

// RenderSearchSection renders the search bar section
func (mc *MainContent) RenderSearchSection() string {
	if !mc.config.ShowSearch {
		return ""
	}
	
	searchStyle := lipgloss.NewStyle().
		BorderStyle(RoundedBorder).
		BorderForeground(Info).
		Padding(0, 1).
		MarginBottom(1).
		Width(mc.config.Width)
	
	return searchStyle.Render(mc.config.SearchBar.View())
}

// RenderListSection renders the list section
func (mc *MainContent) RenderListSection() string {
	listHeight := mc.config.Height
	if mc.config.ShowSearch {
		listHeight -= 3 // Account for search bar
	}
	
	if listHeight < 0 {
		listHeight = 0
	}
	
	return MainPanelStyle.
		Width(mc.config.Width).
		Height(listHeight).
		Render(mc.config.List.View())
}

// Render renders the complete main content area
func (mc *MainContent) Render() string {
	var sections []string
	
	// Add search section if enabled
	if searchSection := mc.RenderSearchSection(); searchSection != "" {
		sections = append(sections, searchSection)
	}
	
	// Add list section
	sections = append(sections, mc.RenderListSection())
	
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// UpdateConfig updates the main content configuration
func (mc *MainContent) UpdateConfig(config MainContentConfig) {
	mc.config = config
}

// GetConfig returns the current main content configuration
func (mc *MainContent) GetConfig() MainContentConfig {
	return mc.config
}

// SetSearchBar updates the search bar
func (mc *MainContent) SetSearchBar(searchBar SearchBarModel) {
	mc.config.SearchBar = searchBar
}

// SetList updates the list
func (mc *MainContent) SetList(list list.Model) {
	mc.config.List = list
}

// SetShowSearch updates the search visibility
func (mc *MainContent) SetShowSearch(show bool) {
	mc.config.ShowSearch = show
}

// SetDimensions updates the width and height
func (mc *MainContent) SetDimensions(width, height int) {
	mc.config.Width = width
	mc.config.Height = height
}