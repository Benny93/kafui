package topic

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components"
	"github.com/Benny93/kafui/pkg/ui/shared"
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

// StartConsuming initiates message consumption for the topic
func (cc *ConsumptionController) StartConsuming() tea.Cmd {
	return func() tea.Msg {
		// Set consuming state
		cc.model.loading = true
		cc.model.SetConnectionStatus(StatusConnecting)

		// Create context and channels for consumption
		ctx, cancel := context.WithCancel(context.Background())
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

		// Create error handler that sends to our error channel
		onError := func(err any) {
			select {
			case errChan <- fmt.Errorf("%v", err):
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

			err := cc.model.dataSource.ConsumeTopic(ctx, cc.model.topicName, cc.model.consumeFlags, handleMessage, onError)
			if err != nil {
				onError(err)
			}
		}()

		return StartConsumingMsg{
			MsgChan: msgChan,
			ErrChan: errChan,
			Cancel:  cancel,
		}
	}
}

// StopConsuming stops message consumption
func (cc *ConsumptionController) StopConsuming() tea.Cmd {
	return func() tea.Msg {
		return StopConsumingMsg{}
	}
}

// ListenForMessages creates a command to listen for incoming messages
func (cc *ConsumptionController) ListenForMessages(msgChan <-chan api.Message) tea.Cmd {
	return func() tea.Msg {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				// Channel closed - this is normal for fetch operations
				// Only report error if we're in continuous consumption mode
				if cc.model.consuming {
					return ErrorMsg(shared.NewUIError(
						shared.ErrorTypeDataLoad,
						"Message stream closed unexpectedly",
						nil,
					))
				}
				// For fetch operations, just return nil to stop listening
				return nil
			}

			// Return the message via the command
			return MessageConsumedMsg{Message: msg}

		case <-time.After(time.Millisecond * 500):
			// No message received, continue listening
			return ContinuousListenMsg{}
		}
	}
}

// ListenForErrors creates a command to listen for consumption errors
func (cc *ConsumptionController) ListenForErrors(errChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case err, ok := <-errChan:
			if !ok {
				// Channel closed — only report if still in live consumption.
				// A deliberate cancel (mode switch, page leave) sets consuming=false
				// before the channel closes, so we silently return nil in that case.
				if cc.model.consuming {
					return ErrorMsg(shared.NewUIError(
						shared.ErrorTypeConnection,
						"Error stream closed unexpectedly",
						nil,
					))
				}
				return nil
			}

			// Return the error via the command
			return ErrorMsg(err)

		case <-time.After(time.Second * 1):
			// No error received, continue listening
			return ContinuousErrorListenMsg{}
		}
	}
}

// RetryConnection attempts to retry the connection after a failure
func (cc *ConsumptionController) RetryConnection() tea.Cmd {
	return func() tea.Msg {
		cc.model.retryCount++

		if cc.model.retryCount > cc.retryPolicy.MaxRetries {
			return ConnectionFailedMsg{
				Attempts:  cc.model.retryCount,
				LastError: cc.model.lastError,
			}
		}

		// Schedule retry after delay
		return cc.ScheduleRetry(cc.model.lastError)
	}
}

// ScheduleRetry schedules a retry attempt after a delay
func (cc *ConsumptionController) ScheduleRetry(err error) tea.Cmd {
	delay := cc.calculateRetryDelay(cc.model.retryCount)

	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return RetryConsumptionMsg{
			Attempt:   cc.model.retryCount,
			LastError: err,
		}
	})
}

// calculateRetryDelay calculates the delay for the next retry attempt
func (cc *ConsumptionController) calculateRetryDelay(attempt int) time.Duration {
	if !cc.retryPolicy.EnableExponential {
		return cc.retryPolicy.InitialDelay
	}

	// Exponential backoff: delay = initial * (backoffFactor ^ (attempt - 1))
	delay := cc.retryPolicy.InitialDelay
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * cc.retryPolicy.BackoffFactor)
	}

	// Cap at maximum delay
	if delay > cc.retryPolicy.MaxDelay {
		delay = cc.retryPolicy.MaxDelay
	}

	return delay
}

