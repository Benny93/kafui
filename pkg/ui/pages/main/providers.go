package mainpage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components"
	"github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	keybindings "github.com/Benny93/kafui/pkg/ui/keys"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	zone "github.com/lrstanley/bubblezone"
)

// Column key constants for bubble-table
const (
	colName         = "name"
	colPartitions   = "partitions"
	colReplication  = "replication"
	colMessages     = "messages"
	colSchemaCompat = "schema_compat"
)

// Broker-specific column keys. The brokers resource uses its own six-column
// layout instead of the shared four-column mapping.
const (
	colBrokerID   = "broker_id"
	colBrokerHost = "broker_host"
	colBrokerPort = "broker_port"
	colBrokerDisk = "broker_disk"
	colBrokerISR  = "broker_isr"
	colBrokerSkew = "broker_skew"
)

// Consumer-group-specific column keys. Consumer groups use their own six-column
// layout (name/state/members/topics/lag/coordinator) enriched lazily.
const (
	colGroupName    = "group_name"
	colGroupState   = "group_state"
	colGroupMembers = "group_members"
	colGroupTopics  = "group_topics"
	colGroupLag     = "group_lag"
	colGroupCoord   = "group_coord"
)

// Topic-specific column keys. Topics use a dedicated six-column layout
// (name/partitions/replication/messages/osr/size) enriched lazily.
const (
	colTopicName        = "topic_name"
	colTopicPartitions  = "topic_partitions"
	colTopicReplication = "topic_replication"
	colTopicMessages    = "topic_messages"
	colTopicOSR         = "topic_osr"
	colTopicSize        = "topic_size"
)

// createResourceTableColumns creates column definitions for a given resource type.
func createResourceTableColumns(resourceType ResourceType) []table.Column {
	right := lipgloss.NewStyle().AlignHorizontal(lipgloss.Right)
	switch resourceType {
	case TopicResourceType:
		return []table.Column{
			table.NewColumn(colTopicName, "Name", 35),
			table.NewColumn(colTopicPartitions, "Partitions", 12).WithStyle(right),
			table.NewColumn(colTopicReplication, "Replication", 12).WithStyle(right),
			table.NewColumn(colTopicMessages, "Messages", 14).WithStyle(right),
			table.NewColumn(colTopicOSR, "OSR", 8).WithStyle(right),
			table.NewColumn(colTopicSize, "Size", 12).WithStyle(right),
		}
	case ContextResourceType:
		return []table.Column{
			table.NewColumn(colName, "Name", 35),
			table.NewColumn(colPartitions, "Brokers", 40),
			table.NewColumn(colReplication, "Status", 12).WithStyle(right),
		}
	case SchemaResourceType:
		return []table.Column{
			table.NewColumn(colName, "Subject", 32),
			table.NewColumn(colPartitions, "Version", 9).WithStyle(right),
			table.NewColumn(colReplication, "ID", 8).WithStyle(right),
			table.NewColumn(colMessages, "Type", 10),
			table.NewColumn(colSchemaCompat, "Compatibility", 20),
		}
	case ConsumerGroupResourceType:
		return []table.Column{
			table.NewColumn(colGroupName, "Name", 34),
			table.NewColumn(colGroupState, "State", 18),
			table.NewColumn(colGroupMembers, "Members", 9).WithStyle(right),
			table.NewColumn(colGroupTopics, "Topics", 8).WithStyle(right),
			table.NewColumn(colGroupLag, "Lag", 12).WithStyle(right),
			table.NewColumn(colGroupCoord, "Coordinator", 12).WithStyle(right),
		}
	case ACLResourceType:
		return []table.Column{
			table.NewColumn(colACLPrincipal, "Principal", 30),
			table.NewColumn(colACLResource, "Resource", 22),
			table.NewColumn(colACLPattern, "Pattern", 12),
			table.NewColumn(colACLHost, "Host", 12),
			table.NewColumn(colACLOperation, "Operation", 14),
			table.NewColumn(colACLPermission, "Permission", 10),
		}
	case QuotaResourceType:
		return []table.Column{
			table.NewColumn(colQuotaUser, "User", 18),
			table.NewColumn(colQuotaClient, "Client ID", 18),
			table.NewColumn(colQuotaIP, "IP", 16),
			table.NewColumn(colQuotaValues, "Quotas", 44),
		}
	case BrokerResourceType:
		return []table.Column{
			table.NewColumn(colBrokerID, "ID", 8).WithStyle(right),
			table.NewColumn(colBrokerHost, "Host", 24),
			table.NewColumn(colBrokerPort, "Port", 8).WithStyle(right),
			table.NewColumn(colBrokerDisk, "Disk Usage", 22),
			table.NewColumn(colBrokerISR, "ISR", 10).WithStyle(right),
			table.NewColumn(colBrokerSkew, "Skew", 10).WithStyle(right),
		}
	case ConnectorResourceType:
		return []table.Column{
			table.NewColumn(colConnName, "Name", 26),
			table.NewColumn(colConnCluster, "Connect", 16),
			table.NewColumn(colConnType, "Type", 8),
			table.NewColumn(colConnPlugin, "Plugin", 22),
			table.NewColumn(colConnTopics, "Topics", 20),
			table.NewColumn(colConnState, "Status", 12),
			table.NewColumn(colConnGroup, "Consumer group", 20),
			table.NewColumn(colConnTasks, "Tasks", 8).WithStyle(right),
		}
	case ConnectClusterResourceType:
		return []table.Column{
			table.NewColumn(colCCName, "Name", 28),
			table.NewColumn(colCCVersion, "Version", 16),
			table.NewColumn(colCCConnectors, "Connectors", 24).WithStyle(right),
			table.NewColumn(colCCTasks, "Running Tasks", 16).WithStyle(right),
		}
	default:
		return []table.Column{
			table.NewColumn(colName, "Name", 35),
			table.NewColumn(colPartitions, "Partitions", 12).WithStyle(right),
			table.NewColumn(colReplication, "Replication", 12).WithStyle(right),
			table.NewColumn(colMessages, "Messages", 14).WithStyle(right),
		}
	}
}

// createResourcesTable creates and configures the resources table using bubble-table
func createResourcesTable() table.Model {
	columns := createResourceTableColumns(TopicResourceType)
	return table.New(columns).
		WithPageSize(20).
		WithHighlightedRow(0).
		Filtered(false).
		Focused(true).
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
		)
}

// truncateMiddle shortens s to maxLen by replacing the middle with "…",
// keeping a larger share of the suffix where topic names tend to differ.
// Returns s unchanged when len(s) <= maxLen or maxLen < 5.
func truncateMiddle(s string, maxLen int) string {
	runes := []rune(s)
	if maxLen < 5 || len(runes) <= maxLen {
		return s
	}
	// Give the prefix 1/3 and the suffix 2/3 of the available space so the
	// distinctive tail of topic names is always visible.
	ellipsis := "…"         // single rune, width 1
	available := maxLen - 1 // 1 for the ellipsis
	prefixLen := available / 3
	suffixLen := available - prefixLen
	return string(runes[:prefixLen]) + ellipsis + string(runes[len(runes)-suffixLen:])
}

// convertItemsToRows converts resource items to bubble-table rows.
// nameMaxWidth > 0 applies middle-truncation to the Name cell so long topic
// names show both the common prefix and the distinctive suffix.
// Pass 0 to skip truncation (e.g. when building a cache of all rows).
// loadingFrame is the current spinner frame shown for topics whose message
// count has not yet been fetched (messageCount < 0). Pass "" to use the
// static "…" placeholder.
func convertItemsToRows(items []interface{}, searchQuery string, nameMaxWidth int) []table.Row {
	return convertItemsToRowsWithSpinner(items, searchQuery, nameMaxWidth, "")
}

