package schemadetail

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/ui/core"
	tea "github.com/charmbracelet/bubbletea"
)

// confirmDeleteSubjectCmd asks for confirmation, then soft-deletes all versions
// of the subject (SR-18). The datasource call runs only after confirmation.
func (m *Model) confirmDeleteSubjectCmd() tea.Cmd {
	subject, ds := m.subject, m.dataSource
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Delete subject",
			Message:      fmt.Sprintf("Delete ALL versions of %q? (soft delete)", subject),
			Danger:       true,
			ConfirmLabel: "Delete",
			OnConfirm: func() tea.Msg {
				_, err := ds.DeleteSubject(subject, false)
				return SchemaDeleteResultMsg{Err: err, BackToList: err == nil}
			},
		}
	}
}

// confirmDeleteVersionCmd asks for confirmation, then soft-deletes one version
// (SR-18).
func (m *Model) confirmDeleteVersionCmd(version int) tea.Cmd {
	subject, ds := m.subject, m.dataSource
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Delete version",
			Message:      fmt.Sprintf("Delete version %d of %q? (soft delete)", version, subject),
			Danger:       true,
			ConfirmLabel: "Delete",
			OnConfirm: func() tea.Msg {
				err := ds.DeleteSchemaVersion(subject, version, false)
				return SchemaDeleteResultMsg{Err: err}
			},
		}
	}
}

// handleDeleteResult applies a delete outcome (SR-18).
func (p *SchemaDetailPageModel) handleDeleteResult(msg SchemaDeleteResultMsg) tea.Cmd {
	m := p.model
	if msg.Err != nil {
		return core.NotifyError("Delete", msg.Err)
	}
	if msg.BackToList {
		return tea.Batch(
			core.NewNotification(core.StatusSuccess, "Delete subject", "Subject deleted"),
			func() tea.Msg { return core.BackMsg{} },
		)
	}
	// Version deleted — refresh the version list in place.
	m.versionsLoaded = false
	m.versionCursor = 0
	return tea.Batch(
		m.loadVersionsCmd(),
		core.NewNotification(core.StatusSuccess, "Delete version", "Version deleted"),
	)
}
