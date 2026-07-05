package mock

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/serde"
)

var currentContext string = "kafka-dev"

// Mock data stores for different contexts
var mockContexts = map[string]*mockContextData{
	"kafka-dev": {
		topics:         generateDevTopics(),
		consumerGroups: generateDevConsumerGroups(),
		description:    "Development Kafka Cluster",
	},
	"kafka-test": {
		topics:         generateTestTopics(),
		consumerGroups: generateTestConsumerGroups(),
		description:    "Test Kafka Cluster",
	},
	"kafka-prod": {
		topics:         generateProdTopics(),
		consumerGroups: generateProdConsumerGroups(),
		description:    "Production Kafka Cluster",
	},
}

type mockContextData struct {
	topics         map[string]api.Topic
	consumerGroups []api.ConsumerGroup
	description    string
}

type KafkaDataSourceMock struct {
	// Schema cache for performance - cached on data source side
	schemaCache map[string]*api.SchemaInfo
	schemaMutex sync.RWMutex
	// Message counters per topic for realistic offset tracking
	messageCounters map[string]int64
	counterMutex    sync.RWMutex
	// Random seed for varied data
	randSource *rand.Rand
	// Broker mock state (lazily initialised; see broker.go).
	brokerMu      sync.Mutex
	brokers       []api.BrokerInfo
	brokerConfigs map[int32][]api.BrokerConfigEntry
	brokerLogDirs map[int32][]api.BrokerLogDir
	// Consumer-group detail/mutation mock state (lazily initialised; see
	// consumer_groups.go).
	groupMu sync.Mutex
	groups  map[string]*mockGroup
	// Topic-administration + analysis mock state (see topic_admin.go).
	topicMu          sync.Mutex
	deletionDisabled bool
	analyses         map[string]*api.TopicAnalysis
	// Produced messages, kept in-memory so they are browsable via ConsumeTopic
	// (MSG-30). Keyed by topic; per-partition offsets tracked in producedOffsets.
	producedMu      sync.Mutex
	produced        map[string][]api.Message
	producedOffsets map[string]map[int32]int64
	// In-memory schema registry state (lazily initialised; see schema_registry.go).
	schemaRegMu sync.Mutex
	schemaReg   *mockRegistry
	// In-memory ACL + client-quota state (lazily initialised; see acls_quotas.go).
	aclMu      sync.Mutex
	acls       []api.ACLEntry
	aclsInit   bool
	quotaMu    sync.Mutex
	quotas     []api.ClientQuotaEntry
	quotasInit bool
	// In-memory Kafka Connect state (lazily initialised; see connect.go).
	connectMu    sync.Mutex
	connectState *mockConnectState
}

func (kp *KafkaDataSourceMock) Init(cfgOption string) {
	// Initialize schema cache
	kp.schemaCache = make(map[string]*api.SchemaInfo)
	// Initialize message counters
	kp.messageCounters = make(map[string]int64)
	// Initialize random source
	kp.randSource = rand.New(rand.NewSource(time.Now().UnixNano()))
	// Initialize produced-message store
	kp.produced = make(map[string][]api.Message)
	kp.producedOffsets = make(map[string]map[int32]int64)

	// Pre-populate schema cache with common schemas
	kp.preloadSchemas()
}

