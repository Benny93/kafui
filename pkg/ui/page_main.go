package ui

import (
	"fmt"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
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
	dataSource    api.KafkaDataSource
	topicList     list.Model
	searchInput   textinput.Model
	spinner       spinner.Model
	statusMessage string
	lastUpdate    time.Time
	width         int
	height        int
	loading       bool
	searchMode    bool
	err           error
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
	topicList.SetFilteringEnabled(true)
	topicList.SetShowFilter(false)
	topicList.Styles.Title = titleStyle
	topicList.FilterInput.Prompt = "search: "
	topicList.FilterInput.PromptStyle = lipgloss.NewStyle().Foreground(highlightColor)

	// Initialize search
	ti := textinput.New()
	ti.Placeholder = "Press / to search topics..."
	ti.CharLimit = 156
	ti.Width = 30
	ti.PromptStyle = lipgloss.NewStyle().Foreground(highlightColor)

	// Initialize spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return MainPageModel{
		dataSource:    ds,
		topicList:     topicList,
		searchInput:   ti,
		spinner:       sp,
		lastUpdate:    time.Now(),
		statusMessage: "Welcome to Kafui",
		searchMode:    false,
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
		return m, nil

	case tea.KeyMsg:
		// Handle general key presses
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "/":
			if !m.searchMode {
				m.searchMode = true
				m.topicList.SetShowFilter(true)
				m.statusMessage = "Search mode: Type to filter topics"
				return m, nil
			}
		case "esc":
			if m.searchMode {
				m.searchMode = false
				m.topicList.SetShowFilter(false)
				m.topicList.ResetFilter()
				m.statusMessage = "Search cancelled"
				return m, nil
			}
		case "enter":
			if m.topicList.SelectedItem() != nil {
				// Let the main UI model handle navigation to topic page
				return m, func() tea.Msg {
					return pageChangeMsg(topicPage)
				}
			}
		}

		// If in search mode, let the list handle filtering
		if m.searchMode {
			var cmd tea.Cmd
			m.topicList, cmd = m.topicList.Update(msg)
			cmds = append(cmds, cmd)
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
	}

	// Handle text input updates
	if m.searchInput.Focused() {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)
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

	// Main content with proper styling
	content := lipgloss.JoinVertical(
		lipgloss.Left,
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

// Timer tick message for periodic updates
type timerTickMsg time.Time

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
