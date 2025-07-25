package kafui

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/rivo/tview"
)

// TestNewResourceGroup tests the ResourceGroup constructor
func TestNewResourceGroup(t *testing.T) {
	mockDataSource := &MockKafkaDataSource{}
	onError := func(err error) {}
	recoverFunc := func() {}

	resource := NewResourceGroup(onError, mockDataSource, recoverFunc)

	if resource == nil {
		t.Fatal("NewResourceGroup() returned nil")
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

	if resource.FetchedConsumerGroups == nil {
		t.Error("FetchedConsumerGroups should be initialized")
	}
}

// TestResourceGroup_GetName tests the GetName method
func TestResourceGroup_GetName(t *testing.T) {
	resource := &ResourceGroup{}
	
	result := resource.GetName()
	expected := "ConsumerGroup"

	if result != expected {
		t.Errorf("GetName() = %v, want %v", result, expected)
	}
}

// TestResourceGroup_FetchConsumerGroups tests the FetchConsumerGroups method
func TestResourceGroup_FetchConsumerGroups(t *testing.T) {
	tests := []struct {
		name           string
		dataSource     api.KafkaDataSource
		expectedResult []api.ConsumerGroup
		expectError    bool
	}{
		{
			name:       "successful fetch",
			dataSource: &MockKafkaDataSource{},
			expectedResult: []api.ConsumerGroup{
				{Name: "group1", State: "Stable", Consumers: 3},
				{Name: "group2", State: "Rebalancing", Consumers: 2},
			},
			expectError: false,
		},
		{
			name:           "error handling",
			dataSource:     &ErrorKafkaDataSource{},
			expectedResult: []api.ConsumerGroup{},
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorCalled := false
			onError := func(err error) {
				errorCalled = true
			}

			resource := &ResourceGroup{
				onError: onError,
			}

			result := resource.FetchConsumerGroups(tt.dataSource)

			if !reflect.DeepEqual(result, tt.expectedResult) {
				t.Errorf("FetchConsumerGroups() = %v, want %v", result, tt.expectedResult)
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

// TestResourceGroup_ShowConsumerGroups tests the table display functionality
func TestResourceGroup_ShowConsumerGroups(t *testing.T) {
	tests := []struct {
		name           string
		groups         map[string]api.ConsumerGroup
		search         string
		expectedRows   int
		expectedCols   int
	}{
		{
			name: "display all groups",
			groups: map[string]api.ConsumerGroup{
				"group1": {Name: "group1", State: "Stable", Consumers: 3},
				"group2": {Name: "group2", State: "Rebalancing", Consumers: 2},
				"group3": {Name: "group3", State: "Dead", Consumers: 0},
			},
			search:       "",
			expectedRows: 4, // 3 data rows + 1 header
			expectedCols: 3, // Name, State, Consumers
		},
		{
			name: "search filter",
			groups: map[string]api.ConsumerGroup{
				"prod-group":    {Name: "prod-group", State: "Stable", Consumers: 5},
				"staging-group": {Name: "staging-group", State: "Stable", Consumers: 2},
				"dev-group":     {Name: "dev-group", State: "Dead", Consumers: 0},
			},
			search:       "prod",
			expectedRows: 2, // 1 data row + 1 header
			expectedCols: 3,
		},
		{
			name: "case insensitive search",
			groups: map[string]api.ConsumerGroup{
				"PROD-GROUP": {Name: "PROD-GROUP", State: "Stable", Consumers: 3},
				"dev-group":  {Name: "dev-group", State: "Dead", Consumers: 0},
			},
			search:       "prod",
			expectedRows: 2, // 1 data row + 1 header
			expectedCols: 3,
		},
		{
			name:         "empty groups",
			groups:       map[string]api.ConsumerGroup{},
			search:       "",
			expectedRows: 1, // header only
			expectedCols: 3,
		},
		{
			name: "no search matches",
			groups: map[string]api.ConsumerGroup{
				"group1": {Name: "group1", State: "Stable", Consumers: 1},
				"group2": {Name: "group2", State: "Dead", Consumers: 0},
			},
			search:       "nonexistent",
			expectedRows: 1, // header only
			expectedCols: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := tview.NewTable()
			resource := ResourceGroup{}

			resource.ShowConsumerGroups(table, tt.groups, tt.search)

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
				expectedHeaders := []string{"Name", "State", "Consumers"}
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

// TestResourceGroup_UpdateTable tests the UpdateTable method
func TestResourceGroup_UpdateTable(t *testing.T) {
	table := tview.NewTable()
	mockDataSource := &MockKafkaDataSource{}
	
	resource := &ResourceGroup{
		FetchedConsumerGroups: map[string]api.ConsumerGroup{
			"test-group": {Name: "test-group", State: "Stable", Consumers: 2},
		},
	}

	// Test that UpdateTable calls ShowConsumerGroups
	resource.UpdateTable(table, mockDataSource, "test")

	// Verify table was updated (should have header + 1 data row)
	if table.GetRowCount() != 2 {
		t.Errorf("Expected 2 rows after UpdateTable, got %d", table.GetRowCount())
	}
}

// TestResourceGroup_StartStopFetching tests the lifecycle methods
func TestResourceGroup_StartStopFetching(t *testing.T) {
	mockDataSource := &MockKafkaDataSource{}
	onError := func(err error) {}
	recoverFunc := func() {}

	resource := NewResourceGroup(onError, mockDataSource, recoverFunc)

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

// TestResourceGroup_FetchGroupsRoutine tests the background fetching routine
func TestResourceGroup_FetchGroupsRoutine(t *testing.T) {
	mockDataSource := &MockKafkaDataSource{}
	recoverFunc := func() {}

	resource := &ResourceGroup{
		dataSource:            mockDataSource,
		recoverFunc:           recoverFunc,
		FetchedConsumerGroups: make(map[string]api.ConsumerGroup),
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start the routine
	resource.FetchGroupsRoutine(ctx, mockDataSource)

	// Give some time for the routine to run
	time.Sleep(50 * time.Millisecond)

	// Cancel the context
	cancel()

	// Give some time for cleanup
	time.Sleep(50 * time.Millisecond)

	// Check that groups were fetched
	if len(resource.FetchedConsumerGroups) == 0 {
		t.Error("FetchedConsumerGroups should be populated")
	}
}

// TestResourceGroup_ConsumerGroupStates tests various consumer group states
func TestResourceGroup_ConsumerGroupStates(t *testing.T) {
	states := []string{"Stable", "Rebalancing", "Dead", "PreparingRebalance", "CompletingRebalance"}
	
	groups := make(map[string]api.ConsumerGroup)
	for i, state := range states {
		groups[state] = api.ConsumerGroup{
			Name:      state,
			State:     state,
			Consumers: i + 1,
		}
	}

	table := tview.NewTable()
	resource := ResourceGroup{}
	resource.ShowConsumerGroups(table, groups, "")

	// Should have header + all states
	expectedRows := len(states) + 1
	if table.GetRowCount() != expectedRows {
		t.Errorf("Expected %d rows, got %d", expectedRows, table.GetRowCount())
	}

	// Verify each state is displayed
	for i := 1; i <= len(states); i++ {
		stateCell := table.GetCell(i, 1) // State column
		if stateCell == nil {
			t.Errorf("State cell[%d] is nil", i)
			continue
		}
		
		found := false
		for _, expectedState := range states {
			if stateCell.Text == expectedState {
				found = true
				break
			}
		}
		
		if !found {
			t.Errorf("Unexpected state in row %d: %s", i, stateCell.Text)
		}
	}
}

// Benchmark tests for performance
func BenchmarkResourceGroup_FetchConsumerGroups(b *testing.B) {
	mockDataSource := &MockKafkaDataSource{}
	resource := &ResourceGroup{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resource.FetchConsumerGroups(mockDataSource)
	}
}

func BenchmarkResourceGroup_ShowConsumerGroups(b *testing.B) {
	groups := map[string]api.ConsumerGroup{
		"group1": {Name: "group1", State: "Stable", Consumers: 3},
		"group2": {Name: "group2", State: "Rebalancing", Consumers: 2},
		"group3": {Name: "group3", State: "Dead", Consumers: 0},
		"group4": {Name: "group4", State: "Stable", Consumers: 5},
	}
	resource := ResourceGroup{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		table := tview.NewTable()
		resource.ShowConsumerGroups(table, groups, "")
	}
}