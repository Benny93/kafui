package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/kafds"
)

// MockDataSource for integration testing
type MockDataSource struct{}

func (m *MockDataSource) Init(cfgOption string) {}

func (m *MockDataSource) GetTopics() (map[string]api.Topic, error) {
	return map[string]api.Topic{
		"test-topic": {
			NumPartitions:     3,
			ReplicationFactor: 2,
			MessageCount:      100,
		},
	}, nil
}

func (m *MockDataSource) GetContexts() ([]string, error) {
	return []string{"test-context"}, nil
}

func (m *MockDataSource) GetContext() string {
	return "test-context"
}

func (m *MockDataSource) SetContext(contextName string) error {
	return nil
}

func (m *MockDataSource) GetConsumerGroups() ([]api.ConsumerGroup, error) {
	return []api.ConsumerGroup{
		{Name: "test-group", State: "Stable", Consumers: 1},
	}, nil
}

func (m *MockDataSource) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) error {
	return nil
}

// TestE2EKafkaIntegration tests end-to-end integration with real Kafka
// This test requires Docker environment to be running
func TestE2EKafkaIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E integration test in short mode")
	}

	// Check if we're in a test environment with Kafka available
	if os.Getenv("KAFUI_E2E_TEST") != "true" {
		t.Skip("Skipping E2E test - set KAFUI_E2E_TEST=true to run")
	}

	// Create test config for Kafka connection
	testConfig := createE2ETestConfig(t)
	defer os.Remove(testConfig)

	t.Run("Real Kafka Data Source Integration", func(t *testing.T) {
		dataSource := &kafds.KafkaDataSourceKaf{}
		dataSource.Init(testConfig)

		// Test basic operations
		topics, err := dataSource.GetTopics()
		if err != nil {
			t.Fatalf("Failed to get topics from real Kafka: %v", err)
		}

		// Verify test topics exist
		expectedTopics := []string{"test-topic-1", "test-topic-2", "test-topic-empty"}
		for _, expectedTopic := range expectedTopics {
			if _, exists := topics[expectedTopic]; !exists {
				t.Errorf("Expected topic %s not found in Kafka", expectedTopic)
			}
		}

		// Test consumer groups
		groups, err := dataSource.GetConsumerGroups()
		if err != nil {
			t.Fatalf("Failed to get consumer groups: %v", err)
		}

		// Verify test consumer groups exist
		expectedGroups := []string{"test-group-1", "test-group-2"}
		groupNames := make(map[string]bool)
		for _, group := range groups {
			groupNames[group.Name] = true
		}

		for _, expectedGroup := range expectedGroups {
			if !groupNames[expectedGroup] {
				t.Errorf("Expected consumer group %s not found", expectedGroup)
			}
		}
	})

	t.Run("Message Consumption E2E", func(t *testing.T) {
		dataSource := &kafds.KafkaDataSourceKaf{}
		dataSource.Init(testConfig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		messagesReceived := 0
		var lastMessage api.Message

		handleMessage := func(msg api.Message) {
			messagesReceived++
			lastMessage = msg
			t.Logf("Received message: Key=%s, Value=%s, Offset=%d, Partition=%d", 
				msg.Key, msg.Value, msg.Offset, msg.Partition)
		}

		onError := func(err any) {
			t.Logf("Consumer error (may be expected): %v", err)
		}

		flags := api.DefaultConsumeFlags()
		flags.Tail = 10
		flags.OffsetFlag = "earliest"

		// Start consumption
		go func() {
			err := dataSource.ConsumeTopic(ctx, "test-topic-1", flags, handleMessage, onError)
			if err != nil && err != context.DeadlineExceeded {
				t.Logf("ConsumeTopic ended with: %v", err)
			}
		}()

		// Wait for messages
		select {
		case <-time.After(8 * time.Second):
			if messagesReceived == 0 {
				t.Error("Expected to receive messages from test-topic-1")
			} else {
				t.Logf("Successfully received %d messages", messagesReceived)
				
				// Verify message structure
				if lastMessage.Key == "" && lastMessage.Value == "" {
					t.Error("Received message has empty key and value")
				}
			}
		case <-ctx.Done():
			t.Logf("Context cancelled, received %d messages", messagesReceived)
		}
	})

	t.Run("Context Switching E2E", func(t *testing.T) {
		dataSource := &kafds.KafkaDataSourceKaf{}
		dataSource.Init(testConfig)

		// Test getting contexts
		contexts, err := dataSource.GetContexts()
		if err != nil {
			t.Fatalf("Failed to get contexts: %v", err)
		}

		if len(contexts) == 0 {
			t.Skip("No contexts available for testing")
		}

		// Test context switching
		originalContext := dataSource.GetContext()
		t.Logf("Original context: %s", originalContext)

		for _, context := range contexts {
			err := dataSource.SetContext(context)
			if err != nil {
				t.Errorf("Failed to set context to %s: %v", context, err)
				continue
			}

			currentContext := dataSource.GetContext()
			if currentContext != context {
				t.Errorf("Expected context %s, got %s", context, currentContext)
			}

			// Verify we can still get topics after context switch
			_, err = dataSource.GetTopics()
			if err != nil {
				t.Errorf("Failed to get topics after switching to context %s: %v", context, err)
			}
		}

		// Restore original context
		if originalContext != "" {
			dataSource.SetContext(originalContext)
		}
	})
}

// TestE2EInitFlow tests the complete initialization flow end-to-end
func TestE2EInitFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E init flow test in short mode")
	}

	testConfig := createE2ETestConfig(t)
	defer os.Remove(testConfig)

	t.Run("Mock Mode Init Flow", func(t *testing.T) {
		// Test that Init doesn't panic with mock mode
		// Note: We can't easily test the full UI, but we can test that Init starts correctly
		
		// This would normally call kafui.Init(testConfig, true)
		// but since it starts the UI, we test the components separately
		
		// Test the data source selection logic that Init() uses
		var dataSource api.KafkaDataSource
		useMock := true
		
		if useMock {
			// Use mock data source for mock mode test
			dataSource = &MockDataSource{}
		} else {
			dataSource = &kafds.KafkaDataSourceKaf{}
		}
		
		// Verify the data source can be initialized
		dataSource.Init(testConfig)
		
		// Test basic functionality
		_, err := dataSource.GetTopics()
		if err != nil {
			t.Errorf("Mock data source should not fail: %v", err)
		}
	})

	t.Run("Real Mode Init Flow", func(t *testing.T) {
		if os.Getenv("KAFUI_E2E_TEST") != "true" {
			t.Skip("Skipping real mode test - set KAFUI_E2E_TEST=true to run")
		}

		// Test the data source selection logic for real Kafka
		var dataSource api.KafkaDataSource
		useMock := false
		
		if !useMock {
			dataSource = &kafds.KafkaDataSourceKaf{}
		}
		
		// Verify the data source can be initialized
		dataSource.Init(testConfig)
		
		// Test basic functionality
		topics, err := dataSource.GetTopics()
		if err != nil {
			t.Errorf("Real Kafka data source initialization failed: %v", err)
		} else {
			t.Logf("Successfully connected to real Kafka, found %d topics", len(topics))
		}
	})
}

// Helper function to create test configuration for E2E tests
func createE2ETestConfig(t *testing.T) string {
	content := `current-cluster: test-local
clusters:
- name: test-local
  brokers:
  - localhost:9092
  schema-registry-url: http://localhost:8085
`
	tmpFile, err := os.CreateTemp("", "kafui-e2e-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp E2E config file: %v", err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp E2E config file: %v", err)
	}

	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp E2E config file: %v", err)
	}

	return tmpFile.Name()
}