package topic

// Performance tests for topic page

import (
	"sync"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/evertras/bubble-table/table"
)

// TestMessageBufferLimit verifies that the message buffer limits messages
func TestMessageBufferLimit(t *testing.T) {
	tests := []struct {
		name            string
		maxMessages     int
		messagesToAdd   int
		expectedMaxSize int
	}{
		{"Small buffer", 50, 100, 50},
		{"Medium buffer", 100, 500, 100},
		{"Large buffer", 200, 1000, 200},
		{"Exact fit", 100, 100, 100},
		{"Under limit", 100, 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModel()
			model.maxMessages = tt.maxMessages

			// Add messages
			for i := int64(0); i < int64(tt.messagesToAdd); i++ {
				msg := api.Message{
					Offset:    i,
					Partition: 0,
					Key:       "key",
					Value:     "value",
				}
				model.AddMessage(msg)
			}

			// Verify buffer limit
			if len(model.messages) > tt.expectedMaxSize {
				t.Errorf("Message buffer exceeded limit: got %d, want <= %d", len(model.messages), tt.expectedMaxSize)
			}

			// Verify oldest messages are removed (FIFO)
			if tt.messagesToAdd > tt.maxMessages {
				firstOffset := model.messages[0].Offset
				expectedFirstOffset := int64(tt.messagesToAdd - tt.maxMessages)
				if firstOffset != expectedFirstOffset {
					t.Errorf("Wrong messages retained: first offset %d, want %d", firstOffset, expectedFirstOffset)
				}
			}
		})
	}
}

// TestPagination verifies pagination works correctly
func TestPagination(t *testing.T) {
	model := createTestModel()
	model.pagination.SetPerPage(20)

	// Add 100 messages
	for i := int64(0); i < 100; i++ {
		msg := api.Message{
			Offset:    i,
			Partition: 0,
			Key:       "key",
			Value:     "value",
		}
		model.AddMessage(msg)
	}

	// Should have 5 pages (100 / 20)
	if model.pagination.TotalPages != 5 {
		t.Errorf("Expected 5 pages, got %d", model.pagination.TotalPages)
	}

	// Page 0 should show messages 80-99 (newest first)
	model.pagination.Page = 0
	visible := model.pagination.GetVisibleMessages(model.filteredMessages)
	if len(visible) != 20 {
		t.Errorf("Expected 20 messages on page, got %d", len(visible))
	}

	// First message on page 0 should be offset 80
	if visible[0].Offset != 80 {
		t.Errorf("Expected first message offset 80, got %d", visible[0].Offset)
	}

	// Last message on page 0 should be offset 99
	if visible[19].Offset != 99 {
		t.Errorf("Expected last message offset 99, got %d", visible[19].Offset)
	}

	// Navigate to last page (page 4)
	model.pagination.LastPage()
	visible = model.pagination.GetVisibleMessages(model.filteredMessages)
	// Should show messages 0-19
	if visible[0].Offset != 0 {
		t.Errorf("Expected first message offset 0 on last page, got %d", visible[0].Offset)
	}
}

// TestPaginationNavigation verifies page navigation works
func TestPaginationNavigation(t *testing.T) {
	model := createTestModel()
	model.pagination.SetPerPage(20)

	// Add 100 messages
	for i := int64(0); i < 100; i++ {
		msg := api.Message{
			Offset:    i,
			Partition: 0,
			Key:       "key",
			Value:     "value",
		}
		model.AddMessage(msg)
	}

	// Start on page 0
	if !model.pagination.OnFirstPage() {
		t.Error("Should start on first page")
	}

	// Navigate to next page
	model.pagination.NextPage()
	if model.pagination.Page != 1 {
		t.Errorf("Expected page 1, got %d", model.pagination.Page)
	}

	// Navigate to last page
	model.pagination.LastPage()
	if !model.pagination.OnLastPage() {
		t.Error("Should be on last page")
	}

	// Navigate to previous page
	model.pagination.PrevPage()
	if model.pagination.Page != 3 {
		t.Errorf("Expected page 3, got %d", model.pagination.Page)
	}

	// Navigate to first page
	model.pagination.FirstPage()
	if !model.pagination.OnFirstPage() {
		t.Error("Should be on first page")
	}
}

// TestMessageBatching verifies messages are batched for performance
func TestMessageBatching(t *testing.T) {
	model := createTestModel()

	var wg sync.WaitGroup

	// Add messages
	for i := int64(0); i < 50; i++ {
		wg.Add(1)
		msg := api.Message{
			Offset:    i,
			Partition: 0,
			Key:       "key",
			Value:     "value",
		}

		go func(m api.Message) {
			defer wg.Done()
			model.AddMessage(m)
		}(msg)
	}

	wg.Wait()

	// Verify messages were added
	if len(model.messages) == 0 {
		t.Error("No messages added")
	}
}