func convertItemsToRowsWithSpinner(items []interface{}, searchQuery string, nameMaxWidth int, loadingFrame string) []table.Row {
	rows := make([]table.Row, 0, len(items))
	placeholder := loadingFrame
	if placeholder == "" {
		placeholder = "…"
	}

	for _, item := range items {
		// Brokers use a dedicated six-column layout with styled ISR/skew cells.
		if bri, ok := brokerItemFrom(item); ok {
			rows = append(rows, table.NewRow(brokerRowData(bri, placeholder)))
			continue
		}

		// Consumer groups also use a dedicated six-column layout.
		if cgri, sq, ok := groupItemFrom(item); ok {
			rows = append(rows, table.NewRow(groupRowData(cgri, sq, placeholder)))
			continue
		}

		// Topics use a dedicated six-column layout with OSR/size cells.
		if tri, sq, ok := topicItemFrom(item); ok {
			effQuery := sq
			if effQuery == "" {
				effQuery = searchQuery
			}
			rows = append(rows, table.NewRow(topicRowData(tri, effQuery, placeholder, nameMaxWidth)))
			continue
		}

		// ACLs use a dedicated six-column layout with pattern/permission badges.
		if ari, sq, ok := aclItemFrom(item); ok {
			effQuery := sq
			if effQuery == "" {
				effQuery = searchQuery
			}
			rows = append(rows, table.NewRow(aclRowData(ari, effQuery)))
			continue
		}

		// Client quotas use a dedicated four-column layout.
		if qri, ok := quotaItemFrom(item); ok {
			rows = append(rows, table.NewRow(quotaRowData(qri)))
			continue
		}

		// Connectors use a dedicated eight-column layout with a coloured state cell.
		if ci, sq, ok := connectorItemFrom(item); ok {
			effQuery := sq
			if effQuery == "" {
				effQuery = searchQuery
			}
			rows = append(rows, table.NewRow(connectorRowData(ci, effQuery)))
			continue
		}

		// Connect clusters use a dedicated four-column layout.
		if cci, ok := connectClusterItemFrom(item); ok {
			rows = append(rows, table.NewRow(connectClusterRowData(cci)))
			continue
		}

		var name, partitions, replication, details string

		switch i := item.(type) {
		case shared.ResourceListItem:
			name = i.ResourceItem.GetID()
			itemDetails := i.ResourceItem.GetDetails()
			if p, ok := itemDetails["Partitions"]; ok {
				if p == "…" {
					partitions = placeholder
				} else {
					partitions = p
				}
			} else if p, ok := itemDetails["Brokers"]; ok {
				partitions = p
			} else if p, ok := itemDetails["Resource"]; ok {
				partitions = p
			} else if v, ok := itemDetails["Version"]; ok {
				if v == "…" {
					partitions = placeholder
				} else {
					partitions = "v" + v
				}
			} else {
				partitions = "-"
			}
			if r, ok := itemDetails["Replication Factor"]; ok {
				if r == "…" {
					replication = placeholder
				} else {
					replication = r
				}
			} else if r, ok := itemDetails["State"]; ok {
				replication = r
			} else if r, ok := itemDetails["Operation"]; ok {
				replication = r
			} else if id, ok := itemDetails["ID"]; ok {
				if id == "…" {
					replication = placeholder
				} else {
					replication = "id:" + id
				}
			} else {
				replication = "-"
			}
			if d, ok := itemDetails["Consumers"]; ok {
				details = d
			} else if d, ok := itemDetails["Permission"]; ok {
				details = d
			} else if d, ok := itemDetails["Message Count"]; ok {
				if d == "…" {
					details = placeholder
				} else {
					details = d
				}
			} else if d, ok := itemDetails["Type"]; ok {
				if d == "…" {
					details = placeholder
				} else {
					details = d
				}
			} else {
				details = "-"
			}
		case TopicItem:
			name = i.name
			partitions = fmt.Sprintf("%d", i.topic.NumPartitions)
			replication = fmt.Sprintf("%d", i.topic.ReplicationFactor)
			details = fmt.Sprintf("%d configs", len(i.topic.ConfigEntries))
		case shared.HighlightedResourceListItem:
			name = i.ResourceItem.GetID()
			itemDetails := i.ResourceItem.GetDetails()
			if p, ok := itemDetails["Partitions"]; ok {
				if p == "…" {
					partitions = placeholder
				} else {
					partitions = p
				}
			} else if p, ok := itemDetails["Brokers"]; ok {
				partitions = p
			} else if p, ok := itemDetails["Resource"]; ok {
				partitions = p
			} else if v, ok := itemDetails["Version"]; ok {
				if v == "…" {
					partitions = placeholder
				} else {
					partitions = "v" + v
				}
			} else {
				partitions = "-"
			}
			if r, ok := itemDetails["Replication Factor"]; ok {
				if r == "…" {
					replication = placeholder
				} else {
					replication = r
				}
			} else if r, ok := itemDetails["State"]; ok {
				replication = r
			} else if r, ok := itemDetails["Operation"]; ok {
				replication = r
			} else if id, ok := itemDetails["ID"]; ok {
				if id == "…" {
					replication = placeholder
				} else {
					replication = "id:" + id
				}
			} else {
				replication = "-"
			}
			if d, ok := itemDetails["Consumers"]; ok {
				details = d
			} else if d, ok := itemDetails["Permission"]; ok {
				details = d
			} else if d, ok := itemDetails["Message Count"]; ok {
				if d == "…" {
					details = placeholder
				} else {
					details = d
				}
			} else if d, ok := itemDetails["Type"]; ok {
				if d == "…" {
					details = placeholder
				} else {
					details = d
				}
			} else {
				details = "-"
			}
			if i.SearchQuery != "" {
				name = shared.HighlightSearchMatches(name, i.SearchQuery)
			}
		case shared.HighlightedTopicItem:
			name = i.Name
			partitions = fmt.Sprintf("%d", i.Topic.NumPartitions)
			replication = fmt.Sprintf("%d", i.Topic.ReplicationFactor)
			details = fmt.Sprintf("%d configs", len(i.Topic.ConfigEntries))
			if i.SearchQuery != "" {
				name = shared.HighlightSearchMatches(name, i.SearchQuery)
			}
		default:
			continue
		}

		// Apply search highlighting if searchQuery is provided and not already highlighted
		if searchQuery != "" {
			switch item.(type) {
			case shared.ResourceListItem, TopicItem:
				name = shared.HighlightSearchMatches(name, searchQuery)
			}
		}

		// Middle-truncate the display name so long names show start + end.
		// Skip truncation when a search filter is active: the name already
		// contains ANSI highlight escape codes, and cutting through them at
		// a raw byte offset produces corrupted output (e.g. stray ";38;5;205m…").
		if nameMaxWidth > 0 && searchQuery == "" {
			name = truncateMiddle(name, nameMaxWidth)
		}

		// Schemas carry an extra Compatibility column (SR-13). Non-schema resources
		// have no "Compatibility" detail, so this stays empty and is ignored.
		var compat string
		if di, ok := item.(interface{ GetDetails() map[string]string }); ok {
			if c, found := di.GetDetails()["Compatibility"]; found {
				if c == "…" {
					compat = placeholder
				} else {
					compat = c
				}
			}
		} else if rli, ok := item.(shared.ResourceListItem); ok {
			if c, found := rli.ResourceItem.GetDetails()["Compatibility"]; found {
				if c == "…" {
					compat = placeholder
				} else {
					compat = c
				}
			}
		} else if hrli, ok := item.(shared.HighlightedResourceListItem); ok {
			if c, found := hrli.ResourceItem.GetDetails()["Compatibility"]; found {
				if c == "…" {
					compat = placeholder
				} else {
					compat = c
				}
			}
		}

		rows = append(rows, table.NewRow(table.RowData{
			colName:         name,
			colPartitions:   partitions,
			colReplication:  replication,
			colMessages:     details,
			colSchemaCompat: compat,
		}))
	}

	return rows
}

// sortBrokerItems sorts a slice of broker list items in place by the given
// column key ("id", "host", "port", "disk", "isr", "skew"). Items whose skew is
// absent (nil) always sort last, regardless of direction. Non-broker items are
// left in place (they sort as equal).
func sortBrokerItems(items []interface{}, col string, desc bool) {
	less := func(a, b *BrokerResourceItem) bool {
		switch col {
		case "host":
			return a.info.Host < b.info.Host
		case "port":
			return a.info.Port < b.info.Port
		case "disk":
			return a.stats.SegmentSize < b.stats.SegmentSize
		case "isr":
			return a.stats.InSyncReplicaCount < b.stats.InSyncReplicaCount
		case "skew":
			as, bs := a.stats.ReplicaSkew, b.stats.ReplicaSkew
			// nil always last
			if as == nil || bs == nil {
				if as == nil && bs == nil {
					return false
				}
				// The absent one is "greater"; when descending we still want it
				// last, so bias it independent of direction (handled by caller flip).
				return as != nil
			}
			return *as < *bs
		default: // "id"
			return a.info.ID < b.info.ID
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		ai, aok := brokerItemFrom(items[i])
		bj, bok := brokerItemFrom(items[j])
		if !aok || !bok {
			return false
		}
		// Absent skews are pinned last for both directions.
		if col == "skew" {
			an, bn := ai.stats.ReplicaSkew == nil, bj.stats.ReplicaSkew == nil
			if an != bn {
				return bn // non-nil (bn==true means j is nil) sorts before nil
			}
			if an && bn {
				return false
			}
		}
		if desc {
			return less(bj, ai)
		}
		return less(ai, bj)
	})
}

// brokerItemFrom unwraps a broker resource item from the list-item wrappers.
func brokerItemFrom(item interface{}) (*BrokerResourceItem, bool) {
	switch v := item.(type) {
	case shared.ResourceListItem:
		bri, ok := v.ResourceItem.(*BrokerResourceItem)
		return bri, ok
	case shared.HighlightedResourceListItem:
		bri, ok := v.ResourceItem.(*BrokerResourceItem)
		return bri, ok
	case *BrokerResourceItem:
		return v, true
	default:
		return nil, false
	}
}

