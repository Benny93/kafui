package topic

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// consumeFakeDS overrides ConsumeTopic with a plain function.
type consumeFakeDS struct {
	*MockDataSource
	consume func(ctx context.Context, handle api.MessageHandlerFunc, onError func(any)) error
}

func (f *consumeFakeDS) ConsumeTopic(ctx context.Context, _ string, _ api.ConsumeFlags, handle api.MessageHandlerFunc, onError func(err any)) error {
	return f.consume(ctx, handle, onError)
}

func newLifecycleModel() *Model {
	return NewModel(&MockDataSource{}, "test-topic", api.Topic{})
}

func TestHandleMessagesFetched_Generations(t *testing.T) {
	loaded := []api.Message{{Offset: 1}, {Offset: 2}}

	t.Run("stale result is dropped", func(t *testing.T) {
		m := newLifecycleModel()
		m.beginGeneration()
		m.messages = loaded
		m.handlers.Handle(m, MessagesFetchedMsg{Messages: []api.Message{{Offset: 99}}, Topic: "test-topic", Gen: m.fetchGen - 1})
		assert.Equal(t, loaded, m.messages)
	})

	t.Run("other topic's result is dropped", func(t *testing.T) {
		m := newLifecycleModel()
		m.beginGeneration()
		m.messages = loaded
		m.handlers.Handle(m, MessagesFetchedMsg{Messages: []api.Message{{Offset: 99}}, Topic: "other", Gen: m.fetchGen})
		assert.Equal(t, loaded, m.messages)
	})

	t.Run("append flag travels with the result", func(t *testing.T) {
		m := newLifecycleModel()
		m.beginGeneration()
		m.handlers.Handle(m, MessagesFetchedMsg{Messages: loaded, Topic: "test-topic", Gen: m.fetchGen})
		m.handlers.Handle(m, MessagesFetchedMsg{Messages: []api.Message{{Offset: 3}}, Append: true, Topic: "test-topic", Gen: m.fetchGen})
		assert.Len(t, m.messages, 3)
		// A fresh result replaces, whatever append fetches were counted.
		m.appendNextFetch = 1
		m.handlers.Handle(m, MessagesFetchedMsg{Messages: []api.Message{{Offset: 5}}, Topic: "test-topic", Gen: m.fetchGen})
		assert.Len(t, m.messages, 1)
	})
}

func TestHandleMessagesFetched_ErrorIsNotAnEmptyTopic(t *testing.T) {
	m := newLifecycleModel()
	m.beginGeneration()
	denied := api.AccessDeniedError{Resource: "topic", Name: "test-topic", Action: "read"}

	_, cmd := m.handlers.Handle(m, MessagesFetchedMsg{Err: denied, Topic: "test-topic", Gen: m.fetchGen})

	require.NotNil(t, cmd, "the error must reach the status bar")
	assert.Equal(t, error(denied), m.error)
	assert.NotContains(t, m.statusMessage, "empty")
	assert.False(t, m.loading)
}

func TestStartForMode_ClearsStickyError(t *testing.T) {
	m := newLifecycleModel()
	m.SetError(errors.New("stream closed"))
	m.retryCount = 2

	m.startForMode()

	assert.Nil(t, m.error)
	assert.Zero(t, m.retryCount)
}

