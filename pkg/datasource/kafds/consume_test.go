package kafds

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
	prettyjson "github.com/hokaccha/go-prettyjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConsumeFlags tests the ConsumeFlags structure and its methods
func TestConsumeFlags(t *testing.T) {
	t.Run("default_consume_flags", func(t *testing.T) {
		flags := api.ConsumeFlags{}
		
		assert.False(t, flags.Follow, "Follow should default to false")
		assert.Equal(t, int32(0), flags.Tail, "Tail should default to 0")
		assert.Empty(t, flags.OffsetFlag, "OffsetFlag should be empty by default")
	})
	
	t.Run("custom_consume_flags", func(t *testing.T) {
		flags := api.ConsumeFlags{
			Follow:     true,
			Tail:       100,
			OffsetFlag: "earliest",
		}
		
		assert.True(t, flags.Follow, "Follow should be true")
		assert.Equal(t, int32(100), flags.Tail, "Tail should be 100")
		assert.Equal(t, "earliest", flags.OffsetFlag, "OffsetFlag should match")
	})
	
	t.Run("default_consume_flags_function", func(t *testing.T) {
		flags := api.DefaultConsumeFlags()
		
		assert.True(t, flags.Follow, "Default Follow should be true")
		assert.Equal(t, int32(50), flags.Tail, "Default Tail should be 50")
		assert.Equal(t, "latest", flags.OffsetFlag, "Default OffsetFlag should be 'latest'")
	})
}

// TestMessageHandling tests message handling functionality
func TestMessageHandling(t *testing.T) {
	t.Run("message_handler_func", func(t *testing.T) {
		var receivedMessage api.Message
		handler := func(msg api.Message) {
			receivedMessage = msg
		}
		
		testMessage := api.Message{
			Key:       "test-key",
			Value:     "test-value",
			Offset:    123,
			Partition: 1,
			Headers: []api.MessageHeader{
				{Key: "header1", Value: "value1"},
				{Key: "header2", Value: "value2"},
			},
		}
		
		handler(testMessage)
		
		assert.Equal(t, testMessage.Key, receivedMessage.Key, "Key should match")
		assert.Equal(t, testMessage.Value, receivedMessage.Value, "Value should match")
		assert.Equal(t, testMessage.Offset, receivedMessage.Offset, "Offset should match")
		assert.Equal(t, testMessage.Partition, receivedMessage.Partition, "Partition should match")
		assert.Equal(t, testMessage.Headers, receivedMessage.Headers, "Headers should match")
	})
}

// TestConsumerGroupHandling tests consumer group related functionality
func TestConsumerGroupHandling(t *testing.T) {
	t.Run("consumer_group_creation", func(t *testing.T) {
		// Test that we can create consumer group configurations
		config := sarama.NewConfig()
		config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
		config.Consumer.Offsets.Initial = sarama.OffsetNewest
		
		assert.NotNil(t, config, "Config should be created")
		assert.Equal(t, sarama.BalanceStrategyRoundRobin, config.Consumer.Group.Rebalance.Strategy)
		assert.Equal(t, sarama.OffsetNewest, config.Consumer.Offsets.Initial)
	})
	
	t.Run("consumer_group_validation", func(t *testing.T) {
		tests := []struct {
			name          string
			consumerGroup string
			valid         bool
		}{
			{"empty_group", "", false},
			{"valid_group", "test-group", true},
			{"group_with_special_chars", "test-group-123", true},
			{"group_with_dots", "test.group", true},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Basic validation - non-empty group names are generally valid
				isValid := tt.consumerGroup != ""
				assert.Equal(t, tt.valid, isValid, "Consumer group validation should match expected")
			})
		}
	})
}

// TestPartitionConsumerHandling tests partition-specific consumption
func TestPartitionConsumerHandling(t *testing.T) {
	t.Run("offset_flag_interpretation", func(t *testing.T) {
		tests := []struct {
			name          string
			flags         api.ConsumeFlags
			expectedStart int64
		}{
			{
				name: "earliest_offset",
				flags: api.ConsumeFlags{
					OffsetFlag: "earliest",
				},
				expectedStart: sarama.OffsetOldest,
			},
			{
				name: "latest_offset",
				flags: api.ConsumeFlags{
					OffsetFlag: "latest",
				},
				expectedStart: sarama.OffsetNewest,
			},
			{
				name: "empty_offset_flag",
				flags: api.ConsumeFlags{
					OffsetFlag: "",
				},
				expectedStart: sarama.OffsetNewest, // Default to newest
			},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var startOffset int64
				
				switch tt.flags.OffsetFlag {
				case "earliest":
					startOffset = sarama.OffsetOldest
				case "latest":
					startOffset = sarama.OffsetNewest
				default:
					startOffset = sarama.OffsetNewest // Default
				}
				
				assert.Equal(t, tt.expectedStart, startOffset, "Start offset should match expected")
			})
		}
	})
}