// brokerRowData builds a bubble-table row for a broker, styling the ISR cell in
// the alert style when under-replicated and the skew cell by severity.
func brokerRowData(b *BrokerResourceItem, placeholder string) table.RowData {
	port := strconv.FormatInt(int64(b.info.Port), 10)
	if !b.hasStats {
		return table.RowData{
			colBrokerID:   b.idCell(),
			colBrokerHost: b.info.Host,
			colBrokerPort: port,
			colBrokerDisk: placeholder,
			colBrokerISR:  placeholder,
			colBrokerSkew: placeholder,
		}
	}

	disk := shared.FormatDiskUsage(b.stats.SegmentSize, b.stats.SegmentCount)

	isrText, alert := shared.FormatISR(b.stats.InSyncReplicaCount, b.stats.ReplicaCount)
	isrCell := isrText
	if alert && isrText != "" {
		isrCell = lipgloss.NewStyle().Foreground(stylesPkg.Error).Render(isrText)
	}

	skewText := shared.FormatSkew(b.stats.ReplicaSkew)
	skewCell := skewText
	switch shared.SkewSeverity(b.stats.ReplicaSkew) {
	case shared.SkewError:
		skewCell = lipgloss.NewStyle().Foreground(stylesPkg.Error).Render(skewText)
	case shared.SkewWarning:
		skewCell = lipgloss.NewStyle().Foreground(stylesPkg.Warning).Render(skewText)
	}

	return table.RowData{
		colBrokerID:   b.idCell(),
		colBrokerHost: b.info.Host,
		colBrokerPort: port,
		colBrokerDisk: disk,
		colBrokerISR:  isrCell,
		colBrokerSkew: skewCell,
	}
}

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
	resourcePickerMode bool
	resourcePickerInput string

	// Data storage
	allRows       []table.Row
	allItems      []interface{}
	filteredRows  []table.Row
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

	// nameColumnWidth tracks the rendered width of the Name column so that
	// middle-truncation is proportional to the actual terminal width.
	// Updated every RenderContent call.
	nameColumnWidth int

	// countLoading is true while a GetTopicMessageCounts command is in flight.
	// The spinner animates the placeholder cells during this window.
	countLoading      bool
	countSpinner      spinner.Model
	countSpinnerFrame string // current animated frame used as placeholder in table cells

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
		allRows:         []table.Row{},
		allItems:        []interface{}{},
		filteredRows:    []table.Row{},
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
		allRows:         []table.Row{},
		allItems:        []interface{}{},
		filteredRows:    []table.Row{},
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

	if k.common != nil && k.common.Layout != nil {
		// Use layout system
		layout := k.common.Layout
		tableHeight = layout.GetAvailableHeight() - 3 // Reserve space for padding
		tableWidth = layout.GetAvailableWidth() - 2
	} else {
		// Fallback to ad-hoc calculation
		tableHeight = height - 6
		tableWidth = width - 4
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
	// Name column is 35/99 of total defined column width; reserve 6 chars for borders.
	const totalDefinedWidth = 35 + 12 + 12 + 14 // 73
	k.nameColumnWidth = (tableWidth - 6) * 35 / totalDefinedWidth
	if k.nameColumnWidth < 20 {
		k.nameColumnWidth = 20
	}
	k.resourcesTable = k.resourcesTable.
		WithPageSize(tableHeight).
		WithTargetWidth(tableWidth).
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
	if k.resourcePickerMode {
		return k.renderResourcePicker(tableWidth)
	}

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
		// The create/clone form swallows all key input while visible.
		if k.showTopicForm && k.topicForm != nil {
			cmd, _ := k.topicForm.Update(msg)
			return cmd
		}
		// ACL / quota overlay forms swallow all key input while visible.
		if f := k.activeOverlayForm(); f != nil {
			cmd, _ := f.Update(msg)
			return cmd
		}

		// Resource picker mode takes precedence over everything else (UI-8).
		if k.resourcePickerMode {
			switch msg.String() {
			case "esc", "escape":
				k.resourcePickerMode = false
				k.resourcePickerInput = ""
				return nil
			case "enter":
				return k.commitResourcePicker(k.resourcePickerInput)
			case "tab":
				if best := k.bestResourceMatch(k.resourcePickerInput); best != "" {
					k.resourcePickerInput = best
				}
				return nil
			case "backspace":
				if len(k.resourcePickerInput) > 0 {
					k.resourcePickerInput = k.resourcePickerInput[:len(k.resourcePickerInput)-1]
				}
				return nil
			default:
				if len(msg.Runes) > 0 {
					k.resourcePickerInput += string(msg.Runes)
				}
				return nil
			}
		}

		// Handle search mode first
		if k.searchMode {
			switch msg.String() {
			case "escape":
				k.searchMode = false
				k.clearSearch()
				return nil
			case "enter":
				// Confirm search and exit search mode
				k.searchMode = false
				return nil
			case "backspace":
				if len(k.currentFilter) > 0 {
					k.currentFilter = k.currentFilter[:len(k.currentFilter)-1]
					k.handleSearch(k.currentFilter)
				}
				return nil
			default:
				// Handle typing in search mode
				if len(msg.Runes) > 0 {
					k.currentFilter += string(msg.Runes)
					k.handleSearch(k.currentFilter)
				}
				return nil
			}
		}

		// Handle normal mode keys
		switch msg.String() {
		case "/":
			k.searchMode = true
			k.currentFilter = "" // Reset filter when starting search
			return nil
		case ":":
			// Open the resource picker (UI-8).
			return func() tea.Msg {
				return StartResourceSwitchingMsg{}
			}
		case "enter":
			// Handle resource selection
			return k.handleResourceSelection()
		// Logical page navigation — handled here so bubble-table doesn't
		// treat them as visual page changes within the current 50-item slice.
		case "pgup", "left", "h":
			if k.pagination.PrevPage() {
				k.updateTableForCurrentPageAndReset()
				cmds = append(cmds, k.loadPageDetails())
			}
			return tea.Batch(cmds...)
		case "pgdown", "right", "l":
			if k.pagination.NextPage() {
				k.updateTableForCurrentPageAndReset()
				cmds = append(cmds, k.loadPageDetails())
			}
			return tea.Batch(cmds...)
		case "home", "g":
			if !k.pagination.OnFirstPage() {
				k.pagination.FirstPage()
				k.updateTableForCurrentPageAndReset()
				cmds = append(cmds, k.loadPageDetails())
			}
			return tea.Batch(cmds...)
		case "end", "G":
			if !k.pagination.OnLastPage() {
				k.pagination.LastPage()
				k.updateTableForCurrentPageAndReset()
				cmds = append(cmds, k.loadPageDetails())
			}
			return tea.Batch(cmds...)
		default:
			if key.Matches(msg, keybindings.DefaultMainKeyMap().Copy) {
				return k.handleCopyRow()
			}
			if k.isBrokerResource() {
				switch msg.String() {
				case "s":
					k.cycleBrokerSortColumn()
					return nil
				case "S":
					k.toggleBrokerSortDir()
					return nil
				case "ctrl+e":
					return k.exportBrokersCSV()
				}
			}
			if k.isGroupResource() {
				switch msg.String() {
				case "s":
					k.cycleGroupSortColumn()
					return nil
				case "S":
					k.toggleGroupSortDir()
					return nil
				case "f":
					k.cycleGroupStateFilter()
					return nil
				case "ctrl+d":
					return k.deleteSelectedGroup()
				case "ctrl+e":
					return k.exportGroupsCSV()
				}
			}
			if k.isTopicResource() {
				switch msg.String() {
				case "s":
					k.cycleTopicSortColumn()
					return nil
				case "S":
					k.toggleTopicSortDir()
					return nil
				case "i":
					return k.toggleHideInternal()
				case "ctrl+e":
					return k.exportTopicsCSV()
				case "n":
					return k.openCreateTopicForm()
				case "ctrl+n":
					return k.openCloneTopicForm()
				case "ctrl+d":
					return k.deleteSelectedTopics()
				case "ctrl+r":
					return k.recreateSelectedTopic()
				case "ctrl+p":
					return k.purgeSelectedTopics()
				case " ":
					k.toggleTopicSelection()
					return nil
				case "ctrl+a":
					k.selectAllVisibleTopics()
					return nil
				case "esc":
					if len(k.selected) > 0 {
						k.clearTopicSelection()
						return nil
					}
				}
			}
			if k.isACLResource() {
				switch msg.String() {
				case "f":
					return k.cycleACLResourceTypeFilter()
				case "p":
					return k.cycleACLPatternFilter()
				case "n":
					return k.openCreateACLForm()
				case "ctrl+d":
					return k.deleteSelectedACL()
				case "ctrl+e":
					return k.exportACLsCSV()
				case "ctrl+i":
					return k.openACLSyncForm()
				}
			}
			if k.isQuotaResource() {
				switch msg.String() {
				case "n":
					return k.openQuotaForm(false)
				case "e":
					return k.openQuotaForm(true)
				case "ctrl+d":
					return k.deleteSelectedQuota()
				}
			}
			if k.isConnectorResource() {
				switch msg.String() {
				case "n":
					return k.openCreateConnectorForm()
				case "ctrl+e":
					return k.exportConnectorsCSV()
				}
			}
			if k.isConnectClusterResource() {
				if msg.String() == "ctrl+e" {
					return k.exportConnectClustersCSV()
				}
			}
		}

		// Delegate remaining keys (↑/↓ row navigation) to bubble-table
		var cmd tea.Cmd
		k.resourcesTable, cmd = k.resourcesTable.Update(msg)
		cmds = append(cmds, cmd)

	case SearchTopicsMsg:
		k.handleSearch(string(msg))

	case ClearSearchMsg:
		k.clearSearch()

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
		// Open the capability-filtered resource picker (UI-8) instead of the
		// old blind cycle.
		k.resourcePickerMode = true
		k.resourcePickerInput = ""
		return nil

	case TopicCountsLoadedMsg:
		k.applyTopicMessageCounts(msg)

	case TopicDetailsExtLoadedMsg:
		k.applyTopicDetailsExt(msg)

	case form.FormSubmitMsg:
		switch {
		case k.showACLForm:
			return k.handleACLFormSubmit(msg.Values)
		case k.showACLSyncForm:
			k.showACLSyncForm = false
			k.aclSyncForm = nil
			return k.handleACLSyncSubmit(msg.Values["path"])
		case k.showQuotaForm:
			return k.handleQuotaFormSubmit(msg.Values)
		case k.showConnectForm:
			return k.handleConnectorFormSubmit(msg.Values)
		default:
			return k.handleTopicFormSubmit(msg.Values)
		}

	case form.FormCancelMsg:
		k.showTopicForm = false
		k.topicForm = nil
		k.showACLForm = false
		k.aclForm = nil
		k.showACLSyncForm = false
		k.aclSyncForm = nil
		k.showQuotaForm = false
		k.quotaForm = nil
		k.showConnectForm = false
		k.connectForm = nil

	case connectorCreatedMsg:
		return k.handleConnectorCreated(msg)

	case topicCreatedMsg:
		if msg.err != nil {
			// Keep the form open so the user can correct the input.
			return core.NotifyError("Create topic failed", msg.err)
		}
		k.showTopicForm = false
		k.topicForm = nil
		return tea.Batch(
			core.NewNotification(core.StatusSuccess, "Topic created", msg.name),
			k.loadCurrentResource(),
			func() tea.Msg {
				return NavigateToResourceDetailMsg{ResourceType: TopicResourceType, ResourceID: msg.name}
			},
		)

	case topicDeletedMsg:
		if msg.err != nil {
			return func() tea.Msg { return shared.NewUIError("delete-topic", "Delete topic failed", msg.err) }
		}
		return tea.Batch(core.NewNotification(core.StatusSuccess, "Topic deleted", msg.name), k.loadCurrentResource())

	case topicRecreatedMsg:
		if msg.err != nil {
			return func() tea.Msg { return shared.NewUIError("recreate-topic", "Recreate topic failed", msg.err) }
		}
		return tea.Batch(core.NewNotification(core.StatusSuccess, "Topic recreated", msg.name), k.loadCurrentResource())

	case topicPurgedMsg:
		if msg.err != nil {
			return func() tea.Msg { return shared.NewUIError("purge-topic", "Clear messages failed", msg.err) }
		}
		return tea.Batch(core.NewNotification(core.StatusSuccess, "Messages cleared", msg.name), k.loadCurrentResource())

	case topicBatchResultMsg:
		k.clearTopicSelection()
		if len(msg.failures) > 0 {
			summary := fmt.Errorf("%d of %d topics failed: %s", len(msg.failures), msg.total, strings.Join(msg.failures, "; "))
			return tea.Batch(
				func() tea.Msg {
					return shared.NewUIError("batch-"+msg.action, "Batch "+msg.action+" completed with errors", summary)
				},
				k.loadCurrentResource(),
			)
		}
		return tea.Batch(
			core.NewNotification(core.StatusSuccess, "Batch "+msg.action+" complete", fmt.Sprintf("%d topics", msg.total)),
			k.loadCurrentResource(),
		)

	case SchemaDetailsLoadedMsg:
		k.applySchemaDetails(msg)

	case BrokerStatsLoadedMsg:
		k.applyBrokerStats(msg)

	case ConsumerGroupDetailsLoadedMsg:
		k.applyGroupDetails(msg)

	case groupDeletedMsg:
		if msg.err != nil {
			return func() tea.Msg { return shared.NewUIError("delete-group", "Delete consumer group failed", msg.err) }
		}
		return tea.Batch(
			core.NewNotification(core.StatusSuccess, "Consumer group deleted", msg.groupID),
			k.loadCurrentResource(),
		)

	case aclDeletedMsg:
		if msg.err != nil {
			return func() tea.Msg { return shared.NewUIError("delete-acl", "Delete ACL failed", msg.err) }
		}
		return tea.Batch(core.NewNotification(core.StatusSuccess, "ACL deleted", msg.summary), k.loadCurrentResource())

	case aclCreatedMsg:
		if msg.err != nil {
			// Validation/expansion error — keep the form open to correct input.
			return core.NotifyError("Create ACL failed", msg.err)
		}
		k.showACLForm = false
		k.aclForm = nil
		if len(msg.failures) > 0 {
			summary := fmt.Errorf("%d created, %d failed: %s", msg.created, len(msg.failures), strings.Join(msg.failures, "; "))
			return tea.Batch(
				func() tea.Msg { return shared.NewUIError("create-acl", "Some ACL bindings failed", summary) },
				k.loadCurrentResource(),
			)
		}
		return tea.Batch(
			core.NewNotification(core.StatusSuccess, "ACLs created", fmt.Sprintf("%d binding(s)", msg.created)),
			k.loadCurrentResource(),
		)

	case aclSyncedMsg:
		if msg.err != nil {
			return func() tea.Msg { return shared.NewUIError("sync-acl", "ACL sync failed", msg.err) }
		}
		return tea.Batch(
			core.NewNotification(core.StatusSuccess, "ACLs synced", fmt.Sprintf("%d created, %d deleted", msg.created, msg.deleted)),
			k.loadCurrentResource(),
		)

	case quotaAlteredMsg:
		if msg.err != nil {
			// Validation error — keep the form open to correct input.
			return core.NotifyError("Quota update failed", msg.err)
		}
		k.showQuotaForm = false
		k.quotaForm = nil
		k.quotaEditEntity = nil
		return tea.Batch(
			core.NewNotification(core.StatusSuccess, "Client quota "+msg.action, ""),
			k.loadCurrentResource(),
		)

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
		// Scroll wheel navigates within the current page.
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			idx := k.resourcesTable.GetHighlightedRowIndex()
			if idx > 0 {
				k.resourcesTable = k.resourcesTable.WithHighlightedRow(idx - 1)
			}
		case tea.MouseButtonWheelDown:
			activeItems := k.allItems
			if k.isFiltered {
				activeItems = k.filteredItems
			}
			pageItems := k.pagination.GetCurrentPageItems(activeItems)
			idx := k.resourcesTable.GetHighlightedRowIndex()
			if idx < len(pageItems)-1 {
				k.resourcesTable = k.resourcesTable.WithHighlightedRow(idx + 1)
			}
		case tea.MouseButtonLeft:
			// Only handle clicks that land inside the table zone.
			z := zone.Get("resource-table")
			if z.InBounds(msg) {
				_, relY := z.Pos(msg)
				// The table renders: border + header + separator = 3 lines before first data row.
				const headerLines = 3
				pageLocalRow := relY - headerLines
				if pageLocalRow >= 0 {
					activeItems := k.allItems
					if k.isFiltered {
						activeItems = k.filteredItems
					}
					pageItems := k.pagination.GetCurrentPageItems(activeItems)
					if pageLocalRow < len(pageItems) {
						k.resourcesTable = k.resourcesTable.WithHighlightedRow(pageLocalRow)
					}
				}
			}
		}
	}

	return tea.Batch(cmds...)
}

