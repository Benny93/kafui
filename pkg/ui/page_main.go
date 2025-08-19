package ui

import (
	"fmt"
	"io"
	"sort"
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
	// Colors
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	info      = lipgloss.AdaptiveColor{Light: "#4A90E2", Dark: "#4A90E2"}
	warning   = lipgloss.AdaptiveColor{Light: "#F5A623", Dark: "#F5A623"}

	// Border styles
	roundedBorder = lipgloss.Border{
		Top:         "",
		Bottom:      "",
		Left:        "",
		Right:       "",
		TopLeft:     "",
		TopRight:    "",
		BottomLeft:  "",
		BottomRight: "",
	}

	// Header styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(highlight).
			Padding(0, 1).
			MarginBottom(1)

	// Main layout styles
	layoutStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Content panel styles
	mainPanelStyle = lipgloss.NewStyle().
			BorderStyle(roundedBorder).
			BorderForeground(subtle).
			Padding(1, 1)

	sidebarPanelStyle = lipgloss.NewStyle().
				BorderStyle(roundedBorder).
				BorderForeground(subtle).
				Padding(1, 2)

	// Main page search bar style (different from global)
	mainPageSearchBarStyle = lipgloss.NewStyle().
			BorderStyle(roundedBorder).
			BorderForeground(info).
			Padding(0, 1).
			MarginBottom(1)

	// Footer styles
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(subtle).
			Padding(0, 1)

	// Text styles
	subtitleStyle = lipgloss.NewStyle().
			Foreground(special).
			Bold(true).
			MarginBottom(1)

	infoStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Italic(true)

	// Resource type indicator
	resourceTypeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(info).
				Bold(true).
				Padding(0, 1).
				MarginRight(1)
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
	allItems        []list.Item
	resourceManager *ResourceManager
	currentResource Resource
	err             error
}

func (m MainPageModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Calculate layout dimensions
	sidebarWidth := 35
	contentWidth := m.width - sidebarWidth - 6 // Account for padding and borders
	contentHeight := m.height - 8              // Account for header and footer (removed status bar)

	// Header section
	resourceIndicator := resourceTypeStyle.Render(strings.ToUpper(string(m.currentResource.GetType())))
	header := headerStyle.
		Width(m.width).
		Render(fmt.Sprintf("%sKafui - Kafka UI", resourceIndicator))

	// Main content area with search and list
	searchSection := mainPageSearchBarStyle.
		Width(contentWidth).
		Render(m.searchBar.View())

	listSection := mainPanelStyle.
		Width(contentWidth).
		Height(contentHeight - 3). // Account for search bar
		Render(m.topicList.View())

	mainContent := lipgloss.JoinVertical(
		lipgloss.Left,
		searchSection,
		listSection,
	)

	// Sidebar with context information
	sidebarContent := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("CONTEXT"),
		infoStyle.Render(m.dataSource.GetContext()),
		lipgloss.NewStyle().MarginTop(2).Render(""),
		subtitleStyle.Render("RESOURCES"),
		lipgloss.NewStyle().MarginBottom(1).Render("Press to switch:"),
		m.renderResourceButtons(),
		lipgloss.NewStyle().MarginTop(2).Render(""),
		subtitleStyle.Render("SHORTCUTS"),
		m.renderShortcuts(),
	)

	sidebar := sidebarPanelStyle.
		Width(sidebarWidth).
		Height(contentHeight).
		Render(sidebarContent)

	// Combine main content and sidebar
	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		mainContent,
		lipgloss.NewStyle().Width(2).Render(""), // Spacer
		sidebar,
	)

	// Footer with key bindings
	footer := footerStyle.Width(m.width).Render(m.renderFooter())

	// Combine all sections
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		layoutStyle.Render(body),
		footer,
	)
}

