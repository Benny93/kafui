package mock

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
)

// mockKsqlStreams / mockKsqlTables are the canned listings the mock returns so
// the ksql overview page is exercisable via `make run-mock`.
var mockKsqlStreams = []api.KsqlStream{
	{Name: "PAGEVIEWS", Topic: "pageviews", KeyFormat: "KAFKA", ValueFormat: "JSON"},
	{Name: "USERS_STREAM", Topic: "users", KeyFormat: "KAFKA", ValueFormat: "AVRO"},
	{Name: "ORDERS_ENRICHED", Topic: "orders_enriched", KeyFormat: "KAFKA", ValueFormat: "PROTOBUF"},
}

var mockKsqlTables = []api.KsqlTable{
	{Name: "USERS", Topic: "users", KeyFormat: "KAFKA", ValueFormat: "AVRO", Windowed: false},
	{Name: "PAGEVIEWS_PER_USER", Topic: "PAGEVIEWS_PER_USER", KeyFormat: "KAFKA", ValueFormat: "JSON", Windowed: true},
}

func (kp *KafkaDataSourceMock) ListKsqlStreams() ([]api.KsqlStream, error) {
	return append([]api.KsqlStream(nil), mockKsqlStreams...), nil
}

func (kp *KafkaDataSourceMock) ListKsqlTables() ([]api.KsqlTable, error) {
	return append([]api.KsqlTable(nil), mockKsqlTables...), nil
}

// ExecuteKsql mirrors the real datasource contract: validation runs first
// (sharing the semantics of the real classifier), SHOW/LIST/DESCRIBE return
// canned tables, DDL returns a success table, and SELECT streams a schema table
// followed by a ticking row every ~300 ms until ctx is cancelled.
func (kp *KafkaDataSourceMock) ExecuteKsql(ctx context.Context, sql string, props map[string]string) (<-chan api.KsqlResultTable, error) {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return emitOne(api.KsqlResultTable{
			Title: "Validation error", Columns: []string{"Error"},
			Rows: [][]string{{"no valid statement was found"}}, IsError: true,
		}), nil
	}

	upper := strings.ToUpper(trimmed)
	keyword := strings.Fields(upper)[0]

	if keyword == "SELECT" {
		return kp.streamMockRows(ctx), nil
	}

	switch keyword {
	case "SHOW", "LIST":
		return emitOne(kp.mockListingTable(upper)), nil
	case "DESCRIBE", "EXPLAIN":
		return emitOne(api.KsqlResultTable{
			Title:   "Description",
			Columns: []string{"Field", "Type"},
			Rows:    [][]string{{"ROWTIME", "BIGINT"}, {"USERID", "STRING"}, {"PAGEID", "STRING"}},
		}), nil
	case "CREATE", "DROP", "INSERT", "TERMINATE", "SET", "UNSET", "ALTER":
		return emitOne(api.KsqlResultTable{
			Title:   "Success",
			Columns: []string{"Result"},
			Rows:    [][]string{{fmt.Sprintf("%s statement executed successfully", keyword)}},
		}), nil
	case "PRINT", "DEFINE", "UNDEFINE":
		return emitOne(api.KsqlResultTable{
			Title: "Validation error", Columns: []string{"Error"},
			Rows: [][]string{{"statement type is unsupported"}}, IsError: true,
		}), nil
	default:
		return emitOne(api.KsqlResultTable{
			Title: "Validation error", Columns: []string{"Error"},
			Rows: [][]string{{"statement type is unsupported"}}, IsError: true,
		}), nil
	}
}

// mockListingTable returns a canned SHOW/LIST result for streams, tables, or
// queries; unknown targets get a generic topics listing.
func (kp *KafkaDataSourceMock) mockListingTable(upper string) api.KsqlResultTable {
	switch {
	case strings.Contains(upper, "STREAM"):
		t := api.KsqlResultTable{Title: "Streams", Columns: []string{"Name", "Topic", "Key Format", "Value Format"}}
		for _, s := range mockKsqlStreams {
			t.Rows = append(t.Rows, []string{s.Name, s.Topic, s.KeyFormat, s.ValueFormat})
		}
		return t
	case strings.Contains(upper, "TABLE"):
		t := api.KsqlResultTable{Title: "Tables", Columns: []string{"Name", "Topic", "Key Format", "Value Format", "Windowed"}}
		for _, tb := range mockKsqlTables {
			t.Rows = append(t.Rows, []string{tb.Name, tb.Topic, tb.KeyFormat, tb.ValueFormat, fmt.Sprintf("%t", tb.Windowed)})
		}
		return t
	case strings.Contains(upper, "QUER"):
		return api.KsqlResultTable{
			Title:   "Queries",
			Columns: []string{"ID", "State", "Sinks", "Query"},
			Rows:    [][]string{{"CTAS_PAGEVIEWS_PER_USER_1", "RUNNING", "PAGEVIEWS_PER_USER", "CREATE TABLE PAGEVIEWS_PER_USER AS SELECT ..."}},
		}
	default:
		return api.KsqlResultTable{
			Title:   "Kafka Topics",
			Columns: []string{"Name", "Partitions"},
			Rows:    [][]string{{"pageviews", "1"}, {"users", "1"}},
		}
	}
}

// streamMockRows emits a schema table then a ticking row roughly every 300 ms
// until ctx is cancelled, so the live-result and abort UI flows are exercisable.
func (kp *KafkaDataSourceMock) streamMockRows(ctx context.Context) <-chan api.KsqlResultTable {
	out := make(chan api.KsqlResultTable, 4)
	cols := []string{"ROWTIME", "USERID", "PAGEID"}
	go func() {
		defer close(out)
		select {
		case out <- api.KsqlResultTable{Title: "Schema", Columns: cols}:
		case <-ctx.Done():
			return
		}
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				row := []string{
					fmt.Sprintf("%d", time.Now().UnixMilli()),
					fmt.Sprintf("User_%d", i%5),
					fmt.Sprintf("Page_%d", i%9),
				}
				select {
				case out <- api.KsqlResultTable{Title: "Row", Columns: cols, Rows: [][]string{row}}:
				case <-ctx.Done():
					return
				}
				i++
			}
		}
	}()
	return out
}

// emitOne returns a closed channel carrying a single table.
func emitOne(t api.KsqlResultTable) <-chan api.KsqlResultTable {
	ch := make(chan api.KsqlResultTable, 1)
	ch <- t
	close(ch)
	return ch
}
