// Package clusters implements the cluster overview dashboard page (page ID
// "clusters"). It renders a full-width table of every configured cluster,
// health-summary counters, an offline-only filter, and per-cluster actions
// (open / refresh / validate). Data comes from the background collector at
// common.Collector; the page re-renders on cluster.ClusterStatsUpdatedMsg.
//
// The page is NOT self-registering: the router registers it under the ID
// "clusters" (see pkg/ui/router/router.go).
package clusters

import (
	"context"
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/cluster"
	"github.com/Benny93/kafui/pkg/ui/core"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// dash marks an unknown (negative) rate.
const dash = "–"

// validationDoneMsg carries the result of a ValidateClusterConnection run.
type validationDoneMsg struct {
	cluster string
	results []api.ValidationResult
	err     error
}

// Model is the cluster overview dashboard page.
type Model struct {
	common *core.Common

	table       table.Model
	dimensions  core.Dimensions
	loaded      bool // set once the first ClusterStatsUpdatedMsg arrives
	offlineOnly bool
	clusters    []api.ClusterOverview

	validating       bool
	validationTarget string
	validation       []api.ValidationResult
	validationErr    error

	keys        pageKeys
	reusableApp *templateui.ReusableApp
}

type pageKeys struct {
	Open     key.Binding
	Offline  key.Binding
	Refresh  key.Binding
	Validate key.Binding
}

func defaultKeys() pageKeys {
	return pageKeys{
		Open:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "switch context")),
		Offline:  key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "offline only")),
		Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Validate: key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "validate")),
	}
}

// NewModelWithCommon builds the clusters dashboard page. The intended router
// page ID is "clusters".
func NewModelWithCommon(common *core.Common) *Model {
	m := &Model{
		common: common,
		keys:   defaultKeys(),
	}

	m.table = table.New(
		table.WithColumns(columns()),
		table.WithFocused(true),
		table.WithHeight(12),
	)
	m.rebuildRows()

	config := &providers.AppConfig{
		ContentProvider:      &contentProvider{model: m},
		ShowSidebarByDefault: false,
	}
	m.reusableApp = templateui.NewReusableApp(config)
	m.reusableApp.SetKeyMap(helpKeyMap{keys: m.keys})

	return m
}

func columns() []table.Column {
	return []table.Column{
		{Title: "Name", Width: 22},
		{Title: "Status", Width: 12},
		{Title: "Version", Width: 10},
		{Title: "Brokers", Width: 8},
		{Title: "Online Partitions", Width: 18},
		{Title: "Topics", Width: 8},
		{Title: "Msgs/s", Width: 10},
		{Title: "Bytes In/s", Width: 12},
		{Title: "Bytes Out/s", Width: 12},
		{Title: "Access", Width: 10},
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
func (p *contentProvider) GetContentSize(width int) int            { return len(p.model.clusters) + 6 }

// helpKeyMap adapts the page bindings to the footer help.KeyMap interface.
type helpKeyMap struct{ keys pageKeys }

func (h helpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{h.keys.Open, h.keys.Offline, h.keys.Refresh, h.keys.Validate}
}
func (h helpKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{h.ShortHelp()} }

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
	if h := height - 10; h > 1 {
		m.table.SetHeight(h)
	}
	m.reusableApp.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

func (m *Model) GetID() string    { return "clusters" }
func (m *Model) GetTitle() string { return "Clusters" }

func (m *Model) GetHelp() []key.Binding {
	return []key.Binding{m.keys.Open, m.keys.Offline, m.keys.Refresh, m.keys.Validate}
}

func (m *Model) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) { return m, nil }

// OnFocus seeds the table from the collector's cache immediately (the initial
// collection cycle may have completed before this page was opened, so waiting
// for the next ClusterStatsUpdatedMsg would leave it on "Loading…"), then kicks
// a fresh collection to refresh.
func (m *Model) OnFocus() tea.Cmd {
	if m.common != nil && m.common.Collector != nil {
		if cached := m.common.Collector.ListClusters(); len(cached) > 0 {
			m.clusters = cached
			m.loaded = true
			m.rebuildRows()
		}
		return m.common.Collector.CollectCmd()
	}
	return nil
}
func (m *Model) OnBlur() tea.Cmd { return nil }

// --- message handling ---

// handle processes messages routed through the template content area. It never
// consumes esc so the router's back handling keeps working.
func (m *Model) handle(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case cluster.ClusterStatsUpdatedMsg:
		m.applyUpdate(msg)
		return nil
	case validationDoneMsg:
		m.validating = false
		m.validationTarget = msg.cluster
		m.validation = msg.results
		m.validationErr = msg.err
		return nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	// Forward everything else (mouse, etc.) to the table for navigation.
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return cmd
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Offline):
		m.offlineOnly = !m.offlineOnly
		m.table.SetCursor(0)
		m.rebuildRows()
		return nil
	case key.Matches(msg, m.keys.Open):
		return m.openSelected()
	case key.Matches(msg, m.keys.Refresh):
		return m.refreshSelected()
	case key.Matches(msg, m.keys.Validate):
		return m.validateSelected()
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return cmd
}

