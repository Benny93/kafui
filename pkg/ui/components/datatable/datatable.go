// Package datatable provides a reusable table component wrapping
// github.com/charmbracelet/bubbles/table. It adds column sorting, pagination,
// a configurable empty state, optional multi-row selection and an optional
// per-column filter prompt. All view state (sort, page, filter) lives on the
// component and can be persisted/restored via State/SetState so a hosting page
// can keep it across navigation.
package datatable

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DefaultPageSize is the number of rows shown per page.
const DefaultPageSize = 25

// SortDir is the sort direction applied to a column.
type SortDir int

const (
	// SortNone means the table is unsorted.
	SortNone SortDir = iota
	// SortAsc sorts ascending (indicator ▲).
	SortAsc
	// SortDesc sorts descending (indicator ▼).
	SortDesc
)

// Column defines a single table column.
type Column struct {
	Title    string
	Width    int
	Sortable bool
}

// Pagination holds paging state.
type Pagination struct {
	Page     int // 1-based
	PageSize int
}

// TableState is the persistable view state of a Table.
type TableState struct {
	SortCol    int // sorted column index, -1 when unsorted
	SortDir    SortDir
	Pagination Pagination
	Filter     string
	FilterCol  int
}

// Key bindings handled by the component.
var (
	keySort     = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort"))
	keySelect   = key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select"))
	keyFilter   = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter"))
	keyPageUp   = key.NewBinding(key.WithKeys("pgup", "left"), key.WithHelp("←/pgup", "prev page"))
	keyPageDown = key.NewBinding(key.WithKeys("pgdown", "right"), key.WithHelp("→/pgdown", "next page"))
)

// Table is the reusable table component.
type Table struct {
	width  int
	height int

	cols    []Column
	allRows [][]string // full, unfiltered dataset

	inner       table.Model
	filterInput textinput.Model

	// view state
	sortCol   int
	sortDir   SortDir
	page      int
	pageSize  int
	filter    string
	filterCol int
	filtering bool // filter prompt open

	pageRows  [][]string   // raw rows currently displayed on this page
	selection map[int]bool // selected page-relative indices

	// config
	selectable   bool
	filterable   bool
	emptyMessage string
	batchHint    string

	styles *styles.Styles
}

// Option configures a Table.
type Option func(*Table)

// WithSelection enables multi-row selection. hint is shown in the
// batch-actions line when at least one row is selected.
func WithSelection(hint string) Option {
	return func(t *Table) { t.selectable = true; t.batchHint = hint }
}

// WithFilter enables the per-column filter prompt on the given column index.
func WithFilter(col int) Option {
	return func(t *Table) { t.filterable = true; t.filterCol = col }
}

// WithEmptyMessage sets the message shown when there are no rows.
func WithEmptyMessage(msg string) Option {
	return func(t *Table) { t.emptyMessage = msg }
}

// WithPageSize overrides the default page size.
func WithPageSize(n int) Option {
	return func(t *Table) {
		if n > 0 {
			t.pageSize = n
		}
	}
}

// New creates a Table from column definitions and rows.
func New(cols []Column, rows [][]string, opts ...Option) *Table {
	st := styles.DefaultStyles()
	ti := textinput.New()
	ti.Placeholder = "filter…"

	t := &Table{
		cols:         cols,
		allRows:      rows,
		filterInput:  ti,
		sortCol:      -1,
		sortDir:      SortNone,
		page:         1,
		pageSize:     DefaultPageSize,
		filterCol:    0,
		selection:    map[int]bool{},
		emptyMessage: "No rows",
		styles:       st,
	}
	for _, o := range opts {
		o(t)
	}

	t.inner = table.New(
		table.WithColumns(t.tableColumns()),
		table.WithFocused(true),
	)
	t.inner.SetStyles(table.Styles{
		Header:   st.TableStyle.Header,
		Cell:     st.TableStyle.Row,
		Selected: st.TableStyle.Selected,
	})
	t.refresh()
	return t
}

// SetRows replaces the full dataset. Selection is cleared and the page clamped.
func (t *Table) SetRows(rows [][]string) {
	t.allRows = rows
	t.clearSelection()
	t.refresh()
}