// ResetRetryCount resets the retry counter
func (cc *ConsumptionController) ResetRetryCount() {
	cc.model.retryCount = 0
}

// SetRetryPolicy updates the retry policy
func (cc *ConsumptionController) SetRetryPolicy(policy RetryPolicy) {
	cc.retryPolicy = policy
}

// GetRetryPolicy returns the current retry policy
func (cc *ConsumptionController) GetRetryPolicy() RetryPolicy {
	return cc.retryPolicy
}

// IsRetrying returns whether the controller is currently retrying
func (cc *ConsumptionController) IsRetrying() bool {
	return cc.model.retryCount > 0 && cc.model.retryCount <= cc.retryPolicy.MaxRetries
}

// GetRetryStatus returns information about retry status
func (cc *ConsumptionController) GetRetryStatus() (current, max int, nextDelay time.Duration) {
	current = cc.model.retryCount
	max = cc.retryPolicy.MaxRetries

	if current < max {
		nextDelay = cc.calculateRetryDelay(current + 1)
	}

	return current, max, nextDelay
}

// HandlePanicRecovery handles panics in consumption goroutines
func (cc *ConsumptionController) HandlePanicRecovery() tea.Cmd {
	return func() tea.Msg {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in consumption: %v", r)
			return ErrorMsg(shared.NewUIError(
				shared.ErrorTypeDataLoad,
				"Unexpected error during message consumption",
				err,
			))
		}
		return nil
	}
}

// ValidateConsumptionFlags validates the consumption configuration
func (cc *ConsumptionController) ValidateConsumptionFlags(flags api.ConsumeFlags) error {
	if flags.OffsetFlag == "" {
		return shared.NewUIError(
			shared.ErrorTypeValidation,
			"Offset flag cannot be empty",
			nil,
		)
	}

	if flags.Tail < 0 {
		return shared.NewUIError(
			shared.ErrorTypeValidation,
			fmt.Sprintf("Invalid tail value: %d (must be non-negative)", flags.Tail),
			nil,
		)
	}

	return nil
}

// GetConsumptionStats returns statistics about message consumption
func (cc *ConsumptionController) GetConsumptionStats() ConsumptionStats {
	return ConsumptionStats{
		MessagesConsumed: len(cc.model.messages),
		ErrorCount:       len(cc.model.errorHistory),
		RetryCount:       cc.model.retryCount,
		IsConsuming:      cc.model.consuming,
		IsPaused:         cc.model.paused,
		ConnectionStatus: cc.model.connectionStatus,
		LastError:        cc.model.lastError,
		StartTime:        time.Time{},
		Duration:         time.Since(time.Time{}),
	}
}

// ConsumptionStats contains statistics about message consumption
type ConsumptionStats struct {
	MessagesConsumed int
	ErrorCount       int
	RetryCount       int
	IsConsuming      bool
	IsPaused         bool
	ConnectionStatus string
	LastError        error
	StartTime        time.Time
	Duration         time.Duration
}

// String returns a string representation of the consumption stats
func (cs ConsumptionStats) String() string {
	return fmt.Sprintf(
		"Messages: %d, Errors: %d, Retries: %d, Status: %s, Consuming: %t, Paused: %t",
		cs.MessagesConsumed,
		cs.ErrorCount,
		cs.RetryCount,
		cs.ConnectionStatus,
		cs.IsConsuming,
		cs.IsPaused,
	)
}

// UpdateConsumptionFlags updates the consumption flags and restarts consumption if needed
func (cc *ConsumptionController) UpdateConsumptionFlags(flags api.ConsumeFlags) tea.Cmd {
	if err := cc.ValidateConsumptionFlags(flags); err != nil {
		return func() tea.Msg {
			return ErrorMsg(err)
		}
	}

	cc.model.consumeFlags = flags

	// If currently consuming, restart with new flags
	if cc.model.consuming {
		return tea.Batch(
			cc.StopConsuming(),
			cc.StartConsuming(),
		)
	}

	return nil
}

// GetHealthStatus returns the health status of the consumption controller
func (cc *ConsumptionController) GetHealthStatus() string {
	if cc.model.error != nil {
		return "unhealthy"
	}

	if cc.model.consuming && !cc.model.paused {
		return "healthy"
	}

	if cc.model.paused {
		return "paused"
	}

	return "idle"
}

