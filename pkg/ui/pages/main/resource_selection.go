package mainpage

import (
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

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

// handleSelectContext switches the active Kafka context and reloads the topic list.
func (k *KafuiContentProvider) handleSelectContext(msg SelectContextMsg) tea.Cmd {
	// On failure stay where we are: reloading would show the old cluster's
	// topics while the user believes the switch happened.
	if err := k.dataSource.SetContext(msg.ContextName); err != nil {
		return core.NotifyError("Switch context failed", err)
	}

	k.resetCounts(msg.ContextName)

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