// applyUpdate refreshes the cached overviews. When a collector is present the
// full cache is the source of truth (a single-cluster RefreshCmd message only
// carries one entry); otherwise the message payload is used directly.
func (m *Model) applyUpdate(msg cluster.ClusterStatsUpdatedMsg) {
	m.loaded = true
	if m.common != nil && m.common.Collector != nil {
		m.clusters = m.common.Collector.ListClusters()
	} else {
		m.clusters = msg.Clusters
	}
	m.rebuildRows()
}

func (m *Model) openSelected() tea.Cmd {
	c, ok := m.selected()
	if !ok || m.common == nil || m.common.DataSource == nil {
		return nil
	}
	// SetContext is in-memory only; it never writes ~/.kaf/config.
	if err := m.common.DataSource.SetContext(c.Name); err != nil {
		return core.NotifyError("Switch context failed", err)
	}
	return core.NewPageChangeMsg("main", nil)
}

func (m *Model) refreshSelected() tea.Cmd {
	c, ok := m.selected()
	if !ok || m.common == nil || m.common.Collector == nil {
		return nil
	}
	return tea.Batch(
		core.NewNotification(core.StatusInfo, "Refreshing", c.Name),
		m.common.Collector.RefreshCmd(c.Name),
	)
}

func (m *Model) validateSelected() tea.Cmd {
	c, ok := m.selected()
	if !ok || m.common == nil || m.common.DataSource == nil {
		return nil
	}
	name := c.Name
	m.validating = true
	m.validationTarget = name
	m.validation = nil
	m.validationErr = nil
	ds := m.common.DataSource
	return func() tea.Msg {
		res, err := ds.ValidateClusterConnection(context.Background(), name)
		return validationDoneMsg{cluster: name, results: res, err: err}
	}
}

// --- rendering ---

func (m *Model) renderContent(width, height int) string {
	m.table.SetWidth(width - 2) // -2 leaves room for the FrameTable border
	var b strings.Builder
	b.WriteString(m.header())
	b.WriteString("\n\n")

	if m.loaded && len(m.clusters) == 0 {
		b.WriteString(m.common.Styles.Muted.Render("No clusters found"))
		return b.String()
	}

	b.WriteString(stylesPkg.FrameTable(m.table.View()))
	b.WriteString("\n")
	if d := m.detail(); d != "" {
		b.WriteString("\n")
		b.WriteString(d)
	}
	return b.String()
}

func (m *Model) header() string {
	online, offline, initializing := statusCounts(m.clusters)
	s := m.common.Styles.StatusStyle
	parts := []string{
		"Online: " + s.Success.Render(fmt.Sprintf("%d", online)),
		"Offline: " + s.Error.Render(fmt.Sprintf("%d", offline)),
		"Initializing: " + s.Warning.Render(fmt.Sprintf("%d", initializing)),
	}
	line := strings.Join(parts, "  ")
	if m.offlineOnly {
		line += "  " + s.Warning.Render("[offline only]")
	}
	return line
}

