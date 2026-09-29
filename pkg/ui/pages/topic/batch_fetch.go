package topic

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
)

// fetchTimeout bounds a single non-streaming fetch.
const fetchTimeout = 5 * time.Second

// FetchLatestMessages fetches the latest N messages from the topic (non-streaming).
// It returns immediately with a StartFetchMsg carrying two channels:
//   - ProgressCh: per-item ProgressMsg updates driven by FetchProgressBar
//   - ResultCh:   the final MessagesFetchedMsg when the fetch completes
func (cc *ConsumptionController) FetchLatestMessages(count int) tea.Cmd {
	// Use a one-shot (non-follow) variant of the flags so the consumer exits as
	// soon as it reaches the end of the partition instead of waiting for new
	// messages. Follow=true would block until the fetch timeout every time.
	flags := cc.model.consumeFlags
	flags.Follow = false
	return cc.startFetch(flags, count, false)
}

// FetchWithFlags fetches a fresh batch using explicit ConsumeFlags (not append).
// Used by the seek dialog and partition filter to (re)start browsing with a
// user-chosen query (MSG-21/MSG-22).
func (cc *ConsumptionController) FetchWithFlags(flags api.ConsumeFlags, count int) tea.Cmd {
	return cc.startFetch(flags, count, false)
}

// FetchNextBatch fetches an additional batch using the provided flags.
// Results are appended to the existing messages (not a fresh start).
func (cc *ConsumptionController) FetchNextBatch(flags api.ConsumeFlags) tea.Cmd {
	return cc.startFetch(flags, int(flags.LimitMessages), true)
}

// startFetch builds the Cmd for one progress-tracked fetch. Call it from
// Update: the topic, generation and datasource are captured here, so the
// background work never touches the model, and the fetch is cancelled when
// the model starts a new generation.
func (cc *ConsumptionController) startFetch(flags api.ConsumeFlags, count int, appendBatch bool) tea.Cmd {
	if count <= 0 {
		count = int(batchSize)
	}
	ds := cc.model.dataSource
	topic, gen := cc.model.topicName, cc.model.fetchGen
	parent := cc.model.generationContext()
	return func() tea.Msg {
		progressCh := components.NewProgressChannel(count)
		resultCh := make(chan MessagesFetchedMsg, 1)
		go func() {
			msgs, err := runFetch(parent, ds, topic, flags, count, progressCh)
			resultCh <- MessagesFetchedMsg{Messages: msgs, Err: err, Append: appendBatch, Topic: topic, Gen: gen}
		}()
		return StartFetchMsg{
			ProgressCh: progressCh,
			ResultCh:   resultCh,
			Total:      count,
			Append:     appendBatch,
			Topic:      topic,
			Gen:        gen,
		}
	}
}

// DecodeVisibleMessages decodes the Key/Value of messages that still hold raw
// Avro bytes. Already-decoded messages pass through unchanged.
// The result is delivered as a VisibleMessagesDecodedMsg.
// Call this once for the full fetched batch so all messages are pre-decoded
// before the user scrolls — avoids per-scroll schema registry round-trips.
// Call it from Update: msgs is copied here, because Update keeps sorting and
// rewriting the model's slice while the decode runs.
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
	msgs = append([]api.Message(nil), msgs...)
	ds := cc.model.dataSource
	topic, gen := cc.model.topicName, cc.model.fetchGen
	ctx := cc.model.generationContext()
	return func() tea.Msg {
		decoded := make([]api.Message, len(msgs))
		for i, msg := range msgs {
			if (msg.Key == "" || msg.Value == "") && (len(msg.RawKey) > 0 || len(msg.RawValue) > 0) {
				if d, err := ds.DecodeMessage(ctx, msg); err == nil {
					decoded[i] = d
				} else {
					decoded[i] = msg
				}
			} else {
				decoded[i] = msg
			}
		}
		return VisibleMessagesDecodedMsg{Messages: decoded, Topic: topic, Gen: gen}
	}
}

// listenForResult returns a Cmd that delivers the MessagesFetchedMsg from resultCh.
func listenForResult(ch <-chan MessagesFetchedMsg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// runFetch consumes up to count messages from topic and returns them together
// with the first error the datasource reported. It stops as soon as count
// messages arrived, the datasource finished or reported an error, or ctx (or
// the fetch timeout) ended, and cancels the consumer at that point.
//
// It sends one ProgressMsg per kept message and a final Done, then closes
// progressCh so every listener still waiting on it returns. Sends never block:
// progress is advisory, and a listener that has moved on must not wedge the
// datasource's partition goroutines. At most count+1 messages are sent, which
// always fits the channel's buffer (see components.NewProgressChannel).
func runFetch(
	parent context.Context,
	ds api.KafkaDataSource,
	topic string,
	flags api.ConsumeFlags,
	count int,
	progressCh chan<- components.ProgressMsg,
) ([]api.Message, error) {
	shared.Log.Info("starting fetch", "topic", topic, "offset", flags.OffsetFlag, "limit", flags.LimitMessages, "target", count)

	ctx, cancel := context.WithTimeout(parent, fetchTimeout)
	defer cancel()

	var (
		messages []api.Message
		mu       sync.Mutex
		done     = make(chan struct{})
		closed   bool
		fetchErr error
	)

	// finish ends the fetch once. Context errors are the fetch's own stop
	// signal (target reached, timeout, superseded), not a failure.
	finish := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		if closed {
			return
		}
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			fetchErr = err
			shared.Log.Error("fetch error", "topic", topic, "err", err)
		}
		closed = true
		close(done)
		cancel()
	}

	handleMsg := func(msg api.Message) {
		mu.Lock()
		if closed || len(messages) >= count {
			mu.Unlock()
			return
		}
		messages = append(messages, msg)
		current := len(messages)
		// Sent under mu so it can never race with the close below.
		select {
		case progressCh <- components.ProgressMsg{Current: current, Total: count}:
		default:
		}
		mu.Unlock()

		if current >= count {
			finish(nil)
		}
	}

	go func() {
		err := ds.ConsumeTopic(ctx, topic, flags, handleMsg, func(e any) {
			finish(asError(e))
		})
		finish(err)
	}()

	<-done

	mu.Lock()
	result := make([]api.Message, len(messages))
	copy(result, messages)
	finalErr := fetchErr
	select {
	case progressCh <- components.ProgressMsg{Current: len(result), Total: count, Done: true}:
	default:
	}
	close(progressCh)
	mu.Unlock()

	shared.Log.Info("fetch complete", "topic", topic, "fetched", len(result), "target", count, "err", finalErr)
	return result, finalErr
}
