package mainpage

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

// internalTopicPrefix is the default prefix that classifies a topic as internal.
// ponytail: a config-driven prefix (core.Common.Config) is deferred; Kafka's own
// internal topics all use "__", which covers the classification requirement.
const internalTopicPrefix = "__"

// isInternalTopicName reports whether a topic name is classified as internal.
func isInternalTopicName(name string) bool {
	return strings.HasPrefix(name, internalTopicPrefix)
}

// isTopicResource reports whether the topics resource is currently active.
func (k *KafuiContentProvider) isTopicResource() bool {
	return k.currentResource != nil && k.currentResource.GetType() == TopicResourceType
}

// topicItemFrom unwraps a *TopicResourceItem from the list-item wrappers,
// returning the item plus any active search query (for highlighting).
func topicItemFrom(item interface{}) (*TopicResourceItem, string, bool) {
	switch v := item.(type) {
	case shared.ResourceListItem:
		tri, ok := v.ResourceItem.(*TopicResourceItem)
		return tri, "", ok
	case shared.HighlightedResourceListItem:
		tri, ok := v.ResourceItem.(*TopicResourceItem)
		return tri, v.SearchQuery, ok
	case *TopicResourceItem:
		return v, "", true
	default:
		return nil, "", false
	}
}

// --- row rendering (TP-14) ---

// topicMessagesCell renders the message-count cell: the count when known, the
// loading placeholder while pending, and "N/A" once the extended fetch has
// completed without yielding a value.
func topicMessagesCell(t *TopicResourceItem, placeholder string) string {
	if t.messageCount >= 0 {
		return strconv.FormatInt(t.messageCount, 10)
	}
	if t.detailsExtLoaded {
		return "N/A"
	}
	return placeholder
}

// topicSizeCell renders the on-disk size cell via FormatBytes2dp, with the same
// loading/"N/A" discipline as topicMessagesCell.
func topicSizeCell(t *TopicResourceItem, placeholder string) string {
	if t.size >= 0 {
		return shared.FormatBytes2dp(t.size)
	}
	if t.detailsExtLoaded {
		return "N/A"
	}
	return placeholder
}

// topicOSRCell renders the out-of-sync (under-replicated partition) cell, styled
// in the alert colour when > 0.
func topicOSRCell(t *TopicResourceItem, placeholder string) string {
	if t.outOfSync >= 0 {
		s := strconv.Itoa(t.outOfSync)
		if t.outOfSync > 0 {
			return lipgloss.NewStyle().Foreground(stylesPkg.Error).Render(s)
		}
		return s
	}
	if t.detailsExtLoaded {
		return "N/A"
	}
	return placeholder
}

// topicRowData builds a bubble-table row for a topic. The Name cell carries the
// multi-select marker and the internal-topic label/styling.
func topicRowData(t *TopicResourceItem, searchQuery, placeholder string, nameMaxWidth int) table.RowData {
	base := t.id
	if searchQuery != "" {
		base = shared.HighlightSearchMatches(base, searchQuery)
	} else if nameMaxWidth > 0 {
		base = truncateMiddle(base, nameMaxWidth)
	}
	nameCell := base
	if t.isInternal || isInternalTopicName(t.id) {
		nameCell = lipgloss.NewStyle().Foreground(stylesPkg.FgMuted).Render(base) +
			lipgloss.NewStyle().Foreground(stylesPkg.FgSubtle).Render(" (internal)")
	}
	if t.selected {
		nameCell = lipgloss.NewStyle().Foreground(stylesPkg.Accent).Render("● ") + nameCell
	}

	partitions := placeholder
	if t.partitions >= 0 {
		partitions = strconv.FormatInt(int64(t.partitions), 10)
	}
	replication := placeholder
	if t.replicationFactor >= 0 {
		replication = strconv.FormatInt(int64(t.replicationFactor), 10)
	}

	return table.RowData{
		colTopicName:        nameCell,
		colTopicPartitions:  partitions,
		colTopicReplication: replication,
		colTopicMessages:    topicMessagesCell(t, placeholder),
		colTopicOSR:         topicOSRCell(t, placeholder),
		colTopicSize:        topicSizeCell(t, placeholder),
	}
}

