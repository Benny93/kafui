package mainpage

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components"
	"github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
	zone "github.com/lrstanley/bubblezone"
)

// KafuiContentProvider provides the main content for Kafui (resource table and search)
// Implements providers.ContentProvider interface
type KafuiContentProvider struct {
	dataSource      api.KafkaDataSource
	common          *core.Common
	resourceManager *ResourceManager
	currentResource Resource

	// Styles
	styles *stylesPkg.Styles

	// Table and search state
	resourcesTable table.Model
	searchMode     bool
	loading        bool
	error          error

	// Resource picker state (UI-8): `:` opens a capability-filtered picker with
	// autocomplete; enter switches, esc cancels back to the current resource.

	// Data storage
	allItems      []interface{}
	filteredItems []interface{}

	// Filter state
	isFiltered    bool
	currentFilter string

	// Pagination (50 items per logical page)
	pagination *ResourcePaginationModel

	// pendingReset signals that the next data load should jump to page 0 / row 0
	// (set by switchResource; NOT set by background auto-refreshes).
	pendingReset bool

	// Current page size (updated from dimensions for click-to-select math)
	perPage int

	// clicks distinguishes a double click from two single clicks on the table.
	clicks core.ClickTracker
	// tableWidth is the width the table was last laid out at, needed to resolve
	// which column a header click landed on.
	tableWidth int

	// nameColumnWidth tracks the rendered width of the Name column so that
	// middle-truncation is proportional to the actual terminal width.
	// Updated every RenderContent call.
	nameColumnWidth int

	// countLoading is true while a GetTopicMessageCounts command is in flight.
	// The spinner animates the placeholder cells during this window.
	countLoading      bool
	countSpinner      spinner.Model
	countSpinnerFrame string // current animated frame used as placeholder in table cells

	// countsInFlight holds the topics of the GetTopicMessageCounts request in
	// flight, or nil when none is. It keeps a second request from starting
	// while one is still running.
	countsInFlight map[string]bool
	// countsFetchedAt is when the known message counts were last re-fetched;
	// they are only fetched again once older than topicCountsMaxAge.
	countsFetchedAt time.Time
	// countsGen tags each GetTopicMessageCounts request; a context switch bumps
	// it so a result from the previous cluster is dropped instead of applied.
	countsGen uint64
	// countsCtx is the context the count state belongs to; see syncCountsContext.
	countsCtx string

	// detailsLoading is true while a GetTopics() detail-fetch is in flight
	// (second phase of two-phase topic loading). The same spinner is used.
	detailsLoading bool

	// clipboardMsg holds a short feedback string shown in the table footer
	// after a successful copy. Cleared by ClearClipboardFeedbackMsg.
	clipboardMsg string

	// brokerSortCol is the column index the brokers resource is sorted by
	// (see brokerSortColumns); -1 means unsorted. brokerSortDesc toggles direction.
	brokerSortCol  int
	brokerSortDesc bool

	// groupSortCol is the column index the consumer-groups resource is sorted by
	// (see groupSortColumns); -1 means default name-asc. groupSortDesc toggles
	// direction. groupStateFilter, when non-empty, restricts rows to one canonical
	// state (composes with the search filter).
	groupSortCol     int
	groupSortDesc    bool
	groupStateFilter string

	// topicSortCol is the column index the topics resource is sorted by
	// (see topicSortColumns); -1 means default name-asc. topicSortDesc toggles
	// direction.
	topicSortCol  int
	topicSortDesc bool

	// hideInternal hides topics whose name matches the internal prefix; persisted
	// in the local prefs file and reapplied on startup.
	hideInternal bool

	// selected tracks the multi-selected topic names (batch delete/purge).
	selected map[string]bool

	// topicForm is the overlay create/clone form; showTopicForm gates rendering
	// and key routing while it is visible.
	topicForm     *form.Form
	showTopicForm bool

	// ACL overlay forms (AQ-16/AQ-17/AQ-18): the create/convenience form and the
	// declarative-sync file-path prompt.
	aclForm         *form.Form
	showACLForm     bool
	aclSyncForm     *form.Form
	showACLSyncForm bool

	// Quota overlay form (AQ-20). quotaEditEntity is non-nil in edit mode (entity
	// fixed) and nil in create mode.
	quotaForm       *form.Form
	showQuotaForm   bool
	quotaEditEntity *api.ClientQuotaEntity

	// Connector create overlay form (KC-17): connect-cluster / name / plugin /
	// JSON config. connectForm is nil unless showConnectForm is true.
	connectForm     *form.Form
	showConnectForm bool
}

