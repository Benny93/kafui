package kafds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
)

// interpretStatementResponse converts a /ksql HTTP status + body into one or
// more KsqlResultTables. Success bodies are interpreted per their @type markers;
// unknown types render generically; an empty body becomes a synthetic success
// table. Non-2xx responses become an error table (structured when the ksql error
// body parses, raw otherwise). Errors always travel as tables, never as returned
// Go errors.
func interpretStatementResponse(status int, body []byte) []api.KsqlResultTable {
	if status < 200 || status >= 300 {
		return []api.KsqlResultTable{errorTableFromResponse(status, body)}
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return []api.KsqlResultTable{successTable()}
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(body, &entries); err != nil {
		return []api.KsqlResultTable{errorTableFromResponse(status, body)}
	}
	var out []api.KsqlResultTable
	for _, e := range entries {
		out = append(out, interpretEntry(e))
	}
	if len(out) == 0 {
		return []api.KsqlResultTable{successTable()}
	}
	return out
}

// interpretEntry renders one @type-tagged response entry into a titled table.
func interpretEntry(raw json.RawMessage) api.KsqlResultTable {
	var typed struct {
		Type string `json:"@type"`
	}
	_ = json.Unmarshal(raw, &typed)

	switch typed.Type {
	case "currentStatus":
		var e struct {
			CommandID     string `json:"commandId"`
			CommandStatus struct {
				Status  string `json:"status"`
				Message string `json:"message"`
			} `json:"commandStatus"`
		}
		_ = json.Unmarshal(raw, &e)
		return api.KsqlResultTable{
			Title:   "Status",
			Columns: []string{"Command", "Status", "Message"},
			Rows:    [][]string{{e.CommandID, e.CommandStatus.Status, e.CommandStatus.Message}},
		}
	case "streams":
		var e ksqlListEntry
		_ = json.Unmarshal(raw, &e)
		t := api.KsqlResultTable{Title: "Streams", Columns: []string{"Name", "Topic", "Key Format", "Value Format"}}
		for _, s := range e.Streams {
			t.Rows = append(t.Rows, []string{s.Name, s.Topic, s.KeyFormat, firstNonEmpty(s.ValueFormat, s.Format)})
		}
		return t
	case "tables":
		var e ksqlListEntry
		_ = json.Unmarshal(raw, &e)
		t := api.KsqlResultTable{Title: "Tables", Columns: []string{"Name", "Topic", "Key Format", "Value Format", "Windowed"}}
		for _, tb := range e.Tables {
			t.Rows = append(t.Rows, []string{tb.Name, tb.Topic, tb.KeyFormat, firstNonEmpty(tb.ValueFormat, tb.Format), fmt.Sprintf("%t", tb.IsWindowed)})
		}
		return t
	case "queries":
		var e struct {
			Queries []struct {
				ID          string   `json:"id"`
				QueryString string   `json:"queryString"`
				Sinks       []string `json:"sinks"`
				State       string   `json:"state"`
			} `json:"queries"`
		}
		_ = json.Unmarshal(raw, &e)
		t := api.KsqlResultTable{Title: "Queries", Columns: []string{"ID", "State", "Sinks", "Query"}}
		for _, q := range e.Queries {
			t.Rows = append(t.Rows, []string{q.ID, q.State, strings.Join(q.Sinks, ", "), q.QueryString})
		}
		return t
	case "kafka_topics":
		var e struct {
			Topics []struct {
				Name            string `json:"name"`
				ReplicaInfo     []int  `json:"replicaInfo"`
				RegisteredCount int    `json:"registeredCount"`
			} `json:"topics"`
		}
		_ = json.Unmarshal(raw, &e)
		t := api.KsqlResultTable{Title: "Kafka Topics", Columns: []string{"Name", "Partitions", "Registered"}}
		for _, tp := range e.Topics {
			t.Rows = append(t.Rows, []string{tp.Name, fmt.Sprintf("%d", len(tp.ReplicaInfo)), fmt.Sprintf("%d", tp.RegisteredCount)})
		}
		return t
	case "properties":
		var e struct {
			Properties json.RawMessage `json:"properties"`
		}
		_ = json.Unmarshal(raw, &e)
		return keyValueTable("Properties", e.Properties)
	case "sourceDescription":
		var e struct {
			SourceDescription struct {
				Name        string `json:"name"`
				Type        string `json:"type"`
				Topic       string `json:"topic"`
				KeyFormat   string `json:"keyFormat"`
				ValueFormat string `json:"valueFormat"`
				Fields      []struct {
					Name   string `json:"name"`
					Schema struct {
						Type string `json:"type"`
					} `json:"schema"`
				} `json:"fields"`
			} `json:"sourceDescription"`
		}
		_ = json.Unmarshal(raw, &e)
		d := e.SourceDescription
		t := api.KsqlResultTable{Title: "Description: " + d.Name, Columns: []string{"Field", "Type"}}
		for _, f := range d.Fields {
			t.Rows = append(t.Rows, []string{f.Name, f.Schema.Type})
		}
		if len(t.Rows) == 0 {
			// No fields available — fall back to a summary row set.
			t.Columns = []string{"Property", "Value"}
			t.Rows = [][]string{
				{"name", d.Name}, {"type", d.Type}, {"topic", d.Topic},
				{"keyFormat", d.KeyFormat}, {"valueFormat", d.ValueFormat},
			}
		}
		return t
	default:
		return genericEntryTable(typed.Type, raw)
	}
}

// genericEntryTable renders an unrecognized entry as a Property/Value table of
// its top-level JSON fields.
func genericEntryTable(typ string, raw json.RawMessage) api.KsqlResultTable {
	title := "Result"
	if typ != "" {
		title = typ
	}
	return keyValueTable(title, raw)
}

// keyValueTable renders a JSON object as sorted Property/Value rows.
func keyValueTable(title string, raw json.RawMessage) api.KsqlResultTable {
	t := api.KsqlResultTable{Title: title, Columns: []string{"Property", "Value"}}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || len(obj) == 0 {
		return t
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		if k == "@type" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Rows = append(t.Rows, []string{k, renderJSONValue(obj[k])})
	}
	return t
}

// errorTableFromResponse builds an error-flagged table from a non-2xx response.
// A parseable ksql error body yields type/error_code/message/statementText/
// entities columns; an unparseable body yields HTTP status + raw text.
func errorTableFromResponse(status int, body []byte) api.KsqlResultTable {
	var e struct {
		Type          string   `json:"@type"`
		ErrorCode     int      `json:"error_code"`
		Message       string   `json:"message"`
		StatementText string   `json:"statementText"`
		Entities      []string `json:"entities"`
	}
	if json.Unmarshal(body, &e) == nil && e.Message != "" {
		return api.KsqlResultTable{
			Title:   "Error",
			IsError: true,
			Columns: []string{"Type", "Error Code", "Message", "Statement", "Entities"},
			Rows: [][]string{{
				e.Type,
				fmt.Sprintf("%d", e.ErrorCode),
				e.Message,
				e.StatementText,
				strings.Join(e.Entities, ", "),
			}},
		}
	}
	return api.KsqlResultTable{
		Title:   "Error",
		IsError: true,
		Columns: []string{"HTTP Status", "Response"},
		Rows:    [][]string{{fmt.Sprintf("%d", status), strings.TrimSpace(string(body))}},
	}
}

// ksqlErrorTable builds an error-flagged table from an arbitrary error message.
func ksqlErrorTable(msg string) api.KsqlResultTable {
	return api.KsqlResultTable{
		Title:   "Error",
		IsError: true,
		Columns: []string{"Error"},
		Rows:    [][]string{{msg}},
	}
}

func successTable() api.KsqlResultTable {
	return api.KsqlResultTable{
		Title:   "Success",
		Columns: []string{"Result"},
		Rows:    [][]string{{"Statement executed successfully"}},
	}
}

// renderJSONValue formats a raw JSON value as a string: JSON strings are
// unquoted; objects/arrays are rendered compactly; scalars use their literal.
func renderJSONValue(raw json.RawMessage) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return ""
	}
	var v interface{}
	if err := json.Unmarshal(trimmed, &v); err != nil {
		return string(trimmed)
	}
	switch val := v.(type) {
	case string:
		return val
	case nil:
		return ""
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		return string(trimmed)
	}
}
