package mainpage

import (
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// Keys handles key bindings for the main page
type Keys struct {
	bindings keyMap
}

type keyMap struct {
	Search       key.Binding
	ResourceMode key.Binding
	Back         key.Binding
	Quit         key.Binding
	Enter        key.Binding
	Navigation   NavigationKeys
}

type NavigationKeys struct {
	Up   key.Binding
	Down key.Binding
	Home key.Binding
	End  key.Binding
}

// NewKeys creates a new Keys instance
func NewKeys() *Keys {
	return &Keys{
		bindings: keyMap{
			Search: key.NewBinding(
				key.WithKeys("/"),
				key.WithHelp("/", "search"),
			),
			ResourceMode: key.NewBinding(
				key.WithKeys(":"),
				key.WithHelp(":", "resource mode"),
			),
			Back: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "back"),
			),
			Quit: key.NewBinding(
				key.WithKeys("ctrl+c", "q"),
				key.WithHelp("ctrl+c/q", "quit"),
			),
			Enter: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "select"),
			),
			Navigation: NavigationKeys{
				Up: key.NewBinding(
					key.WithKeys("k", "up"),
					key.WithHelp("k/↑", "up"),
				),
				Down: key.NewBinding(
					key.WithKeys("j", "down"),
					key.WithHelp("j/↓", "down"),
				),
				Home: key.NewBinding(
					key.WithKeys("g", "home"),
					key.WithHelp("g/home", "top"),
				),
				End: key.NewBinding(
					key.WithKeys("G", "end"),
					key.WithHelp("G/end", "bottom"),
				),
			},
		},
	}
}

// HandleKey processes key events
func (k *Keys) HandleKey(model *Model, msg tea.KeyMsg) tea.Cmd {
	var cmds []tea.Cmd

	// Log key event details
	shared.DebugLog("Key Event - Type: %v, String: %s, SearchMode: %v", msg.Type, msg.String(), model.searchMode)

	// Handle ESC key first (back navigation)
	if key.Matches(msg, k.bindings.Back) {
		shared.DebugLog("Back Key Event - Type: %v, String: %s, SearchMode: %v", msg.Type, msg.String(), model.searchMode)
		return k.handleBack(model)
	}

	// Handle other specific key combinations
	switch {
	case key.Matches(msg, k.bindings.Quit):
		return k.handleQuit(model)
	case key.Matches(msg, k.bindings.Search):
		return k.handleSearch(model)
	case key.Matches(msg, k.bindings.ResourceMode):
		return k.handleResourceMode(model)
	case key.Matches(msg, k.bindings.Enter):
		return k.handleEnter(model, msg)
	}

	// Handle navigation keys if not in search mode
	if !model.searchMode {
		switch {
		case key.Matches(msg, k.bindings.Navigation.Up):
			return k.handleNavigation(model, "up")
		case key.Matches(msg, k.bindings.Navigation.Down):
			return k.handleNavigation(model, "down")
		case key.Matches(msg, k.bindings.Navigation.Home):
			return k.handleNavigation(model, "home")
		case key.Matches(msg, k.bindings.Navigation.End):
			return k.handleNavigation(model, "end")
		}
	}

	// Handle 'q' key for quit (only if not in search mode)
	if msg.String() == "q" && !model.searchMode {
		return tea.Quit
	}

	// If in search mode, let the search bar handle keys
	if model.searchMode {
		var cmd tea.Cmd
		model.searchBar, cmd = model.searchBar.Update(msg)
		cmds = append(cmds, cmd)
		return tea.Batch(cmds...)
	}

	// Default table navigation handling
	var cmd tea.Cmd
	model.resourcesTable, cmd = model.resourcesTable.Update(msg)
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}

