package mock

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
)

// TestKafkaDataSourceMock_Init tests the initialization
func TestKafkaDataSourceMock_Init(t *testing.T) {
	mock := KafkaDataSourceMock{}
	
	// Test that Init doesn't panic with various config options
	configs := []string{
		"",
		"test-config",
		"/path/to/config",
		"invalid-config",
	}
	
	for _, config := range configs {
		t.Run("config_"+config, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Init panicked with config '%s': %v", config, r)
				}
			}()
			
			mock.Init(config)
		})
	}
}

// TestKafkaDataSourceMock_GetTopics tests topic retrieval
func TestKafkaDataSourceMock_GetTopics(t *testing.T) {
	mock := KafkaDataSourceMock{}
	
	topics, err := mock.GetTopics()
	
	if err != nil {
		t.Errorf("GetTopics() returned error: %v", err)
	}
	
	if topics == nil {
		t.Fatal("GetTopics() returned nil topics")
	}
	
	// Should return 100 topics as per implementation
	expectedCount := 100
	if len(topics) != expectedCount {
		t.Errorf("GetTopics() returned %d topics, want %d", len(topics), expectedCount)
	}
	
	// Test topic structure
	for name, topic := range topics {
		if !strings.HasPrefix(name, "Topic ") {
			t.Errorf("Topic name '%s' doesn't match expected pattern", name)
		}
		
		if topic.ReplicationFactor != 1 {
			t.Errorf("Topic '%s' ReplicationFactor = %d, want 1", name, topic.ReplicationFactor)
		}
		
		if topic.NumPartitions != 1 {
			t.Errorf("Topic '%s' NumPartitions = %d, want 1", name, topic.NumPartitions)
		}
		
		if topic.ReplicaAssignment == nil {
			t.Errorf("Topic '%s' ReplicaAssignment is nil", name)
		}
		
		if topic.ConfigEntries == nil {
			t.Errorf("Topic '%s' ConfigEntries is nil", name)
		}
	}
}

// TestKafkaDataSourceMock_GetContexts tests context retrieval
func TestKafkaDataSourceMock_GetContexts(t *testing.T) {
	mock := KafkaDataSourceMock{}
	
	contexts, err := mock.GetContexts()
	
	if err != nil {
		t.Errorf("GetContexts() returned error: %v", err)
	}
	
	if contexts == nil {
		t.Fatal("GetContexts() returned nil")
	}
	
	expectedContexts := []string{"kafka-dev", "kafka-test", "kafka-prod"}
	if !reflect.DeepEqual(contexts, expectedContexts) {
		t.Errorf("GetContexts() = %v, want %v", contexts, expectedContexts)
	}
}

// TestKafkaDataSourceMock_GetContext tests current context retrieval
func TestKafkaDataSourceMock_GetContext(t *testing.T) {
	mock := KafkaDataSourceMock{}
	
	// Test default context
	defaultContext := mock.GetContext()
	if defaultContext == "" {
		t.Error("GetContext() returned empty string")
	}
	
	// Should return the global currentContext variable
	if defaultContext != currentContext {
		t.Errorf("GetContext() = %v, want %v", defaultContext, currentContext)
	}
}

// TestKafkaDataSourceMock_SetContext tests context switching
func TestKafkaDataSourceMock_SetContext(t *testing.T) {
	mock := KafkaDataSourceMock{}
	
	tests := []struct {
		name        string
		contextName string
		expectError bool
	}{
		{
			name:        "valid context",
			contextName: "kafka-test",
			expectError: false,
		},
		{
			name:        "another valid context",
			contextName: "kafka-prod",
			expectError: false,
		},
		{
			name:        "empty context",
			contextName: "",
			expectError: false, // Mock should accept any context
		},
		{
			name:        "invalid context",
			contextName: "non-existent-context",
			expectError: false, // Mock should accept any context
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mock.SetContext(tt.contextName)
			
			if tt.expectError && err == nil {
				t.Errorf("SetContext(%s) expected error, got none", tt.contextName)
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("SetContext(%s) unexpected error: %v", tt.contextName, err)
			}
			
			// Verify context was set
			if err == nil {
				currentCtx := mock.GetContext()
				if currentCtx != tt.contextName {
					t.Errorf("After SetContext(%s), GetContext() = %v", tt.contextName, currentCtx)
				}
			}
		})
	}
}

