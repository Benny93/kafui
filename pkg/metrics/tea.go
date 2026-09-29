package metrics

import (
	"context"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	tea "github.com/charmbracelet/bubbletea"
)

// MetricsUpdatedMsg is emitted after a collection cycle so the metrics page can
// re-render from the collector cache without polling.
type MetricsUpdatedMsg struct {
	Active api.ClusterMetrics
}

// CollectTickMsg triggers a periodic collection; the subscriber turns it into a
// CollectCmd (see pkg/ui wiring), mirroring the cluster collector.
type CollectTickMsg struct{}

// CollectCmd runs a full collection cycle and reports the refreshed active
// snapshot. While a cycle is already running it does nothing and reports
// nothing: the running cycle's MetricsUpdatedMsg serves both callers, and one
// slow cycle can never stack another on top of it against the same brokers.
func (c *Collector) CollectCmd() tea.Cmd {
	return func() tea.Msg {
		if !c.collecting.CompareAndSwap(false, true) {
			return nil
		}
		defer c.collecting.Store(false)
		c.CollectAll(context.Background())
		active, _ := c.Active()
		return MetricsUpdatedMsg{Active: active}
	}
}

// TickCmd schedules the next collection cycle after the configured interval.
// Callers arm it once a cycle has reported back, not alongside the cycle, so
// the interval is the pause between cycles rather than their start rate.
func (c *Collector) TickCmd() tea.Cmd {
	return tea.Tick(c.interval, func(time.Time) tea.Msg { return CollectTickMsg{} })
}
