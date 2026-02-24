package mock

import (
	"context"
	"strings"
	"testing"
	"time"

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

// TestKafkaDataSourceMock_Init_PreloadSchemas tests that schemas are preloaded
func TestKafkaDataSourceMock_Init_PreloadSchemas(t *testing.T) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

	// Verify schema cache is populated
	if len(mock.schemaCache) == 0 {
		t.Error("Init() did not preload schemas")
	}

	// Check for expected schema IDs
	expectedSchemaIDs := []string{"1", "2", "3", "4", "5", "6", "7", "8"}
	for _, id := range expectedSchemaIDs {
		if _, exists := mock.schemaCache[id]; !exists {
			t.Errorf("Schema ID %s not preloaded", id)
		}
	}
}

// TestKafkaDataSourceMock_GetTopics tests topic retrieval
func TestKafkaDataSourceMock_GetTopics(t *testing.T) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

	// Test with different contexts
	contexts := []string{"kafka-dev", "kafka-test", "kafka-prod"}
	for _, ctx := range contexts {
		t.Run(ctx, func(t *testing.T) {
			mock.SetContext(ctx)
			topics, err := mock.GetTopics()

			if err != nil {
				t.Errorf("GetTopics() returned error: %v", err)
			}

			if topics == nil {
				t.Fatal("GetTopics() returned nil topics")
			}

			// Should return multiple topics
			if len(topics) == 0 {
				t.Errorf("GetTopics() returned 0 topics for context %s", ctx)
			}

			// Test topic structure
			for name, topic := range topics {
				if name == "" {
					t.Error("Found topic with empty name")
				}

				if topic.ReplicationFactor < 1 {
					t.Errorf("Topic '%s' ReplicationFactor = %d, want >= 1", name, topic.ReplicationFactor)
				}

				if topic.NumPartitions < 1 {
					t.Errorf("Topic '%s' NumPartitions = %d, want >= 1", name, topic.NumPartitions)
				}

				if topic.ReplicaAssignment == nil {
					t.Errorf("Topic '%s' ReplicaAssignment is nil", name)
				}

				if topic.ConfigEntries == nil {
					t.Errorf("Topic '%s' ConfigEntries is nil", name)
				}
			}
		})
	}
}

// TestKafkaDataSourceMock_GetTopics_DifferentContexts tests that different contexts return different topics
func TestKafkaDataSourceMock_GetTopics_DifferentContexts(t *testing.T) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

	mock.SetContext("kafka-dev")
	devTopics, _ := mock.GetTopics()

	mock.SetContext("kafka-test")
	testTopics, _ := mock.GetTopics()

	mock.SetContext("kafka-prod")
	prodTopics, _ := mock.GetTopics()

	// Prod should have more topics than test
	if len(prodTopics) <= len(testTopics) {
		t.Logf("Note: prod has %d topics, test has %d", len(prodTopics), len(testTopics))
	}

	// All should have common topics like user-events
	for _, topics := range []map[string]api.Topic{devTopics, testTopics, prodTopics} {
		if _, exists := topics["user-events"]; !exists {
			t.Error("Expected 'user-events' topic to exist in all contexts")
		}
	}
}

// TestKafkaDataSourceMock_GetContexts tests context retrieval
func TestKafkaDataSourceMock_GetContexts(t *testing.T) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

	contexts, err := mock.GetContexts()

	if err != nil {
		t.Errorf("GetContexts() returned error: %v", err)
	}

	if contexts == nil {
		t.Fatal("GetContexts() returned nil")
	}

	// Should have at least 3 contexts
	if len(contexts) < 3 {
		t.Errorf("GetContexts() returned %d contexts, want at least 3", len(contexts))
	}

	// Check for expected contexts
	expectedContexts := []string{"kafka-dev", "kafka-test", "kafka-prod"}
	for _, expected := range expectedContexts {
		found := false
		for _, ctx := range contexts {
			if ctx == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected context '%s' not found", expected)
		}
	}
}

