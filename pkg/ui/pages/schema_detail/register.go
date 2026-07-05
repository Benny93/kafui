package schemadetail

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/ui/components/editor"
	"github.com/Benny93/kafui/pkg/ui/core"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// enterRegister opens the editor seeded with the current schema text (SR-16).
func (m *Model) enterRegister() {
	m.mode = modeRegister
	m.registerSeed = m.content
	m.editor = editor.NewEditor(m.content)
}

// validateRegister performs client-side pre-validation. Returns "" when valid.
func (m *Model) validateRegister(text string) string {
	if strings.TrimSpace(text) == "" {
		return "Schema text is empty"
	}
	if text == m.registerSeed {
		return "Schema unchanged from the current version"
	}
	if isJSONType(m.GetSchemaType()) {
		var v interface{}
		if err := json.Unmarshal([]byte(text), &v); err != nil {
			return "Invalid JSON: " + err.Error()
		}
	}
	return ""
}

// checkThenRegisterCmd runs a compatibility check and only registers when the
// candidate is compatible (SR-16/SR-17). Guarantees Check is called before
// Register.
func (m *Model) checkThenRegisterCmd(text string) tea.Cmd {
	subject, typ, ds := m.subject, m.schemaType, m.dataSource
	return func() tea.Msg {
		compatible, messages, err := ds.CheckSchemaCompatibility(subject, text, typ)
		if err != nil {
			return SchemaRegisterResultMsg{Err: err}
		}
		if !compatible {
			return SchemaRegisterResultMsg{Incompatible: true, Messages: messages}
		}
		schema, err := ds.RegisterSchema(subject, text, typ)
		return SchemaRegisterResultMsg{Schema: schema, Err: err}
	}
}

// checkOnlyCmd runs a standalone compatibility check without registering (SR-17).
func (m *Model) checkOnlyCmd() tea.Cmd {
	text := m.content
	if m.editor != nil {
		text = m.editor.Value()
	}
	subject, typ, ds := m.subject, m.schemaType, m.dataSource
	return func() tea.Msg {
		compatible, messages, err := ds.CheckSchemaCompatibility(subject, text, typ)
		return SchemaCheckResultMsg{Compatible: compatible, Messages: messages, Err: err}
	}
}

// checkResultCmd maps a standalone check result to a notification (SR-17).
func checkResultCmd(msg SchemaCheckResultMsg) tea.Cmd {
	if msg.Err != nil {
		return core.NotifyError("Compatibility check", msg.Err)
	}
	if msg.Compatible {
		return core.NewNotification(core.StatusSuccess, "Compatibility check", "Schema is compatible")
	}
	return core.NewNotification(core.StatusWarning, "Compatibility check",
		"Incompatible: "+strings.Join(msg.Messages, "; "))
}

func handleRegisterKey(m *Model, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.mode = modeContent
		m.editor = nil
		return nil
	case "ctrl+s":
		text := m.editor.Value()
		if reason := m.validateRegister(text); reason != "" {
			return core.NewNotification(core.StatusWarning, "Register schema", reason)
		}
		return m.checkThenRegisterCmd(text)
	case "ctrl+k":
		return m.checkOnlyCmd()
	default:
		if m.editor != nil {
			_, cmd := m.editor.Update(msg)
			return cmd
		}
		return nil
	}
}

// handleRegisterResult applies a register outcome (SR-16). On success it reloads
// the latest version; on failure it keeps the editor open with a status message.
func (p *SchemaDetailPageModel) handleRegisterResult(msg SchemaRegisterResultMsg) tea.Cmd {
	m := p.model
	switch {
	case msg.Err != nil:
		return core.NotifyError("Register schema", msg.Err)
	case msg.Incompatible:
		return core.NewNotification(core.StatusWarning, "Register schema",
			"Incompatible: "+strings.Join(msg.Messages, "; "))
	default:
		m.mode = modeContent
		m.editor = nil
		m.version = msg.Schema.Version
		m.schemaID = msg.Schema.ID
		if msg.Schema.SchemaType != "" {
			m.schemaType = msg.Schema.SchemaType
		}
		m.loading = true
		m.versionsLoaded = false
		return tea.Batch(
			m.LoadContentAsync(),
			m.loadVersionsCmd(),
			m.loadMetaCmd(),
			core.NewNotification(core.StatusSuccess, "Register schema",
				fmt.Sprintf("Registered version %d", msg.Schema.Version)),
		)
	}
}

func renderRegister(m *Model, width, height int) string {
	titleStyle := lipgloss.NewStyle().Foreground(stylesPkg.Primary).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)

	header := titleStyle.Render(fmt.Sprintf("Register new version of %s (%s)", m.subject, m.GetSchemaType()))
	var body string
	if m.editor != nil {
		m.editor.SetDimensions(width, height-3)
		body = m.editor.View()
	}
	hint := mutedStyle.Render("ctrl+s check & register · ctrl+k check only · esc cancel")
	return strings.Join([]string{header, body, hint}, "\n")
}
