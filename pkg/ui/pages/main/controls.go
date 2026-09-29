package mainpage

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"

	"github.com/Benny93/kafui/pkg/authz"
	"github.com/Benny93/kafui/pkg/ui/components/menu"
	"github.com/Benny93/kafui/pkg/ui/keys"
)

// This file is the main screen's half of the controls contract: the registry
// actions dispatch to whichever resource is showing, and everything that used
// to sit on an undiscoverable single letter is now an entry in the actions
// menu, where it carries a label, its key, and — when the user may not run it —
// the reason why.

// KeyScope reports the scope the shell resolves this page's keys against.
func (k *KafuiContentProvider) KeyScope() keys.Scope { return keys.ScopeList }

// Unwind consumes one level of esc for this page before the shell navigates to
// the parent screen: an open filter first, then a live multi-selection.
func (k *KafuiContentProvider) Unwind() (tea.Cmd, bool) {
	if k.searchMode {
		k.searchMode = false
		k.clearSearch()
		return k.reloadPageDetails(), true
	}
	if k.isFiltered || k.currentFilter != "" {
		k.clearSearch()
		return k.reloadPageDetails(), true
	}
	if len(k.selected) > 0 {
		k.clearTopicSelection()
		return nil, true
	}
	return nil, false
}

// ---- registry actions dispatched to the showing resource ----

func (k *KafuiContentProvider) cycleSortColumn() {
	switch {
	case k.isBrokerResource():
		k.cycleBrokerSortColumn()
	case k.isGroupResource():
		k.cycleGroupSortColumn()
	case k.isTopicResource():
		k.cycleTopicSortColumn()
	}
}

func (k *KafuiContentProvider) toggleSortDir() {
	switch {
	case k.isBrokerResource():
		k.toggleBrokerSortDir()
	case k.isGroupResource():
		k.toggleGroupSortDir()
	case k.isTopicResource():
		k.toggleTopicSortDir()
	}
}

func (k *KafuiContentProvider) exportCurrentResourceCSV() tea.Cmd {
	switch {
	case k.isBrokerResource():
		return k.exportBrokersCSV()
	case k.isGroupResource():
		return k.exportGroupsCSV()
	case k.isTopicResource():
		return k.exportTopicsCSV()
	case k.isACLResource():
		return k.exportACLsCSV()
	case k.isConnectorResource():
		return k.exportConnectorsCSV()
	case k.isConnectClusterResource():
		return k.exportConnectClustersCSV()
	}
	return nil
}

func (k *KafuiContentProvider) createForCurrentResource() tea.Cmd {
	switch {
	case k.isTopicResource():
		return k.openCreateTopicForm()
	case k.isACLResource():
		return k.openCreateACLForm()
	case k.isQuotaResource():
		return k.openQuotaForm(false)
	case k.isConnectorResource():
		return k.openCreateConnectorForm()
	}
	return nil
}

func (k *KafuiContentProvider) deleteForCurrentResource() tea.Cmd {
	switch {
	case k.isTopicResource():
		return k.deleteSelectedTopics()
	case k.isGroupResource():
		return k.deleteSelectedGroup()
	case k.isACLResource():
		return k.deleteSelectedACL()
	case k.isQuotaResource():
		return k.deleteSelectedQuota()
	}
	return nil
}

// ---- discovery surfaces ----

// PaletteResources is the resource list the shell offers in the command
// palette. It lives here because this screen owns the resource vocabulary, but
// the shell adds the entries from every screen — switching resource must work
// from the metrics or ksqlDB page too, not only from the list you are already
// on.
func PaletteResources() []ResourceType {
	return []ResourceType{
		TopicResourceType, ConsumerGroupResourceType, SchemaResourceType,
		ContextResourceType, ACLResourceType, BrokerResourceType,
		QuotaResourceType, ConnectClusterResourceType, ConnectorResourceType,
	}
}