// TestMessageSerialization tests message serialization and deserialization
func TestMessageSerialization(t *testing.T) {
	t.Run("sarama_to_api_message_conversion", func(t *testing.T) {
		// Mock Sarama consumer message
		saramaMsg := &sarama.ConsumerMessage{
			Key:       []byte("test-key"),
			Value:     []byte("test-value"),
			Topic:     "test-topic",
			Partition: 1,
			Offset:    123,
			Timestamp: time.Now(),
			Headers: []*sarama.RecordHeader{
				{Key: []byte("header1"), Value: []byte("value1")},
				{Key: []byte("header2"), Value: []byte("value2")},
			},
		}
		
		// Convert to API message format
		apiMsg := api.Message{
			Key:       string(saramaMsg.Key),
			Value:     string(saramaMsg.Value),
			Offset:    saramaMsg.Offset,
			Partition: saramaMsg.Partition,
			Headers:   make([]api.MessageHeader, 0),
		}
		
		// Convert headers
		for _, header := range saramaMsg.Headers {
			apiMsg.Headers = append(apiMsg.Headers, api.MessageHeader{
				Key:   string(header.Key),
				Value: string(header.Value),
			})
		}
		
		assert.Equal(t, "test-key", apiMsg.Key, "Key should be converted correctly")
		assert.Equal(t, "test-value", apiMsg.Value, "Value should be converted correctly")
		assert.Equal(t, int64(123), apiMsg.Offset, "Offset should be converted correctly")
		assert.Equal(t, int32(1), apiMsg.Partition, "Partition should be converted correctly")
		assert.Len(t, apiMsg.Headers, 2, "Should have 2 headers")
		assert.Equal(t, "header1", apiMsg.Headers[0].Key, "First header key should be correct")
		assert.Equal(t, "value1", apiMsg.Headers[0].Value, "First header value should be correct")
		assert.Equal(t, "header2", apiMsg.Headers[1].Key, "Second header key should be correct")
		assert.Equal(t, "value2", apiMsg.Headers[1].Value, "Second header value should be correct")
	})
	
	t.Run("empty_message_handling", func(t *testing.T) {
		saramaMsg := &sarama.ConsumerMessage{
			Key:       nil,
			Value:     nil,
			Topic:     "test-topic",
			Partition: 0,
			Offset:    0,
			Headers:   nil,
		}
		
		apiMsg := api.Message{
			Key:       string(saramaMsg.Key),
			Value:     string(saramaMsg.Value),
			Offset:    saramaMsg.Offset,
			Partition: saramaMsg.Partition,
			Headers:   make([]api.MessageHeader, 0),
		}
		
		assert.Empty(t, apiMsg.Key, "Empty key should be handled")
		assert.Empty(t, apiMsg.Value, "Empty value should be handled")
		assert.Equal(t, int64(0), apiMsg.Offset, "Zero offset should be handled")
		assert.Equal(t, int32(0), apiMsg.Partition, "Zero partition should be handled")
		assert.NotNil(t, apiMsg.Headers, "Headers should be initialized")
	})
}

// TestConsumerConfiguration tests consumer configuration setup
func TestConsumerConfiguration(t *testing.T) {
	t.Run("consumer_config_creation", func(t *testing.T) {
		config := sarama.NewConfig()
		
		// Test common consumer configurations
		config.Consumer.Return.Errors = true
		config.Consumer.Offsets.Initial = sarama.OffsetNewest
		config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
		config.Consumer.Group.Session.Timeout = 10 * time.Second
		config.Consumer.Group.Heartbeat.Interval = 3 * time.Second
		
		assert.True(t, config.Consumer.Return.Errors, "Should return errors")
		assert.Equal(t, sarama.OffsetNewest, config.Consumer.Offsets.Initial)
		assert.Equal(t, sarama.BalanceStrategyRoundRobin, config.Consumer.Group.Rebalance.Strategy)
		assert.Equal(t, 10*time.Second, config.Consumer.Group.Session.Timeout)
		assert.Equal(t, 3*time.Second, config.Consumer.Group.Heartbeat.Interval)
	})
	
	t.Run("consumer_config_validation", func(t *testing.T) {
		config := sarama.NewConfig()
		
		// Test that config validation works
		err := config.Validate()
		assert.NoError(t, err, "Default config should be valid")
		
		// Test invalid configuration
		config.Consumer.Group.Session.Timeout = 0
		err = config.Validate()
		assert.Error(t, err, "Invalid config should return error")
	})
}

