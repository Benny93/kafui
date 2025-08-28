package mainpage

import (
	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// KeyMap defines the key bindings for the main page
type KeyMap struct {
	Search         key.Binding
	SwitchResource key.Binding
	Select         key.Binding
	Back           key.Binding
	Quit           key.Binding
	Help           key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Search, k.SwitchResource, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Search, k.SwitchResource, k.Select}, // first column
		{k.Back, k.Help, k.Quit},               // second column
	}
}

// DefaultKeyMap contains the default key bindings for the main page
var DefaultKeyMap = KeyMap{
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	SwitchResource: key.NewBinding(
		key.WithKeys(":"),
		key.WithHelp(":", "switch resource"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back/cancel"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
}

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
		// Note: Removing ShortcutsSection as it will be shown in footer instead
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
	
	// Set the key map for the footer
	reusableApp.SetKeyMap(DefaultKeyMap)
	
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
	// Return key bindings for help using the DefaultKeyMap
	return []key.Binding{
		DefaultKeyMap.Search,
		DefaultKeyMap.SwitchResource,
		DefaultKeyMap.Select,
		DefaultKeyMap.Back,
		DefaultKeyMap.Quit,
		DefaultKeyMap.Help,
	}
}

// HandleNavigation implements the Page interface
func (m *MainPageModel) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
	// Handle navigation messages like NavigateToResourceDetailMsg
	switch msg := msg.(type) {
	case NavigateToResourceDetailMsg:
		shared.DebugLog("MainPageModel.HandleNavigation: Received NavigateToResourceDetailMsg: %+v", msg)
		// Convert to PageChangeMsg for router to handle
		cmd := m.createPageChangeCommand(msg)
		shared.DebugLog("MainPageModel.HandleNavigation: Created PageChangeCommand, returning it")
		return m, cmd
	}
	shared.DebugLog("MainPageModel.HandleNavigation: Received unhandled message type: %T", msg)
	return m, nil
}

// createPageChangeCommand creates a PageChangeMsg for the router to handle
func (m *MainPageModel) createPageChangeCommand(msg NavigateToResourceDetailMsg) tea.Cmd {
	switch msg.ResourceType {
	case TopicResourceType:
		// Get topic details for navigation
		topicName := msg.ResourceID
		topics, err := m.dataSource.GetTopics()
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
		
		// Return PageChangeMsg for router to handle
		return core.NewPageChangeMsg("topic", navData)
		
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