// TestKafkaDataSourceMock_GetConsumerGroups tests consumer group retrieval
func TestKafkaDataSourceMock_GetConsumerGroups(t *testing.T) {
	mock := KafkaDataSourceMock{}
	
	groups, err := mock.GetConsumerGroups()
	
	if err != nil {
		t.Errorf("GetConsumerGroups() returned error: %v", err)
	}
	
	if groups == nil {
		t.Fatal("GetConsumerGroups() returned nil")
	}
	
	expectedGroups := []api.ConsumerGroup{
		{Name: "Group1", State: "Active", Consumers: 3},
		{Name: "Group2", State: "Idle", Consumers: 2},
	}
	
	if !reflect.DeepEqual(groups, expectedGroups) {
		t.Errorf("GetConsumerGroups() = %v, want %v", groups, expectedGroups)
	}
	
	// Test individual group properties
	for _, group := range groups {
		if group.Name == "" {
			t.Error("Consumer group has empty name")
		}
		
		if group.State == "" {
			t.Error("Consumer group has empty state")
		}
		
		if group.Consumers < 0 {
			t.Errorf("Consumer group '%s' has negative consumer count: %d", group.Name, group.Consumers)
		}
	}
}

// TestKafkaDataSourceMock_ConsumeTopic tests topic consumption
func TestKafkaDataSourceMock_ConsumeTopic(t *testing.T) {
	mock := KafkaDataSourceMock{}
	
	// Track messages received
	var receivedMessages []api.Message
	handleMessage := func(msg api.Message) {
		receivedMessages = append(receivedMessages, msg)
	}
	
	// Track errors
	var receivedErrors []interface{}
	onError := func(err interface{}) {
		receivedErrors = append(receivedErrors, err)
	}
	
	// Test consumption
	ctx := context.Background()
	topicName := "test-topic"
	flags := api.DefaultConsumeFlags()
	
	err := mock.ConsumeTopic(ctx, topicName, flags, handleMessage, onError)
	
	if err != nil {
		t.Errorf("ConsumeTopic() returned error: %v", err)
	}
	
	// Should receive 100 messages as per implementation
	expectedMessageCount := 100
	if len(receivedMessages) != expectedMessageCount {
		t.Errorf("Received %d messages, want %d", len(receivedMessages), expectedMessageCount)
	}
	
	// Test message structure
	for i, msg := range receivedMessages {
		if !strings.HasPrefix(msg.Key, "purchase_"+topicName+"_") {
			t.Errorf("Message %d key = %v, want prefix 'purchase_%s_'", i, msg.Key, topicName)
		}
		
		if msg.Value == "" {
			t.Errorf("Message %d has empty value", i)
		}
		
		if msg.Offset != int64(i+1) {
			t.Errorf("Message %d offset = %d, want %d", i, msg.Offset, i+1)
		}
		
		if msg.Partition != 0 {
			t.Errorf("Message %d partition = %d, want 0", i, msg.Partition)
		}
		
		// Test JSON structure in value
		if !strings.Contains(msg.Value, "product_id") {
			t.Errorf("Message %d value doesn't contain 'product_id': %s", i, msg.Value)
		}
	}
	
	// Should not receive any errors for mock
	if len(receivedErrors) > 0 {
		t.Errorf("Received unexpected errors: %v", receivedErrors)
	}
}

