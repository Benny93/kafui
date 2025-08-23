package ui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
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
	msgChan           <-chan api.Message
	errChan           <-chan error

	// Error handling and retry logic
	retryCount       int
	maxRetries       int
	retryDelay       time.Duration
	lastError        error
	errorHistory     []error
	connectionStatus string
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

		// Initialize error handling
		maxRetries:       3,
		retryDelay:       time.Second * 2,
		errorHistory:     make([]error, 0),
		connectionStatus: "disconnected",
	}
}

func (m *TopicPageModel) Init() tea.Cmd {
	shared.DebugLog("Init page topic")

	// Set initial loading state
	m.loading = true
	cmds := []tea.Cmd{
		m.startConsuming(),
		m.spinner.Tick,
	}
	shared.DebugLog("Init returning %d commands", len(cmds))
	return tea.Batch(cmds...)
}

func (m *TopicPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Add type information to debug logging
	switch msg.(type) {
	case spinner.TickMsg:
		shared.DebugLog("Update page topic - spinner.TickMsg")
	case tea.KeyMsg:
		shared.DebugLog("Update page topic - tea.KeyMsg: %s", msg.(tea.KeyMsg).String())
	case messageConsumedMsg:
		shared.DebugLog("Update page topic - messageConsumedMsg")
	case startConsumingMsg:
		shared.DebugLog("Update page topic - startConsumingMsg")
	case continuousListenMsg:
		shared.DebugLog("Update page topic - continuousListenMsg")
	case continuousErrorListenMsg:
		shared.DebugLog("Update page topic - continuousErrorListenMsg")
	default:
		shared.DebugLog("Update page topic - %T: %v", msg, msg)
	}
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
		case "r":
			// Manual retry connection
			if !m.consuming || m.connectionStatus == "failed" {
				m.retryCount = 0 // Reset retry count for manual retry
				m.consuming = true
				m.connectionStatus = "connecting"
				m.statusMessage = "Manually retrying connection..."
				return m, m.startConsuming()
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
		message := api.Message(msg)
		shared.DebugLog("Processing messageConsumedMsg: Key=%s, Value=%s, Offset=%d, Partition=%d",
			message.Key, message.Value, message.Offset, message.Partition)

		// Handle status messages separately
		if message.Key == "__status__" {
			shared.DebugLog("Processing status message: %s", message.Value)
			switch message.Value {
			case "connecting":
				m.connectionStatus = "connecting"
				m.statusMessage = "Connecting to Kafka..."
			case "connected":
				m.connectionStatus = "connected"
				m.consuming = true
				m.loading = false
				m.retryCount = 0 // Reset retry count on successful connection
				m.statusMessage = "Successfully connected to Kafka - waiting for messages..."
			}

			// Always continue listening for more messages when we have valid channels
			if m.msgChan != nil {
				cmds = append(cmds, m.listenForMessages(m.msgChan))
			}
			return m, tea.Batch(cmds...)
		}

		// Process regular messages
		shared.DebugLog("Processing regular message - before adding to messages slice (current count: %d)", len(m.messages))
		key := m.getMessageKey(fmt.Sprint(message.Partition), fmt.Sprint(message.Offset))
		m.consumedMessages[key] = message
		m.messages = append(m.messages, message)
		shared.DebugLog("Added message to slice - new count: %d", len(m.messages))

		// Sort messages by offset
		sort.Slice(m.messages, func(i, j int) bool {
			return m.messages[i].Offset < m.messages[j].Offset
		})
		shared.DebugLog("Messages sorted by offset")

		// Update filtered messages and table
		shared.DebugLog("Calling filterMessages - current filteredMessages count: %d", len(m.filteredMessages))
		m.filterMessages()
		shared.DebugLog("After filterMessages - filteredMessages count: %d", len(m.filteredMessages))

		// Auto-scroll to bottom if not paused
		if !m.paused {
			if len(m.filteredMessages) > 0 {
				m.messageTable.GotoBottom()
				shared.DebugLog("Auto-scrolled table to bottom")
			}
		}

		m.statusMessage = fmt.Sprintf("Consumed %d messages", len(m.messages))
		shared.DebugLog("Updated status message: %s", m.statusMessage)

		// Continue listening for more messages if we have valid channels
		if m.msgChan != nil {
			cmds = append(cmds, m.listenForMessages(m.msgChan))
		}
		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		//shared.DebugLog("Spinner tick received")
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
			//shared.DebugLog("Spinner command added to cmds (type: %T)", cmd)
		} else {
			//shared.DebugLog("Spinner returned nil command, adding manual tick")
			cmds = append(cmds, m.spinner.Tick)
		}
		//shared.DebugLog("Total commands in batch: %d", len(cmds))

	case errorMsg:
		m.loading = false
		m.err = msg
		m.statusMessage = fmt.Sprintf("Error: %v", msg)
		return m, tea.Batch(cmds...)

	case startConsumingMsg:
		shared.DebugLog("Processing startConsumingMsg - setting up channels and listeners")
		// Store the cancellation function and channels
		m.cancelConsumption = msg.cancel
		m.msgChan = msg.msgChan
		m.errChan = msg.errChan
		m.consuming = true
		m.loading = false
		m.statusMessage = "Connected - listening for messages..."
		shared.DebugLog("Channels stored - msgChan: %p, errChan: %p", m.msgChan, m.errChan)

		// Start listening to channels - include accumulated commands to preserve spinner
		cmds = append(cmds,
			m.listenForMessages(msg.msgChan),
			m.listenForErrors(msg.errChan),
		)
		shared.DebugLog("Started listening commands - total commands in batch: %d", len(cmds))
		return m, tea.Batch(cmds...)

	case consumeErrorMsg:
		err := error(msg)
		m.lastError = err
		m.errorHistory = append(m.errorHistory, err)

		// Keep only last 10 errors in history
		if len(m.errorHistory) > 10 {
			m.errorHistory = m.errorHistory[1:]
		}

		// Check if we should retry
		if m.retryCount < m.maxRetries && m.consuming {
			m.retryCount++
			m.connectionStatus = "retrying"
			m.statusMessage = fmt.Sprintf("Connection error (attempt %d/%d): %v",
				m.retryCount, m.maxRetries, err)

			// Schedule retry with exponential backoff
			retryDelay := m.retryDelay * time.Duration(m.retryCount)
			cmds = append(cmds,
				m.scheduleRetry(m.retryCount, retryDelay),
				m.listenForErrors(m.errChan), // Continue listening for errors
			)
			return m, tea.Batch(cmds...)
		} else {
			// Max retries reached or not consuming
			m.consuming = false
			m.connectionStatus = "failed"
			m.statusMessage = fmt.Sprintf("Max retries reached. Last error: %v", err)

			cmds = append(cmds, func() tea.Msg {
				return maxRetriesReachedMsg{
					lastError: err,
					attempts:  m.retryCount,
				}
			})
			return m, tea.Batch(cmds...)
		}

	case retryConsumptionMsg:
		m.statusMessage = fmt.Sprintf("Retrying connection (attempt %d/%d)...",
			msg.attempt, m.maxRetries)

		// Reset channels and restart consumption
		cmds = append(cmds, m.startConsuming())
		return m, tea.Batch(cmds...)

	case connectionStatusMsg:
		m.connectionStatus = string(msg)
		if m.connectionStatus == "connected" {
			m.retryCount = 0 // Reset retry count on successful connection
			m.statusMessage = "Successfully connected to Kafka"
		}
		return m, tea.Batch(cmds...)

	case maxRetriesReachedMsg:
		m.consuming = false
		m.connectionStatus = "failed"
		m.statusMessage = fmt.Sprintf("Failed to connect after %d attempts. Last error: %v",
			msg.attempts, msg.lastError)
		return m, tea.Batch(cmds...)

	case continuousListenMsg:
		// Continue listening for messages if we're still consuming
		if m.consuming && m.msgChan != nil {
			cmds = append(cmds, m.listenForMessages(m.msgChan))
		}
		return m, tea.Batch(cmds...)

	case continuousErrorListenMsg:
		// Continue listening for errors if we're still consuming
		if m.consuming && m.errChan != nil {
			cmds = append(cmds, m.listenForErrors(m.errChan))
		}
		return m, tea.Batch(cmds...)
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
		// Check if we're actively consuming but haven't received messages yet
		if m.consuming {
			return fmt.Sprintf("%s Waiting for messages...", m.spinner.View())
		}
		// Not consuming at all
		return "No messages available. Press 'r' to start consumption or check connection."
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
		"r       Retry connection",
		"Esc     Exit search",
		"q/Esc   Back to topics",
	}

	return lipgloss.JoinVertical(lipgloss.Left, shortcuts...)
}

