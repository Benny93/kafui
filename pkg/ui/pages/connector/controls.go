package connector

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/authz"
	"github.com/Benny93/kafui/pkg/ui/components/menu"
	"github.com/Benny93/kafui/pkg/ui/keys"
)

// KeyScope implements core.KeyScoper.
func (m *Model) KeyScope() keys.Scope { return keys.ScopeConnector }

// Unwind implements core.Unwinder: esc leaves the config editor or collapses an
// expanded task before the shell navigates back to the connector list.
func (m *Model) Unwind() (tea.Cmd, bool) {
	switch {
	case m.editing:
		m.editing = false
	case m.expandedTask >= 0:
		m.expandedTask = -1
	default:
		return nil, false
	}
	return nil, true
}

// ContextActions implements core.ActionProvider. The connector lifecycle used
// to live on p / u / s / R / z / t / T / f — eight bare letters, none of which
// appeared in the help overlay.
func (m *Model) ContextActions() []menu.Entry {
	keyOf := func(a keys.Action) string { return keys.Default.KeyFor(a) }
	name := m.details.Name

	gated := func(e menu.Entry, action authz.Action) menu.Entry {
		if m.common != nil && !m.common.Can(action, authz.ResourceConnector, name) {
			e.Disabled = true
			e.Reason = "not permitted by the active profile or the cluster is read-only"
		}
		return e
	}

	paused := strings.EqualFold(m.details.State, api.ConnectorStatePaused)
	pauseLabel := "Pause connector"
	if paused {
		pauseLabel = "Resume connector"
	}

	entries := []menu.Entry{
		gated(menu.Entry{Label: pauseLabel, Key: keyOf(keys.ActionPause), Run: func() tea.Cmd {
			if paused {
				return m.lifecycle("resume", m.common.DataSource.ResumeConnector)
			}
			return m.lifecycle("pause", m.common.DataSource.PauseConnector)
		}}, authz.ActionPause),
		gated(menu.Entry{Label: "Restart connector", Run: func() tea.Cmd {
			return m.lifecycle("restart", m.common.DataSource.RestartConnector)
		}}, authz.ActionRestart),
		gated(menu.Entry{Label: "Stop connector", Run: func() tea.Cmd {
			return m.lifecycle("stop", m.common.DataSource.StopConnector)
		}}, authz.ActionPause),
		gated(menu.Entry{Label: "Restart the selected task", Run: func() tea.Cmd {
			return m.restartSelectedTask()
		}}, authz.ActionRestart),
		gated(menu.Entry{Label: "Restart all tasks", Run: func() tea.Cmd {
			return m.restartTasks("all", func(api.ConnectorTask) bool { return true })
		}}, authz.ActionRestart),
		gated(menu.Entry{Label: "Restart failed tasks", Run: func() tea.Cmd {
			return m.restartTasks("failed", func(tk api.ConnectorTask) bool {
				return strings.EqualFold(tk.State, api.ConnectorStateFailed)
			})
		}}, authz.ActionRestart),
		gated(menu.Entry{Label: "Edit configuration", Key: keyOf(keys.ActionEdit), Run: func() tea.Cmd {
			return m.beginConfigEdit()
		}}, authz.ActionEdit),
		gated(menu.Entry{Label: "Reset offsets", Destructive: true, Run: func() tea.Cmd {
			return m.resetOffsets()
		}}, authz.ActionResetOffsets),
		gated(menu.Entry{Label: "Delete connector", Key: keyOf(keys.ActionDelete), Destructive: true,
			Run: func() tea.Cmd { return m.deleteConnector() }}, authz.ActionDelete),
		{Label: "Refresh", Key: keyOf(keys.ActionRefresh), Run: func() tea.Cmd { return m.retry() }},
	}
	return entries
}