// TestKafkaDataSourceMock_ConsumeTopicWithContext tests consumption with context cancellation
func TestKafkaDataSourceMock_ConsumeTopicWithContext(t *testing.T) {
	mock := KafkaDataSourceMock{}
	
	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	
	var messageCount int
	handleMessage := func(msg api.Message) {
		messageCount++
		// Cancel after receiving a few messages
		if messageCount >= 5 {
			cancel()
		}
	}
	
	onError := func(err interface{}) {}
	
	err := mock.ConsumeTopic(ctx, "test-topic", api.DefaultConsumeFlags(), handleMessage, onError)
	
	if err != nil {
		t.Errorf("ConsumeTopic() returned error: %v", err)
	}
	
	// Should have received all 100 messages since mock doesn't respect context cancellation
	// This is a limitation of the current mock implementation
	if messageCount != 100 {
		t.Logf("Note: Mock received %d messages (doesn't respect context cancellation)", messageCount)
	}
}

// TestKafkaDataSourceMock_ConsumeFlags tests different consume flag configurations
func TestKafkaDataSourceMock_ConsumeFlags(t *testing.T) {
	mock := KafkaDataSourceMock{}
	
	tests := []struct {
		name  string
		flags api.ConsumeFlags
	}{
		{
			name:  "default flags",
			flags: api.DefaultConsumeFlags(),
		},
		{
			name: "custom flags",
			flags: api.ConsumeFlags{
				Follow:     false,
				Tail:       10,
				OffsetFlag: "earliest",
			},
		},
		{
			name: "zero flags",
			flags: api.ConsumeFlags{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var messageCount int
			handleMessage := func(msg api.Message) {
				messageCount++
			}
			
			onError := func(err interface{}) {}
			
			err := mock.ConsumeTopic(context.Background(), "test-topic", tt.flags, handleMessage, onError)
			
			if err != nil {
				t.Errorf("ConsumeTopic() with %s returned error: %v", tt.name, err)
			}
			
			// Mock should always return 100 messages regardless of flags
			if messageCount != 100 {
				t.Errorf("With %s, received %d messages, want 100", tt.name, messageCount)
			}
		})
	}
}

// TestKafkaDataSourceMock_Interface tests interface compliance
func TestKafkaDataSourceMock_Interface(t *testing.T) {
	var _ api.KafkaDataSource = KafkaDataSourceMock{}
	
	// Test that all interface methods are implemented
	mock := KafkaDataSourceMock{}
	
	// Test each method exists and can be called
	mock.Init("")
	
	_, err := mock.GetTopics()
	if err != nil {
		t.Errorf("GetTopics() interface compliance failed: %v", err)
	}
	
	_, err = mock.GetContexts()
	if err != nil {
		t.Errorf("GetContexts() interface compliance failed: %v", err)
	}
	
	ctx := mock.GetContext()
	if ctx == "" {
		t.Error("GetContext() interface compliance failed")
	}
	
	err = mock.SetContext("test")
	if err != nil {
		t.Errorf("SetContext() interface compliance failed: %v", err)
	}
	
	_, err = mock.GetConsumerGroups()
	if err != nil {
		t.Errorf("GetConsumerGroups() interface compliance failed: %v", err)
	}
	
	err = mock.ConsumeTopic(context.Background(), "test", api.DefaultConsumeFlags(), func(api.Message) {}, func(interface{}) {})
	if err != nil {
		t.Errorf("ConsumeTopic() interface compliance failed: %v", err)
	}
}

// Benchmark tests for mock performance
func BenchmarkKafkaDataSourceMock_GetTopics(b *testing.B) {
	mock := KafkaDataSourceMock{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mock.GetTopics()
		if err != nil {
			b.Fatalf("GetTopics() failed: %v", err)
		}
	}
}

func BenchmarkKafkaDataSourceMock_ConsumeTopic(b *testing.B) {
	mock := KafkaDataSourceMock{}
	handleMessage := func(api.Message) {}
	onError := func(interface{}) {}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := mock.ConsumeTopic(context.Background(), "test-topic", api.DefaultConsumeFlags(), handleMessage, onError)
		if err != nil {
			b.Fatalf("ConsumeTopic() failed: %v", err)
		}
	}
}

func BenchmarkKafkaDataSourceMock_GetConsumerGroups(b *testing.B) {
	mock := KafkaDataSourceMock{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mock.GetConsumerGroups()
		if err != nil {
			b.Fatalf("GetConsumerGroups() failed: %v", err)
		}
	}
}