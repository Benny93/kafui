package components

import (
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
)

// ExamplePageModel demonstrates how to use the reusable components in other pages
type ExamplePageModel struct {
	width       int
	height      int
	searchMode  bool
	
	// Reusable components
	layout      *Layout
	sidebar     *Sidebar
	footer      *Footer
	mainContent *MainContent
	
	// Page-specific data
	list        list.Model
	searchBar   SearchBarModel
	spinner     spinner.Model
}

// NewExamplePage creates a new example page using reusable components
func NewExamplePage() ExamplePageModel {
	// Initialize components with different configurations
	layout := NewLayout(LayoutConfig{
		SidebarWidth: 40, // Different sidebar width
		ShowSidebar:  true,
	})
	
	sidebar := NewSidebar(SidebarConfig{
		Context:       "example-context",
		ShowResources: false, // Hide resources for this page
		ShowShortcuts: true,
		CustomSections: []SidebarSection{
			{
				Title:   "Custom Section",
				Content: "This is a custom section\nwith multiple lines\nof content",
			},
			{
				Title:   "Another Section",
				Content: "More custom content here",
			},
		},
	})
	
	footer := NewFooter(FooterConfig{
		StatusMessage: "Example page loaded",
		LastUpdate:    time.Now(),
	})
	
	mainContent := NewMainContent(MainContentConfig{
		ShowSearch: false, // No search for this example
	})
	
	return ExamplePageModel{
		layout:      layout,
		sidebar:     sidebar,
		footer:      footer,
		mainContent: mainContent,
	}
}

// View renders the example page
func (m ExamplePageModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}
	
	// Update layout configuration
	m.layout.UpdateConfig(LayoutConfig{
		Width:        m.width,
		Height:       m.height,
		SidebarWidth: 40,
		ShowSidebar:  true,
		HeaderTitle:  "Example Page - Reusable Components",
		ResourceType: "EXAMPLE",
	})
	
	// Calculate dimensions
	contentWidth, contentHeight, _ := m.layout.CalculateDimensions()
	
	// Update main content with example content
	m.mainContent.SetDimensions(contentWidth, contentHeight)
	m.mainContent.SetShowSearch(false)
	
	// Create example main content
	exampleContent := DocStyle.Render(`
This is an example page demonstrating how to use
the reusable UI components extracted from page_main.go.

Key benefits:
• Consistent layout across pages
• Reusable sidebar with configurable sections
• Flexible footer component
• Modular main content area
• Easy to maintain and extend

The layout automatically handles:
• Responsive sizing
• Proper spacing and margins
• Component positioning
• Style consistency
	`)
	
	// Update footer
	m.footer.UpdateConfig(FooterConfig{
		Width:         m.width,
		SearchMode:    false,
		SelectedItem:  "Example Item",
		TotalItems:    42,
		StatusMessage: "Example page ready",
		LastUpdate:    time.Now(),
		Spinner:       m.spinner,
	})
	
	// Render complete layout
	return m.layout.RenderComplete(
		exampleContent,
		m.sidebar.Render(),
		m.footer.Render(),
	)
}

// SetDimensions updates the page dimensions
func (m *ExamplePageModel) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}