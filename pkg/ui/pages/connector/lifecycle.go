// Package connector implements the connector detail page (dynamic page ID
// "connector:<connect>:<name>"). It renders a summary strip plus four tabs —
// Overview, Tasks, Config and Topics — over the shared template shell. The page
// is created by the router; see NewModelWithCommon for the constructor the
// router wires to the "connector:<connect>:<name>" dynamic ID.
package connector

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
)

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
