package topic

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/masking"
	"github.com/Benny93/kafui/pkg/serde"
	"github.com/Benny93/kafui/pkg/messagefilter"
	"github.com/Benny93/kafui/pkg/ui/components"
	formpkg "github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
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
)

// Model represents the topic page state (original business logic model)
type Model struct {
	// Common context
	common *core.Common

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
	consumeMode  ConsumeMode

	// UI Components
	messageTable    table.Model
	spinner         spinner.Model
	searchInput     textinput.Model
	fetchProgressBar components.FetchProgressBar // animated progress bar during FetchLatestMessages

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
	widthCache   map[string]map[int]int // column -> width -> cached value
	widthCacheMu sync.RWMutex           // Protects widthCache

	// Update throttling
	lastUpdateTime time.Time
	updateThrottle time.Duration

	// Batching
	batchSize     int
	batchInterval time.Duration

	// Mutex for thread-safe message operations
	mu sync.RWMutex

	// Render caching (avoid re-rendering same content)
	// Render caching: renderVersion is incremented on every change that requires
	// a new View() output. TopicPageModel caches the full rendered page and only
	// rebuilds when this counter advances.
	renderVersion uint64

	// pendingReset signals that the next updateMessageTable call should reset
	// the highlighted row to 0 (used on fresh data loads, filter changes, page nav).
	pendingReset bool

	// appendNextFetch tracks how many in-flight batch fetches are pending.
	// Any MessagesFetchedMsg that arrives while this is > 0 is treated as an
	// append rather than a fresh replace.
	appendNextFetch int

	// cursorRow is the index of the highlighted row in the currently displayed page.
	cursorRow int

	// Row string cache: holds the unstyled row strings for the current page.
	// Rebuilt only when visible rows change (data, page nav, resize, sort).
	// On cursor-only changes the rows are reused — only the highlight is reapplied.
	rowStringCache      []string
	rowStringCacheWidth int  // width at which cache was built (invalidate on resize)
	rowStringsDirty     bool // true when row content must be rebuilt

	// Consumer-groups overlay (CG-21). Fetched on demand (explicit keypress)
	// because GetConsumerGroupsForTopic fans out across group coordinators.
	showGroups    bool
	groups        []api.ConsumerGroup
	groupsCursor  int
	groupsLoading bool

	// Overview + partition table overlay (TP-23). Data fetched on open.
	showOverview    bool
	overviewLoading bool
	overview        *api.TopicDetails
	overviewSize    int64
	overviewErr     error
	partitionCursor int

	// Settings/config overlay (TP-24).
	showSettings    bool
	settingsLoading bool
	settingsConfig  []api.TopicConfigEntry
	settingsErr     error

	// Edit-settings form overlay (TP-25).
	showSettingsEdit bool
	settingsForm     *formpkg.Form
	loadedConfig     []api.TopicConfigEntry // config snapshot at form-open, for diffing

	// Mutation dialog overlay (partition increase / replication factor, TP-26).
	showMutationForm bool
	mutationForm     *formpkg.Form
	mutationKind     mutationKind

	// Statistics/analysis overlay (TP-31).
	showAnalysis    bool
	analysisLoading bool
	analysis        *api.TopicAnalysis

	// --- Message browsing/producing overlays (MSG-21..32) ---

	// Seek dialog (MSG-21).
	showSeek bool
	seekForm *formpkg.Form

	// Partition filter + serde selector (MSG-22).
	showPartitions bool
	partitionForm  *formpkg.Form

	// Produce / reproduce form (MSG-31/MSG-32).
	showProduce bool
	produceForm *formpkg.Form

	// Saved-filters picker (MSG-25).
	showSavedFilters  bool
	savedFilterCursor int

	// Field-projection dialog (MSG-26). Reuses seekForm as the input form.
	showProjections bool

	// Display serde preference (MSG-22): "auto" or an explicit serde name from
	// serdeReg. Applied to displayed key/value cells via the serde registry.
	keySerde   string
	valueSerde string
	// serdeReg is the serde registry backing the display selector (MSG-11..18).
	serdeReg *serde.Registry

	// Field-preview projections (MSG-26): JSON dotted paths for the key/value columns.
	keyProjection   string
	valueProjection string

	// Smart filter (MSG-24/25). When smartFilter != nil, filtering evaluates the
	// compiled expression instead of substring matching; smartFilterErrs counts
	// per-message evaluation errors (skipped rows).
	smartFilter     *messagefilter.Filter
	smartFilterErrs int

	// Data masking applied at display time (MSG-28). Nil when no rules configured.
	masker *masking.Masker

	// Browse statistics (MSG-27): elapsed + byte/message counters for the last fetch.
	browseStart time.Time
	browseStats api.BrowseStats
}

