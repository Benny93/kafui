package ui

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
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
			// Reset the resources list to show all items (without highlighting)
			m.resourcesList.SetItems(m.allItems)
			// Reset filter state
			m.isFiltered = false
			m.currentFilter = ""
			m.filteredItems = []list.Item{}
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
		// If not in search mode and an item is selected, navigate to topic page
		if m.resourcesList.SelectedItem() != nil {
			// Let the main UI model handle navigation to topic page
			return m, func() tea.Msg {
				return pageChangeMsg(topicPage)
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
		// Normal navigation
		switch msg.String() {
		case "j", "down":
			m.resourcesList.CursorDown()
		case "k", "up":
			m.resourcesList.CursorUp()
		case "g", "home":
			m.resourcesList.Select(0)
		case "G", "end":
			m.resourcesList.Select(len(m.resourcesList.Items()) - 1)
		default:
			var cmd tea.Cmd
			m.resourcesList, cmd = m.resourcesList.Update(msg)
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

	// Filter the resources list
	filteredItems := []list.Item{}

	for _, item := range m.allItems {
		// Check if it's a topicItem (legacy) or resourceListItem (new)
		if topicItem, ok := item.(topicItem); ok {
			// Simple case-insensitive search
			if strings.Contains(strings.ToLower(topicItem.name), strings.ToLower(query)) {
				// Create highlighted version of the item
				highlightedItem := CreateHighlightedItem(item, query)
				filteredItems = append(filteredItems, highlightedItem)
			}
		} else if resourceItem, ok := item.(resourceListItem); ok {
			// Simple case-insensitive search on resource ID
			if strings.Contains(strings.ToLower(resourceItem.resourceItem.GetID()), strings.ToLower(query)) {
				// Create highlighted version of the item
				highlightedItem := CreateHighlightedItem(item, query)
				filteredItems = append(filteredItems, highlightedItem)
			}
		}
	}

	// Apply natural sorting to filtered results
	SortResourceListNaturally(filteredItems)

	// Store filtered items
	m.filteredItems = filteredItems
	m.resourcesList.SetItems(filteredItems)
	m.searchBar.SetResultCount(len(filteredItems))
	// Exit search mode and focus resources list
	m.searchMode = false
	m.searchBar.Blur()

	if len(filteredItems) == 0 {
		m.statusMessage = fmt.Sprintf("No items found for: %s", query)
	} else {
		m.statusMessage = fmt.Sprintf("Showing %d of %d items (filtered by: %s)", len(filteredItems), len(m.allItems), query)
		// Focus first item in filtered list if available
		if len(filteredItems) > 0 {
			m.resourcesList.Select(0)
		}
	}

	return m, nil
}

// HandleClearSearch processes clear search messages
func (m *MainPageModel) HandleClearSearch(msg clearSearchMsg) (tea.Model, tea.Cmd) {
	// Clear search and reset resources list to original items (without highlighting)
	m.resourcesList.SetItems(m.allItems)
	m.searchBar.SetResultCount(0)
	m.searchMode = false
	m.searchBar.Blur()
	// Reset filter state
	m.isFiltered = false
	m.currentFilter = ""
	m.filteredItems = []list.Item{}
	m.statusMessage = fmt.Sprintf("Showing %d of %d resources", len(m.allItems), len(m.allItems))
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
