package ui

import (
	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

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

type page int

const (
	mainPage page = iota
	topicPage
	detailPage
	resourceDetailPage
)

// Model represents the main application state
type Model struct {
	dataSource         api.KafkaDataSource
	currentPage        page
	mainPage           *MainPageModel
	topicPage          *TopicPageModel
	detailPage         *DetailPageModel
	resourceDetailPage *ResourceDetailPageModel
	width              int
	height             int
} // Custom key mappings
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

func initialModel(dataSource api.KafkaDataSource) Model {
	mp := NewMainPage(dataSource)
	return Model{
		dataSource:  dataSource,
		currentPage: mainPage,
		mainPage:    &mp,
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

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Back):
			shared.DebugLog("Root UI handling ESC key - CurrentPage: %v", m.currentPage)

			// If we're on the main page, let the main page handle the Back key first
			// (e.g., to exit search mode)
			if m.currentPage == mainPage {
				// Let the main page handle the Back key
				mainModel, cmd := m.mainPage.Update(msg)
				if updatedMainPage, ok := mainModel.(*MainPageModel); ok {
					m.mainPage = updatedMainPage
				}
				return m, cmd
			}

			// Otherwise handle as back navigation for sub-pages
			if m.currentPage > mainPage {
				m.currentPage--
				// Clean up pages when leaving them
				if m.currentPage != topicPage && m.topicPage != nil {
					m.topicPage = nil
				}
				if m.currentPage != detailPage && m.detailPage != nil {
					m.detailPage = nil
				}
				if m.currentPage != resourceDetailPage && m.resourceDetailPage != nil {
					m.resourceDetailPage = nil
				}
			}
			return m, nil
		}

	case pageChangeMsg:
		m.currentPage = page(msg)
		// Initialize topic page if needed
		if m.currentPage == topicPage && m.topicPage == nil {
			// Get selected item from main page table
			if selectedItem := m.mainPage.getSelectedResourceItem(); selectedItem != nil {
				// For now, create a dummy topic since we don't have the full topic structure
				// This is a simplified approach - ideally we'd maintain the original item data
				if rowStruct, ok := selectedItem.(struct{ ID string }); ok {
					topic := api.Topic{
						NumPartitions:     1,
						ReplicationFactor: 1,
						ReplicaAssignment: make(map[int32][]int32),
						ConfigEntries:     make(map[string]*string),
					}
					tp := NewTopicPage(m.dataSource, rowStruct.ID, topic)
					m.topicPage = &tp
					cmds = append(cmds, m.topicPage.Init())
				}
			} else {
				// Fallback to main page if no topic selected
				m.currentPage = mainPage
			}
		}
		// Initialize detail page if needed
		if m.currentPage == detailPage && m.detailPage == nil {
			// Check if we have a selected message in the topic page
			if m.topicPage != nil && m.topicPage.selectedMessage != nil {
				detailPage := NewDetailPage(m.topicPage.topicName, *m.topicPage.selectedMessage)
				m.detailPage = &detailPage
			}
		}
		// Initialize resource detail page if needed
		if m.currentPage == resourceDetailPage && m.resourceDetailPage == nil {
			// Get selected resource from main page table
			if selectedItem := m.mainPage.getSelectedResourceItem(); selectedItem != nil {
				if rowStruct, ok := selectedItem.(struct{ ID string }); ok {
					// Create a minimal resource item that implements ResourceItem interface
					minimalResource := &minimalResourceItem{
						id: rowStruct.ID,
					}
					
					rdp := NewResourceDetailPage(minimalResource, m.mainPage.currentResource.GetType())
					m.resourceDetailPage = &rdp
					cmds = append(cmds, m.resourceDetailPage.Init())
				} else {
					// Fallback to main page if no valid resource selected
					m.currentPage = mainPage
				}
			} else {
				// Fallback to main page if no resource selected
				m.currentPage = mainPage
			}
		}
		// Clean up pages when leaving them
		if m.currentPage != detailPage && m.detailPage != nil {
			m.detailPage = nil
		}
		if m.currentPage != resourceDetailPage && m.resourceDetailPage != nil {
			m.resourceDetailPage = nil
		}
		return m, nil
	}

	// Handle updates for sub-components
	switch m.currentPage {
	case mainPage:
		var cmd tea.Cmd
		mainModel, cmd := m.mainPage.Update(msg)
		if updatedMainPage, ok := mainModel.(*MainPageModel); ok {
			m.mainPage = updatedMainPage
		}
		cmds = append(cmds, cmd)

	case topicPage:
		if m.topicPage != nil {
			var cmd tea.Cmd
			topicModel, cmd := m.topicPage.Update(msg)
			if updatedTopicPage, ok := topicModel.(*TopicPageModel); ok {
				m.topicPage = updatedTopicPage
			}
			cmds = append(cmds, cmd)

			// Check if we need to navigate to detail page
			if pageMsg, ok := msg.(pageChangeMsg); ok && page(pageMsg) == detailPage {
				// Initialize detail page with selected message
				if m.topicPage.selectedMessage != nil {
					detailPageModel := NewDetailPage(m.topicPage.topicName, *m.topicPage.selectedMessage)
					m.detailPage = &detailPageModel
					m.currentPage = detailPage
				}
			}
		}

	case detailPage:
		if m.detailPage != nil {
			var cmd tea.Cmd
			detailModel, cmd := m.detailPage.Update(msg)
			if updatedDetailPage, ok := detailModel.(*DetailPageModel); ok {
				m.detailPage = updatedDetailPage
			}
			cmds = append(cmds, cmd)
		}
		
	case resourceDetailPage:
		if m.resourceDetailPage != nil {
			var cmd tea.Cmd
			resourceDetailModel, cmd := m.resourceDetailPage.Update(msg)
			if updatedResourceDetailPage, ok := resourceDetailModel.(*ResourceDetailPageModel); ok {
				m.resourceDetailPage = updatedResourceDetailPage
			}
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	switch m.currentPage {
	case mainPage:
		return m.mainPage.View()
	case topicPage:
		if m.topicPage != nil {
			return m.topicPage.View()
		}
		return "Topic page not initialized"
	case detailPage:
		if m.detailPage != nil {
			return m.detailPage.View()
		}
		return "Detail page not initialized"
	case resourceDetailPage:
		if m.resourceDetailPage != nil {
			return m.resourceDetailPage.View()
		}
		return "Resource detail page not initialized"
	default:
		return "Unknown page"
	}
}
