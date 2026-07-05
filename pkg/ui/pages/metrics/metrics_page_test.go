package metrics

import (
	"context"
	"testing"
	"time"

	metricssvc "github.com/Benny93/kafui/pkg/metrics"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newCollected builds a page backed by a metrics collector driven for two
// cycles against the mock datasource (so rates are established).
func newCollected(t *testing.T) *Model {
	t.Helper()
	ds := &mock.KafkaDataSourceMock{}
	common := core.NewCommon(ds)
	col := metricssvc.New(ds, time.Second, nil)
	col.CollectAll(context.Background())
	col.CollectAll(context.Background())
	common.MetricsCollector = col

	m := NewModelWithCommon(common)
	m.SetDimensions(160, 40)
	m.refreshFromCache()
	return m
}

func TestPageIdentity(t *testing.T) {
	m := NewModelWithCommon(core.NewCommon(nil))
	assert.Equal(t, "metrics", m.GetID())
	assert.Equal(t, "Metrics", m.GetTitle())
}

func TestUnavailableWhenNoCollector(t *testing.T) {
	m := NewModelWithCommon(core.NewCommon(nil))
	m.SetDimensions(160, 40)
	out := m.renderContent(160, 40)
	assert.Contains(t, out, "unavailable")
	assert.False(t, m.loaded)
}

func TestCollectingStateBeforeFirstCycle(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	common := core.NewCommon(ds)
	common.MetricsCollector = metricssvc.New(ds, time.Second, nil) // seeded, not collected
	m := NewModelWithCommon(common)
	m.SetDimensions(160, 40)
	out := m.renderContent(160, 40)
	assert.Contains(t, out, "Collecting metrics")
	assert.False(t, m.loaded)
}

func TestRendersSummaryAndTopics(t *testing.T) {
	m := newCollected(t)
	require.True(t, m.loaded)
	out := m.renderContent(160, 40)

	assert.Contains(t, out, "Cluster:")
	assert.Contains(t, out, "Brokers:")
	assert.Contains(t, out, "Topics")
	assert.Contains(t, out, "Msgs/s:")
	assert.Contains(t, out, "Brokers") // per-broker section header
	// Topic rows populated from the mock datasource.
	assert.NotEmpty(t, m.metrics.Topics)
	assert.Greater(t, m.table.Height(), 0)
}

func TestByteRateHintWhenNoEndpoint(t *testing.T) {
	m := newCollected(t)
	out := m.renderContent(160, 40)
	assert.Contains(t, out, "metrics endpoint not configured")
	assert.False(t, m.endpointConfigured())
}

func TestUpdatedMsgRefreshes(t *testing.T) {
	m := newCollected(t)
	// Simulate the collector pushing an update; the page pulls from the cache.
	cmd := m.handle(metricssvc.MetricsUpdatedMsg{})
	assert.Nil(t, cmd)
	assert.True(t, m.loaded)
}

func TestHelpListsRefresh(t *testing.T) {
	m := NewModelWithCommon(core.NewCommon(nil))
	var keys []string
	for _, b := range m.GetHelp() {
		keys = append(keys, b.Help().Key)
	}
	assert.Contains(t, keys, "r")
}
