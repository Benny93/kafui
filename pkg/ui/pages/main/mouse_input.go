package mainpage

import (
	"github.com/Benny93/kafui/pkg/ui/core"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

// handleMouse handles wheel, hover and click gestures on the list.
func (k *KafuiContentProvider) handleMouse(msg tea.MouseMsg) tea.Cmd {
	var cmds []tea.Cmd

	// Gesture vocabulary from the controls spec: the wheel moves the view
	// and never the selection; a click selects; a click on the row that is
	// already selected activates it. Right-click is the shell's — it opens
	// the actions menu for whatever is under the pointer.
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		// The list is paginated to exactly what fits, so a page is the
		// scroll unit. The cursor is not dragged along row by row.
		if k.pagination.PrevPage() {
			k.updateTableForCurrentPageAndReset()
			cmds = append(cmds, k.loadPageDetails())
		}
	case tea.MouseButtonWheelDown:
		if k.pagination.NextPage() {
			k.updateTableForCurrentPageAndReset()
			cmds = append(cmds, k.loadPageDetails())
		}
	case tea.MouseButtonNone:
		// Hover: the row under the pointer highlights, so what is clickable
		// is discoverable without clicking.
		if !core.IsHover(msg) {
			break
		}
		if row, ok := k.rowAtMouse(msg); ok {
			k.resourcesTable = k.resourcesTable.WithHighlightedRow(row)
		}

	case tea.MouseButtonLeft:
		if !core.IsLeftRelease(msg) {
			break
		}
		// A click on the header row sorts by THAT column; clicking the
		// active sort column again reverses it.
		if relX, ok := k.clickedHeader(msg); ok {
			resType := TopicResourceType
			if k.currentResource != nil {
				resType = k.currentResource.GetType()
			}
			if col, hit := columnAtX(createResourceTableColumns(resType), k.tableWidth, relX); hit {
				k.setSortColumn(col)
				cmds = append(cmds, k.reloadPageDetails())
			}
			return tea.Batch(cmds...)
		}
		row, ok := k.rowAtMouse(msg)
		if !ok {
			break
		}
		double := k.clicks.Click(msg)
		if double || row == k.resourcesTable.GetHighlightedRowIndex() {
			// Both gestures activate: a double click, and a click on the
			// row that is already selected.
			k.resourcesTable = k.resourcesTable.WithHighlightedRow(row)
			return k.handleResourceSelection()
		}
		k.resourcesTable = k.resourcesTable.WithHighlightedRow(row)
	}

	return tea.Batch(cmds...)
}

// clickedHeader reports whether the pointer is on the table's header row, and
// the X offset within the table so the column can be resolved.
func (k *KafuiContentProvider) clickedHeader(msg tea.MouseMsg) (int, bool) {
	z := zone.Get("resource-table")
	if z == nil || !z.InBounds(msg) {
		return 0, false
	}
	relX, relY := z.Pos(msg)
	// Row 0 is the top border, row 1 the header labels.
	return relX, relY == 1
}

// rowAtMouse maps a mouse position to a page-local row index, or reports false
// when the pointer is not over a data row.
func (k *KafuiContentProvider) rowAtMouse(msg tea.MouseMsg) (int, bool) {
	z := zone.Get("resource-table")
	if z == nil || !z.InBounds(msg) {
		return 0, false
	}
	_, relY := z.Pos(msg)
	// The table renders border + header + separator before the first data row.
	const headerLines = 3
	row := relY - headerLines
	if row < 0 {
		return 0, false
	}
	activeItems := k.allItems
	if k.isFiltered {
		activeItems = k.filteredItems
	}
	if row >= len(k.pagination.GetCurrentPageItems(activeItems)) {
		return 0, false
	}
	return row, true
}
