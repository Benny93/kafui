package kafds

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
	"github.com/birdayz/kaf/pkg/config"
	"github.com/stretchr/testify/assert"
)

// Simple tests that focus on basic functionality without complex mocking

func TestKafkaDataSourceKaf_Init(t *testing.T) {
	tests := []struct {
		name      string
		cfgOption string
	}{
		{"init with empty config", ""},
		{"init with config file", "/path/to/config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kds := &KafkaDataSourceKaf{}
			// This should not panic
			kds.Init(tt.cfgOption)
			if tt.cfgOption != "" {
				assert.Equal(t, tt.cfgOption, cfgFile)
			}
		})
	}
}

func TestKafkaDataSourceKaf_GetTopics_Integration(t *testing.T) {
	// This is an integration test that would require a real Kafka connection
	// For now, we'll just test that the method exists and can be called
	kds := &KafkaDataSourceKaf{}

	// This will likely fail due to no Kafka connection, but tests the method signature
	_, err := kds.GetTopics()
	// We expect an error since there's no real Kafka cluster
	assert.Error(t, err)
}

func TestKafkaDataSourceKaf_GetContexts(t *testing.T) {
	// Save original cfg and restore after test
	originalCfg := cfg
	defer func() { cfg = originalCfg }()

	// Test with empty clusters
	cfg.Clusters = []*config.Cluster{}
	kds := &KafkaDataSourceKaf{}
	contexts, err := kds.GetContexts()
	assert.NoError(t, err)
	assert.Empty(t, contexts)

	// Test with some clusters
	cfg.Clusters = []*config.Cluster{
		{Name: "cluster1"},
		{Name: "cluster2"},
	}
	contexts, err = kds.GetContexts()
	assert.NoError(t, err)
	assert.Len(t, contexts, 2)
	assert.Contains(t, contexts, "cluster1")
	assert.Contains(t, contexts, "cluster2")
}

func TestKafkaDataSourceKaf_GetContext(t *testing.T) {
	// Save original cfg and restore after test
	originalCfg := cfg
	defer func() { cfg = originalCfg }()

	// Initialize cfg for the test with proper clusters slice
	cfg = config.Config{
		Clusters: []*config.Cluster{},
	}

	// Test when no active cluster
	mockConfigManager := &MockConfigManager{
		MockActiveCluster: nil,
	}
	mockClientFactory := &MockKafkaClientFactory{}
	kds := NewKafkaDataSourceKafWithDeps(mockClientFactory, mockConfigManager)

	context := kds.GetContext()
	assert.Equal(t, "default localhost:9092", context)

	// Test when active cluster exists
	mockCluster := &config.Cluster{Name: "test-cluster"}
	mockConfigManager.MockActiveCluster = mockCluster
	kds = NewKafkaDataSourceKafWithDeps(mockClientFactory, mockConfigManager)

	context = kds.GetContext()
	assert.Equal(t, "test-cluster", context)
}

