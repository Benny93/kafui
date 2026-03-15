package router

import (
	"context"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/stretchr/testify/assert"
)

// mockDataSource implements api.KafkaDataSource for testing
type mockDataSource struct{}

func (m *mockDataSource) GetTopics() (map[string]api.Topic, error) {
	return map[string]api.Topic{
		"test-topic": {
			NumPartitions:     3,
			ReplicationFactor: 1,
			ReplicaAssignment: make(map[int32][]int32),
			ConfigEntries:     make(map[string]*string),
		},
	}, nil
}

func (m *mockDataSource) GetMessages(topicName string, partition int32, offset int64, limit int) ([]api.Message, error) {
	return []api.Message{
		{
			Key:       "test-key",
			Value:     "test-value",
			Offset:    offset,
			Partition: partition,
		},
	}, nil
}

func (m *mockDataSource) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handler api.MessageHandlerFunc, stopCallback func(interface{})) error {
	// Mock implementation - just call handler with test message
	message := api.Message{
		Key:       "test-key",
		Value:     "test-value",
		Offset:    0,
		Partition: 0,
	}
	handler(message)
	return nil
}

func (m *mockDataSource) GetConsumerGroups() ([]api.ConsumerGroup, error) {
	return []api.ConsumerGroup{}, nil
}

func (m *mockDataSource) Init(cfgOption string) {}

func (m *mockDataSource) GetContexts() ([]string, error) {
	return []string{"default"}, nil
}

func (m *mockDataSource) GetContext() string {
	return "default"
}

func (m *mockDataSource) SetContext(contextName string) error {
	return nil
}


func (m *mockDataSource) GetMessageSchemaInfo(keySchemaID, valueSchemaID string) (*api.MessageSchemaInfo, error) {
	return nil, nil
}

// mockResourceItem implements shared.ResourceItem for testing
type mockResourceItem struct {
	id      string
	details map[string]string
}

func (m *mockResourceItem) GetID() string {
	return m.id
}

func (m *mockResourceItem) GetValues() []string {
	return []string{m.id}
}

func (m *mockResourceItem) GetDetails() map[string]string {
	return m.details
}

func TestNewRouter(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	if router == nil {
		t.Fatal("NewRouter returned nil")
	}

	// Note: We can't directly compare interfaces, so we'll skip this check

	if router.currentPage != "main" {
		t.Errorf("Expected initial page to be 'main', got '%s'", router.currentPage)
	}

	if len(router.pages) != 0 {
		t.Errorf("Expected empty pages map, got %d pages", len(router.pages))
	}

	if len(router.history) != 0 {
		t.Errorf("Expected empty history, got %d entries", len(router.history))
	}
}

func TestNavigateTo(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	tests := []struct {
		name     string
		pageID   string
		data     interface{}
		expected string
	}{
		{
			name:     "Navigate to main page",
			pageID:   "main",
			data:     nil,
			expected: "main",
		},
		{
			name:   "Navigate to topic page with data",
			pageID: "topic",
			data: &NavigationData{
				TopicName: "test-topic",
				Topic: api.Topic{
					NumPartitions:     3,
					ReplicationFactor: 1,
				},
			},
			expected: "topic:test-topic",
		},
		{
			name:   "Navigate to message detail page",
			pageID: "message_detail",
			data: &NavigationData{
				TopicName: "test-topic",
				Message: api.Message{
					Partition: 0,
					Offset:    0,
					Key:       "test-key",
					Value:     "test-value",
				},
			},
			expected: "detail:test-topic:0:0",
		},
		{
			name:   "Navigate to resource detail page",
			pageID: "resource_detail",
			data: &NavigationData{
				ResourceItem: &mockResourceItem{
					id:      "test-resource",
					details: map[string]string{"Type": "Topic"},
				},
				ResourceType: "topic",
			},
			expected: "resource_detail:test-resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router.NavigateTo(tt.pageID, tt.data)

			// Router stores page by the pageID passed to NavigateTo
			// The page's GetID() may return a different dynamic ID
			if router.GetCurrentPageID() != tt.pageID {
				t.Errorf("Expected current page to be '%s', got '%s'", tt.pageID, router.GetCurrentPageID())
			}

			// Verify page was created
			page := router.GetCurrentPage()
			if page == nil {
				t.Error("Expected page to be created, got nil")
			}
		})
	}
}