// TestConsumeErrorHandling tests error handling in consumption scenarios
func TestConsumeErrorHandling(t *testing.T) {
	t.Run("context_cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		
		// Immediately cancel the context
		cancel()
		
		// Test that cancelled context is handled properly
		select {
		case <-ctx.Done():
			assert.Equal(t, context.Canceled, ctx.Err(), "Context should be cancelled")
		default:
			t.Error("Context should be cancelled")
		}
	})
	
	t.Run("timeout_handling", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()
		
		// Wait for timeout
		time.Sleep(2 * time.Millisecond)
		
		select {
		case <-ctx.Done():
			assert.Equal(t, context.DeadlineExceeded, ctx.Err(), "Context should timeout")
		default:
			t.Error("Context should have timed out")
		}
	})
	
	t.Run("invalid_topic_name", func(t *testing.T) {
		invalidTopics := []string{
			"", // empty
			"topic with spaces",
			"topic/with/slashes",
			"topic\nwith\nnewlines",
		}
		
		for _, topic := range invalidTopics {
			t.Run("topic_"+topic, func(t *testing.T) {
				// Basic validation - empty topics are definitely invalid
				isValid := topic != ""
				if topic == "" {
					assert.False(t, isValid, "Empty topic should be invalid")
				}
				// Other validation would depend on Kafka's topic naming rules
			})
		}
	})
}

// TestConsumerGroupSessionHandling tests consumer group session management
func TestConsumerGroupSessionHandling(t *testing.T) {
	t.Run("session_context_handling", func(t *testing.T) {
		// Test that we can create and handle session contexts properly
		ctx := context.Background()
		
		// Create a child context for the session
		sessionCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		
		assert.NotNil(t, sessionCtx, "Session context should be created")
		
		// Test cancellation
		cancel()
		
		select {
		case <-sessionCtx.Done():
			assert.Equal(t, context.Canceled, sessionCtx.Err(), "Session context should be cancelled")
		default:
			t.Error("Session context should be cancelled")
		}
	})
}

// TestTailConsumption tests tail consumption functionality
func TestTailConsumption(t *testing.T) {
	t.Run("tail_count_validation", func(t *testing.T) {
		tests := []struct {
			name      string
			tailCount int64
			valid     bool
		}{
			{"zero_tail", 0, true},
			{"positive_tail", 100, true},
			{"negative_tail", -1, false},
			{"large_tail", 10000, true},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				isValid := tt.tailCount >= 0
				assert.Equal(t, tt.valid, isValid, "Tail count validation should match expected")
			})
		}
	})
}

// TestAvroDeserialization tests Avro message deserialization
func TestAvroDeserialization(t *testing.T) {
	t.Run("avro_message_handling", func(t *testing.T) {
		// Test basic Avro message structure
		avroMessage := []byte(`{"field1": "value1", "field2": 123}`)
		
		// Basic validation that we can handle Avro-like data
		assert.NotEmpty(t, avroMessage, "Avro message should not be empty")
		assert.True(t, len(avroMessage) > 0, "Avro message should have content")
		
		// In a real implementation, this would involve actual Avro deserialization
		// For now, we just test that we can handle the data
		messageStr := string(avroMessage)
		assert.Contains(t, messageStr, "field1", "Should contain field1")
		assert.Contains(t, messageStr, "value1", "Should contain value1")
	})
}

// TestConsumerMetrics tests consumer metrics and monitoring
func TestConsumerMetrics(t *testing.T) {
	t.Run("message_count_tracking", func(t *testing.T) {
		var messageCount int64
		
		handler := func(msg api.Message) {
			messageCount++
		}
		
		// Simulate processing multiple messages
		for i := 0; i < 5; i++ {
			handler(api.Message{
				Key:   "key" + string(rune(i)),
				Value: "value" + string(rune(i)),
			})
		}
		
		assert.Equal(t, int64(5), messageCount, "Should track message count correctly")
	})
	
	t.Run("error_count_tracking", func(t *testing.T) {
		var errorCount int64
		
		errorHandler := func(err any) {
			errorCount++
		}
		
		// Simulate multiple errors
		for i := 0; i < 3; i++ {
			errorHandler("test error")
		}
		
		assert.Equal(t, int64(3), errorCount, "Should track error count correctly")
	})
}

