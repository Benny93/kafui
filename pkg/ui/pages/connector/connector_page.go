// Package connector implements the connector detail page (dynamic page ID
// "connector:<connect>:<name>"). It renders a summary strip plus four tabs —
// Overview, Tasks, Config and Topics — over the shared template shell. The page
// is created by the router; see NewModelWithCommon for the constructor the
// router wires to the "connector:<connect>:<name>" dynamic ID.
package connector

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components/editor"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model is the connector detail page.
type Model struct {
	common      *core.Common
	keys        pageKeys
	reusableApp *templateui.ReusableApp
	dims        core.Dimensions

	connect string
	name    string

	details       api.ConnectorDetails
	detailsLoaded bool
	notFound      bool
	loadErr       error

	active tab

	// Tasks tab
	tasksTable   table.Model
	expandedTask int // index of the expanded task, -1 when none

	// Config tab
	configEditor *editor.Editor
	editing      bool
	configText   string // the masked JSON currently displayed (edit baseline)
}

// NewModelWithCommon builds the connector detail page for the given Connect
// cluster and connector name. The router wires this to the
// "connector:<connect>:<name>" dynamic page ID.
func NewModelWithCommon(common *core.Common, connectCluster, connectorName string) core.Page {
	return newModel(common, connectCluster, connectorName)
}

func newModel(common *core.Common, connect, name string) *Model {
	m := &Model{
		common:       common,
		keys:         defaultKeys(),
		connect:      connect,
		name:         name,
		expandedTask: -1,
	}

	m.tasksTable = table.New(table.WithColumns(taskColumns()), table.WithFocused(true), table.WithHeight(10))
	m.configEditor = editor.NewEditor("")

	config := &providers.AppConfig{
		ContentProvider:      &contentProvider{model: m},
		ShowSidebarByDefault: false,
	}
	m.reusableApp = templateui.NewReusableApp(config)
	m.reusableApp.SetKeyMap(helpKeyMap{keys: m.keys})
	return m
}

func taskColumns() []table.Column {
	return []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Worker", Width: 20},
		{Title: "State", Width: 14},
		{Title: "Trace", Width: 40},
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
	m.tasksTable.SetWidth(width)
	m.tasksTable.SetHeight(body)
	m.configEditor.SetDimensions(width, body)
	m.reusableApp.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

func (m *Model) GetID() string    { return fmt.Sprintf("connector:%s:%s", m.connect, m.name) }
func (m *Model) GetTitle() string { return m.name }

func (m *Model) GetHelp() []key.Binding {
	return []key.Binding{m.keys.NextTab, m.keys.Pause, m.keys.Resume, m.keys.Stop, m.keys.Restart, m.keys.Delete, m.keys.Edit, m.keys.Retry, m.keys.Back}
}

func (m *Model) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) { return m, nil }
func (m *Model) OnBlur() tea.Cmd                                   { return nil }

// OnFocus kicks off the initial detail load.
func (m *Model) OnFocus() tea.Cmd { return m.loadDetails() }

// --- loads ---

func (m *Model) loadDetails() tea.Cmd {
	ds := m.common.DataSource
	connect, name := m.connect, m.name
	return func() tea.Msg {
		details, err := ds.GetConnectorDetails(connect, name)
		if err != nil {
			var nf api.ConnectorNotFoundError
			if asConnectorNotFound(err, &nf) {
				return detailsLoadedMsg{connect: connect, name: name, found: false}
			}
			return detailsLoadedMsg{connect: connect, name: name, err: err}
		}
		return detailsLoadedMsg{connect: connect, name: name, details: details, found: true}
	}
}

// --- message handling (via the content provider) ---

func (m *Model) handle(msg tea.Msg) tea.Cmd {
	switch v := msg.(type) {
	case detailsLoadedMsg:
		if v.connect != m.connect || v.name != m.name {
			return nil
		}
		if v.err != nil {
			m.loadErr = v.err
			return nil
		}
		m.notFound = !v.found
		m.loadErr = nil
		if v.found {
			m.details = v.details
			m.detailsLoaded = true
			m.rebuildTaskTable()
		}
		return nil
	case lifecycleResultMsg:
		return m.handleLifecycleResult(v)
	case taskRestartResultMsg:
		return m.handleTaskRestartResult(v)
	case configUpdatedMsg:
		return m.handleConfigUpdated(v)
	case tea.KeyMsg:
		return m.handleKey(v)
	}
	return m.forwardToActive(msg)
}

