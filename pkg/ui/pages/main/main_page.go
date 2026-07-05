package mainpage

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	"github.com/Benny93/kafui/pkg/ui/shared"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// NewModelWithCommon creates a new main page model using the Common context pattern
func NewModelWithCommon(common *core.Common) *MainPageModel {
	// Create Kafui-specific providers using Common context
	contentProvider := NewKafuiContentProviderWithCommon(common)
	headerProvider := NewKafuiHeaderDataProviderWithCommon(common)

	// Create sidebar sections - convert to template provider interface
	resourcesSection := NewResourcesSectionWithCommon(common)
	sidebarSections := []providers.SidebarSection{
		resourcesSection,
		NewBrokerSummarySection(),
		NewClusterInfoSectionWithCommon(common),
		NewShortcutsSection(),
	}

	// CLI --resource deep-link (UI-9/BUG-7): apply it synchronously here, before
	// Init() runs, so the content provider's default-resource Init() and the
	// sidebar/breadcrumb all reflect it from the start. Consumed once — an
	// earlier design fired this as an async message after Init(), which raced
	// the page's own default (Topics) initialization and usually lost, leaving
	// the sidebar highlight and footer breadcrumb stuck on "Topics".
	if common.InitialResource != "" {
		if rt := contentProvider.parseResourceType(common.InitialResource); rt != -1 {
			contentProvider.switchResource(SwitchResourceMsg(rt))
			resourcesSection.currentResource = rt
		}
		common.InitialResource = ""
	}

	// Restore the persisted sidebar preference (UI-15); small-size auto-collapse
	// still wins at render time inside ReusableApp.
	showSidebar := true
	if common.Config != nil {
		showSidebar = common.Config.ShowSidebar
	}

	// Create app configuration using template providers
	config := &providers.AppConfig{
		ContentProvider:             contentProvider,
		HeaderDataProvider:          headerProvider,
		SidebarSections:             sidebarSections,
		ShowSidebarByDefault:        showSidebar,
		CompactModeWidthBreakpoint:  120,
		CompactModeHeightBreakpoint: 30,
	}

	// Create the reusable app with our Kafui providers
	reusableApp := templateui.NewReusableApp(config)

	// Use centralized key bindings
	centralizedKeys := keys.DefaultKeyMap()
	reusableApp.SetKeyMap(centralizedKeys.Main)

	return &MainPageModel{
		common:           common,
		reusableApp:      reusableApp,
		contentProvider:  contentProvider,
		resourcesSection: resourcesSection,
	}
}

// MainPageModel wraps the ReusableApp with Kafui-specific providers
type MainPageModel struct {
	common           *core.Common
	reusableApp      *templateui.ReusableApp
	contentProvider  *KafuiContentProvider
	resourcesSection *ResourcesSection
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

// IsInputMode reports whether the page is capturing raw text (resource picker,
// search, or a create/edit form). The root model checks this so single-key
// global hotkeys (q quit, C, K, T, ?, …) don't fire while the user is typing —
// e.g. typing "q" into the ":" resource picker must not quit the app.
func (m *MainPageModel) IsInputMode() bool {
	return m.contentProvider != nil && m.contentProvider.IsInputMode()
}

// GetHelp implements the Page interface
func (m *MainPageModel) GetHelp() []key.Binding {
	// Return key bindings for help using centralized keys
	km := keys.DefaultKeyMap()
	return []key.Binding{
		km.Main.Search,
		km.Main.SwitchResource,
		km.Main.Select,
		km.Main.ScrollUp,
		km.Main.ScrollDown,
		km.Main.PageUp,
		km.Main.PageDown,
		km.Main.GotoStart,
		km.Main.GotoEnd,
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
		topicName := msg.ResourceID

		// Use the topic details already cached in the selected item — no network call.
		topicDetails := topicDetailsFromItem(msg.Item)

		navData := map[string]interface{}{
			"name":  topicName,
			"topic": topicDetails,
		}

		// Return PageChangeMsg for router to handle with a specific topic ID
		return core.NewPageChangeMsg("topic:"+topicName, navData)

	case ConsumerGroupResourceType:
		// Navigate to the dedicated consumer group detail page. The router
		// extracts "groupID" (string) from this map; it can also parse the id
		// from the "consumer_group:<id>" page ID.
		groupID := msg.ResourceID
		navData := map[string]interface{}{
			"groupID": groupID,
		}
		return core.NewPageChangeMsg("consumer_group:"+groupID, navData)
	case SchemaResourceType:
		// Unwrap the shared.ResourceListItem wrapper that allItems uses.
		var sri *SchemaResourceItem
		switch v := msg.Item.(type) {
		case *SchemaResourceItem:
			sri = v
		case shared.ResourceListItem:
			sri, _ = v.ResourceItem.(*SchemaResourceItem)
		}
		if sri != nil {
			navData := map[string]interface{}{
				"schemaItem": sri,
			}
			return core.NewPageChangeMsg("schema_detail:"+sri.Subject(), navData)
		}
		return nil
	case BrokerResourceType:
		// Navigate to the broker detail page, passing the already-loaded broker
		// info so the page can render its summary strip without a refetch.
		// The router extracts "brokerID" (int32) and optional "brokerInfo"
		// (api.BrokerInfo) from this map.
		if bri, ok := brokerItemFrom(msg.Item); ok {
			navData := map[string]interface{}{
				"brokerID":   bri.info.ID,
				"brokerInfo": bri.info,
			}
			return core.NewPageChangeMsg(fmt.Sprintf("broker:%d", bri.info.ID), navData)
		}
		return nil
	case ConnectorResourceType:
		// Navigate to the connector detail page. The router builds the page from
		// the "connect"/"name" keys (or by parsing "connector:<connect>:<name>").
		var conn api.Connector
		switch v := msg.Item.(type) {
		case *ConnectorResourceItem:
			conn = v.Connector()
		case shared.ResourceListItem:
			if ci, ok := v.ResourceItem.(*ConnectorResourceItem); ok {
				conn = ci.Connector()
			}
		case shared.HighlightedResourceListItem:
			if ci, ok := v.ResourceItem.(*ConnectorResourceItem); ok {
				conn = ci.Connector()
			}
		}
		if conn.Name == "" {
			return nil
		}
		navData := map[string]interface{}{
			"connect": conn.ConnectCluster,
			"name":    conn.Name,
		}
		return core.NewPageChangeMsg("connector:"+conn.ConnectCluster+":"+conn.Name, navData)
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

// topicDetailsFromItem extracts api.Topic from the already-loaded navigation item,
// avoiding a synchronous GetTopics() network call on navigation.
func topicDetailsFromItem(item interface{}) api.Topic {
	fallback := api.Topic{
		NumPartitions:     1,
		ReplicationFactor: 1,
		ConfigEntries:     make(map[string]*string),
	}

	var resourceItem ResourceItem
	switch i := item.(type) {
	case shared.ResourceListItem:
		resourceItem = i.ResourceItem
	case shared.HighlightedResourceListItem:
		resourceItem = i.ResourceItem
	default:
		return fallback
	}

	if tri, ok := resourceItem.(*TopicResourceItem); ok {
		topic := tri.GetTopic()
		// Propagate the asynchronously-loaded count so the topic page can skip
		// the fetch immediately when the topic is known to be empty.
		topic.MessageCount = tri.messageCount
		return topic
	}
	return fallback
}