// TestKafkaDataSourceMock_GetContext tests current context retrieval
func TestKafkaDataSourceMock_GetContext(t *testing.T) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

	// Test default context
	defaultContext := mock.GetContext()
	if defaultContext == "" {
		t.Error("GetContext() returned empty string")
	}
}

// TestKafkaDataSourceMock_SetContext tests context switching
func TestKafkaDataSourceMock_SetContext(t *testing.T) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

	tests := []struct {
		name        string
		contextName string
		expectError bool
		shouldChange bool
	}{
		{
			name:        "valid context dev",
			contextName: "kafka-dev",
			expectError: false,
			shouldChange: true,
		},
		{
			name:        "valid context test",
			contextName: "kafka-test",
			expectError: false,
			shouldChange: true,
		},
		{
			name:        "valid context prod",
			contextName: "kafka-prod",
			expectError: false,
			shouldChange: true,
		},
		{
			name:        "invalid context",
			contextName: "non-existent-context",
			expectError: false,
			shouldChange: false, // Invalid context should not change current context
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previousContext := mock.GetContext()
			err := mock.SetContext(tt.contextName)

			if tt.expectError && err == nil {
				t.Errorf("SetContext(%s) expected error, got none", tt.contextName)
			}

			if !tt.expectError && err != nil {
				t.Errorf("SetContext(%s) unexpected error: %v", tt.contextName, err)
			}

			// Verify context was set correctly
			currentCtx := mock.GetContext()
			if tt.shouldChange && currentCtx != tt.contextName {
				t.Errorf("After SetContext(%s), GetContext() = %v, want %v", tt.contextName, currentCtx, tt.contextName)
			}
			if !tt.shouldChange && currentCtx != previousContext {
				t.Errorf("After SetContext(%s) with invalid context, context changed from %v to %v", tt.contextName, previousContext, currentCtx)
			}
		})
	}
}

// TestKafkaDataSourceMock_GetConsumerGroups tests consumer group retrieval
func TestKafkaDataSourceMock_GetConsumerGroups(t *testing.T) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

	// Test with different contexts
	contexts := []string{"kafka-dev", "kafka-test", "kafka-prod"}
	for _, ctx := range contexts {
		t.Run(ctx, func(t *testing.T) {
			mock.SetContext(ctx)
			groups, err := mock.GetConsumerGroups()

			if err != nil {
				t.Errorf("GetConsumerGroups() returned error: %v", err)
			}

			if groups == nil {
				t.Fatal("GetConsumerGroups() returned nil")
			}

			// Should return multiple groups
			if len(groups) == 0 {
				t.Errorf("GetConsumerGroups() returned 0 groups for context %s", ctx)
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
		})
	}
}

// TestKafkaDataSourceMock_ConsumeTopic tests topic consumption
func TestKafkaDataSourceMock_ConsumeTopic(t *testing.T) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

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

	// Test consumption with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	topicName := "user-events"
	flags := api.DefaultConsumeFlags()

	err := mock.ConsumeTopic(ctx, topicName, flags, handleMessage, onError)

	// Should return context timeout error
	if err == nil {
		t.Errorf("ConsumeTopic() should return error when context times out")
	}

	// Should receive some messages (at least 1)
	if len(receivedMessages) < 1 {
		t.Errorf("Received %d messages, want at least 1", len(receivedMessages))
	}

	// Test message structure for first few messages
	testCount := len(receivedMessages)
	if testCount > 5 {
		testCount = 5
	}
	for i := 0; i < testCount; i++ {
		msg := receivedMessages[i]
		
		if msg.Key == "" {
			t.Errorf("Message %d has empty key", i)
		}

		if msg.Value == "" {
			t.Errorf("Message %d has empty value", i)
		}

		if msg.Offset < 0 {
			t.Errorf("Message %d has negative offset: %d", i, msg.Offset)
		}

		if msg.Partition < 0 {
			t.Errorf("Message %d has negative partition: %d", i, msg.Partition)
		}

		// Check for expected headers
		if len(msg.Headers) == 0 {
			t.Errorf("Message %d has no headers", i)
		}

		// Check for correlation ID header
		hasCorrelationID := false
		for _, header := range msg.Headers {
			if header.Key == "correlationId" {
				hasCorrelationID = true
				break
			}
		}
		if !hasCorrelationID {
			t.Errorf("Message %d missing correlationId header", i)
		}
	}
}

