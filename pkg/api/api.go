package api

type Message struct {
	Key   string
	Value string
}

type ConsumerGroup struct {
	Name      string
	State     string
	Consumers int
}

type MessageHandlerFunc func(msg Message)

type KafkaDataSource interface {
	GetTopics() ([]string, error)
	GetContexts() ([]string, error)
	GetConsumerGroups() ([]ConsumerGroup, error)
	ConsumeTopic(topicName string, handleMessage MessageHandlerFunc) error
}
