package topic

import (
	"context"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/serde"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureDS wraps the mock datasource to capture ProduceMessage arguments.
type captureDS struct {
	*MockDataSource
	gotTopic string
	gotRec   api.ProduceRecord
	called   bool
}

func (c *captureDS) ProduceMessage(ctx context.Context, topic string, rec api.ProduceRecord) error {
	c.called = true
	c.gotTopic = topic
	c.gotRec = rec
	return nil
}

func TestProduceFormSubmitCallsProduceMessage(t *testing.T) {
	ds := &captureDS{MockDataSource: &MockDataSource{}}
	m := NewModel(ds, "orders", api.Topic{NumPartitions: 4})
	m.showProduce = true

	_, cmd := m.handlers.handleProduceFormSubmit(m, map[string]string{
		"key": "k", "value": "v", "headers": "h1=x", "partition": "1", "keep": "false",
	})
	require.NotNil(t, cmd)
	cmd() // executes the produce command

	assert.True(t, ds.called)
	assert.Equal(t, "orders", ds.gotTopic)
	assert.Equal(t, []byte("k"), ds.gotRec.Key)
	assert.Equal(t, []byte("v"), ds.gotRec.Value)
	require.Len(t, ds.gotRec.Headers, 1)
	assert.Equal(t, "h1", ds.gotRec.Headers[0].Key)
	require.NotNil(t, ds.gotRec.Partition)
	assert.Equal(t, int32(1), *ds.gotRec.Partition)
	assert.False(t, m.showProduce, "form closes when keep is false")
}

func TestSavedFilterRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	m := NewModel(&MockDataSource{}, "t", api.Topic{})
	require.Nil(t, m.setSmartFilter(`value contains "x"`)) // compiles cleanly
	require.NotNil(t, m.smartFilter)

	cmd := m.saveCurrentFilter()
	require.NotNil(t, cmd)
	cmd() // no error

	prefs := shared.LoadPrefs()
	require.Len(t, prefs.SavedFilters, 1)
	assert.Equal(t, `value contains "x"`, prefs.SavedFilters[0].Expr)

	// Applying the saved filter re-registers it on a fresh model.
	m2 := NewModel(&MockDataSource{}, "t", api.Topic{})
	assert.Nil(t, m2.setSmartFilter(prefs.SavedFilters[0].Expr))
	require.NotNil(t, m2.smartFilter)
	assert.Equal(t, m.smartFilter.ID(), m2.smartFilter.ID())
}

func TestSetSmartFilterCompileError(t *testing.T) {
	m := NewModel(&MockDataSource{}, "t", api.Topic{})
	cmd := m.setSmartFilter("value bogus") // unknown operator
	assert.NotNil(t, cmd, "compile error surfaces as a notification command")
	assert.Nil(t, m.smartFilter)
}

func TestApplyFilterSmartAndErrors(t *testing.T) {
	m := NewModel(&MockDataSource{}, "t", api.Topic{})
	m.messages = []api.Message{
		{Partition: 0, Offset: 1, Value: `{"amount":10}`},
		{Partition: 0, Offset: 2, Value: `not-json`},
		{Partition: 1, Offset: 3, Value: `{"amount":99}`},
	}
	require.Nil(t, m.setSmartFilter(`partition == 0`))
	m.applyFilter()
	assert.Len(t, m.filteredMessages, 2)
	assert.Equal(t, 0, m.smartFilterErrs)
}

