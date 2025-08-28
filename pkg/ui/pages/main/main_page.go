package mainpage

import (
	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// NewModel creates a new main page model (alias for NewMainPageModel for compatibility)
func NewModel(dataSource api.KafkaDataSource) *MainPageModel {
	return NewMainPageModel(dataSource)
}

// NewMainPageModel creates a new main page model using the template system
func NewMainPageModel(dataSource api.KafkaDataSource) *MainPageModel {
	// Create Kafui-specific providers
	contentProvider := NewKafuiContentProvider(dataSource)
	headerProvider := NewKafuiHeaderDataProvider(dataSource)
	
	// Create sidebar sections - convert to template provider interface
	sidebarSections := []providers.SidebarSection{
		NewResourcesSection(dataSource),
		NewClusterInfoSection(dataSource),
		NewShortcutsSection(),
	}
	
	// Create app configuration using template providers
	config := &providers.AppConfig{
		ContentProvider:             contentProvider,
		HeaderDataProvider:          headerProvider,
		SidebarSections:            sidebarSections,
		ShowSidebarByDefault:       true,
		CompactModeWidthBreakpoint: 120,
		CompactModeHeightBreakpoint: 30,
	}
	
	// Create the reusable app with our Kafui providers
	reusableApp := templateui.NewReusableApp(config)
	
	return &MainPageModel{
		dataSource:      dataSource,
		reusableApp:     reusableApp,
		contentProvider: contentProvider,
	}
}

// MainPageModel wraps the ReusableApp with Kafui-specific providers
type MainPageModel struct {
	dataSource      api.KafkaDataSource
	reusableApp     *templateui.ReusableApp
	contentProvider *KafuiContentProvider
}


// Init implements the Page interface
func (m *MainPageModel) Init() tea.Cmd {
	return m.reusableApp.Init()
}

// Update implements the Page interface
func (m *MainPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Delegate to the reusable app
	updatedApp, cmd := m.reusableApp.Update(msg)
	if updatedReusableApp, ok := updatedApp.(*templateui.ReusableApp); ok {
		m.reusableApp = updatedReusableApp
	}
	return m, cmd
}

// View implements the Page interface
func (m *MainPageModel) View() string {
	return m.reusableApp.View()
}

// SetDimensions implements the Page interface
func (m *MainPageModel) SetDimensions(width, height int) {
	// Delegate to the reusable app by sending a WindowSizeMsg
	m.reusableApp.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

// GetID implements the Page interface
func (m *MainPageModel) GetID() string {
	return "main"
}

// GetTitle implements the Page interface
func (m *MainPageModel) GetTitle() string {
	return "Kafui - Kafka TUI"
}

// GetHelp implements the Page interface
func (m *MainPageModel) GetHelp() []key.Binding {
	// Return key bindings for help
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "switch resource"),
		),
		key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back/cancel"),
		),
		key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// HandleNavigation implements the Page interface
func (m *MainPageModel) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
	// Handle navigation messages like NavigateToResourceDetailMsg
	switch msg := msg.(type) {
	case NavigateToResourceDetailMsg:
		// This would typically create and return a new resource detail page
		// For now, just return self
		_ = msg
		return m, nil
	}
	return m, nil
}

// OnFocus implements the Page interface
func (m *MainPageModel) OnFocus() tea.Cmd {
	// Handle focus gain - reload data when page becomes active
	return m.contentProvider.InitContent()
}

// OnBlur implements the Page interface
func (m *MainPageModel) OnBlur() tea.Cmd {
	// Handle focus loss
	return nil
}

// GetSelectedResourceItem returns the currently selected resource item (for compatibility)
func (m *MainPageModel) GetSelectedResourceItem() interface{} {
	return m.contentProvider.GetSelectedResourceItem()
}

