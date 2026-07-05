package shared_test

import (
	"bytes"
	"encoding/csv"
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteBrokerCSV_Golden asserts the CSV writer produces the expected,
// stats-enriched rows from the deterministic mock datasource. Records are parsed
// back so the assertion is independent of CSV quoting details.
func TestWriteBrokerCSV_Golden(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")

	brokers, err := ds.GetBrokers()
	require.NoError(t, err)
	stats, _, err := ds.GetBrokerStats()
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, shared.WriteBrokerCSV(&buf, brokers, stats))

	records, err := csv.NewReader(&buf).ReadAll()
	require.NoError(t, err)

	want := [][]string{
		{"ID", "Host", "Port", "Rack", "Disk Usage", "Leaders", "Replicas", "ISR", "Leader Skew", "Replica Skew"},
		{"1 (Active)", "kafka-1.mock", "9092", "rack-a", "896.00 MB, 3 segment(s)", "10", "30", "30/30", "5.00%", "3.20%"},
		{"2", "kafka-2.mock", "9092", "rack-a", "384.00 MB, 1 segment(s)", "9", "28", "25/28", "-8.50%", "22.50%"},
		{"3", "kafka-3.mock", "9092", "rack-b", "N/A", "11", "32", "32/32", "12.00%", "-10.30%"},
	}
	assert.Equal(t, want, records)
}
