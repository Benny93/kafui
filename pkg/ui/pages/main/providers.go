package mainpage

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Helper functions for table configuration

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

// KafuiContentProvider provides the main content for Kafui (resource table and search)
// Implements providers.ContentProvider interface
type KafuiContentProvider struct {
	dataSource      api.KafkaDataSource
	resourceManager *ResourceManager
	currentResource Resource

	// Table and search state
	resourcesTable table.Model
	searchMode     bool
	loading        bool
	error          error

	// Data storage
	allRows       []table.Row
	allItems      []interface{}
	filteredRows  []table.Row
	filteredItems []interface{}

	// Filter state
	isFiltered    bool
	currentFilter string
}

func NewKafuiContentProvider(dataSource api.KafkaDataSource) *KafuiContentProvider {
	// Initialize resources table
	columns := createResourceTableColumns()
	resourcesTable := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(2),
	)
	resourcesTable.SetStyles(createResourceTableStyles())

	// Initialize resource manager
	resourceManager := NewResourceManager(dataSource)
	currentResource := resourceManager.GetResource(TopicResourceType)

	return &KafuiContentProvider{
		dataSource:      dataSource,
		resourceManager: resourceManager,
		currentResource: currentResource,
		resourcesTable:  resourcesTable,
		allRows:         []table.Row{},
		allItems:        []interface{}{},
		filteredRows:    []table.Row{},
		filteredItems:   []interface{}{},
	}
}

func (k *KafuiContentProvider) RenderContent(width, height int) string {
	// Calculate proper table height - leave space for search bar and padding
	tableHeight := height - 6 // More conservative height calculation
	if k.searchMode {
		tableHeight -= 3 // Additional space for search bar
	}
	if tableHeight < 5 {
		tableHeight = 5 // Minimum table height
	}
	
	// Update table dimensions
	k.resourcesTable.SetHeight(tableHeight)
	k.resourcesTable.SetWidth(width - 4) // Account for content padding

	if k.error != nil {
		return k.renderError()
	}

	if k.loading && len(k.allRows) == 0 {
		return k.renderLoading()
	}

	if len(k.allRows) == 0 && !k.loading {
		return k.renderEmpty()
	}

	var content strings.Builder
	
	// Add search bar if in search mode
	if k.searchMode {
		searchBar := k.renderSearchBar(width)
		content.WriteString(searchBar)
		content.WriteString("\n\n")
	}
	
	// Render the main table
	content.WriteString(k.resourcesTable.View())

	return content.String()
}

// renderSearchBar renders the search input bar
func (k *KafuiContentProvider) renderSearchBar(width int) string {
	searchStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	
	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
	
	// Create search prompt
	prompt := searchStyle.Render("🔍 Search: ")
	filter := k.currentFilter
	if filter == "" {
		filter = promptStyle.Render("(type to filter resources...)")
	}
	
	// Add cursor if in search mode
	cursor := ""
	if k.searchMode {
		cursor = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Render("█")
	}
	
	searchLine := prompt + filter + cursor
	
	// Add help text
	helpText := promptStyle.Render("ESC to cancel • Enter to search")
	
	return searchLine + "\n" + helpText
}

func (k *KafuiContentProvider) renderError() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Padding(1)
	return style.Render(fmt.Sprintf("Error: %v", k.error))
}

func (k *KafuiContentProvider) renderLoading() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Padding(1)
	return style.Render("Loading resources...")
}

func (k *KafuiContentProvider) renderEmpty() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Padding(1)
	return style.Render("No resources found. Try refreshing or checking your connection.")
}

func (k *KafuiContentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle search mode first
		if k.searchMode {
			switch msg.String() {
			case "escape":
				k.searchMode = false
				k.clearSearch()
				return nil
			case "enter":
				// Confirm search and exit search mode
				k.searchMode = false
				return nil
			case "backspace":
				if len(k.currentFilter) > 0 {
					k.currentFilter = k.currentFilter[:len(k.currentFilter)-1]
					k.handleSearch(k.currentFilter)
				}
				return nil
			default:
				// Handle typing in search mode
				if len(msg.Runes) > 0 {
					k.currentFilter += string(msg.Runes)
					k.handleSearch(k.currentFilter)
				}
				return nil
			}
		}

		// Handle normal mode keys
		switch msg.String() {
		case "/":
			k.searchMode = true
			k.currentFilter = "" // Reset filter when starting search
			return nil
		case ":":
			// Resource switching mode - this should be handled differently
			// Return a command to trigger resource switching UI
			return func() tea.Msg {
				return StartResourceSwitchingMsg{}
			}
		case "enter":
			// Handle resource selection
			shared.DebugLog("KafuiContentProvider: Enter key pressed, calling handleResourceSelection()")
			return k.handleResourceSelection()
		}

		// Handle table navigation when not in search mode
		var cmd tea.Cmd
		k.resourcesTable, cmd = k.resourcesTable.Update(msg)
		cmds = append(cmds, cmd)

	case SearchTopicsMsg:
		k.handleSearch(string(msg))

	case ClearSearchMsg:
		k.clearSearch()

	case SwitchResourceMsg:
		k.switchResource(msg)
		cmds = append(cmds, k.loadCurrentResource())

	case SwitchResourceByNameMsg:
		if resourceType := k.parseResourceType(string(msg)); resourceType != -1 {
			k.switchResource(SwitchResourceMsg(resourceType))
			cmds = append(cmds, k.loadCurrentResource())
		}

	case CurrentResourceListMsg:
		k.handleResourceList(msg)

	case TopicListMsg:
		k.handleTopicList(msg)

	case ErrorMsg:
		k.error = error(msg)
		k.loading = false

	case TimerTickMsg:
		// Auto-refresh data
		cmds = append(cmds, k.loadCurrentResource())
		
	case StartResourceSwitchingMsg:
		// Handle resource switching - for now, cycle through resources
		// In a full implementation, this could show a selection UI
		return k.handleResourceSwitching()
	}

	return tea.Batch(cmds...)
}

