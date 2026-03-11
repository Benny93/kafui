package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/pages/topic"
	tea "github.com/charmbracelet/bubbletea"
)

// MockDataSource provides mock Kafka data for testing
type MockDataSource struct {
	messages []api.Message  // Pre-generated message pool
}

func (m *MockDataSource) Init(cfgOption string) {
	// Pre-generate 1000 messages for testing
	m.messages = make([]api.Message, 0, 1000)
	partitions := []int32{0, 1, 2}
	
	for i := int64(0); i < 1000; i++ {
		partition := partitions[i%int64(len(partitions))]
		m.messages = append(m.messages, createMockMessage(i, partition))
	}
}

func (m *MockDataSource) GetTopics() (map[string]api.Topic, error)           { return nil, nil }
func (m *MockDataSource) GetConsumerGroups() ([]api.ConsumerGroup, error)    { return nil, nil }
func (m *MockDataSource) GetContexts() ([]string, error)                     { return nil, nil }
func (m *MockDataSource) GetContext() string                                 { return "mock" }
func (m *MockDataSource) SetContext(contextName string) error                { return nil }
func (m *MockDataSource) GetMessageSchemaInfo(keySchemaID, valueSchemaID string) (*api.MessageSchemaInfo, error) {
	return nil, nil
}

// ConsumeTopic simulates consuming messages from a topic
// For continuous consumption mode, it sends messages and keeps running until context is cancelled
func (m *MockDataSource) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) error {
	totalMessages := len(m.messages)

	// Safety check: prevent divide by zero
	if totalMessages == 0 {
		return fmt.Errorf("no messages available in mock data source")
	}

	// Send initial batch of messages (newest first)
	count := 300
	if totalMessages < count {
		count = totalMessages
	}

	// Send initial batch from the end (newest first)
	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
			idx := totalMessages - 1 - i
			if idx >= 0 && idx < len(m.messages) {
				handleMessage(m.messages[idx])
			}
		}
	}

	// For continuous consumption, keep sending messages periodically
	// This simulates a real Kafka consumer that keeps receiving new messages
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	msgIndex := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Send a few messages periodically to simulate continuous stream
			for j := 0; j < 3; j++ {
				msg := m.messages[msgIndex%totalMessages]
				handleMessage(msg)
				msgIndex++
			}
		}
	}
}