func TestLiveConsumer_IsAlwaysCancelled(t *testing.T) {
	startLive := func(m *Model) *bool {
		cancelled := false
		m.handlers.Handle(m, StartConsumingMsg{
			MsgChan: make(chan api.Message),
			ErrChan: make(chan error),
			Cancel:  func() { cancelled = true },
			Topic:   "test-topic",
			Gen:     m.fetchGen,
		})
		require.True(t, m.consuming)
		return &cancelled
	}

	tests := []struct {
		name  string
		leave func(m *Model)
	}{
		{"leaving the page", func(m *Model) { m.OnBlur() }},
		{"refresh", func(m *Model) { m.keys.handleRefresh(m) }},
		{"seek", func(m *Model) { m.startForFlags(api.ConsumeFlags{OffsetFlag: "oldest", LimitMessages: 10}) }},
		{"retry", func(m *Model) {
			m.handlers.Handle(m, RetryConsumptionMsg{Attempt: 1, Topic: "test-topic", Gen: m.fetchGen})
		}},
		{"a second stream arriving", func(m *Model) { startLive(m) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newLifecycleModel()
			m.consumeMode = ModeLive
			cancelled := startLive(m)

			tt.leave(m)

			assert.True(t, *cancelled, "the previous consumer must be cancelled")
		})
	}

	t.Run("stale stream is cancelled on arrival", func(t *testing.T) {
		m := newLifecycleModel()
		m.beginGeneration()
		cancelled := false
		m.handlers.Handle(m, StartConsumingMsg{Cancel: func() { cancelled = true }, Topic: "test-topic", Gen: m.fetchGen - 1})
		assert.True(t, cancelled)
		assert.False(t, m.consuming)
	})
}

func TestLiveErrors_ScheduleOneRetry(t *testing.T) {
	m := newLifecycleModel()
	m.consuming = true
	m.msgChan = make(chan api.Message)
	m.errChan = make(chan error)

	_, first := m.handlers.Handle(m, liveErrorMsg{Topic: "test-topic", Gen: m.fetchGen, Err: errors.New("partition 0 failed")})
	_, second := m.handlers.Handle(m, liveErrorMsg{Topic: "test-topic", Gen: m.fetchGen, Err: errors.New("partition 1 failed")})
	_, closed := m.handlers.Handle(m, streamClosedMsg{Topic: "test-topic", Gen: m.fetchGen})

	assert.NotNil(t, first, "the first failure schedules a retry")
	assert.Nil(t, second, "later errors of the failed stream are ignored")
	assert.Nil(t, closed, "so is its channel closing")
	assert.Equal(t, 1, m.retryCount)
}

// runFetch must return once count messages arrived, even when the datasource
// delivers far more (the limit is per partition) and nobody reads progress.
func TestRunFetch_NeverBlocksOnProgress(t *testing.T) {
	const partitions, perPartition, count = 4, 60, 10
	ds := &consumeFakeDS{MockDataSource: &MockDataSource{}, consume: func(ctx context.Context, handle api.MessageHandlerFunc, _ func(any)) error {
		var wg sync.WaitGroup
		for p := int32(0); p < partitions; p++ {
			wg.Add(1)
			go func(p int32) {
				defer wg.Done()
				for o := int64(0); o < perPartition; o++ {
					handle(api.Message{Partition: p, Offset: o})
				}
			}(p)
		}
		wg.Wait()
		return nil
	}}
	progressCh := components.NewProgressChannel(count)

	done := make(chan struct{})
	var msgs []api.Message
	var err error
	go func() {
		msgs, err = runFetch(context.Background(), ds, "test-topic", api.ConsumeFlags{}, count, progressCh)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("runFetch blocked")
	}
	assert.NoError(t, err)
	assert.Len(t, msgs, count)

	// The channel is closed after the final Done, so no listener waits forever.
	var last components.ProgressMsg
	for p := range progressCh {
		last = p
	}
	assert.True(t, last.Done)
}

func TestRunFetch_KeepsTypedErrors(t *testing.T) {
	denied := api.AccessDeniedError{Resource: "topic", Name: "t", Action: "read"}
	ds := &consumeFakeDS{MockDataSource: &MockDataSource{}, consume: func(ctx context.Context, _ api.MessageHandlerFunc, onError func(any)) error {
		onError(denied)
		return nil
	}}

	_, err := runFetch(context.Background(), ds, "t", api.ConsumeFlags{}, 10, components.NewProgressChannel(10))

	var target api.AccessDeniedError
	assert.True(t, errors.As(err, &target), "got %v", err)
}