func (m MainPageModel) renderResourceButtons() string {
	resources := []struct {
		name string
		key  string
		typ  ResourceType
	}{
		{"Topics", "F1", TopicResourceType},
		{"Consumer Groups", "F2", ConsumerGroupResourceType},
		{"Schemas", "F3", SchemaResourceType},
		{"Contexts", "F4", ContextResourceType},
	}

	buttons := make([]string, len(resources))
	for i, res := range resources {
		style := infoStyle
		if m.currentResource.GetType() == res.typ {
			style = lipgloss.NewStyle().
				Foreground(special).
				Bold(true)
		}
		
		buttons[i] = style.Render(fmt.Sprintf("%s %s", res.key, res.name))
	}

	return lipgloss.JoinVertical(lipgloss.Left, buttons...)
}

func (m MainPageModel) renderShortcuts() string {
	shortcuts := []string{
		"↑/↓   Navigate items",
		"Enter   Select item",
		"/       Search",
		"Esc     Cancel search",
		"q       Quit",
	}

	return lipgloss.JoinVertical(lipgloss.Left, shortcuts...)
}

func (m MainPageModel) renderFooter() string {
	// Show different help text based on current mode
	if m.searchMode {
		return "Type to search  Enter: confirm  Esc: cancel"
	}
	
	// Left side: Selection information
	selected := "None"
	if item := m.topicList.SelectedItem(); item != nil {
		if rItem, ok := item.(resourceListItem); ok {
			selected = rItem.resourceItem.GetID()
		} else if tItem, ok := item.(topicItem); ok {
			selected = tItem.name
		}
	}
	leftInfo := fmt.Sprintf("Selected: %s  •  %d items total", selected, len(m.allItems))
	
	// Right side: Status information
	rightInfo := fmt.Sprintf("%s %s  •  Last update: %s",
		m.spinner.View(),
		m.statusMessage,
		m.lastUpdate.Format("15:04:05"),
	)
	
	// Calculate available width for each side
	totalWidth := m.width - 4 // Account for padding
	leftWidth := len(leftInfo)
	rightWidth := len(rightInfo)
	
	// If both fit, use them with proper spacing
	if leftWidth + rightWidth + 3 <= totalWidth {
		spacer := strings.Repeat(" ", totalWidth - leftWidth - rightWidth)
		return leftInfo + spacer + rightInfo
	}
	
	// If they don't fit, truncate the left side
	maxLeftWidth := totalWidth - rightWidth - 3
	if maxLeftWidth > 20 {
		if len(leftInfo) > maxLeftWidth {
			leftInfo = leftInfo[:maxLeftWidth-3] + "..."
		}
		spacer := strings.Repeat(" ", totalWidth - len(leftInfo) - rightWidth)
		return leftInfo + spacer + rightInfo
	}
	
	// Fallback: just show the right info if space is very limited
	return rightInfo
}

type customDelegate struct {
	styles     list.DefaultItemStyles
	itemStyles map[int]lipgloss.Style
}

func newCustomDelegate() list.ItemDelegate {
	delegate := customDelegate{
		itemStyles: make(map[int]lipgloss.Style),
	}

	delegate.styles = list.NewDefaultItemStyles()
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("205")).
		Foreground(lipgloss.Color("0"))

	delegate.styles.SelectedTitle = selectedStyle
	delegate.styles.SelectedDesc = selectedStyle

	return &delegate
}

func (d *customDelegate) Height() int { return 1 }

func (d *customDelegate) Spacing() int { return 0 }

func (d *customDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d *customDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var name, partitions, replication string

	if i, ok := item.(resourceListItem); ok {
		name = i.resourceItem.GetID()
		details := i.resourceItem.GetDetails()
		if p, ok := details["partitions"]; ok {
			partitions = fmt.Sprintf("Partitions: %s", p)
		}
		if r, ok := details["replication"]; ok {
			replication = fmt.Sprintf("Replication: %s", r)
		}
	} else if i, ok := item.(topicItem); ok {
		name = i.name
		partitions = fmt.Sprintf("Partitions: %d", i.topic.NumPartitions)
		replication = fmt.Sprintf("Replication: %d", i.topic.ReplicationFactor)
	}

	itemStyle := d.styles.NormalTitle
	if index == m.Index() {
		itemStyle = d.styles.SelectedTitle
	}

	// Create a single row with columns
	row := fmt.Sprintf("%-40s %-20s %-20s", name, partitions, replication)
	fmt.Fprint(w, itemStyle.Render(row))
}