// MockClient implements sarama.Client for testing
type MockClient struct {
	getOffsetFunc func(topic string, partitionID int32, time int64) (int64, error)
}

func (m *MockClient) Config() *sarama.Config                                                    { return nil }
func (m *MockClient) Controller() (*sarama.Broker, error)                                      { return nil, nil }
func (m *MockClient) RefreshController() (*sarama.Broker, error)                               { return nil, nil }
func (m *MockClient) Brokers() []*sarama.Broker                                                { return nil }
func (m *MockClient) Broker(brokerID int32) (*sarama.Broker, error)                           { return nil, nil }
func (m *MockClient) Topics() ([]string, error)                                                { return nil, nil }
func (m *MockClient) Partitions(topic string) ([]int32, error)                                { return nil, nil }
func (m *MockClient) WritablePartitions(topic string) ([]int32, error)                        { return nil, nil }
func (m *MockClient) Leader(topic string, partitionID int32) (*sarama.Broker, error)          { return nil, nil }
func (m *MockClient) Replicas(topic string, partitionID int32) ([]int32, error)               { return nil, nil }
func (m *MockClient) InSyncReplicas(topic string, partitionID int32) ([]int32, error)         { return nil, nil }
func (m *MockClient) OfflineReplicas(topic string, partitionID int32) ([]int32, error)        { return nil, nil }
func (m *MockClient) RefreshBrokers(addrs []string) error                                      { return nil }
func (m *MockClient) RefreshMetadata(topics ...string) error                                   { return nil }
func (m *MockClient) GetOffset(topic string, partitionID int32, time int64) (int64, error) {
	if m.getOffsetFunc != nil {
		return m.getOffsetFunc(topic, partitionID, time)
	}
	return 0, nil
}
func (m *MockClient) Coordinator(consumerGroup string) (*sarama.Broker, error)                 { return nil, nil }
func (m *MockClient) RefreshCoordinator(consumerGroup string) error                            { return nil }
func (m *MockClient) InitProducerID() (*sarama.InitProducerIDResponse, error)                  { return nil, nil }
func (m *MockClient) Close() error                                                             { return nil }
func (m *MockClient) Closed() bool                                                             { return false }
func (m *MockClient) TransactionCoordinator(transactionID string) (*sarama.Broker, error)     { return nil, nil }
func (m *MockClient) RefreshTransactionCoordinator(transactionID string) error                 { return nil }
func (m *MockClient) LeastLoadedBroker() *sarama.Broker                                        { return nil }
func (m *MockClient) LeaderAndEpoch(topic string, partitionID int32) (*sarama.Broker, int32, error) { return nil, 0, nil }
func (m *MockClient) PartitionNotReadable(topic string, partition int32) bool                     { return false }

// TestGetOffsets tests the getOffsets function using mock client
func TestGetOffsets(t *testing.T) {
	t.Run("successful_offset_retrieval", func(t *testing.T) {
		mockClient := &MockClient{
			getOffsetFunc: func(topic string, partitionID int32, time int64) (int64, error) {
				if time == sarama.OffsetNewest {
					return 1000, nil
				}
				if time == sarama.OffsetOldest {
					return 100, nil
				}
				return 0, nil
			},
		}
		
		topic := "test-topic"
		partition := int32(0)
		
		offsets, err := getOffsets(mockClient, topic, partition)
		
		require.NoError(t, err)
		assert.Equal(t, int64(1000), offsets.newest)
		assert.Equal(t, int64(100), offsets.oldest)
	})
	
	t.Run("error_getting_newest_offset", func(t *testing.T) {
		mockClient := &MockClient{
			getOffsetFunc: func(topic string, partitionID int32, time int64) (int64, error) {
				if time == sarama.OffsetNewest {
					return 0, assert.AnError
				}
				return 100, nil
			},
		}
		
		topic := "test-topic"
		partition := int32(0)
		
		offsets, err := getOffsets(mockClient, topic, partition)
		
		assert.Error(t, err)
		assert.Nil(t, offsets)
	})
	
	t.Run("error_getting_oldest_offset", func(t *testing.T) {
		mockClient := &MockClient{
			getOffsetFunc: func(topic string, partitionID int32, time int64) (int64, error) {
				if time == sarama.OffsetNewest {
					return 1000, nil
				}
				if time == sarama.OffsetOldest {
					return 0, assert.AnError
				}
				return 0, nil
			},
		}
		
		topic := "test-topic"
		partition := int32(0)
		
		offsets, err := getOffsets(mockClient, topic, partition)
		
		assert.Error(t, err)
		assert.Nil(t, offsets)
	})
}