// preloadSchemas loads common schemas into cache
func (kp *KafkaDataSourceMock) preloadSchemas() {
	schemas := map[string]*api.SchemaInfo{
		"1": {
			ID:         1,
			Subject:    "user-events-value",
			Version:    1,
			RecordName: "UserRegisteredEvent",
			Schema:     `{"type":"record","name":"UserRegisteredEvent","namespace":"com.example.user","fields":[{"name":"userId","type":"string"},{"name":"email","type":"string"},{"name":"registeredAt","type":"long"},{"name":"preferences","type":{"type":"map","values":"string"}}]}`,
		},
		"2": {
			ID:         2,
			Subject:    "order-events-value",
			Version:    1,
			RecordName: "OrderCreatedEvent",
			Schema:     `{"type":"record","name":"OrderCreatedEvent","namespace":"com.example.orders","fields":[{"name":"orderId","type":"string"},{"name":"customerId","type":"string"},{"name":"amount","type":"double"},{"name":"items","type":{"type":"array","items":"string"}},{"name":"createdAt","type":"long"}]}`,
		},
		"3": {
			ID:         3,
			Subject:    "order-events-value",
			Version:    2,
			RecordName: "OrderCreatedEvent",
			Schema:     `{"type":"record","name":"OrderCreatedEvent","namespace":"com.example.orders","fields":[{"name":"orderId","type":"string"},{"name":"customerId","type":"string"},{"name":"amount","type":"double"},{"name":"items","type":{"type":"array","items":"OrderItem"}},{"name":"createdAt","type":"long"},{"name":"metadata","type":["null",{"type":"map","values":"string"}],"default":null}],"types":[{"name":"OrderItem","type":"record","fields":[{"name":"productId","type":"string"},{"name":"quantity","type":"int"},{"name":"price","type":"double"}]}]}`,
		},
		"4": {
			ID:         4,
			Subject:    "inventory-events-key",
			Version:    1,
			RecordName: "InventoryKey",
			Schema:     `{"type":"record","name":"InventoryKey","namespace":"com.example.inventory","fields":[{"name":"warehouseId","type":"string"},{"name":"productId","type":"string"}]}`,
		},
		"5": {
			ID:         5,
			Subject:    "payment-events-value",
			Version:    1,
			RecordName: "PaymentProcessedEvent",
			Schema:     `{"type":"record","name":"PaymentProcessedEvent","namespace":"com.example.payments","fields":[{"name":"paymentId","type":"string"},{"name":"orderId","type":"string"},{"name":"amount","type":"double"},{"name":"status","type":{"type":"enum","name":"PaymentStatus","symbols":["SUCCESS","FAILED","PENDING"]}},{"name":"processedAt","type":"long"}]}`,
		},
		"6": {
			ID:         6,
			Subject:    "clickstream-value",
			Version:    1,
			RecordName: "PageViewEvent",
			Schema:     `{"type":"record","name":"PageViewEvent","namespace":"com.example.analytics","fields":[{"name":"sessionId","type":"string"},{"name":"userId","type":["null","string"],"default":null},{"name":"url","type":"string"},{"name":"referrer","type":["null","string"],"default":null},{"name":"timestamp","type":"long"},{"name":"userAgent","type":"string"}]}`,
		},
		"7": {
			ID:         7,
			Subject:    "notification-events-value",
			Version:    1,
			RecordName: "NotificationSentEvent",
			Schema:     `{"type":"record","name":"NotificationSentEvent","namespace":"com.example.notifications","fields":[{"name":"notificationId","type":"string"},{"name":"userId","type":"string"},{"name":"channel","type":{"type":"enum","name":"Channel","symbols":["EMAIL","SMS","PUSH","WEBHOOK"]}},{"name":"templateId","type":"string"},{"name":"sentAt","type":"long"}]}`,
		},
		"8": {
			ID:         8,
			Subject:    "audit-log-value",
			Version:    1,
			RecordName: "AuditLogEntry",
			Schema:     `{"type":"record","name":"AuditLogEntry","namespace":"com.example.audit","fields":[{"name":"eventId","type":"string"},{"name":"actor","type":"string"},{"name":"action","type":"string"},{"name":"resource","type":"string"},{"name":"timestamp","type":"long"},{"name":"details","type":"string"}]}`,
		},
		"9": {
			ID:         9,
			Subject:    "dbserver1.inventory.products.Key",
			Version:    1,
			RecordName: "ProductsKey",
			Schema:     `{"type":"record","name":"ProductsKey","namespace":"dbserver1.inventory","fields":[{"name":"id","type":"string"}]}`,
		},
		"10": {
			ID:         10,
			Subject:    "dbserver1.inventory.products.Value",
			Version:    1,
			RecordName: "ProductsValue",
			Schema:     `{"type":"record","name":"ProductsValue","namespace":"dbserver1.inventory","fields":[{"name":"id","type":"string"},{"name":"name","type":"string"},{"name":"quantity","type":"int"},{"name":"price","type":"double"},{"name":"created_at","type":"long"}]}`,
		},
	}

	for id, schema := range schemas {
		kp.schemaCache[id] = schema
	}
}

// SetContext implements api.KafkaDataSource.
func (kp *KafkaDataSourceMock) SetContext(contextName string) error {
	if _, exists := mockContexts[contextName]; exists {
		currentContext = contextName
	}
	return nil
}

func (kp *KafkaDataSourceMock) GetContext() string {
	return currentContext
}

// GetContexts retrieves a list of Kafka contexts
func (kp *KafkaDataSourceMock) GetContexts() ([]string, error) {
	contexts := make([]string, 0, len(mockContexts))
	for ctx := range mockContexts {
		contexts = append(contexts, ctx)
	}
	return contexts, nil
}

// GetClusterDetails returns mock configuration details for the named cluster.
func (kp *KafkaDataSourceMock) GetClusterDetails(clusterName string) (api.ClusterInfo, error) {
	descriptions := map[string]string{
		"kafka-dev":  "localhost:9092",
		"kafka-test": "test-kafka:9092",
		"kafka-prod": "prod-kafka-1:9092,prod-kafka-2:9092",
	}
	brokerStr, exists := descriptions[clusterName]
	if !exists {
		return api.ClusterInfo{}, fmt.Errorf("cluster '%s' not found", clusterName)
	}
	brokers := strings.Split(brokerStr, ",")
	schemaURL := ""
	if clusterName == "kafka-prod" || clusterName == "kafka-dev" {
		schemaURL = "http://" + brokers[0][:strings.LastIndex(brokers[0], ":")] + ":8081"
	}
	return api.ClusterInfo{
		Name:              clusterName,
		Brokers:           brokers,
		SchemaRegistryURL: schemaURL,
		IsCurrent:         clusterName == currentContext,
		ReadOnly:          isReadOnlyMock(clusterName),
	}, nil
}

// GetTopics retrieves a list of Kafka topics for the current context
func (kp *KafkaDataSourceMock) GetTopics() (map[string]api.Topic, error) {
	ctxData, exists := mockContexts[currentContext]
	if !exists {
		return make(map[string]api.Topic), nil
	}

	// Return a copy to prevent external modification
	topics := make(map[string]api.Topic, len(ctxData.topics))
	for name, topic := range ctxData.topics {
		topics[name] = topic
	}
	return topics, nil
}

// GetTopicNames returns only topic names (mock version, same data as GetTopics but names only).
func (kp *KafkaDataSourceMock) GetTopicNames() ([]string, error) {
	ctxData, exists := mockContexts[currentContext]
	if !exists {
		return []string{}, nil
	}
	names := make([]string, 0, len(ctxData.topics))
	for name := range ctxData.topics {
		names = append(names, name)
	}
	return names, nil
}

// GetConsumerGroups retrieves consumer groups for the current context
func (kp *KafkaDataSourceMock) GetConsumerGroups() ([]api.ConsumerGroup, error) {
	ctxData, exists := mockContexts[currentContext]
	if !exists {
		return []api.ConsumerGroup{}, nil
	}

	// Return a copy to prevent external modification
	groups := make([]api.ConsumerGroup, len(ctxData.consumerGroups))
	copy(groups, ctxData.consumerGroups)
	return groups, nil
}