// --- extended lazy enrichment (TP-14) ---

// loadTopicDetailsExt enriches ONLY the visible page of topics with OSR and size
// via GetTopicDetails (per topic) + GetTopicSizes (batch). Off-screen topics are
// never fetched, preserving the visible-page-only discipline.
func (k *KafuiContentProvider) loadTopicDetailsExt() tea.Cmd {
	pageItems := k.pagination.GetCurrentPageItems(k.activeItems())
	names := make([]string, 0, len(pageItems))
	for _, item := range pageItems {
		if tri, _, ok := topicItemFrom(item); ok && !tri.detailsExtLoaded {
			names = append(names, tri.id)
		}
	}
	if len(names) == 0 {
		return nil
	}
	ds := k.dataSource
	return func() tea.Msg {
		sizes, _ := ds.GetTopicSizes(names)

		// GetTopicDetails opens its own broker connection per call, so on a
		// remote cluster (e.g. TLS mutual-auth over the internet) fetching a
		// full page sequentially can take tens of seconds — long enough that
		// the "…" placeholder reads as permanently hung (BUG-4). Fetching the
		// page concurrently bounds the wait to one call's latency instead of
		// page-size times that.
		var mu sync.Mutex
		var wg sync.WaitGroup
		out := make(map[string]topicExtInfo, len(names))
		for _, name := range names {
			info := topicExtInfo{outOfSync: -1, size: -1}
			if sz, ok := sizes[name]; ok {
				info.size = sz
			}
			mu.Lock()
			out[name] = info
			mu.Unlock()

			wg.Add(1)
			go func(name string, info topicExtInfo) {
				defer wg.Done()
				if d, err := ds.GetTopicDetails(name); err == nil {
					info.outOfSync = d.UnderReplicatedPartitions
					info.isInternal = d.IsInternal
				}
				mu.Lock()
				out[name] = info
				mu.Unlock()
			}(name, info)
		}
		wg.Wait()
		return TopicDetailsExtLoadedMsg(out)
	}
}

// applyTopicDetailsExt merges OSR/size back onto the topic items and rebuilds rows.
func (k *KafuiContentProvider) applyTopicDetailsExt(msg TopicDetailsExtLoadedMsg) {
	for _, item := range k.allItems {
		if tri, _, ok := topicItemFrom(item); ok {
			if info, found := msg[tri.id]; found {
				tri.outOfSync = info.outOfSync
				tri.size = info.size
				if info.isInternal {
					tri.isInternal = true
				}
				tri.detailsExtLoaded = true
			}
		}
	}
	k.allRows = convertItemsToRows(k.allItems, "", k.nameColumnWidth)
	if k.isFiltered {
		k.reapplyFilter()
	} else {
		k.updateTableForCurrentPage()
	}
}

// activeItems returns the filtered list when a filter is active, else all items.
func (k *KafuiContentProvider) activeItems() []interface{} {
	if k.isFiltered {
		return k.filteredItems
	}
	return k.allItems
}

// --- sorting (TP-15) ---

// topicSortColumns maps the sort cycle to a sort key.
var topicSortColumns = []string{"name", "partitions", "osr", "replication", "messages", "size"}

// sortTopicItems sorts topic items in place by the given key/direction.
func sortTopicItems(items []interface{}, col string, desc bool) {
	less := func(a, b *TopicResourceItem) bool {
		switch col {
		case "partitions":
			return a.partitions < b.partitions
		case "osr":
			return a.outOfSync < b.outOfSync
		case "replication":
			return a.replicationFactor < b.replicationFactor
		case "messages":
			return a.messageCount < b.messageCount
		case "size":
			return a.size < b.size
		default: // "name"
			return a.id < b.id
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		ai, _, aok := topicItemFrom(items[i])
		bj, _, bok := topicItemFrom(items[j])
		if !aok || !bok {
			return false
		}
		if desc {
			return less(bj, ai)
		}
		return less(ai, bj)
	})
}

func (k *KafuiContentProvider) cycleTopicSortColumn() {
	k.topicSortCol = (k.topicSortCol + 1) % len(topicSortColumns)
	k.applyTopicSortAndRefresh()
}

