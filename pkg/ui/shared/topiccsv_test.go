package shared_test

import (
	"bytes"
	"encoding/csv"
	"testing"

	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteTopicCSV_Golden(t *testing.T) {
	rows := []shared.TopicCSVRow{
		{Name: "b-topic", Partitions: 2, ReplicationFactor: 1, MessageCount: 100, OutOfSync: 1, Size: 2048, Internal: false},
		{Name: "a-topic", Partitions: 3, ReplicationFactor: 2, MessageCount: -1, OutOfSync: 0, Size: -1, Internal: true},
	}

	var buf bytes.Buffer
	require.NoError(t, shared.WriteTopicCSV(&buf, rows))

	records, err := csv.NewReader(&buf).ReadAll()
	require.NoError(t, err)

	want := [][]string{
		{"Name", "Partitions", "Replication Factor", "Message Count", "Out Of Sync", "Size", "Internal"},
		// Sorted by name: a-topic first. Unknown count/size render "N/A".
		{"a-topic", "3", "2", "N/A", "0", "N/A", "true"},
		{"b-topic", "2", "1", "100", "1", "2.00 KB", "false"},
	}
	assert.Equal(t, want, records)
}