// mockStart anchors the simulated message-count growth so the metrics collector
// sees ever-increasing counts and derives a live, stable message-in rate under
// `make run-mock` without needing a real broker.
var mockStart = time.Now()

// mockTopicIngestRate returns a deterministic, per-topic messages-per-second
// used to simulate steady ingestion (varies by topic name for visual variety).
func mockTopicIngestRate(name string) int64 {
	var sum int64
	for _, r := range name {
		sum += int64(r)
	}
	return 5 + sum%20 // 5..24 msg/s
}

// GetTopicMessageCounts returns simulated message counts for the given topics.
// Counts grow with elapsed time so successive calls yield increasing values,
// letting the background collector derive message-in rates from the deltas.
func (kp *KafkaDataSourceMock) GetTopicMessageCounts(topics map[string]int32) (map[string]int64, error) {
	ctxData, exists := mockContexts[currentContext]
	elapsed := int64(time.Since(mockStart).Seconds())
	counts := make(map[string]int64, len(topics))
	for name := range topics {
		base := int64(1000) // sensible mock default
		if exists {
			if topic, ok := ctxData.topics[name]; ok && topic.MessageCount >= 0 {
				base = topic.MessageCount
			}
		}
		counts[name] = base + mockTopicIngestRate(name)*elapsed
	}
	return counts, nil
}

func (kp *KafkaDataSourceMock) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) error {
	// Simulate initial connection delay
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(50 * time.Millisecond):
	}

	// Validate the typed seek model (MSG-1).
	if err := flags.Validate(); err != nil {
		onError(err)
		return err
	}

	// First, deliver any produced messages for this topic (MSG-30) honoring the
	// seek/partition/limit filters (MSG-2/3/4). These are browsable in-memory.
	for _, msg := range kp.browseProduced(topicName, flags) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			handleMessage(msg)
		}
	}

	// Get or initialize message counter for this topic with realistic starting offset
	kp.counterMutex.Lock()
	if _, exists := kp.messageCounters[topicName]; !exists {
		// Initialize with realistic Kafka offsets (millions to billions for active topics)
		kp.messageCounters[topicName] = kp.getRealisticStartingOffset(topicName)
	}
	kp.counterMutex.Unlock()

	// Simulate continuous message consumption like real Kafka
	for {
		// Check if context is cancelled before processing
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue processing
		}

		// Generate a realistic message based on topic name
		msg := kp.generateMessage(topicName)

		// Check context again before calling handler with panic recovery
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Call the message handler function with panic recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Handler panicked (likely due to closed channel)
						// Call onError if context is still active
						select {
						case <-ctx.Done():
							return
						default:
							onError(fmt.Errorf("panic in message handler: %v", r))
						}
					}
				}()
				handleMessage(msg)
			}()
		}

		// Simulate realistic processing time between messages (100ms - 2s)
		delay := time.Duration(100+kp.randSource.Intn(1900)) * time.Millisecond
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next iteration
		}
	}
}

// ProduceMessage appends a record to the in-memory store so it is browsable
// via ConsumeTopic (MSG-30).
func (kp *KafkaDataSourceMock) ProduceMessage(ctx context.Context, topic string, rec api.ProduceRecord) error {
	// Validate that the topic exists in the current context.
	ctxData, ok := mockContexts[currentContext]
	if !ok {
		return api.NewConnectionError("no active context")
	}
	t, exists := ctxData.topics[topic]
	if !exists {
		return api.TopicNotFoundError{TopicName: topic}
	}

	kp.producedMu.Lock()
	defer kp.producedMu.Unlock()

	if kp.producedOffsets[topic] == nil {
		kp.producedOffsets[topic] = make(map[int32]int64)
	}

	// Resolve the target partition.
	var partition int32
	if rec.Partition != nil {
		if *rec.Partition < 0 || *rec.Partition >= t.NumPartitions {
			return api.NewPartitionError(
				fmt.Sprintf("partition %d out of range (topic has %d partitions)", *rec.Partition, t.NumPartitions),
				topic, *rec.Partition)
		}
		partition = *rec.Partition
	} else {
		// Auto: round-robin over the partition count (fallback to 0).
		if t.NumPartitions > 0 {
			partition = int32(kp.producedOffsets[topic][-1] % int64(t.NumPartitions))
			kp.producedOffsets[topic][-1]++
		}
	}

	offset := kp.producedOffsets[topic][partition]
	kp.producedOffsets[topic][partition] = offset + 1

	msg := api.Message{
		Offset:        offset,
		Partition:     partition,
		Headers:       append([]api.MessageHeader(nil), rec.Headers...),
		Timestamp:     time.Now(),
		TimestampType: api.TimestampTypeCreate,
		KeySerde:      "string",
		ValueSerde:    "string",
	}
	if rec.Key != nil {
		msg.Key = string(rec.Key)
		n := len(rec.Key)
		msg.KeySize = &n
	} else {
		msg.KeyNull = true
	}
	if rec.Value != nil {
		msg.Value = string(rec.Value)
		n := len(rec.Value)
		msg.ValueSize = &n
	} else {
		msg.ValueNull = true
	}
	for _, h := range rec.Headers {
		msg.HeadersSize += len(h.Key) + len(h.Value)
	}

	kp.produced[topic] = append(kp.produced[topic], msg)
	return nil
}

// browseProduced returns the produced messages for a topic filtered by the
// seek/partition/limit flags (MSG-2/3/4).
func (kp *KafkaDataSourceMock) browseProduced(topic string, flags api.ConsumeFlags) []api.Message {
	kp.producedMu.Lock()
	msgs := append([]api.Message(nil), kp.produced[topic]...)
	kp.producedMu.Unlock()
	return browseMessages(msgs, flags)
}

