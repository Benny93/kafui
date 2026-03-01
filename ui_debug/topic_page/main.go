package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/pages/topic"
	tea "github.com/charmbracelet/bubbletea"
)

// MockDataSource provides mock Kafka data for testing
type MockDataSource struct{}

func (m *MockDataSource) Init(cfgOption string)                              {}
func (m *MockDataSource) GetTopics() (map[string]api.Topic, error)           { return nil, nil }
func (m *MockDataSource) GetConsumerGroups() ([]api.ConsumerGroup, error)    { return nil, nil }
func (m *MockDataSource) GetContexts() ([]string, error)                     { return nil, nil }
func (m *MockDataSource) GetContext() string                                 { return "mock" }
func (m *MockDataSource) SetContext(contextName string) error                { return nil }
func (m *MockDataSource) GetMessageSchemaInfo(keySchemaID, valueSchemaID string) (*api.MessageSchemaInfo, error) {
	return nil, nil
}

// ConsumeTopic simulates consuming messages from a topic
func (m *MockDataSource) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) error {
	// Generate mock messages continuously
	offset := int64(0)
	partitions := []int32{0, 1, 2}
	
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Create mock message with varying content lengths
			partition := partitions[offset%int64(len(partitions))]
			
			// Simulate different message sizes
			var key, value string
			switch offset % 5 {
			case 0:
				key = fmt.Sprintf("key-%d", offset)
				value = fmt.Sprintf(`{"id": %d, "status": "ok", "data": "short"}`, offset)
			case 1:
				key = fmt.Sprintf("user-event-%d", offset)
				value = fmt.Sprintf(`{"user_id": %d, "event": "login", "timestamp": "%s", "ip": "192.168.1.%d", "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"}`, 
					offset, time.Now().Format(time.RFC3339), offset%255)
			case 2:
				key = fmt.Sprintf("order-%d", offset)
				value = fmt.Sprintf(`{"order_id": %d, "items": [{"sku": "PROD-%d", "qty": 2, "price": 29.99}, {"sku": "PROD-%d", "qty": 1, "price": 49.99}], "total": 109.97, "shipping_address": "123 Main St, Anytown, ST 12345"}`, 
					offset, offset, offset+1)
			case 3:
				key = fmt.Sprintf("log-%d", offset)
				value = fmt.Sprintf(`[ERROR] %s - Database connection timeout after 30000ms. Retrying... (attempt %d/3) | Connection pool: active=5, idle=2, waiting=12 | Last successful query: SELECT * FROM users WHERE id IN (%s)`, 
					time.Now().Format(time.RFC3339), (offset%3)+1, generateLongIDList(offset))
			case 4:
				key = fmt.Sprintf("metric-%d", offset)
				value = fmt.Sprintf(`{"cpu": %.2f, "memory": %.2f, "disk": %.2f, "network_in": %d, "network_out": %d, "processes": %d, "load_avg": [%.2f, %.2f, %.2f]}`, 
					float64(offset%100)/10, float64(offset%80)/10, float64(offset%90)/10, 
					offset*1024, offset*512, 50+offset%100, float64(offset%50)/10, float64(offset%40)/10, float64(offset%30)/10)
			}

			msg := api.Message{
				Offset:    offset,
				Partition: partition,
				Key:       key,
				Value:     value,
			}
			handleMessage(msg)
			offset++
		}
	}
}

func generateLongIDList(offset int64) string {
	ids := make([]byte, 0, 100)
	for i := 0; i < 20; i++ {
		if i > 0 {
			ids = append(ids, ',')
		}
		ids = append(ids, fmt.Sprintf("%d", (offset+int64(i))%1000)...)
	}
	return string(ids)
}

func main() {
	// Create mock data source
	dataSource := &MockDataSource{}

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

	// Run the TUI with a minimum window size suggestion
	// Note: Sidebar requires width >= 120 and height >= 30 to be visible by default
	// You can toggle sidebar with 't' or 'Ctrl+S' when window is wide enough
	p := tea.NewProgram(topicPage, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
	}
}

func strPtr(s string) *string {
	return &s
}