// ContextActions lists everything the user can do to the current selection.
// The registry supplies each entry's key, so the menu teaches the shortcut
// rather than hiding it.
func (k *KafuiContentProvider) ContextActions() []menu.Entry {
	keyOf := func(a keys.Action) string { return keys.Default.KeyFor(a) }

	// gated builds an entry that stays visible but disabled when the active
	// permission profile or read-only mode forbids it, as the spec requires.
	gated := func(e menu.Entry, action authz.Action, rt authz.ResourceType, name string) menu.Entry {
		if k.common != nil && !k.common.Can(action, rt, name) {
			e.Disabled = true
			e.Reason = "not permitted by the active profile or the cluster is read-only"
		}
		return e
	}

	sel := k.selectedResourceName()
	var out []menu.Entry

	switch {
	case k.isTopicResource():
		out = append(out,
			menu.Entry{Label: "Open topic", Key: keyOf(keys.ActionActivate), Run: func() tea.Cmd { return k.handleResourceSelection() }},
			gated(menu.Entry{Label: "New topic…", Key: keyOf(keys.ActionNew), Run: func() tea.Cmd { return k.openCreateTopicForm() }},
				authz.ActionCreate, authz.ResourceTopic, ""),
			gated(menu.Entry{Label: "Clone topic…", Run: func() tea.Cmd { return k.openCloneTopicForm() }},
				authz.ActionCreate, authz.ResourceTopic, ""),
			gated(menu.Entry{Label: "Delete topic", Key: keyOf(keys.ActionDelete), Destructive: true,
				Run: func() tea.Cmd { return k.deleteSelectedTopics() }}, authz.ActionDelete, authz.ResourceTopic, sel),
			gated(menu.Entry{Label: "Recreate topic", Destructive: true, Run: func() tea.Cmd { return k.recreateSelectedTopic() }},
				authz.ActionDelete, authz.ResourceTopic, sel),
			gated(menu.Entry{Label: "Purge messages", Destructive: true, Run: func() tea.Cmd { return k.purgeSelectedTopics() }},
				authz.ActionEdit, authz.ResourceTopic, sel),
			menu.Entry{Label: "Show internal topics", Key: keyOf(keys.ActionToggleInternal), Run: func() tea.Cmd { return k.toggleHideInternal() }},
		)

	case k.isGroupResource():
		out = append(out,
			menu.Entry{Label: "Open consumer group", Key: keyOf(keys.ActionActivate), Run: func() tea.Cmd { return k.handleResourceSelection() }},
			gated(menu.Entry{Label: "Delete group", Key: keyOf(keys.ActionDelete), Destructive: true,
				Run: func() tea.Cmd { return k.deleteSelectedGroup() }}, authz.ActionDelete, authz.ResourceConsumerGroup, sel),
			menu.Entry{Label: "Filter by state", Run: func() tea.Cmd { return k.cycleGroupStateFilterCmd() }},
		)

	case k.isACLResource():
		out = append(out,
			gated(menu.Entry{Label: "New ACL binding…", Key: keyOf(keys.ActionNew), Run: func() tea.Cmd { return k.openCreateACLForm() }},
				authz.ActionCreate, authz.ResourceACL, ""),
			gated(menu.Entry{Label: "Delete binding", Key: keyOf(keys.ActionDelete), Destructive: true,
				Run: func() tea.Cmd { return k.deleteSelectedACL() }}, authz.ActionDelete, authz.ResourceACL, sel),
			gated(menu.Entry{Label: "Sync bindings from CSV…", Run: func() tea.Cmd { return k.openACLSyncForm() }},
				authz.ActionCreate, authz.ResourceACL, ""),
			menu.Entry{Label: "Filter by resource type", Run: func() tea.Cmd { return k.cycleACLResourceTypeFilter() }},
			menu.Entry{Label: "Filter by pattern type", Run: func() tea.Cmd { return k.cycleACLPatternFilter() }},
		)

	case k.isQuotaResource():
		out = append(out,
			gated(menu.Entry{Label: "New quota…", Key: keyOf(keys.ActionNew), Run: func() tea.Cmd { return k.openQuotaForm(false) }},
				authz.ActionCreate, authz.ResourceClientQuota, ""),
			gated(menu.Entry{Label: "Edit quota…", Key: keyOf(keys.ActionEdit), Run: func() tea.Cmd { return k.openQuotaForm(true) }},
				authz.ActionEdit, authz.ResourceClientQuota, sel),
			gated(menu.Entry{Label: "Delete quota", Key: keyOf(keys.ActionDelete), Destructive: true,
				Run: func() tea.Cmd { return k.deleteSelectedQuota() }}, authz.ActionDelete, authz.ResourceClientQuota, sel),
		)

	case k.isConnectorResource():
		out = append(out,
			menu.Entry{Label: "Open connector", Key: keyOf(keys.ActionActivate), Run: func() tea.Cmd { return k.handleResourceSelection() }},
			gated(menu.Entry{Label: "New connector…", Key: keyOf(keys.ActionNew), Run: func() tea.Cmd { return k.openCreateConnectorForm() }},
				authz.ActionCreate, authz.ResourceConnector, ""),
		)

	default:
		out = append(out,
			menu.Entry{Label: "Open", Key: keyOf(keys.ActionActivate), Run: func() tea.Cmd { return k.handleResourceSelection() }},
		)
	}

	// Actions every list offers.
	out = append(out,
		menu.Entry{Label: "Copy row", Key: keyOf(keys.ActionCopy), Run: func() tea.Cmd { return k.handleCopyRow() }},
		menu.Entry{Label: "Export list as CSV", Key: keyOf(keys.ActionExport), Run: func() tea.Cmd { return k.exportCurrentResourceCSV() }},
		menu.Entry{Label: "Sort by next column", Key: keyOf(keys.ActionSort)},
		menu.Entry{Label: "Refresh", Key: keyOf(keys.ActionRefresh), Run: func() tea.Cmd { return k.refreshCurrentResource() }},
	)
	return out
}

