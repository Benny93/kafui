package kafui

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/rivo/tview"
)

// TestNewResourceContext tests the ResourceContext constructor
func TestNewResourceContext(t *testing.T) {
	mockDataSource := &MockKafkaDataSource{}
	onError := func(err error) {}
	recoverFunc := func() {}

	resource := NewResourceContext(mockDataSource, onError, recoverFunc)

	if resource == nil {
		t.Fatal("NewResourceContext() returned nil")
	}

	if resource.Name != "Context" {
		t.Errorf("Name = %v, want %v", resource.Name, "Context")
	}

	// Note: Can't directly compare interface values, so we test functionality instead
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

// TestResourceContext_GetName tests the GetName method
func TestResourceContext_GetName(t *testing.T) {
	resource := &ResourceContext{Name: "TestContext"}
	
	result := resource.GetName()
	expected := "TestContext"

	if result != expected {
		t.Errorf("GetName() = %v, want %v", result, expected)
	}
}

// TestResourceContext_FetchContexts tests the FetchContexts method
func TestResourceContext_FetchContexts(t *testing.T) {
	tests := []struct {
		name           string
		dataSource     api.KafkaDataSource
		expectedResult []string
		expectError    bool
	}{
		{
			name:           "successful fetch",
			dataSource:     &MockKafkaDataSource{},
			expectedResult: []string{"context1", "context2"},
			expectError:    false,
		},
		{
			name:           "error handling",
			dataSource:     &ErrorKafkaDataSource{},
			expectedResult: []string{},
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorCalled := false
			onError := func(err error) {
				errorCalled = true
			}

			resource := &ResourceContext{
				onError: onError,
			}

			result := resource.FetchContexts(tt.dataSource)

			if !reflect.DeepEqual(result, tt.expectedResult) {
				t.Errorf("FetchContexts() = %v, want %v", result, tt.expectedResult)
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

// TestResourceContext_ShowContextsInTable tests the table display functionality
func TestResourceContext_ShowContextsInTable(t *testing.T) {
	tests := []struct {
		name           string
		contexts       map[string]string
		search         string
		expectedRows   int
		expectedHeader string
	}{
		{
			name: "display all contexts",
			contexts: map[string]string{
				"prod":    "prod",
				"staging": "staging",
				"dev":     "dev",
			},
			search:         "",
			expectedRows:   4, // 3 data rows + 1 header
			expectedHeader: "Context",
		},
		{
			name: "search filter",
			contexts: map[string]string{
				"prod":    "prod",
				"staging": "staging",
				"dev":     "dev",
			},
			search:         "prod",
			expectedRows:   2, // 1 data row + 1 header
			expectedHeader: "Context",
		},
		{
			name: "case insensitive search",
			contexts: map[string]string{
				"PROD":    "PROD",
				"staging": "staging",
			},
			search:         "prod",
			expectedRows:   2, // 1 data row + 1 header
			expectedHeader: "Context",
		},
		{
			name:           "empty contexts",
			contexts:       map[string]string{},
			search:         "",
			expectedRows:   1, // header only
			expectedHeader: "Context",
		},
		{
			name: "no search matches",
			contexts: map[string]string{
				"prod": "prod",
				"dev":  "dev",
			},
			search:         "nonexistent",
			expectedRows:   1, // header only
			expectedHeader: "Context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := tview.NewTable()
			resource := ResourceContext{}

			resource.ShowContextsInTable(table, tt.contexts, tt.search)

			// Check row count
			if table.GetRowCount() != tt.expectedRows {
				t.Errorf("GetRowCount() = %v, want %v", table.GetRowCount(), tt.expectedRows)
			}

			// Check header
			if table.GetRowCount() > 0 {
				headerCell := table.GetCell(0, 0)
				if headerCell.Text != tt.expectedHeader {
					t.Errorf("Header text = %v, want %v", headerCell.Text, tt.expectedHeader)
				}
			}
		})
	}
}

// TestResourceContext_UpdateTable tests the UpdateTable method
func TestResourceContext_UpdateTable(t *testing.T) {
	table := tview.NewTable()
	mockDataSource := &MockKafkaDataSource{}
	
	resource := &ResourceContext{
		FetchedContexts: map[string]string{
			"test-context": "test-context",
		},
	}

	// Test that UpdateTable calls ShowContextsInTable
	resource.UpdateTable(table, mockDataSource, "test")

	// Verify table was updated (should have header + 1 data row)
	if table.GetRowCount() != 2 {
		t.Errorf("Expected 2 rows after UpdateTable, got %d", table.GetRowCount())
	}
}

// TestResourceContext_StartStopFetching tests the lifecycle methods
func TestResourceContext_StartStopFetching(t *testing.T) {
	mockDataSource := &MockKafkaDataSource{}
	onError := func(err error) {}
	recoverFunc := func() {}

	resource := NewResourceContext(mockDataSource, onError, recoverFunc)

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

// TestResourceContext_FetchContextRoutine tests the background fetching routine
func TestResourceContext_FetchContextRoutine(t *testing.T) {
	mockDataSource := &MockKafkaDataSource{}
	recoverFunc := func() {
		// Recovery function for testing
	}

	resource := &ResourceContext{
		dataSource:  mockDataSource,
		recoverFunc: recoverFunc,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start the routine
	resource.FetchContextRoutine(ctx, mockDataSource)

	// Give some time for the routine to run
	time.Sleep(50 * time.Millisecond)

	// Cancel the context
	cancel()

	// Give some time for cleanup
	time.Sleep(50 * time.Millisecond)

	// Check that contexts were fetched
	if resource.FetchedContexts == nil {
		t.Error("FetchedContexts should be populated")
	}
}

// ErrorKafkaDataSource for testing error scenarios
type ErrorKafkaDataSource struct{}

func (e *ErrorKafkaDataSource) Init(cfgOption string) {}

func (e *ErrorKafkaDataSource) GetTopics() (map[string]api.Topic, error) {
	return nil, &TestError{message: "failed to get topics"}
}

func (e *ErrorKafkaDataSource) GetContexts() ([]string, error) {
	return nil, &TestError{message: "failed to get contexts"}
}

func (e *ErrorKafkaDataSource) GetContext() string {
	return ""
}

func (e *ErrorKafkaDataSource) SetContext(contextName string) error {
	return &TestError{message: "failed to set context"}
}

func (e *ErrorKafkaDataSource) GetConsumerGroups() ([]api.ConsumerGroup, error) {
	return nil, &TestError{message: "failed to get consumer groups"}
}

func (e *ErrorKafkaDataSource) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) error {
	return &TestError{message: "failed to consume topic"}
}

// TestError implements error interface for testing
type TestError struct {
	message string
}

func (e *TestError) Error() string {
	return e.message
}

// Benchmark tests for performance
func BenchmarkResourceContext_FetchContexts(b *testing.B) {
	mockDataSource := &MockKafkaDataSource{}
	resource := &ResourceContext{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resource.FetchContexts(mockDataSource)
	}
}

func BenchmarkResourceContext_ShowContextsInTable(b *testing.B) {
	contexts := map[string]string{
		"prod":    "prod",
		"staging": "staging",
		"dev":     "dev",
		"test":    "test",
	}
	resource := ResourceContext{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		table := tview.NewTable()
		resource.ShowContextsInTable(table, contexts, "")
	}
}