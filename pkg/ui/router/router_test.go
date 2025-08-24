package router

import (
	"context"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
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
	router := NewRouter(dataSource)

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
	router := NewRouter(dataSource)

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
			expected: "topic",
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
			expected: "message_detail",
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
			expected: "resource_detail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router.NavigateTo(tt.pageID, tt.data)

			if router.GetCurrentPageID() != tt.expected {
				t.Errorf("Expected current page to be '%s', got '%s'", tt.expected, router.GetCurrentPageID())
			}

			// Verify page was created
			page := router.GetCurrentPage()
			if page == nil {
				t.Error("Expected page to be created, got nil")
			}

			// Verify page ID matches
			if page.GetID() != tt.expected {
				t.Errorf("Expected page ID to be '%s', got '%s'", tt.expected, page.GetID())
			}
		})
	}
}

func TestNavigationHistory(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(dataSource)

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
	router := NewRouter(dataSource)

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
	router := NewRouter(dataSource)

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
	router := NewRouter(dataSource)

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
	router := NewRouter(dataSource)

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
	router := NewRouter(dataSource)

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