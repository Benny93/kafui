package mainpage

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// Handlers manages event handling for the main page
type Handlers struct {
	model *Model
}

// NewHandlers creates a new Handlers instance
func NewHandlers(model *Model) *Handlers {
	return &Handlers{
		model: model,
	}
}

// Handle routes messages to appropriate handlers
func (h *Handlers) Handle(model *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	h.model = model // Update model reference
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return h.handleWindowSize(model, msg)

	case tea.KeyMsg:
		return h.handleKeyMsg(model, msg)

	case TopicListMsg:
		return h.handleTopicList(model, msg)

	case spinner.TickMsg:
		return h.handleSpinnerTick(model, msg)

	case TimerTickMsg:
		return h.handleTimerTick(model, msg)

	case ErrorMsg:
		return h.handleError(model, msg)

	case SearchTopicsMsg:
		return h.handleSearchTopics(model, msg)

	case ClearSearchMsg:
		return h.handleClearSearch(model, msg)

	case SwitchResourceMsg:
		return h.handleSwitchResource(model, msg)

	case SwitchResourceByNameMsg:
		return h.handleSwitchResourceByName(model, msg)

	case CurrentResourceListMsg:
		return h.handleCurrentResourceList(model, msg)

	default:
		// Handle any unrecognized messages
		return model, tea.Batch(cmds...)
	}
}

func (h *Handlers) handleWindowSize(model *Model, msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	model.SetDimensions(msg.Width, msg.Height)
	return model, nil
}

func (h *Handlers) handleKeyMsg(model *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Delegate to the keys handler
	cmd := model.keys.HandleKey(model, msg)
	return model, cmd
}

func (h *Handlers) handleTopicList(model *Model, msg TopicListMsg) (tea.Model, tea.Cmd) {
	model.loading = false
	// Convert TopicListMsg to interface slice
	items := make([]interface{}, len(msg))
	for i, item := range msg {
		items[i] = item
	}

	// Only update if we're currently showing topics (not if we've switched to another resource)
	if model.currentResource.GetType() != TopicResourceType {
		// Ignore topic updates when viewing other resources
		return model, tea.Batch(
			model.spinner.Tick,
			model.updateTimer(),
		)
	}

	// Store original items for navigation
	model.allItems = items

	// Convert items to table rows
	rows := convertItemsToRows(items, "")
	model.allRows = rows

	// Update search suggestions with topic names
	searchSuggestions := make([]string, 0, len(items))
	for _, item := range items {
		if topicItem, ok := item.(TopicItem); ok {
			searchSuggestions = append(searchSuggestions, topicItem.name)
		}
	}
	model.searchBar.SetSearchSuggestions(searchSuggestions)

	// If we're currently filtered, reapply the filter to new data
	if model.isFiltered && model.currentFilter != "" {
		h.reapplyFilter(model, items)
	} else {
		// No filter active, show all rows
		shared.SortTableRowsNaturally(rows)
		model.resourcesTable.SetRows(rows)
		model.statusMessage = fmt.Sprintf("Showing %d of %d topics", len(rows), len(rows))
	}

	return model, tea.Batch(
		model.spinner.Tick,
		model.updateTimer(),
	)
}

func (h *Handlers) handleSpinnerTick(model *Model, msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	model.spinner, cmd = model.spinner.Update(msg)
	return model, cmd
}

func (h *Handlers) handleTimerTick(model *Model, msg TimerTickMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	model.lastUpdate = time.Time(msg)
	cmds = append(cmds, model.updateTimer())

	if !model.loading {
		model.loading = true
		// Load data for current resource type instead of always loading topics
		if model.currentResource.GetType() == TopicResourceType {
			cmds = append(cmds, model.loadTopics())
		} else {
			// For other resource types, refresh the current resource
			cmds = append(cmds, model.loadCurrentResource())
		}
	}

	return model, tea.Batch(cmds...)
}

func (h *Handlers) handleError(model *Model, msg ErrorMsg) (tea.Model, tea.Cmd) {
	model.loading = false
	model.error = error(msg)
	model.statusMessage = fmt.Sprintf("Error: %v", msg)
	return model, nil
}

