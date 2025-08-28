package ui

import (
	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/router"
	messagedetailpage "github.com/Benny93/kafui/pkg/ui/pages/message_detail"
	mainpage "github.com/Benny93/kafui/pkg/ui/pages/main"
	resourcedetailpage "github.com/Benny93/kafui/pkg/ui/pages/resource_detail"
	topicpage "github.com/Benny93/kafui/pkg/ui/pages/topic"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Page constants for navigation
type pageType int

const (
	mainPageType pageType = iota
	topicPageType
	detailPageType
	resourceDetailPageType
)

// Model represents the main application state
type Model struct {
	dataSource         api.KafkaDataSource
	Router             *router.Router     // Exported for testing
	ShowHelp           bool               // Exported for testing
	HelpSystem         *core.HelpSystem   // Enhanced help system
	FocusManager       *core.FocusManager // Focus management
	width              int
	height             int
	
	// Legacy fields for backward compatibility (will be removed)
	currentPage        pageType
	mainPage           *mainpage.MainPageModel
	topicPage          *topicpage.Model
	detailPage         *messagedetailpage.Model
	resourceDetailPage *resourcedetailpage.Model
}

// Key mappings
type keyMap struct {
	Search    key.Binding
	TopicMode key.Binding
	Back      key.Binding
	Quit      key.Binding
}

var keys = keyMap{
	Search: key.NewBinding(
		key.WithKeys(":"),
		key.WithHelp(":", "search"),
	),
	TopicMode: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter topics"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
}

// minimalResourceItem is a minimal implementation of ResourceItem for table navigation
type minimalResourceItem struct {
	id string
}

func (m *minimalResourceItem) GetID() string {
	return m.id
}

func (m *minimalResourceItem) GetValues() []string {
	return []string{m.id}
}

func (m *minimalResourceItem) GetDetails() map[string]string {
	return map[string]string{
		"Name": m.id,
	}
}

func initialModel(dataSource api.KafkaDataSource) Model {
	mp := mainpage.NewModel(dataSource)
	return Model{
		dataSource:  dataSource,
		currentPage: mainPageType,
		mainPage:    mp,
	}
}

// initialModelWithRouter creates a new Model using the router-based navigation
func initialModelWithRouter(dataSource api.KafkaDataSource) Model {
	r := router.NewRouter(dataSource)
	helpSystem := core.NewHelpSystem()
	focusManager := core.NewFocusManager()
	
	return Model{
		dataSource:   dataSource,
		Router:       r,
		ShowHelp:     false,
		HelpSystem:   helpSystem,
		FocusManager: focusManager,
	}
}

func (m Model) Init() tea.Cmd {
	// Router-based initialization
	if m.Router != nil {
		return m.Router.Init()
	}
	
	// Legacy initialization
	return m.mainPage.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Router-based update handling
	if m.Router != nil {
		return m.updateWithRouter(msg)
	}
	
	// Legacy update handling
	return m.updateLegacy(msg)
}

// updateWithRouter handles updates using the new router system
func (m Model) updateWithRouter(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.Router.SetDimensions(msg.Width, msg.Height)
		m.HelpSystem.SetDimensions(msg.Width, msg.Height)

	case tea.KeyMsg:
		// Handle focus management first (if not in help mode)
		if !m.ShowHelp {
			if cmd := m.FocusManager.HandleKeyMsg(msg); cmd != nil {
				return m, cmd
			}
		}
		
		// Handle global key bindings
		switch {
		case key.Matches(msg, core.DefaultGlobalKeys.Help):
			m.ShowHelp = !m.ShowHelp
			m.HelpSystem.Toggle()
			// Update help system with current page
			if currentPage := m.Router.GetCurrentPage(); currentPage != nil {
				m.HelpSystem.SetCurrentPage(currentPage)
			}
			return m, nil
		case key.Matches(msg, core.DefaultGlobalKeys.Quit):
			return m, tea.Quit
		case key.Matches(msg, core.DefaultGlobalKeys.Back):
			if !m.ShowHelp {
				return m, m.Router.Back()
			} else {
				// Close help if it's open
				m.ShowHelp = false
				m.HelpSystem.Hide()
				return m, nil
			}
		}
	}

	// Handle router updates if not in help mode
	if !m.ShowHelp {
		updatedRouter, cmd := m.Router.Update(msg)
		if updatedRouter != nil {
			// Router implements tea.Model interface, but we need to maintain our Model type
			// The router updates its internal state, so we don't need to reassign
		}
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// updateLegacy handles updates using the legacy system (for backward compatibility)
func (m Model) updateLegacy(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Propagate dimensions to all pages immediately
		m.updatePageDimensions()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Back):
			shared.DebugLog("Root UI handling ESC key - CurrentPage: %v", m.currentPage)

			// If we're on the main page, let the main page handle the Back key first
			// (e.g., to exit search mode)
			if m.currentPage == mainPageType {
				// Let the main page handle the Back key
				mainModel, cmd := m.mainPage.Update(msg)
				if updatedMainPage, ok := mainModel.(*mainpage.MainPageModel); ok {
					m.mainPage = updatedMainPage
				}
				return m, cmd
			}

			// Otherwise handle as back navigation for sub-pages
			if m.currentPage > mainPageType {
				m.currentPage--
				// Clean up pages when leaving them
				if m.currentPage != topicPageType && m.topicPage != nil {
					m.topicPage = nil
				}
				if m.currentPage != detailPageType && m.detailPage != nil {
					m.detailPage = nil
				}
				if m.currentPage != resourceDetailPageType && m.resourceDetailPage != nil {
					m.resourceDetailPage = nil
				}
			}
			return m, nil
		}

	case core.PageChangeMsg:
		// Handle page changes based on PageID
		switch msg.PageID {
		case "main":
			m.currentPage = mainPageType
		case "topic":
			m.currentPage = topicPageType
		case "detail":
			m.currentPage = detailPageType
		case "resource_detail":
			m.currentPage = resourceDetailPageType
		}
		// Initialize topic page if needed
		if m.currentPage == topicPageType && m.topicPage == nil {
			// Get selected item from main page table
			if selectedItem := m.mainPage.GetSelectedResourceItem(); selectedItem != nil {
				// Handle different item types to extract topic information
				var topicName string
				var topicDetails api.Topic

				// Try to extract topic data from the selected item
				if msg.Data != nil {
					// Use data passed with the page change message
					if topicData, ok := msg.Data.(map[string]interface{}); ok {
						if name, hasName := topicData["name"]; hasName {
							topicName = name.(string)
						}
						if details, hasDetails := topicData["topic"]; hasDetails {
							topicDetails = details.(api.Topic)
						}
					}
				} else {
					// Fallback - create a minimal topic
					topicName = "unknown"
					topicDetails = api.Topic{
						NumPartitions:     1,
						ReplicationFactor: 1,
						ReplicaAssignment: make(map[int32][]int32),
						ConfigEntries:     make(map[string]*string),
					}
				}

				m.topicPage = topicpage.NewModel(m.dataSource, topicName, topicDetails)
				cmds = append(cmds, m.topicPage.Init())
			} else {
				// Fallback to main page if no topic selected
				m.currentPage = mainPageType
			}
		}
		// Initialize detail page if needed
		if m.currentPage == detailPageType && m.detailPage == nil {
			// Check if we have message data passed via the page change
			if msg.Data != nil {
				if messageData, ok := msg.Data.(api.Message); ok {
					// Get topic name from topic page or use default
					topicName := "unknown"
					if m.topicPage != nil {
						topicName = m.topicPage.GetTopicName()
					}
					m.detailPage = messagedetailpage.NewModel(m.dataSource, topicName, messageData)
				}
			}
		}
		// Initialize resource detail page if needed
		if m.currentPage == resourceDetailPageType && m.resourceDetailPage == nil {
			// Get selected resource from main page table
			if selectedItem := m.mainPage.GetSelectedResourceItem(); selectedItem != nil {
				// Try to convert to ResourceItem interface
				if resourceItem, ok := selectedItem.(shared.ResourceItem); ok {
					// Get the resource type from the main page
					resourceType := "unknown"
					// Add getter method to main page to get current resource type
					m.resourceDetailPage = resourcedetailpage.NewModel(resourceItem, resourceType)
					cmds = append(cmds, m.resourceDetailPage.Init())
				} else {
					// Create a minimal resource item if needed
					minimalResource := &minimalResourceItem{
						id: "unknown",
					}
					m.resourceDetailPage = resourcedetailpage.NewModel(minimalResource, "unknown")
					cmds = append(cmds, m.resourceDetailPage.Init())
				}
			} else {
				// Fallback to main page if no resource selected
				m.currentPage = mainPageType
			}
		}
		// Clean up pages when leaving them
		if m.currentPage != detailPageType && m.detailPage != nil {
			m.detailPage = nil
		}
		if m.currentPage != resourceDetailPageType && m.resourceDetailPage != nil {
			m.resourceDetailPage = nil
		}
		
		// Update dimensions for newly created pages
		m.updatePageDimensions()
		
		return m, tea.Batch(cmds...)
	}

	// Handle updates for sub-components
	switch m.currentPage {
	case mainPageType:
		var cmd tea.Cmd
		mainModel, cmd := m.mainPage.Update(msg)
		if updatedMainPage, ok := mainModel.(*mainpage.MainPageModel); ok {
			m.mainPage = updatedMainPage
		}
		cmds = append(cmds, cmd)

	case topicPageType:
		if m.topicPage != nil {
			var cmd tea.Cmd
			topicModel, cmd := m.topicPage.Update(msg)
			if updatedTopicPage, ok := topicModel.(*topicpage.Model); ok {
				m.topicPage = updatedTopicPage
			}
			cmds = append(cmds, cmd)
		}

	case detailPageType:
		if m.detailPage != nil {
			var cmd tea.Cmd
			detailModel, cmd := m.detailPage.Update(msg)
			if updatedDetailPage, ok := detailModel.(*messagedetailpage.Model); ok {
				m.detailPage = updatedDetailPage
			}
			cmds = append(cmds, cmd)
		}

	case resourceDetailPageType:
		if m.resourceDetailPage != nil {
			var cmd tea.Cmd
			resourceDetailModel, cmd := m.resourceDetailPage.Update(msg)
			if updatedResourceDetailPage, ok := resourceDetailModel.(*resourcedetailpage.Model); ok {
				m.resourceDetailPage = updatedResourceDetailPage
			}
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// updatePageDimensions propagates the current dimensions to all active pages
func (m *Model) updatePageDimensions() {
	if m.width > 0 && m.height > 0 {
		// Update dimensions for all active pages
		if m.mainPage != nil {
			m.mainPage.SetDimensions(m.width, m.height)
		}
		if m.topicPage != nil {
			m.topicPage.SetDimensions(m.width, m.height)
		}
		if m.detailPage != nil {
			m.detailPage.SetDimensions(m.width, m.height)
		}
		if m.resourceDetailPage != nil {
			m.resourceDetailPage.SetDimensions(m.width, m.height)
		}
	}
}

func (m Model) View() string {
	// Router-based view rendering
	if m.Router != nil {
		return m.viewWithRouter()
	}
	
	// Legacy view rendering
	return m.viewLegacy()
}

// viewWithRouter renders the view using the router system
func (m Model) viewWithRouter() string {
	if m.ShowHelp {
		return m.renderEnhancedHelp()
	}
	
	return m.Router.View()
}

// renderEnhancedHelp renders the enhanced help overlay
func (m Model) renderEnhancedHelp() string {
	// Use the enhanced help system
	return m.HelpSystem.Render()
}

// renderHelp renders the legacy help overlay (kept for backward compatibility)
func (m Model) renderHelp() string {
	currentPage := m.Router.GetCurrentPage()
	if currentPage == nil {
		return "Help not available"
	}
	
	helpContent := "Kafui Help\n\n"
	helpContent += "Global Keys:\n"
	
	// Add global key bindings
	globalBindings := core.DefaultGlobalKeys.GetAllBindings()
	for _, binding := range globalBindings {
		help := binding.Help()
		helpContent += "  " + help.Key + " - " + help.Desc + "\n"
	}
	
	helpContent += "\nPage-specific Keys:\n"
	
	// Add page-specific key bindings
	pageBindings := currentPage.GetHelp()
	for _, binding := range pageBindings {
		help := binding.Help()
		helpContent += "  " + help.Key + " - " + help.Desc + "\n"
	}
	
	helpContent += "\nPress '?' again to close help"
	
	return helpContent
}

// viewLegacy renders the view using the legacy system
func (m Model) viewLegacy() string {
	// Update page dimensions if needed (fallback in case updatePageDimensions wasn't called)
	if m.width > 0 && m.height > 0 {
		switch m.currentPage {
		case mainPageType:
			if m.mainPage != nil {
				m.mainPage.SetDimensions(m.width, m.height)
			}
		case topicPageType:
			if m.topicPage != nil {
				m.topicPage.SetDimensions(m.width, m.height)
			}
		case detailPageType:
			if m.detailPage != nil {
				m.detailPage.SetDimensions(m.width, m.height)
			}
		case resourceDetailPageType:
			if m.resourceDetailPage != nil {
				m.resourceDetailPage.SetDimensions(m.width, m.height)
			}
		}
	}

	switch m.currentPage {
	case mainPageType:
		if m.mainPage != nil {
			return m.mainPage.View()
		}
		return "Main page not initialized"
	case topicPageType:
		if m.topicPage != nil {
			return m.topicPage.View()
		}
		return "Topic page not initialized"
	case detailPageType:
		if m.detailPage != nil {
			return m.detailPage.View()
		}
		return "Detail page not initialized"
	case resourceDetailPageType:
		if m.resourceDetailPage != nil {
			return m.resourceDetailPage.View()
		}
		return "Resource detail page not initialized"
	default:
		return "Unknown page"
	}
}

// NewUIModel creates a new UI model (public API)
func NewUIModel(dataSource api.KafkaDataSource) Model {
	return initialModel(dataSource)
}

// NewUIModelWithRouter creates a new UI model using the router-based navigation
func NewUIModelWithRouter(dataSource api.KafkaDataSource) Model {
	return initialModelWithRouter(dataSource)
}

// InitializeModel is an alias for backwards compatibility
func InitializeModel(dataSource api.KafkaDataSource) Model {
	return NewUIModel(dataSource)
}