// browseMessages is a pure helper applying partition, seek and limit filters to
// a set of messages, ordered per the seek direction (MSG-2/3/4/6).
func browseMessages(msgs []api.Message, flags api.ConsumeFlags) []api.Message {
	if len(msgs) == 0 {
		return nil
	}

	// Partition filter (empty = all).
	if len(flags.Partitions) > 0 {
		allowed := make(map[int32]bool, len(flags.Partitions))
		for _, p := range flags.Partitions {
			allowed[p] = true
		}
		filtered := msgs[:0:0]
		for _, m := range msgs {
			if allowed[m.Partition] {
				filtered = append(filtered, m)
			}
		}
		msgs = filtered
	}
	if len(msgs) == 0 {
		return nil
	}

	// Offset bounds for clamping.
	minOff, maxOff := msgs[0].Offset, msgs[0].Offset
	for _, m := range msgs {
		if m.Offset < minOff {
			minOff = m.Offset
		}
		if m.Offset > maxOff {
			maxOff = m.Offset
		}
	}
	clamp := func(o int64) int64 {
		if o < minOff {
			return minOff
		}
		if o > maxOff {
			return maxOff
		}
		return o
	}

	keep := msgs[:0:0]
	for _, m := range msgs {
		switch flags.Seek {
		case api.SeekFromOffset:
			if m.Offset >= clamp(*flags.SeekOffset) {
				keep = append(keep, m)
			}
		case api.SeekToOffset:
			if m.Offset <= clamp(*flags.SeekOffset) {
				keep = append(keep, m)
			}
		case api.SeekFromTimestamp:
			if !m.Timestamp.Before(*flags.SeekTimestamp) {
				keep = append(keep, m)
			}
		case api.SeekToTimestamp:
			if m.Timestamp.Before(*flags.SeekTimestamp) {
				keep = append(keep, m)
			}
		default:
			keep = append(keep, m)
		}
	}
	msgs = keep

	api.SortMessages(msgs, flags.Seek.Backward())

	// Limit.
	if flags.LimitMessages > 0 && int64(len(msgs)) > flags.LimitMessages {
		msgs = msgs[:flags.LimitMessages]
	}
	return msgs
}

// generateMessage creates a realistic message based on topic name
func (kp *KafkaDataSourceMock) generateMessage(topicName string) api.Message {
	// Increment counter for this topic
	kp.counterMutex.Lock()
	kp.messageCounters[topicName]++
	counter := kp.messageCounters[topicName]
	kp.counterMutex.Unlock()

	// Generate message based on topic pattern
	var msg api.Message

	switch {
	case strings.Contains(topicName, "dbserver"):
		// CDC topics - these typically have key schemas (check first!)
		msg = kp.generateCDCEvent(topicName, counter)
	case strings.Contains(topicName, "user"):
		msg = kp.generateUserEvent(topicName, counter)
	case strings.Contains(topicName, "order"):
		msg = kp.generateOrderEvent(topicName, counter)
	case strings.Contains(topicName, "payment"):
		msg = kp.generatePaymentEvent(topicName, counter)
	case strings.Contains(topicName, "click"):
		msg = kp.generateClickstreamEvent(topicName, counter)
	case strings.Contains(topicName, "notification"):
		msg = kp.generateNotificationEvent(topicName, counter)
	case strings.Contains(topicName, "audit"):
		msg = kp.generateAuditEvent(topicName, counter)
	case strings.Contains(topicName, "inventory"):
		msg = kp.generateInventoryEvent(topicName, counter)
	default:
		msg = kp.generateGenericEvent(topicName, counter)
	}

	// Simulate realistic Kafka offsets with partition-based variation
	// Real Kafka offsets are per-partition and can be very large
	partition := int32(kp.randSource.Intn(5))
	msg.Partition = partition
	// Add partition-based offset variation (each partition has independent offsets)
	// Offsets in real Kafka can be in the millions/billions for active topics
	msg.Offset = counter + int64(partition)*1000000

	return msg
}

// getRealisticStartingOffset returns a realistic starting offset based on topic name
func (kp *KafkaDataSourceMock) getRealisticStartingOffset(topicName string) int64 {
	// Different topics have different message volumes
	// Production-like topics have higher offsets
	baseOffsets := map[string]int64{
		"clickstream":  15678234, // High-volume analytics
		"order":        8921560,  // Active order processing
		"payment":      4567890,  // Payment transactions
		"user":         1254300,  // User events
		"inventory":    2345670,  // Inventory updates
		"notification": 6789010,  // Notifications
		"audit":        23456780, // Audit logs (long retention)
		"dbserver":     456780,   // CDC data
	}

	// Find matching topic type
	for topicType, baseOffset := range baseOffsets {
		if strings.Contains(topicName, topicType) {
			// Add some randomness (±10%)
			variation := int64(kp.randSource.Intn(20) - 10)
			return baseOffset + (baseOffset * variation / 100)
		}
	}

	// Default offset for unknown topics
	return 100000 + int64(kp.randSource.Intn(50000))
}

// generateUserEvent creates user-related events
func (kp *KafkaDataSourceMock) generateUserEvent(topicName string, counter int64) api.Message {
	eventTypes := []string{"UserRegisteredEvent", "UserLoggedInEvent", "UserUpdatedEvent", "UserDeletedEvent"}
	eventType := eventTypes[counter%int64(len(eventTypes))]

	timestamp := time.Now().Add(-time.Duration(kp.randSource.Intn(3600)) * time.Second)

	value := fmt.Sprintf(`{
		"eventType": "%s",
		"userId": "user-%d",
		"email": "user%d@example.com",
		"registeredAt": %d,
		"preferences": {
			"theme": "%s",
			"language": "%s",
			"notifications": "%s"
		}
	}`, eventType, counter, counter, timestamp.UnixMilli(),
		[]string{"dark", "light", "auto"}[kp.randSource.Intn(3)],
		[]string{"en", "de", "fr", "es"}[kp.randSource.Intn(4)],
		[]string{"enabled", "disabled"}[kp.randSource.Intn(2)])

	return api.Message{
		Key:           fmt.Sprintf("user-%d", counter),
		Value:         value,
		KeySchemaID:   "",
		ValueSchemaID: "1",
		Headers: []api.MessageHeader{
			{Key: "correlationId", Value: fmt.Sprintf("corr-%d", counter)},
			{Key: "source", Value: "user-service"},
			{Key: "timestamp", Value: timestamp.Format(time.RFC3339)},
			{Key: "event-type", Value: eventType},
		},
	}
}