func TestBuildSeekFlags(t *testing.T) {
	ofs := int64(42)
	tests := []struct {
		name    string
		mode    string
		value   string
		wantErr bool
		check   func(t *testing.T, f api.ConsumeFlags)
	}{
		{name: "newest", mode: "newest", check: func(t *testing.T, f api.ConsumeFlags) {
			assert.Equal(t, api.SeekNewest, f.Seek)
			assert.False(t, f.Follow)
		}},
		{name: "oldest", mode: "oldest", check: func(t *testing.T, f api.ConsumeFlags) {
			assert.Equal(t, api.SeekOldest, f.Seek)
			assert.Equal(t, "oldest", f.OffsetFlag)
		}},
		{name: "live", mode: "live", check: func(t *testing.T, f api.ConsumeFlags) {
			assert.Equal(t, api.SeekLive, f.Seek)
			assert.True(t, f.Follow)
			assert.Equal(t, int64(0), f.LimitMessages)
		}},
		{name: "from-offset", mode: "from-offset", value: "42", check: func(t *testing.T, f api.ConsumeFlags) {
			assert.Equal(t, api.SeekFromOffset, f.Seek)
			require.NotNil(t, f.SeekOffset)
			assert.Equal(t, ofs, *f.SeekOffset)
			assert.Equal(t, "42", f.OffsetFlag)
		}},
		{name: "to-offset", mode: "to-offset", value: "42", check: func(t *testing.T, f api.ConsumeFlags) {
			assert.Equal(t, api.SeekToOffset, f.Seek)
			require.NotNil(t, f.SeekOffset)
		}},
		{name: "from-offset bad value", mode: "from-offset", value: "abc", wantErr: true},
		{name: "from-timestamp rfc3339", mode: "from-timestamp", value: "2020-01-02T03:04:05Z", check: func(t *testing.T, f api.ConsumeFlags) {
			assert.Equal(t, api.SeekFromTimestamp, f.Seek)
			require.NotNil(t, f.SeekTimestamp)
			assert.Equal(t, 2020, f.SeekTimestamp.Year())
		}},
		{name: "to-timestamp relative", mode: "to-timestamp", value: "-1h", check: func(t *testing.T, f api.ConsumeFlags) {
			assert.Equal(t, api.SeekToTimestamp, f.Seek)
			require.NotNil(t, f.SeekTimestamp)
			assert.WithinDuration(t, time.Now().Add(-time.Hour), *f.SeekTimestamp, 5*time.Second)
		}},
		{name: "timestamp bad value", mode: "from-timestamp", value: "not-a-time", wantErr: true},
		{name: "unknown mode", mode: "sideways", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f, err := buildSeekFlags(tc.mode, tc.value, 100, nil)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NoError(t, f.Validate())
			if tc.check != nil {
				tc.check(t, f)
			}
		})
	}
}