// FetchLatestMessages fetches the latest N messages from the topic (non-streaming).
// It returns immediately with a StartFetchMsg carrying two channels:
//   - ProgressCh: per-item ProgressMsg updates driven by FetchProgressBar
//   - ResultCh:   the final MessagesFetchedMsg when the fetch completes
func (cc *ConsumptionController) FetchLatestMessages(count int) tea.Cmd {
	return func() tea.Msg {
		progressCh := components.NewProgressChannel(count)
		resultCh := make(chan MessagesFetchedMsg, 1)
		go cc.fetchWithProgress(count, progressCh, resultCh)
		return StartFetchMsg{ProgressCh: progressCh, ResultCh: resultCh, Total: count}
	}
}

// FetchWithFlags fetches a fresh batch using explicit ConsumeFlags (not append).
// Used by the seek dialog and partition filter to (re)start browsing with a
// user-chosen query (MSG-21/MSG-22).
func (cc *ConsumptionController) FetchWithFlags(flags api.ConsumeFlags, count int) tea.Cmd {
	if count <= 0 {
		count = int(batchSize)
	}
	return func() tea.Msg {
		progressCh := components.NewProgressChannel(count)
		resultCh := make(chan MessagesFetchedMsg, 1)
		go cc.fetchWithProgressFlags(flags, count, progressCh, resultCh)
		return StartFetchMsg{ProgressCh: progressCh, ResultCh: resultCh, Total: count, Append: false}
	}
}

// FetchNextBatch fetches an additional batch using the provided flags.
// Results are appended to the existing messages (not a fresh start).
func (cc *ConsumptionController) FetchNextBatch(flags api.ConsumeFlags) tea.Cmd {
	count := int(flags.LimitMessages)
	if count <= 0 {
		count = int(batchSize)
	}
	return func() tea.Msg {
		progressCh := components.NewProgressChannel(count)
		resultCh := make(chan MessagesFetchedMsg, 1)
		go cc.fetchWithProgressFlags(flags, count, progressCh, resultCh)
		return StartFetchMsg{ProgressCh: progressCh, ResultCh: resultCh, Total: count, Append: true}
	}
}

// DecodeVisibleMessages decodes the Key/Value of messages that still hold raw
// Avro bytes. Already-decoded messages pass through unchanged.
// The result is delivered as a VisibleMessagesDecodedMsg.
// Call this once for the full fetched batch so all messages are pre-decoded
// before the user scrolls — avoids per-scroll schema registry round-trips.
func (cc *ConsumptionController) DecodeVisibleMessages(msgs []api.Message) tea.Cmd {
	if len(msgs) == 0 {
		return nil
	}
	// Check whether any message actually needs decoding before spawning a goroutine.
	needsDecode := false
	for _, m := range msgs {
		if (m.Key == "" || m.Value == "") && (len(m.RawKey) > 0 || len(m.RawValue) > 0) {
			needsDecode = true
			break
		}
	}
	if !needsDecode {
		return nil
	}
	return func() tea.Msg {
		decoded := make([]api.Message, len(msgs))
		for i, msg := range msgs {
			if (msg.Key == "" || msg.Value == "") && (len(msg.RawKey) > 0 || len(msg.RawValue) > 0) {
				if d, err := cc.model.dataSource.DecodeMessage(context.Background(), msg); err == nil {
					decoded[i] = d
				} else {
					decoded[i] = msg
				}
			} else {
				decoded[i] = msg
			}
		}
		return VisibleMessagesDecodedMsg{Messages: decoded}
	}
}

