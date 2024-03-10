package api

type Message struct {
	Key   string
	Value string
}

type MessageHandlerFunc func(msg Message)

type KafkaDataSource interface {
	GetTopics() ([]string, error)
	GetContexts() ([]string, error)
	GetConsumerGroups() ([]string, error)
	ConsumeTopic(topicName string, handleMessage MessageHandlerFunc) error
}