func (k *KafuiContentProvider) toggleTopicSortDir() {
	if k.topicSortCol < 0 {
		k.topicSortCol = 0
	}
	k.topicSortDesc = !k.topicSortDesc
	k.applyTopicSortAndRefresh()
}

func (k *KafuiContentProvider) applyTopicSortAndRefresh() {
	k.applyTopicSort()
	k.allRows = convertItemsToRows(k.allItems, "", 0)
	k.applyFilters(true)
}

// applyTopicSort sorts allItems by the active topic sort column/direction.
func (k *KafuiContentProvider) applyTopicSort() {
	if k.topicSortCol < 0 || k.topicSortCol >= len(topicSortColumns) || !k.isTopicResource() {
		return
	}
	sortTopicItems(k.allItems, topicSortColumns[k.topicSortCol], k.topicSortDesc)
}

// --- internal-topic visibility (TP-16) ---

// hasVisibilityFilter reports whether internal topics are currently hidden.
func (k *KafuiContentProvider) hasVisibilityFilter() bool {
	return k.isTopicResource() && k.hideInternal
}

// isInternalItem reports whether a list item is an internal topic.
func (k *KafuiContentProvider) isInternalItem(item interface{}) bool {
	tri, _, ok := topicItemFrom(item)
	return ok && (tri.isInternal || isInternalTopicName(tri.id))
}

// toggleHideInternal flips the internal-topic visibility, persists the preference,
// and reapplies the filter (resetting to the first page).
func (k *KafuiContentProvider) toggleHideInternal() tea.Cmd {
	k.hideInternal = !k.hideInternal
	_ = shared.SavePrefs(shared.Prefs{HideInternalTopics: k.hideInternal})
	k.applyFilters(true)
	return nil
}

// --- multi-select (TP-22) ---

// syncSelectionFlags mirrors the provider's selection map onto each item so the
// row renderer (a free function) can draw the selection marker.
func (k *KafuiContentProvider) syncSelectionFlags() {
	for _, item := range k.allItems {
		if tri, _, ok := topicItemFrom(item); ok {
			tri.selected = k.selected[tri.id]
		}
	}
}

func (k *KafuiContentProvider) rebuildAfterSelectionChange() {
	k.syncSelectionFlags()
	k.allRows = convertItemsToRows(k.allItems, "", 0)
	if k.isFiltered {
		k.reapplyFilter()
	} else {
		k.updateTableForCurrentPage()
	}
}

// toggleTopicSelection toggles selection on the highlighted row. Internal topics
// are not selectable.
func (k *KafuiContentProvider) toggleTopicSelection() {
	tri, _, ok := topicItemFrom(k.GetSelectedResourceItem())
	if !ok || tri.isInternal || isInternalTopicName(tri.id) {
		return
	}
	if k.selected[tri.id] {
		delete(k.selected, tri.id)
	} else {
		k.selected[tri.id] = true
	}
	k.rebuildAfterSelectionChange()
}

// selectAllVisibleTopics selects every non-internal topic in the active view.
func (k *KafuiContentProvider) selectAllVisibleTopics() {
	for _, item := range k.activeItems() {
		if tri, _, ok := topicItemFrom(item); ok && !tri.isInternal && !isInternalTopicName(tri.id) {
			k.selected[tri.id] = true
		}
	}
	k.rebuildAfterSelectionChange()
}

// clearTopicSelection drops all selected topics.
func (k *KafuiContentProvider) clearTopicSelection() {
	k.selected = map[string]bool{}
	k.rebuildAfterSelectionChange()
}

