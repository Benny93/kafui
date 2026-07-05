package datatable

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func cols() []Column {
	return []Column{
		{Title: "Name", Width: 20, Sortable: true},
		{Title: "Value", Width: 10},
	}
}

func rows() [][]string {
	return [][]string{
		{"banana", "1"},
		{"apple", "2"},
		{"cherry", "3"},
	}
}

func keyRunes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

// firstCol returns the first-column values of the currently displayed page.
func firstCol(t *Table) []string {
	out := make([]string, len(t.pageRows))
	for i, r := range t.pageRows {
		out[i] = r[0]
	}
	return out
}

func TestSortCycling(t *testing.T) {
	tbl := New(cols(), rows())

	// Initial: unsorted, no indicator.
	assert.Equal(t, SortNone, tbl.State().SortDir)
	assert.NotContains(t, tbl.View(), "▲")

	// s -> ascending.
	tbl.Update(keyRunes("s"))
	assert.Equal(t, SortAsc, tbl.State().SortDir)
	assert.Equal(t, 0, tbl.State().SortCol)
	assert.Equal(t, []string{"apple", "banana", "cherry"}, firstCol(tbl))
	assert.Contains(t, tbl.View(), "▲")

	// s -> descending.
	tbl.Update(keyRunes("s"))
	assert.Equal(t, SortDesc, tbl.State().SortDir)
	assert.Equal(t, []string{"cherry", "banana", "apple"}, firstCol(tbl))
	assert.Contains(t, tbl.View(), "▼")

	// s -> back to none.
	tbl.Update(keyRunes("s"))
	assert.Equal(t, SortNone, tbl.State().SortDir)
	assert.Equal(t, []string{"banana", "apple", "cherry"}, firstCol(tbl))
}

func TestSortNoopWithoutSortableColumn(t *testing.T) {
	c := []Column{{Title: "A", Width: 10}, {Title: "B", Width: 10}}
	tbl := New(c, rows())
	tbl.Update(keyRunes("s"))
	assert.Equal(t, SortNone, tbl.State().SortDir)
}

func TestPaginationBounds(t *testing.T) {
	many := [][]string{{"a"}, {"b"}, {"c"}, {"d"}, {"e"}} // 5 rows
	tbl := New([]Column{{Title: "X", Width: 10}}, many, WithPageSize(2))

	// 5 rows / 2 per page = 3 pages.
	assert.Equal(t, 3, tbl.pageCount(len(many)))
	assert.Equal(t, 1, tbl.State().Pagination.Page)

	// Can't go before the first page.
	tbl.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	assert.Equal(t, 1, tbl.State().Pagination.Page)

	// Advance to the last page.
	tbl.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.Equal(t, 2, tbl.State().Pagination.Page)
	tbl.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.Equal(t, 3, tbl.State().Pagination.Page)

	// Can't go past the last page.
	tbl.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.Equal(t, 3, tbl.State().Pagination.Page)

	assert.Contains(t, tbl.View(), "Page 3/3")
}

func TestEmptyState(t *testing.T) {
	tbl := New(cols(), nil, WithEmptyMessage("Nothing here"))
	assert.Contains(t, tbl.View(), "Nothing here")

	// Also shown when a filter matches nothing.
	tbl2 := New(cols(), rows(), WithFilter(0), WithEmptyMessage("Nothing here"))
	tbl2.Update(keyRunes("/"))
	tbl2.filterInput.SetValue("zzz")
	tbl2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Contains(t, tbl2.View(), "Nothing here")
}

func TestSelectionClearedOnPageChange(t *testing.T) {
	many := [][]string{{"a"}, {"b"}, {"c"}, {"d"}} // 4 rows, 2 pages of 2
	tbl := New([]Column{{Title: "X", Width: 10}}, many, WithPageSize(2), WithSelection("delete"))

	// Select the current row.
	tbl.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.Len(t, tbl.SelectedRows(), 1)
	assert.Contains(t, tbl.View(), "1 selected")
	assert.Contains(t, tbl.View(), "delete")

	// Changing page clears the selection.
	tbl.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.Empty(t, tbl.SelectedRows())
	assert.NotContains(t, tbl.View(), "selected")
}

func TestFilterResetsPagination(t *testing.T) {
	many := [][]string{{"apple"}, {"banana"}, {"avocado"}, {"cherry"}, {"apricot"}}
	tbl := New([]Column{{Title: "Name", Width: 20}}, many, WithPageSize(2), WithFilter(0))

	// Move to page 2.
	tbl.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.Equal(t, 2, tbl.State().Pagination.Page)

	// Apply a filter -> pagination resets to page 1.
	tbl.Update(keyRunes("/"))
	tbl.filterInput.SetValue("ap")
	tbl.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Equal(t, 1, tbl.State().Pagination.Page)
	assert.Equal(t, "ap", tbl.State().Filter)
	// Only rows whose first column contains "ap" remain.
	for _, r := range tbl.pageRows {
		assert.True(t, strings.Contains(r[0], "ap"))
	}
}

func TestStateRoundTrip(t *testing.T) {
	tbl := New(cols(), rows())
	tbl.Update(keyRunes("s")) // ascending

	saved := tbl.State()

	restored := New(cols(), rows())
	restored.SetState(saved)

	assert.Equal(t, saved, restored.State())
	assert.Equal(t, []string{"apple", "banana", "cherry"}, firstCol(restored))
}