func TestNavigationHistory(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Navigate through several pages
	router.NavigateTo("main", nil)
	router.NavigateTo("topic", &NavigationData{TopicName: "test-topic"})
	router.NavigateTo("message_detail", &NavigationData{TopicName: "test-topic"})

	// Check history
	history := router.GetHistory()
	expectedHistory := []string{"main", "topic"}

	if len(history) != len(expectedHistory) {
		t.Errorf("Expected history length %d, got %d", len(expectedHistory), len(history))
	}

	for i, expected := range expectedHistory {
		if i >= len(history) || history[i] != expected {
			t.Errorf("Expected history[%d] to be '%s', got '%s'", i, expected, history[i])
		}
	}

	// Test back navigation
	cmd := router.Back()
	if cmd != nil {
		// Execute the command to complete navigation
		router.Update(cmd())
	}
	if router.GetCurrentPageID() != "topic" {
		t.Errorf("Expected current page after back to be 'topic', got '%s'", router.GetCurrentPageID())
	}

	cmd = router.Back()
	if cmd != nil {
		router.Update(cmd())
	}
	if router.GetCurrentPageID() != "main" {
		t.Errorf("Expected current page after second back to be 'main', got '%s'", router.GetCurrentPageID())
	}

	// Back from main should do nothing
	cmd = router.Back()
	if cmd != nil {
		router.Update(cmd())
	}
	if router.GetCurrentPageID() != "main" {
		t.Errorf("Expected current page to remain 'main', got '%s'", router.GetCurrentPageID())
	}
}

func TestSetDimensions(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Create a page first
	router.NavigateTo("main", nil)

	// Set dimensions
	width, height := 100, 50
	router.SetDimensions(width, height)

	if router.width != width {
		t.Errorf("Expected router width to be %d, got %d", width, router.width)
	}

	if router.height != height {
		t.Errorf("Expected router height to be %d, got %d", height, router.height)
	}

	// Verify dimensions were propagated to pages
	// Note: We can't easily test this without exposing page internals,
	// but we can verify the method doesn't panic
}

func TestClearHistory(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Build up some history
	router.NavigateTo("main", nil)
	router.NavigateTo("topic", nil)
	router.NavigateTo("message_detail", nil)

	// Verify history exists
	if len(router.GetHistory()) == 0 {
		t.Error("Expected history to have entries before clearing")
	}

	// Clear history
	router.ClearHistory()

	// Verify history is empty
	if len(router.GetHistory()) != 0 {
		t.Errorf("Expected empty history after clearing, got %d entries", len(router.GetHistory()))
	}
}

func TestRouterUpdate(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Initialize router
	router.NavigateTo("main", nil)

	// Test PageChangeMsg handling
	pageChangeMsg := core.PageChangeMsg{
		PageID: "topic",
		Data: map[string]interface{}{
			"name": "test-topic",
			"topic": api.Topic{
				NumPartitions:     3,
				ReplicationFactor: 1,
			},
		},
	}

	// Update router with page change message
	updatedRouter, _ := router.Update(pageChangeMsg)

	// Verify router was updated
	if updatedRouter == nil {
		t.Error("Expected updated router, got nil")
	}

	// Note: cmd might be nil if navigation is immediate, which is fine

	// Verify current page changed
	if router.GetCurrentPageID() != "topic" {
		t.Errorf("Expected current page to be 'topic', got '%s'", router.GetCurrentPageID())
	}
}

func TestRouterView(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Test view with no current page
	view := router.View()
	if view == "" {
		t.Error("Expected non-empty view")
	}

	// Navigate to main page and test view
	router.NavigateTo("main", nil)
	view = router.View()
	if view == "" {
		t.Error("Expected non-empty view after navigation")
	}
}

