package mainpage

import (
	"fmt"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the main page state
type Model struct {
	// Data
	dataSource api.KafkaDataSource

	// State
	dimensions    core.Dimensions
	loading       bool
	searchMode    bool
	error         error
	lastUpdate    time.Time
	statusMessage string

	// Filter state tracking
	isFiltered    bool
	currentFilter string

	// Components
	handlers *Handlers
	keys     *Keys
	view     *View

	// UI Components
	resourcesTable table.Model
	searchBar      components.SearchBarModel
	spinner        spinner.Model

	// Data storage
	allRows       []table.Row
	allItems      []interface{} // Store original items to maintain data connectivity
	filteredRows  []table.Row
	filteredItems []interface{} // Store filtered original items

	// Resource management
	resourceManager *ResourceManager
	currentResource Resource

	// Reusable UI components
	layout      *components.Layout
	sidebar     *components.Sidebar
	footer      *components.Footer
	mainContent *components.MainContent
}

// Dimensions represents width and height
type Dimensions struct {
	Width  int
	Height int
}

// NewModel creates a new main page model
func NewModel(dataSource api.KafkaDataSource) *Model {
	// Initialize resources table
	columns := createResourceTableColumns()
	resourcesTable := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	resourcesTable.SetStyles(createResourceTableStyles())

	// Initialize resource manager
	resourceManager := NewResourceManager(dataSource)
	currentResource := resourceManager.GetResource(TopicResourceType)

	// Initialize search bar
	searchBar := components.NewSearchBar(
		components.WithPlaceholder("Press / to search, : to switch resource..."),
		components.WithOnSearch(func(query string) tea.Msg {
			return SearchTopicsMsg(query)
		}),
		components.WithOnClear(func() tea.Msg {
			return ClearSearchMsg{}
		}),
		components.WithOnResourceSwitch(func(resource string) tea.Msg {
			return SwitchResourceByNameMsg(resource)
		}),
		components.WithSearchSuggestions([]string{}), // Will be populated dynamically
	)

	// Initialize spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Initialize reusable components
	layout := components.NewLayout(components.LayoutConfig{})
	sidebar := components.NewSidebar(components.SidebarConfig{})
	footer := components.NewFooter(components.FooterConfig{})
	mainContent := components.NewMainContent(components.MainContentConfig{})

	m := &Model{
		dataSource:      dataSource,
		resourcesTable:  resourcesTable,
		searchBar:       searchBar,
		spinner:         sp,
		lastUpdate:      time.Now(),
		statusMessage:   "Welcome to Kafui",
		searchMode:      false,
		allRows:         []table.Row{},
		resourceManager: resourceManager,
		currentResource: currentResource,
		// Initialize filter state
		isFiltered:    false,
		currentFilter: "",
		filteredRows:  []table.Row{},
		layout:        layout,
		sidebar:       sidebar,
		footer:        footer,
		mainContent:   mainContent,
	}

	// Initialize components with dependencies
	m.handlers = NewHandlers(m)
	m.keys = NewKeys()
	m.view = NewView()

	return m
}

// Init implements the Page interface
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadTopics(),
		m.spinner.Tick,
		m.updateTimer(),
	)
}

// Update implements the Page interface
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.handlers.Handle(m, msg)
}

// View implements the Page interface
func (m *Model) View() string {
	return m.view.Render(m)
}

// SetDimensions implements the Page interface
func (m *Model) SetDimensions(width, height int) {
	m.dimensions = core.Dimensions{Width: width, Height: height}

	// Calculate available space for content
	sidebarWidth := 35
	mainContentWidth := width - sidebarWidth - 6 // Account for margins and borders
	contentHeight := height - 8                  // Account for header, footer and margins

	// Update table and search bar dimensions
	m.resourcesTable.SetHeight(contentHeight - 3) // Account for borders and margins
	m.searchBar.SetWidth(mainContentWidth - 4)

	m.view.SetDimensions(width, height)
}

// GetID implements the Page interface
func (m *Model) GetID() string {
	return "main"
}

// Business logic methods

// GetSelectedResourceItem returns the currently selected resource item from table
func (m *Model) GetSelectedResourceItem() interface{} {
	selectedRow := m.resourcesTable.Cursor()

	// Use filtered items if we're currently in a filtered state
	if m.isFiltered && len(m.filteredItems) > 0 {
		if selectedRow < 0 || selectedRow >= len(m.filteredItems) {
			return nil
		}
		return m.filteredItems[selectedRow]
	}

	// Otherwise use all items
	if selectedRow < 0 || selectedRow >= len(m.allItems) {
		return nil
	}
	return m.allItems[selectedRow]
}

// IsSearchMode returns whether the page is currently in search mode
func (m *Model) IsSearchMode() bool {
	return m.searchMode
}

// GetSelectedItemName returns the name of the currently selected item
func (m *Model) GetSelectedItemName() string {
	if selectedItem := m.GetSelectedResourceItem(); selectedItem != nil {
		if rowStruct, ok := selectedItem.(struct{ ID string }); ok {
			return rowStruct.ID
		}
		// Handle different item types
		switch item := selectedItem.(type) {
		case shared.ResourceListItem:
			return item.ResourceItem.GetID()
		case shared.TopicListItem:
			return item.Name
		case shared.ConsumerGroupListItem:
			return item.GroupID
		default:
			return "Unknown"
		}
	}
	return "None"
}

// GetTotalItemCount returns the total number of items
func (m *Model) GetTotalItemCount() int {
	return len(m.allRows)
}