// anyModalOverlayOpen reports whether a form-style overlay currently owns input.
func (m *Model) anyOverlayOpen() bool {
	return m.showGroups || m.showOverview || m.showSettings ||
		m.showSettingsEdit || m.showMutationForm || m.showAnalysis ||
		m.showSeek || m.showPartitions || m.showProduce || m.showSavedFilters ||
		m.showProjections
}

// markRenderDirty increments the render version, signalling that the cached
// full-page View() output is stale and must be rebuilt.
func (m *Model) markRenderDirty() {
	m.renderVersion++
}

// getRenderCache and setRenderCache are unused — kept as stubs to avoid breaking tests.

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
				BorderForeground(stylesPkg.FgSubtle),
		).
		HeaderStyle(
			lipgloss.NewStyle().
				Foreground(stylesPkg.FgMuted).
				Bold(true),
		).
		HighlightStyle(
			lipgloss.NewStyle().
				Background(stylesPkg.Primary).
				Foreground(stylesPkg.BgBase).
				Bold(true),
		).
		Focused(true)
		// Note: do NOT call SortByDesc here — that uses lexicographic string
		// comparison which breaks numeric offset ordering (e.g. "99" > "145").
		// Messages are sorted numerically in updateMessageTable() before the
		// rows are passed to the table.

	// Initialize search input
	searchInput := textinput.New()
	searchInput.Placeholder = "Search messages..."
	searchInput.CharLimit = 156
	searchInput.Width = 30

	// Initialize spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(stylesPkg.Primary)

	m := &Model{
		common:           nil, // Will be set by NewTopicPageModelWithCommon
		dataSource:       dataSource,
		topicName:        topicName,
		topicDetails:     topicDetails,
		consumeMode:      ModeNewest,
		consumeFlags:     consumeFlagsForMode(ModeNewest),
		messages:         []api.Message{},
		consumedMessages: make(map[string]api.Message),
		messageTable:     messageTable,
		tableColumns:     columns,
		spinner:          sp,
		fetchProgressBar: components.NewFetchProgressBar(),
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

		// Serde display preference defaults to auto (current decode behaviour).
		keySerde:   serde.Auto,
		valueSerde: serde.Auto,
	}
	// Built-in serde registry for display transforms (schema-registry Avro is
	// handled by the datasource's DecodeMessage, so a nil decoder is fine here).
	m.serdeReg, _ = serde.BuildRegistry(nil, nil)

	// Restore per-topic projections persisted from a previous session (MSG-26).
	if proj, ok := shared.LoadPrefs().Projections[topicName]; ok {
		m.keyProjection = proj.Key
		m.valueProjection = proj.Value
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

// consumeFlagsForMode returns the appropriate ConsumeFlags for the given mode.
func consumeFlagsForMode(mode ConsumeMode) api.ConsumeFlags {
	switch mode {
	case ModeNewest:
		return api.ConsumeFlags{
			Follow:     false,
			Tail:       60,
			OffsetFlag: "latest",
		}
	case ModeOldest:
		return api.ConsumeFlags{
			Follow:     false,
			Tail:       0,
			OffsetFlag: "oldest",
		}
	case ModeLive:
		return api.ConsumeFlags{
			Follow:     true,
			Tail:       0,
			OffsetFlag: "latest",
		}
	}
	return api.DefaultConsumeFlags()
}

const batchSize int64 = 60 // messages per fetch batch

// minLoadedOffset returns the lowest offset across all loaded messages, or -1 if none.
func (m *Model) minLoadedOffset() int64 {
	if len(m.messages) == 0 {
		return -1
	}
	min := m.messages[0].Offset
	for _, msg := range m.messages[1:] {
		if msg.Offset < min {
			min = msg.Offset
		}
	}
	return min
}

// maxLoadedOffset returns the highest offset across all loaded messages, or -1 if none.
func (m *Model) maxLoadedOffset() int64 {
	if len(m.messages) == 0 {
		return -1
	}
	max := m.messages[0].Offset
	for _, msg := range m.messages[1:] {
		if msg.Offset > max {
			max = msg.Offset
		}
	}
	return max
}

// nextBatchFlags returns ConsumeFlags that fetch the next batch beyond the
// currently loaded messages, in the direction appropriate for the current mode.
// Returns nil when there is nowhere to go (already at the start of the topic).
func (m *Model) nextBatchFlags() *api.ConsumeFlags {
	switch m.consumeMode {
	case ModeNewest:
		// Go to older messages: start batchSize offsets before the current minimum.
		min := m.minLoadedOffset()
		if min <= 0 {
			return nil // already at the beginning
		}
		start := min - batchSize
		if start < 0 {
			start = 0
		}
		f := api.ConsumeFlags{
			Follow:        false,
			Tail:          0, // must be 0 so OffsetFlag numeric value is used
			OffsetFlag:    fmt.Sprintf("%d", start),
			LimitMessages: batchSize,
		}
		return &f
	case ModeOldest:
		// Go to newer messages: start right after the current maximum.
		max := m.maxLoadedOffset()
		if max < 0 {
			return nil
		}
		f := api.ConsumeFlags{
			Follow:        false,
			Tail:          0,
			OffsetFlag:    fmt.Sprintf("%d", max+1),
			LimitMessages: batchSize,
		}
		return &f
	}
	return nil
}

// startForMode clears buffered messages and starts consumption for the
// current consumeMode. Returns the initial command(s) to run.
func (m *Model) startForMode() tea.Cmd {
	// Clear existing messages
	m.mu.Lock()
	m.messages = []api.Message{}
	m.consumedMessages = make(map[string]api.Message)
	m.filteredMessages = []api.Message{}
	m.mu.Unlock()
	m.pagination.SetTotalMessages(0)
	m.pagination.FirstPage()
	m.pendingReset = true
	m.markRenderDirty()

	m.consumeFlags = consumeFlagsForMode(m.consumeMode)

	// Sort order: Oldest mode shows lowest offset first; all others show newest first.
	if m.consumeMode == ModeOldest {
		m.pagination.SortOrder = "oldest_first"
	} else {
		m.pagination.SortOrder = "newest_first"
	}

	switch m.consumeMode {
	case ModeNewest, ModeOldest:
		m.loading = true
		m.consuming = false
		return m.consumption.FetchLatestMessages(60)
	case ModeLive:
		m.loading = false
		m.consuming = false
		return m.consumption.StartConsuming()
	}
	return nil
}

// startForFlags clears buffered messages and (re)starts browsing with an
// explicit set of ConsumeFlags chosen by the seek/partition dialogs (MSG-21/22).
func (m *Model) startForFlags(flags api.ConsumeFlags) tea.Cmd {
	m.mu.Lock()
	m.messages = []api.Message{}
	m.consumedMessages = make(map[string]api.Message)
	m.filteredMessages = []api.Message{}
	m.mu.Unlock()
	m.pagination.SetTotalMessages(0)
	m.pagination.FirstPage()
	m.pendingReset = true
	m.rowStringsDirty = true
	m.markRenderDirty()

	m.consumeFlags = flags
	m.browseStart = time.Now()
	m.browseStats = api.BrowseStats{}

	if flags.Seek.Backward() {
		m.pagination.SortOrder = "newest_first"
	} else {
		m.pagination.SortOrder = "oldest_first"
	}

	// Live seek tails in real time; everything else is a bounded fetch.
	if flags.Seek == api.SeekLive {
		m.consumeMode = ModeLive
		m.loading = false
		m.consuming = false
		return m.consumption.StartConsuming()
	}

	m.loading = true
	m.consuming = false
	count := int(flags.LimitMessages)
	if count <= 0 {
		count = seekPageSize
	}
	return m.consumption.FetchWithFlags(flags, count)
}

// Init implements the Page interface for the original model
func (m *Model) Init() tea.Cmd {
	m.loading = true
	return tea.Batch(m.startForMode(), m.spinner.Tick)
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

	// height is the inner content area height (border + padding already excluded by
	// the content component before calling RenderContent).
	// We just need to reserve lines for the table's own chrome: header(1) + separator(1) + footer(1) = 3.
	reservedLines := 3
	if m.searchMode {
		reservedLines += 4 // Search bar lines (prompt + help + spacing)
	}

	tableHeight := height - reservedLines

	// Enforce minimum
	if tableHeight < 2 {
		tableHeight = 2
	}

	shared.Log.Info("table dimensions", "topic", m.topicName,
		"contentW", width, "contentH", height, "pageSize", tableHeight)

	m.pagination.SetPerPage(tableHeight)
	// Note: We use WithPageSize to enable pagination footer and row limiting
	// The table is stateless - we pass only current page's data
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

	// Rebuild table rows for the new page size / column widths.
	m.updateMessageTable()
	m.rowStringsDirty = true // column widths changed — force row string rebuild
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

// GetSelectedMessage returns the message at the current cursor position.
// Pure query — no side effects, safe to call from the render path.
func (m *Model) GetSelectedMessage() *api.Message {
	paginatedMessages := m.pagination.GetVisibleMessages(m.filteredMessages)
	if len(paginatedMessages) == 0 {
		return nil
	}

	// cursorRow is the row position in the DISPLAYED table (may be reversed for
	// newest_first). Map it back to the ascending storage index.
	highlightedIndex := m.cursorRow
	if m.pagination.SortOrder == "newest_first" {
		highlightedIndex = len(paginatedMessages) - 1 - highlightedIndex
	}

	if highlightedIndex >= 0 && highlightedIndex < len(paginatedMessages) {
		return &paginatedMessages[highlightedIndex]
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

// FilterMessages recomputes the filtered view from the current search input or
// active smart filter (MSG-23/24), then refreshes pagination and the table.
func (m *Model) FilterMessages() {
	m.applyFilter()
	m.pagination.SetTotalMessages(len(m.filteredMessages))
	m.pendingReset = true
	m.updateMessageTable()
	m.markRenderDirty()
}

// updateMessageTable updates the table with paginated messages
// updateMessageTable updates pagination state and preserves/resets the
// highlighted row. Row content is rendered on demand by renderTableCustom —
// no table.Row objects are built here.
func (m *Model) updateMessageTable() {
	visibleCount := len(m.pagination.GetVisibleMessages(m.filteredMessages))

	if m.pendingReset || m.cursorRow >= visibleCount {
		m.cursorRow = 0
	}
	m.pendingReset = false
	m.rowStringsDirty = true // visible rows changed — rebuild on next render
}

// renderTableCustom renders the message table as plain text with direct string
// building. Row strings are cached and only rebuilt when data/layout changes;
// on cursor-only moves the cached rows are reused with just the highlight
// reapplied — making scrolling essentially free.
func (m *Model) renderTableCustom(width, height int) string {
	messages := m.pagination.GetVisibleMessages(m.filteredMessages)
	if len(messages) == 0 {
		// Empty state distinguishes an empty topic from a filter with no matches (MSG-27).
		if len(m.messages) > 0 {
			return lipgloss.NewStyle().Foreground(stylesPkg.FgMuted).Padding(1).
				Render("No messages match the current filter.")
		}
		return lipgloss.NewStyle().Foreground(stylesPkg.FgMuted).Padding(1).
			Render("No messages found.")
	}

	// Apply display sort order (storage is always ascending; newest_first reverses it).
	sortedMessages := make([]api.Message, len(messages))
	copy(sortedMessages, messages)
	if m.pagination.SortOrder == "newest_first" {
		sort.Slice(sortedMessages, func(i, j int) bool {
			return sortedMessages[i].Offset > sortedMessages[j].Offset
		})
	}
	messages = sortedMessages

	// Column width calculation
	availableWidth := width - 4
	if availableWidth < 60 {
		availableWidth = 60
	}
	const (
		minOffsetWidth    = 10
		minPartitionWidth = 8
		minTimeWidth      = 19
		minKeyWidth       = 18
		minValueWidth     = 15
	)
	minTotalWidth := minOffsetWidth + minPartitionWidth + minTimeWidth + minKeyWidth + minValueWidth
	if availableWidth < minTotalWidth {
		availableWidth = minTotalWidth
	}
	remainingWidth := availableWidth - minTotalWidth
	offsetWidth := minOffsetWidth + remainingWidth*10/100
	partitionWidth := minPartitionWidth
	timeWidth := minTimeWidth
	keyWidth := minKeyWidth + remainingWidth*30/100
	valueWidth := availableWidth - offsetWidth - partitionWidth - timeWidth - keyWidth
	if valueWidth < minValueWidth {
		valueWidth = minValueWidth
	}

	// Limit to available screen rows
	innerHeight := height - 4
	availableRows := innerHeight - 5 // header(1)+sep(1)+colhdr(1)+sep(1)+footer(1)
	if availableRows < 5 {
		availableRows = 5
	}
	if len(messages) > availableRows {
		messages = messages[:availableRows]
	}

	// Rebuild unstyled row strings only when content changed or width changed.
	if m.rowStringsDirty || m.rowStringCacheWidth != width || len(m.rowStringCache) != len(messages) {
		rowFmt := fmt.Sprintf(" %%-%ds %%-%ds %%-%ds %%-%ds %%-%ds", offsetWidth, partitionWidth, timeWidth, keyWidth, valueWidth)
		m.rowStringCache = make([]string, len(messages))
		for i, msg := range messages {
			ts := ""
			if !msg.Timestamp.IsZero() {
				ts = shared.FormatTimestamp(msg.Timestamp)
			}
			m.rowStringCache[i] = fmt.Sprintf(rowFmt,
				fmt.Sprintf("%d", msg.Offset),
				fmt.Sprintf("%d", msg.Partition),
				truncateString(ts, timeWidth),
				truncateString(m.displayKey(msg), keyWidth),
				truncateString(m.displayValue(msg), valueWidth),
			)
		}
		m.rowStringCacheWidth = width
		m.rowStringsDirty = false
	}

	var sb strings.Builder

	// Header
	header := fmt.Sprintf(" %s | Page %d/%d | %d msgs",
		m.topicName, m.pagination.Page+1, m.pagination.TotalPages, m.pagination.TotalMessages)
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(stylesPkg.Primary).Render(header))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", width))
	sb.WriteString("\n")

	// Column headers
	colHeaderFmt := fmt.Sprintf(" %%-%ds %%-%ds %%-%ds %%-%ds %%-%ds", offsetWidth, partitionWidth, timeWidth, keyWidth, valueWidth)
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf(colHeaderFmt, "Offset", "Partition", "Timestamp", "Key", "Value"),
	))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", width))
	sb.WriteString("\n")

	// Rows — highlight only the cursor row, everything else is plain text
	highlightStyle := lipgloss.NewStyle().Background(stylesPkg.Primary).Foreground(stylesPkg.BgBase)
	for i, rowStr := range m.rowStringCache {
		if i == m.cursorRow {
			sb.WriteString(highlightStyle.Render(rowStr))
		} else {
			sb.WriteString(rowStr)
		}
		sb.WriteString("\n")
	}

	// Footer with browse statistics (MSG-27): message/byte counts, elapsed, filter errors.
	stats := fmt.Sprintf(" %d msgs • %s • %dms",
		m.browseStats.MessagesConsumed, shared.FormatBytes2dp(m.browseStats.BytesConsumed), m.browseStats.ElapsedMs)
	if m.smartFilterErrs > 0 {
		stats += fmt.Sprintf(" • %d filter errors", m.smartFilterErrs)
	}
	var footer string
	if m.pagination.TotalPages > 1 {
		footer = fmt.Sprintf(" [←/→] Page %d/%d |%s | [S] seek [#] parts [P] produce",
			m.pagination.Page+1, m.pagination.TotalPages, stats)
	} else {
		footer = fmt.Sprintf("%s | [S] seek [#] parts [P] produce [/] search", stats)
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(stylesPkg.FgMuted).Render(footer))

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
		highlighted := lipgloss.NewStyle().Foreground(stylesPkg.Primary).Render(match)
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

// sanitizeForDisplay removes control characters from raw Kafka message content.
// Binary payloads (Avro, Protobuf, etc.) can contain bytes like ESC (0x1b),
// VT (0x0b), cursor-up (0x1b 0x5b 0x41) and similar terminal control sequences
// that cause the TUI to drift or break when rendered. Tabs are replaced with a
// single space; all other C0/C1 control characters are dropped.
func sanitizeForDisplay(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\t' {
			b.WriteByte(' ')
		} else if unicode.IsControl(r) {
			// Drop: C0 (0x00-0x1f) and C1 (0x7f-0x9f) control characters.
			// This includes \n, \r, ESC (0x1b), VT (0x0b), FF (0x0c), etc.
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// truncateString sanitizes and truncates a string to fit within maxLen visual
// characters.  It handles ANSI escape codes, multi-byte characters, and strips
// all terminal control bytes that could corrupt the table layout.
func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	s = sanitizeForDisplay(s)
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
	// Shared context
	common *core.Common

	// Original topic model for business logic
	topicModel *Model

	// Template system
	reusableApp     *templateui.ReusableApp
	contentProvider *TopicContentProvider

	// Full-page view cache. The template system (JoinHorizontal/Vertical, border
	// rendering) is expensive. We rebuild it only when renderVersion advances.
	viewCache        string
	viewCacheVersion uint64
}

// GetCommon returns the shared context
func (t *TopicPageModel) GetCommon() *core.Common {
	return t.common
}

// NewTopicPageModel creates a new topic page model using the template system
// Deprecated: Use NewTopicPageModelWithCommon for new code
func NewTopicPageModel(dataSource api.KafkaDataSource, topicName string, topicDetails api.Topic) *TopicPageModel {
	// Create Common context with data source
	common := core.NewCommon(dataSource)
	return NewTopicPageModelWithCommon(common, topicName, topicDetails)
}

// NewTopicPageModelWithCommon creates a new topic page model using the Common context pattern
func NewTopicPageModelWithCommon(common *core.Common, topicName string, topicDetails api.Topic) *TopicPageModel {
	// Create the original topic model for business logic
	topicModel := NewModel(common.DataSource, topicName, topicDetails)
	// Set common context for layout system access
	topicModel.common = common
	// Build the display-time masker from per-cluster masking rules (MSG-28).
	topicModel.masker = buildMaskerFromConfig(common)
	// Apply per-cluster serde config: rebuild the registry with configured
	// serdes and pre-select any topic-bound serde (MSG-17).
	topicModel.applySerdeConfig(common)

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
		common:          common,
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

// View implements the Page interface — caches the full rendered page and only
// rebuilds when the model's renderVersion has advanced.
func (t *TopicPageModel) View() string {
	v := t.topicModel.renderVersion
	if t.viewCache != "" && t.viewCacheVersion == v {
		return t.viewCache
	}
	t.viewCache = t.reusableApp.View()
	t.viewCacheVersion = v
	return t.viewCache
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

// TopicModel returns the internal topic model for testing purposes
func (t *TopicPageModel) TopicModel() *Model {
	return t.topicModel
}

// Navigation message for message detail selection
type NavigateToMessageDetailMsg struct {
	Message api.Message
}
