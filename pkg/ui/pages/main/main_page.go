package mainpage

import (
	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// NewModel creates a new main page model (alias for NewMainPageModel for compatibility)
// Deprecated: Use NewModelWithCommon for new code
func NewModel(dataSource api.KafkaDataSource) *MainPageModel {
	return NewMainPageModel(dataSource)
}

// NewMainPageModel creates a new main page model using the template system
// Deprecated: Use NewModelWithCommon for new code
func NewMainPageModel(dataSource api.KafkaDataSource) *MainPageModel {
	// Create Common context with data source
	common := core.NewCommon(dataSource)
	return NewModelWithCommon(common)
}

// NewModelWithCommon creates a new main page model using the Common context pattern
func NewModelWithCommon(common *core.Common) *MainPageModel {
	// Create Kafui-specific providers using Common context
	contentProvider := NewKafuiContentProviderWithCommon(common)
	headerProvider := NewKafuiHeaderDataProviderWithCommon(common)

	// Create sidebar sections - convert to template provider interface
	sidebarSections := []providers.SidebarSection{
		NewResourcesSectionWithCommon(common),
		NewClusterInfoSectionWithCommon(common),
		// Note: Removing ShortcutsSection as it will be shown in footer instead
	}

	// Create app configuration using template providers
	config := &providers.AppConfig{
		ContentProvider:             contentProvider,
		HeaderDataProvider:          headerProvider,
		SidebarSections:             sidebarSections,
		ShowSidebarByDefault:        true,
		CompactModeWidthBreakpoint:  120,
		CompactModeHeightBreakpoint: 30,
	}

	// Create the reusable app with our Kafui providers
	reusableApp := templateui.NewReusableApp(config)

	// Use centralized key bindings
	centralizedKeys := keys.DefaultKeyMap()
	reusableApp.SetKeyMap(centralizedKeys.Main)

	return &MainPageModel{
		common:          common,
		reusableApp:     reusableApp,
		contentProvider: contentProvider,
	}
}

// MainPageModel wraps the ReusableApp with Kafui-specific providers
type MainPageModel struct {
	common          *core.Common           // Shared context (replaces direct dataSource)
	dataSource      api.KafkaDataSource    // Kept for backward compatibility
	reusableApp     *templateui.ReusableApp
	contentProvider *KafuiContentProvider
}

// GetCommon returns the shared context
func (m *MainPageModel) GetCommon() *core.Common {
	return m.common
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
	// Return key bindings for help using centralized keys
	km := keys.DefaultKeyMap()
	return []key.Binding{
		km.Main.Search,
		km.Main.SwitchResource,
		km.Main.Select,
		km.Main.Back,
		km.Main.Quit,
		km.Main.Help,
	}
}

// HandleNavigation implements the Page interface
func (m *MainPageModel) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
	// Handle navigation messages like NavigateToResourceDetailMsg
	switch msg := msg.(type) {
	case NavigateToResourceDetailMsg:
		// Convert to PageChangeMsg for router to handle
		cmd := m.createPageChangeCommand(msg)
		return m, cmd
	}
	return m, nil
}

// createPageChangeCommand creates a PageChangeMsg for the router to handle
func (m *MainPageModel) createPageChangeCommand(msg NavigateToResourceDetailMsg) tea.Cmd {
	switch msg.ResourceType {
	case TopicResourceType:
		// Get topic details for navigation using Common context
		topicName := msg.ResourceID
		dataSource := m.common.DataSource
		topics, err := dataSource.GetTopics()
		var topicDetails api.Topic
		if err != nil || topics == nil {
			// If we can't get topics, create a basic topic structure
			topicDetails = api.Topic{
				NumPartitions:     1,
				ReplicationFactor: 1,
				ConfigEntries:     make(map[string]*string),
			}
		} else if details, exists := topics[topicName]; exists {
			topicDetails = details
		} else {
			// Topic not found, create a basic topic structure
			topicDetails = api.Topic{
				NumPartitions:     1,
				ReplicationFactor: 1,
				ConfigEntries:     make(map[string]*string),
			}
		}

		// Create navigation data
		navData := map[string]interface{}{
			"name":  topicName,
			"topic": topicDetails,
		}

		// Return PageChangeMsg for router to handle with a specific topic ID
		return core.NewPageChangeMsg("topic:"+topicName, navData)

	case ConsumerGroupResourceType:
		// Navigate to consumer group page (not implemented yet)
		return nil
	case SchemaResourceType:
		// Navigate to schema page (not implemented yet)
		return nil
	case ContextResourceType:
		// Navigate to context page (not implemented yet)
		return nil
	default:
		return nil
	}
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
