package api

import (
	"reflect"
	"testing"
)

// TestMessageHeader tests the MessageHeader struct
func TestMessageHeader(t *testing.T) {
	tests := []struct {
		name   string
		header MessageHeader
		want   MessageHeader
	}{
		{
			name:   "valid header",
			header: MessageHeader{Key: "content-type", Value: "application/json"},
			want:   MessageHeader{Key: "content-type", Value: "application/json"},
		},
		{
			name:   "empty header",
			header: MessageHeader{},
			want:   MessageHeader{Key: "", Value: ""},
		},
		{
			name:   "header with special characters",
			header: MessageHeader{Key: "x-custom-header", Value: "value with spaces & symbols!"},
			want:   MessageHeader{Key: "x-custom-header", Value: "value with spaces & symbols!"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.header, tt.want) {
				t.Errorf("MessageHeader = %v, want %v", tt.header, tt.want)
			}
		})
	}
}

// TestMessageHeaders tests the MessageHeaders type
func TestMessageHeaders(t *testing.T) {
	headers := MessageHeaders{
		{Key: "header1", Value: "value1"},
		{Key: "header2", Value: "value2"},
	}

	if len(headers) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(headers))
	}

	if headers[0].Key != "header1" || headers[0].Value != "value1" {
		t.Errorf("First header incorrect: got %+v", headers[0])
	}
}

// TestMessage tests the Message struct
func TestMessage(t *testing.T) {
	tests := []struct {
		name    string
		message Message
		want    Message
	}{
		{
			name: "complete message",
			message: Message{
				Key:           "test-key",
				Value:         "test-value",
				Offset:        100,
				Partition:     1,
				KeySchemaID:   "key-schema-1",
				ValueSchemaID: "value-schema-1",
				Headers: []MessageHeader{
					{Key: "content-type", Value: "application/json"},
				},
			},
			want: Message{
				Key:           "test-key",
				Value:         "test-value",
				Offset:        100,
				Partition:     1,
				KeySchemaID:   "key-schema-1",
				ValueSchemaID: "value-schema-1",
				Headers: []MessageHeader{
					{Key: "content-type", Value: "application/json"},
				},
			},
		},
		{
			name:    "empty message",
			message: Message{},
			want:    Message{},
		},
		{
			name: "message with negative offset",
			message: Message{
				Offset:    -1,
				Partition: 0,
			},
			want: Message{
				Offset:    -1,
				Partition: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.message, tt.want) {
				t.Errorf("Message = %v, want %v", tt.message, tt.want)
			}
		})
	}
}

// TestTopic tests the Topic struct
func TestTopic(t *testing.T) {
	tests := []struct {
		name  string
		topic Topic
		want  Topic
	}{
		{
			name: "complete topic configuration",
			topic: Topic{
				NumPartitions:     3,
				ReplicationFactor: 2,
				ReplicaAssignment: map[int32][]int32{
					0: {1, 2},
					1: {2, 3},
					2: {3, 1},
				},
				ConfigEntries: map[string]*string{
					"cleanup.policy": stringPtr("compact"),
					"retention.ms":   stringPtr("86400000"),
				},
				MessageCount: 1000,
			},
			want: Topic{
				NumPartitions:     3,
				ReplicationFactor: 2,
				ReplicaAssignment: map[int32][]int32{
					0: {1, 2},
					1: {2, 3},
					2: {3, 1},
				},
				ConfigEntries: map[string]*string{
					"cleanup.policy": stringPtr("compact"),
					"retention.ms":   stringPtr("86400000"),
				},
				MessageCount: 1000,
			},
		},
		{
			name:  "minimal topic",
			topic: Topic{NumPartitions: 1, ReplicationFactor: 1},
			want:  Topic{NumPartitions: 1, ReplicationFactor: 1},
		},
		{
			name:  "empty topic",
			topic: Topic{},
			want:  Topic{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.topic, tt.want) {
				t.Errorf("Topic = %v, want %v", tt.topic, tt.want)
			}
		})
	}
}

// TestConsumeFlags tests the ConsumeFlags struct
func TestConsumeFlags(t *testing.T) {
	tests := []struct {
		name  string
		flags ConsumeFlags
		want  ConsumeFlags
	}{
		{
			name:  "custom flags",
			flags: ConsumeFlags{Follow: false, Tail: 100, OffsetFlag: "earliest"},
			want:  ConsumeFlags{Follow: false, Tail: 100, OffsetFlag: "earliest"},
		},
		{
			name:  "zero values",
			flags: ConsumeFlags{},
			want:  ConsumeFlags{Follow: false, Tail: 0, OffsetFlag: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.flags, tt.want) {
				t.Errorf("ConsumeFlags = %v, want %v", tt.flags, tt.want)
			}
		})
	}
}

// TestDefaultConsumeFlags tests the DefaultConsumeFlags function
func TestDefaultConsumeFlags(t *testing.T) {
	expected := ConsumeFlags{
		Follow:     true,
		Tail:       50,
		OffsetFlag: "latest",
	}

	result := DefaultConsumeFlags()

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("DefaultConsumeFlags() = %v, want %v", result, expected)
	}
}

// TestConsumerGroup tests the ConsumerGroup struct
func TestConsumerGroup(t *testing.T) {
	tests := []struct {
		name  string
		group ConsumerGroup
		want  ConsumerGroup
	}{
		{
			name:  "active consumer group",
			group: ConsumerGroup{Name: "test-group", State: "Stable", Consumers: 3},
			want:  ConsumerGroup{Name: "test-group", State: "Stable", Consumers: 3},
		},
		{
			name:  "empty consumer group",
			group: ConsumerGroup{},
			want:  ConsumerGroup{Name: "", State: "", Consumers: 0},
		},
		{
			name:  "rebalancing group",
			group: ConsumerGroup{Name: "rebalancing-group", State: "PreparingRebalance", Consumers: 2},
			want:  ConsumerGroup{Name: "rebalancing-group", State: "PreparingRebalance", Consumers: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.group, tt.want) {
				t.Errorf("ConsumerGroup = %v, want %v", tt.group, tt.want)
			}
		})
	}
}

// TestMessageHandlerFunc tests the MessageHandlerFunc type
func TestMessageHandlerFunc(t *testing.T) {
	var handledMessage Message
	handler := MessageHandlerFunc(func(msg Message) {
		handledMessage = msg
	})

	testMessage := Message{
		Key:       "test-key",
		Value:     "test-value",
		Offset:    42,
		Partition: 1,
	}

	handler(testMessage)

	if !reflect.DeepEqual(handledMessage, testMessage) {
		t.Errorf("MessageHandlerFunc did not handle message correctly: got %v, want %v", handledMessage, testMessage)
	}
}

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}

// Benchmark tests for performance-critical operations
func BenchmarkDefaultConsumeFlags(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultConsumeFlags()
	}
}

func BenchmarkMessageCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Message{
			Key:       "benchmark-key",
			Value:     "benchmark-value",
			Offset:    int64(i),
			Partition: 1,
			Headers: []MessageHeader{
				{Key: "content-type", Value: "application/json"},
			},
		}
	}
}