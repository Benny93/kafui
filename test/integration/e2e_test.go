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

var _ = (api.KafkaDataSource)(&MockDataSource{})

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

func (m *MockDataSource) ProduceMessage(ctx context.Context, topic string, rec api.ProduceRecord) error {
	return nil
}

func (m *MockDataSource) GetMessageSchemaInfo(keySchemaID, valueSchemaID string) (*api.MessageSchemaInfo, error) {
	return &api.MessageSchemaInfo{}, nil
}

func (m *MockDataSource) GetClusterDetails(clusterName string) (api.ClusterInfo, error) {
	return api.ClusterInfo{Name: clusterName, Brokers: []string{"localhost:9092"}}, nil
}

func (m *MockDataSource) GetTopicMessageCounts(topics map[string]int32) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (m *MockDataSource) GetSchemas() ([]api.Schema, error) {
	return []api.Schema{}, nil
}

func (m *MockDataSource) GetSchemaDetails(subjects []string) ([]api.Schema, error) {
	return []api.Schema{}, nil
}

func (m *MockDataSource) GetSchemaContent(subject string, version int) (string, error) {
	return `{"type":"record","name":"Mock","fields":[]}`, nil
}

func (m *MockDataSource) GetSchemaVersions(subject string) ([]api.SchemaVersion, error) {
	return nil, nil
}
func (m *MockDataSource) GetGlobalCompatibility() (api.CompatibilityLevel, error) { return "", nil }
func (m *MockDataSource) GetSubjectCompatibility(subject string) (api.CompatibilityLevel, bool, error) {
	return "", false, nil
}
func (m *MockDataSource) RegisterSchema(subject, schemaText, schemaType string) (api.Schema, error) {
	return api.Schema{}, nil
}
func (m *MockDataSource) CheckSchemaCompatibility(subject, schemaText, schemaType string) (bool, []string, error) {
	return true, nil, nil
}
func (m *MockDataSource) DeleteSubject(subject string, permanent bool) ([]int, error) {
	return nil, nil
}
func (m *MockDataSource) DeleteSchemaVersion(subject string, version int, permanent bool) error {
	return nil
}
func (m *MockDataSource) SetGlobalCompatibility(level api.CompatibilityLevel) error { return nil }
func (m *MockDataSource) SetSubjectCompatibility(subject string, level api.CompatibilityLevel) error {
	return nil
}

func (m *MockDataSource) GetACLs() ([]api.ACLEntry, error) {
	return []api.ACLEntry{}, nil
}

func (m *MockDataSource) GetACLsFiltered(filter api.ACLFilter) ([]api.ACLEntry, error) {
	return []api.ACLEntry{}, nil
}

func (m *MockDataSource) CreateACL(entry api.ACLEntry) error { return nil }

func (m *MockDataSource) DeleteACL(entry api.ACLEntry) error { return nil }

func (m *MockDataSource) GetClientQuotas() ([]api.ClientQuotaEntry, error) {
	return []api.ClientQuotaEntry{}, nil
}

func (m *MockDataSource) AlterClientQuotas(entity api.ClientQuotaEntity, quotas map[string]float64) error {
	return nil
}

func (m *MockDataSource) GetTopicNames() ([]string, error) {
	topics, err := m.GetTopics()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(topics))
	for name := range topics {
		names = append(names, name)
	}
	return names, nil
}

func (m *MockDataSource) DecodeMessage(_ context.Context, msg api.Message) (api.Message, error) {
	return msg, nil
}

func (m *MockDataSource) ListSerdes() []string { return []string{"string", "hex", "json"} }

func (m *MockDataSource) GetClusterStatistics(_ context.Context, _ string) (api.ClusterStatistics, error) {
	return api.ClusterStatistics{}, nil
}

func (m *MockDataSource) GetClusterCapabilities(_ context.Context, _ string) ([]api.Capability, error) {
	return nil, nil
}

func (m *MockDataSource) ValidateClusterConnection(_ context.Context, _ string) ([]api.ValidationResult, error) {
	return nil, nil
}
func (m *MockDataSource) GetBrokers() ([]api.BrokerInfo, error) { return nil, nil }
func (m *MockDataSource) GetBrokerStats() (map[int32]api.BrokerStats, api.BrokerSummary, error) {
	return nil, api.BrokerSummary{}, nil
}
func (m *MockDataSource) GetBrokerLogDirs(brokerIDs []int32) (map[int32][]api.BrokerLogDir, error) {
	return nil, nil
}
func (m *MockDataSource) GetBrokerConfig(brokerID int32) ([]api.BrokerConfigEntry, error) {
	return nil, nil
}
func (m *MockDataSource) AlterBrokerConfig(brokerID int32, key, value string) error { return nil }
func (m *MockDataSource) AlterReplicaLogDir(brokerID int32, topic string, partition int32, logDir string) error {
	return nil
}
func (m *MockDataSource) GetBrokerMetrics(brokerID int32) (string, error) { return "", nil }

