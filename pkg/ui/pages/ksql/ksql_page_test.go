package ksql

import (
	"context"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/cluster"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/core"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newOverview(t *testing.T) *Model {
	t.Helper()
	m := NewModelWithCommon(core.NewCommon(&mock.KafkaDataSourceMock{})).(*Model)
	m.SetDimensions(160, 40)
	return m
}

// runCmd executes a tea.Cmd and returns its message (nil-safe).
func runCmd(c tea.Cmd) tea.Msg {
	if c == nil {
		return nil
	}
	return c()
}

func TestOverviewRendersStreamsAndTables(t *testing.T) {
	m := newOverview(t)
	// Drive the parallel listing loads through the real mock datasource.
	m.handle(runCmd(m.loadStreams()))
	m.handle(runCmd(m.loadTables()))

	// Tables tab is the default.
	assert.Equal(t, tabTables, m.active)
	out := m.renderContent(160, 40)
	assert.Contains(t, out, "Streams:")
	assert.Contains(t, out, "Tables:")
	// A table from the mock listing.
	assert.Contains(t, out, "USERS")

	// Switch to the Streams tab and confirm stream rows show.
	m.handleKey(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, tabStreams, m.active)
	out = m.renderContent(160, 40)
	assert.Contains(t, out, "PAGEVIEWS")
}

func TestOverviewNotConfiguredState(t *testing.T) {
	m := newOverview(t)
	m.handle(streamsLoadedMsg{err: api.KsqlNotConfiguredError{}})
	m.handle(tablesLoadedMsg{err: api.KsqlNotConfiguredError{}})
	out := m.renderContent(160, 40)
	assert.Contains(t, out, "not configured")
}

func TestOverviewErrorStateAndRetry(t *testing.T) {
	m := newOverview(t)
	m.handle(streamsLoadedMsg{streams: []api.KsqlStream{{Name: "S1"}}})
	m.handle(tablesLoadedMsg{err: api.KsqlServerError{StatusCode: 500, Message: "boom"}})

	// Default tab is Tables, which errored → error state with retry hint.
	out := m.renderContent(160, 40)
	assert.Contains(t, out, "Error:")
	assert.Contains(t, out, "retry")

	// Retry refetches BOTH listings.
	cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	require.NotNil(t, cmd)
	assert.False(t, m.streamsLoaded)
	assert.False(t, m.tablesLoaded)
}

func TestOverviewSortToggling(t *testing.T) {
	m := newOverview(t)
	m.handle(streamsLoadedMsg{streams: []api.KsqlStream{
		{Name: "b_stream", Topic: "t2"},
		{Name: "a_stream", Topic: "t1"},
	}})
	m.handle(tablesLoadedMsg{tables: []api.KsqlTable{{Name: "X"}}})
	m.handleKey(tea.KeyMsg{Type: tea.KeyTab}) // to Streams tab

	// Default sort: name ascending.
	first := m.sortedStreams()[0].Name
	assert.Equal(t, "a_stream", first)

	// Advance the sort column (name→topic) and confirm ordering changes.
	m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	assert.Equal(t, 1, m.sortCol)
	assert.Equal(t, "a_stream", m.sortedStreams()[0].Name) // t1 < t2

	// Cycle through all columns to wrap and flip direction to descending.
	for i := 0; i < len(streamColumns())-1; i++ {
		m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	}
	assert.Equal(t, 0, m.sortCol)
	assert.False(t, m.sortAsc)
	assert.Equal(t, "b_stream", m.sortedStreams()[0].Name) // desc by name
}

func TestOverviewSeedsQueryFromRow(t *testing.T) {
	m := newOverview(t)
	m.handle(tablesLoadedMsg{tables: []api.KsqlTable{{Name: "USERS"}}})
	m.handle(streamsLoadedMsg{streams: nil})

	msg := runCmd(m.handleKey(tea.KeyMsg{Type: tea.KeyEnter}))
	pc, ok := msg.(core.PageChangeMsg)
	require.True(t, ok, "enter on a row emits a PageChangeMsg")
	assert.Equal(t, "ksql_query", pc.PageID)
	data, _ := pc.Data.(map[string]interface{})
	assert.Equal(t, "SELECT * FROM USERS EMIT CHANGES;", data["name"])
}

func TestOverviewOpensQueryEditor(t *testing.T) {
	m := newOverview(t)
	msg := runCmd(m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}))
	pc, ok := msg.(core.PageChangeMsg)
	require.True(t, ok)
	assert.Equal(t, "ksql_query", pc.PageID)
}

func TestPageIdentity(t *testing.T) {
	m := newOverview(t)
	assert.Equal(t, "ksql", m.GetID())
	assert.Equal(t, "ksqlDB", m.GetTitle())
}

// TestCapabilityGating asserts the predicate the shell uses to show/hide the
// ksql entry point: true only when the active cluster advertises CapKsqlDB.
func TestCapabilityGating(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	common := core.NewCommon(ds)
	col := cluster.New(ds, 0, nil)
	col.CollectAll(context.Background())
	common.Collector = col

	require.NoError(t, ds.SetContext("kafka-dev"))
	assert.True(t, common.HasCapability(api.CapKsqlDB), "kafka-dev advertises ksqlDB")

	require.NoError(t, ds.SetContext("kafka-prod"))
	assert.False(t, common.HasCapability(api.CapKsqlDB), "kafka-prod does not advertise ksqlDB")
}
