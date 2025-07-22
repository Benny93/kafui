package kafui

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/rivo/tview"
)

// TestNewResouceTopic tests the ResouceTopic constructor
func TestNewResouceTopic(t *testing.T) {
	mockDataSource := &MockKafkaDataSource{}
	onError := func(err error) {}
	recoverFunc := func() {}

	resource := NewResouceTopic(mockDataSource, onError, recoverFunc)

	if resource == nil {
		t.Fatal("NewResouceTopic() returned nil")
	}

	if resource.dataSource == nil {
		t.Error("dataSource not set")
	}

	if resource.onError == nil {
		t.Error("onError callback not set")
	}

	if resource.recoverFunc == nil {
		t.Error("recoverFunc not set")
	}
}

// TestResouceTopic_GetName tests the GetName method
func TestResouceTopic_GetName(t *testing.T) {
	resource := &ResouceTopic{}
	
	result := resource.GetName()
	expected := "Topic"

	if result != expected {
		t.Errorf("GetName() = %v, want %v", result, expected)
	}
}

// TestResouceTopic_FetchTopics tests the FetchTopics method
func TestResouceTopic_FetchTopics(t *testing.T) {
	tests := []struct {
		name           string
		dataSource     api.KafkaDataSource
		expectedResult map[string]api.Topic
		expectError    bool
	}{
		{
			name:       "successful fetch",
			dataSource: &MockKafkaDataSource{},
			expectedResult: map[string]api.Topic{
				"test-topic": {
					NumPartitions:     3,
					ReplicationFactor: 2,
					MessageCount:      100,
				},
			},
			expectError: false,
		},
		{
			name:           "error handling",
			dataSource:     &ErrorKafkaDataSource{},
			expectedResult: map[string]api.Topic{},
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorCalled := false
			onError := func(err error) {
				errorCalled = true
			}

			resource := &ResouceTopic{
				onError: onError,
			}

			result := resource.FetchTopics(tt.dataSource)

			if !reflect.DeepEqual(result, tt.expectedResult) {
				t.Errorf("FetchTopics() = %v, want %v", result, tt.expectedResult)
			}

			if tt.expectError && !errorCalled {
				t.Error("Expected error callback to be called")
			}

			if !tt.expectError && errorCalled {
				t.Error("Error callback should not have been called")
			}
		})
	}
}

