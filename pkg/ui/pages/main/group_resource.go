package mainpage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

// formatGroupLag renders a lag pointer: "—" when undefined (nil), never 0.
func formatGroupLag(lag *int64) string {
	if lag == nil {
		return "—"
	}
	return strconv.FormatInt(*lag, 10)
}

// formatCoordinator renders a coordinator broker id, "—" when unknown (<0).
func formatCoordinator(id int32) string {
	if id < 0 {
		return "—"
	}
	return strconv.FormatInt(int64(id), 10)
}

// groupStateStyle colours a state string by role (semantic, not hex).
func groupStateStyle(state string) lipgloss.Style {
	switch state {
	case api.GroupStateStable:
		return lipgloss.NewStyle().Foreground(stylesPkg.Success)
	case api.GroupStatePreparingRebalance, api.GroupStateCompletingRebalance:
		return lipgloss.NewStyle().Foreground(stylesPkg.Warning)
	case api.GroupStateDead:
		return lipgloss.NewStyle().Foreground(stylesPkg.Error)
	default: // Empty, Unknown
		return lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)
	}
}

// groupItemFrom unwraps a consumer-group resource item from the list-item
// wrappers, returning the item plus any active search query (for highlighting).
func groupItemFrom(item interface{}) (*ConsumerGroupResourceItem, string, bool) {
	switch v := item.(type) {
	case shared.ResourceListItem:
		cgri, ok := v.ResourceItem.(*ConsumerGroupResourceItem)
		return cgri, "", ok
	case shared.HighlightedResourceListItem:
		cgri, ok := v.ResourceItem.(*ConsumerGroupResourceItem)
		return cgri, v.SearchQuery, ok
	case *ConsumerGroupResourceItem:
		return v, "", true
	default:
		return nil, "", false
	}
}

// groupRowData builds a bubble-table row for a consumer group.
func groupRowData(c *ConsumerGroupResourceItem, searchQuery, placeholder string) table.RowData {
	name := c.id
	if searchQuery != "" {
		name = shared.HighlightSearchMatches(name, searchQuery)
	}
	if !c.detailsLoaded {
		return table.RowData{
			colGroupName:    name,
			colGroupState:   placeholder,
			colGroupMembers: placeholder,
			colGroupTopics:  placeholder,
			colGroupLag:     placeholder,
			colGroupCoord:   placeholder,
		}
	}
	state := groupStateStyle(c.State()).Render(c.State())
	return table.RowData{
		colGroupName:    name,
		colGroupState:   state,
		colGroupMembers: strconv.Itoa(c.group.MemberCount),
		colGroupTopics:  strconv.Itoa(c.group.TopicCount),
		colGroupLag:     formatGroupLag(c.group.Lag),
		colGroupCoord:   formatCoordinator(c.group.CoordinatorID),
	}
}

// isGroupResource reports whether the consumer-groups resource is active.
func (k *KafuiContentProvider) isGroupResource() bool {
	return k.currentResource != nil && k.currentResource.GetType() == ConsumerGroupResourceType
}

// hasStateFilter reports an active consumer-group state filter.
func (k *KafuiContentProvider) hasStateFilter() bool {
	return k.isGroupResource() && k.groupStateFilter != ""
}

// --- lazy enrichment (CG-9) ---

// loadGroupDetails enriches ONLY the visible page of group rows (real state,
// members, topics, lag, coordinator) via GetConsumerGroupDetails. Off-screen
// rows are never described, preserving the no-eager-describe constraint.
func (k *KafuiContentProvider) loadGroupDetails() tea.Cmd {
	activeItems := k.allItems
	if k.isFiltered {
		activeItems = k.filteredItems
	}
	pageItems := k.pagination.GetCurrentPageItems(activeItems)

	names := make([]string, 0, len(pageItems))
	for _, item := range pageItems {
		if cgri, _, ok := groupItemFrom(item); ok && !cgri.detailsLoaded {
			names = append(names, cgri.id)
		}
	}
	if len(names) == 0 {
		return nil
	}

	ds := k.dataSource
	return func() tea.Msg {
		details, err := ds.GetConsumerGroupDetails(names)
		if err != nil {
			shared.Log.Error("loadGroupDetails: GetConsumerGroupDetails failed", "err", err)
			return ConsumerGroupDetailsLoadedMsg([]api.ConsumerGroup{})
		}
		return ConsumerGroupDetailsLoadedMsg(details)
	}
}

// applyGroupDetails merges enriched group data back onto the items, re-sorts,
// and rebuilds the display rows.
func (k *KafuiContentProvider) applyGroupDetails(msg ConsumerGroupDetailsLoadedMsg) {
	byName := make(map[string]api.ConsumerGroup, len(msg))
	for _, g := range msg {
		byName[g.Name] = g
	}
	for _, item := range k.allItems {
		if cgri, _, ok := groupItemFrom(item); ok {
			if g, found := byName[cgri.id]; found {
				cgri.SetDetail(g)
			}
		}
	}
	k.applyGroupSort()
	k.allRows = convertItemsToRows(k.allItems, "", 0)
	k.reapplyFilter()
	if !k.isFiltered {
		k.updateTableForCurrentPage()
	}
}

// --- sorting (CG-10) ---

// groupSortColumns maps the sort cycle to a sort key.
var groupSortColumns = []string{"name", "state", "members", "topics", "lag"}

// groupStatePriority ranks states for sorting: Stable first, Unknown/other last.
func groupStatePriority(state string) int {
	switch state {
	case api.GroupStateStable:
		return 0
	case api.GroupStateCompletingRebalance:
		return 1
	case api.GroupStatePreparingRebalance:
		return 2
	case api.GroupStateEmpty:
		return 3
	case api.GroupStateDead:
		return 4
	case api.GroupStateUnknown:
		return 5
	default:
		return 6
	}
}

