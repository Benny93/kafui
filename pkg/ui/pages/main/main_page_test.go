package mainpage

import (
	"context"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDataSource is a mock implementation of api.KafkaDataSource for testing
type MockDataSource struct {
	mock.Mock
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

func (m *MockDataSource) Init(cfgOption string) {
	m.Called(cfgOption)
}

func (m *MockDataSource) SetContext(contextName string) error {
	args := m.Called(contextName)
	return args.Error(0)
}

func (m *MockDataSource) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) error {
	args := m.Called(ctx, topicName, flags, handleMessage, onError)
	return args.Error(0)
}

func (m *MockDataSource) GetMessageSchemaInfo(keySchemaID, valueSchemaID string) (*api.MessageSchemaInfo, error) {
	args := m.Called(keySchemaID, valueSchemaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*api.MessageSchemaInfo), args.Error(1)
}

func TestNewModel(t *testing.T) {
	// Create mock data source
	mockDS := &MockDataSource{}

	// Create new model (no external calls during construction)
	model := NewModel(mockDS)

	// Verify model is properly initialized
	assert.NotNil(t, model)
	assert.Equal(t, mockDS, model.dataSource)
	assert.NotNil(t, model.handlers)
	assert.NotNil(t, model.keys)
	assert.NotNil(t, model.view)
	assert.NotNil(t, model.resourceManager)
	assert.NotNil(t, model.currentResource)
	assert.Equal(t, TopicResourceType, model.currentResource.GetType())
	assert.False(t, model.loading)
	assert.False(t, model.searchMode)
	assert.False(t, model.isFiltered)
	assert.Equal(t, "", model.currentFilter)
	assert.Equal(t, "Welcome to Kafui", model.statusMessage)

	// No mock expectations to verify since NewModel doesn't make external calls
}

func TestModelImplementsPageInterface(t *testing.T) {
	// Create mock data source
	mockDS := &MockDataSource{}

	// Set up expected method calls for View rendering
	mockDS.On("GetContext").Return("test-context")

	// Create new model
	model := NewModel(mockDS)

	// Test that model implements the Page interface methods
	assert.Equal(t, "main", model.GetID())

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

	// Verify mock expectations
	mockDS.AssertExpectations(t)
}

func TestResourceManager(t *testing.T) {
	// Create mock data source
	mockDS := &MockDataSource{}

	// Create resource manager
	rm := NewResourceManager(mockDS)

	// Test resource manager is properly initialized
	assert.NotNil(t, rm)
	assert.Equal(t, mockDS, rm.dataSource)

	// Test all resource types are available
	resourceTypes := rm.GetResourceTypes()
	assert.Len(t, resourceTypes, 4)
	assert.Contains(t, resourceTypes, TopicResourceType)
	assert.Contains(t, resourceTypes, ConsumerGroupResourceType)
	assert.Contains(t, resourceTypes, SchemaResourceType)
	assert.Contains(t, resourceTypes, ContextResourceType)

	// Test getting resources by type
	topicResource := rm.GetResource(TopicResourceType)
	assert.NotNil(t, topicResource)
	assert.Equal(t, TopicResourceType, topicResource.GetType())
	assert.Equal(t, "Topics", topicResource.GetName())

	consumerGroupResource := rm.GetResource(ConsumerGroupResourceType)
	assert.NotNil(t, consumerGroupResource)
	assert.Equal(t, ConsumerGroupResourceType, consumerGroupResource.GetType())
	assert.Equal(t, "Consumer Groups", consumerGroupResource.GetName())

	// No mock expectations to verify since resource manager doesn't call external methods during creation
}

func TestResourceTypes(t *testing.T) {
	// Test ResourceType string representation
	assert.Equal(t, "topics", TopicResourceType.String())
	assert.Equal(t, "consumer-groups", ConsumerGroupResourceType.String())
	assert.Equal(t, "schemas", SchemaResourceType.String())
	assert.Equal(t, "contexts", ContextResourceType.String())
}

func TestTopicResourceItem(t *testing.T) {
	// Create a topic resource item
	topic := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 2,
		MessageCount:      100,
	}

	item := &TopicResourceItem{
		id:                "test-topic",
		topic:             topic,
		partitions:        topic.NumPartitions,
		replicationFactor: topic.ReplicationFactor,
		messageCount:      topic.MessageCount,
	}

	// Test ID
	assert.Equal(t, "test-topic", item.GetID())

	// Test Values
	values := item.GetValues()
	assert.Len(t, values, 4)
	assert.Equal(t, "test-topic", values[0])
	assert.Equal(t, "3", values[1])
	assert.Equal(t, "2", values[2])
	assert.Equal(t, "100", values[3])

	// Test Details
	details := item.GetDetails()
	assert.Equal(t, "test-topic", details["Name"])
	assert.Equal(t, "3", details["Partitions"])
	assert.Equal(t, "2", details["Replication Factor"])
	assert.Equal(t, "100", details["Message Count"])
}

func TestConsumerGroupResourceItem(t *testing.T) {
	// Create a consumer group resource item
	group := api.ConsumerGroup{
		Name:      "test-group",
		State:     "Stable",
		Consumers: 2,
	}

	item := &ConsumerGroupResourceItem{
		id:        "test-group",
		group:     group,
		state:     group.State,
		consumers: group.Consumers,
	}

	// Test ID
	assert.Equal(t, "test-group", item.GetID())

	// Test Values
	values := item.GetValues()
	assert.Len(t, values, 3)
	assert.Equal(t, "test-group", values[0])
	assert.Equal(t, "Stable", values[1])
	assert.Equal(t, "2", values[2])

	// Test Details
	details := item.GetDetails()
	assert.Equal(t, "test-group", details["Name"])
	assert.Equal(t, "Stable", details["State"])
	assert.Equal(t, "2", details["Consumers"])
}

func TestKeys(t *testing.T) {
	// Create keys handler
	keys := NewKeys()

	// Test keys is properly initialized
	assert.NotNil(t, keys)
	assert.NotNil(t, keys.bindings)

	// Test key bindings
	bindings := keys.GetKeyBindings()
	assert.True(t, len(bindings) > 0)
}

func TestHandlers(t *testing.T) {
	// Create mock data source
	mockDS := &MockDataSource{}

	// Create model
	model := NewModel(mockDS)

	// Test handlers is properly initialized
	assert.NotNil(t, model.handlers)
	assert.Equal(t, model, model.handlers.model)
}

func TestView(t *testing.T) {
	// Create view
	view := NewView()

	// Test view is properly initialized
	assert.NotNil(t, view)
	assert.NotNil(t, view.theme)
	assert.NotNil(t, view.styles)

	// Test SetDimensions
	view.SetDimensions(80, 24)
	assert.Equal(t, 80, view.dimensions.Width)
	assert.Equal(t, 24, view.dimensions.Height)
}
