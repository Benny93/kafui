// Package layout provides centralized layout management for the Kafui UI.
// It handles responsive breakpoints, component sizing, and layout calculations.
package layout

import (
	"math"
)

// Layout represents the complete layout configuration for the application
type Layout struct {
	// Total dimensions
	Width  int
	Height int

	// Component rectangles
	Header     Rectangle
	Sidebar    Rectangle
	Main       Rectangle
	Footer     Rectangle
	Search     Rectangle
	StatusBar  Rectangle

	// Layout mode
	Mode LayoutMode

	// Responsive state
	CompactMode bool
	SmallScreen bool
}

// Rectangle represents a rectangular area in the layout
type Rectangle struct {
	X      int
	Y      int
	Width  int
	Height int
}

// LayoutMode represents the current layout mode
type LayoutMode uint8

const (
	// LayoutNormal is the standard layout with all components visible
	LayoutNormal LayoutMode = iota

	// LayoutCompact is a space-optimized layout for small terminals
	LayoutCompact

	// LayoutMinimal is an ultra-minimal layout for very small terminals
	LayoutMinimal
)

// LayoutConfig contains configuration for layout calculations
type LayoutConfig struct {
	// ShowSidebar indicates whether sidebar should be shown
	ShowSidebar bool

	// ShowHeader indicates whether header should be shown
	ShowHeader bool

	// ShowFooter indicates whether footer should be shown
	ShowFooter bool

	// ShowStatusBar indicates whether status bar should be shown
	ShowStatusBar bool

	// SidebarWidth is the fixed width of the sidebar when visible
	SidebarWidth int

	// HeaderHeight is the fixed height of the header when visible
	HeaderHeight int

	// FooterHeight is the fixed height of the footer when visible
	FooterHeight int

	// SearchHeight is the height of the search bar when active
	SearchHeight int

	// StatusBarHeight is the height of the status bar
	StatusBarHeight int

	// Breakpoints for responsive design
	Breakpoints Breakpoints
}

// Breakpoints defines responsive breakpoints for different screen sizes
type Breakpoints struct {
	// CompactWidth triggers compact mode when terminal width is below this value
	CompactWidth int

	// MinimalWidth triggers minimal mode when terminal width is below this value
	MinimalWidth int

	// CompactHeight triggers compact mode when terminal height is below this value
	CompactHeight int

	// MinimalHeight triggers minimal mode when terminal height is below this value
	MinimalHeight int
}

// DefaultBreakpoints returns the default responsive breakpoints
func DefaultBreakpoints() Breakpoints {
	return Breakpoints{
		CompactWidth:  100,
		MinimalWidth:  60,
		CompactHeight: 24,
		MinimalHeight: 16,
	}
}

// DefaultLayoutConfig returns the default layout configuration
func DefaultLayoutConfig() *LayoutConfig {
	return &LayoutConfig{
		ShowSidebar:       true,
		ShowHeader:        true,
		ShowFooter:        true,
		ShowStatusBar:     true,
		SidebarWidth:      35,
		HeaderHeight:      3,
		FooterHeight:      3,
		SearchHeight:      3,
		StatusBarHeight:   1,
		Breakpoints:       DefaultBreakpoints(),
	}
}

// CalculateLayout computes the complete layout based on available space and configuration
func CalculateLayout(width, height int, config *LayoutConfig) *Layout {
	if config == nil {
		config = DefaultLayoutConfig()
	}

	layout := &Layout{
		Width:  width,
		Height: height,
		Mode:   LayoutNormal,
	}

	// Determine layout mode based on breakpoints
	layout.determineMode(config)

	// Calculate component positions and sizes based on mode
	switch layout.Mode {
	case LayoutNormal:
		layout.calculateNormalLayout(config)
	case LayoutCompact:
		layout.calculateCompactLayout(config)
	case LayoutMinimal:
		layout.calculateMinimalLayout(config)
	}

	return layout
}