func (k *KafuiContentProvider) InitContent() tea.Cmd {
	return k.loadCurrentResource()
}

// Helper methods

func (k *KafuiContentProvider) handleSearch(query string) {
	if query == "" {
		k.clearSearch()
		return
	}

	k.currentFilter = query
	k.isFiltered = true

	// Filter items based on search query
	filteredItems := make([]interface{}, 0)
	for _, item := range k.allItems {
		if k.itemMatchesQuery(item, query) {
			// Create highlighted version
			highlightedItem := k.createHighlightedItem(item, query)
			filteredItems = append(filteredItems, highlightedItem)
		}
	}

	k.filteredItems = filteredItems
	k.filteredRows = convertItemsToRows(filteredItems, query)
	k.resourcesTable.SetRows(k.filteredRows)
}

func (k *KafuiContentProvider) clearSearch() {
	k.isFiltered = false
	k.currentFilter = ""
	k.filteredItems = []interface{}{}
	k.filteredRows = []table.Row{}
	k.resourcesTable.SetRows(k.allRows)
}

func (k *KafuiContentProvider) switchResource(msg SwitchResourceMsg) {
	k.currentResource = k.resourceManager.GetResource(ResourceType(msg))
	k.clearSearch() // Clear any active search when switching resources
}

func (k *KafuiContentProvider) loadCurrentResource() tea.Cmd {
	k.loading = true
	k.error = nil

	return func() tea.Msg {
		items, err := k.currentResource.GetData()
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
			ResourceType: k.currentResource.GetType(),
			Items:        interfaceItems,
		}
	}
}

func (k *KafuiContentProvider) handleResourceList(msg CurrentResourceListMsg) {
	k.loading = false
	
	// Sort items naturally by name
	sortedItems := make([]interface{}, len(msg.Items))
	copy(sortedItems, msg.Items)
	sort.Slice(sortedItems, func(i, j int) bool {
		nameI := k.getItemName(sortedItems[i])
		nameJ := k.getItemName(sortedItems[j])
		return strings.ToLower(nameI) < strings.ToLower(nameJ)
	})
	
	k.allItems = sortedItems
	k.allRows = convertItemsToRows(sortedItems, "")

	if k.isFiltered {
		k.handleSearch(k.currentFilter)
	} else {
		k.resourcesTable.SetRows(k.allRows)
	}
}

func (k *KafuiContentProvider) handleTopicList(msg TopicListMsg) {
	k.loading = false

	// Convert TopicItems to interface slice and sort
	interfaceItems := make([]interface{}, 0, len(msg))
	for _, item := range msg {
		interfaceItems = append(interfaceItems, item)
	}
	
	// Sort items naturally by name
	sort.Slice(interfaceItems, func(i, j int) bool {
		nameI := k.getItemName(interfaceItems[i])
		nameJ := k.getItemName(interfaceItems[j])
		return strings.ToLower(nameI) < strings.ToLower(nameJ)
	})

	k.allItems = interfaceItems
	k.allRows = convertItemsToRows(interfaceItems, "")

	if k.isFiltered {
		k.handleSearch(k.currentFilter)
	} else {
		k.resourcesTable.SetRows(k.allRows)
	}
}

func (k *KafuiContentProvider) handleResourceSelection() tea.Cmd {
	selectedItem := k.GetSelectedResourceItem()
	shared.DebugLog("KafuiContentProvider.handleResourceSelection: selectedItem = %v", selectedItem)
	if selectedItem == nil {
		shared.DebugLog("KafuiContentProvider.handleResourceSelection: selectedItem is nil, returning nil")
		return nil
	}

	resourceType := k.currentResource.GetType()
	resourceID := k.getItemID(selectedItem)
	shared.DebugLog("KafuiContentProvider.handleResourceSelection: Creating NavigateToResourceDetailMsg with ResourceType=%v, ResourceID=%s", resourceType, resourceID)

	// Navigate to resource detail page
	// This would typically send a navigation message
	return func() tea.Msg {
		msg := NavigateToResourceDetailMsg{
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Item:         selectedItem,
		}
		shared.DebugLog("KafuiContentProvider.handleResourceSelection: Returning NavigateToResourceDetailMsg: %+v", msg)
		return msg
	}
}

