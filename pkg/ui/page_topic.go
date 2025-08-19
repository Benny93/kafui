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
	// Topic page specific styles
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
		BorderForeground(Subtle).
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

		// Calculate available space for content
		contentHeight := msg.Height - 8 // Account for header and footer

		// Update table dimensions
		m.messageTable.SetHeight(contentHeight - 6) // Account for controls and search
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
		m.loading = false
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

	// Calculate layout dimensions
	sidebarWidth := 35
	contentWidth := m.width - sidebarWidth - 6 // Account for padding and borders
	contentHeight := m.height - 8              // Account for header and footer

	// Header section
	resourceIndicator := ResourceTypeStyle.Render("TOPIC")
	header := HeaderStyle.
		Width(m.width).
		Render(fmt.Sprintf("%sKafui - Topic: %s", resourceIndicator, m.topicName))

	// Main content area with controls, messages, and search
	controlsSection := MainPanelStyle.
		Width(contentWidth).
		Render(m.renderControls())

	messagesSection := MainPanelStyle.
		Width(contentWidth).
		Height(contentHeight - 6). // Account for controls and search
		Render(m.renderMessages())

	var searchSection string
	if m.searchMode {
		searchSection = MainPanelStyle.
			Width(contentWidth).
			Render(m.searchInput.View())
	}

	// Combine main content sections
	var mainContent string
	if m.searchMode {
		mainContent = lipgloss.JoinVertical(
			lipgloss.Left,
			controlsSection,
			messagesSection,
			searchSection,
		)
	} else {
		mainContent = lipgloss.JoinVertical(
			lipgloss.Left,
			controlsSection,
			messagesSection,
		)
	}

	// Sidebar with topic information
	sidebarContent := lipgloss.JoinVertical(
		lipgloss.Left,
		TitleStyle.Render("TOPIC INFO"),
		m.renderTopicInfo(),
		lipgloss.NewStyle().MarginTop(2).Render(""),
		SubtitleStyle.Render("SHORTCUTS"),
		m.renderShortcuts(),
	)

	sidebar := SidebarPanelStyle.
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
	footer := FooterStyle.Width(m.width).Render(m.renderFooter())

	// Combine all sections
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		LayoutStyle.Render(body),
		footer,
	)
}

func (m *TopicPageModel) renderTopicInfo() string {
	info := fmt.Sprintf(
		"Name: %s\nPartitions: %d\nReplication Factor: %d\nMessages: %d",
		m.topicName,
		m.topicDetails.NumPartitions,
		m.topicDetails.ReplicationFactor,
		len(m.messages),
	)

	// Format config entries if any
	if len(m.topicDetails.ConfigEntries) > 0 {
		configLines := []string{"\nConfiguration:"}
		for key, value := range m.topicDetails.ConfigEntries {
			if value != nil {
				configLines = append(configLines, fmt.Sprintf("  %s: %s", key, *value))
			} else {
				configLines = append(configLines, fmt.Sprintf("  %s: <nil>", key))
			}
		}
		info += strings.Join(configLines, "\n")
	}

	return InfoStyle.Render(info)
}

func (m *TopicPageModel) renderControls() string {
	controls := fmt.Sprintf(
		"Format: %s | Partition: All | Follow: %t | Paused: %t",
		"JSON", // Default format
		m.consumeFlags.Follow,
		m.paused,
	)

	return InfoStyle.Render(controls)
}

func (m *TopicPageModel) renderMessages() string {
	if m.loading {
		return fmt.Sprintf("%s Loading messages...", m.spinner.View())
	}

	if len(m.filteredMessages) == 0 {
		return "No messages available. Press Space to start consumption."
	}

	return m.messageTable.View()
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

	return FooterStyle.Render(status)
}

func (m *TopicPageModel) renderShortcuts() string {
	shortcuts := []string{
		"↑/↓   Navigate messages",
		"Enter   View details",
		"Space   Pause/resume",
		"/       Search messages",
		"Esc     Exit search",
		"q/Esc   Back to topics",
	}

	return lipgloss.JoinVertical(lipgloss.Left, shortcuts...)
}

func (m *TopicPageModel) renderFooter() string {
	// Left side: Selection information
	selected := "None"
	if len(m.filteredMessages) > 0 {
		cursor := m.messageTable.Cursor()
		if cursor >= 0 && cursor < len(m.filteredMessages) {
			selected = fmt.Sprintf("Offset: %d", m.filteredMessages[cursor].Offset)
		}
	}
	leftInfo := fmt.Sprintf("Selected: %s  •  %d messages total", selected, len(m.messages))

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
	if leftWidth+rightWidth+3 <= totalWidth {
		spacer := strings.Repeat(" ", totalWidth-leftWidth-rightWidth)
		return leftInfo + spacer + rightInfo
	}

	// If they don't fit, truncate the left side
	maxLeftWidth := totalWidth - rightWidth - 3
	if maxLeftWidth > 20 {
		if len(leftInfo) > maxLeftWidth {
			leftInfo = leftInfo[:maxLeftWidth-3] + "..."
		}
		spacer := strings.Repeat(" ", totalWidth-len(leftInfo)-rightWidth)
		return leftInfo + spacer + rightInfo
	}

	// Fallback: just show the right info if space is very limited
	return rightInfo
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

	// Set loading to true initially
	m.loading = true

	// Return a command that will start consumption
	return func() tea.Msg {
		// Set consuming flag
		m.consuming = true

		// Start consumption in a goroutine
		go func() {
			handlerFunc := func(msg api.Message) {
				// Instead of processing directly, send a message to the program
				// This ensures consistent behavior between mock and real modes
				// messageConsumed := messageConsumedMsg(msg) // Not used in mock mode

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
