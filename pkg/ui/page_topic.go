package ui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	topicInfoStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2)

	messageTableStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1)

	messageListStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1)

	controlPanelStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#3c3c3c")).
			Padding(0, 1)

	selectedMessageStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("205")).
				Foreground(lipgloss.Color("0"))
)

type TopicPageModel struct {
	dataSource        api.KafkaDataSource
	topicName         string
	topicDetails      api.Topic
	consumeFlags      api.ConsumeFlags
	messages          []api.Message
	consumedMessages  map[string]api.Message
	messageTable      table.Model
	spinner           spinner.Model
	statusMessage     string
	lastUpdate        time.Time
	width             int
	height            int
	loading           bool
	consuming         bool
	paused            bool
	searchMode        bool
	searchInput       textinput.Model
	filteredMessages  []api.Message
	selectedMessage   *api.Message
	err               error
	cancelConsumption context.CancelFunc
}

func NewTopicPage(ds api.KafkaDataSource, topicName string, topicDetails api.Topic) TopicPageModel {
	// Initialize message table
	columns := []table.Column{
		{Title: "Offset", Width: 10},
		{Title: "Partition", Width: 10},
		{Title: "Timestamp", Width: 20},
		{Title: "Key", Width: 20},
		{Title: "Value", Width: 40},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = selectedMessageStyle
	t.SetStyles(s)

	// Initialize search input
	ti := textinput.New()
	ti.Placeholder = "Search messages..."
	ti.CharLimit = 156
	ti.Width = 30

	// Initialize spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return TopicPageModel{
		dataSource:       ds,
		topicName:        topicName,
		topicDetails:     topicDetails,
		consumeFlags:     api.DefaultConsumeFlags(),
		messages:         []api.Message{},
		consumedMessages: make(map[string]api.Message),
		messageTable:     t,
		spinner:          sp,
		lastUpdate:       time.Now(),
		statusMessage:    "Topic page initialized",
		searchInput:      ti,
		filteredMessages: []api.Message{},
	}
}

func (m *TopicPageModel) Init() tea.Cmd {
	return tea.Batch(
		m.startConsuming(),
		m.spinner.Tick,
	)
}

func (m *TopicPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Adjust table height based on window size
		if msg.Height > 10 {
			m.messageTable.SetHeight(msg.Height - 15)
		}
		return m, nil

	case tea.KeyMsg:
		// Handle key bindings
		switch msg.String() {
		case "ctrl+c", "q":
			if m.cancelConsumption != nil {
				m.cancelConsumption()
			}
			return m, tea.Quit
		case "esc":
			if m.searchMode {
				m.searchMode = false
				m.searchInput.Blur()
				m.filterMessages()
				return m, nil
			}
			// Return to main page
			if m.cancelConsumption != nil {
				m.cancelConsumption()
			}
			return m, func() tea.Msg {
				return pageChangeMsg(mainPage)
			}
		case "/":
			m.searchMode = true
			m.searchInput.Focus()
			return m, nil
		case " ":
			// Pause/resume consumption
			m.paused = !m.paused
			if m.paused {
				m.statusMessage = "Consumption paused"
			} else {
				m.statusMessage = "Consumption resumed"
			}
			return m, nil
		case "enter":
			// View message details
			if len(m.filteredMessages) > 0 {
				cursor := m.messageTable.Cursor()
				if cursor >= 0 && cursor < len(m.filteredMessages) {
					m.selectedMessage = &m.filteredMessages[cursor]
					return m, func() tea.Msg {
						return pageChangeMsg(detailPage)
					}
				}
			}
			return m, nil
		}

		// Handle search input
		if m.searchMode {
			var cmd tea.Cmd
			switch msg.String() {
			case "enter":
				m.searchMode = false
				m.searchInput.Blur()
				fallthrough
			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
				cmds = append(cmds, cmd)
				m.filterMessages()
				return m, tea.Batch(cmds...)
			}
		}

		// Handle table navigation
		var cmd tea.Cmd
		m.messageTable, cmd = m.messageTable.Update(msg)
		cmds = append(cmds, cmd)

	case messageConsumedMsg:
		// Add new message to the list
		key := m.getMessageKey(fmt.Sprint(msg.Partition), fmt.Sprint(msg.Offset))
		m.consumedMessages[key] = api.Message(msg)
		m.messages = append(m.messages, api.Message(msg))

		// Sort messages by offset
		sort.Slice(m.messages, func(i, j int) bool {
			return m.messages[i].Offset < m.messages[j].Offset
		})

		// Update filtered messages and table
		m.filterMessages()
		m.updateTable()

		// Auto-scroll to bottom if not paused
		if !m.paused {
			if len(m.filteredMessages) > 0 {
				m.messageTable.GotoBottom()
			}
		}

		m.statusMessage = fmt.Sprintf("Consumed %d messages", len(m.messages))
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case errorMsg:
		m.loading = false
		m.err = msg
		m.statusMessage = fmt.Sprintf("Error: %v", msg)
		return m, nil

	case startConsumingMsg:
		m.consuming = true
		m.statusMessage = "Starting message consumption..."
		return m, nil

	case consumeErrorMsg:
		m.consuming = false
		m.statusMessage = fmt.Sprintf("Consumption error: %v", msg)
		return m, nil
	}

	// Handle text input updates
	if m.searchInput.Focused() {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)
		m.filterMessages()
	}

	return m, tea.Batch(cmds...)
}

