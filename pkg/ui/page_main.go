package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Main page search bar style (different from global)
	mainPageSearchBarStyle = lipgloss.NewStyle().
		BorderStyle(RoundedBorder).
		BorderForeground(Info).
		Padding(0, 1).
		MarginBottom(1)
)

type MainPageModel struct {
	dataSource      api.KafkaDataSource
	resourcesTable  table.Model
	searchBar       components.SearchBarModel
	spinner         spinner.Model
	statusMessage   string
	lastUpdate      time.Time
	width           int
	height          int
	loading         bool
	searchMode      bool
	allRows         []table.Row
	allItems        []interface{} // Store original items to maintain data connectivity
	resourceManager *ResourceManager
	currentResource Resource
	err             error
	// Filter state tracking
	isFiltered    bool
	currentFilter string
	filteredRows  []table.Row
	filteredItems []interface{} // Store filtered original items

	// Reusable components
	layout      *components.Layout
	sidebar     *components.Sidebar
	footer      *components.Footer
	mainContent *components.MainContent
}

func (m MainPageModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Update layout configuration
	m.layout.UpdateConfig(components.LayoutConfig{
		Width:        m.width,
		Height:       m.height,
		SidebarWidth: 35,
		ShowSidebar:  true,
		HeaderTitle:  "Kafui - Kafka UI",
		ResourceType: strings.ToUpper(m.currentResource.GetType().String()),
	})

	// Calculate dimensions
	contentWidth, contentHeight, _ := m.layout.CalculateDimensions()

	// Update main content
	m.mainContent.SetDimensions(contentWidth, contentHeight)
	m.mainContent.SetSearchBar(m.searchBar)
	m.mainContent.SetTable(m.resourcesTable)
	m.mainContent.SetShowSearch(true)

	// Update sidebar
	m.sidebar.UpdateConfig(components.SidebarConfig{
		Context:         m.dataSource.GetContext(),
		CurrentResource: components.ResourceType(m.currentResource.GetType()),
		ShowResources:   true,
		ShowShortcuts:   true,
	})

	// Update footer
	selectedItem := "None"
	if selectedRow := m.getSelectedResourceItem(); selectedRow != nil {
		if rowStruct, ok := selectedRow.(struct{ ID string }); ok {
			selectedItem = rowStruct.ID
		}
	}

	m.footer.UpdateConfig(components.FooterConfig{
		Width:         m.width,
		SearchMode:    m.searchMode,
		SelectedItem:  selectedItem,
		TotalItems:    len(m.allRows),
		StatusMessage: m.statusMessage,
		LastUpdate:    m.lastUpdate,
		Spinner:       m.spinner,
	})

	// Render complete layout
	return m.layout.RenderComplete(
		m.mainContent.Render(),
		m.sidebar.Render(),
		m.footer.Render(),
	)
}

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
		case resourceListItem:
			name = i.resourceItem.GetID()
			// Determine resource type based on the concrete type
			switch i.resourceItem.(type) {
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
			itemDetails := i.resourceItem.GetDetails()
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
		case topicItem:
			name = i.name
			resourceType = "topic"
			partitions = fmt.Sprintf("%d", i.topic.NumPartitions)
			replication = fmt.Sprintf("%d", i.topic.ReplicationFactor)
			details = fmt.Sprintf("%d configs", len(i.topic.ConfigEntries))
		case HighlightedResourceListItem:
			name = i.resourceItem.GetID()
			// Determine resource type based on the concrete type
			switch i.resourceItem.(type) {
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
			itemDetails := i.resourceItem.GetDetails()
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
			if i.searchQuery != "" {
				name = HighlightSearchMatches(name, i.searchQuery)
			}
		case HighlightedTopicItem:
			name = i.name
			resourceType = "topic"
			partitions = fmt.Sprintf("%d", i.topic.NumPartitions)
			replication = fmt.Sprintf("%d", i.topic.ReplicationFactor)
			details = fmt.Sprintf("%d configs", len(i.topic.ConfigEntries))
			// Apply highlighting to name
			if i.searchQuery != "" {
				name = HighlightSearchMatches(name, i.searchQuery)
			}
		default:
			continue // Skip unknown types
		}

		// Apply search highlighting if searchQuery is provided and not already highlighted
		if searchQuery != "" {
			switch item.(type) {
			case resourceListItem, topicItem:
				name = HighlightSearchMatches(name, searchQuery)
			}
		}

		rows = append(rows, table.Row{name, resourceType, partitions, replication, details})
	}

	return rows
}

