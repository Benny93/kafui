package topic

import (
	"fmt"
	"strconv"
	"strings"

	formpkg "github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// mutationKind identifies which input dialog is open (TP-26).
type mutationKind int

const (
	mutIncreasePartitions mutationKind = iota
	mutReplicationFactor
)

// topicMutationMsg reports the outcome of a topic mutation (TP-26/TP-27).
type topicMutationMsg struct {
	Action  string
	Detail  string
	Err     error
	Back    bool // navigate back to the topic list on success (delete)
	Refresh bool // refresh the overview on success (partitions / RF)
}

// isInternalTopic reports whether a topic is a Kafka-internal topic.
func isInternalTopic(name string) bool {
	return strings.HasPrefix(name, "__")
}

// --- TP-26: partition-increase and replication-factor dialogs ---

func (k *Keys) handleIncreasePartitionsDialog(model *Model) tea.Cmd {
	model.mutationKind = mutIncreasePartitions
	model.mutationForm = formpkg.New([]formpkg.Field{{
		Name: "value", Label: "New total partition count", Type: formpkg.Numeric, Required: true,
		Default: strconv.Itoa(int(model.topicDetails.NumPartitions) + 1),
	}})
	return k.openMutationForm(model)
}

func (k *Keys) handleReplicationFactorDialog(model *Model) tea.Cmd {
	model.mutationKind = mutReplicationFactor
	model.mutationForm = formpkg.New([]formpkg.Field{{
		Name: "value", Label: "New replication factor", Type: formpkg.Numeric, Required: true,
		Default: strconv.Itoa(int(model.topicDetails.ReplicationFactor)),
	}})
	return k.openMutationForm(model)
}

func (k *Keys) openMutationForm(model *Model) tea.Cmd {
	model.showMutationForm = true
	if model.dimensions.Width > 0 {
		model.mutationForm.SetDimensions(model.dimensions.Width-4, model.dimensions.Height-6)
	}
	cmd := model.mutationForm.Focus()
	model.markRenderDirty()
	return cmd
}

// handleMutationFormKey routes keys to the open mutation form.
func (k *Keys) handleMutationFormKey(model *Model, msg tea.KeyMsg) tea.Cmd {
	if model.mutationForm == nil {
		model.showMutationForm = false
		return nil
	}
	cmd, _ := model.mutationForm.Update(msg)
	model.markRenderDirty()
	return cmd
}

// handleMutationFormSubmit validates the input and asks for confirmation before
// calling the datasource. Cancel makes no datasource call.
func (h *Handlers) handleMutationFormSubmit(model *Model, values map[string]string) (tea.Model, tea.Cmd) {
	kind := model.mutationKind
	model.showMutationForm = false
	model.mutationForm = nil
	model.markRenderDirty()

	n, err := strconv.Atoi(strings.TrimSpace(values["value"]))
	if err != nil {
		return model, core.NotifyError("Invalid input", fmt.Errorf("%q is not a number", values["value"]))
	}
	ds := model.dataSource
	topic := model.topicName

	switch kind {
	case mutIncreasePartitions:
		return model, func() tea.Msg {
			return core.ShowConfirmMsg{
				Title:        "Increase partitions",
				Message:      fmt.Sprintf("Increase %q to %d partitions? Partition count cannot be reduced later and may reshuffle key ordering.", topic, n),
				ConfirmLabel: "Increase",
				OnConfirm: func() tea.Msg {
					e := ds.IncreasePartitions(topic, int32(n))
					return topicMutationMsg{Action: "Increase partitions", Detail: fmt.Sprintf("%s now has %d partitions", topic, n), Err: e, Refresh: true}
				},
			}
		}
	case mutReplicationFactor:
		return model, func() tea.Msg {
			return core.ShowConfirmMsg{
				Title:        "Change replication factor",
				Message:      fmt.Sprintf("Change replication factor of %q to %d? This triggers partition reassignment across brokers.", topic, n),
				ConfirmLabel: "Change",
				OnConfirm: func() tea.Msg {
					e := ds.ChangeReplicationFactor(topic, int16(n))
					return topicMutationMsg{Action: "Change replication factor", Detail: fmt.Sprintf("%s replication factor is now %d", topic, n), Err: e, Refresh: true}
				},
			}
		}
	}
	return model, nil
}

// --- TP-27: header actions ---

// handleClearAllMessages purges all messages after confirmation (ctrl+p).
func (k *Keys) handleClearAllMessages(model *Model) tea.Cmd {
	if isInternalTopic(model.topicName) {
		return core.NewNotification(core.StatusWarning, "Not allowed", "Cannot clear an internal topic")
	}
	ds := model.dataSource
	topic := model.topicName
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Clear all messages",
			Message:      fmt.Sprintf("Delete ALL messages in %q? This cannot be undone.", topic),
			Danger:       true,
			ConfirmLabel: "Clear",
			OnConfirm: func() tea.Msg {
				e := ds.PurgeTopicMessages(topic, -1)
				return topicMutationMsg{Action: "Clear messages", Detail: fmt.Sprintf("All messages cleared in %s", topic), Err: e}
			},
		}
	}
}

