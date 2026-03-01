package topic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/Benny93/kafui/pkg/ui/template/ui/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Maximum number of messages to display in the table
const MaxDisplayedMessages = 20

// Model represents the topic page state (original business logic model)
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

	// Table dimension tracking
	lastTableWidth  int
	lastTableHeight int
}

// NewModel creates a new topic page model (original business logic)
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
		table.WithHeight(MaxDisplayedMessages),
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

// Business logic methods for the original Model

// Init implements the Page interface for the original model
func (m *Model) Init() tea.Cmd {
	// Set initial loading state
	m.loading = true
	cmds := []tea.Cmd{
		m.consumption.StartConsuming(),
		m.spinner.Tick,
	}
	return tea.Batch(cmds...)
}

// Update implements the Page interface for the original model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.handlers.Handle(m, msg)
}

// View implements the Page interface for the original model
func (m *Model) View() string {
	return m.view.Render(m)
}

// SetDimensions implements the Page interface for the original model
func (m *Model) SetDimensions(width, height int) {
	m.dimensions = core.Dimensions{Width: width, Height: height}

	// Update table dimensions and column widths
	m.updateTableDimensions(width, height)

	m.view.SetDimensions(width, height)
}

// updateTableDimensions updates the table dimensions and column widths based on available space
func (m *Model) updateTableDimensions(width, height int) {
	// Only update if dimensions have changed
	if m.lastTableWidth == width && m.lastTableHeight == height {
		return
	}
	m.lastTableWidth = width
	m.lastTableHeight = height

	// Calculate available space for content
	// Height passed is already inner content height, account for:
	// - Search bar (when active): ~3 lines
	// - Table header: 1 line
	// - Table borders: 2 lines
	// - Bottom padding/controls: 2 lines
	reservedLines := 6
	if m.searchMode {
		reservedLines += 3
	}

	// Update table height (number of visible rows)
	tableHeight := height - reservedLines
	if tableHeight < 5 {
		tableHeight = 5 // Minimum visible rows
	}
	if tableHeight > MaxDisplayedMessages {
		tableHeight = MaxDisplayedMessages // Maximum visible rows
	}
	m.messageTable.SetHeight(tableHeight)

	// Calculate column widths based on available width
	// Account for content padding (4 chars) and borders
	availableWidth := width - 8
	if availableWidth < 60 {
		availableWidth = 60 // Minimum width for all columns
	}

	// Define minimum column widths
	const (
		minOffsetWidth    = 8
		minPartitionWidth = 8
		minTimestampWidth = 15
		minKeyWidth       = 10
		minValueWidth     = 15
	)

	// Calculate total minimum width required
	minTotalWidth := minOffsetWidth + minPartitionWidth + minTimestampWidth + minKeyWidth + minValueWidth

	// Ensure available width is at least the minimum total
	if availableWidth < minTotalWidth {
		availableWidth = minTotalWidth
	}

	// Calculate remaining width after allocating minimums
	remainingWidth := availableWidth - minTotalWidth

	// Distribute remaining width proportionally (10:10:20:20:40 = 1:1:2:2:4)
	// Total ratio = 10
	offsetWidth := minOffsetWidth + remainingWidth*10/100
	partitionWidth := minPartitionWidth + remainingWidth*10/100
	timestampWidth := minTimestampWidth + remainingWidth*20/100
	keyWidth := minKeyWidth + remainingWidth*20/100
	// Value gets the remainder to ensure exact fit
	valueWidth := availableWidth - offsetWidth - partitionWidth - timestampWidth - keyWidth

	// Update column widths
	columns := []table.Column{
		{Title: "Offset", Width: offsetWidth},
		{Title: "Partition", Width: partitionWidth},
		{Title: "Timestamp", Width: timestampWidth},
		{Title: "Key", Width: keyWidth},
		{Title: "Value", Width: valueWidth},
	}
	m.messageTable.SetColumns(columns)

	// Update table width
	m.messageTable.SetWidth(availableWidth)
}

// GetID implements the Page interface for the original model
func (m *Model) GetID() string {
	return "topic"
}

// GetTitle implements the Page interface for the original model
func (m *Model) GetTitle() string {
	if m.topicName != "" {
		return fmt.Sprintf("Topic: %s", m.topicName)
	}
	return "Topic"
}

// GetHelp implements the Page interface for the original model
func (m *Model) GetHelp() []key.Binding {
	if m.keys != nil {
		return m.keys.GetKeyBindings()
	}
	return []key.Binding{}
}

// HandleNavigation implements the Page interface for the original model
func (m *Model) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
	// Handle page-specific navigation
	return m, nil
}

// OnFocus implements the Page interface for the original model
func (m *Model) OnFocus() tea.Cmd {
	// Handle focus gain - restart consumption if it was paused
	if m.paused {
		m.TogglePause()
	}
	return nil
}

