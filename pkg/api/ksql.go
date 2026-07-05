package api

// KsqlStream is one ksqlDB stream as reported by LIST STREAMS. KeyFormat is
// empty when the server reports only a single legacy `format` field (mapped to
// ValueFormat).
type KsqlStream struct {
	Name        string
	Topic       string
	KeyFormat   string
	ValueFormat string
}

// KsqlTable is one ksqlDB table as reported by LIST TABLES. Windowed reports
// whether the table is windowed.
type KsqlTable struct {
	Name        string
	Topic       string
	KeyFormat   string
	ValueFormat string
	Windowed    bool
}

// KsqlResultTable is the universal result unit for every ksqlDB execution
// outcome. A schema announcement, a batch of data rows, a statement response,
// and an error all travel as this one type so the UI has a single render path.
//
// IsError marks the table as an error report (rendered as a status-bar error
// rather than a data grid). A table with no columns is a placeholder the UI
// renders as "no results".
type KsqlResultTable struct {
	Title   string
	Columns []string
	Rows    [][]string
	IsError bool
}