// createMockMessage creates a mock message with varying content lengths and types
func createMockMessage(offset int64, partition int32) api.Message {
	var key, value string
	
	switch offset % 10 {
	case 0:
		// Short JSON message
		key = fmt.Sprintf("key-%d", offset)
		value = fmt.Sprintf(`{"id": %d, "status": "ok", "data": "short"}`, offset)
		
	case 1:
		// User event with timestamps
		key = fmt.Sprintf("user-event-%d", offset)
		value = fmt.Sprintf(`{"user_id": %d, "event": "login", "timestamp": "%s", "ip": "192.168.1.%d", "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36", "session_id": "sess_%d_%s"}`,
			offset, time.Now().Format(time.RFC3339), offset%255, offset, generateRandomString(32))
		
	case 2:
		// Order with nested structure
		key = fmt.Sprintf("order-%d", offset)
		value = fmt.Sprintf(`{"order_id": %d, "customer": {"id": %d, "name": "Customer %d", "email": "customer%d@example.com"}, "items": [{"sku": "PROD-%d", "qty": 2, "price": 29.99, "name": "Product A"}, {"sku": "PROD-%d", "qty": 1, "price": 49.99, "name": "Product B with a very long name that should test wrapping"}], "total": 109.97, "shipping_address": "123 Main Street, Anytown, State 12345, United States of America"}`,
			offset, offset, offset, offset, offset, offset+1)
		
	case 3:
		// Long log message
		key = fmt.Sprintf("log-%d", offset)
		value = fmt.Sprintf(`[ERROR] %s - Database connection timeout after 30000ms. Retrying... (attempt %d/3) | Connection pool: active=5, idle=2, waiting=12 | Last successful query: SELECT * FROM users WHERE id IN (%s) | Stack trace: at com.example.db.ConnectionPool.getConnection(ConnectionPool.java:123) at com.example.service.UserService.getUser(UserService.java:45)`,
			time.Now().Format(time.RFC3339), (offset%3)+1, generateLongIDList(offset))
		
	case 4:
		// Metrics with many fields
		key = fmt.Sprintf("metric-%d", offset)
		value = fmt.Sprintf(`{"timestamp": "%s", "host": "server-%d", "cpu": {"user": %.2f, "system": %.2f, "idle": %.2f}, "memory": {"total": %d, "used": %d, "free": %d, "cached": %d}, "disk": {"read_iops": %d, "write_iops": %d, "read_bps": %d, "write_bps": %d}, "network": {"rx_bytes": %d, "tx_bytes": %d, "rx_packets": %d, "tx_packets": %d}, "load_avg": [%.2f, %.2f, %.2f]}`,
			time.Now().Format(time.RFC3339), offset%10,
			float64(offset%100)/10, float64(offset%80)/10, float64(offset%90)/10,
			16*1024*1024*1024, 8*1024*1024*1024, 4*1024*1024*1024, 2*1024*1024*1024,
			offset*100, offset*50, offset*1024*1024, offset*512*1024,
			offset*1024*1024*10, offset*1024*1024*5, offset*1024*100, offset*1024*50,
			float64(offset%50)/10, float64(offset%40)/10, float64(offset%30)/10)
		
	case 5:
		// Very long text message (tests truncation)
		key = fmt.Sprintf("text-%d", offset)
		value = generateLongText(offset)
		
	case 6:
		// Array of objects
		key = fmt.Sprintf("batch-%d", offset)
		value = fmt.Sprintf(`{"batch_id": %d, "records": [%s], "total_count": %d, "processing_time_ms": %d}`,
			offset, generateRecordsArray(offset, 10), 10, offset%1000)
		
	case 7:
		// Error message with details
		key = fmt.Sprintf("error-%d", offset)
		value = fmt.Sprintf(`{"error": {"code": "ERR_%d", "message": "An error occurred while processing request %d", "details": {"request_id": "req_%d_%s", "timestamp": "%s", "service": "api-gateway", "trace_id": "%s", "span_id": "%s"}}, "context": {"user_agent": "Mozilla/5.0", "path": "/api/v1/users/%d", "method": "GET", "query_params": {"include": "profile,settings,preferences", "fields": "id,name,email,created_at,updated_at"}}}`,
			offset, offset, offset, generateRandomString(16), time.Now().Format(time.RFC3339),
			generateRandomString(32), generateRandomString(16), offset)
		
	case 8:
		// Kafka consumer lag data
		key = fmt.Sprintf("lag-%d", offset)
		value = fmt.Sprintf(`{"topic": "test-topic", "partition": %d, "current_offset": %d, "end_offset": %d, "lag": %d, "consumer_id": "consumer-%d", "group_id": "consumer-group-%d", "last_commit_time": "%s", "rate": %.2f msgs/sec}`,
			partition, offset, offset+1000, 1000, offset%10, offset%5, time.Now().Format(time.RFC3339), float64(offset%1000)/10)
		
	case 9:
		// Schema registry event
		key = fmt.Sprintf("schema-%d", offset)
		value = fmt.Sprintf(`{"subject": "test-topic-value", "version": %d, "id": %d, "schema_type": "AVRO", "schema": "{\"type\":\"record\",\"name\":\"TestRecord\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"data\",\"type\":{\"type\":\"array\",\"items\":\"string\"}}]}", "references": [], "compatibility": "BACKWARD"}`,
			offset%100, offset)
	}
	
	return api.Message{
		Offset:    offset,
		Partition: partition,
		Key:       key,
		Value:     value,
	}
}

