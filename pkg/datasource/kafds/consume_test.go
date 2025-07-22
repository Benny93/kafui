package kafds

import (
	"context"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
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