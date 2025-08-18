package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#1a1a1a")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#3c3c3c")).
			Padding(0, 1)

	// List styles
	docStyle = lipgloss.NewStyle().Margin(1, 2)

	// Custom colors
	highlightColor = lipgloss.Color("205")
)

type MainPageModel struct {
	dataSource      api.KafkaDataSource
	topicList       list.Model
	searchBar       components.SearchBarModel
	spinner         spinner.Model
	statusMessage   string
	lastUpdate      time.Time
	width           int
	height          int
	loading         bool
	searchMode      bool
	allItems        []list.Item // Store all items for filtering
	resourceManager *ResourceManager
	currentResource Resource
	err             error
}

func NewMainPage(ds api.KafkaDataSource) MainPageModel {
	// Initialize topic list with custom delegate
	delegate := list.NewDefaultDelegate()
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("205")).
		Foreground(lipgloss.Color("0"))

	delegate.Styles.SelectedTitle = selectedStyle
	delegate.Styles.SelectedDesc = selectedStyle

	topicList := list.New([]list.Item{}, delegate, 0, 0)
	topicList.Title = "Kafka Topics"
	topicList.SetShowTitle(true)
	topicList.SetShowHelp(true)
	topicList.SetFilteringEnabled(false) // We'll handle filtering ourselves
	topicList.SetShowFilter(false)
	topicList.Styles.Title = titleStyle
	topicList.FilterInput.Prompt = "search: "
	topicList.FilterInput.PromptStyle = lipgloss.NewStyle().Foreground(highlightColor)

	// Initialize resource manager
	resourceManager := NewResourceManager(ds)
	currentResource := resourceManager.GetResource(TopicResourceType)

	// Initialize search bar
	searchBar := components.NewSearchBar(
		components.WithPlaceholder("Press / to search topics..."),
		components.WithOnSearch(func(query string) tea.Msg {
			return searchTopicsMsg(query)
		}),
		components.WithOnClear(func() tea.Msg {
			return clearSearchMsg{}
		}),
	)

	// Initialize spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return MainPageModel{
		dataSource:      ds,
		topicList:       topicList,
		searchBar:       searchBar,
		spinner:         sp,
		lastUpdate:      time.Now(),
		statusMessage:   "Welcome to Kafui",
		searchMode:      false,
		allItems:        []list.Item{},
		resourceManager: resourceManager,
		currentResource: currentResource,
	}
}

func (m *MainPageModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadTopics,
		m.spinner.Tick,
		m.updateTimer,
	)
}

