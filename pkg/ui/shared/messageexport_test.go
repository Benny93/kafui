package shared

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultExportPath(t *testing.T) {
	tests := []struct {
		name  string
		topic string
		msg   api.Message
		want  string
	}{
		{
			name:  "basic",
			topic: "orders",
			msg:   api.Message{Partition: 2, Offset: 42},
			want:  "./orders-2-42.json",
		},
		{
			name:  "zero values",
			topic: "events",
			msg:   api.Message{Partition: 0, Offset: 0},
			want:  "./events-0-0.json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DefaultExportPath(tt.topic, tt.msg))
		})
	}
}

func TestExportMessageJSON(t *testing.T) {
	ts := time.Date(2026, 7, 4, 12, 30, 0, 0, time.UTC)
	msg := api.Message{
		Key:       "my-key",
		Value:     "my-value",
		Offset:    99,
		Partition: 3,
		Timestamp: ts,
		Headers: []api.MessageHeader{
			{Key: "trace", Value: "abc"},
			{Key: "source", Value: "svc"},
		},
	}

	// Include a nested dir to verify parent dirs are created.
	path := filepath.Join(t.TempDir(), "sub", "out.json")
	err := ExportMessageJSON(path, "orders", msg)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var got struct {
		Key       string            `json:"key"`
		Value     string            `json:"value"`
		Offset    int64             `json:"offset"`
		Partition int32             `json:"partition"`
		Headers   map[string]string `json:"headers"`
		Timestamp string            `json:"timestamp"`
	}
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, "my-key", got.Key)
	assert.Equal(t, "my-value", got.Value)
	assert.Equal(t, int64(99), got.Offset)
	assert.Equal(t, int32(3), got.Partition)
	assert.Equal(t, "2026-07-04T12:30:00Z", got.Timestamp)
	assert.Equal(t, map[string]string{"trace": "abc", "source": "svc"}, got.Headers)
}

func TestExportMessageJSON_ZeroTimestamp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "z.json")
	err := ExportMessageJSON(path, "t", api.Message{Partition: 0, Offset: 1})
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, "", got["timestamp"])
}
