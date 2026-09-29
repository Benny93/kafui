package mainpage

import (
	"sort"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
)

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
	k.filteredItems = []interface{}{}
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

// refreshCurrentResource reloads the showing resource on explicit request.
// Unlike the timer refresh it also re-fetches the known topic message counts.
func (k *KafuiContentProvider) refreshCurrentResource() tea.Cmd {
	k.countsFetchedAt = time.Time{}
	return k.loadCurrentResource()
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
	if k.isFiltered {
		k.reapplyFilter()
	} else {
		k.updateTableForCurrentPage()
	}
}

// searchDetailsDelay is how long typing in the filter must pause before the
// rows it brought onto the page are enriched, so a burst of keystrokes costs
// one fetch rather than one per key.
const searchDetailsDelay = 300 * time.Millisecond

// searchSettledMsg fires searchDetailsDelay after a filter keystroke.
type searchSettledMsg struct{ query string }

// pageDetailsWhenSearchSettles schedules reloadPageDetails for once typing in
// the filter pauses on the current query.
func (k *KafuiContentProvider) pageDetailsWhenSearchSettles() tea.Cmd {
	query := k.currentFilter
	return tea.Tick(searchDetailsDelay, func(time.Time) tea.Msg {
		return searchSettledMsg{query: query}
	})
}

// reloadPageDetails re-runs the lazy enrichment after a search, sort or
// visibility change put different rows on the current page; without it those
// rows kept their placeholders until the user paged away and back. Brokers are
// left out: their stats cover every broker in one call and are already loaded.
func (k *KafuiContentProvider) reloadPageDetails() tea.Cmd {
	if k.isBrokerResource() {
		return nil
	}
	return k.loadPageDetails()
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
	// Keep the user's chosen sort: the name order above is only the default,
	// and without this every timer refresh snapped the rows back to it.
	if msg.ResourceType == TopicResourceType {
		k.applyTopicSort()
	}

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
