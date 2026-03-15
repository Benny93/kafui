package layout

import (
	"testing"
)

func TestCalculateLayout_Normal(t *testing.T) {
	config := DefaultLayoutConfig()
	layout := CalculateLayout(120, 40, config)

	if layout.Mode != LayoutNormal {
		t.Errorf("Expected LayoutNormal, got %v", layout.Mode)
	}

	if layout.CompactMode {
		t.Error("Expected CompactMode to be false")
	}

	if layout.SmallScreen {
		t.Error("Expected SmallScreen to be false")
	}
}

func TestCalculateLayout_Compact(t *testing.T) {
	config := DefaultLayoutConfig()
	layout := CalculateLayout(80, 20, config)

	if layout.Mode != LayoutCompact {
		t.Errorf("Expected LayoutCompact, got %v", layout.Mode)
	}

	if !layout.CompactMode {
		t.Error("Expected CompactMode to be true")
	}
}

func TestCalculateLayout_Minimal(t *testing.T) {
	config := DefaultLayoutConfig()
	layout := CalculateLayout(50, 12, config)

	if layout.Mode != LayoutMinimal {
		t.Errorf("Expected LayoutMinimal, got %v", layout.Mode)
	}

	if !layout.CompactMode {
		t.Error("Expected CompactMode to be true")
	}

	if !layout.SmallScreen {
		t.Error("Expected SmallScreen to be true")
	}
}

func TestCalculateLayout_NormalLayoutComponents(t *testing.T) {
	config := DefaultLayoutConfig()
	layout := CalculateLayout(120, 40, config)

	// Check header
	if !layout.IsHeaderVisible() {
		t.Error("Expected header to be visible")
	}
	if layout.Header.Y != 0 {
		t.Errorf("Expected header Y to be 0, got %d", layout.Header.Y)
	}
	if layout.Header.Width != 120 {
		t.Errorf("Expected header width to be 120, got %d", layout.Header.Width)
	}

	// Check sidebar
	if !layout.IsSidebarVisible() {
		t.Error("Expected sidebar to be visible")
	}
	if layout.Sidebar.Width != config.SidebarWidth {
		t.Errorf("Expected sidebar width to be %d, got %d", config.SidebarWidth, layout.Sidebar.Width)
	}

	// Check main area
	if layout.Main.X != config.SidebarWidth {
		t.Errorf("Expected main X to be %d, got %d", config.SidebarWidth, layout.Main.X)
	}
	if layout.Main.Width != 120-config.SidebarWidth {
		t.Errorf("Expected main width to be %d, got %d", 120-config.SidebarWidth, layout.Main.Width)
	}

	// Check footer
	if !layout.IsFooterVisible() {
		t.Error("Expected footer to be visible")
	}

	// Check status bar
	if layout.StatusBar.Height != config.StatusBarHeight {
		t.Errorf("Expected status bar height to be %d, got %d", config.StatusBarHeight, layout.StatusBar.Height)
	}
}

func TestCalculateLayout_MinimalLayoutComponents(t *testing.T) {
	config := DefaultLayoutConfig()
	layout := CalculateLayout(50, 12, config)

	// In minimal mode, sidebar and header should be hidden
	if layout.IsSidebarVisible() {
		t.Error("Expected sidebar to be hidden in minimal mode")
	}

	if layout.IsHeaderVisible() {
		t.Error("Expected header to be hidden in minimal mode")
	}

	// Main area should take full width
	if layout.Main.X != 0 {
		t.Errorf("Expected main X to be 0, got %d", layout.Main.X)
	}
	if layout.Main.Width != 50 {
		t.Errorf("Expected main width to be 50, got %d", layout.Main.Width)
	}

	// Footer should still be visible but minimal
	if !layout.IsFooterVisible() {
		t.Error("Expected footer to be visible")
	}
	if layout.Footer.Height != 1 {
		t.Errorf("Expected footer height to be 1, got %d", layout.Footer.Height)
	}
}

func TestCalculateLayout_ContentArea(t *testing.T) {
	config := DefaultLayoutConfig()
	layout := CalculateLayout(120, 40, config)

	// Test content area with padding
	contentArea := layout.GetContentArea(2)

	if contentArea.X != layout.Main.X+2 {
		t.Errorf("Expected content area X to be %d, got %d", layout.Main.X+2, contentArea.X)
	}

	if contentArea.Y != layout.Main.Y+2 {
		t.Errorf("Expected content area Y to be %d, got %d", layout.Main.Y+2, contentArea.Y)
	}

	expectedWidth := layout.Main.Width - 4
	if contentArea.Width != expectedWidth {
		t.Errorf("Expected content area width to be %d, got %d", expectedWidth, contentArea.Width)
	}

	expectedHeight := layout.Main.Height - 4
	if contentArea.Height != expectedHeight {
		t.Errorf("Expected content area height to be %d, got %d", expectedHeight, contentArea.Height)
	}
}

func TestCalculateLayout_AvailableDimensions(t *testing.T) {
	config := DefaultLayoutConfig()
	layout := CalculateLayout(120, 40, config)

	availableWidth := layout.GetAvailableWidth()
	availableHeight := layout.GetAvailableHeight()

	if availableWidth <= 0 {
		t.Error("Expected available width to be positive")
	}

	if availableHeight <= 0 {
		t.Error("Expected available height to be positive")
	}
}

