package kafui

import (
	"context"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/rivo/tview"
)

// MockResource implements the Resource interface for testing
type MockResource struct {
	name           string
	fetchingActive bool
	updateCalled   bool
}

func (m *MockResource) StartFetchingData() {
	m.fetchingActive = true
}

func (m *MockResource) UpdateTable(table *tview.Table, dataSource api.KafkaDataSource, search string) {
	m.updateCalled = true
}

func (m *MockResource) StopFetching() {
	m.fetchingActive = false
}

func (m *MockResource) GetName() string {
	return m.name
}

// TestResourceInterface tests the Resource interface implementation
func TestResourceInterface(t *testing.T) {
	mockResource := &MockResource{name: "TestResource"}

	// Test that MockResource implements Resource interface
	var resource Resource = mockResource

	// Test GetName
	if resource.GetName() != "TestResource" {
		t.Errorf("GetName() = %v, want %v", resource.GetName(), "TestResource")
	}

	// Test StartFetchingData
	resource.StartFetchingData()
	if !mockResource.fetchingActive {
		t.Error("StartFetchingData() should set fetchingActive to true")
	}

	// Test UpdateTable
	table := tview.NewTable()
	resource.UpdateTable(table, nil, "")
	if !mockResource.updateCalled {
		t.Error("UpdateTable() should set updateCalled to true")
	}

	// Test StopFetching
	resource.StopFetching()
	if mockResource.fetchingActive {
		t.Error("StopFetching() should set fetchingActive to false")
	}
}

// TestResourceInterfaceContract tests that all resource types implement the interface correctly
func TestResourceInterfaceContract(t *testing.T) {
	// Mock error handler and recover function
	onError := func(err error) {}
	recoverFunc := func() {}

	// Create mock data source
	mockDataSource := &MockKafkaDataSource{}

	tests := []struct {
		name     string
		resource Resource
	}{
		{
			name:     "ResourceContext",
			resource: NewResourceContext(mockDataSource, onError, recoverFunc),
		},
		{
			name:     "ResourceGroup",
			resource: NewResourceGroup(onError, mockDataSource, recoverFunc),
		},
		{
			name:     "ResouceTopic",
			resource: NewResouceTopic(mockDataSource, onError, recoverFunc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that each resource implements the interface
			if tt.resource == nil {
				t.Fatalf("Resource %s is nil", tt.name)
			}

			// Test GetName returns non-empty string
			name := tt.resource.GetName()
			if name == "" {
				t.Errorf("GetName() should return non-empty string for %s", tt.name)
			}

			// Test that methods can be called without panic
			tt.resource.StartFetchingData()
			tt.resource.UpdateTable(tview.NewTable(), mockDataSource, "")
			tt.resource.StopFetching()
		})
	}
}

// MockKafkaDataSource for testing
type MockKafkaDataSource struct{}

func (m *MockKafkaDataSource) Init(cfgOption string) {}

func (m *MockKafkaDataSource) GetTopics() (map[string]api.Topic, error) {
	return map[string]api.Topic{
		"test-topic": {
			NumPartitions:     3,
			ReplicationFactor: 2,
			MessageCount:      100,
		},
	}, nil
}

func (m *MockKafkaDataSource) GetContexts() ([]string, error) {
	return []string{"context1", "context2"}, nil
}

func (m *MockKafkaDataSource) GetContext() string {
	return "current-context"
}

func (m *MockKafkaDataSource) SetContext(contextName string) error {
	return nil
}

func (m *MockKafkaDataSource) GetConsumerGroups() ([]api.ConsumerGroup, error) {
	return []api.ConsumerGroup{
		{Name: "group1", State: "Stable", Consumers: 3},
		{Name: "group2", State: "Rebalancing", Consumers: 2},
	}, nil
}

func (m *MockKafkaDataSource) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) error {
	return nil
}