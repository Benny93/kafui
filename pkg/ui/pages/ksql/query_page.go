package ksql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components/editor"
	"github.com/Benny93/kafui/pkg/ui/core"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// maxResultRows caps retained streamed rows; older rows are dropped and the
// footer notes the truncation.
const maxResultRows = 10000

// listenWindow bounds how long a single drain waits before re-arming, keeping
// the Update loop responsive while a SELECT streams.
const listenWindow = 200 * time.Millisecond

// propRow is one streams-property key/value pair in the properties editor.
type propRow struct {
	keyIn textinput.Model
	valIn textinput.Model
}

func newPropRow() propRow {
	k := textinput.New()
	k.Placeholder = "property"
	v := textinput.New()
	v.Placeholder = "value"
	return propRow{keyIn: k, valIn: v}
}

// QueryModel is the ksqlDB query editor page.
type QueryModel struct {
	common      *core.Common
	keys        queryKeys
	reusableApp *templateui.ReusableApp
	dims        core.Dimensions

	editor *editor.Editor
	props  []propRow

	// focusIdx: 0 = editor; 1..2N = property inputs (row r field f → 1+2r+f).
	focusIdx int

	// streaming state
	running   bool
	aborted   bool
	ch        <-chan api.KsqlResultTable
	cancel    context.CancelFunc
	wasSelect bool

	// result display
	resTable    table.Model
	resCols     []string
	resRows     [][]string
	resTitle    string
	hasResult   bool
	placeholder bool
	truncated   bool
	errPanel    string
}

// NewQueryModelWithCommon builds the ksqlDB query editor page. The intended
// router page ID is "ksql_query".
func NewQueryModelWithCommon(common *core.Common) core.Page {
	return newQueryModel(common, "")
}

// NewQueryModelWithSeed builds the query page with the editor pre-seeded (used
// when opened from a stream/table row on the overview page). The router passes
// the seed via the navigation "name" field.
func NewQueryModelWithSeed(common *core.Common, seed string) core.Page {
	return newQueryModel(common, seed)
}

func newQueryModel(common *core.Common, seed string) *QueryModel {
	m := &QueryModel{
		common: common,
		keys:   defaultQueryKeys(),
		editor: editor.NewEditor(seed),
	}
	m.resTable = table.New(table.WithFocused(false), table.WithHeight(10))

	config := &providers.AppConfig{
		ContentProvider:      &queryContentProvider{model: m},
		ShowSidebarByDefault: false,
	}
	m.reusableApp = templateui.NewReusableApp(config)
	m.reusableApp.SetKeyMap(queryHelpKeyMap{keys: m.keys})
	return m
}

// --- core.Page ---

func (m *QueryModel) Init() tea.Cmd { return m.reusableApp.Init() }

func (m *QueryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.reusableApp.Update(msg)
	if app, ok := updated.(*templateui.ReusableApp); ok {
		m.reusableApp = app
	}
	return m, cmd
}

func (m *QueryModel) View() string { return m.reusableApp.View() }

func (m *QueryModel) SetDimensions(width, height int) {
	m.dims = core.Dimensions{Width: width, Height: height}
	edH := 6
	m.editor.SetDimensions(width, edH)
	m.resTable.SetWidth(width)
	body := height - 22
	if body < 3 {
		body = 3
	}
	m.resTable.SetHeight(body)
	m.reusableApp.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

func (m *QueryModel) GetID() string    { return "ksql_query" }
func (m *QueryModel) GetTitle() string { return "ksqlDB Query" }

func (m *QueryModel) GetHelp() []key.Binding {
	return []key.Binding{m.keys.Execute, m.keys.Clear, m.keys.ClearRes, m.keys.AddProp, m.keys.DelProp, m.keys.FocusNext, m.keys.Back}
}

func (m *QueryModel) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) { return m, nil }

func (m *QueryModel) OnFocus() tea.Cmd { return m.editor.Focus() }

// OnBlur cancels any in-flight query when navigating away (KS-15). Leaving the
// page mid-stream surfaces the same "cancelled" notice as an explicit abort.
func (m *QueryModel) OnBlur() tea.Cmd {
	if m.running {
		m.stopQuery()
		return core.NewNotification(core.StatusInfo, "ksqlDB", "consumption cancelled")
	}
	m.stopQuery()
	return nil
}

// IsInputMode reports whether the shell should let keystrokes reach the editor
// unmodified. True while editing (not running) so SQL text (including 'q') is
// typed rather than triggering global hotkeys.
func (m *QueryModel) IsInputMode() bool { return !m.running }

// --- message handling ---

