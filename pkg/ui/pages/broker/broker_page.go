// Package broker implements the broker detail page (dynamic page ID
// "broker:<id>"). It renders a summary strip plus three tabs — Log Dirs,
// Configs and Metrics — over the shared template shell. The page is created by
// the router; see NewModelWithCommon / NewModelWithInfo for the constructors the
// router wires to the "broker:<id>" dynamic ID.
package broker

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components/editor"
	"github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model is the broker detail page.
type Model struct {
	common      *core.Common
	keys        pageKeys
	reusableApp *templateui.ReusableApp
	dims        core.Dimensions

	brokerID   int32
	info       api.BrokerInfo
	infoLoaded bool
	notFound   bool
	stats      api.BrokerStats
	statsOK    bool

	active tab

	// Log Dirs tab
	logDirs       []api.BrokerLogDir
	logDirsLoaded bool
	logDirsErr    error
	logTable      table.Model
	expanded      int // index of the expanded dir, -1 when none
	partTable     table.Model

	// Configs tab
	configs       []api.BrokerConfigEntry
	configsLoaded bool
	configsErr    error
	cfgTable      table.Model
	cfgVisible    []api.BrokerConfigEntry // entries backing the current cfgTable rows
	cfgFilter     string
	searching     bool
	searchInput   textinput.Model
	editing       bool
	editKey       string
	editOld       string
	editInput     textinput.Model

	// Metrics tab
	metricsViewer *editor.Viewer
	metricsLoaded bool
	metricsErr    error

	// Reassignment form
	moveForm *form.Form
}

// NewModelWithCommon builds the broker detail page for the given broker ID.
// The router wires this to the "broker:<id>" dynamic page ID.
func NewModelWithCommon(common *core.Common, brokerID int32) core.Page {
	return newModel(common, brokerID, api.BrokerInfo{}, false)
}

// NewModelWithInfo builds the page with broker metadata already known (passed via
// NavigationData from the list row), avoiding a refetch for the summary strip.
func NewModelWithInfo(common *core.Common, brokerID int32, info api.BrokerInfo) core.Page {
	return newModel(common, brokerID, info, true)
}

func newModel(common *core.Common, brokerID int32, info api.BrokerInfo, haveInfo bool) *Model {
	m := &Model{
		common:   common,
		keys:     defaultKeys(),
		brokerID: brokerID,
		expanded: -1,
	}
	if haveInfo {
		m.info = info
		m.infoLoaded = true
	}

	si := textinput.New()
	si.Prompt = "/"
	m.searchInput = si
	ei := textinput.New()
	m.editInput = ei

	m.logTable = table.New(table.WithColumns(logDirColumns()), table.WithFocused(true), table.WithHeight(10))
	m.partTable = table.New(table.WithColumns(partitionColumns()), table.WithFocused(true), table.WithHeight(8))
	m.cfgTable = table.New(table.WithColumns(configColumns()), table.WithFocused(true), table.WithHeight(12))
	m.metricsViewer = editor.NewViewer("")
	m.metricsViewer.SetHighlight(true)

	config := &providers.AppConfig{
		ContentProvider:      &contentProvider{model: m},
		ShowSidebarByDefault: false,
	}
	m.reusableApp = templateui.NewReusableApp(config)
	m.reusableApp.SetKeyMap(helpKeyMap{keys: m.keys})
	return m
}

// --- column definitions ---

func logDirColumns() []table.Column {
	return []table.Column{
		{Title: "Directory", Width: 34},
		{Title: "Error", Width: 22},
		{Title: "Topics", Width: 8},
		{Title: "Partitions", Width: 12},
	}
}

func partitionColumns() []table.Column {
	return []table.Column{
		{Title: "Topic", Width: 28},
		{Title: "Partition", Width: 10},
		{Title: "Size", Width: 14},
		{Title: "Offset Lag", Width: 12},
	}
}