// TestResouceTopic_ShowTopicsInTable tests the table display functionality
func TestResouceTopic_ShowTopicsInTable(t *testing.T) {
	tests := []struct {
		name           string
		topics         map[string]api.Topic
		search         string
		expectedRows   int
		expectedCols   int
	}{
		{
			name: "display all topics",
			topics: map[string]api.Topic{
				"user-events": {
					NumPartitions:     5,
					ReplicationFactor: 3,
					MessageCount:      1000,
				},
				"order-events": {
					NumPartitions:     3,
					ReplicationFactor: 2,
					MessageCount:      500,
				},
				"payment-events": {
					NumPartitions:     1,
					ReplicationFactor: 1,
					MessageCount:      100,
				},
			},
			search:       "",
			expectedRows: 4, // 3 data rows + 1 header
			expectedCols: 3, // Topic, Num Partitions, Replication Factor
		},
		{
			name: "search filter",
			topics: map[string]api.Topic{
				"prod-events":    {NumPartitions: 5, ReplicationFactor: 3},
				"staging-events": {NumPartitions: 3, ReplicationFactor: 2},
				"dev-events":     {NumPartitions: 1, ReplicationFactor: 1},
			},
			search:       "prod",
			expectedRows: 2, // 1 data row + 1 header
			expectedCols: 3,
		},
		{
			name: "case insensitive search",
			topics: map[string]api.Topic{
				"PROD-EVENTS": {NumPartitions: 5, ReplicationFactor: 3},
				"dev-events":  {NumPartitions: 1, ReplicationFactor: 1},
			},
			search:       "prod",
			expectedRows: 2, // 1 data row + 1 header
			expectedCols: 3,
		},
		{
			name:         "empty topics",
			topics:       map[string]api.Topic{},
			search:       "",
			expectedRows: 1, // header only
			expectedCols: 3,
		},
		{
			name: "no search matches",
			topics: map[string]api.Topic{
				"topic1": {NumPartitions: 1, ReplicationFactor: 1},
				"topic2": {NumPartitions: 2, ReplicationFactor: 2},
			},
			search:       "nonexistent",
			expectedRows: 1, // header only
			expectedCols: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := tview.NewTable()
			resource := &ResouceTopic{}

			resource.ShowTopicsInTable(table, tt.topics, tt.search)

			// Check row count
			if table.GetRowCount() != tt.expectedRows {
				t.Errorf("GetRowCount() = %v, want %v", table.GetRowCount(), tt.expectedRows)
			}

			// Check column count
			if table.GetRowCount() > 0 && table.GetColumnCount() != tt.expectedCols {
				t.Errorf("GetColumnCount() = %v, want %v", table.GetColumnCount(), tt.expectedCols)
			}

			// Check headers
			if table.GetRowCount() > 0 {
				expectedHeaders := []string{"Topic", "Num Partitions", "Replication Factor"}
				for i, expectedHeader := range expectedHeaders {
					headerCell := table.GetCell(0, i)
					if headerCell.Text != expectedHeader {
						t.Errorf("Header[%d] = %v, want %v", i, headerCell.Text, expectedHeader)
					}
				}
			}

			// Check data rows if any
			if table.GetRowCount() > 1 {
				// Verify first data row has correct structure
				for col := 0; col < tt.expectedCols; col++ {
					cell := table.GetCell(1, col)
					if cell == nil {
						t.Errorf("Cell[1][%d] is nil", col)
					}
				}
			}
		})
	}
}

// TestResouceTopic_UpdateTable tests the UpdateTable method
func TestResouceTopic_UpdateTable(t *testing.T) {
	table := tview.NewTable()
	mockDataSource := &MockKafkaDataSource{}
	
	resource := &ResouceTopic{
		LastFetchedTopics: map[string]api.Topic{
			"test-topic": {
				NumPartitions:     3,
				ReplicationFactor: 2,
				MessageCount:      100,
			},
		},
	}

	// Test that UpdateTable calls ShowTopicsInTable
	resource.UpdateTable(table, mockDataSource, "test")

	// Verify table was updated (should have header + 1 data row)
	if table.GetRowCount() != 2 {
		t.Errorf("Expected 2 rows after UpdateTable, got %d", table.GetRowCount())
	}
}

// TestResouceTopic_StartStopFetching tests the lifecycle methods
func TestResouceTopic_StartStopFetching(t *testing.T) {
	mockDataSource := &MockKafkaDataSource{}
	onError := func(err error) {}
	recoverFunc := func() {}

	resource := NewResouceTopic(mockDataSource, onError, recoverFunc)

	// Test StartFetchingData
	resource.StartFetchingData()

	if resource.cancelFetch == nil {
		t.Error("cancelFetch should be set after StartFetchingData")
	}

	// Give some time for the goroutine to start
	time.Sleep(10 * time.Millisecond)

	// Test StopFetching
	resource.StopFetching()

	// Give some time for the goroutine to stop
	time.Sleep(10 * time.Millisecond)
}

