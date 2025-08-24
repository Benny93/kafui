package ui

import (
	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
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
	currentPage        pageType
	mainPage           *mainpage.Model
	topicPage          *topicpage.Model
	detailPage         *messagedetailpage.Model
	resourceDetailPage *resourcedetailpage.Model
	width              int
	height             int
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

func (m Model) Init() tea.Cmd {
	// Initialize the main page
	return m.mainPage.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				if updatedMainPage, ok := mainModel.(*mainpage.Model); ok {
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
		if updatedMainPage, ok := mainModel.(*mainpage.Model); ok {
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

// InitializeModel is an alias for backwards compatibility
func InitializeModel(dataSource api.KafkaDataSource) Model {
	return NewUIModel(dataSource)
}