func (m *QueryModel) handle(msg tea.Msg) tea.Cmd {
	switch v := msg.(type) {
	case queryStartedMsg:
		m.ch = v.ch
		m.cancel = v.cancel
		return listenForResults(v.ch)
	case queryTickMsg:
		if m.running && m.ch != nil {
			return listenForResults(m.ch)
		}
		return nil
	case ksqlResultMsg:
		return m.handleResult(v)
	case tea.KeyMsg:
		return m.handleKey(v)
	}
	return m.forwardToFocused(msg)
}

func (m *QueryModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Execute):
		return m.execute()
	case key.Matches(msg, m.keys.Clear):
		if !m.running {
			m.editor.SetValue("")
		}
		return nil
	case key.Matches(msg, m.keys.ClearRes):
		return m.clearResults()
	case key.Matches(msg, m.keys.AddProp):
		m.addProp()
		return nil
	case key.Matches(msg, m.keys.DelProp):
		m.delProp()
		return nil
	case key.Matches(msg, m.keys.FocusNext):
		m.advanceFocus()
		return nil
	}
	return m.forwardToFocused(msg)
}

// forwardToFocused routes the key to the currently focused input.
func (m *QueryModel) forwardToFocused(msg tea.Msg) tea.Cmd {
	if m.running {
		return nil
	}
	if m.focusIdx == 0 {
		_, cmd := m.editor.Update(msg)
		return cmd
	}
	idx := m.focusIdx - 1
	row := idx / 2
	if row >= len(m.props) {
		return nil
	}
	var cmd tea.Cmd
	if idx%2 == 0 {
		m.props[row].keyIn, cmd = m.props[row].keyIn.Update(msg)
	} else {
		m.props[row].valIn, cmd = m.props[row].valIn.Update(msg)
	}
	return cmd
}

// advanceFocus cycles editor → each property input → back to editor.
func (m *QueryModel) advanceFocus() {
	total := 1 + 2*len(m.props)
	m.focusIdx = (m.focusIdx + 1) % total
	m.syncFocus()
}

func (m *QueryModel) syncFocus() {
	if m.focusIdx == 0 {
		m.editor.Focus()
	} else {
		m.editor.Blur()
	}
	for i := range m.props {
		m.props[i].keyIn.Blur()
		m.props[i].valIn.Blur()
	}
	if m.focusIdx > 0 {
		idx := m.focusIdx - 1
		row := idx / 2
		if row < len(m.props) {
			if idx%2 == 0 {
				m.props[row].keyIn.Focus()
			} else {
				m.props[row].valIn.Focus()
			}
		}
	}
}

// addProp appends a property row, refused while any existing row has an empty
// key (KS-13).
func (m *QueryModel) addProp() {
	for _, r := range m.props {
		if strings.TrimSpace(r.keyIn.Value()) == "" {
			return
		}
	}
	m.props = append(m.props, newPropRow())
	// Focus the new row's key input.
	m.focusIdx = 1 + 2*(len(m.props)-1)
	m.syncFocus()
}

// delProp removes the property row the focus is on. Deleting the only remaining
// row resets it to empty instead of removing it (KS-13).
func (m *QueryModel) delProp() {
	if len(m.props) == 0 || m.focusIdx == 0 {
		return
	}
	row := (m.focusIdx - 1) / 2
	if row >= len(m.props) {
		return
	}
	if len(m.props) == 1 {
		m.props[0] = newPropRow()
		m.focusIdx = 1
		m.syncFocus()
		return
	}
	m.props = append(m.props[:row], m.props[row+1:]...)
	if m.focusIdx > 2*len(m.props) {
		m.focusIdx = 0
	}
	m.syncFocus()
}

