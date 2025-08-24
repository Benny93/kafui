package topic

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/core"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestEnterKeyNavigation(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	// Create test topic
	testTopic := api.Topic{
		NumPartitions:     1,
		ReplicationFactor: 1,
		ReplicaAssignment: make(map[int32][]int32),
		ConfigEntries:     make(map[string]*string),
	}

	// Create new model
	model := NewModel(mockDS, "test-topic", testTopic)

	// Add a test message
	testMessage := api.Message{
		Key:       "test-key",
		Value:     "test-value",
		Offset:    123,
		Partition: 0,
	}
	model.AddMessage(testMessage)

	// Select the first message (index 0)
	model.messageTable.SetCursor(0)

	// Test that Enter key sends the correct PageChangeMsg
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(msg)

	// Should return the same model type
	assert.IsType(t, &Model{}, updatedModel)

	// Should return a command
	assert.NotNil(t, cmd)

	// Execute the command to get the message
	result := cmd()
	
	// Should be a PageChangeMsg
	pageChangeMsg, ok := result.(core.PageChangeMsg)
	assert.True(t, ok, "Expected PageChangeMsg, got %T", result)
	
	// Should navigate to detail page
	assert.Equal(t, "detail", pageChangeMsg.PageID)
	
	// Should contain the selected message
	selectedMsg, ok := pageChangeMsg.Data.(api.Message)
	assert.True(t, ok, "Expected api.Message in Data, got %T", pageChangeMsg.Data)
	assert.Equal(t, testMessage, selectedMsg)
}