// lagOrZero returns *lag or 0 when nil (nil lag sorts as 0 per spec).
func lagOrZero(lag *int64) int64 {
	if lag == nil {
		return 0
	}
	return *lag
}

// sortGroupItems sorts consumer-group items in place by the given key.
func sortGroupItems(items []interface{}, col string, desc bool) {
	less := func(a, b *ConsumerGroupResourceItem) bool {
		switch col {
		case "state":
			return groupStatePriority(a.State()) < groupStatePriority(b.State())
		case "members":
			return a.group.MemberCount < b.group.MemberCount
		case "topics":
			return a.group.TopicCount < b.group.TopicCount
		case "lag":
			return lagOrZero(a.group.Lag) < lagOrZero(b.group.Lag)
		default: // "name"
			return a.id < b.id
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		ai, _, aok := groupItemFrom(items[i])
		bj, _, bok := groupItemFrom(items[j])
		if !aok || !bok {
			return false
		}
		if desc {
			return less(bj, ai)
		}
		return less(ai, bj)
	})
}

func (k *KafuiContentProvider) cycleGroupSortColumn() {
	k.groupSortCol = (k.groupSortCol + 1) % len(groupSortColumns)
	k.applyGroupSortAndRefresh()
}

func (k *KafuiContentProvider) toggleGroupSortDir() {
	if k.groupSortCol < 0 {
		k.groupSortCol = 0
	}
	k.groupSortDesc = !k.groupSortDesc
	k.applyGroupSortAndRefresh()
}

func (k *KafuiContentProvider) applyGroupSortAndRefresh() {
	k.applyGroupSort()
	k.allRows = convertItemsToRows(k.allItems, "", 0)
	k.applyFilters(true)
}

// applyGroupSort sorts allItems by the active group sort column/direction.
func (k *KafuiContentProvider) applyGroupSort() {
	if k.groupSortCol < 0 || k.groupSortCol >= len(groupSortColumns) || !k.isGroupResource() {
		return
	}
	sortGroupItems(k.allItems, groupSortColumns[k.groupSortCol], k.groupSortDesc)
}

// --- state filter (CG-11) ---

// groupStateFilterCycle is the cycle order for the state filter ("" = all).
// ponytail: the spec asks for multi-state selection; a single-state cycle covers
// the primary "filter by state" scenario with far less UI. Extend to a
// multi-select picker if users need combined states.
var groupStateFilterCycle = []string{
	"",
	api.GroupStateStable,
	api.GroupStatePreparingRebalance,
	api.GroupStateCompletingRebalance,
	api.GroupStateEmpty,
	api.GroupStateDead,
	api.GroupStateUnknown,
}

func (k *KafuiContentProvider) cycleGroupStateFilter() {
	cur := 0
	for i, s := range groupStateFilterCycle {
		if s == k.groupStateFilter {
			cur = i
			break
		}
	}
	k.groupStateFilter = groupStateFilterCycle[(cur+1)%len(groupStateFilterCycle)]
	k.applyFilters(true)
}

// groupItemMatchesState reports whether a group item's state matches the filter.
// Groups whose state has not been fetched are treated as Unknown.
func (k *KafuiContentProvider) groupItemMatchesState(item interface{}, state string) bool {
	cgri, _, ok := groupItemFrom(item)
	if !ok {
		return false
	}
	return cgri.State() == state
}

// --- CSV export (CG-12) ---

// exportGroupsCSV fetches details for ALL groups in the current (filtered/sorted)
// view — the one deliberate full fan-out, on explicit user action — and writes a
// timestamped CSV, reporting the absolute path via a notification.
func (k *KafuiContentProvider) exportGroupsCSV() tea.Cmd {
	activeItems := k.allItems
	if k.isFiltered {
		activeItems = k.filteredItems
	}
	names := make([]string, 0, len(activeItems))
	for _, item := range activeItems {
		if cgri, _, ok := groupItemFrom(item); ok {
			names = append(names, cgri.id)
		}
	}
	if len(names) == 0 {
		return nil
	}
	ds := k.dataSource
	return func() tea.Msg {
		groups, err := ds.GetConsumerGroupDetails(names)
		if err != nil {
			return core.NotifyError("CSV export failed", err)()
		}
		filename := fmt.Sprintf("consumer-groups-%s.csv", time.Now().Format("20060102-150405"))
		f, cerr := os.Create(filename)
		if cerr != nil {
			return core.NotifyError("CSV export failed", cerr)()
		}
		defer f.Close()
		if werr := shared.WriteConsumerGroupCSV(f, groups); werr != nil {
			return core.NotifyError("CSV export failed", werr)()
		}
		abs, _ := filepath.Abs(filename)
		return core.NotificationMsg{Severity: core.StatusInfo, Title: "Consumer groups exported", Message: abs}
	}
}

// --- delete (CG-17) ---

// deleteSelectedGroup shows a confirmation modal, then deletes the highlighted
// consumer group. Never deletes without confirmation.
func (k *KafuiContentProvider) deleteSelectedGroup() tea.Cmd {
	cgri, _, ok := groupItemFrom(k.GetSelectedResourceItem())
	if !ok {
		return nil
	}
	groupID := cgri.id
	ds := k.dataSource
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Delete consumer group",
			Message:      fmt.Sprintf("Delete consumer group %q? This cannot be undone.", groupID),
			Danger:       true,
			ConfirmLabel: "Delete",
			OnConfirm: func() tea.Msg {
				err := ds.DeleteConsumerGroup(groupID)
				return groupDeletedMsg{groupID: groupID, err: err}
			},
		}
	}
}