// determineMode sets the layout mode based on terminal dimensions
func (l *Layout) determineMode(config *LayoutConfig) {
	breakpoints := config.Breakpoints

	// Check for minimal mode first (highest priority)
	if l.Width < breakpoints.MinimalWidth || l.Height < breakpoints.MinimalHeight {
		l.Mode = LayoutMinimal
		l.CompactMode = true
		l.SmallScreen = true
		return
	}

	// Check for compact mode
	if l.Width < breakpoints.CompactWidth || l.Height < breakpoints.CompactHeight {
		l.Mode = LayoutCompact
		l.CompactMode = true
		l.SmallScreen = false
		return
	}

	// Normal mode
	l.Mode = LayoutNormal
	l.CompactMode = false
	l.SmallScreen = false
}

// calculateNormalLayout computes layout for normal mode (all components visible)
func (l *Layout) calculateNormalLayout(config *LayoutConfig) {
	currentY := 0

	// Header
	if config.ShowHeader {
		l.Header = Rectangle{
			X:      0,
			Y:      currentY,
			Width:  l.Width,
			Height: config.HeaderHeight,
		}
		currentY += config.HeaderHeight
	}

	// Calculate main area width (accounting for sidebar)
	mainWidth := l.Width
	if config.ShowSidebar {
		mainWidth = l.Width - config.SidebarWidth
	}

	// Calculate main area height (accounting for footer and status bar)
	mainHeight := l.Height - currentY
	if config.ShowFooter {
		mainHeight -= config.FooterHeight
	}
	if config.ShowStatusBar {
		mainHeight -= config.StatusBarHeight
	}

	// Sidebar
	if config.ShowSidebar {
		l.Sidebar = Rectangle{
			X:      0,
			Y:      currentY,
			Width:  config.SidebarWidth,
			Height: mainHeight,
		}
	}

	// Main content area
	l.Main = Rectangle{
		X:      l.Sidebar.Width,
		Y:      currentY,
		Width:  mainWidth,
		Height: mainHeight,
	}

	// Status bar
	if config.ShowStatusBar {
		l.StatusBar = Rectangle{
			X:      0,
			Y:      l.Height - config.StatusBarHeight,
			Width:  l.Width,
			Height: config.StatusBarHeight,
		}
	}

	// Footer
	if config.ShowFooter {
		footerY := l.Height - config.FooterHeight
		if config.ShowStatusBar {
			footerY -= config.StatusBarHeight
		}
		l.Footer = Rectangle{
			X:      0,
			Y:      footerY,
			Width:  l.Width,
			Height: config.FooterHeight,
		}
	}
}

// calculateCompactLayout computes layout for compact mode (optimized for small screens)
func (l *Layout) calculateCompactLayout(config *LayoutConfig) {
	currentY := 0

	// Header (reduced height)
	headerHeight := int(math.Max(1, float64(config.HeaderHeight)-1))
	if config.ShowHeader {
		l.Header = Rectangle{
			X:      0,
			Y:      currentY,
			Width:  l.Width,
			Height: headerHeight,
		}
		currentY += headerHeight
	}

	// In compact mode, hide sidebar by default to save space
	showSidebar := config.ShowSidebar && l.Width >= config.Breakpoints.CompactWidth+20

	// Calculate main area width
	mainWidth := l.Width
	if showSidebar {
		mainWidth = l.Width - config.SidebarWidth
		l.Sidebar = Rectangle{
			X:      0,
			Y:      currentY,
			Width:  config.SidebarWidth,
			Height: l.Height - currentY - config.FooterHeight - config.StatusBarHeight,
		}
	}

	// Calculate main area height
	mainHeight := l.Height - currentY
	if config.ShowFooter {
		mainHeight -= config.FooterHeight
	}
	if config.ShowStatusBar {
		mainHeight -= config.StatusBarHeight
	}

	// Main content area
	l.Main = Rectangle{
		X:      l.Sidebar.Width,
		Y:      currentY,
		Width:  mainWidth,
		Height: mainHeight,
	}

	// Status bar
	if config.ShowStatusBar {
		l.StatusBar = Rectangle{
			X:      0,
			Y:      l.Height - config.StatusBarHeight,
			Width:  l.Width,
			Height: config.StatusBarHeight,
		}
	}

	// Footer
	if config.ShowFooter {
		footerY := l.Height - config.FooterHeight
		if config.ShowStatusBar {
			footerY -= config.StatusBarHeight
		}
		l.Footer = Rectangle{
			X:      0,
			Y:      footerY,
			Width:  l.Width,
			Height: config.FooterHeight,
		}
	}
}

