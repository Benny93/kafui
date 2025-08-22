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
	// Main page search bar style (different from global)
	mainPageSearchBarStyle = lipgloss.NewStyle().
		BorderStyle(RoundedBorder).
		BorderForeground(Info).
		Padding(0, 1).
		MarginBottom(1)
)

type MainPageModel struct {
	dataSource      api.KafkaDataSource
	resourcesList   list.Model
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
	m.mainContent.SetList(m.resourcesList)
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
	if item := m.resourcesList.SelectedItem(); item != nil {
		if rItem, ok := item.(resourceListItem); ok {
			selectedItem = rItem.resourceItem.GetID()
		} else if tItem, ok := item.(topicItem); ok {
			selectedItem = tItem.name
		}
	}

	m.footer.UpdateConfig(components.FooterConfig{
		Width:         m.width,
		SearchMode:    m.searchMode,
		SelectedItem:  selectedItem,
		TotalItems:    len(m.allItems),
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

// Note: renderResourceButtons, renderShortcuts, and renderFooter methods have been
// moved to reusable components in the components package

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
	// Initialize resources list with custom delegate
	delegate := newCustomDelegate()

	resourcesList := list.New([]list.Item{}, delegate, 0, 0)
	resourcesList.Title = "Kafka Topics"
	resourcesList.SetShowTitle(true)
	resourcesList.SetShowHelp(true)
	resourcesList.SetFilteringEnabled(false) // We'll handle filtering ourselves
	resourcesList.SetShowFilter(false)
	resourcesList.Styles.Title = TitleStyle
	resourcesList.FilterInput.Prompt = "search: "
	resourcesList.FilterInput.PromptStyle = lipgloss.NewStyle().Foreground(Highlight)

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
		resourcesList:   resourcesList,
		searchBar:       searchBar,
		spinner:         sp,
		lastUpdate:      time.Now(),
		statusMessage:   "Welcome to Kafui",
		searchMode:      false,
		allItems:        []list.Item{},
		resourceManager: resourceManager,
		currentResource: currentResource,
		layout:          layout,
		sidebar:         sidebar,
		footer:          footer,
		mainContent:     mainContent,
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

		// Update list and search bar dimensions
		m.resourcesList.SetSize(mainContentWidth-4, contentHeight-3) // Account for borders and margins
		m.searchBar.SetWidth(mainContentWidth - 4)
		return m, nil

	case tea.KeyMsg:
		return m.HandleKeyMsg(msg)

	case topicListMsg:
		m.loading = false
		items := []list.Item(msg)
		m.resourcesList.SetItems(items)
		m.allItems = items // Store all items for filtering

		// Update search suggestions with topic names
		searchSuggestions := make([]string, 0, len(items))
		for _, item := range items {
			if topicItem, ok := item.(topicItem); ok {
				searchSuggestions = append(searchSuggestions, topicItem.name)
			}
		}
		m.searchBar.SetSearchSuggestions(searchSuggestions)

		m.statusMessage = fmt.Sprintf("Showing %d of %d topics", len(items), len(items))
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
			cmds = append(cmds, m.loadTopics())
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
	}

	return m, tea.Batch(cmds...)
}

// switchToResource switches the current view to a different resource type
func (m *MainPageModel) switchToResource(resourceType ResourceType) {
	m.currentResource = m.resourceManager.GetResource(resourceType)
	m.resourcesList.Title = m.currentResource.GetName()

	// Load data for the new resource
	items, err := m.currentResource.GetData()
	if err != nil {
		m.statusMessage = fmt.Sprintf("Error loading %s: %v", m.currentResource.GetName(), err)
		return
	}

	// Convert resource items to list items
	listItems := make([]list.Item, 0, len(items))
	searchSuggestions := make([]string, 0, len(items))

	for _, item := range items {
		listItems = append(listItems, resourceListItem{
			resourceItem: item,
		})
		// Add item ID to search suggestions
		searchSuggestions = append(searchSuggestions, item.GetID())
	}

	// Sort items by ID (name)
	sort.Slice(listItems, func(i, j int) bool {
		item1 := listItems[i].(resourceListItem)
		item2 := listItems[j].(resourceListItem)
		return item1.resourceItem.GetID() < item2.resourceItem.GetID()
	})

	// Sort suggestions as well
	sort.Strings(searchSuggestions)

	m.resourcesList.SetItems(listItems)
	m.allItems = listItems

	// Update search suggestions for the new resource
	m.searchBar.SetSearchSuggestions(searchSuggestions)

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
type switchResourceByNameMsg string

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

		// Sort topic names
		sort.Strings(topicNames)

		items := make([]list.Item, 0, len(topics))
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