func NewKafuiContentProvider(dataSource api.KafkaDataSource) *KafuiContentProvider {
	// Initialize resource manager
	resourceManager := NewResourceManager(dataSource)
	currentResource := resourceManager.GetResource(TopicResourceType)

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return &KafuiContentProvider{
		dataSource:      dataSource,
		resourceManager: resourceManager,
		currentResource: currentResource,
		resourcesTable:  createResourcesTable(),
		styles:          stylesPkg.DefaultStyles(),
		allItems:        []interface{}{},
		filteredItems:   []interface{}{},
		pagination:      NewResourcePaginationModel(),
		perPage:         20,
		countSpinner:    sp,
		brokerSortCol:   -1,
		groupSortCol:    -1,
		topicSortCol:    -1,
		hideInternal:    shared.LoadPrefs().HideInternalTopics,
		selected:        map[string]bool{},
	}
}

// NewKafuiContentProviderWithCommon creates a content provider using Common context
func NewKafuiContentProviderWithCommon(common *core.Common) *KafuiContentProvider {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return &KafuiContentProvider{
		dataSource:      common.DataSource,
		common:          common,
		resourceManager: NewResourceManager(common.DataSource),
		currentResource: NewResourceManager(common.DataSource).GetResource(TopicResourceType),
		resourcesTable:  createResourcesTable(),
		styles:          common.Styles,
		allItems:        []interface{}{},
		filteredItems:   []interface{}{},
		pagination:      NewResourcePaginationModel(),
		perPage:         20,
		countSpinner:    sp,
		brokerSortCol:   -1,
		groupSortCol:    -1,
		topicSortCol:    -1,
		hideInternal:    shared.LoadPrefs().HideInternalTopics,
		selected:        map[string]bool{},
	}
}

func (k *KafuiContentProvider) RenderContent(width, height int) string {
	// Use layout system for dimension calculations
	var tableHeight int
	var tableWidth int

	// width is the inner content width the template already computed (pane minus
	// border, padding and the scrollbar column), so use it verbatim — deriving
	// it from Layout.Main.Width instead over-estimates by the chrome and makes
	// the table overflow and wrap on narrow terminals.
	tableWidth = width
	if k.common != nil && k.common.Layout != nil {
		tableHeight = k.common.Layout.GetAvailableHeight() - 3 // Reserve space for padding
	} else {
		tableHeight = height - 6
	}

	// Reserve one line for search bar hint when active (no separate status line needed,
	// page info is shown in the table footer).
	if k.searchMode {
		tableHeight -= 3
	}

	// Reserve one line for the selected-name preview above the table.
	tableHeight -= 1

	// Ensure minimum dimensions
	if tableHeight < 5 {
		tableHeight = 5
	}
	if tableWidth < 20 {
		tableWidth = 20
	}

	// Update table visual dimensions and inject the correct page footer.
	k.perPage = tableHeight
	// Match the middle-truncation budget to the Name column's actual flex width.
	resType := TopicResourceType
	if k.currentResource != nil {
		resType = k.currentResource.GetType()
	}
	k.nameColumnWidth = firstColumnWidth(createResourceTableColumns(resType), tableWidth)
	if k.nameColumnWidth < 20 {
		k.nameColumnWidth = 20
	}
	k.tableWidth = tableWidth
	k.resourcesTable = k.resourcesTable.
		WithPageSize(tableHeight).
		WithTargetWidth(tableWidth).
		// Pad short result sets so the table fills the pane instead of
		// leaving a stub box floating above empty space.
		WithMinimumHeight(tableHeight).
		WithStaticFooter(k.tableFooterText())

	// The create/clone form takes over the content area when active.
	if k.showTopicForm && k.topicForm != nil {
		k.topicForm.SetDimensions(tableWidth, tableHeight)
		return k.topicForm.View()
	}
	// ACL / quota overlay forms take over the content area the same way.
	if f := k.activeOverlayForm(); f != nil {
		f.SetDimensions(tableWidth, tableHeight)
		return f.View()
	}

	// Resource picker overlay takes over the content area (UI-8).
	if k.error != nil {
		return k.renderError()
	}

	if k.loading && len(k.allItems) == 0 {
		return k.renderLoading(tableWidth, tableHeight)
	}

	if len(k.allItems) == 0 && !k.loading {
		return k.renderEmpty()
	}

	var content strings.Builder

	// Add search bar if in search mode
	if k.searchMode {
		searchBar := k.renderSearchBar(width)
		content.WriteString(searchBar)
		content.WriteString("\n\n")
	}

	// Show the full name of the highlighted item so middle-truncated names
	// are always readable.
	content.WriteString(k.renderSelectedName(tableWidth))
	content.WriteString("\n")

	// Render the main table wrapped in a bubblezone mark so mouse events
	// can reference the exact screen bounds of the table.
	content.WriteString(zone.Mark("resource-table", k.resourcesTable.View()))

	return content.String()
}