func TestCreatePageFallbacks(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	tests := []struct {
		name   string
		pageID string
		data   interface{}
	}{
		{
			name:   "Topic page with nil data",
			pageID: "topic",
			data:   nil,
		},
		{
			name:   "Message detail page with nil data",
			pageID: "message_detail",
			data:   nil,
		},
		{
			name:   "Resource detail page with nil data",
			pageID: "resource_detail",
			data:   nil,
		},
		{
			name:   "Unknown page ID",
			pageID: "unknown",
			data:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic and should create a page
			router.NavigateTo(tt.pageID, tt.data)

			page := router.GetCurrentPage()
			if page == nil {
				t.Error("Expected page to be created even with nil data")
			}
		})
	}
}

// TestBackMsgHandling verifies that BackMsg doesn't add to history
func TestBackMsgHandling(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Navigate: main -> topic -> message_detail
	router.NavigateTo("main", nil)
	router.NavigateTo("topic", &NavigationData{TopicName: "test-topic"})
	router.NavigateTo("message_detail", &NavigationData{TopicName: "test-topic"})

	// Verify initial history
	initialHistory := router.GetHistory()
	expectedInitialLen := 2 // [main, topic]
	if len(initialHistory) != expectedInitialLen {
		t.Errorf("Expected initial history length %d, got %d", expectedInitialLen, len(initialHistory))
	}

	// Simulate BackMsg (like pressing Esc on message detail)
	backMsg := core.BackMsg{}
	router.Update(backMsg)

	// After back, should be on topic page
	if router.GetCurrentPageID() != "topic" {
		t.Errorf("Expected current page to be 'topic' after BackMsg, got '%s'", router.GetCurrentPageID())
	}

	// History should NOT have grown - it should have one less entry
	historyAfterBack := router.GetHistory()
	expectedAfterBackLen := 1 // [main]
	if len(historyAfterBack) != expectedAfterBackLen {
		t.Errorf("Expected history length %d after back, got %d. History: %v", expectedAfterBackLen, len(historyAfterBack), historyAfterBack)
	}

	// Now simulate going forward again: topic -> message_detail
	router.NavigateTo("message_detail", &NavigationData{TopicName: "test-topic"})

	// History should be: [main, topic]
	historyAfterForward := router.GetHistory()
	expectedAfterForwardLen := 2 // [main, topic]
	if len(historyAfterForward) != expectedAfterForwardLen {
		t.Errorf("Expected history length %d after forward, got %d. History: %v", expectedAfterForwardLen, len(historyAfterForward), historyAfterForward)
	}

	// Press Esc again (BackMsg)
	router.Update(backMsg)

	// Should be back on topic
	if router.GetCurrentPageID() != "topic" {
		t.Errorf("Expected current page to be 'topic' after second BackMsg, got '%s'", router.GetCurrentPageID())
	}

	// History should still be: [main] (not growing)
	finalHistory := router.GetHistory()
	expectedFinalLen := 1 // [main]
	if len(finalHistory) != expectedFinalLen {
		t.Errorf("Expected final history length %d, got %d. History: %v", expectedFinalLen, len(finalHistory), finalHistory)
	}
}

// TestHistoryDoesNotGrow verifies that back-and-forth navigation doesn't create history loops
func TestHistoryDoesNotGrow(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Navigate: main -> topic -> message_detail
	router.NavigateTo("main", nil)
	router.NavigateTo("topic", &NavigationData{TopicName: "test-topic"})
	router.NavigateTo("message_detail", &NavigationData{TopicName: "test-topic"})

	// Record baseline history length
	baselineHistoryLen := len(router.GetHistory())

	// Simulate multiple back-and-forth navigations
	for i := 0; i < 5; i++ {
		// Go back (Esc)
		router.Update(core.BackMsg{})
		if router.GetCurrentPageID() != "topic" {
			t.Errorf("Iteration %d: Expected 'topic' after back, got '%s'", i, router.GetCurrentPageID())
		}

		// Go forward again
		router.NavigateTo("message_detail", &NavigationData{TopicName: "test-topic"})
		if router.GetCurrentPageID() != "message_detail" {
			t.Errorf("Iteration %d: Expected 'message_detail' after forward, got '%s'", i, router.GetCurrentPageID())
		}
	}

	// History should not have grown beyond the baseline
	finalHistoryLen := len(router.GetHistory())
	if finalHistoryLen != baselineHistoryLen {
		t.Errorf("History grew from %d to %d after back-and-forth navigation. History: %v", baselineHistoryLen, finalHistoryLen, router.GetHistory())
	}
}