func (h *Handlers) handleSearchTopics(model *Model, msg SearchTopicsMsg) (tea.Model, tea.Cmd) {
	// Handle search query
	query := string(msg)
	model.statusMessage = fmt.Sprintf("Searching for: %s", query)

	// Track filter state
	model.isFiltered = true
	model.currentFilter = query

	// Filter the resources using original items
	filteredItems := []interface{}{}

	// Work with the original items instead of table rows
	for _, item := range model.allItems {
		switch i := item.(type) {
		case TopicItem:
			if strings.Contains(strings.ToLower(i.name), strings.ToLower(query)) {
				// Create highlighted version for display
				highlightedItem := shared.CreateHighlightedItem(i, query)
				filteredItems = append(filteredItems, highlightedItem)
			}
		case shared.ResourceListItem:
			if strings.Contains(strings.ToLower(i.ResourceItem.GetID()), strings.ToLower(query)) {
				// Create highlighted version for display
				highlightedItem := shared.CreateHighlightedItem(i, query)
				filteredItems = append(filteredItems, highlightedItem)
			}
		}
	}

	// Store filtered items for navigation
	model.filteredItems = filteredItems

	// Convert filtered items to table rows
	filteredRows := convertItemsToRows(filteredItems, query)

	// Apply natural sorting to filtered results
	shared.SortTableRowsNaturally(filteredRows)

	// Store filtered rows
	model.filteredRows = filteredRows
	model.resourcesTable.SetRows(filteredRows)
	model.searchBar.SetResultCount(len(filteredItems))

	// Exit search mode and focus resources list
	model.searchMode = false
	model.searchBar.Blur()

	if len(filteredRows) == 0 {
		model.statusMessage = fmt.Sprintf("No items found for: %s", query)
	} else {
		model.statusMessage = fmt.Sprintf("Showing %d of %d items (filtered by: %s)", len(filteredRows), len(model.allRows), query)
		// Focus first row in filtered table if available
		if len(filteredRows) > 0 {
			model.resourcesTable.GotoTop()
		}
	}

	return model, nil
}

func (h *Handlers) handleClearSearch(model *Model, msg ClearSearchMsg) (tea.Model, tea.Cmd) {
	// Clear search and reset resources table to original rows (without highlighting)
	model.resourcesTable.SetRows(model.allRows)
	model.searchBar.SetResultCount(0)
	model.searchMode = false
	model.searchBar.Blur()
	// Reset filter state
	model.isFiltered = false
	model.currentFilter = ""
	model.filteredRows = []table.Row{}
	model.filteredItems = []interface{}{} // Clear filtered items as well
	model.statusMessage = fmt.Sprintf("Showing %d of %d resources", len(model.allRows), len(model.allRows))
	return model, nil
}

func (h *Handlers) handleSwitchResource(model *Model, msg SwitchResourceMsg) (tea.Model, tea.Cmd) {
	// Switch to a different resource type
	h.switchToResource(model, ResourceType(msg))
	return model, nil
}

func (h *Handlers) handleSwitchResourceByName(model *Model, msg SwitchResourceByNameMsg) (tea.Model, tea.Cmd) {
	// Switch to a resource type by name
	resourceName := strings.ToLower(string(msg))
	var resourceType ResourceType
	var found bool

	switch resourceName {
	case "topics", "topic":
		resourceType = TopicResourceType
		found = true
	case "consumer-groups", "consumers", "consumer", "groups", "cg":
		resourceType = ConsumerGroupResourceType
		found = true
	case "schemas", "schema":
		resourceType = SchemaResourceType
		found = true
	case "contexts", "context", "ctx":
		resourceType = ContextResourceType
		found = true
	}

	if found {
		h.switchToResource(model, resourceType)
		model.searchMode = false
		model.searchBar.Blur()
		model.searchBar.SetValue("")
		model.statusMessage = fmt.Sprintf("Switched to %s", model.currentResource.GetName())
	} else {
		model.statusMessage = fmt.Sprintf("Unknown resource type: %s. Try: topics, consumer-groups, schemas, contexts", resourceName)
	}
	return model, nil
}

