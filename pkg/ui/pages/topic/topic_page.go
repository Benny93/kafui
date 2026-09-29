package topic

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/masking"
	"github.com/Benny93/kafui/pkg/messagefilter"
	"github.com/Benny93/kafui/pkg/serde"
	"github.com/Benny93/kafui/pkg/ui/components"
	"github.com/Benny93/kafui/pkg/ui/components/editor"
	formpkg "github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
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
	messages         []api.Message
	consumedMessages map[string]api.Message
	filteredMessages []api.Message
	selectedMessage  *api.Message

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
	messageTable     table.Model
	spinner          spinner.Model
	searchInput      textinput.Model
	fetchProgressBar components.FetchProgressBar // animated progress bar during FetchLatestMessages

	// Bubble-table configuration
	tableColumns []table.Column

	// Components
	handlers    *Handlers
	keys        *Keys
	view        *View
	consumption *ConsumptionController

	// Consumption control. cancelConsumption is the one cancel func of the
	// running live stream; stopLive calls it.
	cancelConsumption context.CancelFunc
	msgChan           <-chan api.Message
	errChan           <-chan error

	// resumeLive is set when OnBlur stopped a live stream, so OnFocus can
	// restart it when the user comes back.
	resumeLive bool

	// fetchGen identifies the current fetch/stream generation. beginGeneration
	// advances it on every refresh, seek, mode switch and retry, and results
	// tagged with an older generation are dropped. fetchCtx is cancelled when
	// the generation ends, which stops its in-flight fetches and decodes.
	fetchGen    uint64
	fetchCtx    context.Context
	cancelFetch context.CancelFunc

	// pendingWhilePaused buffers live messages that arrive while paused; they
	// are merged into the table on resume.
	pendingWhilePaused []api.Message

	// Error handling and retry logic
	retryCount       int
	maxRetries       int
	retryDelay       time.Duration
	lastError        error
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

	// appendNextFetch counts the in-flight batch (append) fetches of the
	// current generation; the loading indicator stays on while it is > 0.
	// Whether a result appends is decided by MessagesFetchedMsg.Append.
	appendNextFetch int

	// cursorRow is the index of the highlighted row in the currently displayed page.
	cursorRow int
	// clicks distinguishes a double click from two single clicks.
	clicks core.ClickTracker

	// Row string cache: holds the unstyled row strings for the current page.
	// Rebuilt only when visible rows change (data, page nav, resize, sort).
	// On cursor-only changes the rows are reused — only the highlight is reapplied.
	rowStringCache      []string
	rowStringCacheWidth int  // width at which cache was built (invalidate on resize)
	rowStringsDirty     bool // true when row content must be rebuilt

	// Inline row expansion (row_expand.go): while expanded, the highlighted
	// row shows its full content in a scrollable panel directly beneath it.
	expanded     bool
	expandViewer *editor.Viewer
	expandFor    string      // identity of the message the viewer holds
	rowWindow    int         // first page row drawn while the panel takes space
	tableLayout  tableLayout // where the last render put rows and the panel

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

// OnFocus implements the Page interface for the original model. The router
// calls Init once, when the page is created, and OnFocus on every activation.
// The first load is Init's job, so OnFocus only restarts a live stream that
// OnBlur stopped (e.g. on the way back from a message detail), keeping the
// messages already shown.
func (m *Model) OnFocus() tea.Cmd {
	if !m.resumeLive {
		return nil
	}
	m.resumeLive = false
	if !m.tailsLive() {
		return nil
	}
	m.retryCount = 0
	return tea.Batch(m.startLive(), m.spinner.Tick)
}

// OnBlur implements the Page interface for the original model. It stops the
// live stream: pages stay cached after they are left, and once this page is no
// longer current nothing drains the stream, so the consumer would otherwise
// keep polling the broker for the rest of the session. In Live mode it also
// ends the generation, so a stream or retry still on its way is dropped (and
// cancelled) instead of starting while the page is hidden. A bounded seek
// run from Live mode is not a live stream: it is left alone like any fetch.
func (m *Model) OnBlur() tea.Cmd {
	if m.tailsLive() {
		m.resumeLive = true
		m.beginGeneration()
		m.SetConnectionStatus(StatusDisconnected)
		return nil
	}
	if m.consuming || m.cancelConsumption != nil {
		m.stopLive()
		m.SetConnectionStatus(StatusDisconnected)
	}
	return nil
}

// tailsLive reports whether the page shows a live stream, i.e. one OnFocus
// may restart with the current flags. A bounded seek or partition fetch keeps
// consumeMode at ModeLive but uses non-follow flags; restarting it as a
// stream would end on its own and loop through fail and retry.
func (m *Model) tailsLive() bool {
	return m.consumeMode == ModeLive && m.consumeFlags.Follow
}

// Dispose implements core.Disposer. It stops the live stream, cancels every
// in-flight fetch, decode and stream of the current generation, and makes any
// result still on its way stale. The page is never shown again.
func (m *Model) Dispose() {
	m.resumeLive = false
	m.stopLive()
	if m.cancelFetch != nil {
		m.cancelFetch()
	}
	m.fetchGen++
}

// IsInputMode implements core.InputModeReporter: true while the search/filter
// input is focused, so the shell passes every key to it instead of resolving
// shortcuts.
func (m *Model) IsInputMode() bool {
	return m.searchMode
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

// Navigation message for message detail selection
type NavigateToMessageDetailMsg struct {
	Message api.Message
}
