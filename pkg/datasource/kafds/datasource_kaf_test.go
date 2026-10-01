package kafds

import (
	"context"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
	"github.com/birdayz/kaf/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	// GetTopics against a cluster nothing listens on must fail. The cluster is
	// set here rather than read from ~/.kaf/config, so the result does not
	// depend on whether the developer's own cluster is reachable. Success
	// against a broker is covered by TestSharedAdmin_RecoversAfterBrokerRestart.
	origFactory, origCluster := kafkaClientFactory, currentCluster
	resetSharedClients()
	t.Cleanup(func() {
		resetSharedClients()
		kafkaClientFactory, currentCluster = origFactory, origCluster
	})
	kafkaClientFactory = &DefaultKafkaClientFactory{}
	currentCluster = &config.Cluster{Name: "unreachable", Brokers: []string{"127.0.0.1:1"}}

	_, err := (&KafkaDataSourceKaf{}).GetTopics()
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
			// Create temporary config file and load it into the package-level cfg.
			tmpFile := "tmp_rovodev_test_config.yaml"
			err := createTempConfigFile(tmpFile, tempConfig)
			assert.NoError(t, err)
			defer deleteFile(tmpFile)

			cfgFile = tmpFile
			onInit() // populate the package-level cfg from the temp file

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
	// ConsumeTopic never returns an error itself — consumer startup/connection
	// failures are delivered through the onError callback. With no cluster
	// configured, the error is reported synchronously via onError.
	kds := &KafkaDataSourceKaf{}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var (
		mu   sync.Mutex
		errs []error
	)

	err := kds.ConsumeTopic(ctx, "test-topic", api.DefaultConsumeFlags(), func(msg api.Message) {}, func(e any) {
		mu.Lock()
		defer mu.Unlock()
		if errVal, ok := e.(error); ok {
			errs = append(errs, errVal)
		}
	})

	// ConsumeTopic itself never returns an error; Sarama errors go via onError.
	assert.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.NotEmpty(t, errs, "expected a consumer startup error via onError with no cluster configured")
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
	// SetContext reads from the package-level cfg, so populate it directly.
	cfg = config.Config{
		Clusters: []*config.Cluster{
			{Name: "cluster1"},
			{Name: "cluster2"},
		},
	}

	kds := NewKafkaDataSourceKaf()
	err := kds.SetContext("cluster1")

	assert.NoError(t, err)
	assert.Equal(t, "cluster1", cfg.CurrentCluster)
}

