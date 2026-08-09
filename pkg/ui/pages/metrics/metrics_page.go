// Package metrics implements the metrics & monitoring page (page ID "metrics").
// It renders the active cluster's collected metrics from the background metrics
// collector at common.MetricsCollector: a summary section (broker/topic/
// partition counts, message-in rate with a sparkline + min/max/avg, byte rates),
// a per-topic metrics table, and a per-broker section. It re-renders on
// metrics.MetricsUpdatedMsg.
//
// Message-in rates come from offset deltas and are always available. Byte rates
// are shown only when a metrics endpoint is configured; otherwise a hint is
// shown (see plan MM-4/MM-5, currently stubbed).
//
// The page is NOT self-registering: the router registers it under the ID
// "metrics" (see pkg/ui/router/router.go).
package metrics

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	metricssvc "github.com/Benny93/kafui/pkg/metrics"
	"github.com/Benny93/kafui/pkg/metrics/promquery"
	"github.com/Benny93/kafui/pkg/ui/components"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const dash = "–"

// Model is the metrics & monitoring page.
type Model struct {
	common *core.Common

	table      table.Model
	spark      *components.Sparkline
	graphSpark *components.Sparkline
	dimensions core.Dimensions
	loaded     bool
	metrics    api.ClusterMetrics
	history    api.TimeSeries

	picker      *graphPicker
	reusableApp *templateui.ReusableApp
}

// pageScope is the key scope this page resolves against. The page used to
// carry its own key map, which bound `g` to "graphs" — `g` is the go-to-first
// alias — and `p` to "set params" while `p` means pause elsewhere. Both are
// actions-menu entries now.
func pageScope() keys.Scope { return keys.ScopeListContent }