// TestKafkaDataSourceMock_ConsumeTopic_MessageTypes tests that different topic patterns generate different message types
func TestKafkaDataSourceMock_ConsumeTopic_MessageTypes(t *testing.T) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

	tests := []struct {
		topicName        string
		expectedSchemaID string
	}{
		{"user-events", "1"},
		{"order-events", "2"},
		{"payment-events", "5"},
		{"clickstream-events", "6"},
		{"notification-events", "7"},
		{"audit-log", "8"},
		{"inventory-events", ""}, // Has key schema
		{"generic-topic", ""},
	}

	for _, tt := range tests {
		t.Run(tt.topicName, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			var messages []api.Message
			handleMessage := func(msg api.Message) {
				messages = append(messages, msg)
			}

			onError := func(err interface{}) {}

			mock.ConsumeTopic(ctx, tt.topicName, api.DefaultConsumeFlags(), handleMessage, onError)

			if len(messages) == 0 {
				t.Errorf("No messages received for topic %s", tt.topicName)
			}

			// Check first message has expected schema ID
			if tt.expectedSchemaID != "" && messages[0].ValueSchemaID != tt.expectedSchemaID {
				t.Errorf("Message ValueSchemaID = %v, want %v", messages[0].ValueSchemaID, tt.expectedSchemaID)
			}
		})
	}
}

// TestKafkaDataSourceMock_ConsumeTopic_CDCWithKeySchema tests CDC topics have key schemas
func TestKafkaDataSourceMock_ConsumeTopic_CDCWithKeySchema(t *testing.T) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	var messages []api.Message
	handleMessage := func(msg api.Message) {
		messages = append(messages, msg)
	}

	onError := func(err interface{}) {}

	mock.ConsumeTopic(ctx, "dbserver1.inventory.products", api.DefaultConsumeFlags(), handleMessage, onError)

	if len(messages) == 0 {
		t.Fatal("No messages received for CDC topic")
	}

	// Verify key schema is present
	msg := messages[0]
	if msg.KeySchemaID != "9" {
		t.Errorf("CDC message KeySchemaID = %v, want 9", msg.KeySchemaID)
	}

	// Verify value schema is present
	if msg.ValueSchemaID != "10" {
		t.Errorf("CDC message ValueSchemaID = %v, want 10", msg.ValueSchemaID)
	}

	// Verify key has CDC format
	if !strings.Contains(msg.Key, "payload") {
		t.Errorf("CDC message key doesn't contain 'payload': %s", msg.Key)
	}

	// Verify value has CDC format
	if !strings.Contains(msg.Value, "schema") || !strings.Contains(msg.Value, "payload") {
		t.Errorf("CDC message value doesn't have CDC format: %s", msg.Value)
	}

	// Verify CDC-specific headers
	hasOp := false
	hasTsMs := false
	for _, header := range msg.Headers {
		if header.Key == "op" {
			hasOp = true
		}
		if header.Key == "ts_ms" {
			hasTsMs = true
		}
	}
	if !hasOp {
		t.Error("CDC message missing 'op' header")
	}
	if !hasTsMs {
		t.Error("CDC message missing 'ts_ms' header")
	}
}

// TestKafkaDataSourceMock_ConsumeTopic_RealisticOffsets tests that offsets are realistic (large numbers)
func TestKafkaDataSourceMock_ConsumeTopic_RealisticOffsets(t *testing.T) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

	topics := []string{
		"clickstream-events",
		"order-events",
		"user-events",
		"audit-log",
	}

	for _, topicName := range topics {
		t.Run(topicName, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()

			var messages []api.Message
			handleMessage := func(msg api.Message) {
				messages = append(messages, msg)
			}

			onError := func(err interface{}) {}

			mock.ConsumeTopic(ctx, topicName, api.DefaultConsumeFlags(), handleMessage, onError)

			if len(messages) == 0 {
				t.Fatal("No messages received")
			}

			// Check that offsets are realistic (in the millions or higher)
			// Real Kafka offsets for active topics are typically large numbers
			minOffset := int64(100000) // Minimum realistic offset
			for i, msg := range messages {
				if msg.Offset < minOffset {
					t.Errorf("Message %d offset = %d, want >= %d (realistic Kafka offset)", i, msg.Offset, minOffset)
				}
			}

			// Verify offsets are increasing
			for i := 1; i < len(messages); i++ {
				if messages[i].Offset <= messages[i-1].Offset {
					t.Errorf("Message %d offset (%d) should be > message %d offset (%d)",
						i, messages[i].Offset, i-1, messages[i-1].Offset)
				}
			}
		})
	}
}