// Topic-administration + analysis stubs (TP-1..TP-11, TP-29/TP-30).
func (m *MockDataSource) GetTopicConfig(topicName string) ([]api.TopicConfigEntry, error) {
	return nil, nil
}
func (m *MockDataSource) GetTopicDetails(topicName string) (api.TopicDetails, error) {
	return api.TopicDetails{}, nil
}
func (m *MockDataSource) GetTopicSizes(topicNames []string) (map[string]int64, error) {
	return nil, nil
}
func (m *MockDataSource) CreateTopic(name string, numPartitions int32, replicationFactor int16, configs map[string]*string) error {
	return nil
}
func (m *MockDataSource) DeleteTopic(name string) error         { return nil }
func (m *MockDataSource) IsTopicDeletionEnabled() (bool, error) { return true, nil }
func (m *MockDataSource) UpdateTopicConfig(name string, entries map[string]*string) error {
	return nil
}
func (m *MockDataSource) IncreasePartitions(name string, totalCount int32) error { return nil }
func (m *MockDataSource) PurgeTopicMessages(name string, partition int32) error  { return nil }
func (m *MockDataSource) RecreateTopic(name string) error                        { return nil }
func (m *MockDataSource) ChangeReplicationFactor(name string, newFactor int16) error {
	return nil
}
func (m *MockDataSource) StartTopicAnalysis(ctx context.Context, topicName string) error {
	return nil
}
func (m *MockDataSource) GetTopicAnalysis(topicName string) (*api.TopicAnalysis, error) {
	return nil, nil
}
func (m *MockDataSource) CancelTopicAnalysis(topicName string) error { return nil }

func (m *MockDataSource) GetConnectClusters(withStats bool) ([]api.ConnectCluster, error) {
	return nil, nil
}
func (m *MockDataSource) GetConnectorNames(connect string) ([]string, error) { return nil, nil }
func (m *MockDataSource) GetConnectors() ([]api.Connector, error)            { return nil, nil }
func (m *MockDataSource) GetConnectorDetails(connect, name string) (api.ConnectorDetails, error) {
	return api.ConnectorDetails{}, nil
}
func (m *MockDataSource) CreateConnector(connect, name string, config map[string]string) (api.Connector, error) {
	return api.Connector{}, nil
}
func (m *MockDataSource) UpdateConnectorConfig(connect, name string, config map[string]string) (api.Connector, error) {
	return api.Connector{}, nil
}
func (m *MockDataSource) DeleteConnector(connect, name string) error            { return nil }
func (m *MockDataSource) PauseConnector(connect, name string) error             { return nil }
func (m *MockDataSource) ResumeConnector(connect, name string) error            { return nil }
func (m *MockDataSource) StopConnector(connect, name string) error              { return nil }
func (m *MockDataSource) RestartConnector(connect, name string) error           { return nil }
func (m *MockDataSource) RestartConnectorTask(connect, name string, taskID int) error {
	return nil
}
func (m *MockDataSource) ResetConnectorOffsets(connect, name string) error { return nil }
func (m *MockDataSource) GetConnectorPlugins(connect string) ([]api.ConnectorPlugin, error) {
	return nil, nil
}
func (m *MockDataSource) ValidateConnectorConfig(connect, pluginClass string, config map[string]string) (api.ConnectorValidationResult, error) {
	return api.ConnectorValidationResult{}, nil
}
func (m *MockDataSource) ListKsqlStreams() ([]api.KsqlStream, error) { return nil, nil }
func (m *MockDataSource) ListKsqlTables() ([]api.KsqlTable, error)   { return nil, nil }
func (m *MockDataSource) ExecuteKsql(ctx context.Context, sql string, props map[string]string) (<-chan api.KsqlResultTable, error) {
	return nil, nil
}
func (m *MockDataSource) GetConsumerGroupDetail(groupID string) (api.ConsumerGroupDetail, error) {
	return api.ConsumerGroupDetail{}, nil
}
func (m *MockDataSource) GetConsumerGroupDetails(groupIDs []string) ([]api.ConsumerGroup, error) {
	return nil, nil
}
func (m *MockDataSource) GetConsumerGroupsForTopic(topic string) ([]api.ConsumerGroup, error) {
	return nil, nil
}
func (m *MockDataSource) DeleteConsumerGroup(groupID string) error               { return nil }
func (m *MockDataSource) DeleteConsumerGroupOffsets(groupID, topic string) error { return nil }
func (m *MockDataSource) ResetConsumerGroupOffsets(ctx context.Context, req api.OffsetResetRequest) error {
	return nil
}

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