// TestResouceTopic_UpdateTableDataRoutine tests the background fetching routine
func TestResouceTopic_UpdateTableDataRoutine(t *testing.T) {
	mockDataSource := &MockKafkaDataSource{}
	recoverFunc := func() {}

	resource := &ResouceTopic{
		dataSource:  mockDataSource,
		recoverFunc: recoverFunc,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start the routine
	resource.UpdateTableDataRoutine(ctx, mockDataSource)

	// Give some time for the routine to run
	time.Sleep(50 * time.Millisecond)

	// Cancel the context
	cancel()

	// Give some time for cleanup
	time.Sleep(50 * time.Millisecond)

	// Check that topics were fetched
	if resource.LastFetchedTopics == nil {
		t.Error("LastFetchedTopics should be populated")
	}
}

// TestResouceTopic_TopicConfiguration tests various topic configurations
func TestResouceTopic_TopicConfiguration(t *testing.T) {
	topics := map[string]api.Topic{
		"high-throughput": {
			NumPartitions:     10,
			ReplicationFactor: 3,
			MessageCount:      1000000,
			ConfigEntries: map[string]*string{
				"cleanup.policy": stringPtr("delete"),
				"retention.ms":   stringPtr("604800000"),
			},
		},
		"low-latency": {
			NumPartitions:     1,
			ReplicationFactor: 1,
			MessageCount:      100,
			ConfigEntries: map[string]*string{
				"cleanup.policy": stringPtr("compact"),
			},
		},
		"balanced": {
			NumPartitions:     5,
			ReplicationFactor: 2,
			MessageCount:      50000,
		},
	}

	table := tview.NewTable()
	resource := &ResouceTopic{}
	resource.ShowTopicsInTable(table, topics, "")

	// Should have header + all topics
	expectedRows := len(topics) + 1
	if table.GetRowCount() != expectedRows {
		t.Errorf("Expected %d rows, got %d", expectedRows, table.GetRowCount())
	}

	// Verify topic data is displayed correctly
	for i := 1; i <= len(topics); i++ {
		topicCell := table.GetCell(i, 0) // Topic name column
		partitionsCell := table.GetCell(i, 1) // Partitions column
		replicationCell := table.GetCell(i, 2) // Replication factor column
		
		if topicCell == nil || partitionsCell == nil || replicationCell == nil {
			t.Errorf("Missing cells in row %d", i)
			continue
		}
		
		// Verify cells have content
		if topicCell.Text == "" {
			t.Errorf("Empty topic name in row %d", i)
		}
		if partitionsCell.Text == "" {
			t.Errorf("Empty partitions in row %d", i)
		}
		if replicationCell.Text == "" {
			t.Errorf("Empty replication factor in row %d", i)
		}
	}
}

// TestResouceTopic_EmptyTopicHandling tests edge cases with empty or nil topics
func TestResouceTopic_EmptyTopicHandling(t *testing.T) {
	tests := []struct {
		name   string
		topics map[string]api.Topic
	}{
		{
			name:   "nil topics map",
			topics: nil,
		},
		{
			name:   "empty topics map",
			topics: map[string]api.Topic{},
		},
		{
			name: "topics with zero values",
			topics: map[string]api.Topic{
				"empty-topic": {
					NumPartitions:     0,
					ReplicationFactor: 0,
					MessageCount:      0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := tview.NewTable()
			resource := &ResouceTopic{}

			// Should not panic
			resource.ShowTopicsInTable(table, tt.topics, "")

			// Should at least have header row
			if table.GetRowCount() < 1 {
				t.Error("Table should have at least header row")
			}
		})
	}
}

// Helper function for creating string pointers (defined in api_test.go but needed here too)
func stringPtr(s string) *string {
	return &s
}

// Benchmark tests for performance
func BenchmarkResouceTopic_FetchTopics(b *testing.B) {
	mockDataSource := &MockKafkaDataSource{}
	resource := &ResouceTopic{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resource.FetchTopics(mockDataSource)
	}
}

func BenchmarkResouceTopic_ShowTopicsInTable(b *testing.B) {
	topics := map[string]api.Topic{
		"topic1": {NumPartitions: 3, ReplicationFactor: 2, MessageCount: 1000},
		"topic2": {NumPartitions: 5, ReplicationFactor: 3, MessageCount: 2000},
		"topic3": {NumPartitions: 1, ReplicationFactor: 1, MessageCount: 100},
		"topic4": {NumPartitions: 10, ReplicationFactor: 3, MessageCount: 5000},
	}
	resource := &ResouceTopic{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		table := tview.NewTable()
		resource.ShowTopicsInTable(table, topics, "")
	}
}