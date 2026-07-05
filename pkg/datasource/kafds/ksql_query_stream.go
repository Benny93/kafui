package kafds

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
)

// ExecuteKsql validates and executes a single ksqlDB statement, delivering all
// outcomes on the returned channel. Validation (KS-6) runs first; a failure is
// emitted as a single error table and the channel is closed. Statement-kind
// input is posted to /ksql and its result tables emitted; SELECT input opens the
// /query stream and emits a schema table followed by one table per data row.
// Cancelling ctx closes the underlying HTTP body (terminating the server-side
// query) and closes the channel. A KsqlNotConfiguredError is returned (with a
// nil channel) when no endpoint is configured.
func (kp KafkaDataSourceKaf) ExecuteKsql(ctx context.Context, sql string, props map[string]string) (<-chan api.KsqlResultTable, error) {
	kind, verr := classifyKsqlStatement(sql)
	if verr != nil {
		ch := make(chan api.KsqlResultTable, 1)
		ch <- *verr
		close(ch)
		return ch, nil
	}

	c, err := kp.ksqlClient()
	if err != nil {
		return nil, err
	}

	out := make(chan api.KsqlResultTable, 8)

	if kind == ksqlKindStatement {
		go func() {
			defer close(out)
			for _, t := range executeStatement(ctx, c, sql, props) {
				select {
				case out <- t:
				case <-ctx.Done():
					return
				}
			}
		}()
		return out, nil
	}

	// Query kind: open the /query stream and parse incrementally.
	body, err := c.openStream(ctx, "/query", ksqlStatementRequest(sql, props))
	if err != nil {
		// Surface as an error table on the channel so the UI has one render path.
		ch := make(chan api.KsqlResultTable, 1)
		ch <- ksqlErrorTable(err.Error())
		close(ch)
		return ch, nil
	}
	go func() {
		defer close(out)
		defer body.Close()
		parseQueryStream(ctx, body, out)
	}()
	return out, nil
}

// queryStreamElement is one element of a ksqlDB /query streaming response.
type queryStreamElement struct {
	Header *struct {
		QueryID string `json:"queryId"`
		Schema  string `json:"schema"`
	} `json:"header"`
	Row *struct {
		Columns []json.RawMessage `json:"columns"`
	} `json:"row"`
	ErrorMessage *struct {
		Message       string   `json:"message"`
		StatementText string   `json:"statementText"`
		Entities      []string `json:"entities"`
	} `json:"errorMessage"`
	FinalMessage string `json:"finalMessage"`
}

// parseQueryStream decodes the ksqlDB /query streaming JSON array incrementally,
// emitting a schema table for the header and one row table per data row. An
// in-stream errorMessage terminates with an error table; a finalMessage
// terminates cleanly. A body that ends without closing the JSON array (a known
// server defect) is treated as normal completion, not a parse error.
func parseQueryStream(ctx context.Context, r io.Reader, out chan<- api.KsqlResultTable) {
	dec := json.NewDecoder(r)
	// Consume the opening '[' token. A truncated/empty stream ends cleanly.
	if _, err := dec.Token(); err != nil {
		return
	}

	var columns []string
	for dec.More() {
		if ctx.Err() != nil {
			return
		}
		var el queryStreamElement
		if err := dec.Decode(&el); err != nil {
			// EOF / truncated array (known server defect) ⇒ clean completion.
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return
			}
			return
		}

		switch {
		case el.ErrorMessage != nil:
			t := api.KsqlResultTable{
				Title:   "Error",
				IsError: true,
				Columns: []string{"Message", "Statement", "Entities"},
				Rows:    [][]string{{el.ErrorMessage.Message, el.ErrorMessage.StatementText, strings.Join(el.ErrorMessage.Entities, ", ")}},
			}
			send(ctx, out, t)
			return
		case el.FinalMessage != "":
			return
		case el.Header != nil:
			columns = parseKsqlSchema(el.Header.Schema)
			if !send(ctx, out, api.KsqlResultTable{Title: "Schema", Columns: columns}) {
				return
			}
		case el.Row != nil:
			row := make([]string, len(el.Row.Columns))
			for i, c := range el.Row.Columns {
				row[i] = renderJSONValue(c)
			}
			if !send(ctx, out, api.KsqlResultTable{Title: "Row", Columns: columns, Rows: [][]string{row}}) {
				return
			}
		}
	}
}

// send delivers a table unless the context is cancelled first. It reports
// whether the send succeeded.
func send(ctx context.Context, out chan<- api.KsqlResultTable, t api.KsqlResultTable) bool {
	select {
	case out <- t:
		return true
	case <-ctx.Done():
		return false
	}
}

// parseKsqlSchema extracts column names from a ksqlDB schema string, splitting on
// top-level commas (ignoring commas inside STRUCT<...>) and unwrapping
// backtick-quoted identifiers. Example:
//
//	"`ID` STRING, `ADDR` STRUCT<`CITY` STRING, `ZIP` INT>" -> ["ID", "ADDR"]
func parseKsqlSchema(schema string) []string {
	var cols []string
	depth := 0
	start := 0
	for i := 0; i < len(schema); i++ {
		switch schema[i] {
		case '<':
			depth++
		case '>':
			depth--
		case ',':
			if depth == 0 {
				if name := ksqlColumnName(schema[start:i]); name != "" {
					cols = append(cols, name)
				}
				start = i + 1
			}
		}
	}
	if start < len(schema) {
		if name := ksqlColumnName(schema[start:]); name != "" {
			cols = append(cols, name)
		}
	}
	return cols
}

// ksqlColumnName extracts the column name from a single "`NAME` TYPE" field.
func ksqlColumnName(field string) string {
	field = strings.TrimSpace(field)
	if field == "" {
		return ""
	}
	if field[0] == '`' {
		if end := strings.IndexByte(field[1:], '`'); end >= 0 {
			return field[1 : 1+end]
		}
	}
	if sp := strings.IndexAny(field, " \t"); sp >= 0 {
		return field[:sp]
	}
	return field
}