func TestBuildPartitionFilter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		num     int32
		want    []int32
		wantErr bool
	}{
		{name: "empty = all", input: "", num: 4, want: nil},
		{name: "all keyword", input: "all", num: 4, want: nil},
		{name: "list", input: "0,2,3", num: 4, want: []int32{0, 2, 3}},
		{name: "spaces", input: " 1 , 2 ", num: 4, want: []int32{1, 2}},
		{name: "out of range", input: "0,9", num: 4, wantErr: true},
		{name: "negative", input: "-1", num: 4, wantErr: true},
		{name: "non-numeric", input: "a", num: 4, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildPartitionFilter(tc.input, tc.num)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBuildProduceRecord(t *testing.T) {
	t.Run("full record", func(t *testing.T) {
		rec, err := buildProduceRecord(map[string]string{
			"key": "k1", "value": "v1", "headers": "a=1,b=2", "partition": "2",
		}, 4)
		require.NoError(t, err)
		assert.Equal(t, []byte("k1"), rec.Key)
		assert.Equal(t, []byte("v1"), rec.Value)
		require.Len(t, rec.Headers, 2)
		assert.Equal(t, "a", rec.Headers[0].Key)
		assert.Equal(t, "1", rec.Headers[0].Value)
		require.NotNil(t, rec.Partition)
		assert.Equal(t, int32(2), *rec.Partition)
	})
	t.Run("blank key/value are null (nil)", func(t *testing.T) {
		rec, err := buildProduceRecord(map[string]string{"key": "", "value": "", "partition": "auto"}, 4)
		require.NoError(t, err)
		assert.Nil(t, rec.Key)
		assert.Nil(t, rec.Value)
		assert.Nil(t, rec.Partition)
	})
	t.Run("auto partition", func(t *testing.T) {
		rec, err := buildProduceRecord(map[string]string{"value": "x", "partition": "auto"}, 4)
		require.NoError(t, err)
		assert.Nil(t, rec.Partition)
	})
	t.Run("partition out of range", func(t *testing.T) {
		_, err := buildProduceRecord(map[string]string{"partition": "9"}, 4)
		assert.Error(t, err)
	})
	t.Run("partition not integer", func(t *testing.T) {
		_, err := buildProduceRecord(map[string]string{"partition": "x"}, 4)
		assert.Error(t, err)
	})
}

func TestProduceFieldsReproducePrefill(t *testing.T) {
	// MSG-32: reproduce prefills the form from a browsed message.
	msg := &api.Message{
		Key:   "the-key",
		Value: `{"a":1}`,
		Headers: []api.MessageHeader{
			{Key: "trace", Value: "abc"},
		},
	}
	fields := produceFields(msg)
	byName := map[string]string{}
	for _, f := range fields {
		byName[f.Name] = f.Default
	}
	assert.Equal(t, "the-key", byName["key"])
	assert.Equal(t, `{"a":1}`, byName["value"])
	assert.Equal(t, "trace=abc", byName["headers"])

	// A nil prefill (blank produce, MSG-31) leaves fields empty.
	blank := produceFields(nil)
	for _, f := range blank {
		if f.Name == "key" || f.Name == "value" || f.Name == "headers" {
			assert.Empty(t, f.Default)
		}
	}
}

func TestParseAndFormatHeaderRoundTrip(t *testing.T) {
	headers := parseHeaderField("a=1,b=two, c = three ")
	require.Len(t, headers, 3)
	assert.Equal(t, "a", headers[0].Key)
	assert.Equal(t, "1", headers[0].Value)
	assert.Equal(t, "c", headers[2].Key)
	assert.Equal(t, " three", headers[2].Value)
	assert.Equal(t, "a=1,b=two,c= three", formatHeaderField(headers))
}

func TestStringMatchHeadersAndUnicode(t *testing.T) {
	msg := api.Message{
		Key:   "orderKey",
		Value: `{"name":"widget"}`,
		Headers: []api.MessageHeader{
			{Key: "source", Value: "checkout"},
		},
	}
	assert.True(t, stringMatch(msg, "widget"), "value substring")
	assert.True(t, stringMatch(msg, "orderkey"), "key case-insensitive")
	assert.True(t, stringMatch(msg, "source"), "header name")
	assert.True(t, stringMatch(msg, "checkout"), "header value")
	assert.False(t, stringMatch(msg, "missing"))

	// Unicode-escape matching: a "ü" query matches content storing it escaped
	// as the literal characters ü (MSG-23).
	escapedContent := `{"city":"M` + "\\u00fc" + `nchen"}` // literal backslash u 0 0 f c
	escaped := api.Message{Value: escapedContent}
	assert.True(t, stringMatch(escaped, "ü"), "ü matches escaped \\u00fc")
	literal := api.Message{Value: "München"} // real ü rune
	assert.True(t, stringMatch(literal, "ü"), "ü matches literal ü")
}

func TestProjectField(t *testing.T) {
	payload := `{"user":{"id":42,"name":"ana"},"items":[{"sku":"x"},{"sku":"y"}],"active":true}`
	tests := []struct {
		path   string
		want   string
		wantOK bool
	}{
		{"user.name", "ana", true},
		{"user.id", "42", true},
		{"items.1.sku", "y", true},
		{"active", "true", true},
		{"user.missing", "", false},
		{"", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got, ok := projectField(payload, tc.path)
			assert.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				assert.Equal(t, tc.want, got)
			}
		})
	}
	// Non-JSON payload falls back (not ok).
	_, ok := projectField("not json", "a.b")
	assert.False(t, ok)
}

func TestApplySerde(t *testing.T) {
	m := &Model{}
	m.serdeReg, _ = serde.BuildRegistry(nil, nil)
	raw := []byte{0xDE, 0xAD}
	// "auto" leaves the already-decoded text untouched.
	assert.Equal(t, "decoded", m.applySerde("decoded", raw, "auto"))
	// explicit hex decodes the raw bytes.
	assert.Equal(t, "dead", m.applySerde("decoded", raw, "hex"))
	// explicit string on valid UTF-8 bytes.
	assert.Equal(t, "hello", m.applySerde("decoded", []byte("hello"), "string"))
	// hex without raw bytes hexes the text.
	assert.Equal(t, "6162", m.applySerde("ab", nil, "hex"))
}

func TestUnicodeEscape(t *testing.T) {
	assert.Equal(t, "abc", unicodeEscape("abc"))
	assert.Equal(t, "M\\u00fcnchen", unicodeEscape("München"))
}