func configColumns() []table.Column {
	return []table.Column{
		{Title: "Key", Width: 34},
		{Title: "Value", Width: 26},
		{Title: "Source", Width: 26},
	}
}

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
	m.dims = core.Dimensions{Width: width, Height: height}
	body := height - 12
	if body < 3 {
		body = 3
	}
	m.logTable.SetWidth(width)
	m.logTable.SetHeight(body)
	m.partTable.SetWidth(width)
	m.partTable.SetHeight(body / 2)
	m.cfgTable.SetWidth(width)
	m.cfgTable.SetHeight(body)
	m.metricsViewer.SetDimensions(width, body)
	m.reusableApp.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

func (m *Model) GetID() string    { return fmt.Sprintf("broker:%d", m.brokerID) }
func (m *Model) GetTitle() string { return fmt.Sprintf("Broker %d", m.brokerID) }

func (m *Model) GetHelp() []key.Binding {
	return []key.Binding{m.keys.NextTab, m.keys.Expand, m.keys.Edit, m.keys.Move, m.keys.Search, m.keys.Retry, m.keys.Back}
}

func (m *Model) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) { return m, nil }
func (m *Model) OnBlur() tea.Cmd                                   { return nil }

// OnFocus kicks off the initial data load: broker list (for found/not-found +
// summary), stats (for the summary strip) and the default tab's data.
func (m *Model) OnFocus() tea.Cmd {
	return tea.Batch(m.loadInfo(), m.loadStats(), m.loadTab(tabLogDirs))
}

// --- loads ---

func (m *Model) loadInfo() tea.Cmd {
	ds := m.common.DataSource
	id := m.brokerID
	return func() tea.Msg {
		brokers, err := ds.GetBrokers()
		if err != nil {
			return brokerInfoLoadedMsg{brokerID: id, err: err}
		}
		for _, b := range brokers {
			if b.ID == id {
				return brokerInfoLoadedMsg{brokerID: id, info: b, found: true}
			}
		}
		return brokerInfoLoadedMsg{brokerID: id, found: false}
	}
}

func (m *Model) loadStats() tea.Cmd {
	ds := m.common.DataSource
	id := m.brokerID
	return func() tea.Msg {
		stats, _, err := ds.GetBrokerStats()
		if err != nil {
			return brokerStatsLoadedMsg{brokerID: id}
		}
		s, ok := stats[id]
		return brokerStatsLoadedMsg{brokerID: id, stats: s, ok: ok}
	}
}

func (m *Model) loadTab(t tab) tea.Cmd {
	switch t {
	case tabConfigs:
		if m.configsLoaded {
			return nil
		}
		return m.loadConfigs()
	case tabMetrics:
		if m.metricsLoaded {
			return nil
		}
		return m.loadMetrics()
	default:
		if m.logDirsLoaded {
			return nil
		}
		return m.loadLogDirs()
	}
}

func (m *Model) loadLogDirs() tea.Cmd {
	ds := m.common.DataSource
	id := m.brokerID
	return func() tea.Msg {
		dirs, err := ds.GetBrokerLogDirs([]int32{id})
		if err != nil {
			return logDirsLoadedMsg{brokerID: id, err: err}
		}
		return logDirsLoadedMsg{brokerID: id, dirs: dirs[id]}
	}
}

func (m *Model) loadConfigs() tea.Cmd {
	ds := m.common.DataSource
	id := m.brokerID
	return func() tea.Msg {
		entries, err := ds.GetBrokerConfig(id)
		return configsLoadedMsg{brokerID: id, entries: entries, err: err}
	}
}

func (m *Model) loadMetrics() tea.Cmd {
	ds := m.common.DataSource
	id := m.brokerID
	return func() tea.Msg {
		data, err := ds.GetBrokerMetrics(id)
		return metricsLoadedMsg{brokerID: id, data: data, err: err}
	}
}

// --- message handling (via the content provider) ---