func (k *KafuiContentProvider) GetSelectedResourceItem() interface{} {
	selectedRow := k.resourcesTable.Cursor()

	// Use filtered items if we're currently in a filtered state
	if k.isFiltered && len(k.filteredItems) > 0 {
		if selectedRow < 0 || selectedRow >= len(k.filteredItems) {
			return nil
		}
		return k.filteredItems[selectedRow]
	}

	// Otherwise use all items
	if selectedRow < 0 || selectedRow >= len(k.allItems) {
		return nil
	}
	return k.allItems[selectedRow]
}

func (k *KafuiContentProvider) itemMatchesQuery(item interface{}, query string) bool {
	queryLower := strings.ToLower(query)

	switch i := item.(type) {
	case shared.ResourceListItem:
		return strings.Contains(strings.ToLower(i.ResourceItem.GetID()), queryLower)
	case TopicItem:
		return strings.Contains(strings.ToLower(i.name), queryLower)
	default:
		return false
	}
}

func (k *KafuiContentProvider) createHighlightedItem(item interface{}, query string) interface{} {
	switch i := item.(type) {
	case shared.ResourceListItem:
		return shared.HighlightedResourceListItem{
			ResourceItem: i.ResourceItem,
			SearchQuery:  query,
		}
	case TopicItem:
		return shared.HighlightedTopicItem{
			Name:        i.name,
			Topic:       i.topic,
			SearchQuery: query,
		}
	default:
		return item
	}
}

func (k *KafuiContentProvider) parseResourceType(name string) ResourceType {
	switch strings.ToLower(name) {
	case "topics", "topic":
		return TopicResourceType
	case "consumer-groups", "consumer-group", "groups", "group":
		return ConsumerGroupResourceType
	case "schemas", "schema":
		return SchemaResourceType
	case "contexts", "context":
		return ContextResourceType
	default:
		return -1
	}
}

func (k *KafuiContentProvider) getItemID(item interface{}) string {
	switch i := item.(type) {
	case shared.ResourceListItem:
		return i.ResourceItem.GetID()
	case shared.HighlightedResourceListItem:
		return i.ResourceItem.GetID()
	case TopicItem:
		return i.name
	case shared.HighlightedTopicItem:
		return i.Name
	default:
		return "unknown"
	}
}

func (k *KafuiContentProvider) getItemName(item interface{}) string {
	switch i := item.(type) {
	case shared.ResourceListItem:
		return i.ResourceItem.GetID()
	case shared.HighlightedResourceListItem:
		return i.ResourceItem.GetID()
	case TopicItem:
		return i.name
	case shared.HighlightedTopicItem:
		return i.Name
	default:
		return "unknown"
	}
}

func (k *KafuiContentProvider) handleResourceSwitching() tea.Cmd {
	// Get current resource type
	currentType := k.currentResource.GetType()
	
	// Define the order of resource types
	resourceOrder := []ResourceType{
		TopicResourceType,
		ConsumerGroupResourceType,
		SchemaResourceType,
		ContextResourceType,
	}
	
	// Find current index and switch to next
	var nextType ResourceType = TopicResourceType // Default fallback
	for i, resType := range resourceOrder {
		if resType == currentType {
			// Get next resource type (cycle back to beginning if at end)
			nextIndex := (i + 1) % len(resourceOrder)
			nextType = resourceOrder[nextIndex]
			break
		}
	}
	
	// Switch to the next resource type
	k.switchResource(SwitchResourceMsg(nextType))
	return k.loadCurrentResource()
}

// Navigation message for resource selection
type NavigateToResourceDetailMsg struct {
	ResourceType ResourceType
	ResourceID   string
	Item         interface{}
}

// Message to start resource switching mode
type StartResourceSwitchingMsg struct{}

// KafuiHeaderDataProvider provides header data for Kafui
// Implements providers.HeaderDataProvider interface
type KafuiHeaderDataProvider struct {
	dataSource api.KafkaDataSource
	lastUpdate time.Time
}

func NewKafuiHeaderDataProvider(dataSource api.KafkaDataSource) *KafuiHeaderDataProvider {
	return &KafuiHeaderDataProvider{
		dataSource: dataSource,
		lastUpdate: time.Now(),
	}
}

func (k *KafuiHeaderDataProvider) GetBrandName() string {
	return "Kafui™"
}

func (k *KafuiHeaderDataProvider) GetAppName() string {
	return "Kafka TUI"
}

func (k *KafuiHeaderDataProvider) GetStatusData() map[string]interface{} {
	context := k.dataSource.GetContext()
	return map[string]interface{}{
		"time":    k.lastUpdate.Format("15:04:05"),
		"status":  "connected",
		"context": context,
		"cluster": "kafka-cluster",
	}
}

func (k *KafuiHeaderDataProvider) HandleHeaderUpdate(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case TimerTickMsg:
		k.lastUpdate = time.Time(msg)
		return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
			return TimerTickMsg(t)
		})
	}
	return nil
}

func (k *KafuiHeaderDataProvider) InitHeader() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return TimerTickMsg(t)
	})
}
