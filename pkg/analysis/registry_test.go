package analysis

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// waitForState polls until the analysis reaches want or the deadline elapses.
func waitForState(t *testing.T, reg *Registry, topic string, want api.AnalysisState) *api.TopicAnalysis {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		a, _ := reg.Get(topic)
		if a != nil && a.State == want {
			return a
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for state %q", want)
	return nil
}

func TestRegistry_CompletedScan(t *testing.T) {
	consume := func(ctx context.Context, topic string, flags api.ConsumeFlags, handle api.MessageHandlerFunc, onError func(err any)) error {
		for i := 0; i < 5; i++ {
			handle(api.Message{Partition: 0, Offset: int64(i), Key: "k", Value: "v"})
		}
		return nil
	}
	reg := NewRegistry(consume)
	require.NoError(t, reg.Start(context.Background(), "t", 5))

	a := waitForState(t, reg, "t", api.AnalysisCompleted)
	require.NotNil(t, a.Result)
	assert.Equal(t, int64(5), a.Result.MessageCount)
	assert.Equal(t, int64(5), a.Progress.MessagesScanned)
}

func TestRegistry_DuplicateStartRejected(t *testing.T) {
	release := make(chan struct{})
	consume := func(ctx context.Context, topic string, flags api.ConsumeFlags, handle api.MessageHandlerFunc, onError func(err any)) error {
		<-release // block so the run stays in the running state
		return nil
	}
	reg := NewRegistry(consume)
	require.NoError(t, reg.Start(context.Background(), "t", 10))

	err := reg.Start(context.Background(), "t", 10)
	var e api.AnalysisAlreadyRunningError
	assert.True(t, errors.As(err, &e))
	close(release)
}

func TestRegistry_CancelRecordsNothing(t *testing.T) {
	started := make(chan struct{})
	consume := func(ctx context.Context, topic string, flags api.ConsumeFlags, handle api.MessageHandlerFunc, onError func(err any)) error {
		close(started)
		<-ctx.Done() // block until cancelled
		return ctx.Err()
	}
	reg := NewRegistry(consume)
	require.NoError(t, reg.Start(context.Background(), "t", 10))
	<-started

	require.NoError(t, reg.Cancel("t"))

	// After cancellation the run is dropped: Get returns nil (records nothing).
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		a, _ := reg.Get("t")
		if a == nil {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("cancelled analysis should have been dropped")
}

func TestRegistry_FailureCaptured(t *testing.T) {
	consume := func(ctx context.Context, topic string, flags api.ConsumeFlags, handle api.MessageHandlerFunc, onError func(err any)) error {
		return errors.New("broker exploded")
	}
	reg := NewRegistry(consume)
	require.NoError(t, reg.Start(context.Background(), "t", 10))

	a := waitForState(t, reg, "t", api.AnalysisFailed)
	assert.Contains(t, a.Err, "broker exploded")
	assert.False(t, a.ErrAt.IsZero(), "failure timestamp captured")
	assert.Nil(t, a.Result)
}

func TestRegistry_ProgressMonotonic(t *testing.T) {
	var mu sync.Mutex
	gate := make(chan struct{})
	consume := func(ctx context.Context, topic string, flags api.ConsumeFlags, handle api.MessageHandlerFunc, onError func(err any)) error {
		for i := 0; i < 20; i++ {
			handle(api.Message{Offset: int64(i), Value: "v"})
		}
		<-gate
		return nil
	}
	reg := NewRegistry(consume)
	require.NoError(t, reg.Start(context.Background(), "t", 20))

	var last int64
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		a, _ := reg.Get("t")
		mu.Unlock()
		if a != nil {
			assert.GreaterOrEqual(t, a.Progress.MessagesScanned, last)
			last = a.Progress.MessagesScanned
			if last == 20 {
				break
			}
		}
		time.Sleep(2 * time.Millisecond)
	}
	assert.Equal(t, int64(20), last)
	close(gate)
}
