package ksql

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/core"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDS embeds the mock datasource and instruments ExecuteKsql so tests can
// count calls, inspect the outgoing statement/properties, and feed a channel.
type fakeDS struct {
	*mock.KafkaDataSourceMock
	execCalls int
	lastSQL   string
	lastProps map[string]string
	ch        chan api.KsqlResultTable
	execErr   error
}

func (f *fakeDS) ExecuteKsql(_ context.Context, sql string, props map[string]string) (<-chan api.KsqlResultTable, error) {
	f.execCalls++
	f.lastSQL = sql
	f.lastProps = props
	if f.execErr != nil {
		return nil, f.execErr
	}
	return f.ch, nil
}

func newQuery(t *testing.T, ds api.KafkaDataSource) *QueryModel {
	t.Helper()
	m := NewQueryModelWithCommon(core.NewCommon(ds)).(*QueryModel)
	m.SetDimensions(160, 40)
	return m
}

// pump runs a cmd chain, feeding each resulting message back into handle, until
// the chain ends or maxSteps is reached.
func pump(m *QueryModel, cmd tea.Cmd, maxSteps int) {
	for i := 0; i < maxSteps && cmd != nil; i++ {
		msg := cmd()
		if msg == nil {
			return
		}
		// tea.Batch produces a BatchMsg of cmds; run them all.
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, c := range batch {
				m.handle(c())
			}
			return
		}
		cmd = m.handle(msg)
	}
}

func TestQueryEmptySubmitIsValidationError(t *testing.T) {
	ds := &fakeDS{KafkaDataSourceMock: &mock.KafkaDataSourceMock{}}
	m := newQuery(t, ds)
	m.editor.SetValue("   ")

	cmd := m.execute()
	require.NotNil(t, cmd)
	msg := cmd()
	note, ok := msg.(core.NotificationMsg)
	require.True(t, ok, "empty submit emits a status notification")
	assert.Equal(t, core.StatusError, note.Severity)
	assert.Equal(t, 0, ds.execCalls, "no datasource call for empty input")
	assert.False(t, m.running)
}

func TestQueryExecuteCallsExecuteKsqlAndRendersRows(t *testing.T) {
	ch := make(chan api.KsqlResultTable, 4)
	ch <- api.KsqlResultTable{Title: "Schema", Columns: []string{"A", "B"}}
	ch <- api.KsqlResultTable{Columns: []string{"A", "B"}, Rows: [][]string{{"1", "x"}}}
	ch <- api.KsqlResultTable{Columns: []string{"A", "B"}, Rows: [][]string{{"2", "y"}}}
	close(ch)
	ds := &fakeDS{KafkaDataSourceMock: &mock.KafkaDataSourceMock{}, ch: ch}

	m := newQuery(t, ds)
	m.editor.SetValue("SELECT * FROM S EMIT CHANGES;")

	pump(m, m.execute(), 20)

	assert.Equal(t, 1, ds.execCalls)
	assert.Equal(t, "SELECT * FROM S EMIT CHANGES;", ds.lastSQL)
	assert.Equal(t, []string{"A", "B"}, m.resCols)
	assert.Len(t, m.resRows, 2, "both streamed rows accumulated")
	assert.False(t, m.running, "channel close clears running state")

	out := m.renderContent(160, 40)
	assert.Contains(t, out, "1")
	assert.Contains(t, out, "y")
}

func TestQueryStreamingAppendsThenNewSchemaReplaces(t *testing.T) {
	m := newQuery(t, &mock.KafkaDataSourceMock{})
	// First schema + rows.
	m.applyTable(api.KsqlResultTable{Columns: []string{"A"}})
	m.applyTable(api.KsqlResultTable{Columns: []string{"A"}, Rows: [][]string{{"1"}}})
	m.applyTable(api.KsqlResultTable{Columns: []string{"A"}, Rows: [][]string{{"2"}}})
	assert.Len(t, m.resRows, 2)

	// A table with new columns replaces the display.
	m.applyTable(api.KsqlResultTable{Columns: []string{"X", "Y"}, Rows: [][]string{{"a", "b"}}})
	assert.Equal(t, []string{"X", "Y"}, m.resCols)
	assert.Len(t, m.resRows, 1)
}

func TestQueryAbortCancelsAndClearsRunning(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	m := newQuery(t, ds)
	m.editor.SetValue("SELECT * FROM S EMIT CHANGES;")

	// Start against the real ticking mock stream.
	msg := m.execute()()
	started, ok := msg.(queryStartedMsg)
	require.True(t, ok)
	require.True(t, m.running)
	m.handle(started) // store ch + cancel

	// Abort → context cancelled; the mock stream stops and closes its channel.
	m.abort()
	assert.True(t, m.aborted)

	// Drain the (now-cancelling) channel to its close, then feed the close.
	drained := false
	deadline := time.After(2 * time.Second)
	for !drained {
		select {
		case _, open := <-started.ch:
			if !open {
				drained = true
			}
		case <-deadline:
			t.Fatal("mock stream did not stop after ctx cancel")
		}
	}
	cmd := m.handle(ksqlResultMsg{ok: false})
	assert.False(t, m.running, "running cleared on channel close")
	require.NotNil(t, cmd) // "consumption cancelled" notification (+ focus)
}

func TestQueryLeavingPageCancels(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	m := newQuery(t, ds)
	m.editor.SetValue("SELECT * FROM S EMIT CHANGES;")
	m.handle(m.execute()())
	require.True(t, m.running)

	cmd := m.OnBlur()
	assert.False(t, m.running, "OnBlur cancels the in-flight query")
	require.NotNil(t, cmd, "leaving mid-stream surfaces a cancelled notice")
}