// generateOrderEvent creates order-related events
func (kp *KafkaDataSourceMock) generateOrderEvent(topicName string, counter int64) api.Message {
	statuses := []string{"CREATED", "CONFIRMED", "SHIPPED", "DELIVERED", "CANCELLED"}
	status := statuses[counter%int64(len(statuses))]

	amount := float64(counter*100+int64(kp.randSource.Intn(1000))) / 100.0
	timestamp := time.Now().Add(-time.Duration(kp.randSource.Intn(7200)) * time.Second)

	items := []string{}
	numItems := 1 + kp.randSource.Intn(5)
	for i := 0; i < numItems; i++ {
		items = append(items, fmt.Sprintf(`{"productId": "prod-%d", "quantity": %d, "price": %.2f}`,
			kp.randSource.Intn(1000), 1+kp.randSource.Intn(10), float64(10+kp.randSource.Intn(500))))
	}

	value := fmt.Sprintf(`{
		"orderId": "order-%d",
		"customerId": "customer-%d",
		"status": "%s",
		"amount": %.2f,
		"items": [%s],
		"createdAt": %d
	}`, counter, counter%100, status, amount,
		strings.Join(items, ","), timestamp.UnixMilli())

	return api.Message{
		Key:           fmt.Sprintf("order-%d", counter),
		Value:         value,
		KeySchemaID:   "",
		ValueSchemaID: "2",
		Headers: []api.MessageHeader{
			{Key: "correlationId", Value: fmt.Sprintf("order-corr-%d", counter)},
			{Key: "source", Value: "order-service"},
			{Key: "timestamp", Value: timestamp.Format(time.RFC3339)},
			{Key: "trace-id", Value: fmt.Sprintf("trace-%d", kp.randSource.Intn(10000))},
		},
	}
}

// generatePaymentEvent creates payment-related events
func (kp *KafkaDataSourceMock) generatePaymentEvent(topicName string, counter int64) api.Message {
	statuses := []string{"SUCCESS", "FAILED", "PENDING"}
	status := statuses[counter%int64(len(statuses))]

	amount := float64(counter*50+int64(kp.randSource.Intn(500))) / 100.0
	timestamp := time.Now().Add(-time.Duration(kp.randSource.Intn(3600)) * time.Second)

	value := fmt.Sprintf(`{
		"paymentId": "payment-%d",
		"orderId": "order-%d",
		"amount": %.2f,
		"status": "%s",
		"paymentMethod": "%s",
		"processedAt": %d
	}`, counter, counter%100, amount, status,
		[]string{"CREDIT_CARD", "DEBIT_CARD", "PAYPAL", "BANK_TRANSFER"}[counter%int64(4)],
		timestamp.UnixMilli())

	return api.Message{
		Key:           fmt.Sprintf("payment-%d", counter),
		Value:         value,
		KeySchemaID:   "",
		ValueSchemaID: "5",
		Headers: []api.MessageHeader{
			{Key: "correlationId", Value: fmt.Sprintf("payment-corr-%d", counter)},
			{Key: "source", Value: "payment-service"},
			{Key: "timestamp", Value: timestamp.Format(time.RFC3339)},
			{Key: "idempotency-key", Value: fmt.Sprintf("idem-%d", counter)},
		},
	}
}

// generateClickstreamEvent creates analytics events
func (kp *KafkaDataSourceMock) generateClickstreamEvent(topicName string, counter int64) api.Message {
	pages := []string{"/home", "/products", "/cart", "/checkout", "/account", "/search"}
	page := pages[counter%int64(len(pages))]

	browsers := []string{"Chrome", "Firefox", "Safari", "Edge"}
	browser := browsers[counter%int64(len(browsers))]

	timestamp := time.Now().Add(-time.Duration(kp.randSource.Intn(300)) * time.Second)

	var userId interface{}
	if counter%3 == 0 {
		userId = fmt.Sprintf("user-%d", counter%100)
	} else {
		userId = nil
	}

	value := fmt.Sprintf(`{
		"sessionId": "session-%d",
		"userId": %v,
		"url": "https://example.com%s",
		"referrer": "https://google.com",
		"timestamp": %d,
		"userAgent": "Mozilla/5.0 (%s)",
		"ipAddress": "192.168.%d.%d"
	}`, counter, userId, page, timestamp.UnixMilli(),
		browser, kp.randSource.Intn(256), kp.randSource.Intn(256))

	return api.Message{
		Key:           fmt.Sprintf("session-%d", counter),
		Value:         value,
		KeySchemaID:   "",
		ValueSchemaID: "6",
		Headers: []api.MessageHeader{
			{Key: "correlationId", Value: fmt.Sprintf("click-corr-%d", counter)},
			{Key: "source", Value: "analytics-service"},
			{Key: "timestamp", Value: timestamp.Format(time.RFC3339)},
		},
	}
}