// SetDimensions sets the component size.
func (t *Table) SetDimensions(width, height int) {
	t.width = width
	t.height = height
	t.inner.SetWidth(width)
	h := height - 2 // reserve footer + hint lines
	if h < 1 {
		h = 1
	}
	t.inner.SetHeight(h)
}

// Update handles a message and returns any resulting command.
func (t *Table) Update(msg tea.Msg) tea.Cmd {
	// While the filter prompt is open, keystrokes go to the input.
	if t.filtering {
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.Type {
			case tea.KeyEnter:
				t.applyFilter()
				return nil
			case tea.KeyEsc:
				t.filtering = false
				t.filterInput.Blur()
				return nil
			}
		}
		var cmd tea.Cmd
		t.filterInput, cmd = t.filterInput.Update(msg)
		return cmd
	}

	if km, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(km, keySort):
			t.cycleSort()
			return nil
		case t.filterable && key.Matches(km, keyFilter):
			t.filtering = true
			return t.filterInput.Focus()
		case key.Matches(km, keyPageDown):
			t.nextPage()
			return nil
		case key.Matches(km, keyPageUp):
			t.prevPage()
			return nil
		case t.selectable && key.Matches(km, keySelect):
			t.toggleSelection()
			return nil
		}
	}

	var cmd tea.Cmd
	t.inner, cmd = t.inner.Update(msg)
	return cmd
}

// View renders the table.
func (t *Table) View() string {
	if len(t.filteredRows()) == 0 {
		return lipgloss.NewStyle().Foreground(styles.FgMuted).Italic(true).Render(t.emptyMessage)
	}

	var b strings.Builder
	b.WriteString(t.inner.View())
	b.WriteString("\n")
	b.WriteString(t.footer())
	if t.selectable && len(t.selection) > 0 {
		b.WriteString("\n")
		b.WriteString(t.batchHintLine())
	}
	if t.filtering {
		b.WriteString("\n")
		b.WriteString(t.filterInput.View())
	}
	return b.String()
}

// SelectedRows returns the raw rows currently selected on this page.
func (t *Table) SelectedRows() [][]string {
	var out [][]string
	for i, r := range t.pageRows {
		if t.selection[i] {
			out = append(out, r)
		}
	}
	return out
}

// State returns the current persistable view state.
func (t *Table) State() TableState {
	return TableState{
		SortCol:    t.sortCol,
		SortDir:    t.sortDir,
		Pagination: Pagination{Page: t.page, PageSize: t.pageSize},
		Filter:     t.filter,
		FilterCol:  t.filterCol,
	}
}

// SetState restores previously captured view state.
func (t *Table) SetState(s TableState) {
	t.sortCol = s.SortCol
	t.sortDir = s.SortDir
	if s.Pagination.PageSize > 0 {
		t.pageSize = s.Pagination.PageSize
	}
	if s.Pagination.Page > 0 {
		t.page = s.Pagination.Page
	}
	t.filter = s.Filter
	t.filterCol = s.FilterCol
	t.filterInput.SetValue(s.Filter)
	t.clearSelection()
	t.refresh()
}

// --- internals ---

func (t *Table) applyFilter() {
	t.filter = t.filterInput.Value()
	t.filtering = false
	t.filterInput.Blur()
	t.page = 1 // filtering resets pagination
	t.clearSelection()
	t.refresh()
}

func (t *Table) cycleSort() {
	col := t.firstSortable()
	if col < 0 {
		return
	}
	switch {
	case t.sortCol != col || t.sortDir == SortNone:
		t.sortCol, t.sortDir = col, SortAsc
	case t.sortDir == SortAsc:
		t.sortDir = SortDesc
	default:
		t.sortCol, t.sortDir = -1, SortNone
	}
	t.refresh()
}

func (t *Table) nextPage() {
	if t.page < t.pageCount(len(t.filteredRows())) {
		t.page++
		t.clearSelection() // selection is per page
		t.refresh()
	}
}