// TestWidthCaching verifies column widths are cached
func TestWidthCaching(t *testing.T) {
	model := createTestModel()

	// Initialize width cache
	model.initWidthCache()

	// Create mock columns using table.NewColumn
	tableColumns := []table.Column{
		table.NewColumn("key", "Key", 20),
		table.NewColumn("value", "Value", 40),
	}

	// First call should calculate and cache
	width1 := model.getCachedWidth("Key", tableColumns)

	// Second call should use cache (faster)
	width2 := model.getCachedWidth("Key", tableColumns)

	if width1 != width2 {
		t.Errorf("Width cache inconsistent: %d vs %d", width1, width2)
	}

	// Different column should have different cache entry
	width3 := model.getCachedWidth("Value", tableColumns)
	if width3 == width1 {
		t.Error("Different columns should have different widths")
	}
}

// TestThrottledUpdates verifies updates are throttled
func TestThrottledUpdates(t *testing.T) {
	model := createTestModel()

	startTime := time.Now()
	updateCount := 0

	// Simulate rapid updates
	for i := 0; i < 20; i++ {
		if model.shouldUpdate() {
			updateCount++
			model.lastUpdateTime = time.Now()
		}
	}

	elapsed := time.Since(startTime)

	// With 100ms throttle, 20 rapid calls should result in 1-2 updates
	if updateCount > 5 {
		t.Errorf("Too many updates: got %d, expected <= 5 in %v", updateCount, elapsed)
	}
}

// TestPerformanceWithLargeDataset verifies performance with many messages
func TestPerformanceWithLargeDataset(t *testing.T) {
	model := createTestModel()

	// Add 1000 messages
	startTime := time.Now()
	for i := int64(0); i < 1000; i++ {
		msg := api.Message{
			Offset:    i,
			Partition: int32(i % 3),
			Key:       "key",
			Value:     "value",
		}
		model.AddMessage(msg)
	}
	addElapsed := time.Since(startTime)

	// Adding 1000 messages should be fast (< 1 second)
	if addElapsed > time.Second {
		t.Errorf("Adding messages too slow: %v", addElapsed)
	}

	// Pagination should be set correctly
	if model.pagination.TotalPages == 0 {
		t.Error("Pagination not initialized")
	}

	// Getting visible messages should be fast
	startTime = time.Now()
	visibleMsgs := model.pagination.GetVisibleMessages(model.filteredMessages)
	renderElapsed := time.Since(startTime)

	if renderElapsed > 100*time.Millisecond {
		t.Errorf("Getting visible messages too slow: %v", renderElapsed)
	}

	if len(visibleMsgs) > model.pagination.PerPage {
		t.Errorf("Too many messages returned: %d > %d", len(visibleMsgs), model.pagination.PerPage)
	}
}

// TestConcurrentMessageAdds verifies thread safety
func TestConcurrentMessageAdds(t *testing.T) {
	model := createTestModel()
	model.maxMessages = 500

	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := 100

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(base int64) {
			defer wg.Done()
			for i := int64(0); i < int64(messagesPerGoroutine); i++ {
				msg := api.Message{
					Offset:    base*int64(messagesPerGoroutine) + i,
					Partition: 0,
					Key:       "key",
					Value:     "value",
				}
				model.AddMessage(msg)
			}
		}(int64(g))
	}

	wg.Wait()

	// Should not exceed maxMessages
	if len(model.messages) > model.maxMessages {
		t.Errorf("Buffer overflow: got %d, want <= %d", len(model.messages), model.maxMessages)
	}
}

// BenchmarkMessageAdd benchmarks message addition performance
func BenchmarkMessageAdd(b *testing.B) {
	model := createTestModel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := api.Message{
			Offset:    int64(i),
			Partition: 0,
			Key:       "key",
			Value:     "value",
		}
		model.AddMessage(msg)
	}
}

// BenchmarkPagination benchmarks pagination
func BenchmarkPagination(b *testing.B) {
	model := createTestModel()

	// Pre-populate with messages
	for i := int64(0); i < 1000; i++ {
		msg := api.Message{
			Offset:    i,
			Partition: 0,
			Key:       "key",
			Value:     "value",
		}
		model.AddMessage(msg)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.pagination.GetVisibleMessages(model.filteredMessages)
	}
}

// BenchmarkWidthCache benchmarks width caching
func BenchmarkWidthCache(b *testing.B) {
	model := createTestModel()
	model.initWidthCache()

	columns := []table.Column{
		table.NewColumn("key", "Key", 20),
		table.NewColumn("value", "Value", 40),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.getCachedWidth("Key", columns)
	}
}

// Helper function to create test model
func createTestModel() *Model {
	return &Model{
		messages:         make([]api.Message, 0),
		filteredMessages: make([]api.Message, 0),
		consumedMessages: make(map[string]api.Message),
		maxMessages:      100,
		pagination:       NewPaginationModel(),
		updateThrottle:   100 * time.Millisecond,
		widthCache:       make(map[string]map[int]int),
	}
}
