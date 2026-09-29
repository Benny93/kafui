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
		Topic:   "test-topic",
		Gen:     model.fetchGen,
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
	msg := ContinuousListenMsg{Topic: "test-topic", Gen: model.fetchGen}

	// Handle the message
	_, cmd := model.handlers.Handle(model, msg)

	// Verify a command is returned
	assert.NotNil(t, cmd)
}

// A message from a replaced stream, or from another topic's page, must neither
// land in the table nor re-arm a second listener on the current stream.
func TestHandleMessageConsumed_DropsStale(t *testing.T) {
	tests := []struct {
		name  string
		topic string
		gen   func(m *Model) uint64
	}{
		{"older generation", "test-topic", func(m *Model) uint64 { return m.fetchGen - 1 }},
		{"other topic", "other-topic", func(m *Model) uint64 { return m.fetchGen }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewModel(&MockDataSource{}, "test-topic", api.Topic{})
			model.beginGeneration()
			model.consuming = true
			model.msgChan = make(chan api.Message)

			_, cmd := model.handlers.Handle(model, MessageConsumedMsg{
				Message: api.Message{Offset: 7},
				Topic:   tt.topic,
				Gen:     tt.gen(model),
			})

			assert.Nil(t, cmd)
			assert.Empty(t, model.messages)
		})
	}
}

// While paused, live messages are buffered (the stream keeps draining) and the
// table stays frozen; resuming merges them in.
func TestPause_BuffersLiveMessagesUntilResume(t *testing.T) {
	model := NewModel(&MockDataSource{}, "test-topic", api.Topic{})
	model.consuming = true
	model.msgChan = make(chan api.Message)
	model.TogglePause()

	for i := int64(0); i < 3; i++ {
		_, cmd := model.handlers.Handle(model, MessageConsumedMsg{
			Message: api.Message{Offset: i},
			Topic:   "test-topic",
			Gen:     model.fetchGen,
		})
		assert.NotNil(t, cmd, "the stream must keep being drained while paused")
	}
	assert.Empty(t, model.messages)
	assert.Len(t, model.pendingWhilePaused, 3)

	model.TogglePause()
	assert.Len(t, model.messages, 3)
	assert.Empty(t, model.pendingWhilePaused)
}
