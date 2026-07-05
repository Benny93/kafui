package mock

import (
	"context"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockKsqlListings(t *testing.T) {
	ds := newMock()
	streams, err := ds.ListKsqlStreams()
	require.NoError(t, err)
	assert.NotEmpty(t, streams)
	tables, err := ds.ListKsqlTables()
	require.NoError(t, err)
	assert.NotEmpty(t, tables)
}

func TestMockExecuteKsql_StatementKinds(t *testing.T) {
	ds := newMock()
	ctx := context.Background()

	t.Run("show streams -> table", func(t *testing.T) {
		tables := drain(ds.ExecuteKsql(ctx, "SHOW STREAMS;", nil))
		require.Len(t, tables, 1)
		assert.Equal(t, "Streams", tables[0].Title)
		assert.NotEmpty(t, tables[0].Rows)
	})
	t.Run("ddl -> success", func(t *testing.T) {
		tables := drain(ds.ExecuteKsql(ctx, "CREATE STREAM s AS SELECT * FROM t;", nil))
		require.Len(t, tables, 1)
		assert.False(t, tables[0].IsError)
	})
	t.Run("invalid -> error table", func(t *testing.T) {
		tables := drain(ds.ExecuteKsql(ctx, "PRINT 'x';", nil))
		require.Len(t, tables, 1)
		assert.True(t, tables[0].IsError)
	})
	t.Run("empty -> error table", func(t *testing.T) {
		tables := drain(ds.ExecuteKsql(ctx, "  ", nil))
		require.Len(t, tables, 1)
		assert.True(t, tables[0].IsError)
	})
}

func TestMockExecuteKsql_SelectStreamsAndCancels(t *testing.T) {
	ds := newMock()
	ctx, cancel := context.WithCancel(context.Background())

	ch, err := ds.ExecuteKsql(ctx, "SELECT * FROM PAGEVIEWS EMIT CHANGES;", nil)
	require.NoError(t, err)

	first := <-ch // schema table
	assert.Equal(t, "Schema", first.Title)
	assert.NotEmpty(t, first.Columns)

	// At least one row should arrive within a couple ticks.
	select {
	case row := <-ch:
		assert.Equal(t, "Row", row.Title)
	case <-time.After(2 * time.Second):
		t.Fatal("no row emitted")
	}

	cancel()
	closed := make(chan struct{})
	go func() {
		for range ch {
		}
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("channel not closed after ctx cancel")
	}
}

func drain(ch <-chan api.KsqlResultTable, _ error) []api.KsqlResultTable {
	var out []api.KsqlResultTable
	for t := range ch {
		out = append(out, t)
	}
	return out
}