// generateLongText generates a very long text message for testing truncation
func generateLongText(offset int64) string {
	words := []string{
		"Lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing", "elit",
		"sed", "do", "eiusmod", "tempor", "incididunt", "ut", "labore", "et", "dolore",
		"magna", "aliqua", "Ut", "enim", "ad", "minim", "veniam", "quis", "nostrud",
		"exercitation", "ullamco", "laboris", "nisi", "ut", "aliquip", "ex", "ea",
		"commodo", "consequat", "Duis", "aute", "irure", "dolor", "in", "reprehenderit",
		"voluptate", "velit", "esse", "cillum", "dolore", "eu", "fugiat", "nulla", "pariatur",
	}
	
	var parts []string
	for i := 0; i < 50; i++ {
		parts = append(parts, words[(int(offset)+i)%len(words)])
	}
	return fmt.Sprintf("Message %d: %s...", offset, strings.Join(parts, " "))
}

// generateRecordsArray generates an array of record objects
func generateRecordsArray(offset int64, count int) string {
	var records []string
	for i := 0; i < count; i++ {
		record := fmt.Sprintf(`{"id": %d, "name": "Record %d-%d", "value": %d}`, offset, offset, i, offset*int64(i+1))
		records = append(records, record)
	}
	return strings.Join(records, ", ")
}

// generateLongIDList generates a long comma-separated list of IDs
func generateLongIDList(offset int64) string {
	ids := make([]string, 0, 50)
	for i := 0; i < 50; i++ {
		ids = append(ids, fmt.Sprintf("%d", (offset+int64(i))%10000))
	}
	return strings.Join(ids, ", ")
}

// generateRandomString generates a random hex string of given length
func generateRandomString(length int) string {
	bytes := make([]byte, length/2)
	for i := range bytes {
		bytes[i] = byte((i + 1) * 17 % 256)
	}
	return fmt.Sprintf("%x", bytes)
}

func main() {
	// Create CPU profile
	cpuf, err := os.Create("topic_page_cpu.prof")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating CPU profile: %v\n", err)
		os.Exit(1)
	}
	defer cpuf.Close()

	if err := pprof.StartCPUProfile(cpuf); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting CPU profile: %v\n", err)
		os.Exit(1)
	}
	defer pprof.StopCPUProfile()

	fmt.Println("CPU profiling enabled. Profile will be saved to: topic_page_cpu.prof")
	fmt.Println("Press Ctrl+C to stop and save profile")
	fmt.Println("")

	// Create mock data source
	dataSource := &MockDataSource{}
	dataSource.Init("") // Initialize mock messages

	// Create topic details
	topicDetails := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 2,
		MessageCount:      1000,
		ConfigEntries: map[string]*string{
			"retention.ms":   strPtr("604800000"),
			"cleanup.policy": strPtr("delete"),
		},
	}

	// Create topic page model
	topicPage := topic.NewTopicPageModel(dataSource, "test-topic-with-messages", topicDetails)

	// Handle Ctrl+C to save profiles
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		fmt.Println("\nSaving profiles...")
		pprof.StopCPUProfile()

		// Also save heap profile
		memf, err := os.Create("topic_page_mem.prof")
		if err == nil {
			defer memf.Close()
			pprof.WriteHeapProfile(memf)
			fmt.Println("Memory profile saved to: topic_page_mem.prof")
		}
		os.Exit(0)
	}()

	// Run the TUI with a minimum window size suggestion
	// Note: Sidebar requires width >= 120 and height >= 30 to be visible by default
	// You can toggle sidebar with 't' or 'Ctrl+S' when window is wide enough
	p := tea.NewProgram(topicPage, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		pprof.StopCPUProfile()
		os.Exit(1)
	}

	fmt.Println("\nProfile saved to: topic_page_cpu.prof")
	fmt.Println("Analyze with: go tool pprof topic_page_cpu.prof")
	fmt.Println("Or view in browser: go tool pprof -http=:8080 topic_page_cpu.prof")
}

func strPtr(s string) *string {
	return &s
}