func (m *Model) forwardToActive(msg tea.Msg) tea.Cmd {
	switch m.active {
	case tabConfig:
		if m.editing {
			_, cmd := m.configEditor.Update(msg)
			return cmd
		}
	case tabTasks:
		var cmd tea.Cmd
		m.tasksTable, cmd = m.tasksTable.Update(msg)
		return cmd
	}
	return nil
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Config edit sub-state swallows keys except save/cancel.
	if m.editing {
		switch msg.String() {
		case "esc":
			m.editing = false
			return nil
		case "ctrl+s":
			return m.commitConfigEdit()
		}
		_, cmd := m.configEditor.Update(msg)
		return cmd
	}

	switch msg.String() {
	case "tab":
		return m.switchTab((m.active + 1) % tab(len(tabTitles)))
	case "1":
		return m.switchTab(tabOverview)
	case "2":
		return m.switchTab(tabTasks)
	case "3":
		return m.switchTab(tabConfig)
	case "4":
		return m.switchTab(tabTopics)
	case "r":
		return m.retry()
	}

	// Lifecycle actions available on any tab (state-aware).
	switch msg.String() {
	case "p":
		return m.lifecycle("pause", m.common.DataSource.PauseConnector)
	case "u":
		return m.lifecycle("resume", m.common.DataSource.ResumeConnector)
	case "s":
		return m.lifecycle("stop", m.common.DataSource.StopConnector)
	case "R":
		return m.lifecycle("restart", m.common.DataSource.RestartConnector)
	case "ctrl+d":
		return m.deleteConnector()
	case "z":
		return m.resetOffsets()
	}

	// Tab-specific keys.
	switch m.active {
	case tabTasks:
		return m.handleTasksKey(msg)
	case tabTopics:
		return m.handleTopicsKey(msg)
	case tabConfig:
		if msg.String() == "e" {
			return m.beginConfigEdit()
		}
	}
	return m.forwardToActive(msg)
}

func (m *Model) switchTab(t tab) tea.Cmd {
	m.active = t
	return nil
}

func (m *Model) retry() tea.Cmd {
	m.notFound = false
	m.loadErr = nil
	m.detailsLoaded = false
	return m.loadDetails()
}

// --- lifecycle actions (KC-14/KC-16) ---

// lifecycle wraps a state-changing connector action in a confirmation dialog.
func (m *Model) lifecycle(action string, fn func(connect, name string) error) tea.Cmd {
	connect, name := m.connect, m.name
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        strings.Title(action) + " connector",
			Message:      fmt.Sprintf("%s connector %q?", action, name),
			Danger:       true,
			ConfirmLabel: strings.Title(action),
			OnConfirm: func() tea.Msg {
				return lifecycleResultMsg{action: action, err: fn(connect, name)}
			},
		}
	}
}

func (m *Model) deleteConnector() tea.Cmd {
	connect, name := m.connect, m.name
	ds := m.common.DataSource
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Delete connector",
			Message:      fmt.Sprintf("Delete connector %q? This cannot be undone.", name),
			Danger:       true,
			ConfirmLabel: "Delete",
			OnConfirm: func() tea.Msg {
				err := ds.DeleteConnector(connect, name)
				return lifecycleResultMsg{action: "delete", deleted: err == nil, err: err}
			},
		}
	}
}

// resetOffsets confirms then calls ResetConnectorOffsets. The datasource
// enforces the STOPPED guard and returns ConnectorNotStoppedError otherwise,
// which is surfaced in the status bar.
func (m *Model) resetOffsets() tea.Cmd {
	connect, name := m.connect, m.name
	ds := m.common.DataSource
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Reset offsets",
			Message:      fmt.Sprintf("Reset offsets for connector %q? (requires STOPPED state)", name),
			Danger:       true,
			ConfirmLabel: "Reset",
			OnConfirm: func() tea.Msg {
				return lifecycleResultMsg{action: "reset-offsets", err: ds.ResetConnectorOffsets(connect, name)}
			},
		}
	}
}