func (h *Handlers) handleCurrentResourceList(model *Model, msg CurrentResourceListMsg) (tea.Model, tea.Cmd) {
	model.loading = false
	// Only update if the message is for the current resource type
	if msg.ResourceType == model.currentResource.GetType() {
		// Convert list items to interface slice
		items := make([]interface{}, len(msg.Items))
		for i, item := range msg.Items {
			items[i] = item
		}

		// Store original items for navigation
		model.allItems = items

		// Convert to table rows
		rows := convertItemsToRows(items, "")
		model.allRows = rows

		// Update search suggestions
		searchSuggestions := make([]string, 0, len(items))
		for _, item := range items {
			if resourceItem, ok := item.(shared.ResourceListItem); ok {
				searchSuggestions = append(searchSuggestions, resourceItem.ResourceItem.GetID())
			}
		}
		model.searchBar.SetSearchSuggestions(searchSuggestions)

		// If we're currently filtered, reapply the filter to new data
		if model.isFiltered && model.currentFilter != "" {
			h.reapplyFilter(model, items)
		} else {
			// No filter active, show all rows
			shared.SortTableRowsNaturally(rows)
			model.resourcesTable.SetRows(rows)
			model.statusMessage = fmt.Sprintf("Showing %d of %d %s", len(rows), len(rows), model.currentResource.GetName())
		}
	}
	return model, tea.Batch(
		model.spinner.Tick,
		model.updateTimer(),
	)
}

// Helper methods

func (h *Handlers) reapplyFilter(model *Model, items []interface{}) {
	// Reapply current filter
	filteredItems := []interface{}{}
	for _, item := range items {
		switch i := item.(type) {
		case TopicItem:
			if strings.Contains(strings.ToLower(i.name), strings.ToLower(model.currentFilter)) {
				// Create highlighted version using original TopicItem
				highlightedItem := shared.CreateHighlightedItem(i, model.currentFilter)
				filteredItems = append(filteredItems, highlightedItem)
			}
		case shared.ResourceListItem:
			if strings.Contains(strings.ToLower(i.ResourceItem.GetID()), strings.ToLower(model.currentFilter)) {
				// Create highlighted version using original ResourceListItem
				highlightedItem := shared.CreateHighlightedItem(i, model.currentFilter)
				filteredItems = append(filteredItems, highlightedItem)
			}
		}
	}

	// Store filtered items for navigation
	model.filteredItems = filteredItems
	// Convert filtered items to table rows and apply natural sorting
	filteredRows := convertItemsToRows(filteredItems, model.currentFilter)
	shared.SortTableRowsNaturally(filteredRows)
	model.filteredRows = filteredRows
	model.resourcesTable.SetRows(filteredRows)

	resourceName := model.currentResource.GetName()
	if model.currentResource.GetType() == TopicResourceType {
		resourceName = "topics"
	}

	model.statusMessage = fmt.Sprintf("Showing %d of %d %s (filtered by: %s)", len(filteredRows), len(model.allRows), resourceName, model.currentFilter)
}

// switchToResource switches the current view to a different resource type
func (h *Handlers) switchToResource(model *Model, resourceType ResourceType) {
	model.currentResource = model.resourceManager.GetResource(resourceType)

	// Reset filter state when switching resources
	model.isFiltered = false
	model.currentFilter = ""
	model.filteredRows = []table.Row{}

	// Load data for the new resource
	items, err := model.currentResource.GetData()
	if err != nil {
		model.statusMessage = fmt.Sprintf("Error loading %s: %v", model.currentResource.GetName(), err)
		return
	}

	// Convert resource items to interface slice
	interfaceItems := make([]interface{}, 0, len(items))
	searchSuggestions := make([]string, 0, len(items))

	for _, item := range items {
		interfaceItems = append(interfaceItems, shared.ResourceListItem{
			ResourceItem: item,
		})
		// Add item ID to search suggestions
		searchSuggestions = append(searchSuggestions, item.GetID())
	}

	// Convert to table rows and sort by ID (name) using natural sorting
	rows := convertItemsToRows(interfaceItems, "")
	shared.SortTableRowsNaturally(rows)

	// Sort suggestions using natural sorting as well
	sort.Sort(shared.NaturalSort(searchSuggestions))

	model.resourcesTable.SetRows(rows)
	model.allRows = rows

	// Update search suggestions for the new resource
	model.searchBar.SetSearchSuggestions(searchSuggestions)

	model.statusMessage = fmt.Sprintf("Showing %d of %d %s", len(rows), len(rows), model.currentResource.GetName())
}
