package ksql

import (
	"context"

	"github.com/Benny93/kafui/pkg/api"
)

// --- overview page ---

// tab identifies the active overview tab.
type tab int

const (
	// tabTables is the default per the spec.
	tabTables tab = iota
	tabStreams
)

func (t tab) String() string {
	if t == tabStreams {
		return "Streams"
	}
	return "Tables"
}

// streamsLoadedMsg / tablesLoadedMsg carry the parallel listing fetches.
type (
	streamsLoadedMsg struct {
		streams []api.KsqlStream
		err     error
	}
	tablesLoadedMsg struct {
		tables []api.KsqlTable
		err    error
	}
)

// --- query page ---

// queryStartedMsg is emitted when ExecuteKsql returns its result channel. The
// channel is the single-use "pipe"; Cancel terminates the server-side query.
type queryStartedMsg struct {
	ch     <-chan api.KsqlResultTable
	cancel context.CancelFunc
}

// ksqlResultMsg carries one result table drained from the channel. ok is false
// when the channel has closed (query complete / aborted / errored).
type ksqlResultMsg struct {
	table api.KsqlResultTable
	ok    bool
}

// queryTickMsg re-arms the channel drain when no table arrived within the
// listen window (keeps the Update loop responsive without busy-waiting).
type queryTickMsg struct{}