func (m *Model) handleLifecycleResult(v lifecycleResultMsg) tea.Cmd {
	if v.err != nil {
		return func() tea.Msg { return shared.NewUIError("connector", v.action+" failed", v.err) }
	}
	if v.deleted {
		// Delete → return to the connectors listing via router history.
		return tea.Batch(
			core.NewNotification(core.StatusSuccess, "Connector deleted", m.name),
			func() tea.Msg { return core.BackMsg{} },
		)
	}
	m.detailsLoaded = false
	return tea.Batch(
		core.NewNotification(core.StatusSuccess, "Connector "+v.action, m.name),
		m.loadDetails(),
	)
}

// --- Tasks tab (KC-15) ---

func (m *Model) handleTasksKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		if m.expandedTask >= 0 {
			m.expandedTask = -1
			return nil
		}
		i := m.tasksTable.Cursor()
		if i >= 0 && i < len(m.details.Tasks) {
			m.expandedTask = i
		}
		return nil
	case "esc":
		if m.expandedTask >= 0 {
			m.expandedTask = -1
			return nil
		}
	case "t":
		return m.restartSelectedTask()
	case "T":
		return m.restartTasks("all", func(api.ConnectorTask) bool { return true })
	case "f":
		return m.restartTasks("failed", func(tk api.ConnectorTask) bool {
			return strings.EqualFold(tk.State, api.ConnectorStateFailed)
		})
	}
	return m.forwardToActive(msg)
}

func (m *Model) restartSelectedTask() tea.Cmd {
	i := m.tasksTable.Cursor()
	if i < 0 || i >= len(m.details.Tasks) {
		return nil
	}
	taskID := m.details.Tasks[i].ID
	connect, name := m.connect, m.name
	ds := m.common.DataSource
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Restart task",
			Message:      fmt.Sprintf("Restart task %d of connector %q?", taskID, name),
			Danger:       true,
			ConfirmLabel: "Restart",
			OnConfirm: func() tea.Msg {
				err := ds.RestartConnectorTask(connect, name, taskID)
				failures := []string(nil)
				if err != nil {
					failures = []string{fmt.Sprintf("task %d: %v", taskID, err)}
				}
				return taskRestartResultMsg{total: 1, failures: failures}
			},
		}
	}
}

// restartTasks restarts every task matching pred, reporting per-task failures
// without aborting the batch.
func (m *Model) restartTasks(label string, pred func(api.ConnectorTask) bool) tea.Cmd {
	var ids []int
	for _, tk := range m.details.Tasks {
		if pred(tk) {
			ids = append(ids, tk.ID)
		}
	}
	if len(ids) == 0 {
		return core.NewNotification(core.StatusWarning, "Restart tasks", "no matching tasks")
	}
	connect, name := m.connect, m.name
	ds := m.common.DataSource
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Restart " + label + " tasks",
			Message:      fmt.Sprintf("Restart %d %s task(s) of connector %q?", len(ids), label, name),
			Danger:       true,
			ConfirmLabel: "Restart",
			OnConfirm: func() tea.Msg {
				var failures []string
				for _, id := range ids {
					if err := ds.RestartConnectorTask(connect, name, id); err != nil {
						failures = append(failures, fmt.Sprintf("task %d: %v", id, err))
					}
				}
				return taskRestartResultMsg{total: len(ids), failures: failures}
			},
		}
	}
}

func (m *Model) handleTaskRestartResult(v taskRestartResultMsg) tea.Cmd {
	m.detailsLoaded = false
	if len(v.failures) > 0 {
		return tea.Batch(
			core.NewNotification(core.StatusWarning, "Task restart", fmt.Sprintf("%d/%d failed: %s", len(v.failures), v.total, strings.Join(v.failures, "; "))),
			m.loadDetails(),
		)
	}
	return tea.Batch(
		core.NewNotification(core.StatusSuccess, "Tasks restarted", strconv.Itoa(v.total)),
		m.loadDetails(),
	)
}

// --- Topics tab ---

func (m *Model) handleTopicsKey(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "enter" && len(m.details.Topics) > 0 {
		// Navigate to the first topic (single-list, no cursor state kept here).
		// ponytail: per-row topic cursor deferred; opens the first topic.
		topic := m.details.Topics[0]
		return core.NewPageChangeMsg("topic:"+topic, map[string]interface{}{"name": topic})
	}
	return nil
}

// --- Config tab (KC-18) ---

func (m *Model) beginConfigEdit() tea.Cmd {
	m.configText = m.configJSON()
	m.configEditor.SetValue(m.configText)
	m.editing = true
	return m.configEditor.Focus()
}

