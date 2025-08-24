package components

import (
	"github.com/charmbracelet/lipgloss"
)

// LayoutConfig holds configuration for the main layout
type LayoutConfig struct {
	Width         int
	Height        int
	SidebarWidth  int
	ShowSidebar   bool
	HeaderTitle   string
	ResourceType  string
}

// Layout represents a reusable layout component
type Layout struct {
	config LayoutConfig
}

// NewLayout creates a new layout component
func NewLayout(config LayoutConfig) *Layout {
	if config.SidebarWidth == 0 {
		config.SidebarWidth = 35
	}
	return &Layout{config: config}
}

// CalculateDimensions returns the calculated dimensions for layout components
func (l *Layout) CalculateDimensions() (contentWidth, contentHeight, sidebarWidth int) {
	sidebarWidth = l.config.SidebarWidth
	if !l.config.ShowSidebar {
		sidebarWidth = 0
	}
	
	contentWidth = l.config.Width - sidebarWidth - 6 // Account for padding and borders
	if contentWidth < 0 {
		contentWidth = 0
	}
	
	contentHeight = l.config.Height - 8 // Account for header and footer
	if contentHeight < 0 {
		contentHeight = 0
	}
	
	return contentWidth, contentHeight, sidebarWidth
}

// RenderHeader renders the application header
func (l *Layout) RenderHeader() string {
	if l.config.Width == 0 {
		return ""
	}
	
	resourceIndicator := ""
	if l.config.ResourceType != "" {
		resourceIndicator = ResourceTypeStyle.Render(l.config.ResourceType)
	}
	
	title := l.config.HeaderTitle
	if title == "" {
		title = "Kafui - Kafka UI"
	}
	
	return HeaderStyle.
		Width(l.config.Width).
		Render(resourceIndicator + title)
}

// CombineMainContent combines main content and sidebar
func (l *Layout) CombineMainContent(mainContent, sidebarContent string) string {
	_, contentHeight, sidebarWidth := l.CalculateDimensions()
	
	// In compact mode or when sidebar is disabled, just return main content
	if !l.config.ShowSidebar || sidebarWidth == 0 {
		return LayoutStyle.Render(mainContent)
	}
	
	// Style the sidebar
	sidebar := SidebarPanelStyle.
		Width(sidebarWidth).
		Height(contentHeight).
		Render(sidebarContent)
	
	// Combine main content and sidebar
	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		mainContent,
		lipgloss.NewStyle().Width(2).Render(""), // Spacer
		sidebar,
	)
	
	return LayoutStyle.Render(body)
}

// RenderComplete renders the complete layout with header, body, and footer
func (l *Layout) RenderComplete(mainContent, sidebarContent, footerContent string) string {
	if l.config.Width == 0 {
		return "Loading..."
	}
	
	header := l.RenderHeader()
	body := l.CombineMainContent(mainContent, sidebarContent)
	footer := FooterStyle.Width(l.config.Width).Render(footerContent)
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		body,
		footer,
	)
}

// UpdateConfig updates the layout configuration
func (l *Layout) UpdateConfig(config LayoutConfig) {
	if config.SidebarWidth == 0 {
		config.SidebarWidth = l.config.SidebarWidth
	}
	l.config = config
}

// GetConfig returns the current layout configuration
func (l *Layout) GetConfig() LayoutConfig {
	return l.config
}

// IsCompactMode determines if the layout should be in compact mode
func (l *Layout) IsCompactMode() bool {
	return l.config.Width < 100 || l.config.Height < 25
}