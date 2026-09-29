// Package connector implements the connector detail page (dynamic page ID
// "connector:<connect>:<name>"). It renders a summary strip plus four tabs —
// Overview, Tasks, Config and Topics — over the shared template shell. The page
// is created by the router; see NewModelWithCommon for the constructor the
// router wires to the "connector:<connect>:<name>" dynamic ID.
package connector

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

func taskColumns() []table.Column {
	return []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Worker", Width: 20},
		{Title: "State", Width: 14},
		{Title: "Trace", Width: 40},
	}
}

// --- core.Page ---

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
		b.WriteString(m.common.Styles.Muted.Render(keys.Hint(keys.ScopeConnector, keys.ActionActivate, "expand trace") + "  (task restarts are in the actions menu)"))
	}
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