func (k *KafuiContentProvider) InitContent() tea.Cmd {
	// countSpinner.Tick animates the shared initial-load spinner (UI-12); it
	// self-perpetuates via the spinner.TickMsg handler while loading.
	return tea.Batch(k.loadCurrentResource(), k.breadcrumbCmd(), k.countSpinner.Tick)
}

// breadcrumbCmd returns a Cmd that sends a BreadcrumbUpdateMsg reflecting
// the currently active resource type (e.g. ["Kafka UI", "Topics"]).
func (k *KafuiContentProvider) breadcrumbCmd() tea.Cmd {
	label := "Kafka UI"
	if k.currentResource != nil {
		switch k.currentResource.GetType() {
		case TopicResourceType:
			label = "Topics"
		case ConsumerGroupResourceType:
			label = "Consumer Groups"
		case SchemaResourceType:
			label = "Schemas"
		case ContextResourceType:
			label = "Contexts"
		case ACLResourceType:
			label = "ACLs"
		case BrokerResourceType:
			label = "Brokers"
		case QuotaResourceType:
			label = "Quotas"
		}
	}
	items := []string{"Kafka UI", label}
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
	return k.searchMode || k.resourcePickerMode || k.showTopicForm || k.activeOverlayForm() != nil
}

// GetContentSize returns the estimated content size for scrollbar calculation
func (k *KafuiContentProvider) GetContentSize(width int) int {
	// Estimate based on table rows plus header
	rowCount := len(k.allRows)
	if rowCount == 0 {
		return 5 // Default for empty/loading states
	}
	// Add header lines and account for search bar
	return rowCount + 5
}

