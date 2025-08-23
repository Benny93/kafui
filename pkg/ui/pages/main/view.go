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
	// Update layout configuration
	model.layout.UpdateConfig(components.LayoutConfig{
		Width:        model.dimensions.Width,
		Height:       model.dimensions.Height,
		SidebarWidth: 35,
		ShowSidebar:  true,
		HeaderTitle:  "Kafui - Kafka UI",
		ResourceType: strings.ToUpper(model.currentResource.GetType().String()),
	})

	// Calculate dimensions
	contentWidth, contentHeight, _ := model.layout.CalculateDimensions()

	// Update main content
	model.mainContent.SetDimensions(contentWidth, contentHeight)
	model.mainContent.SetSearchBar(model.searchBar)
	model.mainContent.SetTable(model.resourcesTable)
	model.mainContent.SetShowSearch(true)

	// Update sidebar
	model.sidebar.UpdateConfig(components.SidebarConfig{
		Context:         model.dataSource.GetContext(),
		CurrentResource: components.ResourceType(model.currentResource.GetType()),
		ShowResources:   true,
		ShowShortcuts:   true,
	})

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

	// errorStyle := v.styles.ErrorText().
	// 	Width(model.dimensions.Width - 4).
	// 	Align(lipgloss.Center).
	// 	Padding(2)

	// content := errorStyle.Render("Error: " + err.Error())

	// Still show the basic layout with error in main content
	v.updateComponents(model)

	// Override main content with error
	// TODO: Implement SetCustomContent in components.MainContent
	// model.mainContent.SetCustomContent(content)

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

	// loadingStyle := v.styles.InfoText().
	// 	Width(model.dimensions.Width - 4).
	// 	Align(lipgloss.Center).
	// 	Padding(2)

	// content := loadingStyle.Render("Loading resources...")

	// Still show the basic layout with loading in main content
	v.updateComponents(model)

	// Override main content with loading message
	// TODO: Implement SetCustomContent in components.MainContent
	// model.mainContent.SetCustomContent(content)

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

	// emptyStyle := v.styles.SecondaryText().
	// 	Width(model.dimensions.Width - 4).
	// 	Align(lipgloss.Center).
	// 	Padding(2)

	// content := emptyStyle.Render("No resources found. Try refreshing or checking your connection.")

	// Still show the basic layout with empty message in main content
	v.updateComponents(model)

	// Override main content with empty message
	// TODO: Implement SetCustomContent in components.MainContent
	// model.mainContent.SetCustomContent(content)

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