func NewMainPage(ds api.KafkaDataSource) MainPageModel {
	// Initialize topic list with custom delegate
	delegate := newCustomDelegate()

	topicList := list.New([]list.Item{}, delegate, 0, 0)
	topicList.Title = "Kafka Topics"
	topicList.SetShowTitle(true)
	topicList.SetShowHelp(true)
	topicList.SetFilteringEnabled(false) // We'll handle filtering ourselves
	topicList.SetShowFilter(false)
	topicList.Styles.Title = titleStyle
	topicList.FilterInput.Prompt = "search: "
	topicList.FilterInput.PromptStyle = lipgloss.NewStyle().Foreground(highlight)

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

		// Calculate available space for content
		sidebarWidth := 35
		mainContentWidth := msg.Width - sidebarWidth - 6 // Account for margins and borders
		contentHeight := msg.Height - 8                  // Account for header, footer and margins (removed status bar)

		// Update list and search bar dimensions
		m.topicList.SetSize(mainContentWidth-4, contentHeight-3) // Account for borders and margins
		m.searchBar.SetWidth(mainContentWidth - 4)
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
		m.statusMessage = fmt.Sprintf("Showing %d of %d topics", len(items), len(items))
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
			// Check if it's a topicItem (legacy) or resourceListItem (new)
			if topicItem, ok := item.(topicItem); ok {
				// Simple case-insensitive search
				if strings.Contains(strings.ToLower(topicItem.name), strings.ToLower(query)) {
					filteredItems = append(filteredItems, item)
				}
			} else if resourceItem, ok := item.(resourceListItem); ok {
				// Simple case-insensitive search on resource ID
				if strings.Contains(strings.ToLower(resourceItem.resourceItem.GetID()), strings.ToLower(query)) {
					filteredItems = append(filteredItems, item)
				}
			}
		}

		m.topicList.SetItems(filteredItems)
		m.searchBar.SetResultCount(len(filteredItems))
		m.searchMode = false
		m.searchBar.Blur()

		if len(filteredItems) == 0 {
			m.statusMessage = fmt.Sprintf("No items found for: %s", query)
		} else {
			m.statusMessage = fmt.Sprintf("Showing %d of %d items", len(filteredItems), len(m.allItems))
		}

		return m, nil

	case clearSearchMsg:
		// Clear search and reset topic list
		m.topicList.SetItems(m.allItems)
		m.searchBar.SetResultCount(0)
		m.searchMode = false
		m.searchBar.Blur()
		m.statusMessage = fmt.Sprintf("Showing %d of %d topics", len(m.allItems), len(m.allItems))
		return m, nil
	case switchResourceMsg:
		// Switch to a different resource type
		m.switchToResource(ResourceType(msg))
		return m, nil
	}

	return m, tea.Batch(cmds...)
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

	// Sort items by ID (name)
	sort.Slice(listItems, func(i, j int) bool {
		item1 := listItems[i].(resourceListItem)
		item2 := listItems[j].(resourceListItem)
		return item1.resourceItem.GetID() < item2.resourceItem.GetID()
	})

	m.topicList.SetItems(listItems)
	m.allItems = listItems
	m.statusMessage = fmt.Sprintf("Showing %d of %d %s", len(listItems), len(listItems), m.currentResource.GetName())
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

	// Create a slice of topic names for sorting
	topicNames := make([]string, 0, len(topics))
	for name := range topics {
		topicNames = append(topicNames, name)
	}

	// Sort topic names
	sort.Strings(topicNames)

	items := make([]list.Item, 0, len(topics))
	for _, name := range topicNames {
		topic := topics[name]
		items = append(items, topicItem{
			name:  name,
			topic: topic,
		})
	}

	return topicListMsg(items)
}

