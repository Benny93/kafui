package mainpage

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
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
//
// Numeric/short columns stay fixed; the wide text columns are flex so that
// WithTargetWidth grows them to fill the content pane on a wide terminal and
// shrinks them (truncating cells) instead of overflowing on a narrow one.
//
// Fixed widths are sized to their header plus the widest realistic value and
// nothing more: bubble-table never shrinks a fixed column, so every spare
// character they hold is a character the flex columns cannot use — and on an
// 80-column terminal it is the difference between fitting and wrapping.
//
// Flex factors are kept small for the same reason: bubble-table hands the
// integer remainder of the flex split out one column at a time, so a factor
// sum far above the number of flex columns strands unused width.
func createResourceTableColumns(resourceType ResourceType) []table.Column {
	right := lipgloss.NewStyle().AlignHorizontal(lipgloss.Right)
	switch resourceType {
	case TopicResourceType:
		return []table.Column{
			table.NewFlexColumn(colTopicName, "Name", 1),
			table.NewColumn(colTopicPartitions, "Partitions", 11).WithStyle(right),
			table.NewColumn(colTopicReplication, "Replication", 12).WithStyle(right),
			table.NewColumn(colTopicMessages, "Messages", 10).WithStyle(right),
			table.NewColumn(colTopicOSR, "OSR", 5).WithStyle(right),
			table.NewColumn(colTopicSize, "Size", 10).WithStyle(right),
		}
	case ContextResourceType:
		return []table.Column{
			table.NewFlexColumn(colName, "Name", 1),
			table.NewFlexColumn(colPartitions, "Brokers", 1),
			table.NewColumn(colReplication, "Status", 8).WithStyle(right),
		}
	case SchemaResourceType:
		return []table.Column{
			table.NewFlexColumn(colName, "Subject", 1),
			table.NewColumn(colPartitions, "Version", 9).WithStyle(right),
			table.NewColumn(colReplication, "ID", 5).WithStyle(right),
			table.NewColumn(colMessages, "Type", 10),
			table.NewColumn(colSchemaCompat, "Compatibility", 14),
		}
	case ConsumerGroupResourceType:
		return []table.Column{
			table.NewFlexColumn(colGroupName, "Name", 1),
			table.NewColumn(colGroupState, "State", 12),
			table.NewColumn(colGroupMembers, "Members", 8).WithStyle(right),
			table.NewColumn(colGroupTopics, "Topics", 7).WithStyle(right),
			table.NewColumn(colGroupLag, "Lag", 10).WithStyle(right),
			table.NewColumn(colGroupCoord, "Coordinator", 12).WithStyle(right),
		}
	case ACLResourceType:
		return []table.Column{
			table.NewFlexColumn(colACLPrincipal, "Principal", 2),
			table.NewFlexColumn(colACLResource, "Resource", 1),
			table.NewColumn(colACLPattern, "Pattern", 11),
			table.NewColumn(colACLHost, "Host", 6),
			table.NewColumn(colACLOperation, "Operation", 10),
			table.NewColumn(colACLPermission, "Permission", 10),
		}
	case QuotaResourceType:
		return []table.Column{
			table.NewFlexColumn(colQuotaUser, "User", 1),
			table.NewFlexColumn(colQuotaClient, "Client ID", 1),
			table.NewFlexColumn(colQuotaIP, "IP", 1),
			table.NewFlexColumn(colQuotaValues, "Quotas", 2),
		}
	case BrokerResourceType:
		return []table.Column{
			table.NewColumn(colBrokerID, "ID", 4).WithStyle(right),
			table.NewFlexColumn(colBrokerHost, "Host", 1),
			table.NewColumn(colBrokerPort, "Port", 6).WithStyle(right),
			table.NewFlexColumn(colBrokerDisk, "Disk Usage", 1),
			table.NewColumn(colBrokerISR, "ISR", 5).WithStyle(right),
			table.NewColumn(colBrokerSkew, "Skew", 6).WithStyle(right),
		}
	case ConnectorResourceType:
		return []table.Column{
			table.NewFlexColumn(colConnName, "Name", 2),
			table.NewFlexColumn(colConnCluster, "Connect", 1),
			table.NewColumn(colConnType, "Type", 8),
			table.NewFlexColumn(colConnPlugin, "Plugin", 2),
			table.NewFlexColumn(colConnTopics, "Topics", 1),
			table.NewColumn(colConnState, "Status", 8),
			table.NewFlexColumn(colConnGroup, "Consumer group", 1),
			table.NewColumn(colConnTasks, "Tasks", 6).WithStyle(right),
		}
	case ConnectClusterResourceType:
		return []table.Column{
			table.NewFlexColumn(colCCName, "Name", 2),
			table.NewFlexColumn(colCCVersion, "Version", 1),
			table.NewColumn(colCCConnectors, "Connectors", 11).WithStyle(right),
			table.NewColumn(colCCTasks, "Running Tasks", 14).WithStyle(right),
		}
	default:
		return []table.Column{
			table.NewFlexColumn(colName, "Name", 1),
			table.NewColumn(colPartitions, "Partitions", 11).WithStyle(right),
			table.NewColumn(colReplication, "Replication", 12).WithStyle(right),
			table.NewColumn(colMessages, "Messages", 10).WithStyle(right),
		}
	}
}

// firstColumnWidth returns the rendered width of the leading (Name) column once
// cols are laid out at totalWidth, mirroring bubble-table's flex arithmetic.
// Returns 0 when the leading column is fixed — nothing to compute.
func firstColumnWidth(cols []table.Column, totalWidth int) int {
	if len(cols) == 0 || !cols[0].IsFlex() {
		return 0
	}
	avail, factors := totalWidth-len(cols)-1, 0
	for _, c := range cols {
		if c.IsFlex() {
			factors += c.FlexFactor()
		} else {
			avail -= c.Width()
		}
	}
	if factors == 0 || avail <= 0 {
		return 0
	}
	return avail * cols[0].FlexFactor() / factors
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
