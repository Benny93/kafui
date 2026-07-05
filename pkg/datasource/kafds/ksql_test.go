package kafds

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- client (KS-3) ---

func TestKsqlClient_PostAuthAndContentType(t *testing.T) {
	var gotMethod, gotAuth, gotAccept, gotContentType, gotBody, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		gotContentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c, err := newKsqlClient(&appconfig.KsqlEndpoint{URL: srv.URL, Username: "u", Password: "p"})
	require.NoError(t, err)

	var out map[string]bool
	require.NoError(t, c.doPost("/ksql", map[string]string{"ksql": "LIST STREAMS;"}, &out))
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/ksql", gotPath)
	assert.True(t, strings.HasPrefix(gotAuth, "Basic "))
	assert.Equal(t, ksqlAcceptType, gotAccept)
	assert.Equal(t, ksqlAcceptType, gotContentType)
	assert.JSONEq(t, `{"ksql":"LIST STREAMS;"}`, gotBody)
	assert.True(t, out["ok"])
}

func TestKsqlClient_NoAuthWhenUnset(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	c, err := newKsqlClient(&appconfig.KsqlEndpoint{URL: srv.URL})
	require.NoError(t, err)
	require.NoError(t, c.doPost("/ksql", nil, nil))
	assert.Empty(t, gotAuth)
}

func TestKsqlClient_ServerErrorMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error_code":40001,"message":"bad statement"}`))
	}))
	defer srv.Close()
	c, _ := newKsqlClient(&appconfig.KsqlEndpoint{URL: srv.URL})
	err := c.doPost("/ksql", nil, nil)
	var se api.KsqlServerError
	require.True(t, errors.As(err, &se))
	assert.Equal(t, 400, se.StatusCode)
	assert.Equal(t, 40001, se.ErrorCode)
	assert.Equal(t, "bad statement", se.Message)
}

func TestKsqlClient_Failover(t *testing.T) {
	live := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer live.Close()

	// Dead URL first, live second — failover must reach the live one.
	c, err := newKsqlClient(&appconfig.KsqlEndpoint{URL: "http://127.0.0.1:1," + live.URL})
	require.NoError(t, err)
	var out map[string]bool
	require.NoError(t, c.doPost("/ksql", nil, &out))
	assert.True(t, out["ok"])
}

func TestKsqlClient_AllDead(t *testing.T) {
	c, err := newKsqlClient(&appconfig.KsqlEndpoint{URL: "http://127.0.0.1:1,http://127.0.0.1:2"})
	require.NoError(t, err)
	err = c.doPost("/ksql", nil, nil)
	var noInst api.KsqlNoInstancesError
	require.True(t, errors.As(err, &noInst))
	assert.Equal(t, 2, noInst.Configured)
}

func TestKsqlClient_ResponseSizeLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("A", 1000)))
	}))
	defer srv.Close()
	c, _ := newKsqlClient(&appconfig.KsqlEndpoint{URL: srv.URL, MaxResponseBytes: 10})
	status, body, err := c.doRaw(context.Background(), http.MethodPost, "/ksql", nil)
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Len(t, body, 10) // truncated at the limit
}

func TestNewKsqlClient_NotConfigured(t *testing.T) {
	c, err := newKsqlClient(nil)
	require.NoError(t, err)
	assert.Nil(t, c)
	c, err = newKsqlClient(&appconfig.KsqlEndpoint{URL: "   "})
	require.NoError(t, err)
	assert.Nil(t, c)
}

// withKsql overrides the ksql endpoint resolver for the duration of a test. It
// also neutralizes the package-level cfg.Clusters so KafkaDataSourceKaf{}.
// GetContext takes its safe early-return path (a zero-value datasource has a nil
// configManager) regardless of state left by other tests in the package.
func withKsql(t *testing.T, ep *appconfig.KsqlEndpoint) {
	t.Helper()
	prev := loadKsqlEndpoint
	loadKsqlEndpoint = func(context string) *appconfig.KsqlEndpoint { return ep }
	prevClusters := cfg.Clusters
	cfg.Clusters = nil
	t.Cleanup(func() {
		loadKsqlEndpoint = prev
		cfg.Clusters = prevClusters
	})
}

