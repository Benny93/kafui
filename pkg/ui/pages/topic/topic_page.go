package topic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the topic page state
type Model struct {
	// Data
	dataSource   api.KafkaDataSource
	topicName    string
	topicDetails api.Topic

	// Message data
	messages              []api.Message
	consumedMessages      map[string]api.Message
	filteredMessages      []api.Message
	selectedMessage       *api.Message
	selectedMessageSchema *api.MessageSchemaInfo

	// State
	dimensions    core.Dimensions
	loading       bool
	consuming     bool
	paused        bool
	searchMode    bool
	error         error
	lastUpdate    time.Time
	statusMessage string

	// Consumption configuration
	consumeFlags api.ConsumeFlags

	// UI Components
	messageTable table.Model
	spinner      spinner.Model
	searchInput  textinput.Model

	// Components
	handlers    *Handlers
	keys        *Keys
	view        *View
	consumption *ConsumptionController

	// Consumption control
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

// NewModel creates a new topic page model
func NewModel(dataSource api.KafkaDataSource, topicName string, topicDetails api.Topic) *Model {
	// Initialize message table
	columns := []table.Column{
		{Title: "Offset", Width: 10},
		{Title: "Partition", Width: 10},
		{Title: "Timestamp", Width: 20},
		{Title: "Key", Width: 20},
		{Title: "Value", Width: 40},
	}

	messageTable := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Set table styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = lipgloss.NewStyle().
		Background(lipgloss.Color("205")).
		Foreground(lipgloss.Color("0"))
	messageTable.SetStyles(s)

	// Initialize search input
	searchInput := textinput.New()
	searchInput.Placeholder = "Search messages..."
	searchInput.CharLimit = 156
	searchInput.Width = 30

	// Initialize spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := &Model{
		dataSource:       dataSource,
		topicName:        topicName,
		topicDetails:     topicDetails,
		consumeFlags:     api.DefaultConsumeFlags(),
		messages:         []api.Message{},
		consumedMessages: make(map[string]api.Message),
		messageTable:     messageTable,
		spinner:          sp,
		lastUpdate:       time.Now(),
		statusMessage:    "Topic page initialized",
		searchInput:      searchInput,
		filteredMessages: []api.Message{},

		// Initialize error handling
		maxRetries:       3,
		retryDelay:       time.Second * 2,
		errorHistory:     make([]error, 0),
		connectionStatus: "disconnected",
	}

	// Initialize components with dependencies
	m.handlers = NewHandlers(m)
	m.keys = NewKeys()
	m.view = NewView()
	m.consumption = NewConsumptionController(m)

	return m
}

// Init implements the Page interface
func (m *Model) Init() tea.Cmd {
	shared.DebugLog("Init page topic")

	// Set initial loading state
	m.loading = true
	cmds := []tea.Cmd{
		m.consumption.StartConsuming(),
		m.spinner.Tick,
	}
	shared.DebugLog("Init returning %d commands", len(cmds))
	return tea.Batch(cmds...)
}

// Update implements the Page interface
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.handlers.Handle(m, msg)
}

// View implements the Page interface
func (m *Model) View() string {
	return m.view.Render(m)
}

// SetDimensions implements the Page interface
func (m *Model) SetDimensions(width, height int) {
	m.dimensions = core.Dimensions{Width: width, Height: height}

	// Calculate available space for content
	contentHeight := height - 8 // Account for header and footer

	// Update table dimensions
	m.messageTable.SetHeight(contentHeight - 6) // Account for controls and search

	m.view.SetDimensions(width, height)
}

// GetID implements the Page interface
func (m *Model) GetID() string {
	return "topic"
}

// GetTitle implements the Page interface
func (m *Model) GetTitle() string {
	if m.topicName != "" {
		return fmt.Sprintf("Topic: %s", m.topicName)
	}
	return "Topic"
}

// GetHelp implements the Page interface
func (m *Model) GetHelp() []key.Binding {
	if m.keys != nil {
		return m.keys.GetKeyBindings()
	}
	return []key.Binding{}
}

// HandleNavigation implements the Page interface
func (m *Model) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
	// Handle page-specific navigation
	return m, nil
}

// OnFocus implements the Page interface
func (m *Model) OnFocus() tea.Cmd {
	// Handle focus gain - restart consumption if it was paused
	if m.paused {
		m.TogglePause()
	}
	return nil
}

// OnBlur implements the Page interface
func (m *Model) OnBlur() tea.Cmd {
	// Handle focus loss - pause consumption when page loses focus
	if m.consuming && !m.paused {
		m.TogglePause()
	}
	return nil
}

// GetTopicName returns the current topic name
func (m *Model) GetTopicName() string {
	return m.topicName
}

// Business logic methods

// GetSelectedMessage returns the currently selected message
func (m *Model) GetSelectedMessage() *api.Message {
	if len(m.filteredMessages) == 0 {
		return nil
	}

	cursor := m.messageTable.Cursor()
	if cursor >= 0 && cursor < len(m.filteredMessages) {
		selectedMsg := &m.filteredMessages[cursor]
		// Load schema information when message is selected
		m.loadSchemaInfoForMessage(selectedMsg)
		return selectedMsg
	}

	return nil
}