// confirmPurgePartition purges a single partition after confirmation (TP-27).
func (k *Keys) confirmPurgePartition(model *Model, partition int32) tea.Cmd {
	if isInternalTopic(model.topicName) {
		return core.NewNotification(core.StatusWarning, "Not allowed", "Cannot clear an internal topic")
	}
	ds := model.dataSource
	topic := model.topicName
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Clear partition",
			Message:      fmt.Sprintf("Delete all messages in partition %d of %q?", partition, topic),
			Danger:       true,
			ConfirmLabel: "Clear",
			OnConfirm: func() tea.Msg {
				e := ds.PurgeTopicMessages(topic, partition)
				return topicMutationMsg{Action: "Clear partition", Detail: fmt.Sprintf("Partition %d cleared in %s", partition, topic), Err: e}
			},
		}
	}
}

// handleRecreateTopic recreates the topic after confirmation (ctrl+r).
func (k *Keys) handleRecreateTopic(model *Model) tea.Cmd {
	if isInternalTopic(model.topicName) {
		return core.NewNotification(core.StatusWarning, "Not allowed", "Cannot recreate an internal topic")
	}
	ds := model.dataSource
	topic := model.topicName
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Recreate topic",
			Message:      fmt.Sprintf("Delete and recreate %q with the same config? All messages will be lost.", topic),
			Danger:       true,
			ConfirmLabel: "Recreate",
			OnConfirm: func() tea.Msg {
				e := ds.RecreateTopic(topic)
				return topicMutationMsg{Action: "Recreate topic", Detail: fmt.Sprintf("%s recreated", topic), Err: e}
			},
		}
	}
}

// handleDeleteTopic deletes the topic after confirmation, then returns to the
// list (ctrl+d). Disabled for internal topics or when deletion is disabled.
func (k *Keys) handleDeleteTopic(model *Model) tea.Cmd {
	if isInternalTopic(model.topicName) {
		return core.NewNotification(core.StatusWarning, "Not allowed", "Cannot delete an internal topic")
	}
	ds := model.dataSource
	topic := model.topicName
	return func() tea.Msg {
		if enabled, err := ds.IsTopicDeletionEnabled(); err == nil && !enabled {
			return core.NotificationMsg{Severity: core.StatusWarning, Title: "Not allowed",
				Message: "Topic deletion is disabled on this cluster"}
		}
		return core.ShowConfirmMsg{
			Title:        "Delete topic",
			Message:      fmt.Sprintf("Delete topic %q? This cannot be undone.", topic),
			Danger:       true,
			ConfirmLabel: "Delete",
			OnConfirm: func() tea.Msg {
				e := ds.DeleteTopic(topic)
				return topicMutationMsg{Action: "Delete topic", Detail: fmt.Sprintf("%s deleted", topic), Err: e, Back: true}
			},
		}
	}
}

// handleTopicMutation reports the outcome of a mutation and follows up.
func (h *Handlers) handleTopicMutation(model *Model, msg topicMutationMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		return model, core.NotifyError(msg.Action+" failed", msg.Err)
	}
	cmds := []tea.Cmd{core.NewNotification(core.StatusSuccess, msg.Action, msg.Detail)}
	if msg.Back {
		cmds = append(cmds, func() tea.Msg { return core.BackMsg{} })
	} else if msg.Refresh {
		cmds = append(cmds, fetchTopicOverview(model.dataSource, model.topicName))
	}
	return model, tea.Batch(cmds...)
}

// renderMutationOverlay renders the partition/replication input form (TP-26).
func (m *Model) renderMutationOverlay(width int) string {
	muted := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)
	header := lipgloss.NewStyle().Foreground(stylesPkg.Primary).Bold(true)
	title := "Increase partitions"
	current := fmt.Sprintf("current: %d partitions", m.topicDetails.NumPartitions)
	if m.mutationKind == mutReplicationFactor {
		title = "Change replication factor"
		current = fmt.Sprintf("current: RF %d", m.topicDetails.ReplicationFactor)
	}
	var b strings.Builder
	b.WriteString(header.Render(title + ": " + m.topicName))
	b.WriteString("\n")
	b.WriteString(muted.Render("  " + current))
	b.WriteString("\n\n")
	if m.mutationForm != nil {
		b.WriteString(m.mutationForm.View())
	}
	b.WriteString("\n")
	b.WriteString(muted.Render("enter: submit • esc: cancel"))
	return b.String()
}