// NewModelWithCommon builds the metrics page. The intended router page ID is
// "metrics".
func NewModelWithCommon(common *core.Common) *Model {
	m := &Model{common: common}

	m.table = table.New(
		table.WithColumns(columns()),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	m.spark = components.NewSparkline(lipgloss.NewStyle().Foreground(stylesPkg.Accent))
	m.spark.SetDimensions(40, 1)
	m.graphSpark = components.NewSparkline(lipgloss.NewStyle().Foreground(stylesPkg.Accent))
	m.graphSpark.SetDimensions(60, 1)
	m.picker = newGraphPicker()

	// Populate immediately from any already-collected snapshot.
	m.refreshFromCache()

	config := &providers.AppConfig{
		ContentProvider:      &contentProvider{model: m},
		ShowSidebarByDefault: false,
	}
	m.reusableApp = templateui.NewReusableApp(config)
	m.reusableApp.SetKeyMap(keys.Hints(pageScope()))
	return m
}

func columns() []table.Column {
	return []table.Column{
		{Title: "Topic", Width: 30},
		{Title: "Partitions", Width: 12},
		{Title: "Messages", Width: 14},
		{Title: "Msgs/s", Width: 10},
	}
}

// contentProvider bridges the template content area to the page model.
type contentProvider struct{ model *Model }

func (p *contentProvider) RenderContent(width, height int) string {
	return p.model.renderContent(width, height)
}
func (p *contentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd { return p.model.handle(msg) }
func (p *contentProvider) InitContent() tea.Cmd                    { return nil }
func (p *contentProvider) IsInputMode() bool                       { return false }
func (p *contentProvider) GetContentSize(width int) int            { return len(p.model.metrics.Topics) + 12 }

// --- core.Page ---

func (m *Model) Init() tea.Cmd { return m.reusableApp.Init() }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.reusableApp.Update(msg)
	if app, ok := updated.(*templateui.ReusableApp); ok {
		m.reusableApp = app
	}
	return m, cmd
}

func (m *Model) View() string { return m.reusableApp.View() }

func (m *Model) SetDimensions(width, height int) {
	m.dimensions = core.Dimensions{Width: width, Height: height}
	m.table.SetWidth(width)
	if h := height - 14; h > 1 {
		m.table.SetHeight(h)
	}
	m.reusableApp.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

func (m *Model) GetID() string    { return "metrics" }
func (m *Model) GetTitle() string { return "Metrics" }

func (m *Model) GetHelp() []key.Binding { return keys.Help(pageScope()) }

func (m *Model) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) { return m, nil }

// OnFocus refreshes from the cache so the page shows current data on entry.
func (m *Model) OnFocus() tea.Cmd {
	m.refreshFromCache()
	if m.common != nil && m.common.MetricsCollector != nil {
		return m.common.MetricsCollector.CollectCmd()
	}
	return nil
}
func (m *Model) OnBlur() tea.Cmd { return nil }

// --- message handling ---

func (m *Model) handle(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case metricssvc.MetricsUpdatedMsg:
		m.refreshFromCache()
		return nil
	case graphResultMsg:
		m.picker.lastID = msg.id
		m.picker.result = msg.result
		m.picker.runErr = msg.err
		return nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return cmd
}

// handleKey routes key presses, giving the graph picker priority when it is
// active so table navigation does not steal its keys.
func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	m.picker.setStorage(m.storageConfigured())

	// Parameter entry captures all keys until submit/cancel.
	if m.picker.prompting {
		switch msg.String() {
		case "enter":
			if m.picker.submitParam() {
				return m.picker.runCmd(m.buildGraphClient(), m.activeCluster())
			}
			return nil
		case "esc":
			m.picker.cancelParams()
			return nil
		}
		var cmd tea.Cmd
		m.picker.input, cmd = m.picker.input.Update(msg)
		return cmd
	}

	action, bound := keys.Default.Resolve(pageScope(), msg.String())
	if !bound {
		return nil
	}

	if !m.picker.visible || !m.picker.hasGraphs() {
		switch action {
		case keys.ActionRefresh:
			if m.common != nil && m.common.MetricsCollector != nil {
				return m.common.MetricsCollector.CollectCmd()
			}
			return nil
		}
	}

	if m.picker.visible && m.picker.hasGraphs() {
		switch action {
		case keys.ActionUp:
			m.picker.moveCursor(-1)
			return nil
		case keys.ActionDown:
			m.picker.moveCursor(1)
			return nil
		case keys.ActionActivate:
			if gr, ok := m.picker.selected(); ok {
				if m.picker.beginParams(gr) {
					return m.picker.runCmd(m.buildGraphClient(), m.activeCluster())
				}
			}
			return nil
		case keys.ActionCancel:
			m.picker.visible = false
			return nil
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return cmd
}

// activeCluster returns the current cluster name.
func (m *Model) activeCluster() string {
	if m.common == nil || m.common.DataSource == nil {
		return ""
	}
	return m.common.DataSource.GetContext()
}

// activeMetricsSettings returns the active cluster's metrics settings, if any.
func (m *Model) storageConfigured() bool {
	if m.common == nil || m.common.AppConfig == nil || m.common.DataSource == nil {
		return false
	}
	ext, ok := m.common.AppConfig.Clusters[m.common.DataSource.GetContext()]
	return ok && len(ext.MetricsSettings().TimeSeriesURLs) > 0
}

// buildGraphClient builds a Prometheus query client from the active cluster's
// TimeSeriesURLs (nil when none configured, yielding MetricsNotConfiguredError).
func (m *Model) buildGraphClient() *promquery.Client {
	if m.common == nil || m.common.AppConfig == nil || m.common.DataSource == nil {
		return nil
	}
	ext, ok := m.common.AppConfig.Clusters[m.common.DataSource.GetContext()]
	if !ok {
		return nil
	}
	s := ext.MetricsSettings()
	c, err := promquery.New(s.TimeSeriesURLs, s.TLSCAPath)
	if err != nil {
		return nil
	}
	return c
}

// refreshFromCache reads the active cluster's cached metrics + history.
func (m *Model) refreshFromCache() {
	if m.common == nil || m.common.MetricsCollector == nil {
		return
	}
	cm, ok := m.common.MetricsCollector.Active()
	if !ok {
		return
	}
	m.metrics = cm
	m.history = m.common.MetricsCollector.ActiveMessagesInHistory()
	m.loaded = !cm.CollectedAt.IsZero()
	if m.picker != nil {
		m.picker.setStorage(m.storageConfigured())
	}
	m.rebuildRows()
}

func (m *Model) rebuildRows() {
	rows := make([]table.Row, 0, len(m.metrics.Topics))
	for _, t := range m.metrics.Topics {
		rows = append(rows, table.Row{
			t.Name,
			fmt.Sprintf("%d", t.PartitionCount),
			fmt.Sprintf("%d", t.MessageCount),
			components.FormatRate(t.MessagesInPerSec),
		})
	}
	m.table.SetRows(rows)
}

// --- rendering ---

func (m *Model) renderContent(width, height int) string {
	m.table.SetWidth(width - 2) // -2 leaves room for the FrameTable border
	s := m.common.Styles
	if m.common.MetricsCollector == nil {
		return s.Muted.Render("Metrics collection is unavailable.")
	}
	if !m.loaded {
		return s.Muted.Render("Collecting metrics… (message-in rates appear after the second cycle)")
	}

	var b strings.Builder
	cm := m.metrics
	b.WriteString(s.Header.Render(fmt.Sprintf("Cluster: %s", cm.Cluster)))
	b.WriteString("  ")
	b.WriteString(s.Muted.Render("last refresh " + cm.CollectedAt.Format("15:04:05")))
	b.WriteString("\n\n")

	// Summary counters.
	b.WriteString(fmt.Sprintf("Brokers: %d   Topics: %d   Partitions: %d   Messages: %d\n",
		cm.BrokerCount, cm.TopicCount, cm.PartitionCount, cm.MessageCount))

	// Message-in rate + sparkline + summary.
	m.spark.SetData(m.history.Values())
	sum := m.history.Summary()
	b.WriteString(fmt.Sprintf("Msgs/s: %s  ", components.FormatRate(cm.MessagesInPerSec)))
	b.WriteString(m.spark.View())
	if sum.OK {
		b.WriteString("  " + s.Muted.Render(fmt.Sprintf("min %.1f  max %.1f  avg %.1f", sum.Min, sum.Max, sum.Avg)))
	}
	b.WriteString("\n")

	// Byte rates: shown when known, hinted when no endpoint configured.
	if cm.BytesInPerSec >= 0 || cm.BytesOutPerSec >= 0 {
		b.WriteString(fmt.Sprintf("Bytes In/s: %s   Bytes Out/s: %s\n",
			components.FormatBytesPerSec(cm.BytesInPerSec), components.FormatBytesPerSec(cm.BytesOutPerSec)))
	} else if !m.endpointConfigured() {
		b.WriteString(s.Muted.Render("Bytes In/Out: metrics endpoint not configured"))
		b.WriteString("\n")
	} else {
		b.WriteString(s.Muted.Render("Bytes In/Out: " + dash))
		b.WriteString("\n")
	}

	// Topic metrics table.
	b.WriteString("\n")
	b.WriteString(s.Header.Render("Topics"))
	b.WriteString("\n")
	if len(cm.Topics) == 0 {
		b.WriteString(s.Muted.Render("No topics"))
	} else {
		b.WriteString(stylesPkg.FrameTable(m.table.View()))
	}

	// Broker metrics section.
	if len(cm.Brokers) > 0 {
		b.WriteString("\n\n")
		b.WriteString(s.Header.Render("Brokers"))
		b.WriteString("\n")
		for _, br := range cm.Brokers {
			b.WriteString(fmt.Sprintf("  broker %d   leaders %d   replicas %d   segments %s\n",
				br.ID, br.LeaderCount, br.ReplicaCount, humanBytes(br.SegmentSize)))
		}
	}

	// Graph picker (MM-15): shown when a time-series backend is configured.
	if m.picker != nil {
		m.picker.setStorage(m.storageConfigured())
		if m.picker.hasGraphs() && (m.picker.visible || m.picker.prompting) {
			m.picker.render(&b,
				func(t string) string { return s.Muted.Render(t) },
				func(t string) string { return s.Header.Render(t) },
				m.graphSpark)
		}
	}

	return b.String()
}

// endpointConfigured reports whether the active cluster has a metrics endpoint
// configured (enabling byte-rate scraping, currently stubbed).
func (m *Model) endpointConfigured() bool {
	if m.common == nil || m.common.AppConfig == nil || m.common.DataSource == nil {
		return false
	}
	ext, ok := m.common.AppConfig.Clusters[m.common.DataSource.GetContext()]
	return ok && ext.MetricsSettings().Endpoint != ""
}

// humanBytes formats a byte count with IEC units (for segment sizes).
func humanBytes(v int64) string {
	const unit = 1024.0
	f := float64(v)
	if f < unit {
		return fmt.Sprintf("%d B", v)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	val, i := f, -1
	for val >= unit && i < len(units)-1 {
		val /= unit
		i++
	}
	return fmt.Sprintf("%.1f %s", val, units[i])
}
