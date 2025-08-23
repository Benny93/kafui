package topic

import (
	"context"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	tea "github.com/charmbracelet/bubbletea"
)

// MockDataSource is a mock implementation of api.KafkaDataSource for testing
type MockDataSource struct {
	mock.Mock
}

func (m *MockDataSource) Init(cfgOption string) {
	m.Called(cfgOption)
}

func (m *MockDataSource) GetTopics() (map[string]api.Topic, error) {
	args := m.Called()
	return args.Get(0).(map[string]api.Topic), args.Error(1)
}

func (m *MockDataSource) GetConsumerGroups() ([]api.ConsumerGroup, error) {
	args := m.Called()
	return args.Get(0).([]api.ConsumerGroup), args.Error(1)
}

func (m *MockDataSource) GetContexts() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockDataSource) GetContext() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockDataSource) SetContext(contextName string) error {
	args := m.Called(contextName)
	return args.Error(0)
}

func (m *MockDataSource) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) error {
	args := m.Called(ctx, topicName, flags, handleMessage, onError)
	return args.Error(0)
}

func TestNewModel(t *testing.T) {
	// Create mock data source
	mockDS := &MockDataSource{}
	
	// Create test topic details
	topicDetails := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 2,
		MessageCount:      100,
	}
	
	// Create new model
	model := NewModel(mockDS, "test-topic", topicDetails)
	
	// Verify model is properly initialized
	assert.NotNil(t, model)
	assert.Equal(t, mockDS, model.dataSource)
	assert.Equal(t, "test-topic", model.topicName)
	assert.Equal(t, topicDetails, model.topicDetails)
	assert.NotNil(t, model.handlers)
	assert.NotNil(t, model.keys)
	assert.NotNil(t, model.view)
	assert.NotNil(t, model.consumption)
	assert.NotNil(t, model.messageTable)
	assert.NotNil(t, model.spinner)
	assert.NotNil(t, model.searchInput)
	assert.False(t, model.loading)
	assert.False(t, model.consuming)
	assert.False(t, model.paused)
	assert.False(t, model.searchMode)
	assert.Equal(t, "Topic page initialized", model.statusMessage)
	assert.Equal(t, 3, model.maxRetries)
	assert.Equal(t, "disconnected", model.connectionStatus)
}

func TestModelImplementsPageInterface(t *testing.T) {
	// Create mock data source
	mockDS := &MockDataSource{}
	
	// Create test topic details
	topicDetails := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 2,
		MessageCount:      100,
	}
	
	// Create new model
	model := NewModel(mockDS, "test-topic", topicDetails)
	
	// Test that model implements the Page interface methods
	assert.Equal(t, "topic", model.GetID())
	
	// Test Init returns a command
	cmd := model.Init()
	assert.NotNil(t, cmd)
	
	// Test SetDimensions
	model.SetDimensions(80, 24)
	assert.Equal(t, 80, model.dimensions.Width)
	assert.Equal(t, 24, model.dimensions.Height)
	
	// Test View returns a string (basic test)
	model.SetDimensions(80, 24) // Ensure dimensions are set
	view := model.View()
	assert.IsType(t, "", view)
}

func TestAddMessage(t *testing.T) {
	// Create mock data source
	mockDS := &MockDataSource{}
	
	// Create test topic details
	topicDetails := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 2,
		MessageCount:      100,
	}
	
	// Create new model
	model := NewModel(mockDS, "test-topic", topicDetails)
	
	// Test adding a message
	testMessage := api.Message{
		Key:       "test-key",
		Value:     "test-value",
		Offset:    123,
		Partition: 0,
	}
	
	// Initially should have no messages
	assert.Len(t, model.messages, 0)
	assert.Len(t, model.filteredMessages, 0)
	
	// Add message
	model.AddMessage(testMessage)
	
	// Should now have one message
	assert.Len(t, model.messages, 1)
	assert.Len(t, model.filteredMessages, 1)
	assert.Equal(t, testMessage, model.messages[0])
	assert.Contains(t, model.statusMessage, "Consumed 1 messages")
	
	// Add duplicate message (should not be added)
	model.AddMessage(testMessage)
	assert.Len(t, model.messages, 1) // Should still be 1
}