func TestLayoutCalculator_TableDimensions(t *testing.T) {
	config := DefaultLayoutConfig()
	layout := CalculateLayout(120, 40, config)
	calc := NewLayoutCalculator(config)

	tableHeight := calc.CalculateTableHeight(layout, 5)
	if tableHeight < 5 {
		t.Errorf("Expected table height to be at least 5, got %d", tableHeight)
	}

	tableWidth := calc.CalculateTableWidth(layout, 10)
	if tableWidth < 20 {
		t.Errorf("Expected table width to be at least 20, got %d", tableWidth)
	}
}

func TestLayoutCalculator_TableDimensions_Minimum(t *testing.T) {
	config := DefaultLayoutConfig()
	// Very small layout
	layout := CalculateLayout(30, 10, config)
	calc := NewLayoutCalculator(config)

	tableHeight := calc.CalculateTableHeight(layout, 5)
	if tableHeight < 5 {
		t.Errorf("Expected minimum table height to be 5, got %d", tableHeight)
	}

	tableWidth := calc.CalculateTableWidth(layout, 20)
	if tableWidth < 20 {
		t.Errorf("Expected minimum table width to be 20, got %d", tableWidth)
	}
}

func TestLayoutCalculator_ShouldShowComponent(t *testing.T) {
	config := DefaultLayoutConfig()
	calc := NewLayoutCalculator(config)

	// Normal mode
	normalLayout := CalculateLayout(120, 40, config)
	if !calc.ShouldShowComponent(normalLayout, "header") {
		t.Error("Expected header to be shown in normal mode")
	}
	if !calc.ShouldShowComponent(normalLayout, "status") {
		t.Error("Expected status bar to be shown in normal mode")
	}

	// Minimal mode
	minimalLayout := CalculateLayout(50, 12, config)
	if calc.ShouldShowComponent(minimalLayout, "header") {
		t.Error("Expected header to be hidden in minimal mode")
	}
	if calc.ShouldShowComponent(minimalLayout, "status") {
		t.Error("Expected status bar to be hidden in minimal mode")
	}
	if !calc.ShouldShowComponent(minimalLayout, "main") {
		t.Error("Expected main area to be shown in minimal mode")
	}
}

func TestLayoutCalculator_ResponsiveBreakpoint(t *testing.T) {
	config := DefaultLayoutConfig()
	calc := NewLayoutCalculator(config)

	normalLayout := CalculateLayout(120, 40, config)
	if calc.GetResponsiveBreakpoint(normalLayout) != "normal" {
		t.Error("Expected 'normal' breakpoint")
	}

	compactLayout := CalculateLayout(80, 20, config)
	if calc.GetResponsiveBreakpoint(compactLayout) != "compact" {
		t.Error("Expected 'compact' breakpoint")
	}

	minimalLayout := CalculateLayout(50, 12, config)
	if calc.GetResponsiveBreakpoint(minimalLayout) != "minimal" {
		t.Error("Expected 'minimal' breakpoint")
	}
}

func TestDefaultBreakpoints(t *testing.T) {
	breakpoints := DefaultBreakpoints()

	if breakpoints.CompactWidth != 100 {
		t.Errorf("Expected CompactWidth to be 100, got %d", breakpoints.CompactWidth)
	}

	if breakpoints.MinimalWidth != 60 {
		t.Errorf("Expected MinimalWidth to be 60, got %d", breakpoints.MinimalWidth)
	}

	if breakpoints.CompactHeight != 24 {
		t.Errorf("Expected CompactHeight to be 24, got %d", breakpoints.CompactHeight)
	}

	if breakpoints.MinimalHeight != 16 {
		t.Errorf("Expected MinimalHeight to be 16, got %d", breakpoints.MinimalHeight)
	}
}

func TestDefaultLayoutConfig(t *testing.T) {
	config := DefaultLayoutConfig()

	if !config.ShowSidebar {
		t.Error("Expected ShowSidebar to be true")
	}

	if !config.ShowHeader {
		t.Error("Expected ShowHeader to be true")
	}

	if !config.ShowFooter {
		t.Error("Expected ShowFooter to be true")
	}

	if config.SidebarWidth != 35 {
		t.Errorf("Expected SidebarWidth to be 35, got %d", config.SidebarWidth)
	}
}

func TestCalculateLayout_EdgeCases(t *testing.T) {
	config := DefaultLayoutConfig()

	// Very large terminal
	largeLayout := CalculateLayout(500, 200, config)
	if largeLayout.Mode != LayoutNormal {
		t.Error("Expected large terminal to use normal layout")
	}

	// Exact breakpoint (at boundary, should use normal)
	exactLayout := CalculateLayout(100, 24, config)
	if exactLayout.Mode != LayoutNormal {
		t.Error("Expected exact breakpoint to use normal layout")
	}

	// Zero dimensions (should not panic)
	zeroLayout := CalculateLayout(0, 0, config)
	if zeroLayout.Mode != LayoutMinimal {
		t.Error("Expected zero dimensions to use minimal layout")
	}
}

func TestCommon_UpdateLayout(t *testing.T) {
	// This test would require the core package to be tested separately
	// The layout update functionality is tested in the layout package tests
}
