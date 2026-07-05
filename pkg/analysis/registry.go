package analysis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Benny93/kafui/pkg/api"
)

// defaultSampleLimit bounds how many messages a scan reads. The scan is a sample
// (lazy but correct); the progress percentage is relative to the captured end
// offsets so a partial sample still reports meaningful progress.
//
// ponytail: a full earliest→end scan is possible (Follow=false already stops at
// the high-watermark) but can be very expensive on large topics. A fixed sample
// keeps `make run-mock` and real usage responsive; raise the cap when exactness
// matters.
const defaultSampleLimit = 50_000

// ConsumeFunc matches api.KafkaDataSource.ConsumeTopic and is the only coupling
// between the engine and the datasource.
type ConsumeFunc func(ctx context.Context, topic string, flags api.ConsumeFlags, handle api.MessageHandlerFunc, onError func(err any)) error

// Registry owns per-topic analysis runs (single result retained per topic).
type Registry struct {
	mu          sync.Mutex
	consume     ConsumeFunc
	sampleLimit int64
	runs        map[string]*run
}

type run struct {
	mu       sync.Mutex
	agg      *Aggregator
	state    api.AnalysisState
	progress api.AnalysisProgress
	result   *api.TopicAnalysisResult
	errMsg   string
	errAt    time.Time
	cancel   context.CancelFunc
}

// NewRegistry builds a Registry driven by the given ConsumeFunc.
func NewRegistry(consume ConsumeFunc) *Registry {
	return &Registry{
		consume:     consume,
		sampleLimit: defaultSampleLimit,
		runs:        make(map[string]*run),
	}
}

// Start begins a background analysis of topic. totalOffsets is the message total
// used for the progress percentage (0 when unknown). It returns an
// AnalysisAlreadyRunningError when one is already in progress.
func (reg *Registry) Start(ctx context.Context, topic string, totalOffsets int64) error {
	reg.mu.Lock()
	if r, ok := reg.runs[topic]; ok {
		r.mu.Lock()
		running := r.state == api.AnalysisRunning
		r.mu.Unlock()
		if running {
			reg.mu.Unlock()
			return api.AnalysisAlreadyRunningError{TopicName: topic}
		}
	}

	runCtx, cancel := context.WithCancel(ctx)
	r := &run{
		agg:    NewAggregator(topic),
		state:  api.AnalysisRunning,
		cancel: cancel,
		progress: api.AnalysisProgress{
			StartTime:    time.Now(),
			TotalOffsets: totalOffsets,
		},
	}
	reg.runs[topic] = r
	reg.mu.Unlock()

	go reg.scan(runCtx, topic, r)
	return nil
}

// scan drives the consume loop, feeding the aggregator and updating progress.
func (reg *Registry) scan(ctx context.Context, topic string, r *run) {
	flags := api.ConsumeFlags{
		Follow:        false,
		OffsetFlag:    "oldest",
		LimitMessages: reg.sampleLimit,
	}

	var consumeErr error
	handle := func(msg api.Message) {
		r.mu.Lock()
		r.agg.Add(msg)
		r.progress.MessagesScanned++
		r.progress.ProcessedOffsets++
		r.progress.BytesScanned += messageBytes(msg)
		r.mu.Unlock()
	}
	onError := func(err any) {
		if consumeErr == nil {
			consumeErr = fmt.Errorf("%v", err)
		}
	}

	err := reg.consume(ctx, topic, flags, handle, onError)

	// Cancellation records nothing: drop the run entirely.
	if ctx.Err() != nil {
		reg.mu.Lock()
		delete(reg.runs, topic)
		reg.mu.Unlock()
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if err != nil {
		consumeErr = err
	}
	if consumeErr != nil {
		r.state = api.AnalysisFailed
		r.errMsg = consumeErr.Error()
		r.errAt = time.Now()
		return
	}
	result := r.agg.Result()
	r.result = &result
	r.state = api.AnalysisCompleted
}

// messageBytes estimates the on-wire size of a message's key+value.
func messageBytes(msg api.Message) int64 {
	k, _ := fieldSize(msg.Key, msg.RawKey)
	v, _ := fieldSize(msg.Value, msg.RawValue)
	return k + v
}

// Get returns a snapshot of the latest analysis for topic, or (nil, nil) when
// none has ever been started.
func (reg *Registry) Get(topic string) (*api.TopicAnalysis, error) {
	reg.mu.Lock()
	r, ok := reg.runs[topic]
	reg.mu.Unlock()
	if !ok {
		return nil, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	snap := &api.TopicAnalysis{
		Topic:    topic,
		State:    r.state,
		Progress: r.progress,
		Err:      r.errMsg,
		ErrAt:    r.errAt,
	}
	if r.result != nil {
		res := *r.result
		snap.Result = &res
	}
	return snap, nil
}

// Cancel stops a running analysis and discards its run (records nothing).
func (reg *Registry) Cancel(topic string) error {
	reg.mu.Lock()
	r, ok := reg.runs[topic]
	reg.mu.Unlock()
	if !ok {
		return nil
	}
	r.mu.Lock()
	cancel := r.cancel
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}