// generateNotificationEvent creates notification events
func (kp *KafkaDataSourceMock) generateNotificationEvent(topicName string, counter int64) api.Message {
	channels := []string{"EMAIL", "SMS", "PUSH", "WEBHOOK"}
	channel := channels[counter%int64(len(channels))]

	timestamp := time.Now().Add(-time.Duration(kp.randSource.Intn(1800)) * time.Second)

	value := fmt.Sprintf(`{
		"notificationId": "notif-%d",
		"userId": "user-%d",
		"channel": "%s",
		"templateId": "template-%d",
		"subject": "Notification %d",
		"sentAt": %d,
		"status": "SENT"
	}`, counter, counter%100, channel, counter%10, counter, timestamp.UnixMilli())

	return api.Message{
		Key:           fmt.Sprintf("notif-%d", counter),
		Value:         value,
		KeySchemaID:   "",
		ValueSchemaID: "7",
		Headers: []api.MessageHeader{
			{Key: "correlationId", Value: fmt.Sprintf("notif-corr-%d", counter)},
			{Key: "source", Value: "notification-service"},
			{Key: "timestamp", Value: timestamp.Format(time.RFC3339)},
			{Key: "priority", Value: []string{"LOW", "MEDIUM", "HIGH"}[counter%3]},
		},
	}
}

// generateAuditEvent creates audit log events
func (kp *KafkaDataSourceMock) generateAuditEvent(topicName string, counter int64) api.Message {
	actions := []string{"CREATE", "READ", "UPDATE", "DELETE"}
	action := actions[counter%int64(len(actions))]

	resources := []string{"USER", "ORDER", "PRODUCT", "INVOICE"}
	resource := resources[counter%int64(len(resources))]

	timestamp := time.Now().Add(-time.Duration(kp.randSource.Intn(86400)) * time.Second)

	value := fmt.Sprintf(`{
		"eventId": "audit-%d",
		"actor": "user-%d",
		"action": "%s",
		"resource": "%s",
		"resourceId": "res-%d",
		"timestamp": %d,
		"details": "Action performed successfully",
		"ipAddress": "10.0.%d.%d"
	}`, counter, counter%50, action, resource, counter%1000, timestamp.UnixMilli(),
		kp.randSource.Intn(256), kp.randSource.Intn(256))

	return api.Message{
		Key:           fmt.Sprintf("audit-%d", counter),
		Value:         value,
		KeySchemaID:   "",
		ValueSchemaID: "8",
		Headers: []api.MessageHeader{
			{Key: "correlationId", Value: fmt.Sprintf("audit-corr-%d", counter)},
			{Key: "source", Value: "audit-service"},
			{Key: "timestamp", Value: timestamp.Format(time.RFC3339)},
			{Key: "severity", Value: "INFO"},
		},
	}
}

// generateInventoryEvent creates inventory events
func (kp *KafkaDataSourceMock) generateInventoryEvent(topicName string, counter int64) api.Message {
	warehouses := []string{"WH-EAST", "WH-WEST", "WH-CENTRAL", "WH-EUROPE"}
	warehouse := warehouses[counter%int64(len(warehouses))]

	timestamp := time.Now().Add(-time.Duration(kp.randSource.Intn(7200)) * time.Second)

	value := fmt.Sprintf(`{
		"warehouseId": "%s",
		"productId": "prod-%d",
		"quantity": %d,
		"reserved": %d,
		"available": %d,
		"lastUpdated": %d
	}`, warehouse, counter%500, 100+kp.randSource.Intn(900),
		kp.randSource.Intn(100), 100+kp.randSource.Intn(800), timestamp.UnixMilli())

	return api.Message{
		Key:           fmt.Sprintf("%s:prod-%d", warehouse, counter%500),
		Value:         value,
		KeySchemaID:   "4",
		ValueSchemaID: "",
		Headers: []api.MessageHeader{
			{Key: "correlationId", Value: fmt.Sprintf("inv-corr-%d", counter)},
			{Key: "source", Value: "inventory-service"},
			{Key: "timestamp", Value: timestamp.Format(time.RFC3339)},
		},
	}
}

// generateCDCEvent creates CDC (Change Data Capture) events with key schemas
func (kp *KafkaDataSourceMock) generateCDCEvent(topicName string, counter int64) api.Message {
	// Parse topic name to extract table info (e.g., dbserver1.inventory.products)
	parts := strings.Split(topicName, ".")
	operation := []string{"c", "u", "d"}[counter%3] // create, update, delete

	timestamp := time.Now().Add(-time.Duration(kp.randSource.Intn(7200)) * time.Second)

	var key, value string

	if len(parts) >= 3 {
		// CDC key format (Debezium style)
		key = fmt.Sprintf(`{"schema": {"type": "struct", "fields": [{"type": "string", "field": "id"}], "optional": false}, "payload": {"id": %d}}`, counter)

		// CDC value format (Debezium style)
		value = fmt.Sprintf(`{
			"schema": {
				"type": "struct",
				"fields": [
					{"type": "string", "field": "id"},
					{"type": "string", "field": "name"},
					{"type": "int32", "field": "quantity"},
					{"type": "double", "field": "price"},
					{"type": "long", "field": "created_at"}
				]
			},
			"payload": {
				"id": "%d",
				"name": "Product %d",
				"quantity": %d,
				"price": %.2f,
				"created_at": %d
			}
		}`, counter, counter%1000, 100+kp.randSource.Intn(900),
			float64(10+kp.randSource.Intn(500)), timestamp.UnixMilli())
	} else {
		// Fallback for non-standard CDC topics
		key = fmt.Sprintf(`{"id": %d}`, counter)
		value = fmt.Sprintf(`{"data": "record-%d", "timestamp": %d}`, counter, timestamp.UnixMilli())
	}

	return api.Message{
		Key:           key,
		Value:         value,
		KeySchemaID:   "9",  // CDC key schema
		ValueSchemaID: "10", // CDC value schema
		Headers: []api.MessageHeader{
			{Key: "correlationId", Value: fmt.Sprintf("cdc-corr-%d", counter)},
			{Key: "source", Value: "debezium-connector"},
			{Key: "timestamp", Value: timestamp.Format(time.RFC3339)},
			{Key: "op", Value: operation},
			{Key: "ts_ms", Value: fmt.Sprintf("%d", timestamp.UnixMilli())},
		},
	}
}