func (m *TopicPageModel) renderFooter() string {
	// Left side: Selection and connection information
	selected := "None"
	if len(m.filteredMessages) > 0 {
		cursor := m.messageTable.Cursor()
		if cursor >= 0 && cursor < len(m.filteredMessages) {
			selected = fmt.Sprintf("Offset: %d", m.filteredMessages[cursor].Offset)
		}
	}

	// Connection status indicator
	var statusIndicator string
	switch m.connectionStatus {
	case "connected":
		statusIndicator = "🟢"
	case "connecting":
		statusIndicator = "🟡"
	case "retrying":
		statusIndicator = "🟠"
	case "failed":
		statusIndicator = "🔴"
	default:
		statusIndicator = "⚪"
	}

	leftInfo := fmt.Sprintf("%s %s  •  Selected: %s  •  %d messages",
		statusIndicator, m.connectionStatus, selected, len(m.messages))

	// Right side: Status and error information
	var rightInfo string
	if m.lastError != nil && m.connectionStatus != "connected" {
		errorType := m.categorizeError(m.lastError)
		rightInfo = fmt.Sprintf("%s [%s] %s  •  %s",
			m.spinner.View(),
			errorType,
			m.statusMessage,
			m.lastUpdate.Format("15:04:05"),
		)
	} else {
		rightInfo = fmt.Sprintf("%s %s  •  %s",
			m.spinner.View(),
			m.statusMessage,
			m.lastUpdate.Format("15:04:05"),
		)
	}

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
// startConsuming returns a Cmd that starts the message consumption with proper channel pattern
func (m *TopicPageModel) startConsuming() tea.Cmd {
	return func() tea.Msg {
		shared.DebugLog("Starting consumption for topic: %s", m.topicName)
		// Create context for consumption
		ctx, cancel := context.WithCancel(context.Background())

		// Create channels for thread-safe communication
		msgChan := make(chan api.Message, 100) // Buffered channel for messages
		errChan := make(chan error, 1)         // Channel for errors
		shared.DebugLog("Created channels for consumption - msgChan buffer: 100, errChan buffer: 1")

		// Start consumption in a goroutine
		go func() {
			defer close(msgChan)
			defer close(errChan)
			shared.DebugLog("Consumption goroutine started - channels will be closed when goroutine exits")

			// Send connection status update
			select {
			case msgChan <- api.Message{
				Key:   "__status__",
				Value: "connecting",
			}:
				shared.DebugLog("Sent 'connecting' status message")
			case <-ctx.Done():
				shared.DebugLog("Context cancelled while sending connecting status")
				return
			}

			handlerFunc := func(msg api.Message) {
				shared.DebugLog("Handler received message: Key=%s, Value=%s, Offset=%d, Partition=%d",
					msg.Key, msg.Value, msg.Offset, msg.Partition)

				// Check if context is cancelled first
				select {
				case <-ctx.Done():
					shared.DebugLog("Context cancelled, not sending message to channel")
					return
				default:
				}

				// Try to send message with timeout and context check
				select {
				case msgChan <- msg:
					shared.DebugLog("Message sent to channel successfully")
				case <-ctx.Done():
					shared.DebugLog("Context cancelled while sending message to channel")
					return
				case <-time.After(100 * time.Millisecond):
					shared.DebugLog("Timeout sending message to channel, context may be cancelled")
					return
				}
			}

			errorHandler := func(err any) {
				shared.DebugLog("Error handler called with: %v (type: %T)", err, err)
				if e, ok := err.(error); ok {
					// Check if context is cancelled first
					select {
					case <-ctx.Done():
						shared.DebugLog("Context cancelled, not sending error to channel")
						return
					default:
					}

					// Categorize and enhance error information
					errorType := m.categorizeError(e)
					enhancedErr := fmt.Errorf("[%s] %w", errorType, e)
					shared.DebugLog("Enhanced error: %v", enhancedErr)

					// Try to send error with timeout and context check
					select {
					case errChan <- enhancedErr:
						shared.DebugLog("Error sent to error channel successfully")
					case <-ctx.Done():
						shared.DebugLog("Context cancelled while sending error to channel")
						return
					case <-time.After(100 * time.Millisecond):
						shared.DebugLog("Timeout sending error to channel, context may be cancelled")
						return
					}
				}
			}

			// Attempt to start consumption
			shared.DebugLog("Calling ConsumeTopic with topic=%s, flags=%+v", m.topicName, m.consumeFlags)
			err := m.dataSource.ConsumeTopic(ctx, m.topicName, m.consumeFlags, handlerFunc, errorHandler)
			shared.DebugLog("ConsumeTopic returned: %v", err)

			// Send connected status when consumption starts successfully
			if err == nil {
				// Check if context is cancelled first
				select {
				case <-ctx.Done():
					shared.DebugLog("Context cancelled, not sending connected status")
					return
				default:
				}

				// Try to send connected status with timeout
				select {
				case msgChan <- api.Message{
					Key:   "__status__",
					Value: "connected",
				}:
					shared.DebugLog("Sent 'connected' status message")
				case <-ctx.Done():
					shared.DebugLog("Context cancelled while sending connected status")
					return
				case <-time.After(100 * time.Millisecond):
					shared.DebugLog("Timeout sending connected status, context may be cancelled")
					return
				}

				// Keep the goroutine alive to maintain channels until context is cancelled
				shared.DebugLog("Consumption started successfully, keeping goroutine alive until context cancellation")
				<-ctx.Done()
				shared.DebugLog("Context cancelled, consumption goroutine exiting")
			} else {
				shared.DebugLog("ConsumeTopic failed, sending error: %v", err)

				// Check if context is cancelled first
				select {
				case <-ctx.Done():
					shared.DebugLog("Context cancelled, not sending startup error")
					return
				default:
				}

				// Categorize and enhance error information
				errorType := m.categorizeError(err)
				enhancedErr := fmt.Errorf("[%s] %w", errorType, err)

				// Try to send startup error with timeout
				select {
				case errChan <- enhancedErr:
					shared.DebugLog("Startup error sent to error channel")
				case <-ctx.Done():
					shared.DebugLog("Context cancelled while sending startup error")
				case <-time.After(100 * time.Millisecond):
					shared.DebugLog("Timeout sending startup error, context may be cancelled")
				}
			}
		}()

		shared.DebugLog("Returning startConsumingMsg with channels")
		// Return consumption setup with channels
		return startConsumingMsg{
			ctx:     ctx,
			cancel:  cancel,
			msgChan: msgChan,
			errChan: errChan,
		}
	}
}

// Message filtering
func (m *TopicPageModel) filterMessages() {
	shared.DebugLog("filterMessages called - input messages: %d, search value: '%s'", len(m.messages), m.searchInput.Value())
	if m.searchInput.Value() == "" {
		m.filteredMessages = m.messages
		shared.DebugLog("No search filter - using all messages: %d", len(m.filteredMessages))
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
		shared.DebugLog("Search filtered messages: %d", len(m.filteredMessages))
	}

	shared.DebugLog("filterMessages completed - calling updateTable")
	m.updateTable()
}

func (m *TopicPageModel) updateTable() {
	shared.DebugLog("updateTable called with %d filtered messages", len(m.filteredMessages))
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

		row := table.Row{
			fmt.Sprint(msg.Offset),
			fmt.Sprint(msg.Partition),
			timestamp,
			key,
			value,
		}
		rows[i] = row
		shared.DebugLog("Created row %d: [%s, %s, %s, %s, %s]", i, row[0], row[1], row[2], row[3], row[4])
	}

	shared.DebugLog("Setting %d rows to messageTable", len(rows))
	m.messageTable.SetRows(rows)
	shared.DebugLog("messageTable now has %d rows", len(m.messageTable.Rows()))
}

