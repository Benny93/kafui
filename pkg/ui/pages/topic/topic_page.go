package topic

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/Benny93/kafui/pkg/ui/template/ui/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

// Performance optimization constants
const (
	// MaxVisibleRows limits rendered rows (virtual scrolling) - default, overridden by height
	MaxVisibleRows = 50
	// UpdateThrottle prevents excessive re-renders
	UpdateThrottle = 100 * time.Millisecond
	// BatchSize for message processing
	BatchSize = 20
	// UseCustomRenderer threshold - use custom renderer for large datasets
	UseCustomRenderer = 100
)

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

	// Bubble-table configuration
	tableColumns []table.Column

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

	// === PERFORMANCE OPTIMIZATIONS ===

	// Message buffer limit (prevents unbounded growth)
	maxMessages int

	// Pagination (replaces virtual scrolling)
	pagination *PaginationModel

	// Width caching (avoids recalculation)
	widthCache     map[string]map[int]int // column -> width -> cached value
	widthCacheTime time.Time
	widthCacheMu   sync.RWMutex // Protects widthCache

	// Update throttling
	lastUpdateTime time.Time
	updateThrottle time.Duration

	// Batching
	batchSize     int
	batchInterval time.Duration
	batchCount    int

	// Mutex for thread-safe message operations
	mu sync.RWMutex

	// Render caching (avoid re-rendering same content)
	lastRenderHash uint64
	lastRenderTime time.Time
	renderCache    string
	renderCacheMu  sync.RWMutex

	// Dirty flag for render invalidation
	dirtyRender bool
}

// markRenderDirty marks the render cache as invalid
func (m *Model) markRenderDirty() {
	m.dirtyRender = true
}

// getRenderCache returns cached render if valid
func (m *Model) getRenderCache() (string, bool) {
	m.renderCacheMu.RLock()
	defer m.renderCacheMu.RUnlock()
	if !m.dirtyRender && m.renderCache != "" {
		return m.renderCache, true
	}
	return "", false
}

// setRenderCache updates the render cache
func (m *Model) setRenderCache(render string) {
	m.renderCacheMu.Lock()
	defer m.renderCacheMu.Unlock()
	m.renderCache = render
	m.dirtyRender = false
}

// NewModel creates a new topic page model (original business logic)
func NewModel(dataSource api.KafkaDataSource, topicName string, topicDetails api.Topic) *Model {
	// Define table columns using bubble-table
	const (
		colOffset    = "offset"
		colPartition = "partition"
		colTimestamp = "timestamp"
		colKey       = "key"
		colValue     = "value"
	)

	columns := []table.Column{
		table.NewColumn(colOffset, "Offset", 10),
		table.NewColumn(colPartition, "Partition", 10),
		table.NewColumn(colTimestamp, "Timestamp", 20),
		table.NewColumn(colKey, "Key", 20),
		table.NewColumn(colValue, "Value", 40),
	}

	// Initialize message table with bubble-table
	messageTable := table.New(columns).
		WithPageSize(DefaultPerPage).
		WithHighlightedRow(0).
		WithBaseStyle(
			lipgloss.NewStyle().
				BorderForeground(lipgloss.Color("240")),
		).
		HeaderStyle(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Bold(true),
		).
		HighlightStyle(
			lipgloss.NewStyle().
				Background(lipgloss.Color("205")).
				Foreground(lipgloss.Color("0")).
				Bold(true),
		).
		Focused(true).
		SortByDesc(colOffset) // Sort by newest first

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
		tableColumns:     columns,
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

		// === PERFORMANCE OPTIMIZATIONS ===
		maxMessages:    MaxMessageBuffer,
		pagination:     NewPaginationModel(),
		widthCache:     make(map[string]map[int]int),
		updateThrottle: UpdateThrottle,
		batchSize:      BatchSize,
		batchInterval:  50 * time.Millisecond,
	}

	// Initialize components with dependencies
	m.handlers = NewHandlers(m)
	m.keys = NewKeys()
	m.view = NewView()
	m.consumption = NewConsumptionController(m)

	return m
}

