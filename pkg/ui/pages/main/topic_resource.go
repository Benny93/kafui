package mainpage

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
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

// topicOSRCell renders the out-of-sync replica count, styled in the alert
// colour when > 0. Zero is the healthy value and the common one: a cluster with
// every replica in its ISR reports 0 for every topic, which is correct, not a
// failure to load — that renders as N/A.
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

// loadTopicDetailsExt enriches ONLY the visible page of topics with OSR and
// size, using two batched calls. Off-screen topics are never fetched,
// preserving the visible-page-only discipline.
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
		// Two batched calls for the whole page. This used to be one
		// GetTopicDetails per topic, each opening its own cluster admin AND its
		// own client and then fetching offsets sequentially per partition — so
		// a 50-topic page cost ~100 connections and several hundred round trips
		// to fill two columns, neither of which needs offsets at all. On a
		// remote cluster that was the "…" that seemed to hang forever.
		sizes, _ := ds.GetTopicSizes(names)
		health, _ := ds.GetTopicHealth(names)

		out := make(map[string]topicExtInfo, len(names))
		for _, name := range names {
			// -1 means "not loaded" and renders as N/A, which is what an
			// unreachable broker must show — never a confident 0.
			info := topicExtInfo{outOfSync: -1, size: -1}
			if sz, ok := sizes[name]; ok {
				info.size = sz
			}
			if h, ok := health[name]; ok {
				info.outOfSync = h.OutOfSyncReplicas
				info.isInternal = h.IsInternal
			}
			out[name] = info
		}
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
	return k.reloadPageDetails()
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

// topicShape is the partition count and replication factor of a topic; -1
// means not loaded yet.
type topicShape struct {
	partitions        int32
	replicationFactor int16
}

// exportTopicsCSV writes ALL topics in the current filtered/visibility/sort view
// to a timestamped CSV. The absolute path is reported via a notification.
//
// It costs a fixed number of batched calls whatever the topic count: sizes,
// health and message counts for all names at once, plus one GetTopics when some
// rows have not loaded their partitions yet. (It used to call GetTopicDetails
// per topic, a fresh admin and client each, which on a large cluster meant
// thousands of connections for one export.)
func (k *KafuiContentProvider) exportTopicsCSV() tea.Cmd {
	items := k.activeItems()
	names := make([]string, 0, len(items))
	// Snapshot what the list knows now; the Cmd runs off the Update goroutine
	// and must not read the live items.
	shapes := make(map[string]topicShape, len(items))
	missingShape := false
	for _, item := range items {
		if tri, _, ok := topicItemFrom(item); ok {
			names = append(names, tri.id)
			shapes[tri.id] = topicShape{tri.partitions, tri.replicationFactor}
			if tri.partitions < 0 {
				missingShape = true
			}
		}
	}
	if len(names) == 0 {
		return nil
	}
	ds := k.dataSource
	ctx := ds.GetContext()
	return func() tea.Msg {
		if missingShape {
			if topics, err := ds.GetTopics(); err == nil {
				for name, s := range shapes {
					if t, ok := topics[name]; ok && s.partitions < 0 {
						shapes[name] = topicShape{t.NumPartitions, t.ReplicationFactor}
					}
				}
			}
		}
		sizes, _ := ds.GetTopicSizes(names)
		health, _ := ds.GetTopicHealth(names)
		countInput := make(map[string]int32, len(shapes))
		for name, s := range shapes {
			if s.partitions > 0 {
				countInput[name] = s.partitions
			}
		}
		counts, _ := ds.GetTopicMessageCounts(countInput)

		rows := make([]shared.TopicCSVRow, 0, len(names))
		for _, name := range names {
			row := shared.TopicCSVRow{Name: name, MessageCount: -1, Size: -1}
			if s := shapes[name]; s.partitions >= 0 {
				row.Partitions = s.partitions
				row.ReplicationFactor = s.replicationFactor
			}
			if c, ok := counts[name]; ok {
				row.MessageCount = c
			}
			if sz, ok := sizes[name]; ok {
				row.Size = sz
			}
			if h, ok := health[name]; ok {
				row.OutOfSync = h.UnderReplicatedPartitions
				row.Internal = h.IsInternal
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
// The lookups run in the returned Cmd, not in Update, so a slow broker does not
// freeze the UI; the form opens when topicCloneDefaultsMsg arrives.
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
	// The list already knows the shape once its details have loaded, which
	// saves a per-topic describe plus offset lookups.
	knownShape := tri.partitions >= 0 && tri.replicationFactor >= 0
	if knownShape {
		defaults.partitions = strconv.Itoa(int(tri.partitions))
		defaults.replicationFactor = strconv.Itoa(int(tri.replicationFactor))
	}
	ds := k.dataSource
	return func() tea.Msg {
		if !knownShape {
			if d, err := ds.GetTopicDetails(name); err == nil {
				defaults.partitions = strconv.Itoa(len(d.Partitions))
				defaults.replicationFactor = strconv.Itoa(int(d.ReplicationFactor))
			}
		}
		if cfg, err := ds.GetTopicConfig(name); err == nil {
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
		return topicCloneDefaultsMsg{defaults: defaults}
	}
}

// openTopicFormWith shows the create form prefilled with defaults.
func (k *KafuiContentProvider) openTopicFormWith(defaults topicFormDefaults) tea.Cmd {
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
	ds := k.dataSource
	return func() tea.Msg {
		// Checked here, not in Update: the first check per cluster is a
		// broker round trip.
		if enabled, err := ds.IsTopicDeletionEnabled(); err == nil && !enabled {
			return statusHint("topic deletion is disabled on this cluster")()
		}
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
	ds := k.dataSource
	return func() tea.Msg {
		if !topicAllowsDelete(ds, name) {
			return statusHint("cleanup.policy must include 'delete' to clear messages")()
		}
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
func topicAllowsDelete(ds api.KafkaDataSource, name string) bool {
	cfg, err := ds.GetTopicConfig(name)
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
	ds := k.dataSource
	return func() tea.Msg {
		if enabled, err := ds.IsTopicDeletionEnabled(); err == nil && !enabled {
			return statusHint("topic deletion is disabled on this cluster")()
		}
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
