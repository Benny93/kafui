package shared

import (
	"bytes"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteMessagesCSV(t *testing.T) {
	ts1 := time.Date(2026, 7, 4, 12, 30, 0, 0, time.UTC)

	tests := []struct {
		name   string
		msgs   []api.Message
		format CSVFormat
		want   string
	}{
		{
			name: "default format golden two messages",
			msgs: []api.Message{
				{
					Partition: 0,
					Offset:    10,
					Timestamp: ts1,
					Key:       "k1",
					Value:     "v1",
					Headers:   []api.MessageHeader{{Key: "h1", Value: "a"}, {Key: "h2", Value: "b"}},
				},
				{
					Partition: 1,
					Offset:    20,
					Key:       "k2",
					Value:     "v2",
				},
			},
			format: DefaultCSVFormat(),
			want: "Partition,Offset,Timestamp,Key,Value,Headers\n" +
				"0,10,2026-07-04T12:30:00Z,k1,v1,\"h1=a,h2=b\"\n" +
				"1,20,,k2,v2,\n",
		},
		{
			name: "semicolon separator",
			msgs: []api.Message{
				{Partition: 2, Offset: 5, Key: "kk", Value: "vv"},
			},
			format: CSVFormat{Separator: ';', Quote: '"', LineTerminator: "\n"},
			want: "Partition;Offset;Timestamp;Key;Value;Headers\n" +
				"2;5;;kk;vv;\n",
		},
		{
			name:   "empty msgs header only",
			msgs:   nil,
			format: DefaultCSVFormat(),
			want:   "Partition,Offset,Timestamp,Key,Value,Headers\n",
		},
		{
			name: "quote all with special chars",
			msgs: []api.Message{
				{
					Partition: 0,
					Offset:    1,
					Key:       "key,with,comma",
					Value:     "line1\nline2 \"quoted\"",
				},
			},
			format: CSVFormat{Separator: ',', Quote: '"', QuoteAll: true, LineTerminator: "\n"},
			want: "\"Partition\",\"Offset\",\"Timestamp\",\"Key\",\"Value\",\"Headers\"\n" +
				"\"0\",\"1\",\"\",\"key,with,comma\",\"line1\nline2 \"\"quoted\"\"\",\"\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteMessagesCSV(&buf, tt.msgs, tt.format)
			require.NoError(t, err)
			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestDefaultCSVFormat(t *testing.T) {
	f := DefaultCSVFormat()
	assert.Equal(t, ',', f.Separator)
	assert.Equal(t, '"', f.Quote)
	assert.False(t, f.QuoteAll)
	assert.Equal(t, "\n", f.LineTerminator)
}