func TestRunFetch_OwnCancellationIsNotAnError(t *testing.T) {
	ds := &consumeFakeDS{MockDataSource: &MockDataSource{}, consume: func(ctx context.Context, handle api.MessageHandlerFunc, _ func(any)) error {
		for o := int64(0); ; o++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				handle(api.Message{Offset: o})
			}
		}
	}}

	msgs, err := runFetch(context.Background(), ds, "t", api.ConsumeFlags{}, 5, components.NewProgressChannel(5))

	assert.NoError(t, err)
	assert.Len(t, msgs, 5)
}

func TestInitContent_ZeroCountOnlySkipsWhenKnown(t *testing.T) {
	tests := []struct {
		name      string
		details   api.Topic
		wantFetch bool
	}{
		{"no topic data (consumer group, connector, deep link)", api.Topic{}, true},
		{"unknown count", api.Topic{NumPartitions: 3, MessageCount: -1}, true},
		{"known empty", api.Topic{NumPartitions: 3, MessageCount: 0}, false},
		{"known non-empty", api.Topic{NumPartitions: 3, MessageCount: 10}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(&MockDataSource{}, "test-topic", tt.details)
			cmd := NewTopicContentProvider(m).InitContent()
			assert.Equal(t, tt.wantFetch, cmd != nil)
			assert.Equal(t, tt.wantFetch, m.loading)
		})
	}
}

func TestHandleTopicMetaLoaded_FillsDetails(t *testing.T) {
	m := newLifecycleModel()
	details := api.TopicDetails{
		ReplicationFactor: 3,
		Partitions: []api.PartitionInfo{
			{ID: 0, EarliestOffset: 0, LatestOffset: 5},
			{ID: 1, EarliestOffset: 0, LatestOffset: 7},
		},
	}

	m.handlers.Handle(m, topicMetaLoadedMsg{Topic: "test-topic", Details: details})

	assert.EqualValues(t, 2, m.topicDetails.NumPartitions)
	assert.EqualValues(t, 3, m.topicDetails.ReplicationFactor)
	assert.EqualValues(t, 12, m.topicDetails.MessageCount)
}

var (
	_ core.InputModeReporter = (*TopicPageModel)(nil)
	_ core.Disposer          = (*TopicPageModel)(nil)
)

func TestIsInputMode(t *testing.T) {
	p := NewTopicPageModel(&MockDataSource{}, "test-topic", api.Topic{})
	assert.False(t, p.IsInputMode())
	p.topicModel.keys.handleSearch(p.topicModel)
	assert.True(t, p.IsInputMode())
	p.topicModel.Unwind()
	assert.False(t, p.IsInputMode())
}

