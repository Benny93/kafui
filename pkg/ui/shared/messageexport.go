package shared

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Benny93/kafui/pkg/api"
)

// DefaultExportPath returns "./<topic>-<partition>-<offset>.json".
func DefaultExportPath(topic string, m api.Message) string {
	return fmt.Sprintf("./%s-%d-%d.json", topic, m.Partition, m.Offset)
}

// exportedMessage is the JSON shape written by ExportMessageJSON.
type exportedMessage struct {
	Key       string            `json:"key"`
	Value     string            `json:"value"`
	Offset    int64             `json:"offset"`
	Partition int32             `json:"partition"`
	Headers   map[string]string `json:"headers"`
	Timestamp string            `json:"timestamp"`
}

// ExportMessageJSON writes the message to path as pretty JSON containing
// key, value, offset, partition, headers (as an object/map name->value), and
// timestamp (RFC3339). Creates parent dirs as needed.
func ExportMessageJSON(path string, topic string, m api.Message) error {
	headers := make(map[string]string, len(m.Headers))
	for _, h := range m.Headers {
		headers[h.Key] = h.Value
	}
	ts := ""
	if !m.Timestamp.IsZero() {
		ts = m.Timestamp.Format(time.RFC3339)
	}
	out := exportedMessage{
		Key:       m.Key,
		Value:     m.Value,
		Offset:    m.Offset,
		Partition: m.Partition,
		Headers:   headers,
		Timestamp: ts,
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0644)
}