func (m *Model) handle(msg tea.Msg) tea.Cmd {
	switch v := msg.(type) {
	case brokerInfoLoadedMsg:
		if v.brokerID != m.brokerID {
			return nil
		}
		if v.err != nil {
			m.notFound = true
			return nil
		}
		m.notFound = !v.found
		if v.found {
			m.info = v.info
			m.infoLoaded = true
		}
		return nil
	case brokerStatsLoadedMsg:
		if v.brokerID == m.brokerID {
			m.stats = v.stats
			m.statsOK = v.ok
		}
		return nil
	case logDirsLoadedMsg:
		if v.brokerID == m.brokerID {
			m.logDirs = v.dirs
			m.logDirsErr = v.err
			m.logDirsLoaded = true
			m.rebuildLogTable()
		}
		return nil
	case configsLoadedMsg:
		if v.brokerID == m.brokerID {
			m.configs = v.entries
			m.configsErr = v.err
			m.configsLoaded = true
			m.rebuildConfigTable()
		}
		return nil
	case metricsLoadedMsg:
		if v.brokerID == m.brokerID {
			m.metricsLoaded = true
			m.metricsErr = v.err
			if v.err == nil {
				m.metricsViewer.SetContent(v.data)
			}
		}
		return nil
	case configAlteredMsg:
		return m.handleConfigAltered(v)
	case replicaMovedMsg:
		return m.handleReplicaMoved(v)
	case form.FormSubmitMsg:
		return m.handleMoveSubmit(v)
	case form.FormCancelMsg:
		m.moveForm = nil
		return nil
	case tea.KeyMsg:
		return m.handleKey(v)
	}
	// Forward other messages (mouse, viewport) to the active component.
	return m.forwardToActive(msg)
}

func (m *Model) forwardToActive(msg tea.Msg) tea.Cmd {
	switch m.active {
	case tabConfigs:
		var cmd tea.Cmd
		m.cfgTable, cmd = m.cfgTable.Update(msg)
		return cmd
	case tabMetrics:
		_, cmd := m.metricsViewer.Update(msg)
		return cmd
	default:
		var cmd tea.Cmd
		if m.expanded >= 0 {
			m.partTable, cmd = m.partTable.Update(msg)
		} else {
			m.logTable, cmd = m.logTable.Update(msg)
		}
		return cmd
	}
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Modal-ish sub-states first.
	if m.moveForm != nil {
		cmd, _ := m.moveForm.Update(msg)
		return cmd
	}
	if m.editing {
		return m.handleEditKey(msg)
	}
	if m.searching {
		return m.handleSearchKey(msg)
	}

	// Tab switching: tab key + number keys 1/2/3.
	switch msg.String() {
	case "tab":
		return m.switchTab((m.active + 1) % tab(len(tabTitles)))
	case "1":
		return m.switchTab(tabLogDirs)
	case "2":
		return m.switchTab(tabConfigs)
	case "3":
		return m.switchTab(tabMetrics)
	case "r":
		return m.retry()
	}

	switch m.active {
	case tabConfigs:
		return m.handleConfigsKey(msg)
	case tabLogDirs:
		return m.handleLogDirsKey(msg)
	}
	return m.forwardToActive(msg)
}

func (m *Model) switchTab(t tab) tea.Cmd {
	m.active = t
	return m.loadTab(t)
}

func (m *Model) retry() tea.Cmd {
	if m.notFound {
		m.notFound = false
		return m.loadInfo()
	}
	switch m.active {
	case tabConfigs:
		m.configsLoaded = false
		return m.loadConfigs()
	case tabMetrics:
		m.metricsLoaded = false
		return m.loadMetrics()
	default:
		m.logDirsLoaded = false
		return m.loadLogDirs()
	}
}

// --- Log Dirs tab ---

func (m *Model) handleLogDirsKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		if m.expanded >= 0 {
			m.expanded = -1 // collapse
			return nil
		}
		i := m.logTable.Cursor()
		if i >= 0 && i < len(m.logDirs) {
			m.expanded = i
			m.rebuildPartTable()
		}
		return nil
	case "esc":
		if m.expanded >= 0 {
			m.expanded = -1
			return nil
		}
	case "m":
		if m.expanded >= 0 {
			return m.openMoveForm()
		}
	}
	return m.forwardToActive(msg)
}