func TestListKsqlStreams_NotConfigured(t *testing.T) {
	withKsql(t, nil)
	_, err := KafkaDataSourceKaf{}.ListKsqlStreams()
	var nc api.KsqlNotConfiguredError
	assert.True(t, errors.As(err, &nc))
}

// --- listings (KS-5) ---

func TestListKsqlStreams_ModernAndLegacy(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantKey string
		wantVal string
	}{
		{
			name:    "modern",
			body:    `[{"@type":"streams","streams":[{"name":"S1","topic":"t1","keyFormat":"KAFKA","valueFormat":"JSON"}]}]`,
			wantKey: "KAFKA", wantVal: "JSON",
		},
		{
			name:    "legacy format field",
			body:    `[{"@type":"streams","streams":[{"name":"S1","topic":"t1","format":"AVRO"}]}]`,
			wantKey: "", wantVal: "AVRO",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			withKsql(t, &appconfig.KsqlEndpoint{URL: srv.URL})
			streams, err := KafkaDataSourceKaf{}.ListKsqlStreams()
			require.NoError(t, err)
			require.Len(t, streams, 1)
			assert.Equal(t, "S1", streams[0].Name)
			assert.Equal(t, tc.wantKey, streams[0].KeyFormat)
			assert.Equal(t, tc.wantVal, streams[0].ValueFormat)
		})
	}
}

func TestListKsqlTables_WindowedAndUnexpected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"@type":"tables","tables":[{"name":"T1","topic":"t1","keyFormat":"KAFKA","valueFormat":"JSON","isWindowed":true}]}]`))
	}))
	defer srv.Close()
	withKsql(t, &appconfig.KsqlEndpoint{URL: srv.URL})
	tables, err := KafkaDataSourceKaf{}.ListKsqlTables()
	require.NoError(t, err)
	require.Len(t, tables, 1)
	assert.True(t, tables[0].Windowed)

	// Unexpected payload -> descriptive error.
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"@type":"currentStatus"}]`))
	}))
	defer srv2.Close()
	withKsql(t, &appconfig.KsqlEndpoint{URL: srv2.URL})
	_, err = KafkaDataSourceKaf{}.ListKsqlTables()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "table list could not be retrieved")
}

// --- statement classification (KS-6) ---

func TestClassifyKsqlStatement(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		wantKind ksqlStatementKind
		wantErr  bool
		errMsg   string
	}{
		{name: "select", sql: "SELECT * FROM S1;", wantKind: ksqlKindQuery},
		{name: "select emit changes", sql: "SELECT * FROM S1 EMIT CHANGES;", wantKind: ksqlKindQuery},
		{name: "select no semicolon", sql: "select a from b", wantKind: ksqlKindQuery},
		{name: "create", sql: "CREATE STREAM s AS SELECT * FROM t;", wantKind: ksqlKindStatement},
		{name: "insert", sql: "INSERT INTO s VALUES (1);", wantKind: ksqlKindStatement},
		{name: "show streams", sql: "SHOW STREAMS;", wantKind: ksqlKindStatement},
		{name: "describe", sql: "DESCRIBE s;", wantKind: ksqlKindStatement},
		{name: "terminate", sql: "TERMINATE query1;", wantKind: ksqlKindStatement},
		{name: "empty", sql: "   ", wantErr: true, errMsg: "no valid statement was found"},
		{name: "comment only", sql: "-- just a comment", wantErr: true, errMsg: "no valid statement"},
		{name: "block comment only", sql: "/* nothing */", wantErr: true, errMsg: "no valid statement"},
		{name: "multi statement", sql: "SELECT 1; SELECT 2;", wantErr: true, errMsg: "only a single statement"},
		{name: "print", sql: "PRINT 'topic';", wantErr: true, errMsg: "unsupported"},
		{name: "define", sql: "DEFINE x = '1';", wantErr: true, errMsg: "unsupported"},
		{name: "undefine", sql: "UNDEFINE x;", wantErr: true, errMsg: "unsupported"},
		{name: "gibberish", sql: "foobar baz;", wantErr: true, errMsg: "unsupported"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kind, errTable := classifyKsqlStatement(tc.sql)
			if tc.wantErr {
				require.NotNil(t, errTable)
				assert.True(t, errTable.IsError)
				assert.Contains(t, errTable.Rows[0][0], tc.errMsg)
				return
			}
			require.Nil(t, errTable)
			assert.Equal(t, tc.wantKind, kind)
		})
	}
}

