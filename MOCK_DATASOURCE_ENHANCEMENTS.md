# Mock Data Source Enhancements

## Overview
Enhanced the mock Kafka data source (`pkg/datasource/mock/kafka_data_source_mock.go`) to provide more realistic testing data for better UI/UX testing in mock mode.

## Key Enhancements

### 1. Context-Aware Mock Data
The mock now supports **three distinct environments** with different data:
- **kafka-dev**: Development cluster with moderate topic count and message volumes
- **kafka-test**: Test cluster with fewer topics and lower message counts
- **kafka-prod**: Production cluster with many topics and high message volumes

Each context has:
- Unique topic configurations
- Different consumer groups
- Realistic partition counts and replication factors

### 2. Realistic Topics
Replaced generic "Topic N" names with **realistic topic patterns**:

**Event-Driven Topics:**
- `user-events` - User lifecycle events (registration, login, updates)
- `order-events` - Order processing events
- `payment-events` - Payment transactions
- `inventory-events` - Inventory updates
- `notification-events` - Notification delivery
- `clickstream-events` - User analytics
- `audit-log` - Audit trail

**CDC Topics (Debezium-style):**
- `dbserver1.inventory.products` - Product table CDC with key schemas
- `dbserver1.customers.users` - User table CDC with key schemas

**System Topics:**
- `dead-letter-queue` - Failed messages

Each topic has:
- Realistic partition counts (1-64 based on environment)
- Appropriate replication factors (1-3)
- Custom configuration entries (retention, cleanup policy)
- Message counts across partitions

### 3. Realistic Kafka Offsets
Offsets now simulate **real Kafka behavior**:
- **Starting offsets** are in the millions/billions (like active production topics)
- **Per-partition offset tracking** with partition-based variation
- **Monotonically increasing** offsets within each partition
- Topic-specific base offsets based on expected volume:
  - `clickstream`: 15+ million (high-volume analytics)
  - `order`: 8+ million (active order processing)
  - `audit`: 23+ million (long retention)
  - `user`: 1+ million (user events)

### 4. Intelligent Message Generation
Messages are now **generated based on topic name patterns**:

#### User Events
```json
{
  "eventType": "UserRegisteredEvent",
  "userId": "user-123",
  "email": "user123@example.com",
  "registeredAt": 1708776543210,
  "preferences": {
    "theme": "dark",
    "language": "en",
    "notifications": "enabled"
  }
}
```

#### Order Events
```json
{
  "orderId": "order-456",
  "customerId": "customer-78",
  "status": "CREATED",
  "amount": 1234.56,
  "items": [
    {"productId": "prod-123", "quantity": 2, "price": 99.99}
  ],
  "createdAt": 1708776543210
}
```

#### Payment Events
```json
{
  "paymentId": "payment-789",
  "orderId": "order-456",
  "amount": 1234.56,
  "status": "SUCCESS",
  "paymentMethod": "CREDIT_CARD",
  "processedAt": 1708776543210
}
```

#### Clickstream Events
```json
{
  "sessionId": "session-abc",
  "userId": "user-123",
  "url": "https://example.com/products",
  "referrer": "https://google.com",
  "timestamp": 1708776543210,
  "userAgent": "Mozilla/5.0 (Chrome)",
  "ipAddress": "192.168.1.100"
}
```

### 4. Enhanced Message Features

#### Headers
Every message includes realistic headers:
- `correlationId` - For distributed tracing
- `source` - Service name (e.g., "user-service", "order-service")
- `timestamp` - RFC3339 formatted timestamp
- Additional context-specific headers (e.g., `event-type`, `trace-id`, `priority`)

#### Schema Integration
Messages include **schema IDs** for Avro schema lookup:
- User events → Schema ID 1
- Order events → Schema ID 2
- Payment events → Schema ID 5
- Clickstream → Schema ID 6
- Notifications → Schema ID 7
- Audit logs → Schema ID 8
- Inventory (key schema) → Schema ID 4

#### CDC Events (Debezium-style)
```json
{
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
    "id": "123",
    "name": "Product 123",
    "quantity": 500,
    "price": 99.99,
    "created_at": 1708776543210
  }
}
```

CDC messages include:
- **Key schema** (ID 9) - ProductsKey with id field
- **Value schema** (ID 10) - ProductsValue with full record
- Debezium-style headers: `op` (c/u/d), `ts_ms`, `source`
- Proper CDC topic format: `dbserver.database.table`