// listenForResult returns a Cmd that delivers the MessagesFetchedMsg from resultCh.
func listenForResult(ch <-chan MessagesFetchedMsg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// fetchWithProgress runs the actual Kafka consumption in the background,
// sending one ProgressMsg per received message via progressCh and delivering
// the complete result set to resultCh when done.
func (cc *ConsumptionController) fetchWithProgress(
	count int,
	progressCh chan<- components.ProgressMsg,
	resultCh chan<- MessagesFetchedMsg,
) {
	shared.Log.Info("starting fetch", "topic", cc.model.topicName, "target", count)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var (
		messages []api.Message
		mu       sync.Mutex
		done     = make(chan struct{})
		closed   bool
		fetchErr error
	)

	closeDone := func() {
		mu.Lock()
		defer mu.Unlock()
		if !closed {
			closed = true
			close(done)
		}
	}

	handleMsg := func(msg api.Message) {
		mu.Lock()
		messages = append(messages, msg)
		current := len(messages)
		mu.Unlock()

		progressCh <- components.ProgressMsg{Current: current, Total: count}
		if current >= count {
			closeDone()
		}
	}

	go func() {
		// Use a one-shot (non-follow) variant of the flags so the consumer
		// exits as soon as it reaches the end of the partition instead of
		// waiting for new messages. Follow=true would block until the 5s
		// context timeout every single fetch.
		fetchFlags := cc.model.consumeFlags
		fetchFlags.Follow = false
		err := cc.model.dataSource.ConsumeTopic(
			ctx, cc.model.topicName, fetchFlags, handleMsg,
			func(e any) {
				if e != nil {
					mu.Lock()
					fetchErr = fmt.Errorf("%v", e)
					mu.Unlock()
					shared.Log.Error("fetch error callback", "topic", cc.model.topicName, "err", fetchErr)
				}
				closeDone()
			},
		)
		if err != nil {
			mu.Lock()
			fetchErr = err
			mu.Unlock()
			shared.Log.Error("ConsumeTopic returned error", "topic", cc.model.topicName, "err", err)
		}
		closeDone()
	}()

	<-done

	mu.Lock()
	result := make([]api.Message, len(messages))
	copy(result, messages)
	finalErr := fetchErr
	mu.Unlock()

	shared.Log.Info("fetch complete", "topic", cc.model.topicName, "fetched", len(result), "target", count, "err", finalErr)

	progressCh <- components.ProgressMsg{Current: len(result), Total: count, Done: true}
	resultCh <- MessagesFetchedMsg{Messages: result}
}

// fetchWithProgressFlags is identical to fetchWithProgress but uses an explicit
// set of ConsumeFlags instead of deriving them from the current model state.
// Used by FetchNextBatch to load additional messages at a specific offset.
func (cc *ConsumptionController) fetchWithProgressFlags(
	flags api.ConsumeFlags,
	count int,
	progressCh chan<- components.ProgressMsg,
	resultCh chan<- MessagesFetchedMsg,
) {
	shared.Log.Info("starting batch fetch", "topic", cc.model.topicName, "offset", flags.OffsetFlag, "limit", flags.LimitMessages)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var (
		messages []api.Message
		mu       sync.Mutex
		done     = make(chan struct{})
		closed   bool
		fetchErr error
	)

	closeDone := func() {
		mu.Lock()
		defer mu.Unlock()
		if !closed {
			closed = true
			close(done)
		}
	}

	handleMsg := func(msg api.Message) {
		mu.Lock()
		messages = append(messages, msg)
		current := len(messages)
		mu.Unlock()

		progressCh <- components.ProgressMsg{Current: current, Total: count}
		if count > 0 && current >= count {
			closeDone()
		}
	}

	go func() {
		err := cc.model.dataSource.ConsumeTopic(
			ctx, cc.model.topicName, flags, handleMsg,
			func(e any) {
				if e != nil {
					mu.Lock()
					fetchErr = fmt.Errorf("%v", e)
					mu.Unlock()
					shared.Log.Error("batch fetch error callback", "topic", cc.model.topicName, "err", fetchErr)
				}
				closeDone()
			},
		)
		if err != nil {
			mu.Lock()
			fetchErr = err
			mu.Unlock()
			shared.Log.Error("ConsumeTopic (batch) returned error", "topic", cc.model.topicName, "err", err)
		}
		closeDone()
	}()

	<-done

	mu.Lock()
	result := make([]api.Message, len(messages))
	copy(result, messages)
	finalErr := fetchErr
	mu.Unlock()

	shared.Log.Info("batch fetch complete", "topic", cc.model.topicName, "fetched", len(result), "err", finalErr)

	progressCh <- components.ProgressMsg{Current: len(result), Total: count, Done: true}
	resultCh <- MessagesFetchedMsg{Messages: result}
}
