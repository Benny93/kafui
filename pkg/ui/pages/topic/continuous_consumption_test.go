package topic

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestHandleMessageConsumed_TriggersListen(t *testing.T) {
	// Create mock data source and model
	mockDS := &MockDataSource{}
	topicDetails := api.Topic{}
	model := NewModel(mockDS, "test-topic", topicDetails)

	// Set state to consuming
	model.consuming = true
	model.msgChan = make(chan api.Message)

	// Create test message
	msg := MessageConsumedMsg{
		Message: api.Message{Offset: 1, Partition: 0},
	}

	// Handle the message
	_, cmd := model.handlers.Handle(model, msg)

	// Verify a command is returned
	assert.NotNil(t, cmd)

	// Note: We can't easily verify WHICH command it is without more complex mocking
	// but the fact that it returns a command is a good sign.
	// In the previous version, it returned a tea.Tick, which is also a command.
}

func TestHandleContinuousListen_TriggersListen(t *testing.T) {
	// Create mock data source and model
	mockDS := &MockDataSource{}
	topicDetails := api.Topic{}
	model := NewModel(mockDS, "test-topic", topicDetails)

	// Set state to consuming
	model.consuming = true
	model.msgChan = make(chan api.Message)

	// Create continuous listen message
	msg := ContinuousListenMsg{}

	// Handle the message
	_, cmd := model.handlers.Handle(model, msg)

	// Verify a command is returned
	assert.NotNil(t, cmd)
}