// renderSelectedName returns a single line showing the full (untruncated) name
// of the currently highlighted resource, clipped to tableWidth if necessary.
func (k *KafuiContentProvider) renderSelectedName(tableWidth int) string {
	item := k.GetSelectedResourceItem()
	if item == nil {
		return k.styles.Muted.Render("—")
	}
	fullName := k.getItemName(item)
	if fullName == "" {
		return k.styles.Muted.Render("—")
	}
	// Hard-clip to available width so the line never wraps.
	runes := []rune(fullName)
	if tableWidth > 4 && len(runes) > tableWidth-4 {
		fullName = string(runes[:tableWidth-4]) + "…"
	}
	label := k.styles.Muted.Render("▶ ")
	name := k.styles.Header.Bold(true).Render(fullName)
	return label + name
}

// tableFooterText returns the page indicator string for the bubble-table footer.
// When clipboard feedback is active it is shown instead of pagination info.
func (k *KafuiContentProvider) tableFooterText() string {
	if k.clipboardMsg != "" {
		return k.clipboardMsg
	}
	if k.pagination == nil || k.pagination.TotalItems == 0 {
		return ""
	}
	total := k.pagination.TotalPages
	current := k.pagination.Page + 1
	if total <= 0 {
		return fmt.Sprintf("Page %d/?", current)
	}
	return fmt.Sprintf("Page %d/%d", current, total)
}

// renderSearchBar renders the search input bar
func (k *KafuiContentProvider) renderSearchBar(width int) string {
	// Use semantic colors from style system
	searchStyle := k.styles.SearchStyle.Prompt
	promptStyle := k.styles.Muted

	// Create search prompt
	prompt := searchStyle.Render("🔍 Search: ")
	filter := k.currentFilter
	if filter == "" {
		filter = promptStyle.Render("(type to filter resources...)")
	}

	// Add cursor if in search mode
	cursor := ""
	if k.searchMode {
		cursor = searchStyle.Render("█")
	}

	searchLine := prompt + filter + cursor

	// Add help text
	helpText := k.styles.SearchStyle.Help.Render("ESC to cancel • Enter to search")

	return searchLine + "\n" + helpText
}

func (k *KafuiContentProvider) renderError() string {
	// SR-22: a cluster without a configured schema registry is not an error state
	// — surface a friendly, non-alarming message instead of a raw error.
	var notConfigured api.SchemaRegistryNotConfiguredError
	if errors.As(k.error, &notConfigured) {
		return k.styles.Muted.Render("No schema registry configured for this cluster.")
	}
	return k.styles.Error.Render(fmt.Sprintf("Error: %v", k.error))
}