func TestFilterMessages(t *testing.T) {
	// Create mock data source
	mockDS := &MockDataSource{}
	
	// Create test topic details
	topicDetails := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 2,
		MessageCount:      100,
	}
	
	// Create new model
	model := NewModel(mockDS, "test-topic", topicDetails)
	
	// Add test messages
	messages := []api.Message{
		{Key: "user-123", Value: "user data", Offset: 1, Partition: 0},
		{Key: "order-456", Value: "order data", Offset: 2, Partition: 0},
		{Key: "user-789", Value: "more user data", Offset: 3, Partition: 0},
	}
	
	for _, msg := range messages {
		model.AddMessage(msg)
	}
	
	// Test no filter (should show all messages)
	model.FilterMessages()
	assert.Len(t, model.filteredMessages, 3)
	
	// Test filter by key
	model.searchMode = true
	model.searchInput.SetValue("user")
	model.FilterMessages()
	assert.Len(t, model.filteredMessages, 2) // Should match 2 messages with "user" in key or value
	
	// Test filter by value
	model.searchInput.SetValue("order")
	model.FilterMessages()
	assert.Len(t, model.filteredMessages, 1) // Should match 1 message with "order"
	
	// Test no matches
	model.searchInput.SetValue("nonexistent")
	model.FilterMessages()
	assert.Len(t, model.filteredMessages, 0) // Should match no messages
}

func TestTogglePause(t *testing.T) {
	// Create mock data source
	mockDS := &MockDataSource{}
	
	// Create test topic details
	topicDetails := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 2,
		MessageCount:      100,
	}
	
	// Create new model
	model := NewModel(mockDS, "test-topic", topicDetails)
	
	// Initially should not be paused
	assert.False(t, model.paused)
	
	// Toggle pause on
	model.TogglePause()
	assert.True(t, model.paused)
	assert.Contains(t, model.statusMessage, "paused")
	
	// Toggle pause off
	model.TogglePause()
	assert.False(t, model.paused)
	assert.Contains(t, model.statusMessage, "resumed")
}

func TestSetError(t *testing.T) {
	// Create mock data source
	mockDS := &MockDataSource{}
	
	// Create test topic details
	topicDetails := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 2,
		MessageCount:      100,
	}
	
	// Create new model
	model := NewModel(mockDS, "test-topic", topicDetails)
	
	// Test setting an error
	testError := assert.AnError
	model.SetError(testError)
	
	assert.Equal(t, testError, model.error)
	assert.Equal(t, testError, model.lastError)
	assert.Len(t, model.errorHistory, 1)
	assert.Equal(t, "failed", model.connectionStatus)
	assert.Contains(t, model.statusMessage, "Error:")
}

func TestSetConnectionStatus(t *testing.T) {
	// Create mock data source
	mockDS := &MockDataSource{}
	
	// Create test topic details
	topicDetails := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 2,
		MessageCount:      100,
	}
	
	// Create new model
	model := NewModel(mockDS, "test-topic", topicDetails)
	
	// Test different connection statuses
	testCases := []struct {
		status          string
		expectedMessage string
	}{
		{"connected", "Connected and consuming messages"},
		{"connecting", "Connecting to topic..."},
		{"disconnected", "Disconnected"},
		{"failed", "Connection failed"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.status, func(t *testing.T) {
			model.SetConnectionStatus(tc.status)
			assert.Equal(t, tc.status, model.connectionStatus)
			assert.Contains(t, model.statusMessage, tc.expectedMessage)
		})
	}
}

func TestGetSelectedMessage(t *testing.T) {
	// Create mock data source
	mockDS := &MockDataSource{}
	
	// Create test topic details
	topicDetails := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 2,
		MessageCount:      100,
	}
	
	// Create new model
	model := NewModel(mockDS, "test-topic", topicDetails)
	
	// Test with no messages
	selected := model.GetSelectedMessage()
	assert.Nil(t, selected)
	
	// Add a test message
	testMessage := api.Message{
		Key:       "test-key",
		Value:     "test-value",
		Offset:    123,
		Partition: 0,
	}
	model.AddMessage(testMessage)
	
	// Test with messages (cursor should be at 0)
	selected = model.GetSelectedMessage()
	assert.NotNil(t, selected)
	assert.Equal(t, testMessage, *selected)
}

func TestWindowSizeUpdate(t *testing.T) {
	// Create mock data source
	mockDS := &MockDataSource{}
	
	// Create test topic details
	topicDetails := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 2,
		MessageCount:      100,
	}
	
	// Create new model
	model := NewModel(mockDS, "test-topic", topicDetails)
	
	// Test window size message
	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	updatedModel, cmd := model.Update(msg)
	
	// Should return the same model type
	assert.IsType(t, &Model{}, updatedModel)
	assert.NotNil(t, cmd) // May return a command
	
	// Check dimensions were updated
	updatedTopicModel := updatedModel.(*Model)
	assert.Equal(t, 100, updatedTopicModel.dimensions.Width)
	assert.Equal(t, 30, updatedTopicModel.dimensions.Height)
}