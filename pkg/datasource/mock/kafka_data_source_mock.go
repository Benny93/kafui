package mock

import (
	"com/emptystate/kafui/pkg/api"
	"fmt"
	"time"
)

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

func (kp KafkaDataSourceMock) ConsumeTopic(topicName string, handleMessage api.MessageHandlerFunc) error {
	// Simulate consuming messages from the topic
	for i := 0; i < 5; i++ {
		// Simulate receiving a message
		msg := api.Message{
			Key:   fmt.Sprintf("purchase_%s_%d", topicName, i),
			Value: fmt.Sprintf(`{"product_id": %d, "quantity": %d, "timestamp": "%s"}`, i+1, i*2+1, time.Now().Format(time.RFC3339)),
		}

		// Call the message handler function
		handleMessage(msg)

		// Simulate some processing time
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}