func (m *Model) rebuildLogTable() {
	rows := make([]table.Row, 0, len(m.logDirs))
	for _, d := range m.logDirs {
		parts := 0
		for _, t := range d.Topics {
			parts += len(t.Partitions)
		}
		rows = append(rows, table.Row{d.Path, d.Error, strconv.Itoa(len(d.Topics)), strconv.Itoa(parts)})
	}
	m.logTable.SetRows(rows)
}

func (m *Model) rebuildPartTable() {
	if m.expanded < 0 || m.expanded >= len(m.logDirs) {
		m.partTable.SetRows(nil)
		return
	}
	var rows []table.Row
	for _, t := range m.logDirs[m.expanded].Topics {
		for _, p := range t.Partitions {
			rows = append(rows, table.Row{
				t.Topic,
				strconv.FormatInt(int64(p.Partition), 10),
				shared.FormatBytes2dp(p.Size),
				strconv.FormatInt(p.OffsetLag, 10),
			})
		}
	}
	m.partTable.SetRows(rows)
	m.partTable.SetCursor(0)
}

// selectedPartition returns the topic/partition currently highlighted in the
// expanded partition table.
func (m *Model) selectedPartition() (topic string, partition int32, ok bool) {
	if m.expanded < 0 || m.expanded >= len(m.logDirs) {
		return "", 0, false
	}
	idx := 0
	cursor := m.partTable.Cursor()
	for _, t := range m.logDirs[m.expanded].Topics {
		for _, p := range t.Partitions {
			if idx == cursor {
				return t.Topic, p.Partition, true
			}
			idx++
		}
	}
	return "", 0, false
}

// --- Reassignment form (BR-17) ---

func (m *Model) openMoveForm() tea.Cmd {
	topic, part, ok := m.selectedPartition()
	if !ok {
		return core.NewNotification(core.StatusWarning, "Move replica", "no partition selected")
	}
	// Offer the broker's other log directories as targets.
	var targets []string
	for i, d := range m.logDirs {
		if i == m.expanded {
			continue
		}
		targets = append(targets, d.Path)
	}
	def := ""
	if len(targets) > 0 {
		def = targets[0]
	}
	m.moveForm = form.New([]form.Field{
		{Name: "topic", Label: "Topic", Type: form.Text, Default: topic},
		{Name: "partition", Label: "Partition", Type: form.Text, Default: strconv.FormatInt(int64(part), 10)},
		{Name: "logdir", Label: "Target log dir", Type: form.Text, Required: true, Default: def, Options: targets},
	})
	m.moveForm.SetDimensions(m.dims.Width, m.dims.Height)
	return m.moveForm.Focus()
}

func (m *Model) handleMoveSubmit(msg form.FormSubmitMsg) tea.Cmd {
	m.moveForm = nil
	topic := msg.Values["topic"]
	logDir := msg.Values["logdir"]
	part, _ := strconv.ParseInt(msg.Values["partition"], 10, 32)
	partition := int32(part)
	id := m.brokerID
	ds := m.common.DataSource
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Move replica",
			Message:      fmt.Sprintf("Move %s-%d to %s?", topic, partition, logDir),
			ConfirmLabel: "Move",
			OnConfirm: func() tea.Msg {
				err := ds.AlterReplicaLogDir(id, topic, partition, logDir)
				return replicaMovedMsg{brokerID: id, topic: topic, partition: partition, logDir: logDir, err: err}
			},
		}
	}
}

func (m *Model) handleReplicaMoved(v replicaMovedMsg) tea.Cmd {
	if v.err != nil {
		return func() tea.Msg { return shared.NewUIError("reassign", "Replica move failed", v.err) }
	}
	m.logDirsLoaded = false
	m.expanded = -1
	return tea.Batch(core.NewNotification(core.StatusSuccess, "Replica moved", fmt.Sprintf("%s-%d → %s", v.topic, v.partition, v.logDir)), m.loadLogDirs())
}

