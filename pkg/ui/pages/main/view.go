package mainpage

import (
	"strings"

	"github.com/Benny93/kafui/pkg/ui/components"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/charmbracelet/lipgloss"
)

// View handles rendering for the main page
type View struct {
	dimensions core.Dimensions
	theme      core.Theme
	styles     *core.GlobalStyles
}

// NewView creates a new View instance
func NewView() *View {
	theme := core.DefaultTheme()
	return &View{
		theme:  theme,
		styles: core.NewGlobalStyles(theme),
	}
}

// Render renders the main page view
func (v *View) Render(model *Model) string {
	if model.dimensions.Width == 0 {
		return "Loading..."
	}

	// Update component configurations
	v.updateComponents(model)

	// Check for special states
	if model.error != nil {
		return v.RenderError(model, model.error)
	}

	if model.loading && len(model.allRows) == 0 {
		return v.RenderLoading(model)
	}

	if len(model.allRows) == 0 && !model.loading {
		return v.RenderEmpty(model)
	}

	// Render complete layout
	return model.layout.RenderComplete(
		model.mainContent.Render(),
		model.sidebar.Render(),
		model.footer.Render(),
	)
}

// SetDimensions updates the view dimensions
func (v *View) SetDimensions(width, height int) {
	v.dimensions = core.Dimensions{Width: width, Height: height}
}

// SetTheme updates the view theme
func (v *View) SetTheme(theme core.Theme) {
	v.theme = theme
	v.styles = core.NewGlobalStyles(theme)
}

func (v *View) updateComponents(model *Model) {
	// Determine layout mode based on screen size
	compactMode := model.dimensions.Width < 100 || model.dimensions.Height < 25

	// Update layout configuration
	layoutConfig := components.LayoutConfig{
		Width:       model.dimensions.Width,
		Height:      model.dimensions.Height,
		HeaderTitle: "Kafui - Kafka UI",
		ResourceType: strings.ToUpper(model.currentResource.GetType().String()),
	}

	// In compact mode, hide sidebar
	if !compactMode {
		layoutConfig.SidebarWidth = 35
		layoutConfig.ShowSidebar = true
	} else {
		layoutConfig.SidebarWidth = 0
		layoutConfig.ShowSidebar = false
	}

	model.layout.UpdateConfig(layoutConfig)

	// Calculate dimensions
	contentWidth, contentHeight, _ := model.layout.CalculateDimensions()

	// Update main content
	model.mainContent.SetDimensions(contentWidth, contentHeight)
	model.mainContent.SetSearchBar(model.searchBar)
	model.mainContent.SetTable(model.resourcesTable)
	model.mainContent.SetShowSearch(true)

	// Update sidebar (only if not in compact mode)
	if !compactMode {
		model.sidebar.UpdateConfig(components.SidebarConfig{
			Context:         model.dataSource.GetContext(),
			CurrentResource: components.ResourceType(model.currentResource.GetType()),
			ShowResources:   true,
			ShowShortcuts:   true,
		})
	}

	// Update footer
	selectedItem := model.GetSelectedItemName()

	model.footer.UpdateConfig(components.FooterConfig{
		Width:         model.dimensions.Width,
		SearchMode:    model.searchMode,
		SelectedItem:  selectedItem,
		TotalItems:    model.GetTotalItemCount(),
		StatusMessage: model.statusMessage,
		LastUpdate:    model.lastUpdate,
		Spinner:       model.spinner,
	})
}

// RenderError renders an error state
func (v *View) RenderError(model *Model, err error) string {
	if model.dimensions.Width == 0 {
		return "Error: " + err.Error()
	}

	// Update main content to show error
	model.mainContent.SetDimensions(model.dimensions.Width - 10, model.dimensions.Height - 10)

	// Update footer to show error
	model.footer.UpdateConfig(components.FooterConfig{
		Width:         model.dimensions.Width,
		SearchMode:    model.searchMode,
		SelectedItem:  "Error",
		TotalItems:    0,
		StatusMessage: "Error: " + err.Error(),
		LastUpdate:    model.lastUpdate,
		Spinner:       model.spinner,
	})

	return model.layout.RenderComplete(
		model.mainContent.Render(),
		model.sidebar.Render(),
		model.footer.Render(),
	)
}

// RenderLoading renders a loading state
func (v *View) RenderLoading(model *Model) string {
	if model.dimensions.Width == 0 {
		return "Loading..."
	}

	// Update status message to show loading
	model.statusMessage = "Loading resources... " + model.spinner.View()
	
	// Update footer to show loading state
	model.footer.UpdateConfig(components.FooterConfig{
		Width:         model.dimensions.Width,
		SearchMode:    model.searchMode,
		SelectedItem:  model.GetSelectedItemName(),
		TotalItems:    model.GetTotalItemCount(),
		StatusMessage: model.statusMessage,
		LastUpdate:    model.lastUpdate,
		Spinner:       model.spinner,
	})

	return model.layout.RenderComplete(
		model.mainContent.Render(),
		model.sidebar.Render(),
		model.footer.Render(),
	)
}

// RenderEmpty renders an empty state when no resources are available
func (v *View) RenderEmpty(model *Model) string {
	if model.dimensions.Width == 0 {
		return "No resources found"
	}

	// Update status message to show empty state
	model.statusMessage = "No resources found. Try refreshing or checking your connection."
	
	// Update footer to show empty state
	model.footer.UpdateConfig(components.FooterConfig{
		Width:         model.dimensions.Width,
		SearchMode:    model.searchMode,
		SelectedItem:  model.GetSelectedItemName(),
		TotalItems:    model.GetTotalItemCount(),
		StatusMessage: model.statusMessage,
		LastUpdate:    model.lastUpdate,
		Spinner:       model.spinner,
	})

	return model.layout.RenderComplete(
		model.mainContent.Render(),
		model.sidebar.Render(),
		model.footer.Render(),
	)
}

// Helper method to get status style based on message content
func (v *View) getStatusStyle(statusMessage string) lipgloss.Style {
	baseStyle := v.styles.InfoText()

	// Determine style based on message content
	if strings.Contains(strings.ToLower(statusMessage), "error") {
		return v.styles.ErrorText()
	} else if strings.Contains(strings.ToLower(statusMessage), "success") {
		return v.styles.SuccessText()
	} else if strings.Contains(strings.ToLower(statusMessage), "warning") {
		return v.styles.WarningText()
	}

	return baseStyle
}

// RenderHelp renders help information
func (v *View) RenderHelp(model *Model) string {
	helpContent := []string{
		"Main Page Help",
		"",
		"Navigation:",
		"  j/↓        Move down",
		"  k/↑        Move up",
		"  g/Home     Go to top",
		"  G/End      Go to bottom",
		"  Enter      Select item",
		"",
		"Search:",
		"  /          Start search",
		"  :          Switch resource type",
		"  Esc        Cancel search/back",
		"",
		"Resources:",
		"  topics           Show topics",
		"  consumer-groups  Show consumer groups",
		"  schemas          Show schemas",
		"  contexts         Show contexts",
		"",
		"General:",
		"  q/Ctrl+C   Quit application",
	}

	helpStyle := v.styles.InfoText().
		Width(model.dimensions.Width - 4).
		Padding(1)

	content := helpStyle.Render(strings.Join(helpContent, "\n"))

	return v.styles.Box().Render(content)
}