// loadSchemaInfoForMessage loads schema information for a message
func (m *Model) loadSchemaInfoForMessage(msg *api.Message) {
	// Only load if schema IDs are present
	if msg.KeySchemaID == "" && msg.ValueSchemaID == "" {
		m.selectedMessageSchema = nil
		return
	}

	// Load schema information from data source
	schemaInfo, err := m.dataSource.GetMessageSchemaInfo(msg.KeySchemaID, msg.ValueSchemaID)
	if err != nil {
		shared.DebugLog("Failed to load schema info: %v", err)
		m.selectedMessageSchema = nil
		return
	}

	m.selectedMessageSchema = schemaInfo
	shared.DebugLog("Loaded schema info for message - KeySchema: %v, ValueSchema: %v",
		schemaInfo != nil && schemaInfo.KeySchema != nil,
		schemaInfo != nil && schemaInfo.ValueSchema != nil)
}

// FilterMessages filters messages based on search input
func (m *Model) FilterMessages() {
	if !m.searchMode || m.searchInput.Value() == "" {
		m.filteredMessages = m.messages
		m.updateMessageTable()
		return
	}

	query := m.searchInput.Value()
	filtered := []api.Message{}

	for _, msg := range m.messages {
		// Search in key and value
		if msg.Key != "" && contains(msg.Key, query) {
			filtered = append(filtered, msg)
			continue
		}
		if msg.Value != "" && contains(msg.Value, query) {
			filtered = append(filtered, msg)
			continue
		}
	}

	m.filteredMessages = filtered
	m.updateMessageTable()
}

// UpdateMessageTable updates the table with current filtered messages
func (m *Model) updateMessageTable() {
	rows := make([]table.Row, len(m.filteredMessages))

	searchQuery := ""
	if m.searchMode {
		searchQuery = m.searchInput.Value()
	}

	for i, msg := range m.filteredMessages {
		offset := "N/A"
		if msg.Offset >= 0 {
			offset = fmt.Sprintf("%d", msg.Offset)
		}

		partition := fmt.Sprintf("%d", msg.Partition)
		timestamp := "N/A" // API doesn't have timestamp field

		key := "<null>"
		if msg.Key != "" {
			key = truncateString(msg.Key, 18)
			if searchQuery != "" {
				key = highlightMatchingText(key, searchQuery)
			}
		}

		value := "<null>"
		if msg.Value != "" {
			value = truncateString(msg.Value, 38)
			if searchQuery != "" {
				value = highlightMatchingText(value, searchQuery)
			}
		}

		rows[i] = table.Row{offset, partition, timestamp, key, value}
	}

	m.messageTable.SetRows(rows)
}

// highlightMatchingText highlights matching parts of text with color
func highlightMatchingText(text, query string) string {
	if query == "" {
		return text
	}
	
	// Convert to lowercase for case-insensitive comparison
	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)
	
	// Find all occurrences of the query in the text
	var result strings.Builder
	start := 0
	
	for {
		index := strings.Index(lowerText[start:], lowerQuery)
		if index == -1 {
			// No more matches, add the rest of the text
			result.WriteString(text[start:])
			break
		}
		
		// Add text before the match
		actualIndex := start + index
		result.WriteString(text[start:actualIndex])
		
		// Add highlighted match (using Lip Gloss styling)
		match := text[actualIndex : actualIndex+len(query)]
		highlighted := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(match)
		result.WriteString(highlighted)
		
		// Move start position
		start = actualIndex + len(query)
	}
	
	return result.String()
}

// AddMessage adds a new message to the collection
func (m *Model) AddMessage(msg api.Message) {
	// Create unique key for deduplication using partition and offset
	key := fmt.Sprintf("%d-%d", msg.Partition, msg.Offset)

	// Only add if not already consumed
	if _, exists := m.consumedMessages[key]; !exists {
		m.messages = append(m.messages, msg)
		m.consumedMessages[key] = msg

		// Update filtered messages if needed
		m.FilterMessages()

		// Update status
		m.statusMessage = fmt.Sprintf("Consumed %d messages", len(m.messages))
		m.lastUpdate = time.Now()
	}
}

// TogglePause toggles consumption pause state
func (m *Model) TogglePause() {
	m.paused = !m.paused
	if m.paused {
		m.statusMessage = "Consumption paused"
	} else {
		m.statusMessage = "Consumption resumed"
	}
}

// SetError sets an error and updates error history
func (m *Model) SetError(err error) {
	m.error = err
	m.lastError = err
	m.errorHistory = append(m.errorHistory, err)

	// Keep only recent errors
	if len(m.errorHistory) > 10 {
		m.errorHistory = m.errorHistory[1:]
	}

	m.connectionStatus = "failed"
	m.statusMessage = fmt.Sprintf("Error: %v", err)
}

// SetConnectionStatus updates the connection status
func (m *Model) SetConnectionStatus(status string) {
	m.connectionStatus = status
	switch status {
	case "connected":
		m.statusMessage = "Connected and consuming messages"
	case "connecting":
		m.statusMessage = "Connecting to topic..."
	case "disconnected":
		m.statusMessage = "Disconnected"
	case "failed":
		m.statusMessage = "Connection failed"
	}
}

// Utility functions

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}