func TestKafkaDataSourceKaf_SetContext_Legacy(t *testing.T) {
	// Create a temporary config file for testing
	tempConfig := `
clusters:
  - name: test-cluster
    brokers: ["localhost:9092"]
  - name: prod-cluster
    brokers: ["prod:9092"]
currentCluster: test-cluster
`

	tests := []struct {
		name        string
		contextName string
		expectError bool
	}{
		{"valid context", "test-cluster", false},
		{"another valid context", "prod-cluster", false},
		{"invalid context", "nonexistent-cluster", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpFile := "tmp_rovodev_test_config.yaml"
			err := createTempConfigFile(tmpFile, tempConfig)
			assert.NoError(t, err)
			defer deleteFile(tmpFile)

			cfgFile = tmpFile

			// Use the new constructor with real config manager for this integration test
			kds := NewKafkaDataSourceKaf()
			err = kds.SetContext(tt.contextName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestKafkaDataSourceKaf_GetConsumerGroups_Integration(t *testing.T) {
	// Integration test - will fail without real Kafka but tests method signature
	kds := &KafkaDataSourceKaf{}
	_, err := kds.GetConsumerGroups()
	// We expect an error since there's no real Kafka cluster
	assert.Error(t, err)
}

func TestKafkaDataSourceKaf_ConsumeTopic_Integration(t *testing.T) {
	// Integration test - will fail without real Kafka but tests method signature
	kds := &KafkaDataSourceKaf{}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := kds.ConsumeTopic(ctx, "test-topic", api.DefaultConsumeFlags(), func(msg api.Message) {}, func(err any) {})
	// We expect an error since there's no real Kafka cluster
	assert.Error(t, err)
}

// Helper functions for testing
func createTempConfigFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

func deleteFile(filename string) {
	os.Remove(filename)
}

// TestNewKafkaDataSourceKaf tests the constructor
func TestNewKafkaDataSourceKaf(t *testing.T) {
	kds := NewKafkaDataSourceKaf()
	assert.NotNil(t, kds)
	assert.NotNil(t, kds.clientFactory)
	assert.NotNil(t, kds.configManager)
}

// TestNewKafkaDataSourceKafWithDeps tests the constructor with dependencies
func TestNewKafkaDataSourceKafWithDeps(t *testing.T) {
	mockClientFactory := &MockKafkaClientFactory{}
	mockConfigManager := &MockConfigManager{}

	kds := NewKafkaDataSourceKafWithDeps(mockClientFactory, mockConfigManager)
	assert.NotNil(t, kds)
	assert.Equal(t, mockClientFactory, kds.clientFactory)
	assert.Equal(t, mockConfigManager, kds.configManager)
}

// TestKafkaDataSourceKaf_GetTopics_Success tests successful topic retrieval
func TestKafkaDataSourceKaf_GetTopics_Success(t *testing.T) {
	mockTopics := map[string]sarama.TopicDetail{
		"topic1": {
			NumPartitions:     3,
			ReplicationFactor: 2,
			ReplicaAssignment: map[int32][]int32{},
			ConfigEntries:     map[string]*string{},
		},
		"topic2": {
			NumPartitions:     1,
			ReplicationFactor: 1,
			ReplicaAssignment: map[int32][]int32{},
			ConfigEntries:     map[string]*string{},
		},
	}

	mockAdmin := &MockClusterAdmin{
		MockTopics: mockTopics,
	}

	mockClientFactory := &MockKafkaClientFactory{
		MockClusterAdmin: mockAdmin,
	}

	mockConfigManager := &MockConfigManager{}

	// Set up global state for getClusterAdmin
	originalFactory := kafkaClientFactory
	defer func() { kafkaClientFactory = originalFactory }()
	kafkaClientFactory = mockClientFactory

	kds := NewKafkaDataSourceKafWithDeps(mockClientFactory, mockConfigManager)

	topics, err := kds.GetTopics()

	assert.NoError(t, err)
	assert.Len(t, topics, 2)
	assert.Contains(t, topics, "topic1")
	assert.Contains(t, topics, "topic2")
	assert.Equal(t, int32(3), topics["topic1"].NumPartitions)
	assert.Equal(t, int16(2), topics["topic1"].ReplicationFactor)
}

// TestKafkaDataSourceKaf_GetTopics_AdminError tests error in cluster admin creation
func TestKafkaDataSourceKaf_GetTopics_AdminError(t *testing.T) {
	mockClientFactory := &MockKafkaClientFactory{
		ShouldFailClusterAdmin: true,
	}

	mockConfigManager := &MockConfigManager{}

	// Set up global state for getClusterAdmin
	originalFactory := kafkaClientFactory
	defer func() { kafkaClientFactory = originalFactory }()
	kafkaClientFactory = mockClientFactory

	kds := NewKafkaDataSourceKafWithDeps(mockClientFactory, mockConfigManager)

	topics, err := kds.GetTopics()

	assert.Error(t, err)
	assert.Nil(t, topics)
	assert.Contains(t, err.Error(), "mock cluster admin creation failed")
}

// TestKafkaDataSourceKaf_GetTopics_ListTopicsError tests error in listing topics
func TestKafkaDataSourceKaf_GetTopics_ListTopicsError(t *testing.T) {
	mockAdmin := &MockClusterAdmin{
		ShouldFailListTopics: true,
	}

	mockClientFactory := &MockKafkaClientFactory{
		MockClusterAdmin: mockAdmin,
	}

	mockConfigManager := &MockConfigManager{}

	// Set up global state for getClusterAdmin
	originalFactory := kafkaClientFactory
	defer func() { kafkaClientFactory = originalFactory }()
	kafkaClientFactory = mockClientFactory

	kds := NewKafkaDataSourceKafWithDeps(mockClientFactory, mockConfigManager)

	topics, err := kds.GetTopics()

	assert.Error(t, err)
	assert.Nil(t, topics)
	assert.Contains(t, err.Error(), "mock list topics failed")
}

// TestKafkaDataSourceKaf_GetContext_WithActiveCluster tests context retrieval with active cluster
func TestKafkaDataSourceKaf_GetContext_WithActiveCluster(t *testing.T) {
	// Save original cfg and restore after test
	originalCfg := cfg
	defer func() { cfg = originalCfg }()

	// Initialize cfg for the test with proper clusters slice
	cfg = config.Config{
		Clusters: []*config.Cluster{},
	}

	mockCluster := &config.Cluster{
		Name: "test-cluster",
	}

	mockConfigManager := &MockConfigManager{
		MockActiveCluster: mockCluster,
	}

	mockClientFactory := &MockKafkaClientFactory{}

	kds := NewKafkaDataSourceKafWithDeps(mockClientFactory, mockConfigManager)

	context := kds.GetContext()

	assert.Equal(t, "test-cluster", context)
}

// TestKafkaDataSourceKaf_GetContext_NoActiveCluster tests context retrieval without active cluster
func TestKafkaDataSourceKaf_GetContext_NoActiveCluster(t *testing.T) {
	// Save original cfg and restore after test
	originalCfg := cfg
	defer func() { cfg = originalCfg }()

	// Initialize cfg for the test with proper clusters slice
	cfg = config.Config{
		Clusters: []*config.Cluster{},
	}

	mockConfigManager := &MockConfigManager{
		MockActiveCluster: nil,
	}

	mockClientFactory := &MockKafkaClientFactory{}

	kds := NewKafkaDataSourceKafWithDeps(mockClientFactory, mockConfigManager)

	context := kds.GetContext()

	assert.Equal(t, "default localhost:9092", context)
}

// TestKafkaDataSourceKaf_SetContext_Success tests successful context setting
func TestKafkaDataSourceKaf_SetContext_Success(t *testing.T) {
	mockConfig := config.Config{
		Clusters: []*config.Cluster{
			{Name: "cluster1"},
			{Name: "cluster2"},
		},
	}

	mockConfigManager := &MockConfigManager{
		MockConfig: mockConfig,
	}

	mockClientFactory := &MockKafkaClientFactory{}

	kds := NewKafkaDataSourceKafWithDeps(mockClientFactory, mockConfigManager)

	err := kds.SetContext("cluster1")

	assert.NoError(t, err)
}

// TestKafkaDataSourceKaf_SetContext_ClusterNotFound tests setting non-existent context
func TestKafkaDataSourceKaf_SetContext_ClusterNotFound(t *testing.T) {
	mockConfig := config.Config{
		Clusters: []*config.Cluster{
			{Name: "cluster1"},
			{Name: "cluster2"},
		},
	}

	mockConfigManager := &MockConfigManager{
		MockConfig: mockConfig,
	}

	mockClientFactory := &MockKafkaClientFactory{}

	kds := NewKafkaDataSourceKafWithDeps(mockClientFactory, mockConfigManager)

	err := kds.SetContext("nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestKafkaDataSourceKaf_SetContext_ConfigReadError tests config read error
func TestKafkaDataSourceKaf_SetContext_ConfigReadError(t *testing.T) {
	mockConfigManager := &MockConfigManager{
		ShouldFailReadConfig: true,
	}

	mockClientFactory := &MockKafkaClientFactory{}

	kds := NewKafkaDataSourceKafWithDeps(mockClientFactory, mockConfigManager)

	err := kds.SetContext("any-cluster")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock config read failed")
}

// TestKafkaDataSourceKaf_GetConsumerGroups_Success tests successful consumer group retrieval
func TestKafkaDataSourceKaf_GetConsumerGroups_Success(t *testing.T) {
	mockGroups := map[string]string{
		"group1": "consumer",
		"group2": "consumer",
	}

	mockGroupDescs := []*sarama.GroupDescription{
		{
			GroupId: "group1",
			State:   "Stable",
			Members: map[string]*sarama.GroupMemberDescription{
				"member1": {MemberId: "member1"},
			},
		},
		{
			GroupId: "group2",
			State:   "Empty",
			Members: map[string]*sarama.GroupMemberDescription{},
		},
	}

	mockAdmin := &MockClusterAdmin{
		MockConsumerGroups:    mockGroups,
		MockGroupDescriptions: mockGroupDescs,
	}

	mockClientFactory := &MockKafkaClientFactory{
		MockClusterAdmin: mockAdmin,
	}

	mockConfigManager := &MockConfigManager{}

	// Set up global state for getClusterAdmin
	originalFactory := kafkaClientFactory
	defer func() { kafkaClientFactory = originalFactory }()
	kafkaClientFactory = mockClientFactory

	kds := NewKafkaDataSourceKafWithDeps(mockClientFactory, mockConfigManager)

	groups, err := kds.GetConsumerGroups()

	assert.NoError(t, err)
	assert.Len(t, groups, 2)
	assert.Equal(t, "group1", groups[0].Name)
	assert.Equal(t, "Stable", groups[0].State)
	assert.Equal(t, 1, groups[0].Consumers)
	assert.Equal(t, "group2", groups[1].Name)
	assert.Equal(t, "Empty", groups[1].State)
	assert.Equal(t, 0, groups[1].Consumers)
}