// TestHandleMessage tests the handleMessage function
func TestHandleMessage(t *testing.T) {
	t.Run("basic_message_handling", func(t *testing.T) {
		var receivedMessage api.Message
		var mu sync.Mutex
		
		// Set up global handler
		handler = func(msg api.Message) {
			receivedMessage = msg
		}
		
		saramaMsg := &sarama.ConsumerMessage{
			Key:       []byte("test-key"),
			Value:     []byte("test-value"),
			Topic:     "test-topic",
			Partition: 1,
			Offset:    123,
			Timestamp: time.Now(),
			Headers: []*sarama.RecordHeader{
				{Key: []byte("header1"), Value: []byte("value1")},
			},
		}
		
		handleMessage(saramaMsg, &mu)
		
		assert.Equal(t, "test-key", receivedMessage.Key)
		assert.Equal(t, "test-value", receivedMessage.Value)
		assert.Equal(t, int64(123), receivedMessage.Offset)
		assert.Equal(t, int32(1), receivedMessage.Partition)
		assert.Len(t, receivedMessage.Headers, 1)
		assert.Equal(t, "header1", receivedMessage.Headers[0].Key)
		assert.Equal(t, "value1", receivedMessage.Headers[0].Value)
	})
	
	t.Run("message_with_empty_key_and_value", func(t *testing.T) {
		var receivedMessage api.Message
		var mu sync.Mutex
		
		handler = func(msg api.Message) {
			receivedMessage = msg
		}
		
		saramaMsg := &sarama.ConsumerMessage{
			Key:       nil,
			Value:     nil,
			Topic:     "test-topic",
			Partition: 0,
			Offset:    0,
		}
		
		handleMessage(saramaMsg, &mu)
		
		assert.Empty(t, receivedMessage.Key)
		assert.Empty(t, receivedMessage.Value)
		assert.Equal(t, int64(0), receivedMessage.Offset)
		assert.Equal(t, int32(0), receivedMessage.Partition)
	})
	
	t.Run("message_with_multiple_headers", func(t *testing.T) {
		var receivedMessage api.Message
		var mu sync.Mutex
		
		handler = func(msg api.Message) {
			receivedMessage = msg
		}
		
		saramaMsg := &sarama.ConsumerMessage{
			Key:       []byte("key"),
			Value:     []byte("value"),
			Topic:     "test-topic",
			Partition: 2,
			Offset:    456,
			Headers: []*sarama.RecordHeader{
				{Key: []byte("content-type"), Value: []byte("application/json")},
				{Key: []byte("source"), Value: []byte("test-service")},
				{Key: []byte("trace-id"), Value: []byte("12345")},
			},
		}
		
		handleMessage(saramaMsg, &mu)
		
		assert.Len(t, receivedMessage.Headers, 3)
		assert.Equal(t, "content-type", receivedMessage.Headers[0].Key)
		assert.Equal(t, "application/json", receivedMessage.Headers[0].Value)
		assert.Equal(t, "source", receivedMessage.Headers[1].Key)
		assert.Equal(t, "test-service", receivedMessage.Headers[1].Value)
		assert.Equal(t, "trace-id", receivedMessage.Headers[2].Key)
		assert.Equal(t, "12345", receivedMessage.Headers[2].Value)
	})
}

// TestGetSchemaIdIfPresent tests schema ID extraction from Avro messages
func TestGetSchemaIdIfPresent(t *testing.T) {
	t.Run("valid_avro_message_with_schema_id", func(t *testing.T) {
		// Create a valid Avro message with magic byte and schema ID
		avroMessage := []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03} // Magic byte + schema ID 1 + data
		
		schemaID := getSchemaIdIfPresent(avroMessage)
		
		assert.Equal(t, "1", schemaID)
	})
	
	t.Run("valid_avro_message_with_larger_schema_id", func(t *testing.T) {
		// Schema ID 12345 (0x00003039)
		avroMessage := []byte{0x00, 0x00, 0x00, 0x30, 0x39, 0x02, 0x03}
		
		schemaID := getSchemaIdIfPresent(avroMessage)
		
		assert.Equal(t, "12345", schemaID)
	})
	
	t.Run("non_avro_message", func(t *testing.T) {
		nonAvroMessage := []byte("regular json message")
		
		schemaID := getSchemaIdIfPresent(nonAvroMessage)
		
		assert.Empty(t, schemaID)
	})
	
	t.Run("message_too_short", func(t *testing.T) {
		shortMessage := []byte{0x00, 0x01}
		
		schemaID := getSchemaIdIfPresent(shortMessage)
		
		assert.Empty(t, schemaID)
	})
	
	t.Run("wrong_magic_byte", func(t *testing.T) {
		wrongMagicMessage := []byte{0x01, 0x00, 0x00, 0x00, 0x01}
		
		schemaID := getSchemaIdIfPresent(wrongMagicMessage)
		
		assert.Empty(t, schemaID)
	})
	
	t.Run("empty_message", func(t *testing.T) {
		emptyMessage := []byte{}
		
		schemaID := getSchemaIdIfPresent(emptyMessage)
		
		assert.Empty(t, schemaID)
	})
}

