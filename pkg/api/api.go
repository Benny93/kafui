package api

import "context"

type MessageHeader struct {
	Key   string
	Value string
}
type MessageHeaders []MessageHeader

type Message struct {
	Key           string
	Value         string
	Offset        int64
	Partition     int32
	KeySchemaID   string
	ValueSchemaID string
	Headers       []MessageHeader
}

type Topic struct {
	// NumPartitions contains the number of partitions to create in the topic
	NumPartitions int32
	// ReplicationFactor contains the number of replicas to create for each partition
	ReplicationFactor int16
	// ReplicaAssignment contains the manual partition assignment, or the empty
	// array if we are using automatic assignment.
	ReplicaAssignment map[int32][]int32
	// ConfigEntries contains the custom topic configurations to set.
	ConfigEntries map[string]*string
	// Num of messages in the topic across all partitions
	MessageCount int64
}

type ConsumeFlags struct {
	Follow     bool
	Tail       int32
	OffsetFlag string
	GroupFlag  string
}

func DefaultConsumeFlags() ConsumeFlags {
	return ConsumeFlags{
		Follow:     true,
		Tail:       50,
		OffsetFlag: "latest",
	}
}

type ConsumerGroup struct {
	Name      string
	State     string
	Consumers int
}

// SchemaInfo represents Avro schema information
type SchemaInfo struct {
	ID         int    `json:"id"`
	Schema     string `json:"schema"`
	Subject    string `json:"subject"`
	Version    int    `json:"version"`
	RecordName string `json:"recordName"` // The type name (e.g., AddedItemToChartEvent)
}

// MessageSchemaInfo contains schema information for a message's key and value
type MessageSchemaInfo struct {
	KeySchema   *SchemaInfo `json:"keySchema,omitempty"`
	ValueSchema *SchemaInfo `json:"valueSchema,omitempty"`
}

type MessageHandlerFunc func(msg Message)

type KafkaDataSource interface {
	Init(cfgOption string)
	GetTopics() (map[string]Topic, error)
	GetContexts() ([]string, error)
	GetContext() string
	SetContext(contextName string) error
	GetConsumerGroups() ([]ConsumerGroup, error)
	ConsumeTopic(ctx context.Context, topicName string, flags ConsumeFlags, handleMessage MessageHandlerFunc, onError func(err any)) error
	// GetMessageSchemaInfo retrieves schema information for a message's key and value
	// Returns nil for non-Avro messages or when schema information is not available
	GetMessageSchemaInfo(keySchemaID, valueSchemaID string) (*MessageSchemaInfo, error)
}