// --- Configs tab (BR-15/BR-16) ---

func (m *Model) handleConfigsKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "/":
		m.searching = true
		m.searchInput.SetValue(m.cfgFilter)
		return m.searchInput.Focus()
	case "e":
		return m.beginEdit()
	}
	return m.forwardToActive(msg)
}

func (m *Model) handleSearchKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		m.searching = false
		m.cfgFilter = m.searchInput.Value()
		m.searchInput.Blur()
		m.rebuildConfigTable()
		return nil
	case "esc":
		m.searching = false
		m.searchInput.Blur()
		return nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return cmd
}

func (m *Model) selectedConfig() (api.BrokerConfigEntry, bool) {
	i := m.cfgTable.Cursor()
	if i < 0 || i >= len(m.cfgVisible) {
		return api.BrokerConfigEntry{}, false
	}
	return m.cfgVisible[i], true
}

func (m *Model) beginEdit() tea.Cmd {
	entry, ok := m.selectedConfig()
	if !ok {
		return nil
	}
	if entry.ReadOnly {
		return core.NewNotification(core.StatusWarning, "Config", "Property is read-only")
	}
	m.editing = true
	m.editKey = entry.Name
	m.editOld = entry.Value
	m.editInput.SetValue(entry.Value)
	return m.editInput.Focus()
}

func (m *Model) handleEditKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.cancelEdit()
		return nil
	case "enter":
		return m.commitEdit()
	}
	var cmd tea.Cmd
	m.editInput, cmd = m.editInput.Update(msg)
	return cmd
}

func (m *Model) cancelEdit() {
	m.editing = false
	m.editInput.Blur()
	m.editKey = ""
}

// commitEdit implements the save state machine: unchanged value is a no-op;
// a changed value asks for confirmation before calling AlterBrokerConfig.
func (m *Model) commitEdit() tea.Cmd {
	newVal := m.editInput.Value()
	if newVal == m.editOld {
		m.cancelEdit()
		return nil
	}
	key := m.editKey
	id := m.brokerID
	ds := m.common.DataSource
	// Keep edit mode open until the change is confirmed + applied.
	m.editInput.Blur()
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Change config",
			Message:      "Are you sure you want to change the value?",
			Danger:       true,
			ConfirmLabel: "Change",
			OnConfirm: func() tea.Msg {
				err := ds.AlterBrokerConfig(id, key, newVal)
				return configAlteredMsg{brokerID: id, key: key, value: newVal, err: err}
			},
		}
	}
}

func (m *Model) handleConfigAltered(v configAlteredMsg) tea.Cmd {
	if v.brokerID != m.brokerID {
		return nil
	}
	if v.err != nil {
		// Stay in edit mode and surface the cluster's rejection message.
		m.editing = true
		m.editInput.SetValue(v.value)
		var invalid api.InvalidConfigError
		msg := v.err.Error()
		if ok := asInvalidConfig(v.err, &invalid); ok {
			msg = invalid.Error()
		}
		return tea.Batch(
			func() tea.Msg { return shared.NewUIError("config", msg, nil) },
			m.editInput.Focus(),
		)
	}
	m.cancelEdit()
	m.configsLoaded = false
	return tea.Batch(core.NewNotification(core.StatusSuccess, "Config updated", v.key), m.loadConfigs())
}

func (m *Model) rebuildConfigTable() {
	entries := sortedFilteredConfigs(m.configs, m.cfgFilter)
	m.cfgVisible = entries
	rows := make([]table.Row, 0, len(entries))
	for _, e := range entries {
		val := shared.FormatConfigValue(e.Name, e.Value, e.Sensitive)
		rows = append(rows, table.Row{e.Name, val, e.Source})
	}
	m.cfgTable.SetRows(rows)
	if m.cfgTable.Cursor() >= len(rows) {
		m.cfgTable.SetCursor(0)
	}
}

// --- rendering ---

