package consumergroup

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
)

// deleteGroup (CG-17) shows a confirmation modal, then deletes the group. On
// success it navigates back to the group list; failures surface as a UIError.
func (m *Model) deleteGroup() tea.Cmd {
	ds := m.common.DataSource
	id := m.groupID
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Delete consumer group",
			Message:      fmt.Sprintf("Delete consumer group %q? This cannot be undone.", id),
			Danger:       true,
			ConfirmLabel: "Delete",
			OnConfirm: func() tea.Msg {
				err := ds.DeleteConsumerGroup(id)
				return groupDeletedMsg{groupID: id, err: err}
			},
		}
	}
}

func (m *Model) handleGroupDeleted(v groupDeletedMsg) tea.Cmd {
	if v.err != nil {
		return func() tea.Msg { return shared.NewUIError("delete-group", "Delete consumer group failed", v.err) }
	}
	return tea.Batch(
		core.NewNotification(core.StatusSuccess, "Consumer group deleted", v.groupID),
		func() tea.Msg { return core.BackMsg{} },
	)
}

// deleteSelectedTopicOffsets (CG-18) confirms then deletes the committed offsets
// of the highlighted topic; on success the detail is re-fetched so the topic
// disappears from the breakdown while other topics remain.
func (m *Model) deleteSelectedTopicOffsets() tea.Cmd {
	tr, ok := m.selectedTopicRow()
	if !ok {
		return nil
	}
	ds := m.common.DataSource
	id := m.groupID
	topic := tr.topic
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Delete committed offsets",
			Message:      fmt.Sprintf("Delete committed offsets of topic %q for group %q?", topic, id),
			Danger:       true,
			ConfirmLabel: "Delete",
			OnConfirm: func() tea.Msg {
				err := ds.DeleteConsumerGroupOffsets(id, topic)
				return offsetsDeletedMsg{groupID: id, topic: topic, err: err}
			},
		}
	}
}

func (m *Model) handleOffsetsDeleted(v offsetsDeletedMsg) tea.Cmd {
	if v.err != nil {
		return func() tea.Msg { return shared.NewUIError("delete-offsets", "Delete offsets failed", v.err) }
	}
	m.expanded = -1
	return tea.Batch(
		core.NewNotification(core.StatusSuccess, "Offsets deleted", v.topic),
		m.loadDetail(),
	)
}