// generateGenericEvent creates generic events for unknown topics
func (kp *KafkaDataSourceMock) generateGenericEvent(topicName string, counter int64) api.Message {
	timestamp := time.Now().Add(-time.Duration(kp.randSource.Intn(3600)) * time.Second)

	value := fmt.Sprintf(`{
		"id": "event-%d",
		"topic": "%s",
		"timestamp": %d,
		"data": {
			"index": %d,
			"value": "data-point-%d"
		}
	}`, counter, topicName, timestamp.UnixMilli(), counter, counter)

	return api.Message{
		Key:           fmt.Sprintf("key-%d", counter),
		Value:         value,
		KeySchemaID:   "",
		ValueSchemaID: "",
		Headers: []api.MessageHeader{
			{Key: "correlationId", Value: fmt.Sprintf("corr-%d", counter)},
			{Key: "source", Value: "generic-service"},
			{Key: "timestamp", Value: timestamp.Format(time.RFC3339)},
		},
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

// DecodeMessage implements api.KafkaDataSource.
// Mock messages already have decoded Key/Value; RawKey/RawValue are treated as plain text.
func (kp *KafkaDataSourceMock) DecodeMessage(_ context.Context, msg api.Message) (api.Message, error) {
	reg := mockSerdeRegistry()
	if len(msg.RawKey) > 0 && msg.Key == "" {
		text, name, _ := serde.Decode(reg, "", msg.RawKey)
		msg.Key, msg.KeySerde = text, name
	}
	if len(msg.RawValue) > 0 && msg.Value == "" {
		text, name, _ := serde.Decode(reg, "", msg.RawValue)
		msg.Value, msg.ValueSerde = text, name
	}
	return msg, nil
}

// mockSerde builds a built-in-only registry once (no schema registry).
var mockSerdeReg *serde.Registry

func mockSerdeRegistry() *serde.Registry {
	if mockSerdeReg == nil {
		mockSerdeReg, _ = serde.BuildRegistry(nil, nil)
	}
	return mockSerdeReg
}

// ListSerdes returns a plausible static list of serde names. (MSG-18)
func (kp *KafkaDataSourceMock) ListSerdes() []string {
	return mockSerdeRegistry().Names()
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
	// Return from preloaded cache if exists
	if schema, exists := kp.schemaCache[schemaID]; exists {
		return schema
	}

	// Return nil for non-Avro or unknown schemas
	return nil
}

// generateDevTopics generates realistic development environment topics
func generateDevTopics() map[string]api.Topic {
	topics := make(map[string]api.Topic)

	// Event-driven architecture topics
	topics["user-events"] = api.Topic{
		NumPartitions:     6,
		ReplicationFactor: 3,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":   strPtr("604800000"), // 7 days
			"cleanup.policy": strPtr("delete"),
		},
		MessageCount: 125430,
	}

	topics["order-events"] = api.Topic{
		NumPartitions:     12,
		ReplicationFactor: 3,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":   strPtr("2592000000"), // 30 days
			"cleanup.policy": strPtr("delete"),
		},
		MessageCount: 892156,
	}

	topics["payment-events"] = api.Topic{
		NumPartitions:     8,
		ReplicationFactor: 3,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":   strPtr("7776000000"), // 90 days
			"cleanup.policy": strPtr("delete"),
		},
		MessageCount: 456789,
	}

	topics["inventory-events"] = api.Topic{
		NumPartitions:     6,
		ReplicationFactor: 3,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":   strPtr("604800000"),
			"cleanup.policy": strPtr("compact"),
		},
		MessageCount: 234567,
	}

	topics["notification-events"] = api.Topic{
		NumPartitions:     4,
		ReplicationFactor: 3,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":   strPtr("259200000"), // 3 days
			"cleanup.policy": strPtr("delete"),
		},
		MessageCount: 678901,
	}

	topics["clickstream-events"] = api.Topic{
		NumPartitions:     16,
		ReplicationFactor: 2,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":   strPtr("86400000"), // 1 day
			"cleanup.policy": strPtr("delete"),
		},
		MessageCount: 15678234,
	}

	topics["audit-log"] = api.Topic{
		NumPartitions:     8,
		ReplicationFactor: 3,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":   strPtr("31536000000"), // 1 year
			"cleanup.policy": strPtr("delete"),
		},
		MessageCount: 2345678,
	}

	// CDC topics
	topics["dbserver1.inventory.products"] = api.Topic{
		NumPartitions:     4,
		ReplicationFactor: 3,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":   strPtr("604800000"),
			"cleanup.policy": strPtr("compact"),
		},
		MessageCount: 45678,
	}

	topics["dbserver1.customers.users"] = api.Topic{
		NumPartitions:     4,
		ReplicationFactor: 3,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":   strPtr("604800000"),
			"cleanup.policy": strPtr("compact"),
		},
		MessageCount: 23456,
	}

	// Dead letter queue
	topics["dead-letter-queue"] = api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 3,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":   strPtr("2592000000"), // 30 days
			"cleanup.policy": strPtr("delete"),
		},
		MessageCount: 1234,
	}

	return topics
}

