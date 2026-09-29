package mainpage

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
)

// handleMutationResult handles form submit/cancel and the results of
// create, delete and alter operations. Unknown messages return nil.
func (k *KafuiContentProvider) handleMutationResult(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case form.FormSubmitMsg:
		switch {
		case k.showACLForm:
			return k.handleACLFormSubmit(msg.Values)
		case k.showACLSyncForm:
			k.showACLSyncForm = false
			k.aclSyncForm = nil
			return k.handleACLSyncSubmit(msg.Values["path"])
		case k.showQuotaForm:
			return k.handleQuotaFormSubmit(msg.Values)
		case k.showConnectForm:
			return k.handleConnectorFormSubmit(msg.Values)
		default:
			return k.handleTopicFormSubmit(msg.Values)
		}

	case form.FormCancelMsg:
		k.showTopicForm = false
		k.topicForm = nil
		k.showACLForm = false
		k.aclForm = nil
		k.showACLSyncForm = false
		k.aclSyncForm = nil
		k.showQuotaForm = false
		k.quotaForm = nil
		k.showConnectForm = false
		k.connectForm = nil

	case connectorCreatedMsg:
		return k.handleConnectorCreated(msg)

	case topicCloneDefaultsMsg:
		return k.openTopicFormWith(msg.defaults)

	case topicCreatedMsg:
		if msg.err != nil {
			// Keep the form open so the user can correct the input.
			return core.NotifyError("Create topic failed", msg.err)
		}
		k.showTopicForm = false
		k.topicForm = nil
		return tea.Batch(
			core.NewNotification(core.StatusSuccess, "Topic created", msg.name),
			k.loadCurrentResource(),
			func() tea.Msg {
				return NavigateToResourceDetailMsg{ResourceType: TopicResourceType, ResourceID: msg.name}
			},
		)

	case topicDeletedMsg:
		if msg.err != nil {
			return func() tea.Msg { return shared.NewUIError("delete-topic", "Delete topic failed", msg.err) }
		}
		return tea.Batch(core.NewNotification(core.StatusSuccess, "Topic deleted", msg.name), k.loadCurrentResource())

	case topicRecreatedMsg:
		if msg.err != nil {
			return func() tea.Msg { return shared.NewUIError("recreate-topic", "Recreate topic failed", msg.err) }
		}
		return tea.Batch(core.NewNotification(core.StatusSuccess, "Topic recreated", msg.name), k.refreshCurrentResource())

	case topicPurgedMsg:
		if msg.err != nil {
			return func() tea.Msg { return shared.NewUIError("purge-topic", "Clear messages failed", msg.err) }
		}
		return tea.Batch(core.NewNotification(core.StatusSuccess, "Messages cleared", msg.name), k.refreshCurrentResource())

	case topicBatchResultMsg:
		k.clearTopicSelection()
		if len(msg.failures) > 0 {
			summary := fmt.Errorf("%d of %d topics failed: %s", len(msg.failures), msg.total, strings.Join(msg.failures, "; "))
			return tea.Batch(
				func() tea.Msg {
					return shared.NewUIError("batch-"+msg.action, "Batch "+msg.action+" completed with errors", summary)
				},
				k.refreshCurrentResource(),
			)
		}
		return tea.Batch(
			core.NewNotification(core.StatusSuccess, "Batch "+msg.action+" complete", fmt.Sprintf("%d topics", msg.total)),
			k.refreshCurrentResource(),
		)

	case groupDeletedMsg:
		if msg.err != nil {
			return func() tea.Msg { return shared.NewUIError("delete-group", "Delete consumer group failed", msg.err) }
		}
		return tea.Batch(
			core.NewNotification(core.StatusSuccess, "Consumer group deleted", msg.groupID),
			k.loadCurrentResource(),
		)

	case aclDeletedMsg:
		if msg.err != nil {
			return func() tea.Msg { return shared.NewUIError("delete-acl", "Delete ACL failed", msg.err) }
		}
		return tea.Batch(core.NewNotification(core.StatusSuccess, "ACL deleted", msg.summary), k.loadCurrentResource())

	case aclCreatedMsg:
		if msg.err != nil {
			// Validation/expansion error — keep the form open to correct input.
			return core.NotifyError("Create ACL failed", msg.err)
		}
		k.showACLForm = false
		k.aclForm = nil
		if len(msg.failures) > 0 {
			summary := fmt.Errorf("%d created, %d failed: %s", msg.created, len(msg.failures), strings.Join(msg.failures, "; "))
			return tea.Batch(
				func() tea.Msg { return shared.NewUIError("create-acl", "Some ACL bindings failed", summary) },
				k.loadCurrentResource(),
			)
		}
		return tea.Batch(
			core.NewNotification(core.StatusSuccess, "ACLs created", fmt.Sprintf("%d binding(s)", msg.created)),
			k.loadCurrentResource(),
		)

	case aclSyncedMsg:
		if msg.err != nil {
			return func() tea.Msg { return shared.NewUIError("sync-acl", "ACL sync failed", msg.err) }
		}
		return tea.Batch(
			core.NewNotification(core.StatusSuccess, "ACLs synced", fmt.Sprintf("%d created, %d deleted", msg.created, msg.deleted)),
			k.loadCurrentResource(),
		)

	case quotaAlteredMsg:
		if msg.err != nil {
			// Validation error — keep the form open to correct input.
			return core.NotifyError("Quota update failed", msg.err)
		}
		k.showQuotaForm = false
		k.quotaForm = nil
		k.quotaEditEntity = nil
		return tea.Batch(
			core.NewNotification(core.StatusSuccess, "Client quota "+msg.action, ""),
			k.loadCurrentResource(),
		)
	}
	return nil
}