// Get currently selected resource item from table
func (m *MainPageModel) getSelectedResourceItem() interface{} {
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

func NewMainPage(ds api.KafkaDataSource) MainPageModel {
	// Initialize resources table
	columns := createResourceTableColumns()
	resourcesTable := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	resourcesTable.SetStyles(createResourceTableStyles())
	// Table doesn't need filtering configuration - we handle it ourselves
	// Remove old list-specific configurations (no longer needed for table)
	// Table styling is handled by createResourceTableStyles()

	// Initialize resource manager
	resourceManager := NewResourceManager(ds)
	currentResource := resourceManager.GetResource(TopicResourceType)

	// Initialize search bar
	searchBar := components.NewSearchBar(
		components.WithPlaceholder("Press / to search, : to switch resource..."),
		components.WithOnSearch(func(query string) tea.Msg {
			return searchTopicsMsg(query)
		}),
		components.WithOnClear(func() tea.Msg {
			return clearSearchMsg{}
		}),
		components.WithOnResourceSwitch(func(resource string) tea.Msg {
			return switchResourceByNameMsg(resource)
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

	return MainPageModel{
		dataSource:      ds,
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
}

func (m *MainPageModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadTopics(),
		m.spinner.Tick,
		m.updateTimer(),
	)
}

func (m *MainPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate available space for content
		sidebarWidth := 35
		mainContentWidth := msg.Width - sidebarWidth - 6 // Account for margins and borders
		contentHeight := msg.Height - 8                  // Account for header, footer and margins (removed status bar)

		// Update table and search bar dimensions
		m.resourcesTable.SetHeight(contentHeight - 3) // Account for borders and margins
		m.searchBar.SetWidth(mainContentWidth - 4)
		return m, nil

	case tea.KeyMsg:
		return m.HandleKeyMsg(msg)

	case topicListMsg:
		m.loading = false
		// topicListMsg is already a slice of interfaces, no need to convert
		items := make([]interface{}, len(msg))
		for i, item := range msg {
			items[i] = item
		}

		// Only update if we're currently showing topics (not if we've switched to another resource)
		if m.currentResource.GetType() != TopicResourceType {
			// Ignore topic updates when viewing other resources
			return m, tea.Batch(
				m.spinner.Tick,
				m.updateTimer(),
			)
		}

		// Store original items for navigation
		m.allItems = items

		// Convert items to table rows
		rows := convertItemsToRows(items, "")
		m.allRows = rows

		// Update search suggestions with topic names
		searchSuggestions := make([]string, 0, len(items))
		for _, item := range items {
			if topicItem, ok := item.(topicItem); ok {
				searchSuggestions = append(searchSuggestions, topicItem.name)
			}
		}
		m.searchBar.SetSearchSuggestions(searchSuggestions)

		// If we're currently filtered, reapply the filter to new data
		if m.isFiltered && m.currentFilter != "" {
			// Reapply current filter
			filteredItems := []interface{}{}
			for _, item := range items {
				if topicItem, ok := item.(topicItem); ok {
					if strings.Contains(strings.ToLower(topicItem.name), strings.ToLower(m.currentFilter)) {
						// Create highlighted version using original topicItem
						highlightedItem := CreateHighlightedItem(topicItem, m.currentFilter)
						filteredItems = append(filteredItems, highlightedItem)
					}
				}
			}
			// Store filtered items for navigation
			m.filteredItems = filteredItems
			// Convert filtered items to table rows and apply natural sorting
			filteredRows := convertItemsToRows(filteredItems, m.currentFilter)
			SortTableRowsNaturally(filteredRows)
			m.filteredRows = filteredRows
			m.resourcesTable.SetRows(filteredRows)
			m.statusMessage = fmt.Sprintf("Showing %d of %d topics (filtered by: %s)", len(filteredRows), len(rows), m.currentFilter)
		} else {
			// No filter active, show all rows
			SortTableRowsNaturally(rows)
			m.resourcesTable.SetRows(rows)
			m.statusMessage = fmt.Sprintf("Showing %d of %d topics", len(rows), len(rows))
		}

		return m, tea.Batch(
			m.spinner.Tick,
			m.updateTimer(),
		)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case timerTickMsg:
		m.lastUpdate = time.Now()
		cmds = append(cmds, m.updateTimer())
		if !m.loading {
			m.loading = true
			// Load data for current resource type instead of always loading topics
			if m.currentResource.GetType() == TopicResourceType {
				cmds = append(cmds, m.loadTopics())
			} else {
				// For other resource types, refresh the current resource
				cmds = append(cmds, m.loadCurrentResource())
			}
		}

	case errorMsg:
		m.loading = false
		m.err = msg
		m.statusMessage = fmt.Sprintf("Error: %v", msg)

	case searchTopicsMsg:
		return m.HandleSearchTopics(msg)

	case clearSearchMsg:
		return m.HandleClearSearch(msg)

	case switchResourceMsg:
		return m.HandleSwitchResource(msg)

	case switchResourceByNameMsg:
		return m.HandleSwitchResourceByName(msg)

	case currentResourceListMsg:
		m.loading = false
		// Only update if the message is for the current resource type
		if msg.resourceType == m.currentResource.GetType() {
			// Convert list items to interface slice
			items := make([]interface{}, len(msg.items))
			for i, item := range msg.items {
				items[i] = item
			}

			// Store original items for navigation
			m.allItems = items

			// Convert to table rows
			rows := convertItemsToRows(items, "")
			m.allRows = rows

			// Update search suggestions
			searchSuggestions := make([]string, 0, len(items))
			for _, item := range items {
				if resourceItem, ok := item.(resourceListItem); ok {
					searchSuggestions = append(searchSuggestions, resourceItem.resourceItem.GetID())
				}
			}
			m.searchBar.SetSearchSuggestions(searchSuggestions)

			// If we're currently filtered, reapply the filter to new data
			if m.isFiltered && m.currentFilter != "" {
				// Reapply current filter
				filteredItems := []interface{}{}
				for _, item := range items {
					if resourceItem, ok := item.(resourceListItem); ok {
						if strings.Contains(strings.ToLower(resourceItem.resourceItem.GetID()), strings.ToLower(m.currentFilter)) {
							// Create highlighted version using original resourceListItem
							highlightedItem := CreateHighlightedItem(resourceItem, m.currentFilter)
							filteredItems = append(filteredItems, highlightedItem)
						}
					}
				}
				// Store filtered items for navigation
				m.filteredItems = filteredItems
				// Convert filtered items to table rows and apply natural sorting
				filteredRows := convertItemsToRows(filteredItems, m.currentFilter)
				SortTableRowsNaturally(filteredRows)
				m.filteredRows = filteredRows
				m.resourcesTable.SetRows(filteredRows)
				m.statusMessage = fmt.Sprintf("Showing %d of %d %s (filtered by: %s)", len(filteredRows), len(rows), m.currentResource.GetName(), m.currentFilter)
			} else {
				// No filter active, show all rows
				SortTableRowsNaturally(rows)
				m.resourcesTable.SetRows(rows)
				m.statusMessage = fmt.Sprintf("Showing %d of %d %s", len(rows), len(rows), m.currentResource.GetName())
			}
		}
		return m, tea.Batch(
			m.spinner.Tick,
			m.updateTimer(),
		)
	}

	return m, tea.Batch(cmds...)
}

// switchToResource switches the current view to a different resource type
func (m *MainPageModel) switchToResource(resourceType ResourceType) {
	m.currentResource = m.resourceManager.GetResource(resourceType)
	// Note: Tables don't have titles like lists - title is handled by layout

	// Reset filter state when switching resources
	m.isFiltered = false
	m.currentFilter = ""
	m.filteredRows = []table.Row{}

	// Load data for the new resource
	items, err := m.currentResource.GetData()
	if err != nil {
		m.statusMessage = fmt.Sprintf("Error loading %s: %v", m.currentResource.GetName(), err)
		return
	}

	// Convert resource items to interface slice
	interfaceItems := make([]interface{}, 0, len(items))
	searchSuggestions := make([]string, 0, len(items))

	for _, item := range items {
		interfaceItems = append(interfaceItems, resourceListItem{
			resourceItem: item,
		})
		// Add item ID to search suggestions
		searchSuggestions = append(searchSuggestions, item.GetID())
	}

	// Convert to table rows and sort by ID (name) using natural sorting
	rows := convertItemsToRows(interfaceItems, "")
	SortTableRowsNaturally(rows)

	// Sort suggestions using natural sorting as well
	sort.Sort(NaturalSort(searchSuggestions))

	m.resourcesTable.SetRows(rows)
	m.allRows = rows

	// Update search suggestions for the new resource
	m.searchBar.SetSearchSuggestions(searchSuggestions)

	m.statusMessage = fmt.Sprintf("Showing %d of %d %s", len(rows), len(rows), m.currentResource.GetName())
}

// resourceListItem wraps a ResourceItem to implement list.Item interface
type resourceListItem struct {
	resourceItem ResourceItem
}

func (r resourceListItem) FilterValue() string {
	return r.resourceItem.GetID()
}

// Custom message types
type searchTopicsMsg string
type clearSearchMsg struct{}
type switchResourceMsg ResourceType
type switchResourceByNameMsg string
type currentResourceListMsg struct {
	resourceType ResourceType
	items        []interface{} // Changed from []list.Item to []interface{}
}

func (m MainPageModel) updateTimer() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return timerTickMsg(t)
	})
}