// GetStatusMessage returns the current status message
func (m *Model) GetStatusMessage() string {
	return m.statusMessage
}

// GetLastUpdate returns the last update time
func (m *Model) GetLastUpdate() time.Time {
	return m.lastUpdate
}

// Data loading methods

func (m *Model) loadTopics() tea.Cmd {
	return func() tea.Msg {
		topics, err := m.dataSource.GetTopics()
		if err != nil {
			return ErrorMsg(err)
		}

		topicItems := make([]TopicItem, 0, len(topics))
		for topicName, topic := range topics {
			topicItems = append(topicItems, TopicItem{
				name:  topicName,
				topic: topic,
			})
		}

		return TopicListMsg(topicItems)
	}
}

func (m *Model) loadCurrentResource() tea.Cmd {
	return func() tea.Msg {
		items, err := m.currentResource.GetData()
		if err != nil {
			return ErrorMsg(err)
		}

		// Convert resource items to interface slice
		interfaceItems := make([]interface{}, 0, len(items))
		for _, item := range items {
			interfaceItems = append(interfaceItems, shared.ResourceListItem{
				ResourceItem: item,
			})
		}

		return CurrentResourceListMsg{
			ResourceType: m.currentResource.GetType(),
			Items:        interfaceItems,
		}
	}
}

func (m *Model) updateTimer() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return TimerTickMsg(t)
	})
}

// Utility methods

// Table configuration for resource list
func createResourceTableColumns() []table.Column {
	return []table.Column{
		{Title: "Name", Width: 35},
		{Title: "Type", Width: 15},
		{Title: "Partitions", Width: 12},
		{Title: "Replication", Width: 12},
		{Title: "Details", Width: 25},
	}
}

// Table styles for resource list
func createResourceTableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	return s
}

// Convert resource items to table rows
func convertItemsToRows(items []interface{}, searchQuery string) []table.Row {
	rows := make([]table.Row, 0, len(items))

	for _, item := range items {
		var name, resourceType, partitions, replication, details string

		switch i := item.(type) {
		case shared.ResourceListItem:
			name = i.ResourceItem.GetID()
			// Determine resource type based on the concrete type
			switch i.ResourceItem.(type) {
			case *TopicResourceItem:
				resourceType = "topic"
			case *ConsumerGroupResourceItem:
				resourceType = "consumer-group"
			case *SchemaResourceItem:
				resourceType = "schema"
			case *ContextResourceItem:
				resourceType = "context"
			default:
				resourceType = "unknown"
			}
			itemDetails := i.ResourceItem.GetDetails()
			if p, ok := itemDetails["Partitions"]; ok {
				partitions = p
			} else {
				partitions = "-"
			}
			if r, ok := itemDetails["Replication Factor"]; ok {
				replication = r
			} else if r, ok := itemDetails["State"]; ok {
				replication = r // For consumer groups, use state
			} else {
				replication = "-"
			}
			if d, ok := itemDetails["Consumers"]; ok {
				details = d + " consumers"
			} else if d, ok := itemDetails["Message Count"]; ok {
				details = d + " messages"
			} else {
				details = "-"
			}
		case TopicItem:
			name = i.name
			resourceType = "topic"
			partitions = fmt.Sprintf("%d", i.topic.NumPartitions)
			replication = fmt.Sprintf("%d", i.topic.ReplicationFactor)
			details = fmt.Sprintf("%d configs", len(i.topic.ConfigEntries))
		case shared.HighlightedResourceListItem:
			name = i.ResourceItem.GetID()
			// Determine resource type based on the concrete type
			switch i.ResourceItem.(type) {
			case *TopicResourceItem:
				resourceType = "topic"
			case *ConsumerGroupResourceItem:
				resourceType = "consumer-group"
			case *SchemaResourceItem:
				resourceType = "schema"
			case *ContextResourceItem:
				resourceType = "context"
			default:
				resourceType = "unknown"
			}
			itemDetails := i.ResourceItem.GetDetails()
			if p, ok := itemDetails["Partitions"]; ok {
				partitions = p
			} else {
				partitions = "-"
			}
			if r, ok := itemDetails["Replication Factor"]; ok {
				replication = r
			} else if r, ok := itemDetails["State"]; ok {
				replication = r // For consumer groups, use state
			} else {
				replication = "-"
			}
			if d, ok := itemDetails["Consumers"]; ok {
				details = d + " consumers"
			} else if d, ok := itemDetails["Message Count"]; ok {
				details = d + " messages"
			} else {
				details = "-"
			}
			// Apply highlighting to name
			if i.SearchQuery != "" {
				name = shared.HighlightSearchMatches(name, i.SearchQuery)
			}
		case shared.HighlightedTopicItem:
			name = i.Name
			resourceType = "topic"
			partitions = fmt.Sprintf("%d", i.Topic.NumPartitions)
			replication = fmt.Sprintf("%d", i.Topic.ReplicationFactor)
			details = fmt.Sprintf("%d configs", len(i.Topic.ConfigEntries))
			// Apply highlighting to name
			if i.SearchQuery != "" {
				name = shared.HighlightSearchMatches(name, i.SearchQuery)
			}
		default:
			continue // Skip unknown types
		}

		// Apply search highlighting if searchQuery is provided and not already highlighted
		if searchQuery != "" {
			switch item.(type) {
			case shared.ResourceListItem, TopicItem:
				name = shared.HighlightSearchMatches(name, searchQuery)
			}
		}

		rows = append(rows, table.Row{name, resourceType, partitions, replication, details})
	}

	return rows
}
