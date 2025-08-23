package mock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Benny93/kafui/pkg/api"
)

var currentContext string = "kafka-dev"

type KafkaDataSourceMock struct {
	// Schema cache for performance - cached on data source side
	schemaCache map[string]*api.SchemaInfo
	schemaMutex sync.RWMutex
}

func (kp *KafkaDataSourceMock) Init(cfgOption string) {
	// Initialize schema cache
	kp.schemaCache = make(map[string]*api.SchemaInfo)
}

// SetContext implements api.KafkaDataSource.
func (kp *KafkaDataSourceMock) SetContext(contextName string) error {
	currentContext = contextName
	return nil
}

// GetTopics retrieves a list of Kafka topics
func (kp *KafkaDataSourceMock) GetTopics() (map[string]api.Topic, error) {
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

func (kp *KafkaDataSourceMock) GetContext() string {
	return currentContext
}

// GetContexts retrieves a list of Kafka contexts
func (kp *KafkaDataSourceMock) GetContexts() ([]string, error) {
	// Logic to fetch the list of contexts from Kafka
	contexts := []string{"kafka-dev", "kafka-test", "kafka-prod"} // Example contexts
	return contexts, nil
}

func (kp *KafkaDataSourceMock) GetConsumerGroups() ([]api.ConsumerGroup, error) {
	// Mocked data
	groups := []api.ConsumerGroup{
		{Name: "Group1", State: "Active", Consumers: 3},
		{Name: "Group2", State: "Idle", Consumers: 2},
		// Add more mock ConsumerGroup structs as needed
	}

	// Return mocked data
	return groups, nil
}

func (kp *KafkaDataSourceMock) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) error {
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
		
		// Simulate some messages having Avro schemas (about 30% of messages)
		if messageIndex%3 == 0 {
			// Simulate key schema for some messages
			if messageIndex%6 == 0 {
				msg.KeySchemaID = fmt.Sprintf("%d", (messageIndex%4)+1)
			}
			// Simulate value schema
			msg.ValueSchemaID = fmt.Sprintf("%d", (messageIndex%5)+1)
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

// GetMessageSchemaInfo implements api.KafkaDataSource
func (kp *KafkaDataSourceMock) GetMessageSchemaInfo(keySchemaID, valueSchemaID string) (*api.MessageSchemaInfo, error) {
	schemaInfo := &api.MessageSchemaInfo{}
	
	// Handle key schema if provided
	if keySchemaID != "" {
		if schema := kp.getSchemaFromCache(keySchemaID); schema != nil {
			schemaInfo.KeySchema = schema
		} else {
			// Simulate fetching from schema registry
			keySchema := kp.simulateSchemaFetch(keySchemaID, "key")
			if keySchema != nil {
				kp.cacheSchema(keySchemaID, keySchema)
				schemaInfo.KeySchema = keySchema
			}
		}
	}
	
	// Handle value schema if provided
	if valueSchemaID != "" {
		if schema := kp.getSchemaFromCache(valueSchemaID); schema != nil {
			schemaInfo.ValueSchema = schema
		} else {
			// Simulate fetching from schema registry
			valueSchema := kp.simulateSchemaFetch(valueSchemaID, "value")
			if valueSchema != nil {
				kp.cacheSchema(valueSchemaID, valueSchema)
				schemaInfo.ValueSchema = valueSchema
			}
		}
	}
	
	// Return nil if no schema information is available
	if schemaInfo.KeySchema == nil && schemaInfo.ValueSchema == nil {
		return nil, nil
	}
	
	return schemaInfo, nil
}

// getSchemaFromCache retrieves schema from cache (thread-safe)
func (kp *KafkaDataSourceMock) getSchemaFromCache(schemaID string) *api.SchemaInfo {
	kp.schemaMutex.RLock()
	defer kp.schemaMutex.RUnlock()
	return kp.schemaCache[schemaID]
}

// cacheSchema stores schema in cache (thread-safe)
func (kp *KafkaDataSourceMock) cacheSchema(schemaID string, schema *api.SchemaInfo) {
	kp.schemaMutex.Lock()
	defer kp.schemaMutex.Unlock()
	kp.schemaCache[schemaID] = schema
}

// simulateSchemaFetch simulates fetching schema from registry
func (kp *KafkaDataSourceMock) simulateSchemaFetch(schemaID, schemaType string) *api.SchemaInfo {
	// Simulate some common Avro schema types
	mockSchemas := map[string]*api.SchemaInfo{
		"1": {
			ID:         1,
			Subject:    "user-events-value",
			Version:    1,
			RecordName: "UserRegisteredEvent",
			Schema:     `{"type":"record","name":"UserRegisteredEvent","fields":[{"name":"userId","type":"string"},{"name":"email","type":"string"}]}`},
		"2": {
			ID:         2,
			Subject:    "order-events-value",
			Version:    1,
			RecordName: "OrderCreatedEvent",
			Schema:     `{"type":"record","name":"OrderCreatedEvent","fields":[{"name":"orderId","type":"string"},{"name":"amount","type":"double"}]}`},
		"3": {
			ID:         3,
			Subject:    "product-events-value",
			Version:    2,
			RecordName: "AddedItemToCartEvent",
			Schema:     `{"type":"record","name":"AddedItemToCartEvent","fields":[{"name":"productId","type":"string"},{"name":"quantity","type":"int"}]}`},
		"4": {
			ID:         4,
			Subject:    "inventory-events-key",
			Version:    1,
			RecordName: "InventoryKey",
			Schema:     `{"type":"record","name":"InventoryKey","fields":[{"name":"warehouseId","type":"string"},{"name":"productId","type":"string"}]}`},
		"5": {
			ID:         5,
			Subject:    "payment-events-value",
			Version:    1,
			RecordName: "PaymentProcessedEvent",
			Schema:     `{"type":"record","name":"PaymentProcessedEvent","fields":[{"name":"paymentId","type":"string"},{"name":"status","type":"string"}]}`},
	}
	
	// Return mock schema if it exists
	if schema, exists := mockSchemas[schemaID]; exists {
		return schema
	}
	
	// Return nil for non-Avro or unknown schemas
	return nil
}