// sortMessages sorts messages by offset ascending (required for pagination logic)
func (m *Model) sortMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Sort messages by offset ascending
	sort.Slice(m.messages, func(i, j int) bool {
		if m.messages[i].Offset != m.messages[j].Offset {
			return m.messages[i].Offset < m.messages[j].Offset
		}
		return m.messages[i].Partition < m.messages[j].Partition
	})

	// Also sort filtered messages
	if len(m.filteredMessages) > 0 && len(m.filteredMessages) != len(m.messages) {
		sort.Slice(m.filteredMessages, func(i, j int) bool {
			if m.filteredMessages[i].Offset != m.filteredMessages[j].Offset {
				return m.filteredMessages[i].Offset < m.filteredMessages[j].Offset
			}
			return m.filteredMessages[i].Partition < m.filteredMessages[j].Partition
		})
	} else {
		// Make sure filteredMessages is updated to the sorted messages
		m.filteredMessages = m.messages
	}
}

// Business logic methods for the original Model

// Init implements the Page interface for the original model
func (m *Model) Init() tea.Cmd {
	// Set initial loading state
	m.loading = true

	// Fetch latest 60 messages (3 pages of 20)
	const fetchCount = 60
	cmds := []tea.Cmd{
		m.consumption.FetchLatestMessages(fetchCount),
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

	m.markRenderDirty()

	// Height passed is from content component, but actual usable height is less:
	// - Content component has border (2) + padding (2) = 4 lines overhead
	// So actual inner height = height - 4
	innerHeight := height - 4

	// Account for table elements within the inner area:
	// - Table header: 1 line
	// - Header separator: 1 line
	reservedLines := 2
	if m.searchMode {
		reservedLines += 4 // Search bar lines (prompt + help + spacing)
	}

	// Update table height (number of visible rows) based on available height
	tableHeight := innerHeight - reservedLines
	
	// CRITICAL: Clamp table height to innerHeight to prevent layout overflow
	// Minimum of 2 rows if space allows
	if tableHeight < 2 && innerHeight > reservedLines {
		tableHeight = 2
	}
	
	// Final absolute clamp
	if tableHeight > innerHeight - reservedLines {
		tableHeight = innerHeight - reservedLines
	}
	if tableHeight < 0 {
		tableHeight = 0
	}
	
	m.pagination.SetPerPage(tableHeight)
	m.messageTable = m.messageTable.WithPageSize(tableHeight)

	// Calculate column widths based on available width
	// Account for table border (2) and separators (4 for 5 columns) = 6 chars total overhead
	availableWidth := width - 6
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

	// Ensure value column gets at least its minimum
	if valueWidth < minValueWidth {
		valueWidth = minValueWidth
	}

	// Update column widths
	columns := []table.Column{
		table.NewColumn("offset", "Offset", offsetWidth),
		table.NewColumn("partition", "Partition", partitionWidth),
		table.NewColumn("timestamp", "Timestamp", timestampWidth),
		table.NewColumn("key", "Key", keyWidth),
		table.NewColumn("value", "Value", valueWidth),
	}
	m.messageTable = m.messageTable.WithColumns(columns)
	m.tableColumns = columns
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
	// Get messages for current page
	paginatedMessages := m.pagination.GetVisibleMessages(m.filteredMessages)
	if len(paginatedMessages) == 0 {
		return nil
	}

	highlightedIndex := m.messageTable.GetHighlightedRowIndex()
	if highlightedIndex >= 0 && highlightedIndex < len(paginatedMessages) {
		selectedMsg := &paginatedMessages[highlightedIndex]
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
		m.pagination.SetTotalMessages(len(m.messages))
		m.updateMessageTable()
		m.markRenderDirty()
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
	m.pagination.SetTotalMessages(len(filtered))
	m.updateMessageTable()
	m.markRenderDirty()
}

// updateMessageTable updates the table with paginated messages
func (m *Model) updateMessageTable() {
	// Ensure messages are sorted for pagination
	m.sortMessages()

	// Get paginated messages
	paginatedMessages := m.pagination.GetVisibleMessages(m.filteredMessages)

	// Sort messages for display based on current sort order
	// Create a copy to avoid modifying the original slice
	sortedMessages := make([]api.Message, len(paginatedMessages))
	copy(sortedMessages, paginatedMessages)
	sort.Slice(sortedMessages, func(i, j int) bool {
		if m.pagination.SortOrder == "newest_first" {
			return sortedMessages[i].Offset > sortedMessages[j].Offset
		}
		return sortedMessages[i].Offset < sortedMessages[j].Offset
	})

	// For large datasets, use fast rendering (bypass table component overhead)
	if len(m.filteredMessages) > UseCustomRenderer {
		m.renderTableFast(sortedMessages)
		return
	}

	// For small datasets, use standard table rendering
	rows := make([]table.Row, len(sortedMessages))

	searchQuery := ""
	if m.searchMode {
		searchQuery = m.searchInput.Value()
	}

	// Get current column widths for proper truncation
	var keyMaxLen, valueMaxLen int
	for _, col := range m.tableColumns {
		if col.Title() == "Key" {
			keyMaxLen = col.Width() - 1 // allow 1 padding
		} else if col.Title() == "Value" {
			valueMaxLen = col.Width() - 1
		}
	}
	if keyMaxLen < 1 {
		keyMaxLen = 20
	}
	if valueMaxLen < 1 {
		valueMaxLen = 40
	}

	for i, msg := range sortedMessages {
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

		rows[i] = table.NewRow(table.RowData{
			"offset":    offset,
			"partition": partition,
			"timestamp": timestamp,
			"key":       key,
			"value":     value,
		})
	}

	m.messageTable = m.messageTable.WithRows(rows)
}

// renderTableFast renders table rows with minimal overhead for large datasets
func (m *Model) renderTableFast(messages []api.Message) {
	rows := make([]table.Row, len(messages))
	for i, msg := range messages {
		rows[i] = table.NewRow(table.RowData{
			"offset":    fmt.Sprintf("%d", msg.Offset),
			"partition": fmt.Sprintf("%d", msg.Partition),
			"timestamp": "N/A",
			"key":       truncateString(msg.Key, 18),
			"value":     truncateString(msg.Value, 38),
		})
	}
	m.messageTable = m.messageTable.WithRows(rows)
}

// renderTableCustom renders a custom table view for large datasets with pagination
// This bypasses the bubbles table component overhead entirely
func (m *Model) renderTableCustom(width, height int) string {
	// Ensure messages are sorted for pagination
	m.sortMessages()

	messages := m.pagination.GetVisibleMessages(m.filteredMessages)
	if len(messages) == 0 {
		return "No messages"
	}

	// Sort messages for display based on current sort order
	sortedMessages := make([]api.Message, len(messages))
	copy(sortedMessages, messages)
	sort.Slice(sortedMessages, func(i, j int) bool {
		if m.pagination.SortOrder == "newest_first" {
			return sortedMessages[i].Offset > sortedMessages[j].Offset
		}
		return sortedMessages[i].Offset < sortedMessages[j].Offset
	})
	messages = sortedMessages

	// Calculate column widths based on available width
	// Account for spaces before each of the 4 columns
	availableWidth := width - 4
	if availableWidth < 60 {
		availableWidth = 60
	}

	// Define minimum column widths
	const (
		minOffsetWidth    = 10
		minPartitionWidth = 10
		minKeyWidth       = 20
		minValueWidth     = 15
	)

	// Calculate total minimum width required
	minTotalWidth := minOffsetWidth + minPartitionWidth + minKeyWidth + minValueWidth

	// Ensure available width is at least the minimum total
	if availableWidth < minTotalWidth {
		availableWidth = minTotalWidth
	}

	// Calculate remaining width after allocating minimums
	remainingWidth := availableWidth - minTotalWidth

	// Distribute remaining width: Value column gets most (60%), others share the rest
	offsetWidth := minOffsetWidth + remainingWidth*10/100
	partitionWidth := minPartitionWidth + remainingWidth*10/100
	keyWidth := minKeyWidth + remainingWidth*20/100
	// Value gets the remainder to ensure exact fit
	valueWidth := availableWidth - offsetWidth - partitionWidth - keyWidth

	// Ensure value column gets at least its minimum
	if valueWidth < minValueWidth {
		valueWidth = minValueWidth
	}

	// Calculate available rows based on height
	// Height passed is from content component:
	// - Content component has border (2) + padding (2) = 4 lines overhead
	// So actual inner height = height - 4
	innerHeight := height - 4

	// Reserve lines for: header (1), separator (1), column headers (1), separator (1), footer (1)
	reservedLines := 5
	availableRows := innerHeight - reservedLines
	if availableRows < 5 {
		availableRows = 5
	}
	// Limit messages to available rows
	if len(messages) > availableRows {
		messages = messages[:availableRows]
	}

	var sb strings.Builder

	// Header with pagination info
	header := fmt.Sprintf(" %s | Page %d/%d | %d msgs",
		m.topicName, m.pagination.Page+1, m.pagination.TotalPages, m.pagination.TotalMessages)
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render(header))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", width))
	sb.WriteString("\n")

	// Column headers with dynamic widths
	colHeaderFmt := fmt.Sprintf(" %%-%ds %%-%ds %%-%ds %%-%ds", offsetWidth, partitionWidth, keyWidth, valueWidth)
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf(colHeaderFmt, "Offset", "Partition", "Key", "Value"),
	))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", width))
	sb.WriteString("\n")

	// Get current cursor position for highlighting
	cursorRow := m.messageTable.GetHighlightedRowIndex()

	// Rows with dynamic widths
	rowFmt := fmt.Sprintf(" %%-%ds %%-%ds %%-%ds %%-%ds", offsetWidth, partitionWidth, keyWidth, valueWidth)
	for i, msg := range messages {
		offset := fmt.Sprintf("%d", msg.Offset)
		partition := fmt.Sprintf("%d", msg.Partition)
		key := truncateString(msg.Key, keyWidth)
		value := truncateString(msg.Value, valueWidth)

		line := fmt.Sprintf(rowFmt, offset, partition, key, value)

		// Highlight selected row
		if i == cursorRow {
			line = lipgloss.NewStyle().Background(lipgloss.Color("205")).Foreground(lipgloss.Color("0")).Render(line)
		}

		sb.WriteString(line)
		sb.WriteString("\n")
	}

	// Footer with pagination controls
	var footer string
	if m.pagination.TotalPages > 1 {
		footer = fmt.Sprintf(" [←/→] Page %d/%d | [R] refresh | [/] search | [space] pause",
			m.pagination.Page+1, m.pagination.TotalPages)
	} else {
		footer = fmt.Sprintf(" %d message(s) | [R] refresh | [/] search | [space] pause", m.pagination.TotalMessages)
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(footer))

	return sb.String()
}