// Helper methods

func (k *KafuiContentProvider) handleSearch(query string) {
	k.currentFilter = query
	k.applyFilters(true)
}

// reapplyFilter re-runs the active filter after underlying data changed
// (e.g. timer refresh, async detail load) preserving the current page.
func (k *KafuiContentProvider) reapplyFilter() {
	k.applyFilters(false)
}

func (k *KafuiContentProvider) clearSearch() {
	k.currentFilter = ""
	k.applyFilters(true)
}

// applyFilters rebuilds the filtered view from the active search query and (for
// the consumer-groups resource) the active state filter. The two predicates
// compose: a row must match both. When neither is active, the view falls back to
// the unfiltered list. reset jumps to page 0/row 0; otherwise the current page is
// preserved (clamped to the new total).
func (k *KafuiContentProvider) applyFilters(reset bool) {
	if k.currentFilter == "" && !k.hasStateFilter() && !k.hasVisibilityFilter() {
		k.isFiltered = false
		k.filteredItems = []interface{}{}
		k.filteredRows = []table.Row{}
		if reset {
			k.pagination.Page = 0
		}
		k.pagination.SetTotalItems(len(k.allItems))
		if reset {
			k.updateTableForCurrentPageAndReset()
		} else {
			k.clampPage()
			k.updateTableForCurrentPage()
		}
		return
	}

	filtered := make([]interface{}, 0, len(k.allItems))
	for _, item := range k.allItems {
		if k.currentFilter != "" && !k.itemMatchesQuery(item, k.currentFilter) {
			continue
		}
		if k.hasStateFilter() && !k.groupItemMatchesState(item, k.groupStateFilter) {
			continue
		}
		if k.hasVisibilityFilter() && k.isInternalItem(item) {
			continue
		}
		it := item
		if k.currentFilter != "" {
			it = k.createHighlightedItem(item, k.currentFilter)
		}
		filtered = append(filtered, it)
	}

	k.filteredItems = filtered
	k.filteredRows = convertItemsToRows(filtered, k.currentFilter, 0)
	k.isFiltered = true
	k.pagination.SetTotalItems(len(filtered))
	if reset {
		k.pagination.Page = 0
		k.updateTableForCurrentPageAndReset()
	} else {
		k.clampPage()
		k.updateTableForCurrentPage()
	}
}

// clampPage keeps the current pagination page within [0, TotalPages).
func (k *KafuiContentProvider) clampPage() {
	if k.pagination.Page >= k.pagination.TotalPages {
		k.pagination.Page = k.pagination.TotalPages - 1
	}
	if k.pagination.Page < 0 {
		k.pagination.Page = 0
	}
}

// updateTableForCurrentPage rebuilds the table rows from the current pagination
// page, preserving the current highlighted row (clamped to the new page size).
func (k *KafuiContentProvider) updateTableForCurrentPage() {
	activeItems := k.allItems
	if k.isFiltered {
		activeItems = k.filteredItems
	}
	pageItems := k.pagination.GetCurrentPageItems(activeItems)
	pageRows := convertItemsToRowsWithSpinner(pageItems, k.currentFilter, k.nameColumnWidth, k.countSpinnerFrame)

	// Preserve current cursor position, clamped to the new page length.
	row := k.resourcesTable.GetHighlightedRowIndex()
	if len(pageRows) == 0 {
		row = 0
	} else if row >= len(pageRows) {
		row = len(pageRows) - 1
	}
	k.resourcesTable = k.resourcesTable.WithRows(pageRows).WithHighlightedRow(row)
}

// updateTableForCurrentPageAndReset rebuilds the table rows and resets the
// highlighted row to 0. Use this when the dataset changes meaningfully
// (page navigation, resource switch, search change).
func (k *KafuiContentProvider) updateTableForCurrentPageAndReset() {
	activeItems := k.allItems
	if k.isFiltered {
		activeItems = k.filteredItems
	}
	pageItems := k.pagination.GetCurrentPageItems(activeItems)
	pageRows := convertItemsToRowsWithSpinner(pageItems, k.currentFilter, k.nameColumnWidth, k.countSpinnerFrame)
	k.resourcesTable = k.resourcesTable.WithRows(pageRows).WithHighlightedRow(0)
}

func (k *KafuiContentProvider) switchResource(msg SwitchResourceMsg) {
	k.currentResource = k.resourceManager.GetResource(ResourceType(msg))
	k.pendingReset = true
	// Reset per-resource sort/filter state so it doesn't leak across resources.
	k.groupSortCol = -1
	k.groupSortDesc = false
	k.groupStateFilter = ""
	// Reset topic-specific state (sort + multi-selection + any open form).
	// hideInternal is a persisted preference and deliberately preserved.
	k.topicSortCol = -1
	k.topicSortDesc = false
	k.selected = map[string]bool{}
	k.showTopicForm = false
	k.topicForm = nil
	k.clearSearch()
	// Clear stale data immediately so the loading screen shows while the
	// new resource's data is being fetched (avoids showing the previous
	// resource's rows until the new data arrives).
	k.allItems = []interface{}{}
	k.allRows = []table.Row{}
	k.filteredItems = []interface{}{}
	k.filteredRows = []table.Row{}
	// Update column headers to match the new resource type.
	cols := createResourceTableColumns(ResourceType(msg))
	k.resourcesTable = k.resourcesTable.WithColumns(cols).WithRows(nil)
}

func (k *KafuiContentProvider) loadCurrentResource() tea.Cmd {
	k.loading = true
	k.error = nil

	// Capture the resource reference now so the goroutine always calls
	// GetData() and tags the message with the type that was active when
	// loadCurrentResource was invoked, regardless of later resource switches.
	resource := k.currentResource

	// For topics with an empty list (first load / after resource switch), use
	// two-phase loading: names first (fast) then full details async.
	if resource.GetType() == TopicResourceType && len(k.allItems) == 0 {
		return k.loadTopicsQuick()
	}

	fetch := func() tea.Msg {
		items, err := resource.GetData()
		if err != nil {
			return ErrorMsg(err)
		}

		// Convert resource items to interface slice
		interfaceItems := make([]interface{}, 0, len(items))
		for _, item := range items {
			interfaceItems = append(interfaceItems, shared.ResourceListItem{
				ResourceItem: item,
			})
		}

		return CurrentResourceListMsg{
			ResourceType: resource.GetType(),
			Items:        interfaceItems,
		}
	}
	return fetch
}

// loadTopicsQuick fetches only topic names and returns stub rows immediately.
// Each stub has partitions=-1 and replicationFactor=-1 (shown as "…").
// loadTopicDetails() is then fired to fill in the real values asynchronously.
func (k *KafuiContentProvider) loadTopicsQuick() tea.Cmd {
	ds := k.dataSource
	return func() tea.Msg {
		names, err := ds.GetTopicNames()
		if err != nil {
			return ErrorMsg(err)
		}
		items := make([]interface{}, 0, len(names))
		for _, name := range names {
			items = append(items, shared.ResourceListItem{
				ResourceItem: &TopicResourceItem{
					id:                name,
					partitions:        -1, // filled by loadTopicDetails
					replicationFactor: -1,
					messageCount:      -1,
					outOfSync:         -1, // filled by loadTopicDetailsExt
					size:              -1,
					isInternal:        isInternalTopicName(name),
				},
			})
		}
		shared.Log.Info("loadTopicsQuick: names loaded", "count", len(names))
		return CurrentResourceListMsg{
			ResourceType: TopicResourceType,
			Items:        items,
		}
	}
}

// loadTopicDetails fetches the full topic map (partitions, replication factor)
// and returns TopicDetailsLoadedMsg so handleContentUpdate can update stubs.
func (k *KafuiContentProvider) loadTopicDetails() tea.Cmd {
	ds := k.dataSource
	k.detailsLoading = true
	return tea.Batch(k.countSpinner.Tick, func() tea.Msg {
		topics, err := ds.GetTopics()
		if err != nil {
			shared.Log.Error("loadTopicDetails: GetTopics failed", "err", err)
			return TopicDetailsLoadedMsg(nil) // leave stubs visible
		}
		shared.Log.Info("loadTopicDetails: details loaded", "count", len(topics))
		return TopicDetailsLoadedMsg(topics)
	})
}

// handleCopyRow copies the highlighted row's values to the system clipboard
// and briefly shows "📋 Copied!" in the table footer.
func (k *KafuiContentProvider) handleCopyRow() tea.Cmd {
	rows := k.resourcesTable.GetVisibleRows()
	idx := k.resourcesTable.GetHighlightedRowIndex()
	if idx < 0 || idx >= len(rows) {
		return nil
	}
	row := rows[idx]

	// Build a tab-separated string of all column values. Topics use their own
	// column keys; the shared keys are absent for those rows (and vice versa),
	// so listing both is safe.
	var parts []string
	for _, col := range []string{colName, colPartitions, colReplication, colMessages, colTopicName, colTopicPartitions, colTopicReplication, colTopicMessages, colTopicOSR, colTopicSize} {
		if v, ok := row.Data[col]; ok {
			parts = append(parts, fmt.Sprintf("%v", v))
		}
	}
	text := strings.Join(parts, "\t")

	if err := clipboard.WriteAll(text); err != nil {
		shared.Log.Error("clipboard copy failed", "err", err)
		k.clipboardMsg = "⚠ Copy failed"
	} else {
		k.clipboardMsg = "📋 Copied!"
	}

	// Clear the feedback after 2 seconds.
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return ClearClipboardFeedbackMsg{}
	})
}

