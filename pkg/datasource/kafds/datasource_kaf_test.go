package kafds

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
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

	kds := &KafkaDataSourceKaf{}
	
	// Test when no active cluster
	cfg = config.Config{}
	context := kds.GetContext()
	assert.Equal(t, "default localhost:9092", context)
	
	// Test when active cluster exists
	cfg.CurrentCluster = "test-cluster"
	cfg.Clusters = []*config.Cluster{
		{Name: "test-cluster"},
	}
	context = kds.GetContext()
	assert.Equal(t, "test-cluster", context)
}

func TestKafkaDataSourceKaf_SetContext(t *testing.T) {
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
			kds := &KafkaDataSourceKaf{}
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