func (m *Model) render(width, height int) string {
	// Size the tables to the actual content area (minus the frame border) so
	// they fill the pane and never overflow it. height budget: summary(1) +
	// tabbar(1) + blank(1) + hint(1) + frame(2) ≈ 6, plus slack.
	innerW := width - 2
	if innerW < 20 {
		innerW = 20
	}
	th := height - 8
	if th < 3 {
		th = 3
	}
	m.logTable.SetWidth(innerW)
	m.cfgTable.SetWidth(innerW)
	if m.expanded >= 0 {
		half := th/2 - 1
		if half < 2 {
			half = 2
		}
		m.logTable.SetHeight(half)
		m.partTable.SetWidth(innerW)
		m.partTable.SetHeight(half)
	} else {
		m.logTable.SetHeight(th)
	}
	m.cfgTable.SetHeight(th)

	var b strings.Builder
	b.WriteString(m.summaryStrip())
	b.WriteString("\n")
	b.WriteString(m.tabBar())
	b.WriteString("\n\n")

	if m.notFound {
		b.WriteString(m.common.Styles.Error.Render(fmt.Sprintf("Broker %d not found.", m.brokerID)))
		b.WriteString("\n")
		b.WriteString(m.common.Styles.Muted.Render("Press r to retry."))
		return b.String()
	}

	if m.moveForm != nil {
		b.WriteString(m.common.Styles.Header.Render("Move replica log directory"))
		b.WriteString("\n\n")
		b.WriteString(m.moveForm.View())
		return b.String()
	}

	switch m.active {
	case tabConfigs:
		b.WriteString(m.renderConfigs())
	case tabMetrics:
		b.WriteString(m.renderMetrics())
	default:
		b.WriteString(m.renderLogDirs())
	}
	return b.String()
}

func (m *Model) summaryStrip() string {
	host := m.info.Host
	port := strconv.FormatInt(int64(m.info.Port), 10)
	seg := "N/A"
	if m.statsOK {
		seg = shared.FormatDiskUsage(m.stats.SegmentSize, m.stats.SegmentCount)
	}
	parts := []string{
		m.common.Styles.Header.Render(fmt.Sprintf("Broker %d", m.brokerID)),
		"Host: " + host,
		"Port: " + port,
		"Disk: " + seg,
	}
	if m.info.IsController {
		parts = append(parts, m.common.Styles.StatusStyle.Success.Render("★ Active Controller"))
	}
	return strings.Join(parts, "   ")
}