func (t *Table) prevPage() {
	if t.page > 1 {
		t.page--
		t.clearSelection()
		t.refresh()
	}
}

func (t *Table) toggleSelection() {
	i := t.inner.Cursor()
	if i < 0 || i >= len(t.pageRows) {
		return
	}
	if t.selection[i] {
		delete(t.selection, i)
	} else {
		t.selection[i] = true
	}
	t.refresh()
	t.inner.SetCursor(i)
}

func (t *Table) clearSelection() { t.selection = map[int]bool{} }

func (t *Table) firstSortable() int {
	for i, c := range t.cols {
		if c.Sortable {
			return i
		}
	}
	return -1
}

// filteredRows returns a fresh slice of the rows matching the current filter.
func (t *Table) filteredRows() [][]string {
	if t.filter == "" {
		rows := make([][]string, len(t.allRows))
		copy(rows, t.allRows)
		return rows
	}
	q := strings.ToLower(t.filter)
	var out [][]string
	for _, r := range t.allRows {
		if strings.Contains(strings.ToLower(cell(r, t.filterCol)), q) {
			out = append(out, r)
		}
	}
	return out
}

func (t *Table) sortRows(rows [][]string) {
	if t.sortCol < 0 || t.sortDir == SortNone {
		return
	}
	col := t.sortCol
	sort.SliceStable(rows, func(i, j int) bool {
		vi, vj := strings.ToLower(cell(rows[i], col)), strings.ToLower(cell(rows[j], col))
		if t.sortDir == SortDesc {
			return vi > vj
		}
		return vi < vj
	})
}

func (t *Table) pageCount(n int) int {
	if t.pageSize <= 0 {
		return 1
	}
	c := (n + t.pageSize - 1) / t.pageSize
	if c < 1 {
		return 1
	}
	return c
}

// refresh recomputes the visible page and feeds it to the inner table.
func (t *Table) refresh() {
	rows := t.filteredRows()
	t.sortRows(rows)

	pc := t.pageCount(len(rows))
	if t.page > pc {
		t.page = pc
	}
	if t.page < 1 {
		t.page = 1
	}

	start := (t.page - 1) * t.pageSize
	end := start + t.pageSize
	if start > len(rows) {
		start = len(rows)
	}
	if end > len(rows) {
		end = len(rows)
	}
	t.pageRows = rows[start:end]

	trows := make([]table.Row, len(t.pageRows))
	for i, r := range t.pageRows {
		trows[i] = t.decorate(i, r)
	}
	t.inner.SetColumns(t.tableColumns())
	t.inner.SetRows(trows)
}

// decorate builds a display row, prefixing a selection marker when selectable.
func (t *Table) decorate(i int, r []string) table.Row {
	row := make(table.Row, len(t.cols))
	for c := range t.cols {
		row[c] = cell(r, c)
	}
	if t.selectable && len(row) > 0 {
		mark := "  "
		if t.selection[i] {
			mark = "✓ "
		}
		row[0] = mark + row[0]
	}
	return row
}

// tableColumns builds inner columns with a sort indicator on the sorted column.
func (t *Table) tableColumns() []table.Column {
	cols := make([]table.Column, len(t.cols))
	for i, c := range t.cols {
		title := c.Title
		if i == t.sortCol {
			switch t.sortDir {
			case SortAsc:
				title += " ▲"
			case SortDesc:
				title += " ▼"
			}
		}
		cols[i] = table.Column{Title: title, Width: c.Width}
	}
	return cols
}

func (t *Table) footer() string {
	s := fmt.Sprintf("Page %d/%d", t.page, t.pageCount(len(t.filteredRows())))
	return lipgloss.NewStyle().Foreground(styles.FgMuted).Render(s)
}

func (t *Table) batchHintLine() string {
	text := fmt.Sprintf("%d selected", len(t.selection))
	if t.batchHint != "" {
		text += " · " + t.batchHint
	}
	return lipgloss.NewStyle().Foreground(styles.Accent).Render(text)
}

func cell(r []string, i int) string {
	if i >= 0 && i < len(r) {
		return r[i]
	}
	return ""
}
