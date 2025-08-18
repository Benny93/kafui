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
	currentTopic api.Topic
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
		currentPage: mainPage, // This is the page enum value, not the MainPageModel
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
			}
			return m, nil
		}
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
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	switch m.currentPage {
	case mainPage:
		return m.mainPage.View()
	case topicPage:
		return "Topic page (to be implemented)"
	case detailPage:
		return "Detail page (to be implemented)"
	default:
		return "Unknown page"
	}
}

// loadTopics loads the topics from the data source
func (m Model) loadTopics() tea.Msg {
	topics, err := m.dataSource.GetTopics()
	if err != nil {
		return err
	}

	// Convert topics to list items
	items := make([]list.Item, len(topics))
	for name, topic := range topics {
		items = append(items, topicItem{
			name:  name,
			topic: topic,
		})
	}

	return topicListMsg(items)
}

// Custom types for messages
type topicListMsg []list.Item
type errorMsg error
