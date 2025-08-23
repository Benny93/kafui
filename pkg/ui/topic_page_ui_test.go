package ui

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/stretchr/testify/assert"
)

// TestTopicPageUIStates tests the UI rendering in different states
func TestTopicPageUIStates(t *testing.T) {
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

	// Test 1: Initial state (before Init)
	messagesSection := topicPage.renderMessages()
	assert.Contains(t, messagesSection, "No messages available", "Should show no messages initially")
	assert.NotContains(t, messagesSection, "Loading", "Should not be loading initially")

	// Test 2: Loading state (after Init)
	topicPage.loading = true
	messagesSection = topicPage.renderMessages()
	assert.Contains(t, messagesSection, "Loading messages", "Should show loading when loading is true")

	// Test 3: Consuming but no messages yet
	topicPage.loading = false
	topicPage.consuming = true
	messagesSection = topicPage.renderMessages()
	assert.Contains(t, messagesSection, "Waiting for messages", "Should show waiting when consuming but no messages")

	// Test 4: Not consuming, no messages
	topicPage.consuming = false
	messagesSection = topicPage.renderMessages()
	assert.Contains(t, messagesSection, "No messages available", "Should show no messages when not consuming")
	assert.Contains(t, messagesSection, "start consumption", "Should provide instruction to start consumption")

	// Test 5: Has messages
	testMessage := api.Message{
		Key:       "test-key",
		Value:     "test-value",
		Offset:    100,
		Partition: 0,
	}
	topicPage.messages = []api.Message{testMessage}
	topicPage.filteredMessages = []api.Message{testMessage}
	topicPage.updateTable()

	messagesSection = topicPage.renderMessages()
	assert.NotContains(t, messagesSection, "No messages", "Should not show 'no messages' when messages exist")
	assert.NotContains(t, messagesSection, "Loading", "Should not show loading when messages exist")
	assert.NotContains(t, messagesSection, "Waiting", "Should not show waiting when messages exist")
	// Should show the table (we can't easily test the exact table content here)
	assert.NotEmpty(t, messagesSection, "Should show table content when messages exist")
}

// TestConnectionStatusHandling tests how different connection statuses are handled
func TestConnectionStatusHandling(t *testing.T) {
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

	// Test initial connection status
	assert.Equal(t, "disconnected", topicPage.connectionStatus, "Should start disconnected")

	// Test connecting status message
	connectingMsg := messageConsumedMsg(api.Message{
		Key:   "__status__",
		Value: "connecting",
	})
	updatedModel, _ := topicPage.Update(connectingMsg)
	topicPage = *updatedModel.(*TopicPageModel)

	assert.Equal(t, "connecting", topicPage.connectionStatus, "Should update to connecting")
	assert.Contains(t, topicPage.statusMessage, "Connecting to Kafka", "Should show connecting message")

	// Test connected status message
	connectedMsg := messageConsumedMsg(api.Message{
		Key:   "__status__",
		Value: "connected",
	})
	updatedModel, _ = topicPage.Update(connectedMsg)
	topicPage = *updatedModel.(*TopicPageModel)

	assert.Equal(t, "connected", topicPage.connectionStatus, "Should update to connected")
	assert.Contains(t, topicPage.statusMessage, "Successfully connected", "Should show success message")
	assert.Equal(t, 0, topicPage.retryCount, "Should reset retry count on successful connection")
}