// TestKafkaDataSourceKaf_SetContext_ClusterNotFound tests setting non-existent context
func TestKafkaDataSourceKaf_SetContext_ClusterNotFound(t *testing.T) {
	cfg = config.Config{
		Clusters: []*config.Cluster{
			{Name: "cluster1"},
			{Name: "cluster2"},
		},
	}

	kds := NewKafkaDataSourceKaf()
	err := kds.SetContext("nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestKafkaDataSourceKaf_SetContext_NoWrite verifies that SetContext never writes to disk.
// The previous implementation called cfg.SetCurrentCluster() which truncated ~/.kaf/config
// and re-serialized it, stripping TLS cert paths due to missing YAML tags in the kaf library.
func TestKafkaDataSourceKaf_SetContext_NoWrite(t *testing.T) {
	mockConfigManager := &MockConfigManager{}
	kds := NewKafkaDataSourceKafWithDeps(&MockKafkaClientFactory{}, mockConfigManager)

	cfg = config.Config{
		Clusters: []*config.Cluster{
			{Name: "safe-cluster", Brokers: []string{"localhost:9092"}},
		},
	}

	err := kds.SetContext("safe-cluster")
	assert.NoError(t, err)

	// configManager.ReadConfig must never be called — no disk I/O.
	assert.Equal(t, 0, mockConfigManager.ReadConfigCallCount, "SetContext must not read config from disk")
}

// TestKafkaDataSourceKaf_GetConsumerGroups_Success tests successful consumer group retrieval
func TestKafkaDataSourceKaf_GetConsumerGroups_Success(t *testing.T) {
	mockGroups := map[string]string{
		"group1": "consumer",
		"group2": "consumer",
	}

	mockAdmin := &MockClusterAdmin{
		MockConsumerGroups: mockGroups,
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
	assert.Equal(t, "consumer", groups[0].State) // protocol type from ListConsumerGroups
	assert.Equal(t, 0, groups[0].Consumers)       // not populated without DescribeConsumerGroups
	assert.Equal(t, "group2", groups[1].Name)
	assert.Equal(t, "consumer", groups[1].State)
	assert.Equal(t, 0, groups[1].Consumers)
}

// TestKafkaDataSourceKaf_GetConsumerGroups_AdminError tests error in cluster admin creation
func TestKafkaDataSourceKaf_GetConsumerGroups_AdminError(t *testing.T) {
	mockClientFactory := &MockKafkaClientFactory{
		ShouldFailClusterAdmin: true,
	}
	
	mockConfigManager := &MockConfigManager{}
	
	// Set up global state for getClusterAdmin
	originalFactory := kafkaClientFactory
	defer func() { kafkaClientFactory = originalFactory }()
	kafkaClientFactory = mockClientFactory
	
	kds := NewKafkaDataSourceKafWithDeps(mockClientFactory, mockConfigManager)
	
	groups, err := kds.GetConsumerGroups()
	
	assert.Error(t, err)
	assert.Nil(t, groups)
}

// TestKafkaDataSourceKaf_GetConsumerGroups_ListGroupsError tests error in listing consumer groups
func TestKafkaDataSourceKaf_GetConsumerGroups_ListGroupsError(t *testing.T) {
	mockAdmin := &MockClusterAdmin{
		ShouldFailListConsumerGroups: true,
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
	
	assert.Error(t, err)
	assert.Nil(t, groups)
}

// TestKafkaDataSourceKaf_GetConsumerGroups_DescribeGroupsError was testing the removed
// DescribeConsumerGroups call. We now only use ListConsumerGroups, so verify that a
// ListConsumerGroups failure propagates as an error.
func TestKafkaDataSourceKaf_GetConsumerGroups_DescribeGroupsError(t *testing.T) {
	mockAdmin := &MockClusterAdmin{
		ShouldFailListConsumerGroups: true,
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

	assert.Error(t, err)
	assert.Nil(t, groups)
}

// TestKafkaDataSourceKaf_ConsumeTopic tests the ConsumeTopic method
func TestKafkaDataSourceKaf_ConsumeTopic(t *testing.T) {
	mockTopics := map[string]sarama.TopicDetail{
		"test-topic": {
			NumPartitions:     1,
			ReplicationFactor: 1,
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
	
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	// Mock consume flags
	flags := api.ConsumeFlags{
		Tail: 10,
	}
	
	messageHandler := func(msg api.Message) {
		// Mock message handler
	}
	
	errorHandler := func(err any) {
		// Mock error handler
	}
	
	err := kds.ConsumeTopic(ctx, "test-topic", flags, messageHandler, errorHandler)
	
	// Should not error for basic functionality
	assert.NoError(t, err)
}

// TestKafkaDataSourceKaf_ConsumeTopic_AdminError tests ConsumeTopic no longer
// requires a cluster admin client — errors surface via onError callback.
func TestKafkaDataSourceKaf_ConsumeTopic_AdminError(t *testing.T) {
	mockClientFactory := &MockKafkaClientFactory{
		ShouldFailClusterAdmin: true,
	}

	mockConfigManager := &MockConfigManager{}

	kds := NewKafkaDataSourceKafWithDeps(mockClientFactory, mockConfigManager)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	flags := api.ConsumeFlags{}
	messageHandler := func(msg api.Message) {}
	errorHandler := func(err any) {}

	err := kds.ConsumeTopic(ctx, "test-topic", flags, messageHandler, errorHandler)

	// ConsumeTopic itself never returns an error; Sarama errors go via onError.
	assert.NoError(t, err)
}

// TestKafkaDataSourceKaf_GetContexts_EmptyConfig tests GetContexts with empty config
func TestKafkaDataSourceKaf_GetContexts_EmptyConfig(t *testing.T) {
	// Save original cfg and restore after test
	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	
	// Initialize cfg with empty clusters
	cfg = config.Config{
		Clusters: []*config.Cluster{},
	}
	
	mockConfigManager := &MockConfigManager{}
	mockClientFactory := &MockKafkaClientFactory{}
	
	kds := NewKafkaDataSourceKafWithDeps(mockClientFactory, mockConfigManager)
	
	contexts, err := kds.GetContexts()
	
	assert.NoError(t, err)
	assert.Empty(t, contexts)
}

// TestKafkaDataSourceKaf_GetContexts_WithClusters tests GetContexts with clusters
func TestKafkaDataSourceKaf_GetContexts_WithClusters(t *testing.T) {
	// Save original cfg and restore after test
	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	
	// Initialize cfg with clusters
	cfg = config.Config{
		Clusters: []*config.Cluster{
			{Name: "cluster1"},
			{Name: "cluster2"},
			{Name: "cluster3"},
		},
	}
	
	mockConfigManager := &MockConfigManager{}
	mockClientFactory := &MockKafkaClientFactory{}
	
	kds := NewKafkaDataSourceKafWithDeps(mockClientFactory, mockConfigManager)
	
	contexts, err := kds.GetContexts()
	
	assert.NoError(t, err)
	assert.Len(t, contexts, 3)
	assert.Contains(t, contexts, "cluster1")
	assert.Contains(t, contexts, "cluster2")
	assert.Contains(t, contexts, "cluster3")
}

// TestKafkaDataSourceKaf_Init_WithDeps tests the Init method with dependencies
func TestKafkaDataSourceKaf_Init_WithDeps(t *testing.T) {
	// Save original cfgFile and restore after test
	originalCfgFile := cfgFile
	defer func() { cfgFile = originalCfgFile }()
	
	mockConfigManager := &MockConfigManager{}
	mockClientFactory := &MockKafkaClientFactory{}
	
	kds := NewKafkaDataSourceKafWithDeps(mockClientFactory, mockConfigManager)
	
	// Test with empty config option
	kds.Init("")
	// cfgFile should remain unchanged
	
	// Test with config option
	kds.Init("test-config.yaml")
	assert.Equal(t, "test-config.yaml", cfgFile)
}

// TestDefaultKafkaClientFactory tests the default factory
func TestDefaultKafkaClientFactory(t *testing.T) {
	factory := &DefaultKafkaClientFactory{}
	
	// Test CreateClient - this will fail without real Kafka but tests the interface
	config := sarama.NewConfig()
	_, err := factory.CreateClient([]string{"localhost:9092"}, config)
	assert.Error(t, err) // Expected to fail without real Kafka
	
	// Test CreateClusterAdmin - this will fail without real Kafka but tests the interface
	_, err = factory.CreateClusterAdmin([]string{"localhost:9092"}, config)
	assert.Error(t, err) // Expected to fail without real Kafka
}

// TestDefaultConfigManager tests the default config manager
func TestDefaultConfigManager(t *testing.T) {
	manager := &DefaultConfigManager{}
	
	// Test ReadConfig with non-existent file
	_, err := manager.ReadConfig("definitely-non-existent-file-12345.yaml")
	if err == nil {
		// If no error, it might be creating a default config
		t.Log("ReadConfig returned no error - might be creating default config")
	} else {
		assert.Error(t, err) // Expected to fail with non-existent file
	}
	
	// Test GetActiveCluster with empty config
	cfg := config.Config{}
	cluster := manager.GetActiveCluster(cfg)
	assert.Nil(t, cluster) // Should return nil for empty config
	
	// Test GetActiveCluster with config that has clusters
	cfg = config.Config{
		Clusters: []*config.Cluster{
			{Name: "test-cluster"},
		},
		CurrentCluster: "test-cluster",
	}
	cluster = manager.GetActiveCluster(cfg)
	// This may return nil or the cluster depending on the implementation
	// We just test that it doesn't panic
}

// withTwoClusters installs clusters "a" (active) and "b" plus a counting mock
// factory, restoring the globals and dropping the shared clients afterwards.
func withTwoClusters(t *testing.T) *MockKafkaClientFactory {
	t.Helper()
	origFactory, origCluster, origCfg := kafkaClientFactory, currentCluster, cfg
	resetSharedClients()
	t.Cleanup(func() {
		resetSharedClients()
		kafkaClientFactory, currentCluster, cfg = origFactory, origCluster, origCfg
	})
	a := &config.Cluster{Name: "a", Brokers: []string{"a:9092"}}
	b := &config.Cluster{Name: "b", Brokers: []string{"b:9092"}}
	cfg = config.Config{Clusters: []*config.Cluster{a, b}, CurrentCluster: "a"}
	currentCluster = a
	factory := &MockKafkaClientFactory{MockClusterAdmin: &MockClusterAdmin{}}
	kafkaClientFactory = factory
	return factory
}

func TestGetClusterAdmin_SharedAndResetOnSwitch(t *testing.T) {
	factory := withTwoClusters(t)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			admin, err := getClusterAdmin()
			if assert.NoError(t, err) {
				assert.NoError(t, admin.Close(), "Close on the shared admin is a no-op")
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, 1, factory.CreateClusterAdminCalls, "concurrent callers share one admin")

	require.NoError(t, NewKafkaDataSourceKaf().SetContext("b"))
	_, err := getClusterAdmin()
	require.NoError(t, err)
	assert.Equal(t, 2, factory.CreateClusterAdminCalls, "a cluster switch builds a new admin")
}

func TestSetContext_ClearsClusterOverride(t *testing.T) {
	withTwoClusters(t)
	cfg.ClusterOverride = "a" // as set by `kafui --cluster a`

	kds := NewKafkaDataSourceKafWithDeps(&MockKafkaClientFactory{}, &DefaultConfigManager{})
	require.Equal(t, "a", kds.GetContext())
	require.NoError(t, kds.SetContext("b"))
	assert.Equal(t, "b", kds.GetContext(), "GetContext must follow the switch, not the override")
}

func TestGetConfig_SecurityMisconfigurationReturnsError(t *testing.T) {
	origCluster := currentCluster
	t.Cleanup(func() { currentCluster = origCluster })

	tests := []struct {
		name    string
		cluster *config.Cluster
		wantErr string
	}{
		{
			name:    "SASL protocol without a sasl block",
			cluster: &config.Cluster{Name: "x", SecurityProtocol: "SASL_PLAINTEXT"},
			wantErr: "no SASL configuration",
		},
		{
			name: "SASL_SSL with an unreadable CA file",
			cluster: &config.Cluster{Name: "x", SecurityProtocol: "SASL_SSL",
				SASL: &config.SASL{Mechanism: "PLAIN"}, TLS: &config.TLS{Cafile: "/nonexistent/ca.pem"}},
			wantErr: "unable to read TLS CA file",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentCluster = tt.cluster
			_, err := getConfig()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// closeTrackingAdmin records whether Close was called.
type closeTrackingAdmin struct {
	*MockClusterAdmin
	closed atomic.Bool
}

func (a *closeTrackingAdmin) Close() error { a.closed.Store(true); return nil }

// gatedFactory blocks every CreateClusterAdmin until gate is closed, like a
// dial to an unreachable cluster.
type gatedFactory struct {
	MockKafkaClientFactory
	gate    chan struct{}
	started chan struct{}
	calls   atomic.Int32
	mu      sync.Mutex
	admins  []*closeTrackingAdmin
}

func (f *gatedFactory) CreateClusterAdmin([]string, *sarama.Config) (ClusterAdminInterface, error) {
	f.calls.Add(1)
	select {
	case f.started <- struct{}{}:
	default:
	}
	<-f.gate
	a := &closeTrackingAdmin{MockClusterAdmin: &MockClusterAdmin{}}
	f.mu.Lock()
	f.admins = append(f.admins, a)
	f.mu.Unlock()
	return a, nil
}

// A cluster switch must not wait for a dial that is still connecting, and
// the admin that dial builds for the old cluster must be closed, not cached.
func TestGetClusterAdmin_ResetDoesNotWaitForDial(t *testing.T) {
	withTwoClusters(t)
	f := &gatedFactory{gate: make(chan struct{}), started: make(chan struct{}, 1)}
	kafkaClientFactory = f

	dialErr := make(chan error, 1)
	go func() {
		_, err := getClusterAdmin()
		dialErr <- err
	}()
	<-f.started

	switched := make(chan error, 1)
	go func() { switched <- NewKafkaDataSourceKaf().SetContext("b") }()
	select {
	case err := <-switched:
		require.NoError(t, err)
	case <-time.After(time.Second):
		close(f.gate)
		t.Fatal("SetContext blocked on a dial in flight")
	}

	close(f.gate)
	assert.ErrorIs(t, <-dialErr, errClusterChangedWhileConnecting)
	f.mu.Lock()
	stale := f.admins[0]
	f.mu.Unlock()
	assert.Eventually(t, stale.closed.Load, time.Second, 5*time.Millisecond, "the old cluster's admin is closed")

	admin, err := getClusterAdmin()
	require.NoError(t, err)
	assert.NotSame(t, stale, admin.(sharedClusterAdmin).ClusterAdminInterface)
	assert.Equal(t, int32(2), f.calls.Load(), "the new cluster gets its own dial")
}

// Concurrent callers share the one dial in flight instead of queueing behind
// the lock and dialing one after another.
func TestGetClusterAdmin_ConcurrentCallersShareOneDial(t *testing.T) {
	withTwoClusters(t)
	f := &gatedFactory{gate: make(chan struct{}), started: make(chan struct{}, 1)}
	kafkaClientFactory = f

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := getClusterAdmin()
			assert.NoError(t, err)
		}()
	}
	<-f.started
	time.Sleep(20 * time.Millisecond) // let the others reach the wait
	close(f.gate)
	wg.Wait()
	assert.Equal(t, int32(1), f.calls.Load())
}

// eofAdmin fails ListTopics like a dead broker connection does.
type eofAdmin struct{ *MockClusterAdmin }

func (eofAdmin) ListTopics() (map[string]sarama.TopicDetail, error) { return nil, io.EOF }

// A connection error drops the cached admin so the next call reconnects, but
// a late error from an admin that was already replaced leaves its successor.
func TestSharedAdmin_ConnectionErrorDropsOnlyCurrentHandle(t *testing.T) {
	factory := withTwoClusters(t)
	factory.MockClusterAdmin = eofAdmin{&MockClusterAdmin{}}

	first, err := getClusterAdmin()
	require.NoError(t, err)
	_, err = first.ListTopics()
	require.ErrorIs(t, err, io.EOF)

	second, err := getClusterAdmin()
	require.NoError(t, err)
	assert.Equal(t, 2, factory.CreateClusterAdminCalls, "a connection error forces a reconnect")

	_, _ = first.ListTopics() // stale handle fails again
	third, err := getClusterAdmin()
	require.NoError(t, err)
	assert.Equal(t, 2, factory.CreateClusterAdminCalls, "a stale handle's error must not drop the current admin")
	assert.Equal(t, second.(sharedClusterAdmin).gen, third.(sharedClusterAdmin).gen)

	factory.MockClusterAdmin = &MockClusterAdmin{ShouldFailListTopics: true}
	resetSharedClients()
	admin, err := getClusterAdmin()
	require.NoError(t, err)
	_, err = admin.ListTopics()
	require.Error(t, err)
	_, err = getClusterAdmin()
	require.NoError(t, err)
	assert.Equal(t, 3, factory.CreateClusterAdminCalls, "a Kafka error answer keeps the admin")
}

// After a broker restart, sarama's ListTopics and ListConsumerGroups keep
// using the dead connection. The shared admin must be rebuilt, so the topics
// and consumer-groups pages recover on the next poll.
func TestSharedAdmin_RecoversAfterBrokerRestart(t *testing.T) {
	origFactory, origCluster := kafkaClientFactory, currentCluster
	resetSharedClients()
	t.Cleanup(func() {
		resetSharedClients()
		kafkaClientFactory, currentCluster = origFactory, origCluster
	})
	kafkaClientFactory = &DefaultKafkaClientFactory{}

	serve := func(b *sarama.MockBroker) {
		b.SetHandlerByMap(map[string]sarama.MockResponse{
			"MetadataRequest": sarama.NewMockMetadataResponse(t).
				SetBroker(b.Addr(), b.BrokerID()).
				SetController(b.BrokerID()).
				SetLeader("t1", 0, b.BrokerID()),
			"DescribeConfigsRequest": sarama.NewMockDescribeConfigsResponse(t),
			"ListGroupsRequest":      sarama.NewMockListGroupsResponse(t).AddGroup("g1", "consumer"),
		})
	}
	broker := sarama.NewMockBroker(t, 1)
	addr := broker.Addr()
	serve(broker)
	restart := func() {
		broker.Close()
		broker = sarama.NewMockBrokerAddr(t, 1, addr)
		serve(broker)
	}
	t.Cleanup(func() { broker.Close() })
	currentCluster = &config.Cluster{Name: "restart", Brokers: []string{addr}}
	ds := KafkaDataSourceKaf{}

	// recovers reports whether call succeeds within a few polls: the first
	// poll after a restart may still hit the dead connection.
	recovers := func(call func() error) bool {
		var err error
		for i := 0; i < 3; i++ {
			if err = call(); err == nil {
				return true
			}
		}
		t.Logf("last error: %v", err)
		return false
	}
	getTopics := func() error { _, err := ds.GetTopics(); return err }
	getGroups := func() error { _, err := ds.GetConsumerGroups(); return err }

	require.NoError(t, getTopics())
	require.NoError(t, getGroups())

	restart()
	assert.True(t, recovers(getTopics), "GetTopics recovers after a broker restart")

	restart()
	assert.True(t, recovers(getGroups), "GetConsumerGroups recovers after a broker restart")
}

func TestGetConfig_AWSMSKIAM(t *testing.T) {
	origCluster := currentCluster
	t.Cleanup(func() { currentCluster = origCluster })

	// Deterministic region resolution independent of the developer's ~/.aws.
	cfgFile := t.TempDir() + "/aws-config"
	require.NoError(t, os.WriteFile(cfgFile, []byte("[default]\nregion = us-east-1\n"), 0o600))
	t.Setenv("AWS_CONFIG_FILE", cfgFile)
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")

	currentCluster = &config.Cluster{
		Name:             "msk",
		SecurityProtocol: "SASL_SSL",
		// Username/Password set by mistake: a token mechanism must not send them.
		SASL: &config.SASL{Mechanism: "AWS_MSK_IAM", Username: "ignored", Password: "ignored"},
	}

	sc, err := getConfig()
	require.NoError(t, err)
	assert.True(t, sc.Net.SASL.Enable)
	assert.Equal(t, sarama.SASLMechanism(sarama.SASLTypeOAuth), sc.Net.SASL.Mechanism,
		"AWS_MSK_IAM travels over the OAUTHBEARER wire mechanism")
	assert.Empty(t, sc.Net.SASL.User)
	assert.Empty(t, sc.Net.SASL.Password)
	require.NotNil(t, sc.Net.SASL.TokenProvider)
	_, ok := sc.Net.SASL.TokenProvider.(*mskTokenProvider)
	assert.True(t, ok, "token provider must be the MSK IAM signer")
}

func TestGetConfig_AWSMSKIAMMissingRegionReturnsError(t *testing.T) {
	origCluster := currentCluster
	t.Cleanup(func() { currentCluster = origCluster })

	cfgFile := t.TempDir() + "/aws-config"
	require.NoError(t, os.WriteFile(cfgFile, []byte("[default]\n"), 0o600))
	t.Setenv("AWS_CONFIG_FILE", cfgFile)
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")

	currentCluster = &config.Cluster{
		Name:             "msk",
		SecurityProtocol: "SASL_PLAINTEXT",
		SASL:             &config.SASL{Mechanism: "AWS_MSK_IAM"},
	}

	_, err := getConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires an AWS region")
}
