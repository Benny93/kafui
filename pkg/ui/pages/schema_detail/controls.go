package schemadetail

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/authz"
	"github.com/Benny93/kafui/pkg/ui/components/menu"
	"github.com/Benny93/kafui/pkg/ui/keys"
)

// KeyScope implements core.KeyScoper.
func (m *Model) KeyScope() keys.Scope { return keys.ScopeContent }

// ContextActions implements core.ActionProvider. Versions, diff, the picker and
// the compatibility check used to be bare v / d / c / ctrl+k with no help text.
func (m *Model) ContextActions() []menu.Entry {
	keyOf := func(a keys.Action) string { return keys.Default.KeyFor(a) }
	subject := m.subject

	gated := func(e menu.Entry, action authz.Action) menu.Entry {
		if m.common != nil && !m.common.Can(action, authz.ResourceSchema, subject) {
			e.Disabled = true
			e.Reason = "not permitted by the active profile or the cluster is read-only"
		}
		return e
	}

	return []menu.Entry{
		{Label: "Browse versions", Run: func() tea.Cmd { return m.enterVersions() }},
		{Label: "Diff against another version", Run: func() tea.Cmd { return m.enterDiffFromContent() }},
		{Label: "Copy schema", Key: keyOf(keys.ActionCopy), Run: func() tea.Cmd {
			m.CopyToClipboard()
			return nil
		}},
		gated(menu.Entry{Label: "Register a new version…", Key: keyOf(keys.ActionNew), Run: func() tea.Cmd {
			m.enterRegister()
			return nil
		}}, authz.ActionCreate),
		gated(menu.Entry{Label: "Run a compatibility check", Run: func() tea.Cmd {
			m.enterRegister()
			return m.checkOnlyCmd()
		}}, authz.ActionCreate),
		gated(menu.Entry{Label: "Set the compatibility level…", Run: func() tea.Cmd {
			m.enterPicker()
			return nil
		}}, authz.ActionModifyCompat),
		gated(menu.Entry{Label: "Delete subject", Key: keyOf(keys.ActionDelete), Destructive: true,
			Run: func() tea.Cmd { return m.confirmDeleteSubjectCmd() }}, authz.ActionDelete),
	}
}

// The router holds the page wrapper, not the inner model, so the wrapper must
// forward the controls-spec interfaces. Without this the actions menu falls
// back to its empty placeholder — which is exactly what the re-recorded demo
// caught.

// ContextActions implements core.ActionProvider.
func (p *SchemaDetailPageModel) ContextActions() []menu.Entry {
	if p.model == nil {
		return nil
	}
	return p.model.ContextActions()
}

// KeyScope implements core.KeyScoper.
func (p *SchemaDetailPageModel) KeyScope() keys.Scope { return keys.ScopeContent }