// getCachedWidth returns cached column width or calculates and caches it
func (m *Model) getCachedWidth(columnTitle string, columns []table.Column) int {
	// Find the width for this column
	var width int
	for _, col := range columns {
		if col.Title() == columnTitle {
			width = col.Width()
			break
		}
	}

	// Try read lock first for better performance
	m.widthCacheMu.RLock()
	if m.widthCache[columnTitle] != nil {
		if cached, ok := m.widthCache[columnTitle][width]; ok {
			m.widthCacheMu.RUnlock()
			return cached
		}
	}
	m.widthCacheMu.RUnlock()

	// Need to write - acquire write lock
	m.widthCacheMu.Lock()
	defer m.widthCacheMu.Unlock()

	// Initialize cache for this column if needed
	if m.widthCache[columnTitle] == nil {
		m.widthCache[columnTitle] = make(map[int]int)
	}

	// Calculate and cache
	maxLen := width - 2 // Account for padding
	m.widthCache[columnTitle][width] = maxLen
	return maxLen
}

// initWidthCache initializes the width cache
func (m *Model) initWidthCache() {
	if m.widthCache == nil {
		m.widthCache = make(map[string]map[int]int)
	}
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

// addMessageInternal adds a message without triggering view update (for background consumption)
func (m *Model) addMessageInternal(msg api.Message) {
	key := fmt.Sprintf("%d-%d", msg.Partition, msg.Offset)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Only add if not already consumed
	if _, exists := m.consumedMessages[key]; !exists {
		m.consumedMessages[key] = msg

		// Enforce message buffer limit (FIFO)
		if len(m.messages) >= m.maxMessages {
			oldMsg := m.messages[0]
			oldKey := fmt.Sprintf("%d-%d", oldMsg.Partition, oldMsg.Offset)
			delete(m.consumedMessages, oldKey)
			m.messages = m.messages[1:]
		}

		// Add new message
		m.messages = append(m.messages, msg)

		// Update filtered messages but don't trigger render
		m.filteredMessages = m.messages
		m.pagination.SetTotalMessages(len(m.filteredMessages))
	}
}

// AddMessage adds a new message and triggers view update (for manual refresh)
func (m *Model) AddMessage(msg api.Message) {
	m.addMessageInternal(msg)
	m.statusMessage = fmt.Sprintf("Consumed %d messages", len(m.messages))
	m.markRenderDirty()
}

// shouldUpdate checks if enough time has passed since last update (throttling)
func (m *Model) shouldUpdate() bool {
	if m.updateThrottle <= 0 {
		return true
	}
	return time.Since(m.lastUpdateTime) >= m.updateThrottle
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
// It also strips newlines to prevent layout corruption in tables
func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	// Strip newlines and carriage returns to prevent layout breaking
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	
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
	// Only update the reusable app. The topic model's dimensions will be 
	// updated by the ContentProvider with the correct inner content dimensions.
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
