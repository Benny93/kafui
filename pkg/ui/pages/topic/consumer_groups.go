package topic

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TopicGroupsLoadedMsg carries the result of GetConsumerGroupsForTopic (CG-21).
type TopicGroupsLoadedMsg struct {
	Topic  string
	Groups []api.ConsumerGroup
	Err    error
}

// handleShowGroups opens the consumer-groups overlay and fetches the groups
// related to this topic. The fetch is explicit (never automatic) because it
// fans out across group coordinators.
func (k *Keys) handleShowGroups(model *Model) tea.Cmd {
	model.showGroups = true
	model.groupsLoading = true
	model.groupsCursor = 0
	model.markRenderDirty()
	ds := model.dataSource
	topic := model.topicName
	return func() tea.Msg {
		groups, err := ds.GetConsumerGroupsForTopic(topic)
		return TopicGroupsLoadedMsg{Topic: topic, Groups: groups, Err: err}
	}
}

// handleGroupsOverlayKey handles keys while the consumer-groups overlay is open.
func (k *Keys) handleGroupsOverlayKey(model *Model, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "q":
		model.showGroups = false
		model.markRenderDirty()
		return nil
	case "up", "k":
		if model.groupsCursor > 0 {
			model.groupsCursor--
			model.markRenderDirty()
		}
		return nil
	case "down", "j":
		if model.groupsCursor < len(model.groups)-1 {
			model.groupsCursor++
			model.markRenderDirty()
		}
		return nil
	case "enter":
		if model.groupsCursor >= 0 && model.groupsCursor < len(model.groups) {
			id := model.groups[model.groupsCursor].Name
			return func() tea.Msg {
				return core.PageChangeMsg{PageID: "consumer_group:" + id, Data: map[string]interface{}{"groupID": id}}
			}
		}
		return nil
	}
	return nil
}

// handleTopicGroupsLoaded stores the fetched groups.
func (h *Handlers) handleTopicGroupsLoaded(model *Model, msg TopicGroupsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Topic != model.topicName {
		return model, nil
	}
	model.groupsLoading = false
	if msg.Err != nil {
		model.groups = nil
		model.statusMessage = "Failed to load consumer groups: " + msg.Err.Error()
	} else {
		model.groups = msg.Groups
	}
	if model.groupsCursor >= len(model.groups) {
		model.groupsCursor = 0
	}
	model.markRenderDirty()
	return model, nil
}

// renderGroupsOverlay renders the consumer-groups overlay for the topic.
func (m *Model) renderGroupsOverlay(width int) string {
	styles := m.common.Styles
	var b strings.Builder
	b.WriteString(styles.Header.Render("Consumer groups for " + m.topicName))
	b.WriteString("\n\n")
	if m.groupsLoading {
		b.WriteString(styles.Muted.Render("Loading consumer groups… (fans out across coordinators)"))
		return b.String()
	}
	if len(m.groups) == 0 {
		b.WriteString(styles.Muted.Render("No consumer groups are consuming this topic."))
		b.WriteString("\n\n")
		b.WriteString(styles.Muted.Render("esc: close"))
		return b.String()
	}
	header := fmt.Sprintf("  %-32s %-16s %-6s %-10s %-8s", "Group", "State", "Coord", "Assignor", "Lag")
	b.WriteString(styles.Muted.Render(header))
	b.WriteString("\n")
	for i, g := range m.groups {
		lag := "—"
		if g.Lag != nil {
			lag = strconv.FormatInt(*g.Lag, 10)
		}
		coord := "—"
		if g.CoordinatorID >= 0 {
			coord = strconv.FormatInt(int64(g.CoordinatorID), 10)
		}
		line := fmt.Sprintf("  %-32s %-16s %-6s %-10s %-8s", truncate(g.Name, 32), g.State, coord, g.PartitionAssignor, lag)
		if i == m.groupsCursor {
			line = lipgloss.NewStyle().Foreground(stylesPkg.BgBase).Background(stylesPkg.Primary).Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(styles.Muted.Render("↑/↓: select • enter: open group • esc: close"))
	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n < 2 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