// TestRouter_DynamicPageIDs tests that the router correctly handles dynamic page IDs
func TestRouter_DynamicPageIDs(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Test topic page with dynamic ID
	router.NavigateTo("topic:my-topic", &NavigationData{TopicName: "my-topic"})
	assert.Equal(t, "topic:my-topic", router.GetCurrentPageID())

	// Test message detail page with dynamic ID
	router.NavigateTo("detail:my-topic:0:123", &NavigationData{
		TopicName: "my-topic",
		Message: api.Message{
			Partition: 0,
			Offset:    123,
		},
	})
	assert.Equal(t, "detail:my-topic:0:123", router.GetCurrentPageID())

	// Test resource detail page with dynamic ID
	router.NavigateTo("resource_detail:group-1", &NavigationData{
		ResourceType: "consumer-group",
	})
	assert.Equal(t, "resource_detail:group-1", router.GetCurrentPageID())
}

// TestRouter_BaseIDExtraction tests that the router correctly extracts base IDs
func TestRouter_BaseIDExtraction(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	testCases := []struct {
		dynamicID string
		baseID    string
		pageType  string
	}{
		{"topic:my-topic", "topic", "topic"},
		{"topic:another-topic", "topic", "topic"},
		{"detail:topic:0:123", "detail", "message_detail"},
		{"detail:topic:1:456", "detail", "message_detail"},
		{"resource_detail:group-1", "resource_detail", "resource_detail"},
		{"main", "main", "main"},
	}

	for _, tc := range testCases {
		t.Run(tc.dynamicID, func(t *testing.T) {
			router.NavigateTo(tc.dynamicID, nil)
			
			// Verify current page ID is preserved (full dynamic ID)
			assert.Equal(t, tc.dynamicID, router.GetCurrentPageID())
			
			// Verify page was created (not nil)
			currentPage := router.GetCurrentPage()
			assert.NotNil(t, currentPage)
		})
	}
}

// TestRouter_DifferentMessageIDs tests that different message IDs create different pages
func TestRouter_DifferentMessageIDs(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Navigate to first message
	router.NavigateTo("detail:topic1:0:100", &NavigationData{
		TopicName: "topic1",
		Message: api.Message{
			Partition: 0,
			Offset:    100,
		},
	})
	
	page1 := router.GetCurrentPage()
	page1ID := router.GetCurrentPageID()

	// Navigate to different message
	router.NavigateTo("detail:topic1:0:200", &NavigationData{
		TopicName: "topic1",
		Message: api.Message{
			Partition: 0,
			Offset:    200,
		},
	})
	
	page2 := router.GetCurrentPage()
	page2ID := router.GetCurrentPageID()

	// Should have different page IDs
	assert.NotEqual(t, page1ID, page2ID)
	assert.Contains(t, page1ID, "100")
	assert.Contains(t, page2ID, "200")
	
	// Both should be message detail pages (same type, different instances)
	assert.NotNil(t, page1)
	assert.NotNil(t, page2)
}

// TestRouter_NavigationWithUniqueTopicIDs tests navigation between different topics
func TestRouter_NavigationWithUniqueTopicIDs(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Start at main
	router.NavigateTo("main", nil)
	assert.Equal(t, "main", router.GetCurrentPageID())

	// Navigate to first topic
	router.NavigateTo("topic:topic-1", &NavigationData{TopicName: "topic-1"})
	assert.Equal(t, "topic:topic-1", router.GetCurrentPageID())

	// Navigate to second topic
	router.NavigateTo("topic:topic-2", &NavigationData{TopicName: "topic-2"})
	assert.Equal(t, "topic:topic-2", router.GetCurrentPageID())

	// Navigate back
	router.Update(core.BackMsg{})
	assert.Equal(t, "topic:topic-1", router.GetCurrentPageID())

	// Navigate back again
	router.Update(core.BackMsg{})
	assert.Equal(t, "main", router.GetCurrentPageID())
}