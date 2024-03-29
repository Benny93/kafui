package api

import "context"

type Message struct {
	Key       string
	Value     string
	Offset    int64
	Partition int32
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
	Init()
	GetTopics() (map[string]Topic, error)
	GetContexts() ([]string, error)
	GetContext() string
	SetContext(contextName string) error
	GetConsumerGroups() ([]ConsumerGroup, error)
	ConsumeTopic(ctx context.Context, topicName string, flags ConsumeFlags, handleMessage MessageHandlerFunc) error
}