func (k *Keys) handleBack(model *Model) tea.Cmd {
	// If search bar is focused, blur it and return focus to list
	if model.searchMode {
		model.searchMode = false
		model.searchBar.Blur()
		model.searchBar.SetValue("")
		// Reset the resources table to show all rows (without highlighting)
		model.resourcesTable.SetRows(model.allRows)
		// Reset filter state
		model.isFiltered = false
		model.currentFilter = ""
		model.filteredRows = []table.Row{}
		model.statusMessage = "Search cancelled"
		return nil
	}
	// If not in search mode, this will be handled by the parent UI
	return nil
}

func (k *Keys) handleQuit(model *Model) tea.Cmd {
	return tea.Quit
}

func (k *Keys) handleSearch(model *Model) tea.Cmd {
	// Focus search bar for normal search
	model.searchMode = true
	model.searchBar.EnterSearchMode()
	model.statusMessage = "Search mode: Type to filter items"
	return model.searchBar.Focus()
}

func (k *Keys) handleResourceMode(model *Model) tea.Cmd {
	// Focus search bar for resource switching
	model.searchMode = true
	model.searchBar.EnterResourceMode()
	model.statusMessage = "Resource mode: Type resource name (topics, consumer-groups, schemas, contexts)"
	return model.searchBar.Focus()
}

func (k *Keys) handleEnter(model *Model, msg tea.KeyMsg) tea.Cmd {
	var cmds []tea.Cmd

	shared.DebugLog("Enter Key event - Type: %v, String: %s, SearchMode: %v", msg.Type, msg.String(), model.searchMode)

	// If in search mode, handle based on search type
	if model.searchMode {
		if model.searchBar.IsResourceMode() {
			// Resource mode: let search bar handle to switch resource
			var cmd tea.Cmd
			model.searchBar, cmd = model.searchBar.Update(msg)
			cmds = append(cmds, cmd)
			return tea.Batch(cmds...)
		} else {
			// Normal search mode: trigger search and exit search mode to focus resources list
			query := model.searchBar.Value()
			if query != "" {
				// Trigger search
				var cmd tea.Cmd
				model.searchBar, cmd = model.searchBar.Update(msg)
				cmds = append(cmds, cmd)
				return tea.Batch(cmds...)
			} else {
				// Empty search, just exit search mode and focus resources list
				model.searchMode = false
				model.searchBar.Blur()
				model.statusMessage = "Search cancelled"
				return nil
			}
		}
	}

	// If not in search mode and a row is selected, navigate to appropriate page
	if selectedItem := model.GetSelectedResourceItem(); selectedItem != nil {
		// Check the current resource type to determine navigation
		if model.currentResource.GetType() == TopicResourceType {
			// Navigate to topic page for topics and include topic data
			if topicItem, ok := selectedItem.(*TopicResourceItem); ok {
				// Create topic data structure for the page change
				topicData := map[string]interface{}{
					"name":  topicItem.GetID(),
					"topic": topicItem.GetTopic(),
				}
				return core.NewPageChangeMsg("topic", topicData)
			} else {
				// Fallback if type assertion fails
				return core.NewPageChangeMsg("topic", nil)
			}
		} else {
			// Navigate to resource detail page for other resources
			return core.NewPageChangeMsg("resource_detail", selectedItem)
		}
	}

	return tea.Batch(cmds...)
}

func (k *Keys) handleNavigation(model *Model, direction string) tea.Cmd {
	switch direction {
	case "up":
		model.resourcesTable.MoveUp(1)
	case "down":
		model.resourcesTable.MoveDown(1)
	case "home":
		model.resourcesTable.GotoTop()
	case "end":
		model.resourcesTable.GotoBottom()
	}
	return nil
}

// GetKeyBindings returns the key bindings for help display
func (k *Keys) GetKeyBindings() []key.Binding {
	return []key.Binding{
		k.bindings.Search,
		k.bindings.ResourceMode,
		k.bindings.Back,
		k.bindings.Quit,
		k.bindings.Enter,
		k.bindings.Navigation.Up,
		k.bindings.Navigation.Down,
		k.bindings.Navigation.Home,
		k.bindings.Navigation.End,
	}
}