// Channel listening commands for thread-safe communication
func (m *TopicPageModel) listenForMessages(msgChan <-chan api.Message) tea.Cmd {
	return func() tea.Msg {
		shared.DebugLog("Listening for messages on channel")
		// Use select with a timeout to prevent blocking indefinitely
		select {
		case msg, ok := <-msgChan:
			if !ok {
				shared.DebugLog("Message channel closed")
				// Channel closed, consumption ended
				return consumeErrorMsg(fmt.Errorf("message channel closed"))
			}
			shared.DebugLog("Received message from channel: Offset=%d, Partition=%d, Key=%s", msg.Offset, msg.Partition, msg.Key)
			return messageConsumedMsg(msg)
		case <-time.After(100 * time.Millisecond):
			shared.DebugLog("Timeout waiting for message - continuing to listen")
			// Timeout - continue listening but allow UI to remain responsive
			return continuousListenMsg{}
		}
	}
}

func (m *TopicPageModel) listenForErrors(errChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		// Use select with a timeout to prevent blocking indefinitely
		select {
		case err, ok := <-errChan:
			if !ok {
				// Channel closed, no more errors
				return nil
			}
			return consumeErrorMsg(err)
		case <-time.After(100 * time.Millisecond):
			// Timeout - continue listening but allow UI to remain responsive
			return continuousErrorListenMsg{}
		}
	}
}