// loadTopicMessageCounts returns a Cmd that fetches message counts for the
// topics currently visible on screen (current pagination page only).
// Fetching all topics at once would be extremely slow on large clusters
// (2000+ topics × N partitions = thousands of broker round-trips).
func (k *KafuiContentProvider) loadTopicMessageCounts() tea.Cmd {
	// Only query the topics on the current page.
	activeItems := k.allItems
	if k.isFiltered {
		activeItems = k.filteredItems
	}
	pageItems := k.pagination.GetCurrentPageItems(activeItems)

	input := make(map[string]int32, len(pageItems))
	for _, item := range pageItems {
		if rli, ok := item.(shared.ResourceListItem); ok {
			if tri, ok := rli.ResourceItem.(*TopicResourceItem); ok && tri.partitions > 0 {
				input[tri.id] = tri.partitions
			}
		}
	}
	if len(input) == 0 {
		shared.Log.Info("loadTopicMessageCounts: no visible topics, skipping")
		return nil
	}

	shared.Log.Info("loadTopicMessageCounts: starting", "topics", len(input))
	k.countLoading = true
	ds := k.dataSource
	return tea.Batch(k.countSpinner.Tick, func() tea.Msg {
		counts, err := ds.GetTopicMessageCounts(input)
		if err != nil {
			shared.Log.Error("loadTopicMessageCounts: GetTopicMessageCounts failed", "err", err)
			return TopicCountsLoadedMsg(map[string]int64{})
		}
		shared.Log.Info("loadTopicMessageCounts: got counts", "topics", len(counts))
		return TopicCountsLoadedMsg(counts)
	})
}

// applyTopicMessageCounts updates messageCount on all TopicResourceItem entries
// and rebuilds the display rows so the Details column refreshes.
func (k *KafuiContentProvider) applyTopicMessageCounts(counts TopicCountsLoadedMsg) {
	shared.Log.Info("applyTopicMessageCounts", "counts", len(counts))
	updated := 0
	for _, item := range k.allItems {
		if rli, ok := item.(shared.ResourceListItem); ok {
			if tri, ok := rli.ResourceItem.(*TopicResourceItem); ok {
				if count, found := counts[tri.id]; found {
					tri.messageCount = count
					updated++
				}
			}
		}
	}
	shared.Log.Info("applyTopicMessageCounts: done", "updated", updated)
	k.countLoading = false
	k.countSpinnerFrame = ""
	// Rebuild rows so the spinner placeholder is replaced by the real count.
	k.allRows = convertItemsToRows(k.allItems, "", k.nameColumnWidth)
	if k.isFiltered {
		k.reapplyFilter()
	} else {
		k.updateTableForCurrentPage()
	}
}

// hasTopicStubs returns true if any topic item in allItems has partitions == -1
// (i.e. was created by loadTopicsQuick and hasn't been filled by loadTopicDetails yet).
func (k *KafuiContentProvider) hasTopicStubs() bool {
	for _, item := range k.allItems {
		if rli, ok := item.(shared.ResourceListItem); ok {
			if tri, ok := rli.ResourceItem.(*TopicResourceItem); ok {
				if tri.partitions < 0 {
					return true
				}
			}
		}
	}
	return false
}

// applyTopicDetails updates every TopicResourceItem stub in allItems with the
// real partition/replication data from GetTopics(), then rebuilds the table.
func (k *KafuiContentProvider) applyTopicDetails(details TopicDetailsLoadedMsg) {
	k.detailsLoading = false
	if details == nil {
		// Fetch failed; leave stubs as-is so the user still sees the names.
		return
	}
	updated := 0
	for _, item := range k.allItems {
		if rli, ok := item.(shared.ResourceListItem); ok {
			if tri, ok := rli.ResourceItem.(*TopicResourceItem); ok {
				if topic, found := details[tri.id]; found {
					tri.partitions = topic.NumPartitions
					tri.replicationFactor = topic.ReplicationFactor
					tri.topic = topic
					updated++
				}
			}
		}
	}
	shared.Log.Info("applyTopicDetails: done", "updated", updated)
	k.allRows = convertItemsToRows(k.allItems, "", k.nameColumnWidth)
	if k.isFiltered {
		k.reapplyFilter()
	} else {
		k.updateTableForCurrentPage()
	}
}

// loadSchemaDetails fetches version/ID/type for the schema subjects currently
// visible on the active page (same lazy-load pattern as loadTopicMessageCounts).
func (k *KafuiContentProvider) loadSchemaDetails() tea.Cmd {
	activeItems := k.allItems
	if k.isFiltered {
		activeItems = k.filteredItems
	}
	pageItems := k.pagination.GetCurrentPageItems(activeItems)

	subjects := make([]string, 0, len(pageItems))
	for _, item := range pageItems {
		if rli, ok := item.(shared.ResourceListItem); ok {
			if sri, ok := rli.ResourceItem.(*SchemaResourceItem); ok && !sri.detailsLoaded {
				subjects = append(subjects, sri.subject)
			}
		}
	}
	if len(subjects) == 0 {
		return nil
	}

	k.countLoading = true
	ds := k.dataSource
	return tea.Batch(k.countSpinner.Tick, func() tea.Msg {
		details, err := ds.GetSchemaDetails(subjects)
		if err != nil {
			shared.Log.Error("loadSchemaDetails: GetSchemaDetails failed", "err", err)
			return SchemaDetailsLoadedMsg([]api.Schema{})
		}
		return SchemaDetailsLoadedMsg(details)
	})
}

// applySchemaDetails merges loaded version/ID/type back onto the SchemaResourceItems
// and rebuilds the display rows.
func (k *KafuiContentProvider) applySchemaDetails(msg SchemaDetailsLoadedMsg) {
	// Index by subject name for O(1) lookup.
	bySubject := make(map[string]api.Schema, len(msg))
	for _, s := range msg {
		bySubject[s.Subject] = s
	}
	for _, item := range k.allItems {
		if rli, ok := item.(shared.ResourceListItem); ok {
			if sri, ok := rli.ResourceItem.(*SchemaResourceItem); ok {
				if s, found := bySubject[sri.subject]; found {
					sri.version = s.Version
					sri.schemaID = s.ID
					schemaType := s.SchemaType
					if schemaType == "" {
						schemaType = "AVRO"
					}
					sri.schemaType = schemaType
					sri.compatibility = s.Compatibility
					sri.detailsLoaded = true
				}
			}
		}
	}
	k.countLoading = false
	k.countSpinnerFrame = ""
	k.allRows = convertItemsToRows(k.allItems, "", k.nameColumnWidth)
	if k.isFiltered {
		k.reapplyFilter()
	} else {
		k.updateTableForCurrentPage()
	}
}

// loadPageDetails triggers the right lazy-detail loader for the current resource type.
func (k *KafuiContentProvider) loadPageDetails() tea.Cmd {
	if k.currentResource == nil {
		return nil
	}
	switch k.currentResource.GetType() {
	case TopicResourceType:
		return tea.Batch(k.loadTopicMessageCounts(), k.loadTopicDetailsExt())
	case SchemaResourceType:
		return k.loadSchemaDetails()
	case BrokerResourceType:
		return k.loadBrokerStats()
	case ConsumerGroupResourceType:
		return k.loadGroupDetails()
	}
	return nil
}

// isBrokerResource reports whether the brokers resource is currently active.
func (k *KafuiContentProvider) isBrokerResource() bool {
	return k.currentResource != nil && k.currentResource.GetType() == BrokerResourceType
}

// loadBrokerStats fetches per-broker statistics + summary asynchronously and
// returns a BrokerStatsLoadedMsg (second phase of the two-phase broker load).
func (k *KafuiContentProvider) loadBrokerStats() tea.Cmd {
	ds := k.dataSource
	return func() tea.Msg {
		stats, summary, err := ds.GetBrokerStats()
		if err != nil {
			shared.Log.Error("loadBrokerStats: GetBrokerStats failed", "err", err)
			return BrokerStatsLoadedMsg{Stats: map[int32]api.BrokerStats{}}
		}
		return BrokerStatsLoadedMsg{Stats: stats, Summary: summary}
	}
}

// applyBrokerStats merges enriched stats onto the broker items and rebuilds rows,
// preserving the active sort order.
func (k *KafuiContentProvider) applyBrokerStats(msg BrokerStatsLoadedMsg) {
	for _, item := range k.allItems {
		if bri, ok := brokerItemFrom(item); ok {
			if s, found := msg.Stats[bri.info.ID]; found {
				bri.SetStats(s)
			}
		}
	}
	k.applyBrokerSort()
	k.allRows = convertItemsToRows(k.allItems, "", 0)
	if k.isFiltered {
		k.reapplyFilter()
	} else {
		k.updateTableForCurrentPage()
	}
}

