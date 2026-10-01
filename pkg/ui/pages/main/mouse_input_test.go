package mainpage

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mouseZones ensures the whole package run shares one bubblezone global
// manager: RenderContent zone-marks the resource table, and clicks / hovers
// only mean something once a manager exists to answer InBounds().
var mouseZones sync.Once

// mouseProvider returns a topics-view provider seeded with stub items named
// after the given list. Stubs carry partitions == -1 so the lazy detail
// loaders handleMouse fires stay inert (they return nil without touching the
// datasource) — the commands are only constructed here, never executed.
func mouseProvider(t *testing.T, names ...string) *KafuiContentProvider {
	t.Helper()
	mouseZones.Do(func() { zone.NewGlobal() })
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	k := NewKafuiContentProvider(ds)
	k.HandleContentUpdate(SwitchResourceMsg(TopicResourceType))

	items := make([]interface{}, 0, len(names))
	for _, n := range names {
		items = append(items, shared.ResourceListItem{ResourceItem: &TopicResourceItem{
			id:                n,
			partitions:        -1,
			replicationFactor: -1,
			messageCount:      -1,
			outOfSync:         -1,
			size:              -1,
		}})
	}
	k.allItems = items
	k.pagination.SetTotalItems(len(items))
	return k
}

// renderTable primes the table dimensions, builds the rows, then zone-scans a
// final render so the "resource-table" zone is registered. It returns the
// rendered lines and the absolute index of the header row (the row columnAtX
// and rowAtMouse measure their offsets from).
func renderTable(t *testing.T, k *KafuiContentProvider) (lines []string, headerIdx int) {
	t.Helper()
	k.RenderContent(120, 40)      // sets nameColumnWidth + tableWidth
	k.updateTableForCurrentPage() // build rows now that widths are known
	out := zone.Scan(k.RenderContent(120, 40))
	lines = strings.Split(out, "\n")

	for i, l := range lines {
		if strings.Contains(stripANSI(l), "Partitions") {
			return lines, i
		}
	}
	t.Fatal("header row not found in rendered table")
	return nil, -1
}

// dataRowY returns the absolute Y of page-local data row r, given the header's
// absolute Y. The table draws border, header, separator before the first data
// row, so row 0 sits two lines below the header.
func dataRowY(headerIdx, r int) int { return headerIdx + 2 + r }

func motion(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonNone, Action: tea.MouseActionMotion}
}

func release(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease}
}