func (k *KafuiContentProvider) renderLoading(width, height int) string {
	// Shared loading-indicator mechanism (UI-12): a centered animated spinner.
	frame := k.styles.StatusStyle.Info.Render(k.countSpinner.View())
	return components.CenteredLoading(frame, k.styles.Muted.Render("Loading resources…"), width, height)
}

func (k *KafuiContentProvider) renderEmpty() string {
	msg := "No resources found. Try refreshing or checking your connection."
	if k.currentResource != nil {
		switch k.currentResource.GetType() {
		case ConsumerGroupResourceType:
			msg = "No consumer groups found — the broker returned an empty list.\n" +
				"The connected certificate likely lacks the DESCRIBE ACL on consumer-group resources.\n" +
				"Check with: kaf group ls"
		case ACLResourceType:
			msg = "No ACL entries found — the broker returned an empty list.\n" +
				"The connected certificate may lack the DESCRIBE ACL on cluster resources.\n" +
				"Check with: kaf acl ls (or equivalent)"
		case BrokerResourceType:
			msg = "No brokers online."
		case SchemaResourceType:
			msg = "No schemas found in the registry."
		case QuotaResourceType:
			msg = "No client quotas configured — quotas may be unsupported on this cluster or none are set."
		case ConnectorResourceType:
			msg = "No connectors found."
		case ConnectClusterResourceType:
			msg = "No Connect clusters configured for this Kafka cluster."
		}
	}
	return k.styles.Muted.Render(msg)
}

func (k *KafuiContentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return k.handleKey(msg)

	case SearchTopicsMsg:
		k.handleSearch(string(msg))
		cmds = append(cmds, k.reloadPageDetails())

	case ClearSearchMsg:
		k.clearSearch()
		cmds = append(cmds, k.reloadPageDetails())

	case searchSettledMsg:
		// Typing has paused on this query: enrich the rows it brought onto
		// the page. A stale tick (the query changed since) does nothing.
		if msg.query == k.currentFilter {
			cmds = append(cmds, k.reloadPageDetails())
		}

	case SwitchResourceMsg:
		k.switchResource(msg)
		cmds = append(cmds, k.loadCurrentResource(), k.breadcrumbCmd(), k.countSpinner.Tick)

	case connectClusterSelectedMsg:
		// Drill into the connectors view pre-filtered to the chosen cluster.
		k.switchResource(SwitchResourceMsg(ConnectorResourceType))
		k.currentFilter = "connect:" + msg.cluster
		k.isFiltered = true
		cmds = append(cmds, k.loadCurrentResource(), k.breadcrumbCmd())

	case CurrentResourceListMsg:
		k.handleResourceList(msg)
		// If this was a quick (names-only) topic load, kick off the full detail
		// fetch; message counts will follow after TopicDetailsLoadedMsg arrives.
		// For all other resources (or full topic refreshes), load page details normally.
		if msg.ResourceType == TopicResourceType && k.hasTopicStubs() {
			cmds = append(cmds, k.loadTopicDetails())
		} else {
			cmds = append(cmds, k.loadPageDetails())
		}

	case TopicDetailsLoadedMsg:
		k.applyTopicDetails(msg)
		cmds = append(cmds, k.loadPageDetails())

	case TopicListMsg:
		k.handleTopicList(msg)

	case ErrorMsg:
		k.error = error(msg)
		k.loading = false

	case TimerTickMsg:
		// Only auto-refresh topics — they're the only resource that changes
		// frequently enough to warrant a 5-second poll. Consumer groups,
		// schemas, and contexts either change rarely or are too slow to re-fetch
		// on every tick (which would cause perpetual "Loading resources..." flicker).
		if !k.loading && k.currentResource != nil &&
			k.currentResource.GetType() == TopicResourceType {
			cmds = append(cmds, k.loadCurrentResource())
		}

	case StartResourceSwitchingMsg:
		// Superseded by the shell command palette, which this page feeds via
		// PaletteEntries. Kept as a message so existing senders still compile.
		return nil

	case TopicCountsLoadedMsg:
		cmds = append(cmds, k.applyTopicMessageCounts(msg))

	case TopicDetailsExtLoadedMsg:
		k.applyTopicDetailsExt(msg)

	case SchemaDetailsLoadedMsg:
		k.applySchemaDetails(msg)

	case BrokerStatsLoadedMsg:
		k.applyBrokerStats(msg)

	case ConsumerGroupDetailsLoadedMsg:
		k.applyGroupDetails(msg)

	case ClearClipboardFeedbackMsg:
		k.clipboardMsg = ""

	case spinner.TickMsg:
		// Also animate while the initial resource list is loading (UI-12).
		if k.countLoading || k.detailsLoading || (k.loading && len(k.allItems) == 0) {
			var tickCmd tea.Cmd
			k.countSpinner, tickCmd = k.countSpinner.Update(msg)
			k.countSpinnerFrame = k.countSpinner.View()
			k.updateTableForCurrentPage()
			return tickCmd
		}

	case SelectContextMsg:
		return k.handleSelectContext(msg)

	case tea.MouseMsg:
		return k.handleMouse(msg)

	default:
		return k.handleMutationResult(msg)
	}

	return tea.Batch(cmds...)
}

