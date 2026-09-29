package ui

import (
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/metrics"
	"github.com/stretchr/testify/assert"
)

// Metrics collection polls every topic's offsets, so it must run only while
// someone reads it, and the next tick is armed only once a cycle reported
// back, so slow cycles never overlap (PERF-2).
func TestMetricsCollectOnlyWhileRead(t *testing.T) {
	tests := []struct {
		name        string
		exposed     bool
		onMetrics   bool
		wantCollect bool
	}{
		{"main page, no exposition", false, false, false},
		{"metrics page showing", false, true, true},
		{"exposition endpoint serving", true, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newShell(t)
			m.common.MetricsCollector = metrics.New(m.common.DataSource, time.Hour, nil)
			m.metricsExposed = tt.exposed
			if tt.onMetrics {
				m.Router.NavigateTo("metrics", nil)
			}

			m.Update(metrics.MetricsUpdatedMsg{})
			assert.Equal(t, tt.wantCollect, m.metricsTickArmed, "a finished cycle arms the next tick only when read")

			_, cmd := m.Update(metrics.CollectTickMsg{})
			assert.Equal(t, tt.wantCollect, cmd != nil, "a tick collects only when read")
			assert.False(t, m.metricsTickArmed, "the next tick waits for this cycle to report")
		})
	}
}

func TestMetricsTickChainStaysSingle(t *testing.T) {
	m := newShell(t)
	m.common.MetricsCollector = metrics.New(m.common.DataSource, time.Hour, nil)
	m.Router.NavigateTo("metrics", nil)

	_, first := m.Update(metrics.MetricsUpdatedMsg{})
	assert.NotNil(t, first)
	assert.True(t, m.metricsTickArmed)

	// A second report (the page's own refresh) must not start a second chain.
	m.Update(metrics.MetricsUpdatedMsg{})
	assert.True(t, m.metricsTickArmed)
}
