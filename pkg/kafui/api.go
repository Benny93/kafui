package kafui

type KafkaDataSource interface {
	GetTopics() ([]string, error)
	GetContexts() ([]string, error)
	GetConsumerGroups() ([]string, error)
}
