package ui

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// HandleKeyMsg processes key events for the MainPageModel
func (m *MainPageModel) HandleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Log key event details
	shared.DebugLog("Key Event - Type: %v, String: %s, SearchMode: %v", msg.Type, msg.String(), m.searchMode)

	switch {
	case key.Matches(msg, keys.Back):
		shared.DebugLog("Back Key Event - Type: %v, String: %s, SearchMode: %v", msg.Type, msg.String(), m.searchMode)
		// If search bar is focused, blur it and return focus to list
		if m.searchMode {
			m.searchMode = false
			m.searchBar.Blur()
			m.searchBar.SetValue("")
			// Reset the resources table to show all rows (without highlighting)
			m.resourcesTable.SetRows(m.allRows)
			// Reset filter state
			m.isFiltered = false
			m.currentFilter = ""
			m.filteredRows = []table.Row{}
			m.statusMessage = "Search cancelled"
			return m, nil
		}
	}

	// Handle general key presses
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		// Only quit if not in search mode (to allow typing 'q' in search terms)
		if !m.searchMode {
			return m, tea.Quit
		}
	case "/":
		// Focus search bar for normal search
		m.searchMode = true
		m.searchBar.EnterSearchMode()
		m.statusMessage = "Search mode: Type to filter items"
		cmds = append(cmds, m.searchBar.Focus())
		return m, tea.Batch(cmds...)
	case ":":
		// Focus search bar for resource switching
		m.searchMode = true
		m.searchBar.EnterResourceMode()
		m.statusMessage = "Resource mode: Type resource name (topics, consumer-groups, schemas, contexts)"
		cmds = append(cmds, m.searchBar.Focus())
		return m, tea.Batch(cmds...)
	case "enter":
		shared.DebugLog("Enter Key event - Type: %v, String: %s, SearchMode: %v", msg.Type, msg.String(), m.searchMode)

		// If in search mode, handle based on search type
		if m.searchMode {
			if m.searchBar.IsResourceMode() {
				// Resource mode: let search bar handle to switch resource
				var cmd tea.Cmd
				m.searchBar, cmd = m.searchBar.Update(msg)
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			} else {
				// Normal search mode: trigger search and exit search mode to focus resources list
				query := m.searchBar.Value()
				if query != "" {
					// Trigger search
					var cmd tea.Cmd
					m.searchBar, cmd = m.searchBar.Update(msg)
					cmds = append(cmds, cmd)
					return m, tea.Batch(cmds...)
				} else {
					// Empty search, just exit search mode and focus resources list
					m.searchMode = false
					m.searchBar.Blur()
					m.statusMessage = "Search cancelled"
					return m, nil
				}
			}
		}
		// If not in search mode and a row is selected, navigate to appropriate page
		if m.getSelectedResourceItem() != nil {
			// Check the current resource type to determine navigation
			if m.currentResource.GetType() == TopicResourceType {
				// Navigate to topic page for topics
				return m, func() tea.Msg {
					return pageChangeMsg(topicPage)
				}
			} else {
				// Navigate to resource detail page for other resources
				return m, func() tea.Msg {
					return pageChangeMsg(resourceDetailPage)
				}
			}
		}
	}

	// If in search mode, let the search bar handle keys
	if m.searchMode {
		debugLog("Handling key in search mode - Key: %s", msg.String())
		var cmd tea.Cmd
		m.searchBar, cmd = m.searchBar.Update(msg)
		cmds = append(cmds, cmd)
		debugLog("Search bar update complete, commands: %v", cmd != nil)
		return m, tea.Batch(cmds...)
	} else {
		// Normal table navigation
		switch msg.String() {
		case "j", "down":
			m.resourcesTable.MoveDown(1)
		case "k", "up":
			m.resourcesTable.MoveUp(1)
		case "g", "home":
			m.resourcesTable.GotoTop()
		case "G", "end":
			m.resourcesTable.GotoBottom()
		default:
			var cmd tea.Cmd
			m.resourcesTable, cmd = m.resourcesTable.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// HandleSearchTopics processes search topic messages
func (m *MainPageModel) HandleSearchTopics(msg searchTopicsMsg) (tea.Model, tea.Cmd) {
	// Handle search query
	query := string(msg)
	m.statusMessage = fmt.Sprintf("Searching for: %s", query)

	// Track filter state
	m.isFiltered = true
	m.currentFilter = query

	// Filter the resources
	filteredItems := []interface{}{}

	// Convert current table rows back to items for filtering
	// We need to work with the original data stored elsewhere
	// For now, we'll work with a simplified approach
	for _, row := range m.allRows {
		if len(row) > 0 {
			name := row[0] // First column is the name
			// Simple case-insensitive search
			if strings.Contains(strings.ToLower(name), strings.ToLower(query)) {
				// Create a simplified item for highlighting
				// This is a workaround - ideally we'd maintain the original items
				filteredItems = append(filteredItems, struct {
					Name string
					Row  table.Row
				}{
					Name: name,
					Row:  row,
				})
			}
		}
	}

	// Convert filtered items to highlighted table rows
	filteredRows := make([]table.Row, 0, len(filteredItems))
	for _, item := range filteredItems {
		if simpleItem, ok := item.(struct {
			Name string
			Row  table.Row
		}); ok {
			// Apply highlighting to the name (first column)
			highlightedName := HighlightSearchMatches(simpleItem.Name, query)
			// Create new row with highlighted name
			newRow := make(table.Row, len(simpleItem.Row))
			copy(newRow, simpleItem.Row)
			newRow[0] = highlightedName // Replace first column with highlighted version
			filteredRows = append(filteredRows, newRow)
		}
	}

	// Apply natural sorting to filtered results
	SortTableRowsNaturally(filteredRows)

	// Store filtered rows
	m.filteredRows = filteredRows
	m.resourcesTable.SetRows(filteredRows)
	m.searchBar.SetResultCount(len(filteredItems))
	// Exit search mode and focus resources list
	m.searchMode = false
	m.searchBar.Blur()

	if len(filteredRows) == 0 {
		m.statusMessage = fmt.Sprintf("No items found for: %s", query)
	} else {
		m.statusMessage = fmt.Sprintf("Showing %d of %d items (filtered by: %s)", len(filteredRows), len(m.allRows), query)
		// Focus first row in filtered table if available
		if len(filteredRows) > 0 {
			m.resourcesTable.GotoTop()
		}
	}

	return m, nil
}

// HandleClearSearch processes clear search messages
func (m *MainPageModel) HandleClearSearch(msg clearSearchMsg) (tea.Model, tea.Cmd) {
	// Clear search and reset resources table to original rows (without highlighting)
	m.resourcesTable.SetRows(m.allRows)
	m.searchBar.SetResultCount(0)
	m.searchMode = false
	m.searchBar.Blur()
	// Reset filter state
	m.isFiltered = false
	m.currentFilter = ""
	m.filteredRows = []table.Row{}
	m.statusMessage = fmt.Sprintf("Showing %d of %d resources", len(m.allRows), len(m.allRows))
	return m, nil
}

// HandleSwitchResource processes switch resource messages
func (m *MainPageModel) HandleSwitchResource(msg switchResourceMsg) (tea.Model, tea.Cmd) {
	// Switch to a different resource type
	m.switchToResource(ResourceType(msg))
	return m, nil
}

// HandleSwitchResourceByName processes switch resource by name messages
func (m *MainPageModel) HandleSwitchResourceByName(msg switchResourceByNameMsg) (tea.Model, tea.Cmd) {
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
		m.switchToResource(resourceType)
		m.searchMode = false
		m.searchBar.Blur()
		m.searchBar.SetValue("")
		m.statusMessage = fmt.Sprintf("Switched to %s", m.currentResource.GetName())
	} else {
		m.statusMessage = fmt.Sprintf("Unknown resource type: %s. Try: topics, consumer-groups, schemas, contexts", resourceName)
	}
	return m, nil
}