func TestQueryServerErrorShowsErrorPanel(t *testing.T) {
	m := newQuery(t, &mock.KafkaDataSourceMock{})
	m.running = true
	cmd := m.handleResult(ksqlResultMsg{ok: true, table: api.KsqlResultTable{
		Title: "ksqlDB error", Columns: []string{"Error"},
		Rows: [][]string{{"line 1:8: syntax error"}}, IsError: true,
	}})
	assert.Contains(t, m.errPanel, "syntax error")
	assert.False(t, m.running, "error terminates the stream")
	out := m.renderContent(160, 40)
	assert.Contains(t, out, "Error:")
	assert.Contains(t, out, "syntax error")
	require.NotNil(t, cmd) // error notification + finish
}

func TestQueryEmptyColumnsPlaceholder(t *testing.T) {
	m := newQuery(t, &mock.KafkaDataSourceMock{})
	m.applyTable(api.KsqlResultTable{})
	out := m.renderContent(160, 40)
	assert.Contains(t, out, "no results")
}

func TestQueryClearResults(t *testing.T) {
	m := newQuery(t, &mock.KafkaDataSourceMock{})
	m.applyTable(api.KsqlResultTable{Columns: []string{"A"}, Rows: [][]string{{"1"}}})
	require.True(t, m.hasResult)

	// Disabled while running.
	m.running = true
	assert.Nil(t, m.clearResults())
	m.running = false

	cmd := m.clearResults()
	require.NotNil(t, cmd)
	assert.False(t, m.hasResult)
	assert.Nil(t, m.resRows)
	assert.Equal(t, 0, m.focusIdx, "focus returns to the editor")
}

func TestQueryPropertiesEditor(t *testing.T) {
	m := newQuery(t, &mock.KafkaDataSourceMock{})

	// No rows → no properties map.
	assert.Nil(t, m.buildProps())

	// Add a row, then adding another is refused while its key is empty.
	m.addProp()
	require.Len(t, m.props, 1)
	m.addProp()
	assert.Len(t, m.props, 1, "add refused while a key is empty")

	// Fill the key/value and add a second row.
	m.props[0].keyIn.SetValue("auto.offset.reset")
	m.props[0].valIn.SetValue("earliest")
	m.addProp()
	require.Len(t, m.props, 2)

	// A row with an empty key is dropped from the submitted map.
	props := m.buildProps()
	require.NotNil(t, props)
	assert.Equal(t, map[string]string{"auto.offset.reset": "earliest"}, props)

	// Deleting the only remaining row resets it to empty instead of removing.
	m.props = m.props[:1]
	m.focusIdx = 1
	m.delProp()
	require.Len(t, m.props, 1)
	assert.Equal(t, "", m.props[0].keyIn.Value())
}

func TestQueryEditsRejectedWhileRunning(t *testing.T) {
	m := newQuery(t, &mock.KafkaDataSourceMock{})
	m.running = true
	m.editor.SetValue("original")
	// Forwarding a keystroke while running must not reach the editor.
	m.forwardToFocused(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	assert.Equal(t, "original", m.editor.Value())
	// A second execute while running is a no-op.
	assert.Nil(t, m.execute())
}

func TestQueryNonSelectSuccessNotification(t *testing.T) {
	m := newQuery(t, &mock.KafkaDataSourceMock{})
	m.wasSelect = false
	cmd := m.applyTable(api.KsqlResultTable{Title: "Success", Columns: []string{"Result"}, Rows: [][]string{{"ok"}}})
	require.NotNil(t, cmd)
	note, ok := cmd().(core.NotificationMsg)
	require.True(t, ok)
	assert.Equal(t, core.StatusSuccess, note.Severity)
}

func TestPrettyJSONCell(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello", "hello"},
		{"number", "42", "42"},
		{"object", `{"a":1}`, "{\n  \"a\": 1\n}"},
		{"array", `[1,2]`, "[\n  1,\n  2\n]"},
		{"invalid", "{not json", "{not json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, prettyJSONCell(tt.in))
		})
	}
}

// TestResultTableFitsContentWidth guards against bug #9 regressing: a result
// set with many columns must not render wider than the content pane. Before
// the fix, rebuildResults() floored each column to a minimum of 8 chars with
// no cap on the total, so e.g. 20 columns at a narrow width produced a row
// wider than the pane — the terminal then hard-wraps it, breaking the
// FrameTable border into a ragged, unreadable mess.
func TestResultTableFitsContentWidth(t *testing.T) {
	m := newQuery(t, &mock.KafkaDataSourceMock{})
	const width = 100
	m.SetDimensions(width, 40)
	// Mirror renderContent's pre-render step (-2 for the FrameTable border)
	// exactly, since this test calls renderResults() directly to avoid
	// asserting on the (unrelated, already wide) static footer/help line.
	m.resTable.SetWidth(width - 2)

	cols := make([]string, 20)
	row := make([]string, 20)
	for i := range cols {
		cols[i] = fmt.Sprintf("COLUMN_%d", i)
		row[i] = fmt.Sprintf("value_%d", i)
	}
	m.applyTable(api.KsqlResultTable{Columns: cols, Rows: [][]string{row}})

	out := m.renderResults()
	for _, line := range strings.Split(out, "\n") {
		assert.LessOrEqual(t, lipgloss.Width(line), width,
			"rendered line exceeds the %d-wide content pane: %q", width, line)
	}
}

func TestQueryInputModeGate(t *testing.T) {
	m := newQuery(t, &mock.KafkaDataSourceMock{})
	assert.True(t, m.IsInputMode(), "editing → input mode so SQL letters type through")
	m.running = true
	assert.False(t, m.IsInputMode(), "running → global hotkeys active")
}