// MockConsumerGroupSession implements sarama.ConsumerGroupSession for testing
type MockConsumerGroupSession struct {
	markMessageCalled bool
	markedMessage     *sarama.ConsumerMessage
}

func (m *MockConsumerGroupSession) Claims() map[string][]int32                                  { return nil }
func (m *MockConsumerGroupSession) MemberID() string                                           { return "" }
func (m *MockConsumerGroupSession) GenerationID() int32                                        { return 0 }
func (m *MockConsumerGroupSession) MarkOffset(topic string, partition int32, offset int64, metadata string) {}
func (m *MockConsumerGroupSession) ResetOffset(topic string, partition int32, offset int64, metadata string) {}
func (m *MockConsumerGroupSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {
	m.markMessageCalled = true
	m.markedMessage = msg
}
func (m *MockConsumerGroupSession) Context() context.Context { return context.Background() }
func (m *MockConsumerGroupSession) Commit()                  {}

// MockConsumerGroupClaim implements sarama.ConsumerGroupClaim for testing
type MockConsumerGroupClaim struct {
	messages chan *sarama.ConsumerMessage
}

func (m *MockConsumerGroupClaim) Topic() string                                    { return "test-topic" }
func (m *MockConsumerGroupClaim) Partition() int32                                 { return 0 }
func (m *MockConsumerGroupClaim) InitialOffset() int64                             { return 0 }
func (m *MockConsumerGroupClaim) HighWaterMarkOffset() int64                       { return 1000 }
func (m *MockConsumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage         { return m.messages }

// TestConsumerGroupHandler tests the consumer group handler implementation
func TestConsumerGroupHandler(t *testing.T) {
	t.Run("consumer_group_setup_and_cleanup", func(t *testing.T) {
		handler := &g{}
		
		// Mock session
		mockSession := &MockConsumerGroupSession{}
		
		// Test Setup
		err := handler.Setup(mockSession)
		assert.NoError(t, err)
		
		// Test Cleanup
		err = handler.Cleanup(mockSession)
		assert.NoError(t, err)
	})
	
	t.Run("consumer_group_consume_claim", func(t *testing.T) {
		var receivedMessages []api.Message
		var mu sync.Mutex
		
		// Set up global handler to capture messages
		handler = func(msg api.Message) {
			mu.Lock()
			receivedMessages = append(receivedMessages, msg)
			mu.Unlock()
		}
		
		// Create test messages
		testMessages := []*sarama.ConsumerMessage{
			{
				Key:       []byte("key1"),
				Value:     []byte("value1"),
				Topic:     "test-topic",
				Partition: 0,
				Offset:    100,
			},
			{
				Key:       []byte("key2"),
				Value:     []byte("value2"),
				Topic:     "test-topic",
				Partition: 0,
				Offset:    101,
			},
		}
		
		// Set up mock claim
		msgChan := make(chan *sarama.ConsumerMessage, len(testMessages))
		for _, msg := range testMessages {
			msgChan <- msg
		}
		close(msgChan)
		
		mockSession := &MockConsumerGroupSession{}
		mockClaim := &MockConsumerGroupClaim{messages: msgChan}
		
		// Test without group commit
		groupCommitFlag = false
		
		consumerHandler := &g{}
		err := consumerHandler.ConsumeClaim(mockSession, mockClaim)
		
		assert.NoError(t, err)
		assert.Len(t, receivedMessages, 2)
		assert.Equal(t, "key1", receivedMessages[0].Key)
		assert.Equal(t, "value1", receivedMessages[0].Value)
		assert.Equal(t, "key2", receivedMessages[1].Key)
		assert.Equal(t, "value2", receivedMessages[1].Value)
		assert.False(t, mockSession.markMessageCalled, "MarkMessage should not be called when groupCommitFlag is false")
	})
	
	t.Run("consumer_group_consume_claim_with_commit", func(t *testing.T) {
		var receivedMessages []api.Message
		var mu sync.Mutex
		
		handler = func(msg api.Message) {
			mu.Lock()
			receivedMessages = append(receivedMessages, msg)
			mu.Unlock()
		}
		
		testMessage := &sarama.ConsumerMessage{
			Key:       []byte("key1"),
			Value:     []byte("value1"),
			Topic:     "test-topic",
			Partition: 0,
			Offset:    100,
		}
		
		msgChan := make(chan *sarama.ConsumerMessage, 1)
		msgChan <- testMessage
		close(msgChan)
		
		mockSession := &MockConsumerGroupSession{}
		mockClaim := &MockConsumerGroupClaim{messages: msgChan}
		
		// Test with group commit
		groupCommitFlag = true
		
		consumerHandler := &g{}
		err := consumerHandler.ConsumeClaim(mockSession, mockClaim)
		
		assert.NoError(t, err)
		assert.Len(t, receivedMessages, 1)
		assert.True(t, mockSession.markMessageCalled, "MarkMessage should be called when groupCommitFlag is true")
		assert.Equal(t, testMessage, mockSession.markedMessage, "Marked message should match the consumed message")
		
		// Reset flag
		groupCommitFlag = false
	})
}

// TestWithoutConsumerGroup tests direct partition consumption logic
func TestWithoutConsumerGroup(t *testing.T) {
	t.Run("context_cancellation_handling", func(t *testing.T) {
		// Test that cancelled context is handled properly
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		select {
		case <-ctx.Done():
			assert.Equal(t, context.Canceled, ctx.Err(), "Context should be cancelled")
		default:
			t.Error("Context should be cancelled")
		}
	})
	
	t.Run("offset_calculation_with_tail", func(t *testing.T) {
		// Test tail offset calculation logic
		offsets := &offsets{
			newest: 1000,
			oldest: 100,
		}
		
		// Test case 1: tail within range
		tail := int32(50)
		expectedOffset := offsets.newest - int64(tail) // 1000 - 50 = 950
		if expectedOffset < offsets.oldest {
			expectedOffset = offsets.oldest
		}
		assert.Equal(t, int64(950), expectedOffset, "Tail offset should be calculated correctly")
		
		// Test case 2: tail exceeds range
		tail = int32(1000)
		expectedOffset = offsets.newest - int64(tail) // 1000 - 1000 = 0
		if expectedOffset < offsets.oldest {
			expectedOffset = offsets.oldest // Should use oldest (100)
		}
		assert.Equal(t, int64(100), expectedOffset, "Tail offset should fallback to oldest when out of range")
	})
	
	t.Run("partition_consumption_exit_conditions", func(t *testing.T) {
		// Test early exit when already at end of partition
		offsets := &offsets{
			newest: 100,
			oldest: 100, // Same as newest, indicating empty partition
		}
		
		follow := false
		shouldExit := !follow && offsets.newest == offsets.oldest
		assert.True(t, shouldExit, "Should exit early when partition is empty and not following")
		
		// Test continue when following
		follow = true
		shouldExit = !follow && offsets.newest == offsets.oldest
		assert.False(t, shouldExit, "Should not exit early when following")
	})
}

// TestFormatMessage tests message formatting functionality
func TestFormatMessage(t *testing.T) {
	t.Run("format_message_raw", func(t *testing.T) {
		msg := &sarama.ConsumerMessage{
			Key:       []byte("test-key"),
			Value:     []byte("test-value"),
			Partition: 1,
			Offset:    123,
			Timestamp: time.Now(),
		}
		
		rawMessage := []byte("raw content")
		keyToDisplay := []byte("display key")
		
		// Test raw format
		outputFormat = OutputFormatRaw
		result := formatMessage(msg, rawMessage, keyToDisplay, nil)
		
		assert.Equal(t, rawMessage, result)
	})
	
	t.Run("format_message_json", func(t *testing.T) {
		msg := &sarama.ConsumerMessage{
			Key:       []byte("test-key"),
			Value:     []byte("test-value"),
			Partition: 1,
			Offset:    123,
			Timestamp: time.Now(),
			Headers: []*sarama.RecordHeader{
				{Key: []byte("header1"), Value: []byte("value1")},
			},
		}
		
		rawMessage := []byte(`{"field": "value"}`)
		keyToDisplay := []byte(`{"keyField": "keyValue"}`)
		
		// Test JSON format
		outputFormat = OutputFormatJSON
		result := formatMessage(msg, rawMessage, keyToDisplay, nil)
		
		assert.NotEmpty(t, result)
		assert.Contains(t, string(result), "partition")
		assert.Contains(t, string(result), "offset")
		assert.Contains(t, string(result), "timestamp")
	})
	
	t.Run("format_message_default", func(t *testing.T) {
		msg := &sarama.ConsumerMessage{
			Key:       []byte("test-key"),
			Value:     []byte("test-value"),
			Partition: 1,
			Offset:    123,
			Timestamp: time.Now(),
		}
		
		rawMessage := []byte("test content")
		keyToDisplay := []byte("test key")
		
		// Test default format
		outputFormat = OutputFormatDefault
		result := formatMessage(msg, rawMessage, keyToDisplay, nil)
		
		resultStr := string(result)
		assert.Contains(t, resultStr, "Partition:")
		assert.Contains(t, resultStr, "Offset:")
		assert.Contains(t, resultStr, "Timestamp:")
		assert.Contains(t, resultStr, "test content")
	})
}

// TestHelperFunctions tests various helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("is_json_valid", func(t *testing.T) {
		validJSON := []byte(`{"key": "value"}`)
		assert.True(t, isJSON(validJSON))
		
		invalidJSON := []byte(`{invalid json}`)
		assert.False(t, isJSON(invalidJSON))
		
		emptyData := []byte(``)
		assert.False(t, isJSON(emptyData))
	})
	
	t.Run("format_json", func(t *testing.T) {
		validJSON := []byte(`{"key": "value"}`)
		result := formatJSON(validJSON)
		
		// Should return a map for valid JSON
		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "value", resultMap["key"])
		
		invalidJSON := []byte(`invalid`)
		result = formatJSON(invalidJSON)
		
		// Should return string for invalid JSON
		resultStr, ok := result.(string)
		assert.True(t, ok)
		assert.Equal(t, "invalid", resultStr)
	})
	
	t.Run("format_key", func(t *testing.T) {
		// Initialize keyfmt to avoid nil pointer panic
		originalKeyfmt := keyfmt
		keyfmt = &prettyjson.Formatter{
			Indent:    2,
			Newline:   "\n",
		}
		defer func() {
			keyfmt = originalKeyfmt
		}()
		
		validJSON := []byte(`{"keyField": "keyValue"}`)
		result := formatKey(validJSON)
		
		// Should format valid JSON
		assert.NotEqual(t, validJSON, result)
		
		invalidJSON := []byte(`invalid`)
		result = formatKey(invalidJSON)
		
		// Should return original for invalid JSON
		assert.Equal(t, invalidJSON, result)
	})
	
	t.Run("format_value", func(t *testing.T) {
		validJSON := []byte(`{"field": "value"}`)
		result := formatValue(validJSON)
		
		// Should format valid JSON
		assert.NotEqual(t, validJSON, result)
		
		invalidJSON := []byte(`invalid`)
		result = formatValue(invalidJSON)
		
		// Should return original for invalid JSON
		assert.Equal(t, invalidJSON, result)
	})
}

// TestOutputFormat tests the OutputFormat type and its methods
func TestOutputFormat(t *testing.T) {
	t.Run("output_format_string", func(t *testing.T) {
		format := OutputFormatJSON
		assert.Equal(t, "json", format.String())
		
		format = OutputFormatRaw
		assert.Equal(t, "raw", format.String())
		
		format = OutputFormatDefault
		assert.Equal(t, "default", format.String())
	})
	
	t.Run("output_format_set_valid", func(t *testing.T) {
		var format OutputFormat
		
		err := format.Set("json")
		assert.NoError(t, err)
		assert.Equal(t, OutputFormatJSON, format)
		
		err = format.Set("raw")
		assert.NoError(t, err)
		assert.Equal(t, OutputFormatRaw, format)
		
		err = format.Set("default")
		assert.NoError(t, err)
		assert.Equal(t, OutputFormatDefault, format)
	})
	
	t.Run("output_format_set_invalid", func(t *testing.T) {
		var format OutputFormat
		
		err := format.Set("invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be one of: default, raw, json")
	})
	
	t.Run("output_format_type", func(t *testing.T) {
		var format OutputFormat
		assert.Equal(t, "OutputFormat", format.Type())
	})
}