// generateTestTopics generates realistic test environment topics
func generateTestTopics() map[string]api.Topic {
	topics := make(map[string]api.Topic)

	// Mirror of prod topics but smaller
	topics["user-events"] = api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 2,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":   strPtr("86400000"), // 1 day
			"cleanup.policy": strPtr("delete"),
		},
		MessageCount: 5000,
	}

	topics["order-events"] = api.Topic{
		NumPartitions:     6,
		ReplicationFactor: 2,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":   strPtr("604800000"), // 7 days
			"cleanup.policy": strPtr("delete"),
		},
		MessageCount: 25000,
	}

	topics["payment-events"] = api.Topic{
		NumPartitions:     4,
		ReplicationFactor: 2,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":   strPtr("604800000"),
			"cleanup.policy": strPtr("delete"),
		},
		MessageCount: 12000,
	}

	topics["test-topic-a"] = api.Topic{
		NumPartitions:     1,
		ReplicationFactor: 1,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries:     map[string]*string{},
		MessageCount:      100,
	}

	topics["test-topic-b"] = api.Topic{
		NumPartitions:     1,
		ReplicationFactor: 1,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries:     map[string]*string{},
		MessageCount:      50,
	}

	return topics
}

// generateProdTopics generates realistic production environment topics
func generateProdTopics() map[string]api.Topic {
	topics := make(map[string]api.Topic)

	topics["user-events"] = api.Topic{
		NumPartitions:     24,
		ReplicationFactor: 3,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":        strPtr("604800000"),
			"cleanup.policy":      strPtr("delete"),
			"min.insync.replicas": strPtr("2"),
		},
		MessageCount: 45678901,
	}

	topics["order-events"] = api.Topic{
		NumPartitions:     48,
		ReplicationFactor: 3,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":        strPtr("2592000000"),
			"cleanup.policy":      strPtr("delete"),
			"min.insync.replicas": strPtr("2"),
		},
		MessageCount: 123456789,
	}

	topics["payment-events"] = api.Topic{
		NumPartitions:     32,
		ReplicationFactor: 3,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":        strPtr("7776000000"),
			"cleanup.policy":      strPtr("delete"),
			"min.insync.replicas": strPtr("2"),
		},
		MessageCount: 67890123,
	}

	topics["inventory-events"] = api.Topic{
		NumPartitions:     24,
		ReplicationFactor: 3,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":        strPtr("604800000"),
			"cleanup.policy":      strPtr("compact"),
			"min.insync.replicas": strPtr("2"),
		},
		MessageCount: 34567890,
	}

	topics["notification-events"] = api.Topic{
		NumPartitions:     16,
		ReplicationFactor: 3,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":        strPtr("259200000"),
			"cleanup.policy":      strPtr("delete"),
			"min.insync.replicas": strPtr("2"),
		},
		MessageCount: 89012345,
	}

	topics["clickstream-events"] = api.Topic{
		NumPartitions:     64,
		ReplicationFactor: 2,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":   strPtr("86400000"),
			"cleanup.policy": strPtr("delete"),
		},
		MessageCount: 987654321,
	}

	topics["audit-log"] = api.Topic{
		NumPartitions:     32,
		ReplicationFactor: 3,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":        strPtr("31536000000"),
			"cleanup.policy":      strPtr("delete"),
			"min.insync.replicas": strPtr("2"),
		},
		MessageCount: 234567890,
	}

	topics["fraud-detection"] = api.Topic{
		NumPartitions:     16,
		ReplicationFactor: 3,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries: map[string]*string{
			"retention.ms":        strPtr("86400000"),
			"cleanup.policy":      strPtr("delete"),
			"min.insync.replicas": strPtr("2"),
		},
		MessageCount: 12345678,
	}

	return topics
}

// generateDevConsumerGroups generates consumer groups for dev environment
func generateDevConsumerGroups() []api.ConsumerGroup {
	return []api.ConsumerGroup{
		{Name: "order-processor", State: "Active", Consumers: 3},
		{Name: "payment-service", State: "Active", Consumers: 2},
		{Name: "notification-sender", State: "Active", Consumers: 4},
		{Name: "analytics-consumer", State: "Idle", Consumers: 1},
		{Name: "audit-logger", State: "Active", Consumers: 2},
		{Name: "inventory-sync", State: "Empty", Consumers: 0},
		{Name: "search-indexer", State: "Active", Consumers: 3},
		{Name: "email-service", State: "Active", Consumers: 2},
	}
}

// generateTestConsumerGroups generates consumer groups for test environment
func generateTestConsumerGroups() []api.ConsumerGroup {
	return []api.ConsumerGroup{
		{Name: "test-consumer-group", State: "Active", Consumers: 1},
		{Name: "integration-test-consumer", State: "Idle", Consumers: 0},
		{Name: "load-test-processor", State: "Empty", Consumers: 0},
	}
}

// generateProdConsumerGroups generates consumer groups for prod environment
func generateProdConsumerGroups() []api.ConsumerGroup {
	return []api.ConsumerGroup{
		{Name: "order-processor-prod", State: "Active", Consumers: 12},
		{Name: "payment-service-prod", State: "Active", Consumers: 8},
		{Name: "notification-sender-prod", State: "Active", Consumers: 6},
		{Name: "analytics-consumer-prod", State: "Active", Consumers: 4},
		{Name: "audit-logger-prod", State: "Active", Consumers: 3},
		{Name: "inventory-sync-prod", State: "Active", Consumers: 4},
		{Name: "search-indexer-prod", State: "Active", Consumers: 6},
		{Name: "email-service-prod", State: "Active", Consumers: 4},
		{Name: "fraud-detection-prod", State: "Active", Consumers: 8},
		{Name: "reporting-service-prod", State: "Idle", Consumers: 2},
		{Name: "data-warehouse-sync", State: "Active", Consumers: 4},
		{Name: "ml-pipeline-consumer", State: "Active", Consumers: 6},
	}
}

// strPtr returns a pointer to a string
func strPtr(s string) *string {
	return &s
}