func (m *MainPageModel) loadTopics() tea.Cmd {
	return func() tea.Msg {
		topics, err := m.dataSource.GetTopics()
		if err != nil {
			return errorMsg(err)
		}

		// Create a slice of topic names for sorting
		topicNames := make([]string, 0, len(topics))
		for name := range topics {
			topicNames = append(topicNames, name)
		}

		// Sort topic names using natural sorting
		sort.Sort(NaturalSort(topicNames))

		items := make([]interface{}, 0, len(topics)) // Changed to []interface{}
		searchSuggestions := make([]string, 0, len(topics))

		for _, name := range topicNames {
			topic := topics[name]
			items = append(items, topicItem{
				name:  name,
				topic: topic,
			})
			// Add topic name to search suggestions
			searchSuggestions = append(searchSuggestions, name)
		}

		return topicListMsg(items)
	}
}

func (m *MainPageModel) loadCurrentResource() tea.Cmd {
	return func() tea.Msg {
		// Load data for the current resource
		items, err := m.currentResource.GetData()
		if err != nil {
			return errorMsg(err)
		}

		// Convert resource items to interface slice
		interfaceItems := make([]interface{}, 0, len(items))
		for _, item := range items {
			interfaceItems = append(interfaceItems, resourceListItem{
				resourceItem: item,
			})
		}

		return currentResourceListMsg{
			resourceType: m.currentResource.GetType(),
			items:        interfaceItems,
		}
	}
}