// --- statement execution + interpretation (KS-7) ---

func TestExecuteKsql_StatementProperties(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`[{"@type":"currentStatus","commandId":"stream/S/create","commandStatus":{"status":"SUCCESS","message":"Created"}}]`))
	}))
	defer srv.Close()
	withKsql(t, &appconfig.KsqlEndpoint{URL: srv.URL})

	// With properties.
	ch, err := KafkaDataSourceKaf{}.ExecuteKsql(context.Background(), "CREATE STREAM s AS SELECT * FROM t;", map[string]string{"auto.offset.reset": "earliest"})
	require.NoError(t, err)
	tables := drain(ch)
	require.Len(t, tables, 1)
	assert.Equal(t, "Status", tables[0].Title)
	assert.Equal(t, "SUCCESS", tables[0].Rows[0][1])
	props, _ := gotBody["streamsProperties"].(map[string]interface{})
	assert.Equal(t, "earliest", props["auto.offset.reset"])

	// Without properties -> empty streamsProperties object.
	_, err = KafkaDataSourceKaf{}.ExecuteKsql(context.Background(), "DROP STREAM s;", nil)
	require.NoError(t, err)
	time.Sleep(20 * time.Millisecond)
	props, ok := gotBody["streamsProperties"].(map[string]interface{})
	require.True(t, ok)
	assert.Empty(t, props)
}

func TestInterpretStatementResponse(t *testing.T) {
	t.Run("empty body -> success", func(t *testing.T) {
		out := interpretStatementResponse(200, []byte(""))
		require.Len(t, out, 1)
		assert.Equal(t, "Success", out[0].Title)
		assert.False(t, out[0].IsError)
	})
	t.Run("streams type", func(t *testing.T) {
		out := interpretStatementResponse(200, []byte(`[{"@type":"streams","streams":[{"name":"S","topic":"t","keyFormat":"KAFKA","valueFormat":"JSON"}]}]`))
		require.Len(t, out, 1)
		assert.Equal(t, "Streams", out[0].Title)
		assert.Equal(t, []string{"S", "t", "KAFKA", "JSON"}, out[0].Rows[0])
	})
	t.Run("unknown type -> generic", func(t *testing.T) {
		out := interpretStatementResponse(200, []byte(`[{"@type":"mystery","foo":"bar","n":3}]`))
		require.Len(t, out, 1)
		assert.Equal(t, "mystery", out[0].Title)
		assert.Equal(t, []string{"Property", "Value"}, out[0].Columns)
	})
	t.Run("structured error", func(t *testing.T) {
		out := interpretStatementResponse(400, []byte(`{"@type":"statement_error","error_code":40001,"message":"boom","statementText":"CREATE ...","entities":["e1"]}`))
		require.Len(t, out, 1)
		assert.True(t, out[0].IsError)
		assert.Equal(t, "boom", out[0].Rows[0][2])
		assert.Equal(t, "e1", out[0].Rows[0][4])
	})
	t.Run("garbage error body", func(t *testing.T) {
		out := interpretStatementResponse(500, []byte(`not json at all`))
		require.Len(t, out, 1)
		assert.True(t, out[0].IsError)
		assert.Equal(t, []string{"HTTP Status", "Response"}, out[0].Columns)
		assert.Equal(t, "not json at all", out[0].Rows[0][1])
	})
}

// --- streaming query (KS-8) ---