func (k *KafuiContentProvider) InitContent() tea.Cmd {
	// countSpinner.Tick animates the shared initial-load spinner (UI-12); it
	// self-perpetuates via the spinner.TickMsg handler while loading. It also
	// runs when the user returns to the page, so the known message counts are
	// re-fetched to reflect changes made elsewhere (e.g. producing on a topic).
	return tea.Batch(k.refreshCurrentResource(), k.breadcrumbCmd(), k.countSpinner.Tick)
}

// resourceLabel names the resource list being shown (e.g. "Topics"). It is the
// page title, so it is also the root breadcrumb on the pages opened from here.
func (k *KafuiContentProvider) resourceLabel() string {
	if k.currentResource == nil {
		return "Topics"
	}
	switch k.currentResource.GetType() {
	case ConsumerGroupResourceType:
		return "Consumer Groups"
	case SchemaResourceType:
		return "Schemas"
	case ContextResourceType:
		return "Contexts"
	case ACLResourceType:
		return "ACLs"
	case BrokerResourceType:
		return "Brokers"
	case QuotaResourceType:
		return "Quotas"
	case ConnectClusterResourceType:
		return "Connect Clusters"
	case ConnectorResourceType:
		return "Connectors"
	}
	return "Topics"
}

// breadcrumbCmd returns a Cmd that sends a BreadcrumbUpdateMsg for the resource
// list being shown. It matches the page title the router uses once the user
// navigates deeper, so the root crumb does not change name.
func (k *KafuiContentProvider) breadcrumbCmd() tea.Cmd {
	items := []string{k.resourceLabel()}
	return func() tea.Msg {
		return core.BreadcrumbUpdateMsg{Items: items}
	}
}

// activeOverlayForm returns the currently visible ACL/quota overlay form, or nil.
func (k *KafuiContentProvider) activeOverlayForm() *form.Form {
	switch {
	case k.showACLForm && k.aclForm != nil:
		return k.aclForm
	case k.showACLSyncForm && k.aclSyncForm != nil:
		return k.aclSyncForm
	case k.showQuotaForm && k.quotaForm != nil:
		return k.quotaForm
	case k.showConnectForm && k.connectForm != nil:
		return k.connectForm
	}
	return nil
}

// IsInputMode returns true when the search bar is active so that
// ReusableApp suppresses app-level hotkeys that would otherwise steal keystrokes.
func (k *KafuiContentProvider) IsInputMode() bool {
	return k.searchMode || k.showTopicForm || k.activeOverlayForm() != nil
}

// GetContentSize returns the estimated content size for scrollbar calculation
func (k *KafuiContentProvider) GetContentSize(width int) int {
	// Estimate based on the active row count plus header
	rowCount := len(k.activeItems())
	if rowCount == 0 {
		return 5 // Default for empty/loading states
	}
	// Add header lines and account for search bar
	return rowCount + 5
}

// Helper methods