// selectedResourceName is the authz subject for the highlighted row, or "" when
// nothing is selected.
func (k *KafuiContentProvider) selectedResourceName() string {
	if item := k.GetSelectedResourceItem(); item != nil {
		if named, ok := item.(interface{ GetName() string }); ok {
			return named.GetName()
		}
	}
	return ""
}

// cycleGroupStateFilterCmd adapts the void-returning filter cycler to a Cmd so
// it can be a menu entry.
func (k *KafuiContentProvider) cycleGroupStateFilterCmd() tea.Cmd {
	k.cycleGroupStateFilter()
	return k.reloadPageDetails()
}

// columnAtX maps a pointer X offset inside the table to a column index, using
// the same widths bubble-table laid the header out with. It is what makes
// "click a header to sort by that column" possible rather than only cycling.
func columnAtX(cols []table.Column, tableWidth, relX int) (int, bool) {
	// Mirror bubble-table's flex arithmetic so the boundaries match what was
	// drawn: fixed columns keep their width, flex columns share what is left.
	avail, factors := tableWidth-len(cols)-1, 0
	for _, c := range cols {
		if c.IsFlex() {
			factors += c.FlexFactor()
		} else {
			avail -= c.Width()
		}
	}

	x := 1 // the table's left border
	for i, c := range cols {
		w := c.Width()
		if c.IsFlex() && factors > 0 && avail > 0 {
			w = avail * c.FlexFactor() / factors
		}
		if relX >= x && relX < x+w {
			return i, true
		}
		x += w + 1 // the separator between cells
	}
	return 0, false
}

// setSortColumn sorts the showing resource by a specific column, reversing the
// direction when it is already the sort column — the behaviour the controls
// spec asks of a header click.
func (k *KafuiContentProvider) setSortColumn(i int) {
	switch {
	case k.isTopicResource():
		if k.topicSortCol == i {
			k.toggleTopicSortDir()
			return
		}
		k.topicSortCol = i
		k.topicSortDesc = false
		k.applyTopicSortAndRefresh()
	case k.isGroupResource():
		if k.groupSortCol == i {
			k.toggleGroupSortDir()
			return
		}
		k.groupSortCol = i
		k.groupSortDesc = false
		k.applyGroupSortAndRefresh()
	case k.isBrokerResource():
		if k.brokerSortCol == i {
			k.toggleBrokerSortDir()
			return
		}
		k.brokerSortCol = i
		k.brokerSortDesc = false
		k.applyBrokerSortAndRefresh()
	}
}
