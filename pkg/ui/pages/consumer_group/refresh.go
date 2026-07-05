package consumergroup

import (
	"time"

	"github.com/Benny93/kafui/pkg/ui/core"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// autoIntervals is the cycle of selectable auto-refresh intervals (off first).
var autoIntervals = []time.Duration{0, 10 * time.Second, 30 * time.Second, 60 * time.Second}

// cycleAutoRefresh advances to the next auto-refresh interval, persists the
// choice (session-scoped, via core.Common.Config), and (re)arms the tick loop.
func (m *Model) cycleAutoRefresh() tea.Cmd {
	cur := 0
	for i, d := range autoIntervals {
		if d == m.autoInterval {
			cur = i
			break
		}
	}
	m.autoInterval = autoIntervals[(cur+1)%len(autoIntervals)]
	if m.common != nil && m.common.Config != nil {
		m.common.Config.ConsumerGroupRefreshInterval = m.autoInterval
	}
	if m.autoInterval == 0 {
		// Turning auto-refresh off clears the trend baseline so arrows disappear.
		m.trendBaseline = nil
		m.rebuildTopicRows(m.detail)
		return core.NewNotification(core.StatusInfo, "Auto-refresh", "off")
	}
	return tea.Batch(
		core.NewNotification(core.StatusInfo, "Auto-refresh", m.autoInterval.String()),
		m.scheduleTick(),
	)
}

// scheduleTick arms a single auto-refresh tick for the current interval.
func (m *Model) scheduleTick() tea.Cmd {
	interval := m.autoInterval
	id := m.groupID
	if interval <= 0 {
		return nil
	}
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return autoRefreshTickMsg{groupID: id, interval: interval}
	})
}

// handleAutoTick captures the current lags as the trend baseline, re-fetches the
// detail, and re-arms the next tick (ignoring stale ticks after a change).
func (m *Model) handleAutoTick(v autoRefreshTickMsg) tea.Cmd {
	if v.groupID != m.groupID || v.interval != m.autoInterval || m.autoInterval == 0 {
		return nil // stale tick (interval changed or turned off)
	}
	m.captureBaseline()
	return tea.Batch(m.loadDetail(), m.scheduleTick())
}

// captureBaseline snapshots the current per-topic aggregate lags so the next
// reading can render rise/fall arrows.
func (m *Model) captureBaseline() {
	base := make(map[string]int64, len(m.topicRows))
	for _, tr := range m.topicRows {
		if tr.aggLag != nil {
			base[tr.topic] = *tr.aggLag
		}
	}
	m.trendBaseline = base
}

// trendArrow renders ↑ (rising) / ↓ (falling) for a topic vs. its baseline.
// Empty when there is no baseline or no change.
func (m *Model) trendArrow(tr topicRow) string {
	if tr.prevAggLag == nil || tr.aggLag == nil {
		return ""
	}
	up := lipgloss.NewStyle().Foreground(stylesPkg.Error)
	down := lipgloss.NewStyle().Foreground(stylesPkg.Success)
	switch {
	case *tr.aggLag > *tr.prevAggLag:
		return up.Render("↑")
	case *tr.aggLag < *tr.prevAggLag:
		return down.Render("↓")
	default:
		return ""
	}
}