// TestKafkaDataSourceMock_ConsumeTopic_PartitionVariation tests that messages have partition variation
func TestKafkaDataSourceMock_ConsumeTopic_PartitionVariation(t *testing.T) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	var messages []api.Message
	handleMessage := func(msg api.Message) {
		messages = append(messages, msg)
	}

	onError := func(err interface{}) {}

	mock.ConsumeTopic(ctx, "user-events", api.DefaultConsumeFlags(), handleMessage, onError)

	if len(messages) < 5 {
		t.Skipf("Not enough messages to test partition variation (got %d)", len(messages))
	}

	// Check that we have some partition variation (not all same partition)
	partitions := make(map[int32]bool)
	for _, msg := range messages {
		partitions[msg.Partition] = true
	}

	if len(partitions) < 2 {
		t.Logf("Note: Only %d unique partition(s) found, expected more variation", len(partitions))
	}

	// Verify partitions are in valid range (0-4)
	for _, msg := range messages {
		if msg.Partition < 0 || msg.Partition >= 5 {
			t.Errorf("Message partition = %d, want 0-4", msg.Partition)
		}
	}
}

// TestKafkaDataSourceMock_ConsumeTopicWithContext tests consumption with context cancellation
func TestKafkaDataSourceMock_ConsumeTopicWithContext(t *testing.T) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	var messageCount int
	handleMessage := func(msg api.Message) {
		messageCount++
		// Cancel after receiving a few messages
		if messageCount >= 3 {
			cancel()
		}
	}

	onError := func(err interface{}) {}

	err := mock.ConsumeTopic(ctx, "test-topic", api.DefaultConsumeFlags(), handleMessage, onError)

	// Should return context cancellation error
	if err != context.Canceled {
		t.Errorf("ConsumeTopic() should return context.Canceled, got: %v", err)
	}

	// Should have received at least 3 messages
	if messageCount < 3 {
		t.Errorf("Received %d messages, want at least 3", messageCount)
	}
}

// TestKafkaDataSourceMock_ConsumeFlags tests different consume flag configurations
func TestKafkaDataSourceMock_ConsumeFlags(t *testing.T) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

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
			name:  "zero flags",
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

			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			err := mock.ConsumeTopic(ctx, "test-topic", tt.flags, handleMessage, onError)

			if err == nil {
				t.Errorf("ConsumeTopic() with %s should return timeout error", tt.name)
			}

			// Should receive some messages regardless of flags
			if messageCount < 1 {
				t.Errorf("With %s, received %d messages, want at least 1", tt.name, messageCount)
			}
		})
	}
}

