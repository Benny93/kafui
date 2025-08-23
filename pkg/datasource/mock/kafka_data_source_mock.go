package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/Benny93/kafui/pkg/api"
)

var currentContext string = "kafka-dev"

type KafkaDataSourceMock struct {
	// Additional fields can be added here if needed
}

func (kp KafkaDataSourceMock) Init(cfgOption string) {
	// nothing todo here
}

// SetContext implements api.KafkaDataSource.
func (kp KafkaDataSourceMock) SetContext(contextName string) error {
	currentContext = contextName
	return nil
}

// GetTopics retrieves a list of Kafka topics
func (kp KafkaDataSourceMock) GetTopics() (map[string]api.Topic, error) {
	// Logic to fetch the list of topics from Kafka
	topics := make(map[string]api.Topic)
	for i := 0; i < 100; i++ {
		topics[fmt.Sprintf("Topic %d", i)] = api.Topic{
			ReplicationFactor: 1,
			ReplicaAssignment: map[int32][]int32{},
			NumPartitions:     1,
			ConfigEntries:     make(map[string]*string),
		}

	} // Additional topics

	return topics, nil
}

func (kp KafkaDataSourceMock) GetContext() string {
	return currentContext
}

// GetContexts retrieves a list of Kafka contexts
func (kp KafkaDataSourceMock) GetContexts() ([]string, error) {
	// Logic to fetch the list of contexts from Kafka
	contexts := []string{"kafka-dev", "kafka-test", "kafka-prod"} // Example contexts
	return contexts, nil
}

func (kp KafkaDataSourceMock) GetConsumerGroups() ([]api.ConsumerGroup, error) {
	// Mocked data
	groups := []api.ConsumerGroup{
		{Name: "Group1", State: "Active", Consumers: 3},
		{Name: "Group2", State: "Idle", Consumers: 2},
		// Add more mock ConsumerGroup structs as needed
	}

	// Return mocked data
	return groups, nil
}

func (kp KafkaDataSourceMock) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) error {
	// Simulate initial connection delay
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(50 * time.Millisecond):
	}

	// Simulate continuous message consumption like real Kafka
	// Keep generating messages until context is cancelled
	messageIndex := 0
	for {
		// Check if context is cancelled before processing
		select {
		case <-ctx.Done():
			return ctx.Err() // Return when context is cancelled
		default:
			// Continue processing
		}

		description := "Lorem ipsum dolor sit amet con et just me incididunt ut lab inductor laris martinus"
		// Simulate receiving a message
		msg := api.Message{
			Key:       fmt.Sprintf("purchase_%s_%d", topicName, messageIndex),
			Value:     fmt.Sprintf(`{"product_id": %d, "quantity": %d, "timestamp": "%s", "description": "%s"}`, messageIndex+1, messageIndex*2+1, time.Now().Format(time.RFC3339), description),
			Offset:    int64(messageIndex + 1),
			Partition: int32(messageIndex % 3), // Distribute across 3 partitions
		}

		// Check context again before calling handler with panic recovery
		select {
		case <-ctx.Done():
			return ctx.Err() // Context cancelled, stop processing
		default:
			// Call the message handler function with panic recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Handler panicked (likely due to closed channel)
						// Call onError if context is still active
						select {
						case <-ctx.Done():
							// Context cancelled, don't call onError
							return
						default:
							onError(fmt.Errorf("panic in message handler: %v", r))
						}
					}
				}()
				handleMessage(msg)
			}()
		}

		messageIndex++

		// Simulate realistic processing time between messages with context check
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond): // Slower message rate for better visibility
			// Continue to next iteration
		}
	}
}