// brokerSortColumns maps the visible column order to a sort key.
var brokerSortColumns = []string{"id", "host", "port", "disk", "isr", "skew"}

// cycleBrokerSortColumn advances the sort column (wrapping) and re-sorts.
func (k *KafuiContentProvider) cycleBrokerSortColumn() {
	k.brokerSortCol = (k.brokerSortCol + 1) % len(brokerSortColumns)
	k.applyBrokerSortAndRefresh()
}

// toggleBrokerSortDir flips the sort direction and re-sorts.
func (k *KafuiContentProvider) toggleBrokerSortDir() {
	if k.brokerSortCol < 0 {
		k.brokerSortCol = 0
	}
	k.brokerSortDesc = !k.brokerSortDesc
	k.applyBrokerSortAndRefresh()
}

func (k *KafuiContentProvider) applyBrokerSortAndRefresh() {
	k.applyBrokerSort()
	k.pagination.Page = 0
	k.pagination.SetTotalItems(len(k.allItems))
	k.updateTableForCurrentPageAndReset()
}

// applyBrokerSort sorts allItems by the active broker sort column/direction.
// Absent (nil) skews always sort last, regardless of direction.
func (k *KafuiContentProvider) applyBrokerSort() {
	if k.brokerSortCol < 0 || k.brokerSortCol >= len(brokerSortColumns) || !k.isBrokerResource() {
		return
	}
	sortBrokerItems(k.allItems, brokerSortColumns[k.brokerSortCol], k.brokerSortDesc)
}

// exportBrokersCSV writes the current broker list (with stats) to a timestamped
// CSV file in the working directory and reports the path via a notification.
func (k *KafuiContentProvider) exportBrokersCSV() tea.Cmd {
	stats := map[int32]api.BrokerStats{}
	brokers := make([]api.BrokerInfo, 0, len(k.allItems))
	for _, item := range k.allItems {
		if bri, ok := brokerItemFrom(item); ok {
			brokers = append(brokers, bri.info)
			if bri.hasStats {
				stats[bri.info.ID] = bri.stats
			}
		}
	}
	if len(brokers) == 0 {
		return nil
	}
	filename := fmt.Sprintf("brokers-%s.csv", time.Now().Format("20060102-150405"))
	f, err := os.Create(filename)
	if err != nil {
		return core.NotifyError("CSV export failed", err)
	}
	defer f.Close()
	if err := shared.WriteBrokerCSV(f, brokers, stats); err != nil {
		return core.NotifyError("CSV export failed", err)
	}
	abs, _ := filepath.Abs(filename)
	return core.NewNotification(core.StatusInfo, "Brokers exported", abs)
}

func (k *KafuiContentProvider) handleResourceList(msg CurrentResourceListMsg) {
	shared.Log.Info("handleResourceList called", "type", msg.ResourceType, "items", len(msg.Items))
	// Discard stale responses from a previous resource type (e.g. a topic
	// fetch that was in-flight when the user switched to schemas).
	if k.currentResource != nil && msg.ResourceType != k.currentResource.GetType() {
		shared.Log.Info("handleResourceList: discarding stale response", "got", msg.ResourceType, "want", k.currentResource.GetType())
		return
	}
	k.loading = false

	// Snapshot the enriched topic state we already know so it survives the reload.
	// Without this, every 5-second timer refresh would replace all *TopicResourceItem
	// objects with fresh stubs, resetting message counts, OSR/size, and selection.
	prevTopics := make(map[string]*TopicResourceItem)
	if msg.ResourceType == TopicResourceType {
		for _, item := range k.allItems {
			if rli, ok := item.(shared.ResourceListItem); ok {
				if tri, ok := rli.ResourceItem.(*TopicResourceItem); ok {
					prevTopics[tri.id] = tri
				}
			}
		}
	}

	// Sort items naturally by name
	sortedItems := make([]interface{}, len(msg.Items))
	copy(sortedItems, msg.Items)
	sort.Slice(sortedItems, func(i, j int) bool {
		nameI := k.getItemName(sortedItems[i])
		nameJ := k.getItemName(sortedItems[j])
		return strings.ToLower(nameI) < strings.ToLower(nameJ)
	})

	k.allItems = sortedItems

	// Restore previously-known enrichment/selection onto the fresh items so the
	// "…" placeholder and selection markers don't flash back on every refresh.
	if len(prevTopics) > 0 {
		for _, item := range k.allItems {
			if rli, ok := item.(shared.ResourceListItem); ok {
				if tri, ok := rli.ResourceItem.(*TopicResourceItem); ok {
					if prev, found := prevTopics[tri.id]; found {
						if prev.messageCount >= 0 {
							tri.messageCount = prev.messageCount
						}
						tri.outOfSync = prev.outOfSync
						tri.size = prev.size
						tri.detailsExtLoaded = prev.detailsExtLoaded
						tri.selected = prev.selected
					}
				}
			}
		}
	}

	k.allRows = convertItemsToRows(sortedItems, "", 0)

	if k.pendingReset {
		// Resource was switched — jump to page 0, row 0.
		k.pendingReset = false
		k.pagination.Page = 0
	}
	if k.isFiltered {
		k.reapplyFilter()
	} else {
		k.pagination.SetTotalItems(len(k.allItems))
		k.updateTableForCurrentPage()
	}
}

func (k *KafuiContentProvider) handleTopicList(msg TopicListMsg) {
	k.loading = false

	// Convert TopicItems to interface slice and sort
	interfaceItems := make([]interface{}, 0, len(msg))
	for _, item := range msg {
		interfaceItems = append(interfaceItems, item)
	}

	// Sort items naturally by name
	sort.Slice(interfaceItems, func(i, j int) bool {
		nameI := k.getItemName(interfaceItems[i])
		nameJ := k.getItemName(interfaceItems[j])
		return strings.ToLower(nameI) < strings.ToLower(nameJ)
	})

	k.allItems = interfaceItems
	k.allRows = convertItemsToRows(interfaceItems, "", 0)

	if k.pendingReset {
		// Resource was switched — jump to page 0, row 0.
		k.pendingReset = false
		k.pagination.Page = 0
	}
	if k.isFiltered {
		k.reapplyFilter()
	} else {
		k.pagination.SetTotalItems(len(k.allItems))
		k.updateTableForCurrentPage()
	}
}

func (k *KafuiContentProvider) handleResourceSelection() tea.Cmd {
	selectedItem := k.GetSelectedResourceItem()
	if selectedItem == nil {
		return nil
	}

	resourceType := k.currentResource.GetType()
	resourceID := k.getItemID(selectedItem)

	// Context items are handled locally: switch context and reload topics.
	if resourceType == ContextResourceType {
		return func() tea.Msg {
			return SelectContextMsg{ContextName: resourceID}
		}
	}

	// A Connect-cluster row drills into the connectors view (KC-11). The active
	// cluster name is stashed as a search filter so the aggregated listing opens
	// pre-filtered to that Connect cluster.
	if resourceType == ConnectClusterResourceType {
		cluster := resourceID
		return func() tea.Msg {
			return connectClusterSelectedMsg{cluster: cluster}
		}
	}

	// All other resource types navigate to the detail page.
	return func() tea.Msg {
		return NavigateToResourceDetailMsg{
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Item:         selectedItem,
		}
	}
}

func (k *KafuiContentProvider) GetSelectedResourceItem() interface{} {
	localIndex := k.resourcesTable.GetHighlightedRowIndex()
	globalIndex := k.pagination.GlobalIndex(localIndex)

	// Use filtered items if we're currently in a filtered state
	if k.isFiltered && len(k.filteredItems) > 0 {
		if globalIndex < 0 || globalIndex >= len(k.filteredItems) {
			return nil
		}
		return k.filteredItems[globalIndex]
	}

	// Otherwise use all items
	if globalIndex < 0 || globalIndex >= len(k.allItems) {
		return nil
	}
	return k.allItems[globalIndex]
}

func (k *KafuiContentProvider) itemMatchesQuery(item interface{}, query string) bool {
	queryLower := strings.ToLower(query)

	// The ACL search bar is scoped to the principal (AQ-14): match against the
	// principal only, not the composite GetID.
	if k.isACLResource() {
		if ari, _, ok := aclItemFrom(item); ok {
			return strings.Contains(strings.ToLower(ari.principal), queryLower)
		}
	}

	// Connectors support substring matching across name/status/type/plugin plus
	// search-syntax prefixes (status:FAILED, type:sink) — KC-12.
	if k.isConnectorResource() {
		if ci, _, ok := connectorItemFrom(item); ok {
			return connectorMatchesQuery(ci.conn, query)
		}
	}

	switch i := item.(type) {
	case shared.ResourceListItem:
		return strings.Contains(strings.ToLower(i.ResourceItem.GetID()), queryLower)
	case TopicItem:
		return strings.Contains(strings.ToLower(i.name), queryLower)
	default:
		return false
	}
}

