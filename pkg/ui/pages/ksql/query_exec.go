package ksql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// maxResultRows caps retained streamed rows; older rows are dropped and the
// footer notes the truncation.
const maxResultRows = 10000

// listenWindow bounds how long a single drain waits before re-arming, keeping
// the Update loop responsive while a SELECT streams.
const listenWindow = 200 * time.Millisecond

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
	// Hold the cancel func now rather than when queryStartedMsg arrives, so an
	// abort or a page leave while the stream is still opening cancels it.
	m.cancel = cancel
	m.gen++
	gen := m.gen
	return func() tea.Msg {
		ch, err := ds.ExecuteKsql(ctx, sql, props)
		if err != nil {
			cancel()
			return ksqlResultMsg{gen: gen, ok: true, table: api.KsqlResultTable{
				Title: "Error", Columns: []string{"Error"},
				Rows: [][]string{{err.Error()}}, IsError: true,
			}}
		}
		return queryStartedMsg{gen: gen, ch: ch, cancel: cancel}
	}
}

// listenForResults drains one table from the channel, re-arming on timeout.
func listenForResults(ch <-chan api.KsqlResultTable, gen int) tea.Cmd {
	return func() tea.Msg {
		select {
		case t, ok := <-ch:
			if !ok {
				return ksqlResultMsg{gen: gen, ok: false}
			}
			return ksqlResultMsg{gen: gen, table: t, ok: true}
		case <-time.After(listenWindow):
			return queryTickMsg{gen: gen}
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
			return tea.Batch(notify, listenForResults(m.ch, m.gen))
		}
		return listenForResults(m.ch, m.gen)
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
