// Package connector implements the connector detail page (dynamic page ID
// "connector:<connect>:<name>"). It renders a summary strip plus four tabs —
// Overview, Tasks, Config and Topics — over the shared template shell. The page
// is created by the router; see NewModelWithCommon for the constructor the
// router wires to the "connector:<connect>:<name>" dynamic ID.
package connector

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components/tabstrip"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	"github.com/charmbracelet/lipgloss"
)

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

// tabBar renders the shared, click-and-hover-aware tab strip.
func (m *Model) tabBar() string {
	m.tabs().SetActive(int(m.active))
	return m.tabs().View()
}

// tabs lazily builds this screen's tab strip. Its zone ids must stay stable
// across renders, so the strip is created once and reused.
func (m *Model) tabs() *tabstrip.Model {
	if m.tabStrip == nil {
		titles := make([]string, 0, len(tabTitles))
		for _, t := range tabTitles {
			titles = append(titles, t.String())
		}
		m.tabStrip = tabstrip.New("connector", titles)
	}
	return m.tabStrip
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
	b.WriteString("\n" + m.common.Styles.Muted.Render(keys.Hint(keys.ScopeConnector, keys.ActionPause, "pause/resume", keys.ActionDelete, "delete")+"  (restart, stop and offset reset are in the actions menu)"))
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
