package cluster

import (
	"context"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	tea "github.com/charmbracelet/bubbletea"
)

// ClusterStatsUpdatedMsg is emitted after a collection cycle so subscribed pages
// can re-render from the collector cache without polling.
type ClusterStatsUpdatedMsg struct {
	Clusters []api.ClusterOverview
}

// CollectCmd runs a full collection cycle and reports the refreshed overviews.
func (c *Collector) CollectCmd() tea.Cmd {
	return func() tea.Msg {
		c.CollectAll(context.Background())
		return ClusterStatsUpdatedMsg{Clusters: c.ListClusters()}
	}
}

// TickCmd schedules the next collection cycle after the configured interval.
func (c *Collector) TickCmd() tea.Cmd {
	return tea.Tick(c.interval, func(time.Time) tea.Msg { return CollectTickMsg{} })
}

// CollectTickMsg triggers a periodic collection; the subscriber turns it into a
// CollectCmd (see pkg/ui wiring).
type CollectTickMsg struct{}

// RefreshCmd forces an immediate single-cluster refresh and reports the result.
func (c *Collector) RefreshCmd(name string) tea.Cmd {
	return func() tea.Msg {
		ov, err := c.RefreshCluster(context.Background(), name)
		if err != nil {
			return api.ClusterNotFoundError{Name: name}
		}
		return ClusterStatsUpdatedMsg{Clusters: []api.ClusterOverview{ov}}
	}
}