// The router runs Init once, when the page is created, then OnFocus on every
// activation including the first.
func TestRouterLifecycle(t *testing.T) {
	details := api.Topic{NumPartitions: 1, MessageCount: 10}

	t.Run("first open fetches exactly once", func(t *testing.T) {
		p := NewTopicPageModel(&MockDataSource{}, "test-topic", details)
		p.Init()
		gen := p.topicModel.fetchGen
		assert.EqualValues(t, 1, gen)

		assert.Nil(t, p.OnFocus())
		assert.Equal(t, gen, p.topicModel.fetchGen, "OnFocus must not start a second fetch")
	})

	t.Run("back to a live page resumes the stream", func(t *testing.T) {
		p := NewTopicPageModel(&MockDataSource{}, "test-topic", details)
		m := p.topicModel
		p.Init()
		p.OnFocus()
		m.consumeMode = ModeLive
		m.consumeFlags = consumeFlagsForMode(ModeLive)
		m.startLive()
		cancelled := false
		m.handlers.Handle(m, StartConsumingMsg{Cancel: func() { cancelled = true }, Topic: "test-topic", Gen: m.fetchGen})
		require.True(t, m.consuming)
		kept := []api.Message{{Offset: 7}}
		m.messages = kept

		p.OnBlur()
		assert.True(t, cancelled)
		assert.False(t, m.consuming)

		gen := m.fetchGen
		require.NotNil(t, p.OnFocus())
		assert.Greater(t, m.fetchGen, gen, "the stream restarts in a new generation")
		assert.Equal(t, StatusConnecting, m.connectionStatus)
		assert.Equal(t, kept, m.messages, "messages already shown are kept")

		assert.Nil(t, p.OnFocus(), "a second activation does not restart it again")
	})

	t.Run("a live connect still in flight is dropped on blur", func(t *testing.T) {
		m := newLifecycleModel()
		m.consumeMode = ModeLive
		m.consumeFlags = consumeFlagsForMode(ModeLive)
		m.startLive()
		gen := m.fetchGen

		m.OnBlur()

		cancelled := false
		m.handlers.Handle(m, StartConsumingMsg{Cancel: func() { cancelled = true }, Topic: "test-topic", Gen: gen})
		assert.True(t, cancelled)
		assert.False(t, m.consuming)
	})

	t.Run("back to a non-live page does nothing", func(t *testing.T) {
		p := NewTopicPageModel(&MockDataSource{}, "test-topic", details)
		p.Init()
		p.OnFocus()
		gen := p.topicModel.fetchGen
		p.OnBlur()
		assert.Nil(t, p.OnFocus())
		assert.Equal(t, gen, p.topicModel.fetchGen)
	})
}

func TestDispose_CancelsEverything(t *testing.T) {
	t.Run("running stream", func(t *testing.T) {
		p := NewTopicPageModel(&MockDataSource{}, "test-topic", api.Topic{})
		m := p.topicModel
		m.consumeMode = ModeLive
		m.beginGeneration()
		cancelled := false
		m.handlers.Handle(m, StartConsumingMsg{Cancel: func() { cancelled = true }, Topic: "test-topic", Gen: m.fetchGen})

		p.Dispose()

		assert.True(t, cancelled)
		assert.False(t, m.consuming)
		assert.Nil(t, p.OnFocus(), "a disposed page never restarts")
	})

	t.Run("stream whose start message is never delivered", func(t *testing.T) {
		stopped := make(chan struct{})
		ds := &consumeFakeDS{MockDataSource: &MockDataSource{}, consume: func(ctx context.Context, _ api.MessageHandlerFunc, _ func(any)) error {
			<-ctx.Done()
			close(stopped)
			return nil
		}}
		m := NewModel(ds, "test-topic", api.Topic{})
		m.consumeMode = ModeLive
		cmd := m.startLive()
		require.NotNil(t, cmd)
		_ = cmd() // the StartConsumingMsg is dropped, as for an evicted page

		m.Dispose()

		select {
		case <-stopped:
		case <-time.After(2 * time.Second):
			t.Fatal("the consumer was not cancelled")
		}
	})

	t.Run("in-flight results are stale", func(t *testing.T) {
		m := newLifecycleModel()
		m.beginGeneration()
		gen := m.fetchGen
		ctx := m.generationContext()

		m.Dispose()

		assert.Error(t, ctx.Err(), "in-flight fetches are cancelled")
		assert.False(t, m.isCurrent("test-topic", gen))
	})
}

