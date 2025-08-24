package topic

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/stretchr/testify/assert"
)

func TestHighlightMatchingText(t *testing.T) {
	// Test basic highlighting
	result := highlightMatchingText("hello world", "world")
	assert.Contains(t, result, "world") // Should contain the highlighted text
	
	// Test case insensitive matching
	result = highlightMatchingText("Hello World", "world")
	assert.Contains(t, result, "World") // Should highlight "World"
	
	// Test no match
	result = highlightMatchingText("hello world", "xyz")
	assert.Equal(t, "hello world", result) // Should return original text
	
	// Test empty query
	result = highlightMatchingText("hello world", "")
	assert.Equal(t, "hello world", result) // Should return original text
}

func TestSearchFilteringAndHighlighting(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	// Create test topic
	testTopic := api.Topic{
		NumPartitions:     1,
		ReplicationFactor: 1,
		ReplicaAssignment: make(map[int32][]int32),
		ConfigEntries:     make(map[string]*string),
	}

	// Create new model
	model := NewModel(mockDS, "test-topic", testTopic)

	// Add test messages
	messages := []api.Message{
		{Key: "key1", Value: "hello world message", Offset: 1, Partition: 0},
		{Key: "key2", Value: "another test message", Offset: 2, Partition: 0},
		{Key: "search-key", Value: "find this message", Offset: 3, Partition: 0},
	}
	
	for _, msg := range messages {
		model.AddMessage(msg)
	}

	// Enable search mode and set search query
	model.searchMode = true
	model.searchInput.SetValue("test")
	
	// Filter messages
	model.FilterMessages()
	
	// Should have 1 matching message (the second one)
	assert.Equal(t, 1, len(model.filteredMessages))
	assert.Equal(t, "another test message", model.filteredMessages[0].Value)
	
	// Check that the table was updated
	assert.Equal(t, 1, len(model.messageTable.Rows()))
}