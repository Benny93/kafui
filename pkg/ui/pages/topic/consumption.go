package topic

import (
	"context"
	"fmt"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	tea "github.com/charmbracelet/bubbletea"
)

// ConsumptionController handles message consumption logic and error recovery
type ConsumptionController struct {
	model       *Model
	retryPolicy RetryPolicy
}

// NewConsumptionController creates a new consumption controller
func NewConsumptionController(model *Model) *ConsumptionController {
	return &ConsumptionController{
		model:       model,
		retryPolicy: DefaultRetryPolicy(),
	}
}

// StartConsuming starts a live stream for the model's current topic, flags
// and fetch generation. Call it from Update: everything the stream needs is
// captured here, so the returned Cmd never touches the model. The caller owns
// the loading and connection-status state (see Model.startLive).
func (cc *ConsumptionController) StartConsuming() tea.Cmd {
	ds := cc.model.dataSource
	topic, flags, gen := cc.model.topicName, cc.model.consumeFlags, cc.model.fetchGen
	// The stream lives under its generation's context, so ending the
	// generation (or disposing the page) stops it even if its
	// StartConsumingMsg is never delivered.
	parent := cc.model.generationContext()
	return func() tea.Msg {
		// Create context and channels for consumption
		ctx, cancel := context.WithCancel(parent)
		msgChan := make(chan api.Message, 100)
		errChan := make(chan error, 1)

		// Create message handler that sends to our channel
		handleMessage := func(msg api.Message) {
			select {
			case msgChan <- msg:
				// Message sent successfully
			case <-ctx.Done():
				// Context cancelled, stop sending
				return
			default:
				// Channel full, skip message to prevent blocking
			}
		}

		// Create error handler that sends to our error channel. Typed errors
		// (api.AccessDeniedError, ...) are passed through unchanged.
		onError := func(e any) {
			err := asError(e)
			if err == nil {
				return
			}
			select {
			case errChan <- err:
				// Error sent successfully
			case <-ctx.Done():
				// Context cancelled, stop sending
				return
			default:
				// Channel full, skip error to prevent blocking
			}
		}

		// Start consumption in a goroutine
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// Handle panic by sending error
					onError(fmt.Errorf("panic in consumption: %v", r))
				}
				close(msgChan)
				close(errChan)
			}()

			err := ds.ConsumeTopic(ctx, topic, flags, handleMessage, onError)
			if err != nil && ctx.Err() == nil {
				onError(err)
			}
		}()

		return StartConsumingMsg{
			MsgChan: msgChan,
			ErrChan: errChan,
			Cancel:  cancel,
			Topic:   topic,
			Gen:     gen,
		}
	}
}

// asError turns the value a datasource passes to its onError callback into an
// error, keeping typed errors intact so errors.As still finds them.
func asError(e any) error {
	switch v := e.(type) {
	case nil:
		return nil
	case error:
		return v
	default:
		return fmt.Errorf("%v", v)
	}
}

// StopConsuming stops message consumption
func (cc *ConsumptionController) StopConsuming() tea.Cmd {
	return func() tea.Msg {
		return StopConsumingMsg{}
	}
}

// ListenForMessages creates a command to listen for incoming messages on the
// current generation's stream. A closed channel is reported as a
// streamClosedMsg; the handler decides whether that is an error.
func (cc *ConsumptionController) ListenForMessages(msgChan <-chan api.Message) tea.Cmd {
	topic, gen := cc.model.topicName, cc.model.fetchGen
	return func() tea.Msg {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				return streamClosedMsg{Topic: topic, Gen: gen}
			}
			return MessageConsumedMsg{Message: msg, Topic: topic, Gen: gen}

		case <-time.After(time.Millisecond * 500):
			// No message received, continue listening
			return ContinuousListenMsg{Topic: topic, Gen: gen}
		}
	}
}

// ListenForErrors creates a command to listen for consumption errors on the
// current generation's stream.
func (cc *ConsumptionController) ListenForErrors(errChan <-chan error) tea.Cmd {
	topic, gen := cc.model.topicName, cc.model.fetchGen
	return func() tea.Msg {
		select {
		case err, ok := <-errChan:
			if !ok {
				return streamClosedMsg{Topic: topic, Gen: gen}
			}
			return liveErrorMsg{Topic: topic, Gen: gen, Err: err}

		case <-time.After(time.Second * 1):
			// No error received, continue listening
			return ContinuousErrorListenMsg{Topic: topic, Gen: gen}
		}
	}
}