// A bounded seek run from Live mode keeps consumeMode at ModeLive but uses
// non-follow flags. Leaving and returning must not restart it as a live
// stream, which would end on its own and loop through fail and retry.
func TestRouterLifecycle_BoundedSeekInLiveModeIsNotResumed(t *testing.T) {
	p := NewTopicPageModel(&MockDataSource{}, "test-topic", api.Topic{NumPartitions: 1, MessageCount: 10})
	m := p.topicModel
	p.Init()
	p.OnFocus()
	m.consumeMode = ModeLive
	m.startForMode()
	require.True(t, m.tailsLive())

	flags, err := buildSeekFlags(string(api.SeekOldest), "", seekPageSize, nil)
	require.NoError(t, err)
	m.startForFlags(flags)
	require.Equal(t, ModeLive, m.consumeMode)
	kept := []api.Message{{Offset: 3}}
	m.handlers.Handle(m, MessagesFetchedMsg{Messages: kept, Topic: "test-topic", Gen: m.fetchGen})
	gen := m.fetchGen

	p.OnBlur()
	assert.Nil(t, p.OnFocus(), "a bounded seek is not restarted as a stream")
	assert.Equal(t, gen, m.fetchGen)
	assert.Nil(t, m.error)
	assert.Len(t, m.messages, 1, "the seek results stay on screen")

	t.Run("a live seek is resumed", func(t *testing.T) {
		live, err := buildSeekFlags(string(api.SeekLive), "", seekPageSize, nil)
		require.NoError(t, err)
		m.startForFlags(live)
		p.OnBlur()
		assert.NotNil(t, p.OnFocus())
	})
}

func TestPause_MarksRenderDirty(t *testing.T) {
	m := newLifecycleModel()
	m.consumeMode = ModeLive
	m.beginGeneration()
	m.consuming = true

	v := m.renderVersion
	m.TogglePause()
	require.True(t, m.paused)
	assert.Greater(t, m.renderVersion, v, "pausing must redraw the paused indicator")

	v = m.renderVersion
	m.handlers.Handle(m, MessageConsumedMsg{Message: api.Message{Offset: 1}, Topic: "test-topic", Gen: m.fetchGen})
	assert.Len(t, m.pendingWhilePaused, 1)
	assert.Empty(t, m.messages, "the table stays frozen")
	assert.Greater(t, m.renderVersion, v, "the buffered count must redraw")
}

func TestPause_DoesNotSurviveAFreshStart(t *testing.T) {
	tests := []struct {
		name  string
		start func(m *Model)
	}{
		{"mode switch", func(m *Model) { m.consumeMode = ModeLive; m.startForMode() }},
		{"seek", func(m *Model) {
			m.startForFlags(api.ConsumeFlags{Seek: api.SeekLive, Follow: true, OffsetFlag: "latest"})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newLifecycleModel()
			m.TogglePause() // paused in Newest mode, where it has no visible effect
			require.True(t, m.paused)

			tt.start(m)
			m.consuming = true
			m.handlers.Handle(m, MessageConsumedMsg{Message: api.Message{Offset: 1}, Topic: "test-topic", Gen: m.fetchGen})

			assert.False(t, m.paused)
			assert.Len(t, m.messages, 1, "the new stream's messages are shown")
			assert.Empty(t, m.pendingWhilePaused)
		})
	}
}

// Progress read from a superseded fetch's channel must not end or drive the
// current fetch's progress bar.
func TestProgress_FromSupersededFetchIsIgnored(t *testing.T) {
	m := newLifecycleModel()
	chA := components.NewProgressChannel(40)
	m.fetchProgressBar.StartListening(chA, 40)

	m.beginGeneration()
	chB := components.NewProgressChannel(10)
	m.handlers.Handle(m, StartFetchMsg{Topic: "test-topic", Gen: m.fetchGen, Total: 10, ProgressCh: chB, ResultCh: make(chan MessagesFetchedMsg)})
	require.True(t, m.fetchProgressBar.IsActive())

	chA <- components.ProgressMsg{Current: 40, Total: 40}
	chA <- components.ProgressMsg{Current: 40, Total: 40, Done: true}
	for range 2 {
		msg := components.ListenForProgress(chA)()
		_, cmd := m.handlers.Handle(m, msg)
		assert.Nil(t, cmd, "no listener is re-armed for a superseded fetch")
	}
	assert.True(t, m.fetchProgressBar.IsActive(), "fetch B is still loading")
	assert.Zero(t, m.fetchProgressBar.Current())

	chB <- components.ProgressMsg{Current: 4, Total: 10}
	m.handlers.Handle(m, components.ListenForProgress(chB)())
	assert.Equal(t, 4, m.fetchProgressBar.Current())
}
