package metrics

import (
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// blockingDS holds GetTopics until release is closed.
type blockingDS struct {
	*fakeDS
	entered chan struct{}
	release chan struct{}
}

func (b *blockingDS) GetTopics() (map[string]api.Topic, error) {
	b.entered <- struct{}{}
	<-b.release
	return b.fakeDS.GetTopics()
}

// A second CollectCmd while a cycle runs must not start another cycle against
// the same brokers (PERF-2); the running cycle reports for both.
func TestCollectCmdSkipsWhileACycleRuns(t *testing.T) {
	ds := &blockingDS{fakeDS: newFake(), entered: make(chan struct{}, 2), release: make(chan struct{})}
	c := New(ds, time.Second, nil)

	first := make(chan any, 1)
	go func() { first <- c.CollectCmd()() }()
	<-ds.entered

	assert.Nil(t, c.CollectCmd()(), "an overlapping cycle must report nothing")
	assert.Empty(t, ds.entered, "an overlapping cycle must not reach the data source")

	close(ds.release)
	msg := <-first
	_, ok := msg.(MetricsUpdatedMsg)
	require.True(t, ok, "the running cycle reports its result, got %T", msg)

	// Once it finished, the next cycle runs normally.
	_, ok = c.CollectCmd()().(MetricsUpdatedMsg)
	assert.True(t, ok)
}