func TestExecuteKsql_StreamsSchemaThenRows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", ksqlAcceptType)
		_, _ = io.WriteString(w, "[")
		_, _ = io.WriteString(w, `{"header":{"queryId":"q1","schema":"`+"`USERID`"+` STRING, `+"`ADDR`"+` STRUCT<`+"`CITY`"+` STRING, `+"`ZIP`"+` INT>"}},`)
		if fl != nil {
			fl.Flush()
		}
		_, _ = io.WriteString(w, `{"row":{"columns":["u1",{"CITY":"NY","ZIP":10001}]}},`)
		_, _ = io.WriteString(w, `{"row":{"columns":["u2",{"CITY":"LA","ZIP":90001}]}}`)
		_, _ = io.WriteString(w, "]")
	}))
	defer srv.Close()
	withKsql(t, &appconfig.KsqlEndpoint{URL: srv.URL})

	ch, err := KafkaDataSourceKaf{}.ExecuteKsql(context.Background(), "SELECT * FROM S1 EMIT CHANGES;", nil)
	require.NoError(t, err)
	tables := drain(ch)
	require.GreaterOrEqual(t, len(tables), 3)
	// First is the schema (nested struct collapses to two columns).
	assert.Equal(t, "Schema", tables[0].Title)
	assert.Equal(t, []string{"USERID", "ADDR"}, tables[0].Columns)
	assert.Empty(t, tables[0].Rows)
	// Following are row tables.
	assert.Equal(t, "Row", tables[1].Title)
	assert.Equal(t, "u1", tables[1].Rows[0][0])
}

func TestExecuteKsql_InStreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `[{"header":{"schema":"`+"`A`"+` INT"}},`)
		_, _ = io.WriteString(w, `{"errorMessage":{"message":"query failed","statementText":"SELECT ...","entities":[]}}]`)
	}))
	defer srv.Close()
	withKsql(t, &appconfig.KsqlEndpoint{URL: srv.URL})
	ch, err := KafkaDataSourceKaf{}.ExecuteKsql(context.Background(), "SELECT * FROM S1;", nil)
	require.NoError(t, err)
	tables := drain(ch)
	require.NotEmpty(t, tables)
	last := tables[len(tables)-1]
	assert.True(t, last.IsError)
	assert.Equal(t, "query failed", last.Rows[0][0])
}

func TestExecuteKsql_TruncatedStreamCleanCompletion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No closing ']' — known server defect must be treated as completion.
		_, _ = io.WriteString(w, `[{"header":{"schema":"`+"`A`"+` INT"}},`)
		_, _ = io.WriteString(w, `{"row":{"columns":[1]}}`)
	}))
	defer srv.Close()
	withKsql(t, &appconfig.KsqlEndpoint{URL: srv.URL})
	ch, err := KafkaDataSourceKaf{}.ExecuteKsql(context.Background(), "SELECT * FROM S1;", nil)
	require.NoError(t, err)
	tables := drain(ch)
	require.Len(t, tables, 2)
	assert.False(t, tables[len(tables)-1].IsError)
}

func TestExecuteKsql_ContextCancelClosesChannel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		_, _ = io.WriteString(w, `[{"header":{"schema":"`+"`A`"+` INT"}},`)
		if fl != nil {
			fl.Flush()
		}
		// Keep the connection open until the client disconnects.
		<-r.Context().Done()
	}))
	defer srv.Close()
	withKsql(t, &appconfig.KsqlEndpoint{URL: srv.URL})

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := KafkaDataSourceKaf{}.ExecuteKsql(ctx, "SELECT * FROM S1;", nil)
	require.NoError(t, err)
	<-ch // schema table
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
		t.Fatal("channel not closed promptly after ctx cancel")
	}
}

func TestExecuteKsql_ValidationErrorNoServerCall(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer srv.Close()
	withKsql(t, &appconfig.KsqlEndpoint{URL: srv.URL})
	ch, err := KafkaDataSourceKaf{}.ExecuteKsql(context.Background(), "PRINT 'topic';", nil)
	require.NoError(t, err)
	tables := drain(ch)
	require.Len(t, tables, 1)
	assert.True(t, tables[0].IsError)
	assert.False(t, called, "invalid statement must not reach the server")
}

func drain(ch <-chan api.KsqlResultTable) []api.KsqlResultTable {
	var out []api.KsqlResultTable
	for t := range ch {
		out = append(out, t)
	}
	return out
}
