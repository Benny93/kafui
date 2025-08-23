package ui

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/stretchr/testify/assert"
)

// TestTopicPageMessageConsumption tests that the topic page can consume and display messages
func TestTopicPageMessageConsumption(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}

	// Create topic details
	topicDetails := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 1,
		ConfigEntries:     make(map[string]*string),
	}

	// Create topic page
	topicPage := NewTopicPage(mockDS, "test-topic", topicDetails)
	topicPage.width = 120
	topicPage.height = 40

	// Test initial state before Init
	assert.False(t, topicPage.consuming, "Should not be consuming initially")
	assert.False(t, topicPage.loading, "Should not be loading before Init")
	assert.Equal(t, "disconnected", topicPage.connectionStatus, "Should be disconnected initially")

	// Call Init to set loading state
	initCmd := topicPage.Init()
	assert.NotNil(t, initCmd, "Init should return a command")
	assert.True(t, topicPage.loading, "Should be loading after Init")

	// Simulate direct message processing (bypassing the actual ConsumeTopic call)
	// This tests the message handling logic without depending on the mock data source

	// Simulate the startConsumingMsg being processed
	msgChan := make(chan api.Message, 10)
	errChan := make(chan error, 1)
	cancel := func() {} // Dummy cancel function

	startMsg := startConsumingMsg{
		cancel:  cancel,
		msgChan: msgChan,
		errChan: errChan,
	}

	updatedModel, _ := topicPage.Update(startMsg)
	topicPage = *updatedModel.(*TopicPageModel)

	// Verify consumption setup
	assert.True(t, topicPage.consuming, "Should be consuming after setup")
	assert.False(t, topicPage.loading, "Should not be loading after setup")
	assert.NotNil(t, topicPage.msgChan, "Message channel should be set")
	assert.NotNil(t, topicPage.errChan, "Error channel should be set")
	assert.NotNil(t, topicPage.cancelConsumption, "Cancel function should be set")

	// Test message processing
	testMessages := []api.Message{
		{
			Key:       "test-key-1",
			Value:     "test-value-1",
			Offset:    100,
			Partition: 0,
		},
		{
			Key:       "test-key-2",
			Value:     "test-value-2",
			Offset:    101,
			Partition: 1,
		},
		{
			Key:       "test-key-3",
			Value:     "test-value-3",
			Offset:    102,
			Partition: 0,
		},
	}

	// Process each message
	for _, message := range testMessages {
		msgReceived := messageConsumedMsg(message)
		updatedModel, _ = topicPage.Update(msgReceived)
		topicPage = *updatedModel.(*TopicPageModel)
	}

	// Verify messages were processed
	assert.Len(t, topicPage.messages, 3, "Should have 3 messages")
	assert.Len(t, topicPage.filteredMessages, 3, "Should have 3 filtered messages")

	// Check that messages are sorted by offset
	assert.Equal(t, int64(100), topicPage.messages[0].Offset, "First message should have offset 100")
	assert.Equal(t, int64(101), topicPage.messages[1].Offset, "Second message should have offset 101")
	assert.Equal(t, int64(102), topicPage.messages[2].Offset, "Third message should have offset 102")

	// Verify table was updated
	assert.Contains(t, topicPage.statusMessage, "Consumed 3 messages", "Status should show message count")

	// Test the rendering - should show messages now
	messagesSection := topicPage.renderMessages()
	assert.NotContains(t, messagesSection, "Loading messages", "Should not be loading when messages exist")
	assert.NotContains(t, messagesSection, "No messages available", "Should not show 'no messages' when messages exist")
	// The message table should contain the table view now
	assert.NotEmpty(t, messagesSection, "Messages section should not be empty")

	// Test that we can see message content in the actual table
	assert.Len(t, topicPage.messageTable.Rows(), 3, "Table should have 3 rows")
}

// TestTopicPageMessageFiltering tests message search/filtering functionality
func TestTopicPageMessageFiltering(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}

	// Create topic details
	topicDetails := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 1,
	}

	// Create topic page
	topicPage := NewTopicPage(mockDS, "test-topic", topicDetails)
	topicPage.width = 120
	topicPage.height = 40

	// Add some test messages directly
	testMessages := []api.Message{
		{
			Key:       "user-events",
			Value:     "user login event",
			Offset:    100,
			Partition: 0,
		},
		{
			Key:       "order-events",
			Value:     "order created event",
			Offset:    101,
			Partition: 1,
		},
		{
			Key:       "user-analytics",
			Value:     "user action event",
			Offset:    102,
			Partition: 0,
		},
	}

	topicPage.messages = testMessages
	topicPage.filterMessages() // Initial filter with no search

	// Verify all messages are shown initially
	assert.Len(t, topicPage.filteredMessages, 3, "Should show all messages initially")

	// Test filtering by key
	topicPage.searchInput.SetValue("user")
	topicPage.filterMessages()

	assert.Len(t, topicPage.filteredMessages, 2, "Should show 2 messages containing 'user'")

	// Verify the correct messages are filtered
	assert.Contains(t, topicPage.filteredMessages[0].Key, "user", "First filtered message should contain 'user'")
	assert.Contains(t, topicPage.filteredMessages[1].Key, "user", "Second filtered message should contain 'user'")

	// Test filtering by value
	topicPage.searchInput.SetValue("login")
	topicPage.filterMessages()

	assert.Len(t, topicPage.filteredMessages, 1, "Should show 1 message containing 'login'")

	// Test filtering with no results
	topicPage.searchInput.SetValue("nonexistent")
	topicPage.filterMessages()

	assert.Len(t, topicPage.filteredMessages, 0, "Should show no messages for non-matching search")

	// Test clearing filter
	topicPage.searchInput.SetValue("")
	topicPage.filterMessages()

	assert.Len(t, topicPage.filteredMessages, 3, "Should show all messages when search is cleared")
}