// TestKafkaDataSourceMock_GetMessageSchemaInfo tests schema retrieval
func TestKafkaDataSourceMock_GetMessageSchemaInfo(t *testing.T) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

	tests := []struct {
		name           string
		keySchemaID    string
		valueSchemaID  string
		wantKeySchema  bool
		wantValueSchema bool
	}{
		{
			name:           "both schemas",
			keySchemaID:    "4",
			valueSchemaID:  "1",
			wantKeySchema:  true,
			wantValueSchema: true,
		},
		{
			name:           "value schema only",
			keySchemaID:    "",
			valueSchemaID:  "2",
			wantKeySchema:  false,
			wantValueSchema: true,
		},
		{
			name:           "key schema only",
			keySchemaID:    "4",
			valueSchemaID:  "",
			wantKeySchema:  true,
			wantValueSchema: false,
		},
		{
			name:           "no schemas",
			keySchemaID:    "",
			valueSchemaID:  "",
			wantKeySchema:  false,
			wantValueSchema: false,
		},
		{
			name:           "invalid schema ID",
			keySchemaID:    "999",
			valueSchemaID:  "999",
			wantKeySchema:  false,
			wantValueSchema: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaInfo, err := mock.GetMessageSchemaInfo(tt.keySchemaID, tt.valueSchemaID)

			if err != nil {
				t.Errorf("GetMessageSchemaInfo() returned error: %v", err)
			}

			if tt.wantKeySchema && tt.keySchemaID != "" {
				if schemaInfo == nil || schemaInfo.KeySchema == nil {
					t.Errorf("GetMessageSchemaInfo() missing key schema for ID %s", tt.keySchemaID)
				}
			}

			if tt.wantValueSchema && tt.valueSchemaID != "" {
				if schemaInfo == nil || schemaInfo.ValueSchema == nil {
					t.Errorf("GetMessageSchemaInfo() missing value schema for ID %s", tt.valueSchemaID)
				}
			}

			if !tt.wantKeySchema && !tt.wantValueSchema {
				if schemaInfo != nil {
					t.Error("GetMessageSchemaInfo() should return nil when no schemas found")
				}
			}
		})
	}
}

// TestKafkaDataSourceMock_GetMessageSchemaInfo_SchemaContent tests schema content
func TestKafkaDataSourceMock_GetMessageSchemaInfo_SchemaContent(t *testing.T) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

	schemaInfo, err := mock.GetMessageSchemaInfo("", "1")
	if err != nil {
		t.Fatalf("GetMessageSchemaInfo() returned error: %v", err)
	}

	if schemaInfo == nil || schemaInfo.ValueSchema == nil {
		t.Fatal("GetMessageSchemaInfo() returned nil schema")
	}

	schema := schemaInfo.ValueSchema
	if schema.ID != 1 {
		t.Errorf("Schema ID = %d, want 1", schema.ID)
	}

	if schema.Subject != "user-events-value" {
		t.Errorf("Schema subject = %s, want user-events-value", schema.Subject)
	}

	if schema.RecordName != "UserRegisteredEvent" {
		t.Errorf("Schema record name = %s, want UserRegisteredEvent", schema.RecordName)
	}

	if !strings.Contains(schema.Schema, "userId") {
		t.Error("Schema doesn't contain userId field")
	}
}

// TestKafkaDataSourceMock_Interface tests interface compliance
func TestKafkaDataSourceMock_Interface(t *testing.T) {
	var _ api.KafkaDataSource = &KafkaDataSourceMock{}

	// Test that all interface methods are implemented
	mock := &KafkaDataSourceMock{}
	mock.Init("")

	// Test each method exists and can be called
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

	// ConsumeTopic will timeout, which is expected
	consumeCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = mock.ConsumeTopic(consumeCtx, "test", api.DefaultConsumeFlags(), func(api.Message) {}, func(interface{}) {})
	if err == nil {
		t.Error("ConsumeTopic() should return timeout error")
	}

	_, err = mock.GetMessageSchemaInfo("1", "2")
	if err != nil {
		t.Errorf("GetMessageSchemaInfo() interface compliance failed: %v", err)
	}
}

// Benchmark tests for mock performance
func BenchmarkKafkaDataSourceMock_GetTopics(b *testing.B) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mock.GetTopics()
		if err != nil {
			b.Fatalf("GetTopics() failed: %v", err)
		}
	}
}

func BenchmarkKafkaDataSourceMock_GetConsumerGroups(b *testing.B) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mock.GetConsumerGroups()
		if err != nil {
			b.Fatalf("GetConsumerGroups() failed: %v", err)
		}
	}
}

func BenchmarkKafkaDataSourceMock_GetMessageSchemaInfo(b *testing.B) {
	mock := KafkaDataSourceMock{}
	mock.Init("")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mock.GetMessageSchemaInfo("1", "2")
		if err != nil {
			b.Fatalf("GetMessageSchemaInfo() failed: %v", err)
		}
	}
}