// OnBlur implements the Page interface for the original model
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
		m.selectedMessageSchema = nil
		return
	}

	m.selectedMessageSchema = schemaInfo
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

	// Get current column widths for proper truncation
	columns := m.messageTable.Columns()
	keyMaxLen := 18
	valueMaxLen := 38
	for _, col := range columns {
		if col.Title == "Key" {
			keyMaxLen = col.Width - 2 // Account for padding
		} else if col.Title == "Value" {
			valueMaxLen = col.Width - 2 // Account for padding
		}
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
			key = msg.Key
			if searchQuery != "" {
				key = highlightMatchingText(key, searchQuery)
			}
			// Truncate AFTER highlighting to ensure it fits
			key = truncateString(key, keyMaxLen)
		}

		value := "<null>"
		if msg.Value != "" {
			value = msg.Value
			if searchQuery != "" {
				value = highlightMatchingText(value, searchQuery)
			}
			// Truncate AFTER highlighting to ensure it fits
			value = truncateString(value, valueMaxLen)
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

		// Keep only the last MaxDisplayedMessages to prevent table from growing indefinitely
		if len(m.messages) > MaxDisplayedMessages {
			// Remove the oldest message from the display slice
			m.messages = m.messages[1:]
		}

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

// truncateString truncates a string to fit within maxLen visual characters
// It properly handles ANSI escape codes and multi-byte characters
func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	// Use the styles utility which properly handles ANSI codes
	return styles.TruncateWithEllipsis(s, maxLen)
}

// View handles rendering for the topic page (minimal implementation for compatibility)
type View struct {
	dimensions core.Dimensions
}

// NewView creates a new View instance
func NewView() *View {
	return &View{}
}

// Render renders the topic page view (minimal implementation)
func (v *View) Render(model *Model) string {
	// For the template-based topic page, this is not used
	// The rendering is handled by the template providers
	return "Topic page (template-based rendering)"
}

// SetDimensions updates the view dimensions
func (v *View) SetDimensions(width, height int) {
	v.dimensions = core.Dimensions{Width: width, Height: height}
}

// TopicPageModel wraps the ReusableApp with topic-specific providers
type TopicPageModel struct {
	// Original topic model for business logic
	topicModel *Model

	// Template system
	reusableApp     *templateui.ReusableApp
	contentProvider *TopicContentProvider
}

// NewTopicPageModel creates a new topic page model using the template system
func NewTopicPageModel(dataSource api.KafkaDataSource, topicName string, topicDetails api.Topic) *TopicPageModel {
	// Create the original topic model for business logic
	topicModel := NewModel(dataSource, topicName, topicDetails)

	// Create topic-specific providers
	contentProvider := NewTopicContentProvider(topicModel)
	headerProvider := NewTopicHeaderDataProvider(topicModel)

	// Create sidebar sections
	sidebarSections := []providers.SidebarSection{
		NewTopicInfoSection(topicModel),
		NewMessageInfoSection(topicModel),
		NewConsumptionControlSection(topicModel),
		NewTopicShortcutsSection(topicModel),
	}

	// Create app configuration using template providers
	config := &providers.AppConfig{
		ContentProvider:             contentProvider,
		HeaderDataProvider:          headerProvider,
		SidebarSections:             sidebarSections,
		ShowSidebarByDefault:        true,
		CompactModeWidthBreakpoint:  120,
		CompactModeHeightBreakpoint: 30,
	}

	// Create the reusable app with our topic providers
	reusableApp := templateui.NewReusableApp(config)

	return &TopicPageModel{
		topicModel:      topicModel,
		reusableApp:     reusableApp,
		contentProvider: contentProvider,
	}
}

// Init implements the Page interface
func (t *TopicPageModel) Init() tea.Cmd {
	// Initialize the reusable app (which will initialize providers)
	return t.reusableApp.Init()
}

// Update implements the Page interface
func (t *TopicPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Update the reusable app (which delegates to content provider)
	updatedApp, cmd := t.reusableApp.Update(msg)
	if updatedReusableApp, ok := updatedApp.(*templateui.ReusableApp); ok {
		t.reusableApp = updatedReusableApp
	}
	return t, cmd
}

// View implements the Page interface
func (t *TopicPageModel) View() string {
	return t.reusableApp.View()
}

// SetDimensions implements the Page interface
func (t *TopicPageModel) SetDimensions(width, height int) {
	// Update both models
	t.topicModel.SetDimensions(width, height)
	t.reusableApp.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

// GetID implements the Page interface
func (t *TopicPageModel) GetID() string {
	if t.topicModel != nil && t.topicModel.topicName != "" {
		return "topic:" + t.topicModel.topicName
	}
	return "topic"
}

// GetTitle implements the Page interface
func (t *TopicPageModel) GetTitle() string {
	return t.topicModel.GetTitle()
}

// GetHelp implements the Page interface
func (t *TopicPageModel) GetHelp() []key.Binding {
	return t.topicModel.GetHelp()
}

// HandleNavigation implements the Page interface
func (t *TopicPageModel) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
	// Handle topic-specific navigation
	switch msg := msg.(type) {
	case NavigateToMessageDetailMsg:
		// This would typically create and return a message detail page
		// For now, just return self
		_ = msg
		return t, nil
	}
	return t, nil
}

// OnFocus implements the Page interface
func (t *TopicPageModel) OnFocus() tea.Cmd {
	return t.topicModel.OnFocus()
}

// OnBlur implements the Page interface
func (t *TopicPageModel) OnBlur() tea.Cmd {
	return t.topicModel.OnBlur()
}

// GetTopicName returns the current topic name
func (t *TopicPageModel) GetTopicName() string {
	return t.topicModel.GetTopicName()
}

// GetSelectedMessage returns the currently selected message
func (t *TopicPageModel) GetSelectedMessage() *api.Message {
	return t.topicModel.GetSelectedMessage()
}

// Navigation message for message detail selection
type NavigateToMessageDetailMsg struct {
	Message api.Message
}
