package mainpage

import (
	"strings"

	"github.com/Benny93/kafui/pkg/ui/shared"
)

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