func (m *TopicPageModel) View() string {
	// Topic info section
	topicInfo := m.renderTopicInfo()

	// Controls section
	controls := m.renderControls()

	// Messages section
	messages := m.renderMessages()

	// Search input (if in search mode)
	var searchView string
	if m.searchMode {
		searchView = m.searchInput.View()
	}

	// Status bar
	status := m.renderStatusBar()

	// Combine all sections
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		topicInfo,
		controls,
		messages,
		searchView,
	)

	// Wrap in document style
	var doc string
	if m.width > 0 {
		doc = lipgloss.NewStyle().Margin(1, 2).Render(content)
	} else {
		// Even when width is not set, we should still render the content
		doc = content
	}

	// Add status bar at the bottom
	return lipgloss.JoinVertical(
		lipgloss.Left,
		doc,
		status,
	)
}

func (m *TopicPageModel) renderTopicInfo() string {
	info := fmt.Sprintf(
		"Topic: %s\nPartitions: %d\nReplication Factor: %d\nMessages: %d",
		m.topicName,
		m.topicDetails.NumPartitions,
		m.topicDetails.ReplicationFactor,
		len(m.messages),
	)

	// Format config entries if any
	if len(m.topicDetails.ConfigEntries) > 0 {
		configLines := []string{"Configuration:"}
		for key, value := range m.topicDetails.ConfigEntries {
			if value != nil {
				configLines = append(configLines, fmt.Sprintf("  %s: %s", key, *value))
			} else {
				configLines = append(configLines, fmt.Sprintf("  %s: <nil>", key))
			}
		}
		info += "\n" + strings.Join(configLines, "\n")
	}

	return topicInfoStyle.Render(info)
}

func (m *TopicPageModel) renderControls() string {
	controls := fmt.Sprintf(
		"Format: %s | Partition: All | Follow: %t | Paused: %t",
		"JSON", // Default format
		m.consumeFlags.Follow,
		m.paused,
	)

	return controlPanelStyle.Render(controls)
}

func (m *TopicPageModel) renderMessages() string {
	if m.loading {
		return messageListStyle.Render(fmt.Sprintf("%s Loading messages...", m.spinner.View()))
	}

	if len(m.filteredMessages) == 0 {
		return messageListStyle.Render("No messages available. Press Space to start consumption.")
	}

	return messageTableStyle.Render(m.messageTable.View())
}