func (m *Model) tabBar() string {
	active := lipgloss.NewStyle().Foreground(stylesPkg.BgBase).Background(stylesPkg.Primary).Bold(true).Padding(0, 1)
	inactive := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted).Padding(0, 1)
	var cells []string
	for i, t := range tabTitles {
		label := fmt.Sprintf("%d %s", i+1, t.String())
		if t == m.active {
			cells = append(cells, active.Render(label))
		} else {
			cells = append(cells, inactive.Render(label))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

func (m *Model) renderLogDirs() string {
	if !m.logDirsLoaded {
		return m.common.Styles.Muted.Render("Loading log directories…")
	}
	if m.logDirsErr != nil {
		return m.common.Styles.Error.Render("Error: "+m.logDirsErr.Error()) + "\n" + m.common.Styles.Muted.Render("Press r to retry.")
	}
	if len(m.logDirs) == 0 {
		return m.common.Styles.Muted.Render("Log dir data not available")
	}
	var b strings.Builder
	b.WriteString(stylesPkg.FrameTable(m.logTable.View()))
	if m.expanded >= 0 && m.expanded < len(m.logDirs) {
		b.WriteString("\n\n")
		b.WriteString(m.common.Styles.Header.Render("Partitions in " + m.logDirs[m.expanded].Path))
		b.WriteString("\n")
		b.WriteString(stylesPkg.FrameTable(m.partTable.View()))
		b.WriteString("\n")
		b.WriteString(m.common.Styles.Muted.Render("m: move replica • enter/esc: collapse"))
	} else {
		b.WriteString("\n")
		b.WriteString(m.common.Styles.Muted.Render("enter: expand directory"))
	}
	return b.String()
}

func (m *Model) renderConfigs() string {
	if !m.configsLoaded {
		return m.common.Styles.Muted.Render("Loading configs…")
	}
	if m.configsErr != nil {
		return m.common.Styles.Error.Render("Error: "+m.configsErr.Error()) + "\n" + m.common.Styles.Muted.Render("Press r to retry.")
	}
	var b strings.Builder
	b.WriteString(stylesPkg.FrameTable(m.cfgTable.View()))
	b.WriteString("\n")
	if m.searching {
		b.WriteString(m.searchInput.View())
		return b.String()
	}
	if m.editing {
		b.WriteString(m.common.Styles.Header.Render("Edit " + m.editKey + ": "))
		b.WriteString(m.editInput.View())
		b.WriteString("\n")
		b.WriteString(m.common.Styles.Muted.Render("enter: save • esc: cancel"))
		return b.String()
	}
	b.WriteString(m.configFooter())
	return b.String()
}

// configFooter shows the source-category hint and sensitive/exact-byte hints for
// the selected row (the TUI adaptation of the hover tooltips).
func (m *Model) configFooter() string {
	entry, ok := m.selectedConfig()
	if !ok {
		return m.common.Styles.Muted.Render("e: edit • /: search")
	}
	hint := m.common.Styles.Muted.Render(sourceExplanation(entry.Source))
	extra := ""
	if entry.Sensitive {
		extra = "  •  Sensitive Value"
	} else if n, err := strconv.ParseInt(entry.Value, 10, 64); err == nil && n > 0 && strings.HasSuffix(entry.Name, ".bytes") {
		extra = fmt.Sprintf("  •  %d bytes", n)
	}
	return hint + m.common.Styles.Muted.Render(extra) + "\n" + m.common.Styles.Muted.Render("e: edit • /: search")
}

func (m *Model) renderMetrics() string {
	if !m.metricsLoaded {
		return m.common.Styles.Muted.Render("Loading metrics…")
	}
	if m.metricsErr != nil {
		return m.common.Styles.Muted.Render("Metrics data not available")
	}
	return m.metricsViewer.View()
}

// --- ordering / filtering helpers (pure) ---

// sourceRank orders config sources: dynamic* first, then static broker, default,
// then unknown/other.
func sourceRank(source string) int {
	switch source {
	case "Dynamic broker config":
		return 0
	case "Dynamic default broker config":
		return 1
	case "Static broker config":
		return 2
	case "Default config":
		return 3
	case "Unknown":
		return 4
	default:
		return 5
	}
}

func sourceExplanation(source string) string {
	switch source {
	case "Dynamic broker config":
		return "Dynamic broker config: set per-broker at runtime"
	case "Dynamic default broker config":
		return "Dynamic default broker config: cluster-wide runtime default"
	case "Static broker config":
		return "Static broker config: from server.properties (needs restart)"
	case "Default config":
		return "Default config: Kafka built-in default"
	default:
		return "Unknown config source"
	}
}

// sortedFilteredConfigs returns entries filtered by a case-insensitive substring
// match on key OR value, ordered by source priority (stable within groups).
func sortedFilteredConfigs(entries []api.BrokerConfigEntry, filter string) []api.BrokerConfigEntry {
	out := make([]api.BrokerConfigEntry, 0, len(entries))
	q := strings.ToLower(filter)
	for _, e := range entries {
		if q == "" || strings.Contains(strings.ToLower(e.Name), q) || strings.Contains(strings.ToLower(e.Value), q) {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return sourceRank(out[i].Source) < sourceRank(out[j].Source)
	})
	return out
}

// asInvalidConfig reports whether err is (or wraps) an api.InvalidConfigError,
// copying it into dst when so.
func asInvalidConfig(err error, dst *api.InvalidConfigError) bool {
	if ic, ok := err.(api.InvalidConfigError); ok {
		*dst = ic
		return true
	}
	return false
}
