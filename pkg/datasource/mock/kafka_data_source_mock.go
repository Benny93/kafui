package mock

type KafkaDataSourceMock struct {
	// Additional fields can be added here if needed
}

// GetTopics retrieves a list of Kafka topics
func (kp KafkaDataSourceMock) GetTopics() ([]string, error) {
	// Logic to fetch the list of topics from Kafka
	topics := []string{
		"topic1",
		"topic2",
		"topic3",
		"topic4",
		"topic5",
		"topic6",
		"topic7",
		"topic8",
		"topic9",
		"topic10",
	} // Additional topics
	return topics, nil
}

// GetContexts retrieves a list of Kafka contexts
func (kp KafkaDataSourceMock) GetContexts() ([]string, error) {
	// Logic to fetch the list of contexts from Kafka
	contexts := []string{"kafka-dev", "kafka-test", "kafka-prod"} // Example contexts
	return contexts, nil
}

func (kp KafkaDataSourceMock) GetConsumerGroups() ([]string, error) {
	cgs := []string{"consumer1", "consumer2", "consumer3"} // Example
	return cgs, nil
}

func (kp KafkaDataSourceMock) ConsumeTopic(topicName string) ([]string, error) {
	cgs := []string{"message1", "message2", "message3"} // Example
	return cgs, nil
}
