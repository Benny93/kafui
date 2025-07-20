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

type MessageHandlerFunc func(msg Message)

type KafkaDataSource interface {
	Init(cfgOption string)
	GetTopics() (map[string]Topic, error)
	GetContexts() ([]string, error)
	GetContext() string
	SetContext(contextName string) error
	GetConsumerGroups() ([]ConsumerGroup, error)
	ConsumeTopic(ctx context.Context, topicName string, flags ConsumeFlags, handleMessage MessageHandlerFunc, onError func(err any)) error
}