// getSelectedTopicNames returns the selected topic names in deterministic order.
func (k *KafuiContentProvider) getSelectedTopicNames() []string {
	names := make([]string, 0, len(k.selected))
	for n := range k.selected {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// statusHint surfaces a short informational message (a disabled-action hint).
func statusHint(message string) tea.Cmd {
	return core.NewNotification(core.StatusInfo, "Topics", message)
}

// --- CSV export (TP-17) ---

// exportTopicsCSV writes ALL topics in the current filtered/visibility/sort view
// to a timestamped CSV, fetching sizes/details via a full fan-out on this explicit
// action only. The absolute path is reported via a notification.
func (k *KafuiContentProvider) exportTopicsCSV() tea.Cmd {
	items := k.activeItems()
	names := make([]string, 0, len(items))
	for _, item := range items {
		if tri, _, ok := topicItemFrom(item); ok {
			names = append(names, tri.id)
		}
	}
	if len(names) == 0 {
		return nil
	}
	ds := k.dataSource
	ctx := ds.GetContext()
	return func() tea.Msg {
		sizes, _ := ds.GetTopicSizes(names)
		rows := make([]shared.TopicCSVRow, 0, len(names))
		for _, name := range names {
			row := shared.TopicCSVRow{Name: name, MessageCount: -1, Size: -1}
			if sz, ok := sizes[name]; ok {
				row.Size = sz
			}
			if d, err := ds.GetTopicDetails(name); err == nil {
				row.Partitions = int32(len(d.Partitions))
				row.ReplicationFactor = d.ReplicationFactor
				row.MessageCount = d.MessageCount()
				row.OutOfSync = d.UnderReplicatedPartitions
				row.Internal = d.IsInternal
			}
			rows = append(rows, row)
		}
		filename := fmt.Sprintf("kafui-topics-%s-%s.csv", ctx, time.Now().Format("20060102-150405"))
		f, err := os.Create(filename)
		if err != nil {
			return core.NotifyError("CSV export failed", err)()
		}
		defer f.Close()
		if werr := shared.WriteTopicCSV(f, rows); werr != nil {
			return core.NotifyError("CSV export failed", werr)()
		}
		abs, _ := filepath.Abs(filename)
		return core.NotificationMsg{Severity: core.StatusInfo, Title: "Topics exported", Message: abs}
	}
}

// --- create / clone form (TP-18, TP-19) ---

var topicNameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,249}$`)

func topicNameValidator(v string) error {
	if !topicNameRe.MatchString(v) {
		return fmt.Errorf("allowed: a-z A-Z 0-9 . _ - (1-249 chars)")
	}
	return nil
}

func partitionCountValidator(v string) error {
	if v == "" {
		return nil // Required handles emptiness
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("must be a whole number")
	}
	if n < 1 {
		return fmt.Errorf("must be >= 1")
	}
	return nil
}

func retentionMsValidator(v string) error {
	if v == "" {
		return nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fmt.Errorf("must be a whole number")
	}
	if n < -1 {
		return fmt.Errorf("must be >= -1 (-1 = unlimited)")
	}
	return nil
}

// topicFormDefaults holds prefill values for the create/clone form.
type topicFormDefaults struct {
	name              string
	partitions        string
	replicationFactor string
	cleanupPolicy     string
	retentionMs       string
	maxMessageBytes   string
	minInsyncReplicas string
}

func buildTopicForm(d topicFormDefaults) *form.Form {
	if d.cleanupPolicy == "" {
		d.cleanupPolicy = "delete"
	}
	return form.New([]form.Field{
		{Name: "name", Label: "Name", Type: form.Text, Required: true, Default: d.name, Validator: topicNameValidator},
		{Name: "partitions", Label: "Partitions", Type: form.Numeric, Required: true, Default: d.partitions, Validator: partitionCountValidator},
		{Name: "replication_factor", Label: "Replication Factor (empty = cluster default)", Type: form.Numeric, Default: d.replicationFactor},
		{Name: "cleanup.policy", Label: "cleanup.policy", Type: form.Select, Options: []string{"delete", "compact", "compact,delete"}, Default: d.cleanupPolicy},
		{Name: "retention.ms", Label: "retention.ms (-1 = unlimited)", Type: form.Text, Default: d.retentionMs, Validator: retentionMsValidator},
		{Name: "max.message.bytes", Label: "max.message.bytes", Type: form.Numeric, Default: d.maxMessageBytes},
		{Name: "min.insync.replicas", Label: "min.insync.replicas", Type: form.Numeric, Default: d.minInsyncReplicas},
	})
}

// openCreateTopicForm opens the create form as an overlay.
func (k *KafuiContentProvider) openCreateTopicForm() tea.Cmd {
	k.topicForm = buildTopicForm(topicFormDefaults{})
	k.showTopicForm = true
	return k.topicForm.Focus()
}

// openCloneTopicForm opens the create form prefilled from the highlighted topic's
// details + non-default config. Disabled (hint only) when != 1 topic is selected.
func (k *KafuiContentProvider) openCloneTopicForm() tea.Cmd {
	if len(k.selected) > 1 {
		return statusHint("clone requires exactly one topic; clear the selection first")
	}
	tri, _, ok := topicItemFrom(k.GetSelectedResourceItem())
	if !ok {
		return statusHint("no topic selected to clone")
	}
	name := tri.id
	defaults := topicFormDefaults{name: name}
	if d, err := k.dataSource.GetTopicDetails(name); err == nil {
		defaults.partitions = strconv.Itoa(len(d.Partitions))
		defaults.replicationFactor = strconv.Itoa(int(d.ReplicationFactor))
	}
	if cfg, err := k.dataSource.GetTopicConfig(name); err == nil {
		for _, e := range cfg {
			if e.Sensitive || e.Value == e.Default {
				continue // defaults and sensitive entries are not copied
			}
			switch e.Name {
			case "cleanup.policy":
				defaults.cleanupPolicy = e.Value
			case "retention.ms":
				defaults.retentionMs = e.Value
			case "max.message.bytes":
				defaults.maxMessageBytes = e.Value
			case "min.insync.replicas":
				defaults.minInsyncReplicas = e.Value
			}
		}
	}
	k.topicForm = buildTopicForm(defaults)
	k.showTopicForm = true
	return k.topicForm.Focus()
}

// handleTopicFormSubmit builds the CreateTopic request from submitted values,
// omitting empty config entries, and dispatches the call.
func (k *KafuiContentProvider) handleTopicFormSubmit(values map[string]string) tea.Cmd {
	name := values["name"]
	parts, _ := strconv.Atoi(values["partitions"])
	rf := int16(-1) // empty -> cluster default
	if v := strings.TrimSpace(values["replication_factor"]); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rf = int16(n)
		}
	}
	configs := map[string]*string{}
	for _, key := range []string{"cleanup.policy", "retention.ms", "max.message.bytes", "min.insync.replicas"} {
		val := strings.TrimSpace(values[key])
		if val == "" {
			continue // omit empty-valued config entries
		}
		v := val
		configs[key] = &v
	}
	ds := k.dataSource
	return func() tea.Msg {
		return topicCreatedMsg{name: name, err: ds.CreateTopic(name, int32(parts), rf, configs)}
	}
}

// --- row mutations (TP-20, TP-21, TP-22) ---

// selectedTopicName returns the highlighted topic's name, or "" when none.
func (k *KafuiContentProvider) selectedTopicName() string {
	name := k.getItemID(k.GetSelectedResourceItem())
	if name == "" || name == "unknown" {
		return ""
	}
	return name
}

// deleteSelectedTopics deletes the highlighted topic, or batch-deletes the current
// multi-selection when >= 2 topics are selected.
func (k *KafuiContentProvider) deleteSelectedTopics() tea.Cmd {
	if len(k.selected) >= 2 {
		return k.batchDeleteTopics()
	}
	name := k.selectedTopicName()
	if name == "" {
		return nil
	}
	if isInternalTopicName(name) {
		return statusHint("internal topics cannot be deleted")
	}
	if enabled, err := k.dataSource.IsTopicDeletionEnabled(); err == nil && !enabled {
		return statusHint("topic deletion is disabled on this cluster")
	}
	ds := k.dataSource
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Delete topic",
			Message:      fmt.Sprintf("Delete topic %q? All data will be lost and this cannot be undone.", name),
			Danger:       true,
			ConfirmLabel: "Delete",
			OnConfirm:    func() tea.Msg { return topicDeletedMsg{name: name, err: ds.DeleteTopic(name)} },
		}
	}
}

// recreateSelectedTopic recreates (delete + create) the highlighted topic.
func (k *KafuiContentProvider) recreateSelectedTopic() tea.Cmd {
	name := k.selectedTopicName()
	if name == "" {
		return nil
	}
	if isInternalTopicName(name) {
		return statusHint("internal topics cannot be recreated")
	}
	ds := k.dataSource
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Recreate topic",
			Message:      fmt.Sprintf("Recreate topic %q? It will be deleted and recreated, discarding all messages.", name),
			Danger:       true,
			ConfirmLabel: "Recreate",
			OnConfirm:    func() tea.Msg { return topicRecreatedMsg{name: name, err: ds.RecreateTopic(name)} },
		}
	}
}

// purgeSelectedTopics clears messages on the highlighted topic, or batch-purges the
// multi-selection when >= 2 topics are selected.
func (k *KafuiContentProvider) purgeSelectedTopics() tea.Cmd {
	if len(k.selected) >= 2 {
		return k.batchPurgeTopics()
	}
	name := k.selectedTopicName()
	if name == "" {
		return nil
	}
	if !k.topicAllowsDelete(name) {
		return statusHint("cleanup.policy must include 'delete' to clear messages")
	}
	ds := k.dataSource
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Clear messages",
			Message:      fmt.Sprintf("Clear all messages in topic %q? This cannot be undone.", name),
			Danger:       true,
			ConfirmLabel: "Clear",
			OnConfirm:    func() tea.Msg { return topicPurgedMsg{name: name, err: ds.PurgeTopicMessages(name, -1)} },
		}
	}
}

// topicAllowsDelete reports whether a topic's cleanup.policy permits message
// deletion. Unknown (fetch failed) is treated as allowed so the datasource can
// reject with its typed error.
func (k *KafuiContentProvider) topicAllowsDelete(name string) bool {
	cfg, err := k.dataSource.GetTopicConfig(name)
	if err != nil {
		return true
	}
	for _, e := range cfg {
		if e.Name != "cleanup.policy" {
			continue
		}
		for _, p := range strings.Split(e.Value, ",") {
			if strings.TrimSpace(p) == "delete" {
				return true
			}
		}
		return false
	}
	return true
}

// truncateNameList joins topic names for a confirm message, truncating the tail.
func truncateNameList(names []string) string {
	const max = 5
	if len(names) <= max {
		return strings.Join(names, ", ")
	}
	return strings.Join(names[:max], ", ") + fmt.Sprintf(", … (+%d more)", len(names)-max)
}

func (k *KafuiContentProvider) batchDeleteTopics() tea.Cmd {
	names := k.getSelectedTopicNames()
	if len(names) == 0 {
		return nil
	}
	if enabled, err := k.dataSource.IsTopicDeletionEnabled(); err == nil && !enabled {
		return statusHint("topic deletion is disabled on this cluster")
	}
	ds := k.dataSource
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Delete topics",
			Message:      fmt.Sprintf("Delete %d topics? This cannot be undone.\n%s", len(names), truncateNameList(names)),
			Danger:       true,
			ConfirmLabel: "Delete all",
			OnConfirm: func() tea.Msg {
				var failures []string
				for _, n := range names {
					if e := ds.DeleteTopic(n); e != nil {
						failures = append(failures, n+": "+e.Error())
					}
				}
				return topicBatchResultMsg{action: "delete", total: len(names), failures: failures}
			},
		}
	}
}

func (k *KafuiContentProvider) batchPurgeTopics() tea.Cmd {
	names := k.getSelectedTopicNames()
	if len(names) == 0 {
		return nil
	}
	ds := k.dataSource
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Clear messages",
			Message:      fmt.Sprintf("Clear all messages in %d topics? This cannot be undone.\n%s", len(names), truncateNameList(names)),
			Danger:       true,
			ConfirmLabel: "Clear all",
			OnConfirm: func() tea.Msg {
				var failures []string
				for _, n := range names {
					if e := ds.PurgeTopicMessages(n, -1); e != nil {
						failures = append(failures, n+": "+e.Error())
					}
				}
				return topicBatchResultMsg{action: "purge", total: len(names), failures: failures}
			},
		}
	}
}