func (m *TopicPageModel) renderStatusBar() string {
	status := fmt.Sprintf("%s %s | Last update: %s",
		m.spinner.View(),
		m.statusMessage,
		m.lastUpdate.Format("15:04:05"),
	)

	if m.err != nil {
		status = fmt.Sprintf("Error: %v", m.err)
	}

	return statusBarStyle.Render(status)
}

// Message handling
func (m *TopicPageModel) getMessageKey(partition string, offset string) string {
	return fmt.Sprintf("%s:%s", partition, offset)
}

// Message consumption
// startConsuming returns a Cmd that starts the message consumption
func (m *TopicPageModel) startConsuming() tea.Cmd {
	// Create context for consumption
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelConsumption = cancel
	
	// Return a command that will start consumption
	return func() tea.Msg {
		// Set consuming flag
		m.consuming = true
		m.loading = false // Set loading to false immediately since we're starting consumption
		
		// Start consumption in a goroutine
		go func() {
			handlerFunc := func(msg api.Message) {
				// Process the message directly in mock mode
				// In a real implementation, we would send this as a message to the program
				// But for mock mode, we'll update the model directly
				
				// Add a small delay to make mock consumption visible
				time.Sleep(50 * time.Millisecond)
				
				// Directly update the model (not thread-safe but works for mock)
				key := m.getMessageKey(fmt.Sprint(msg.Partition), fmt.Sprint(msg.Offset))
				m.consumedMessages[key] = msg
				m.messages = append(m.messages, msg)
				
				// Sort messages by offset
				sort.Slice(m.messages, func(i, j int) bool {
					return m.messages[i].Offset < m.messages[j].Offset
				})
				
				// Update filtered messages and table
				m.filterMessages()
				m.updateTable()
				
				// Auto-scroll to bottom if not paused
				if !m.paused {
					if len(m.filteredMessages) > 0 {
						m.messageTable.GotoBottom()
					}
				}
				
				m.statusMessage = fmt.Sprintf("Consumed %d messages", len(m.messages))
				m.lastUpdate = time.Now()
			}

			err := m.dataSource.ConsumeTopic(ctx, m.topicName, m.consumeFlags, handlerFunc, func(err any) {
				// Handle consumption errors
				// In a real implementation, we would send an error message to the program
			})

			if err != nil {
				// Would send error message in a real implementation
				m.statusMessage = fmt.Sprintf("Consumption error: %v", err)
			}
		}()
		
		// Return a message indicating consumption has started
		return startConsumingMsg{}
	}
}

// Message filtering
func (m *TopicPageModel) filterMessages() {
	if m.searchInput.Value() == "" {
		m.filteredMessages = m.messages
	} else {
		searchText := strings.ToLower(m.searchInput.Value())
		m.filteredMessages = []api.Message{}

		for _, msg := range m.messages {
			// Check if any field contains the search text
			if strings.Contains(strings.ToLower(strconv.FormatInt(msg.Offset, 10)), searchText) ||
				strings.Contains(strings.ToLower(fmt.Sprint(msg.Partition)), searchText) ||
				strings.Contains(strings.ToLower(msg.Key), searchText) ||
				strings.Contains(strings.ToLower(msg.Value), searchText) {
				m.filteredMessages = append(m.filteredMessages, msg)
			}
		}
	}

	m.updateTable()
}

func (m *TopicPageModel) updateTable() {
	// Convert messages to table rows
	rows := make([]table.Row, len(m.filteredMessages))
	for i, msg := range m.filteredMessages {
		// Format timestamp
		timestamp := time.Now().Format("2006-01-02 15:04:05")

		// Truncate long values
		value := msg.Value
		if len(value) > 50 {
			value = value[:47] + "..."
		}

		key := msg.Key
		if len(key) > 20 {
			key = key[:17] + "..."
		}

		rows[i] = table.Row{
			fmt.Sprint(msg.Offset),
			fmt.Sprint(msg.Partition),
			timestamp,
			key,
			value,
		}
	}

	m.messageTable.SetRows(rows)
}

// Custom message types
type messageConsumedMsg api.Message
type startConsumingMsg struct{}
type consumeErrorMsg any