#### Partitioning
Messages are distributed across **5 partitions** with realistic variation:
- Random partition assignment (simulates Kafka's partitioning strategy)
- Partition-based offset calculation (each partition has independent offsets)
- Offset formula: `baseOffset + counter + (partition * 1,000,000)`

#### Offsets
Each topic maintains **independent offset counters** for realistic offset progression.

### 5. Preloaded Schemas
The mock now **preloads 10 Avro schemas** on initialization:

1. **UserRegisteredEvent** - User registration schema
2. **OrderCreatedEvent** (v1) - Basic order schema
3. **OrderCreatedEvent** (v2) - Enhanced order with nested types
4. **InventoryKey** - Inventory composite key
5. **PaymentProcessedEvent** - Payment with enum status
6. **PageViewEvent** - Clickstream analytics
7. **NotificationSentEvent** - Notification with enum channel
8. **AuditLogEntry** - Audit logging
9. **ProductsKey** - CDC key schema for products table
10. **ProductsValue** - CDC value schema for products table

Schemas include:
- Proper Avro type definitions
- Namespaces
- Complex types (arrays, maps, enums, nested records)
- Schema evolution (v1/v2 for orders)

### 6. Realistic Consumer Groups
Each environment has **context-appropriate consumer groups**:

**Development (8 groups):**
- order-processor, payment-service, notification-sender, etc.

**Test (3 groups):**
- test-consumer-group, integration-test-consumer, etc.

**Production (12 groups):**
- order-processor-prod, payment-service-prod, fraud-detection-prod, etc.

Consumer groups have:
- Realistic consumer counts (0-12)
- Various states (Active, Idle, Empty)
- Environment-specific naming

### 7. Improved Timing
- **Initial connection delay**: 50ms (simulates connection setup)
- **Message interval**: 100ms - 2s random delay (simulates realistic message rates)
- **Context-aware cancellation**: Properly respects context cancellation

### 8. Thread Safety
Enhanced thread safety with:
- `schemaMutex` for schema cache access
- `counterMutex` for message counter updates
- Independent counters per topic

## Testing Improvements

### Updated Test Suite
The test file (`kafka_data_source_mock_test.go`) has been completely rewritten with:

1. **Context-aware tests** - Tests for all three environments
2. **Schema validation** - Tests for preloaded schemas
3. **Message type tests** - Validates message generation per topic pattern
4. **Consumer group tests** - Tests for each environment
5. **Schema info tests** - Tests for schema retrieval and content
6. **Interface compliance** - Ensures API interface implementation

### Test Coverage
All tests pass with comprehensive coverage:
- ✅ Initialization and schema preloading
- ✅ Topic retrieval across contexts
- ✅ Context switching
- ✅ Consumer group retrieval
- ✅ Message consumption with various scenarios
- ✅ Schema information retrieval
- ✅ Interface compliance

## Usage

### Running in Mock Mode
```bash
# Run with mock data source
go run main.go --mock
```

### Testing Different Contexts
```bash
# The mock automatically uses kafka-dev as default
# Switch contexts in the UI to see different data:
# - kafka-dev: Moderate data volume
# - kafka-test: Lower data volume  
# - kafka-prod: High data volume
```

### Testing Topic Consumption
1. Start in mock mode: `go run main.go --mock`
2. Navigate to any topic (e.g., `user-events`, `order-events`)
3. Press Enter to consume messages
4. Observe realistic JSON messages with proper structure
5. Test schema viewing (messages with schema IDs will show schema info)

## Benefits

1. **Better UI Testing** - Realistic data helps test UI rendering with various message formats
2. **Schema Testing** - Test Avro schema display without Schema Registry
3. **Performance Testing** - Different message volumes per environment
4. **Feature Testing** - Test search, filtering, and navigation with realistic data
5. **Demo Mode** - Great for demonstrations without Kafka infrastructure

## Files Modified

- `pkg/datasource/mock/kafka_data_source_mock.go` - Complete rewrite
- `pkg/datasource/mock/kafka_data_source_mock_test.go` - Complete rewrite

## Backward Compatibility

The mock maintains full backward compatibility with the `api.KafkaDataSource` interface. All existing code using the mock will continue to work without changes.