// commitConfigEdit validates the edited JSON, requires a change from the loaded
// value, then confirms before calling UpdateConnectorConfig.
func (m *Model) commitConfigEdit() tea.Cmd {
	newText := m.configEditor.Value()
	if newText == m.configText {
		return core.NewNotification(core.StatusWarning, "Config", "no changes to save")
	}
	var parsed map[string]string
	if err := json.Unmarshal([]byte(newText), &parsed); err != nil {
		return core.NotifyError("Invalid JSON config", err)
	}
	connect, name := m.connect, m.name
	ds := m.common.DataSource
	m.configEditor.Blur()
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Update config",
			Message:      "Save connector configuration? Masked secrets (********) will overwrite the stored values as-is.",
			Danger:       true,
			ConfirmLabel: "Save",
			OnConfirm: func() tea.Msg {
				_, err := ds.UpdateConnectorConfig(connect, name, parsed)
				return configUpdatedMsg{err: err}
			},
		}
	}
}

func (m *Model) handleConfigUpdated(v configUpdatedMsg) tea.Cmd {
	if v.err != nil {
		m.editing = true
		return tea.Batch(
			func() tea.Msg { return shared.NewUIError("connector", "Config update failed", v.err) },
			m.configEditor.Focus(),
		)
	}
	m.editing = false
	m.detailsLoaded = false
	return tea.Batch(core.NewNotification(core.StatusSuccess, "Config updated", m.name), m.loadDetails())
}