// calculateMinimalLayout computes layout for minimal mode (ultra-small screens)
func (l *Layout) calculateMinimalLayout(config *LayoutConfig) {
	// Hide sidebar, header, and status bar in minimal mode
	// Only show main content and footer

	currentY := 0

	// No header in minimal mode
	l.Header = Rectangle{}

	// No sidebar in minimal mode
	l.Sidebar = Rectangle{}

	// Calculate main area height (only footer takes space)
	mainHeight := l.Height - config.FooterHeight

	// Main content area (full width)
	l.Main = Rectangle{
		X:      0,
		Y:      currentY,
		Width:  l.Width,
		Height: mainHeight,
	}

	// No status bar in minimal mode
	l.StatusBar = Rectangle{}

	// Footer (minimal height)
	l.Footer = Rectangle{
		X:      0,
		Y:      l.Height - 1,
		Width:  l.Width,
		Height: 1,
	}
}

// GetContentArea returns the available content area (main area minus padding)
func (l *Layout) GetContentArea(padding int) Rectangle {
	return Rectangle{
		X:      l.Main.X + padding,
		Y:      l.Main.Y + padding,
		Width:  int(math.Max(0, float64(l.Main.Width-2*padding))),
		Height: int(math.Max(0, float64(l.Main.Height-2*padding))),
	}
}

// IsSidebarVisible returns true if the sidebar should be visible
func (l *Layout) IsSidebarVisible() bool {
	return l.Sidebar.Width > 0 && l.Sidebar.Height > 0
}

// IsHeaderVisible returns true if the header should be visible
func (l *Layout) IsHeaderVisible() bool {
	return l.Header.Height > 0
}

// IsFooterVisible returns true if the footer should be visible
func (l *Layout) IsFooterVisible() bool {
	return l.Footer.Height > 0
}

// GetAvailableHeight returns the available height for content in the main area
func (l *Layout) GetAvailableHeight() int {
	return l.Main.Height
}

// GetAvailableWidth returns the available width for content in the main area
func (l *Layout) GetAvailableWidth() int {
	return l.Main.Width
}

// LayoutCalculator provides methods for calculating layout-related values
type LayoutCalculator struct {
	config *LayoutConfig
}

// NewLayoutCalculator creates a new layout calculator with the given configuration
func NewLayoutCalculator(config *LayoutConfig) *LayoutCalculator {
	if config == nil {
		config = DefaultLayoutConfig()
	}
	return &LayoutCalculator{
		config: config,
	}
}

// CalculateTableHeight calculates the appropriate table height for the main content area
func (lc *LayoutCalculator) CalculateTableHeight(layout *Layout, reservedLines int) int {
	if layout == nil {
		return 10 // Default fallback
	}

	availableHeight := layout.GetAvailableHeight()
	tableHeight := availableHeight - reservedLines

	// Ensure minimum height
	if tableHeight < 5 {
		tableHeight = 5
	}

	return tableHeight
}

// CalculateTableWidth calculates the appropriate table width for the main content area
func (lc *LayoutCalculator) CalculateTableWidth(layout *Layout, reservedColumns int) int {
	if layout == nil {
		return 40 // Default fallback
	}

	availableWidth := layout.GetAvailableWidth()
	tableWidth := availableWidth - reservedColumns

	// Ensure minimum width
	if tableWidth < 20 {
		tableWidth = 20
	}

	return tableWidth
}

// ShouldShowComponent determines if a component should be shown based on layout mode
func (lc *LayoutCalculator) ShouldShowComponent(layout *Layout, component string) bool {
	if layout == nil {
		return true
	}

	switch layout.Mode {
	case LayoutMinimal:
		// In minimal mode, only show essential components
		return component == "main" || component == "footer"
	case LayoutCompact:
		// In compact mode, show most components but hide non-essential ones
		return component != "status" && component != "header"
	default:
		// Normal mode shows all components
		return true
	}
}

// GetResponsiveBreakpoint returns the breakpoint that matches the current layout mode
func (lc *LayoutCalculator) GetResponsiveBreakpoint(layout *Layout) string {
	if layout == nil {
		return "normal"
	}

	switch layout.Mode {
	case LayoutMinimal:
		return "minimal"
	case LayoutCompact:
		return "compact"
	default:
		return "normal"
	}
}