func wheel(button tea.MouseButton) tea.MouseMsg {
	return tea.MouseMsg{X: 2, Y: 4, Button: button, Action: tea.MouseActionPress}
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEsc = true
		case inEsc && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'):
			inEsc = false
		case !inEsc:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// The wheel moves the page, never the cursor, and stops at the edges.
func TestHandleMouse_WheelPages(t *testing.T) {
	k := mouseProvider(t, "a", "b", "c", "d", "e")
	k.pagination.PerPage = 2 // 5 items -> 3 pages
	k.pagination.SetTotalItems(5)
	k.updateTableForCurrentPage()

	k.handleMouse(wheel(tea.MouseButtonWheelDown))
	assert.Equal(t, 1, k.pagination.Page, "wheel down advances a page")

	k.handleMouse(wheel(tea.MouseButtonWheelDown))
	assert.Equal(t, 2, k.pagination.Page)

	// At the last page the wheel is a no-op.
	k.handleMouse(wheel(tea.MouseButtonWheelDown))
	assert.Equal(t, 2, k.pagination.Page, "cannot page past the last page")

	k.handleMouse(wheel(tea.MouseButtonWheelUp))
	assert.Equal(t, 1, k.pagination.Page, "wheel up retreats a page")

	// Walk to the first page and confirm it clamps.
	k.handleMouse(wheel(tea.MouseButtonWheelUp))
	k.handleMouse(wheel(tea.MouseButtonWheelUp))
	assert.Equal(t, 0, k.pagination.Page, "cannot page before the first page")
}

// Hovering a data row highlights it; hovering chrome (not a row) does not.
func TestHandleMouse_Hover(t *testing.T) {
	k := mouseProvider(t, "alpha", "beta", "gamma")
	_, h := renderTable(t, k)

	require.Eventually(t, func() bool {
		z := zone.Get("resource-table")
		return z != nil && z.InBounds(motion(2, dataRowY(h, 1)))
	}, 2*time.Second, 10*time.Millisecond)

	k.handleMouse(motion(2, dataRowY(h, 1)))
	assert.Equal(t, 1, k.resourcesTable.GetHighlightedRowIndex(), "pointer motion highlights the row under it")

	// The separator line maps above row 0 -> not a data row, highlight preserved.
	before := k.resourcesTable.GetHighlightedRowIndex()
	k.handleMouse(motion(2, h+1)) // separator: relY 2 -> row < 0
	assert.Equal(t, before, k.resourcesTable.GetHighlightedRowIndex(), "hover on chrome leaves the highlight put")

	// A None-button press (not motion) is ignored entirely.
	k.handleMouse(tea.MouseMsg{X: 2, Y: dataRowY(h, 0), Button: tea.MouseButtonNone, Action: tea.MouseActionPress})
	assert.Equal(t, before, k.resourcesTable.GetHighlightedRowIndex())
}

// A header click sorts by the clicked column and requests a detail reload.
func TestHandleMouse_HeaderClickSorts(t *testing.T) {
	k := mouseProvider(t, "alpha", "beta")
	_, h := renderTable(t, k)

	require.Eventually(t, func() bool {
		z := zone.Get("resource-table")
		return z != nil && z.InBounds(release(2, h))
	}, 2*time.Second, 10*time.Millisecond)

	k.topicSortCol = -1
	cmd := k.handleMouse(release(2, h)) // x=2 lands on the flex Name column (index 0)
	assert.Equal(t, 0, k.topicSortCol, "header click sorts by that column")
	require.NotNil(t, cmd, "a sort change reloads the page details")
}

// A single click on an unselected row selects it without activating it.
func TestHandleMouse_RowClickSelects(t *testing.T) {
	k := mouseProvider(t, "alpha", "beta", "gamma")
	_, h := renderTable(t, k)
	y := dataRowY(h, 2)

	require.Eventually(t, func() bool {
		z := zone.Get("resource-table")
		return z != nil && z.InBounds(release(2, y))
	}, 2*time.Second, 10*time.Millisecond)

	cmd := k.handleMouse(release(2, y))
	assert.Equal(t, 2, k.resourcesTable.GetHighlightedRowIndex(), "click moves the selection")
	assert.Nil(t, cmd, "a first click only selects")

	// Left-press (not release) does nothing.
	k.handleMouse(tea.MouseMsg{X: 2, Y: dataRowY(h, 0), Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	assert.Equal(t, 2, k.resourcesTable.GetHighlightedRowIndex(), "press is ignored; only release acts")
}

// Clicking the already-selected row activates it, navigating to the detail page.
func TestHandleMouse_ClickSelectedRowActivates(t *testing.T) {
	k := mouseProvider(t, "alpha", "beta", "gamma")
	_, h := renderTable(t, k)
	y := dataRowY(h, 0) // the table starts highlighted on row 0

	require.Eventually(t, func() bool {
		z := zone.Get("resource-table")
		return z != nil && z.InBounds(release(2, y))
	}, 2*time.Second, 10*time.Millisecond)

	cmd := k.handleMouse(release(2, y))
	require.NotNil(t, cmd, "clicking the selected row activates it")
	msg := cmd()
	nav, ok := msg.(NavigateToResourceDetailMsg)
	require.True(t, ok, "activation navigates to the resource detail")
	assert.Equal(t, "alpha", nav.ResourceID)
}

// Two clicks on the same spot inside the double-click window activate.
func TestHandleMouse_DoubleClickActivates(t *testing.T) {
	k := mouseProvider(t, "alpha", "beta", "gamma")
	_, h := renderTable(t, k)
	y := dataRowY(h, 1) // start on a non-selected row so the first click only selects

	require.Eventually(t, func() bool {
		z := zone.Get("resource-table")
		return z != nil && z.InBounds(release(2, y))
	}, 2*time.Second, 10*time.Millisecond)

	first := k.handleMouse(release(2, y))
	assert.Nil(t, first, "first click selects")

	second := k.handleMouse(release(2, y)) // same spot, within 400ms -> double
	require.NotNil(t, second, "the second click completes a double click and activates")
	assert.Equal(t, "beta", second().(NavigateToResourceDetailMsg).ResourceID)
}

// Unrecognised buttons fall through to a no-op.
func TestHandleMouse_UnknownButtonIsNoop(t *testing.T) {
	k := mouseProvider(t, "alpha", "beta")
	cmd := k.handleMouse(tea.MouseMsg{X: 2, Y: 4, Button: tea.MouseButtonRight, Action: tea.MouseActionPress})
	assert.Nil(t, cmd)
}

// Without a registered zone the pointer helpers report "no hit", so a click
// outside the table neither sorts nor selects.
func TestRowAtMouseAndClickedHeaderOutOfBounds(t *testing.T) {
	k := mouseProvider(t, "alpha")
	_, h := renderTable(t, k)

	// A far-off coordinate is outside the table zone.
	far := release(5000, 5000)
	_, ok := k.clickedHeader(far)
	assert.False(t, ok, "a click outside the table is not a header click")
	_, ok = k.rowAtMouse(far)
	assert.False(t, ok, "a click outside the table maps to no row")

	// A point inside the table's padded blank area is in-bounds but past the
	// single data row, so it still maps to no row.
	blankY := dataRowY(h, 5)
	require.Eventually(t, func() bool {
		z := zone.Get("resource-table")
		return z != nil && z.InBounds(release(2, blankY))
	}, 2*time.Second, 10*time.Millisecond)
	_, ok = k.rowAtMouse(release(2, blankY))
	assert.False(t, ok, "a pointer below the last row maps to no row")
}