// configJSON renders the (masked) config map as indented JSON.
func (m *Model) configJSON() string {
	if len(m.details.Config) == 0 {
		return "{}"
	}
	b, err := json.MarshalIndent(m.details.Config, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (m *Model) configHasMasked() bool {
	for _, v := range m.details.Config {
		if v == api.ConnectorSecretPlaceholder {
			return true
		}
	}
	return false
}

// --- rendering ---

func (m *Model) rebuildTaskTable() {
	rows := make([]table.Row, 0, len(m.details.Tasks))
	for _, tk := range m.details.Tasks {
		rows = append(rows, table.Row{
			strconv.Itoa(tk.ID),
			tk.WorkerID,
			tk.State,
			core.TruncateString(strings.ReplaceAll(tk.Trace, "\n", " "), 40),
		})
	}
	m.tasksTable.SetRows(rows)
}

func stateStyle(common *core.Common, state string) lipgloss.Style {
	switch strings.ToUpper(state) {
	case api.ConnectorStateRunning:
		return common.Styles.StatusStyle.Success
	case api.ConnectorStateFailed:
		return common.Styles.StatusStyle.Error
	case api.ConnectorStatePaused, api.ConnectorStateRestarting:
		return common.Styles.StatusStyle.Warning
	default:
		return common.Styles.Muted
	}
}

func (m *Model) render(width, height int) string {
	m.tasksTable.SetWidth(width - 2) // -2 leaves room for the FrameTable border
	var b strings.Builder
	b.WriteString(m.summaryStrip())
	b.WriteString("\n")
	b.WriteString(m.tabBar())
	b.WriteString("\n\n")

	if m.notFound {
		b.WriteString(m.common.Styles.Error.Render(fmt.Sprintf("Connector %q not found on %q.", m.name, m.connect)))
		b.WriteString("\n")
		b.WriteString(m.common.Styles.Muted.Render("Press r to retry."))
		return b.String()
	}
	if m.loadErr != nil {
		b.WriteString(m.common.Styles.Error.Render("Error: " + m.loadErr.Error()))
		b.WriteString("\n")
		b.WriteString(m.common.Styles.Muted.Render("Press r to retry."))
		return b.String()
	}
	if !m.detailsLoaded {
		b.WriteString(m.common.Styles.Muted.Render("Loading connector…"))
		return b.String()
	}

	switch m.active {
	case tabTasks:
		b.WriteString(m.renderTasks())
	case tabConfig:
		b.WriteString(m.renderConfig())
	case tabTopics:
		b.WriteString(m.renderTopics())
	default:
		b.WriteString(m.renderOverview())
	}
	return b.String()
}

func (m *Model) summaryStrip() string {
	state := m.details.State
	if state == "" {
		state = api.ConnectorStateUnassigned
	}
	running := len(m.details.Tasks) - failedTaskCount(m.details.Tasks)
	failed := failedTaskCount(m.details.Tasks)
	tasks := fmt.Sprintf("Tasks: %d/%d", running, len(m.details.Tasks))
	if failed > 0 {
		tasks = m.common.Styles.Error.Render(fmt.Sprintf("Tasks: %d/%d (%d failed)", running, len(m.details.Tasks), failed))
	}
	parts := []string{
		m.common.Styles.Header.Render(m.name),
		"Connect: " + m.connect,
		"Type: " + string(m.details.Type),
		stateStyle(m.common, state).Render(state),
		tasks,
	}
	if m.details.WorkerID != "" {
		parts = append(parts, "Worker: "+m.details.WorkerID)
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

func (m *Model) renderOverview() string {
	var b strings.Builder
	b.WriteString(m.common.Styles.Header.Render("Class") + "\n")
	b.WriteString(m.details.Class + "\n\n")
	if m.details.ConsumerGroup != "" {
		b.WriteString(m.common.Styles.Header.Render("Consumer Group") + "\n")
		b.WriteString(m.details.ConsumerGroup + "\n\n")
	}
	if strings.EqualFold(m.details.State, api.ConnectorStateFailed) && m.details.Trace != "" {
		b.WriteString(m.common.Styles.Error.Render("Error trace (worker "+m.details.WorkerID+")") + "\n")
		b.WriteString(m.details.Trace + "\n")
	}
	b.WriteString("\n" + m.common.Styles.Muted.Render("p pause • u resume • s stop • R restart • z reset offsets • ctrl+d delete"))
	return b.String()
}

func (m *Model) renderTasks() string {
	if len(m.details.Tasks) == 0 {
		return m.common.Styles.Muted.Render("No tasks reported for this connector.")
	}
	var b strings.Builder
	b.WriteString(stylesPkg.FrameTable(m.tasksTable.View()))
	b.WriteString("\n")
	if m.expandedTask >= 0 && m.expandedTask < len(m.details.Tasks) {
		tk := m.details.Tasks[m.expandedTask]
		b.WriteString(m.common.Styles.Header.Render(fmt.Sprintf("Task %d — %s (worker %s)", tk.ID, tk.State, tk.WorkerID)) + "\n")
		if tk.Trace != "" {
			b.WriteString(tk.Trace + "\n")
		} else {
			b.WriteString(m.common.Styles.Muted.Render("no error trace") + "\n")
		}
		b.WriteString(m.common.Styles.Muted.Render("enter/esc: collapse"))
	} else {
		b.WriteString(m.common.Styles.Muted.Render("enter: expand trace • t: restart task • T: restart all • f: restart failed"))
	}
	return b.String()
}

func (m *Model) renderConfig() string {
	var b strings.Builder
	if m.editing {
		if m.configHasMasked() {
			b.WriteString(m.common.Styles.StatusStyle.Warning.Render("⚠ Masked secrets (********) must be replaced with real values before saving.") + "\n\n")
		}
		b.WriteString(m.configEditor.View())
		b.WriteString("\n")
		b.WriteString(m.common.Styles.Muted.Render("ctrl+s: save • esc: cancel"))
		return b.String()
	}
	if m.configHasMasked() {
		b.WriteString(m.common.Styles.StatusStyle.Warning.Render("⚠ Secret values are masked (********).") + "\n\n")
	}
	b.WriteString(m.configJSON())
	b.WriteString("\n\n")
	b.WriteString(m.common.Styles.Muted.Render("e: edit config"))
	return b.String()
}

func (m *Model) renderTopics() string {
	if len(m.details.Topics) == 0 {
		return m.common.Styles.Muted.Render("No topics associated with this connector.")
	}
	topics := append([]string(nil), m.details.Topics...)
	sort.Strings(topics)
	var b strings.Builder
	for _, t := range topics {
		b.WriteString("• " + t + "\n")
	}
	b.WriteString("\n" + m.common.Styles.Muted.Render("enter: open first topic"))
	return b.String()
}

// failedTaskCount counts tasks in the FAILED state.
func failedTaskCount(tasks []api.ConnectorTask) int {
	n := 0
	for _, t := range tasks {
		if strings.EqualFold(t.State, api.ConnectorStateFailed) {
			n++
		}
	}
	return n
}

// asConnectorNotFound reports whether err is (or wraps) an api.ConnectorNotFoundError.
func asConnectorNotFound(err error, dst *api.ConnectorNotFoundError) bool {
	if nf, ok := err.(api.ConnectorNotFoundError); ok {
		*dst = nf
		return true
	}
	return false
}
