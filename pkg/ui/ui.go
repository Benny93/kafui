package ui

import (
	"github.com/Benny93/kafui/pkg/api"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

)

type page int

const (
	mainPage page = iota
	topicPage
	detailPage
)

// Model represents the main application state
type Model struct {
	dataSource   api.KafkaDataSource
	currentPage  page
	mainPage     *MainPageModel
	topicPage    *TopicPageModel
	width        int
	height       int
}// Custom key mappings
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
			if m.currentPage > mainPage {
				m.currentPage--
				// Clean up topic page when leaving
				if m.currentPage != topicPage && m.topicPage != nil {
					m.topicPage = nil
				}
			}
			return m, nil
		}
		
	case pageChangeMsg:
		m.currentPage = page(msg)
		// Initialize topic page if needed
		if m.currentPage == topicPage && m.topicPage == nil {
			// Get selected topic from main page
			if m.mainPage.topicList.SelectedItem() != nil {
				topic := m.mainPage.topicList.SelectedItem().(topicItem)
				tp := NewTopicPage(m.dataSource, topic.name, topic.topic)
				m.topicPage = &tp
				cmds = append(cmds, m.topicPage.Init())
			} else {
				// Fallback to main page if no topic selected
				m.currentPage = mainPage
			}
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
		
		// Check if we need to navigate to topic page
		if m.mainPage.topicList.SelectedItem() != nil {
			if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
				m.currentPage = topicPage
				topic := m.mainPage.topicList.SelectedItem().(topicItem)
				tp := NewTopicPage(m.dataSource, topic.name, topic.topic)
				m.topicPage = &tp
				cmds = append(cmds, m.topicPage.Init())
			}
		}
		
	case topicPage:
		if m.topicPage != nil {
			var cmd tea.Cmd
			topicModel, cmd := m.topicPage.Update(msg)
			if updatedTopicPage, ok := topicModel.(*TopicPageModel); ok {
				m.topicPage = updatedTopicPage
			}
			cmds = append(cmds, cmd)
		}
		
	case detailPage:
		// TODO: Implement detail page
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
		return "Detail page (to be implemented)"
	default:
		return "Unknown page"
	}
}

// Custom types for messages
type topicListMsg []list.Item
type errorMsg error
type pageChangeMsg page