func (k *KafuiContentProvider) createHighlightedItem(item interface{}, query string) interface{} {
	switch i := item.(type) {
	case shared.ResourceListItem:
		return shared.HighlightedResourceListItem{
			ResourceItem: i.ResourceItem,
			SearchQuery:  query,
		}
	case TopicItem:
		return shared.HighlightedTopicItem{
			Name:        i.name,
			Topic:       i.topic,
			SearchQuery: query,
		}
	default:
		return item
	}
}

func (k *KafuiContentProvider) parseResourceType(name string) ResourceType {
	switch strings.ToLower(name) {
	case "topics", "topic":
		return TopicResourceType
	case "consumer-groups", "consumer-group", "groups", "group":
		return ConsumerGroupResourceType
	case "schemas", "schema":
		return SchemaResourceType
	case "contexts", "context":
		return ContextResourceType
	case "acls", "acl":
		return ACLResourceType
	case "brokers", "broker":
		return BrokerResourceType
	case "quotas", "quota":
		return QuotaResourceType
	case "connectors", "connector":
		return ConnectorResourceType
	case "connect", "connect-clusters", "connect-cluster", "connects":
		return ConnectClusterResourceType
	default:
		return -1
	}
}

func (k *KafuiContentProvider) getItemID(item interface{}) string {
	switch i := item.(type) {
	case shared.ResourceListItem:
		return i.ResourceItem.GetID()
	case shared.HighlightedResourceListItem:
		return i.ResourceItem.GetID()
	case TopicItem:
		return i.name
	case shared.HighlightedTopicItem:
		return i.Name
	default:
		return "unknown"
	}
}

func (k *KafuiContentProvider) getItemName(item interface{}) string {
	switch i := item.(type) {
	case shared.ResourceListItem:
		return i.ResourceItem.GetID()
	case shared.HighlightedResourceListItem:
		return i.ResourceItem.GetID()
	case TopicItem:
		return i.name
	case shared.HighlightedTopicItem:
		return i.Name
	default:
		return "unknown"
	}
}

// resourceChoice is one selectable entry in the resource picker.
type resourceChoice struct {
	name string       // canonical display/command name (e.g. "consumer-groups")
	rt   ResourceType // target resource type
}

// pickerCapabilityAllows reports whether a resource type is available on the
// active cluster. Core resources always show; optional integrations are gated
// on capabilities (mirrors sidebar_sections.enabled).
func (k *KafuiContentProvider) pickerCapabilityAllows(rt ResourceType) bool {
	if k.common == nil {
		return true
	}
	switch rt {
	case SchemaResourceType:
		return k.common.HasCapability(api.CapSchemaRegistry)
	case ACLResourceType:
		return k.common.HasCapability(api.CapACLView)
	case ConnectClusterResourceType, ConnectorResourceType:
		return k.common.HasCapability(api.CapKafkaConnect)
	default:
		return true
	}
}

// availableResourceChoices returns the capability-filtered pickable resources.
func (k *KafuiContentProvider) availableResourceChoices() []resourceChoice {
	all := []resourceChoice{
		{"topics", TopicResourceType},
		{"consumer-groups", ConsumerGroupResourceType},
		{"contexts", ContextResourceType},
		{"brokers", BrokerResourceType},
		{"quotas", QuotaResourceType},
		{"schemas", SchemaResourceType},
		{"acls", ACLResourceType},
		{"connect-clusters", ConnectClusterResourceType},
		{"connectors", ConnectorResourceType},
	}
	out := make([]resourceChoice, 0, len(all))
	for _, c := range all {
		if k.pickerCapabilityAllows(c.rt) {
			out = append(out, c)
		}
	}
	return out
}

// matchedResourceChoices returns the picker suggestions matching the query
// (case-insensitive substring), or all choices when the query is empty.
func (k *KafuiContentProvider) matchedResourceChoices(query string) []resourceChoice {
	q := strings.ToLower(strings.TrimSpace(query))
	choices := k.availableResourceChoices()
	if q == "" {
		return choices
	}
	out := make([]resourceChoice, 0, len(choices))
	for _, c := range choices {
		if strings.Contains(c.name, q) {
			out = append(out, c)
		}
	}
	return out
}

// bestResourceMatch returns the canonical name of the first suggestion for the
// query (used for tab-completion), or "" when there is no match.
func (k *KafuiContentProvider) bestResourceMatch(query string) string {
	m := k.matchedResourceChoices(query)
	if len(m) == 0 {
		return ""
	}
	return m[0].name
}

// commitResourcePicker resolves the input (typed name or the first suggestion),
// switches to that capability-allowed resource, and closes the picker.
func (k *KafuiContentProvider) commitResourcePicker(input string) tea.Cmd {
	rt := k.parseResourceType(strings.TrimSpace(input))
	if rt == -1 {
		// Fall back to the first suggestion for the partial input.
		if best := k.bestResourceMatch(input); best != "" {
			rt = k.parseResourceType(best)
		}
	}
	k.resourcePickerMode = false
	k.resourcePickerInput = ""
	if rt == -1 || !k.pickerCapabilityAllows(rt) {
		return nil
	}
	k.switchResource(SwitchResourceMsg(rt))
	return tea.Batch(k.loadCurrentResource(), k.breadcrumbCmd(), k.countSpinner.Tick)
}

// renderResourcePicker renders the picker input line and the capability-filtered
// suggestion list.
func (k *KafuiContentProvider) renderResourcePicker(width int) string {
	promptStyle := k.styles.SearchStyle.Prompt
	var b strings.Builder
	b.WriteString(promptStyle.Render(": ") + k.resourcePickerInput + promptStyle.Render("█"))
	b.WriteString("\n\n")
	current := ResourceType(-1)
	if k.currentResource != nil {
		current = k.currentResource.GetType()
	}
	for _, c := range k.matchedResourceChoices(k.resourcePickerInput) {
		if c.rt == current {
			b.WriteString(k.styles.SearchStyle.Prompt.Render("› "+c.name) + "\n")
		} else {
			b.WriteString(k.styles.Muted.Render("  "+c.name) + "\n")
		}
	}
	b.WriteString("\n" + k.styles.SearchStyle.Help.Render("Enter switch • Tab complete • Esc cancel"))
	return b.String()
}

// handleSelectContext switches the active Kafka context and reloads the topic list.
func (k *KafuiContentProvider) handleSelectContext(msg SelectContextMsg) tea.Cmd {
	// Attempt to switch context; ignore errors so the UI can still reload.
	_ = k.dataSource.SetContext(msg.ContextName)

	// Always return to the Topics view after a context switch.
	k.switchResource(SwitchResourceMsg(TopicResourceType))
	return tea.Batch(k.loadCurrentResource(), k.breadcrumbCmd())
}

// Navigation message for resource selection
type NavigateToResourceDetailMsg struct {
	ResourceType ResourceType
	ResourceID   string
	Item         interface{}
}

// Message to start resource switching mode
type StartResourceSwitchingMsg struct{}

// KafuiHeaderDataProvider provides header data for Kafui
// Implements providers.HeaderDataProvider interface
type KafuiHeaderDataProvider struct {
	dataSource api.KafkaDataSource
	common     *core.Common
	lastUpdate time.Time
}

func NewKafuiHeaderDataProvider(dataSource api.KafkaDataSource) *KafuiHeaderDataProvider {
	return &KafuiHeaderDataProvider{
		dataSource: dataSource,
		lastUpdate: time.Now(),
	}
}

func (k *KafuiHeaderDataProvider) GetBrandName() string {
	return "Kafui™"
}

func (k *KafuiHeaderDataProvider) GetAppName() string {
	return "Kafka TUI"
}

func (k *KafuiHeaderDataProvider) GetStatusData() map[string]interface{} {
	context := k.dataSource.GetContext()
	data := map[string]interface{}{
		"time":    k.lastUpdate.Format("15:04:05"),
		"status":  "connected",
		"context": context,
		"cluster": "kafka-cluster",
	}
	// Identity, active profile and read-only badge (AA-11). Identity + profile
	// are only shown when authorization is enabled: with authz off (the default
	// single-user mode) the identity is just the OS username, which carries no
	// authorization meaning and is PII we shouldn't surface in the header.
	if k.common != nil {
		if k.common.AuthzEnabled() {
			if k.common.Identity != "" {
				data["identity"] = k.common.Identity
			}
			data["profile"] = k.common.ActiveProfileName()
		}
		data["readonly"] = k.common.IsReadOnly()
	}
	return data
}

func (k *KafuiHeaderDataProvider) HandleHeaderUpdate(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case TimerTickMsg:
		k.lastUpdate = time.Time(msg)
		return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
			return TimerTickMsg(t)
		})
	}
	return nil
}

func (k *KafuiHeaderDataProvider) InitHeader() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return TimerTickMsg(t)
	})
}

// NewKafuiHeaderDataProviderWithCommon creates a header provider using Common context
func NewKafuiHeaderDataProviderWithCommon(common *core.Common) *KafuiHeaderDataProvider {
	p := NewKafuiHeaderDataProvider(common.DataSource)
	p.common = common
	return p
}
