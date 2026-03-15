package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/pages/topic"
	tea "github.com/charmbracelet/bubbletea"
)

// MockDataSource provides mock Kafka data for testing
type MockDataSource struct {
	messages []api.Message  // Pre-generated message pool
}

func (m *MockDataSource) Init(cfgOption string) {
	// Pre-generate 100 simple messages for testing (single partition, offsets 1-100)
	m.messages = make([]api.Message, 0, 100)

	for i := int64(1); i <= 100; i++ {
		m.messages = append(m.messages, createMockMessage(i, 0))
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
// Sends all 100 messages (newest first) and stops
func (m *MockDataSource) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) error {
	totalMessages := len(m.messages)

	// Safety check: prevent divide by zero
	if totalMessages == 0 {
		return fmt.Errorf("no messages available in mock data source")
	}

	// Send all messages (newest first, offset 100 down to 1)
	for i := totalMessages - 1; i >= 0; i-- {
		select {
		case <-ctx.Done():
			return nil
		default:
			handleMessage(m.messages[i])
		}
	}

	// Wait for context cancellation (no continuous consumption for simplicity)
	<-ctx.Done()
	return nil
}

// createMockMessage creates a simple mock message
func createMockMessage(offset int64, partition int32) api.Message {
	return api.Message{
		Offset:    offset,
		Partition: partition,
		Key:       fmt.Sprintf("key-%d", offset),
		Value:     fmt.Sprintf(`{"id": %d, "data": "message content %d"}`, offset, offset),
	}
}

func main() {
	// Create mock data source
	dataSource := &MockDataSource{}
	dataSource.Init("") // Initialize mock messages (100 messages, partition 0, offsets 1-100)

	// Create topic details
	topicDetails := api.Topic{
		NumPartitions:     1,
		ReplicationFactor: 1,
		MessageCount:      100,
	}

	// Create topic page model
	topicPage := topic.NewTopicPageModel(dataSource, "test-topic", topicDetails)

	// Run the TUI
	p := tea.NewProgram(topicPage, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