// Retry scheduling with exponential backoff
func (m *TopicPageModel) scheduleRetry(attempt int, delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return retryConsumptionMsg{
			attempt: attempt,
			delay:   delay,
		}
	})
}

// Enhanced error categorization
func (m *TopicPageModel) categorizeError(err error) string {
	if err == nil {
		return "none"
	}

	errStr := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errStr, "connection"):
		return "connection"
	case strings.Contains(errStr, "timeout"):
		return "timeout"
	case strings.Contains(errStr, "authentication"):
		return "auth"
	case strings.Contains(errStr, "authorization"):
		return "permission"
	case strings.Contains(errStr, "topic"):
		return "topic"
	case strings.Contains(errStr, "partition"):
		return "partition"
	default:
		return "unknown"
	}
}

// Custom message types
type messageConsumedMsg api.Message
type startConsumingMsg struct {
	ctx     context.Context
	cancel  context.CancelFunc
	msgChan <-chan api.Message
	errChan <-chan error
}
type consumeErrorMsg error
type retryConsumptionMsg struct {
	attempt int
	delay   time.Duration
}
type connectionStatusMsg string
type maxRetriesReachedMsg struct {
	lastError error
	attempts  int
}
type continuousListenMsg struct{}
type continuousErrorListenMsg struct{}