func (m *MainPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.topicList.SetSize(msg.Width, msg.Height-4)
		m.searchBar.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		// Handle general key presses
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "/":
			// Focus search bar
			m.searchMode = true
			m.statusMessage = "Search mode: Type to filter topics"
			cmds = append(cmds, m.searchBar.Focus())
			return m, tea.Batch(cmds...)
		case "esc":
			// If search bar is focused, blur it and return focus to list
			if m.searchMode {
				m.searchMode = false
				m.searchBar.Blur()
				m.searchBar.SetValue("")
				// Reset the topic list to show all items
				m.topicList.SetItems(m.allItems)
				m.statusMessage = "Search cancelled"
				return m, nil
			}
		case "enter":
			if m.searchMode && m.searchBar.Value() != "" {
				// Check if the search text matches a resource type
				searchText := strings.ToLower(m.searchBar.Value())
				switch searchText {
				case "topics", "topic":
					m.switchToResource(TopicResourceType)
					m.searchMode = false
					m.searchBar.Blur()
					m.searchBar.SetValue("")
					return m, nil
				case "consumers", "consumer", "groups", "consumer-groups":
					m.switchToResource(ConsumerGroupResourceType)
					m.searchMode = false
					m.searchBar.Blur()
					m.searchBar.SetValue("")
					return m, nil
				case "schemas", "schema":
					m.switchToResource(SchemaResourceType)
					m.searchMode = false
					m.searchBar.Blur()
					m.searchBar.SetValue("")
					return m, nil
				case "contexts", "context":
					m.switchToResource(ContextResourceType)
					m.searchMode = false
					m.searchBar.Blur()
					m.searchBar.SetValue("")
					return m, nil
				default:
					// Add search to history and trigger search
					query := m.searchBar.Value()
					m.searchBar.SetValue("")
					return m, func() tea.Msg {
						return searchTopicsMsg(query)
					}
				}
			} else if m.topicList.SelectedItem() != nil && !m.searchMode {
				// Let the main UI model handle navigation to topic page
				return m, func() tea.Msg {
					return pageChangeMsg(topicPage)
				}
			}
		}

		// If in search mode, let the search bar handle keys
		if m.searchMode {
			var cmd tea.Cmd
			m.searchBar, cmd = m.searchBar.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		} else {
			// Normal navigation
			switch msg.String() {
			case "j", "down":
				m.topicList.CursorDown()
			case "k", "up":
				m.topicList.CursorUp()
			case "g", "home":
				m.topicList.Select(0)
			case "G", "end":
				m.topicList.Select(len(m.topicList.Items()) - 1)
			default:
				var cmd tea.Cmd
				m.topicList, cmd = m.topicList.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case topicListMsg:
		m.loading = false
		items := []list.Item(msg)
		m.topicList.SetItems(items)
		m.allItems = items // Store all items for filtering
		m.statusMessage = fmt.Sprintf("Loaded %d topics", len(items))
		return m, tea.Batch(
			m.spinner.Tick,
			m.updateTimer,
		)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case timerTickMsg:
		m.lastUpdate = time.Now()
		cmds = append(cmds, m.updateTimer)
		if !m.loading {
			cmds = append(cmds, m.loadTopics)
		}

	case errorMsg:
		m.loading = false
		m.err = msg
		m.statusMessage = fmt.Sprintf("Error: %v", msg)

	case searchTopicsMsg:
		// Handle search query
		query := string(msg)
		m.statusMessage = fmt.Sprintf("Searching for: %s", query)
		
		// Filter the topic list
		filteredItems := []list.Item{}
		
		for _, item := range m.allItems {
			if topicItem, ok := item.(topicItem); ok {
				// Simple case-insensitive search
				if strings.Contains(strings.ToLower(topicItem.name), strings.ToLower(query)) {
					filteredItems = append(filteredItems, item)
				}
			}
		}
		
		m.topicList.SetItems(filteredItems)
		m.searchBar.SetResultCount(len(filteredItems))
		m.searchMode = false
		m.searchBar.Blur()
		
		if len(filteredItems) == 0 {
			m.statusMessage = fmt.Sprintf("No topics found for: %s", query)
		} else {
			m.statusMessage = fmt.Sprintf("Found %d topics for: %s", len(filteredItems), query)
		}
		
		return m, nil

	case clearSearchMsg:
		// Clear search and reset topic list
		m.topicList.SetItems(m.allItems)
		m.searchBar.SetResultCount(0)
		m.searchMode = false
		m.searchBar.Blur()
		m.statusMessage = "Search cleared"
		return m, nil
	case switchResourceMsg:
		// Switch to a different resource type
		m.switchToResource(ResourceType(msg))
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

func (m *MainPageModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Status bar
	status := fmt.Sprintf("%s %s Last update: %s",
		m.spinner.View(),
		m.statusMessage,
		m.lastUpdate.Format("15:04:05"),
	)

	statusBar := statusStyle.Render(status)

	// Search bar
	searchBar := m.searchBar.View()

	// Main content with proper styling
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		searchBar,
		m.topicList.View(),
	)

	// Wrap in document style
	doc := docStyle.Render(content)

	// Add status bar at the bottom
	return lipgloss.JoinVertical(
		lipgloss.Left,
		doc,
		statusBar,
	)
}

// switchToResource switches the current view to a different resource type
func (m *MainPageModel) switchToResource(resourceType ResourceType) {
	m.currentResource = m.resourceManager.GetResource(resourceType)
	m.topicList.Title = m.currentResource.GetName()
	
	// Load data for the new resource
	items, err := m.currentResource.GetData()
	if err != nil {
		m.statusMessage = fmt.Sprintf("Error loading %s: %v", m.currentResource.GetName(), err)
		return
	}
	
	// Convert resource items to list items
	listItems := make([]list.Item, 0, len(items))
	for _, item := range items {
		listItems = append(listItems, resourceListItem{
			resourceItem: item,
		})
	}
	
	m.topicList.SetItems(listItems)
	m.allItems = listItems
	m.statusMessage = fmt.Sprintf("Showing %d %s", len(listItems), m.currentResource.GetName())
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

func (m MainPageModel) updateTimer() tea.Msg {
	time.Sleep(5 * time.Second)
	return timerTickMsg(time.Now())
}

func (m *MainPageModel) loadTopics() tea.Msg {
	m.loading = true
	topics, err := m.dataSource.GetTopics()
	if err != nil {
		return errorMsg(err)
	}

	items := make([]list.Item, 0, len(topics))
	for name, topic := range topics {
		items = append(items, topicItem{
			name:  name,
			topic: topic,
		})
	}

	return topicListMsg(items)
}