// detail renders the footer area: the selected offline cluster's last error and
// any validation results.
func (m *Model) detail() string {
	var b strings.Builder
	s := m.common.Styles.StatusStyle
	if c, ok := m.selected(); ok && c.Status == api.ClusterOffline && c.LastError != "" {
		b.WriteString(s.Error.Render("Error: " + c.LastError))
		b.WriteString("\n")
	}
	if m.validating {
		b.WriteString(s.Info.Render("Validating " + m.validationTarget + "…"))
		b.WriteString("\n")
	} else if m.validationTarget != "" && (len(m.validation) > 0 || m.validationErr != nil) {
		b.WriteString(m.common.Styles.Muted.Render("Validation: " + m.validationTarget))
		b.WriteString("\n")
		if m.validationErr != nil {
			b.WriteString(s.Error.Render("  failed: " + m.validationErr.Error()))
			b.WriteString("\n")
		}
		for _, r := range m.validation {
			status := s.Success.Render("OK")
			if !r.OK {
				status = s.Error.Render("failed")
			}
			line := fmt.Sprintf("  %s: %s", r.Component, status)
			if r.Err != "" {
				line += " — " + r.Err
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// rebuildRows repopulates the table from the visible cluster set.
func (m *Model) rebuildRows() {
	if !m.loaded {
		m.table.SetRows([]table.Row{{"Loading…", "", "", "", "", "", "", "", "", ""}})
		return
	}
	vis := m.visibleClusters()
	rows := make([]table.Row, 0, len(vis))
	for _, c := range vis {
		rows = append(rows, m.row(c))
	}
	m.table.SetRows(rows)
}

func (m *Model) row(c api.ClusterOverview) table.Row {
	name := c.Name
	if m.isCurrent(c.Name) {
		name = "* " + name
	}
	access := ""
	if c.ReadOnly {
		access = "readonly"
	}
	return table.Row{
		name,
		string(c.Status),
		versionOr(c.Version),
		fmt.Sprintf("%d", c.BrokerCount),
		fmt.Sprintf("%d", c.OnlinePartitionCount),
		fmt.Sprintf("%d", c.TopicCount),
		msgRate(c.MessagesInPerSec),
		humanRate(c.BytesInPerSec),
		humanRate(c.BytesOutPerSec),
		access,
	}
}

func (m *Model) isCurrent(name string) bool {
	if m.common == nil || m.common.DataSource == nil {
		return false
	}
	return m.common.DataSource.GetContext() == name
}

// visibleClusters applies the offline-only filter.
func (m *Model) visibleClusters() []api.ClusterOverview {
	if !m.offlineOnly {
		return m.clusters
	}
	out := make([]api.ClusterOverview, 0, len(m.clusters))
	for _, c := range m.clusters {
		if c.Status == api.ClusterOffline {
			out = append(out, c)
		}
	}
	return out
}

// selected returns the currently highlighted cluster from the visible set.
func (m *Model) selected() (api.ClusterOverview, bool) {
	if !m.loaded {
		return api.ClusterOverview{}, false
	}
	vis := m.visibleClusters()
	i := m.table.Cursor()
	if i < 0 || i >= len(vis) {
		return api.ClusterOverview{}, false
	}
	return vis[i], true
}

// --- helpers ---

func statusCounts(clusters []api.ClusterOverview) (online, offline, initializing int) {
	for _, c := range clusters {
		switch c.Status {
		case api.ClusterOnline:
			online++
		case api.ClusterOffline:
			offline++
		case api.ClusterInitializing:
			initializing++
		}
	}
	return
}

func versionOr(v string) string {
	if v == "" {
		return dash
	}
	return v
}

func msgRate(v float64) string {
	if v < 0 {
		return dash
	}
	return fmt.Sprintf("%.1f", v)
}

// humanRate formats a bytes-per-second rate with IEC units; negative is unknown.
func humanRate(v float64) string {
	if v < 0 {
		return dash
	}
	const unit = 1024.0
	if v < unit {
		return fmt.Sprintf("%.0f B/s", v)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	val, i := v, -1
	for val >= unit && i < len(units)-1 {
		val /= unit
		i++
	}
	return fmt.Sprintf("%.1f %s/s", val, units[i])
}