// buildProps returns the streams-properties map, dropping rows with empty keys.
// Returns nil when no non-empty rows remain (request carries no map at all).
func (m *QueryModel) buildProps() map[string]string {
	out := map[string]string{}
	for _, r := range m.props {
		k := strings.TrimSpace(r.keyIn.Value())
		if k == "" {
			continue
		}
		out[k] = r.valIn.Value()
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// --- execution / streaming (KS-14/KS-15) ---

func (m *QueryModel) execute() tea.Cmd {
	if m.running {
		return nil
	}
	sql := strings.TrimSpace(m.editor.Value())
	if sql == "" {
		// Validation error in the status bar; no datasource call (KS-12).
		return core.NewNotification(core.StatusError, "Invalid statement", "no valid statement was found")
	}
	m.running = true
	m.aborted = false
	m.errPanel = ""
	m.wasSelect = strings.HasPrefix(strings.ToUpper(sql), "SELECT")
	m.editor.Blur()

	ds := m.common.DataSource
	props := m.buildProps()
	ctx, cancel := context.WithCancel(context.Background())
	return func() tea.Msg {
		ch, err := ds.ExecuteKsql(ctx, sql, props)
		if err != nil {
			cancel()
			return ksqlResultMsg{ok: true, table: api.KsqlResultTable{
				Title: "Error", Columns: []string{"Error"},
				Rows: [][]string{{err.Error()}}, IsError: true,
			}}
		}
		return queryStartedMsg{ch: ch, cancel: cancel}
	}
}

// listenForResults drains one table from the channel, re-arming on timeout.
func listenForResults(ch <-chan api.KsqlResultTable) tea.Cmd {
	return func() tea.Msg {
		select {
		case t, ok := <-ch:
			if !ok {
				return ksqlResultMsg{ok: false}
			}
			return ksqlResultMsg{table: t, ok: true}
		case <-time.After(listenWindow):
			return queryTickMsg{}
		}
	}
}

func (m *QueryModel) handleResult(v ksqlResultMsg) tea.Cmd {
	if !v.ok {
		// Channel closed: complete, aborted, or errored.
		return m.finish()
	}
	if v.table.IsError {
		return m.handleErrorTable(v.table)
	}
	notify := m.applyTable(v.table)
	// Keep draining until the channel closes.
	if m.ch != nil {
		if notify != nil {
			return tea.Batch(notify, listenForResults(m.ch))
		}
		return listenForResults(m.ch)
	}
	// No live channel (single closed-channel statement) — finish.
	return tea.Batch(notify, m.finish())
}

func (m *QueryModel) handleErrorTable(t api.KsqlResultTable) tea.Cmd {
	title := t.Title
	if title == "" {
		title = "ksqlDB error"
	}
	msg := title
	if len(t.Rows) > 0 && len(t.Rows[0]) > 0 {
		msg = strings.Join(t.Rows[0], " ")
	}
	m.errPanel = msg
	finish := m.finish()
	return tea.Batch(core.NotifyError(title, fmt.Errorf("%s", msg)), finish)
}

// applyTable folds one non-error result table into the display. A schema table
// (columns, no rows) re-initializes; row tables with matching columns append;
// a table with new columns replaces the display. Returns an optional success
// notification for non-returning statements.
func (m *QueryModel) applyTable(t api.KsqlResultTable) tea.Cmd {
	if len(t.Columns) == 0 && len(t.Rows) == 0 {
		m.placeholder = true
		return nil
	}
	m.placeholder = false
	if len(t.Rows) == 0 {
		// Schema announcement.
		m.resCols = append([]string(nil), t.Columns...)
		m.resRows = nil
		m.resTitle = t.Title
		m.truncated = false
		m.hasResult = true
		m.rebuildResults()
		return nil
	}
	// Rows present.
	if !equalCols(t.Columns, m.resCols) {
		m.resCols = append([]string(nil), t.Columns...)
		m.resRows = nil
		m.resTitle = t.Title
		m.truncated = false
	}
	for _, r := range t.Rows {
		row := make([]string, len(r))
		for i, c := range r {
			row[i] = prettyJSONCell(c)
		}
		m.resRows = append(m.resRows, row)
	}
	if len(m.resRows) > maxResultRows {
		m.resRows = m.resRows[len(m.resRows)-maxResultRows:]
		m.truncated = true
	}
	m.hasResult = true
	m.rebuildResults()

	if !m.wasSelect {
		title := t.Title
		if title == "" {
			title = "Statement executed"
		}
		return core.NewNotification(core.StatusSuccess, "ksqlDB", title)
	}
	return nil
}

func (m *QueryModel) rebuildResults() {
	cols := make([]table.Column, len(m.resCols))
	w := m.dims.Width
	if w <= 0 {
		w = 100
	}
	w -= 2 // leave room for the FrameTable border, matching renderContent's SetWidth
	// bubbles/table pads every cell by 1 char on each side (its default Cell
	// style), so a column's actual on-screen footprint is Width+2 — budget for
	// that instead of an unconditional 8-char floor, which used to push wide
	// result sets (many columns) past the pane width no matter how narrow the
	// pane was, wrapping the FrameTable border into a broken mess (BUG-9).
	const cellPadding = 2
	each := 18
	if n := len(m.resCols); n > 0 {
		each = w/n - cellPadding
		if each < 1 {
			each = 1
		}
	}
	for i, c := range m.resCols {
		cols[i] = table.Column{Title: c, Width: each}
	}
	rows := make([]table.Row, 0, len(m.resRows))
	for _, r := range m.resRows {
		rows = append(rows, table.Row(r))
	}
	// Clear rows before changing columns: bubbles' table re-renders existing
	// rows against the new column set on SetColumns and panics if a row is
	// wider than the (possibly now shorter) column list.
	m.resTable.SetRows(nil)
	m.resTable.SetColumns(cols)
	m.resTable.SetRows(rows)
}

// finish clears running state, cancels the context, and refocuses the editor.
func (m *QueryModel) finish() tea.Cmd {
	wasAborted := m.aborted
	m.stopQuery()
	cmd := m.editor.Focus()
	m.focusIdx = 0
	if wasAborted {
		return tea.Batch(core.NewNotification(core.StatusInfo, "ksqlDB", "consumption cancelled"), cmd)
	}
	return cmd
}

// stopQuery cancels the context and clears streaming state (idempotent).
func (m *QueryModel) stopQuery() {
	if m.cancel != nil {
		m.cancel()
	}
	m.cancel = nil
	m.ch = nil
	m.running = false
}

// abort cancels a running query (KS-15). The channel then closes, driving the
// "cancelled" notification through finish().
func (m *QueryModel) abort() {
	if m.running {
		m.aborted = true
		if m.cancel != nil {
			m.cancel()
		}
	}
}

// clearResults discards the displayed table and refocuses the editor. Enabled
// only when results exist and nothing is running (KS-15).
func (m *QueryModel) clearResults() tea.Cmd {
	if m.running || !m.hasResult {
		return nil
	}
	m.resCols = nil
	m.resRows = nil
	m.resTitle = ""
	m.hasResult = false
	m.placeholder = false
	m.truncated = false
	m.errPanel = ""
	m.rebuildResults()
	m.focusIdx = 0
	return m.editor.Focus()
}

// --- rendering ---

func (m *QueryModel) renderContent(width, height int) string {
	// ponytail: keyword syntax highlighting inside the textarea is deferred —
	// bubbles/textarea renders its own buffer, so lipgloss token styling would
	// require reimplementing its view; the spec marks highlighting best-effort.
	// ponytail: statement history (up/down recall) deferred — not required by
	// KS-12 and cheap to add later on top of the existing editor.
	m.resTable.SetWidth(width - 2) // -2 leaves room for the FrameTable border
	var b strings.Builder
	b.WriteString(m.common.Styles.Header.Render("Statement"))
	if m.running {
		b.WriteString("  " + m.common.Styles.StatusStyle.Info.Render("● streaming… (esc: abort)"))
	}
	b.WriteString("\n")
	b.WriteString(m.editor.View())
	b.WriteString("\n\n")

	b.WriteString(m.renderProps())
	b.WriteString("\n")

	b.WriteString(m.renderResults())
	b.WriteString("\n")
	b.WriteString(m.common.Styles.Muted.Render(
		"ctrl+x: execute • ctrl+l: clear editor • ctrl+n/ctrl+d: add/del property • ctrl+r: clear results • tab: focus • esc: back"))
	return b.String()
}

func (m *QueryModel) renderProps() string {
	var b strings.Builder
	b.WriteString(m.common.Styles.Header.Render("Properties"))
	b.WriteString("\n")
	if len(m.props) == 0 {
		b.WriteString(m.common.Styles.Muted.Render("(none — ctrl+n to add a streams property)"))
		return b.String()
	}
	for _, r := range m.props {
		b.WriteString(r.keyIn.View() + " = " + r.valIn.View() + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *QueryModel) renderResults() string {
	var b strings.Builder
	title := m.resTitle
	if title == "" {
		title = "Results"
	}
	b.WriteString(m.common.Styles.Header.Render(title))
	b.WriteString("\n")

	if m.errPanel != "" {
		b.WriteString(m.common.Styles.Error.Render("Error: " + m.errPanel))
		return b.String()
	}
	if m.placeholder {
		b.WriteString(m.common.Styles.Muted.Render("(no results)"))
		return b.String()
	}
	if !m.hasResult {
		b.WriteString(m.common.Styles.Muted.Render("Run a statement to see results."))
		return b.String()
	}
	b.WriteString(stylesPkg.FrameTable(m.resTable.View()))
	if m.truncated {
		b.WriteString("\n" + m.common.Styles.StatusStyle.Warning.Render(
			fmt.Sprintf("… showing last %d rows (older rows dropped)", maxResultRows)))
	}
	return b.String()
}

// --- helpers ---

func equalCols(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// prettyJSONCell indents a cell whose value is a JSON object or array; other
// values are returned unchanged (KS-14).
func prettyJSONCell(s string) string {
	t := strings.TrimSpace(s)
	if len(t) < 2 || (t[0] != '{' && t[0] != '[') {
		return s
	}
	var v interface{}
	if err := json.Unmarshal([]byte(t), &v); err != nil {
		return s
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return s
	}
	return string(out)
}
