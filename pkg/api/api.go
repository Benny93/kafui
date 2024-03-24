package api

import "context"

type Message struct {
	Key    string
	Value  string
	Offset int64
}

type ConsumerGroup struct {
	Name      string
	State     string
	Consumers int
}

type MessageHandlerFunc func(msg Message)

type KafkaDataSource interface {
	Init()
	GetTopics() ([]string, error)
	GetContexts() ([]string, error)
	GetContext() string
	SetContext(contextName string) error
	GetConsumerGroups() ([]ConsumerGroup, error)
	ConsumeTopic(ctx context.Context, topicName string, handleMessage MessageHandlerFunc) error
}
