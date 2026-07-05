package topic

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	formpkg "github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EditConfigLoadedMsg carries config for prefilling the edit form (TP-25).
type EditConfigLoadedMsg struct {
	Topic   string
	Entries []api.TopicConfigEntry
	Err     error
}

// settingsUpdatedMsg reports the outcome of an UpdateTopicConfig call.
type settingsUpdatedMsg struct {
	Topic string
	Count int
	Err   error
}

// wellKnownEditKeys are the config keys given dedicated (non-custom) form fields.
var wellKnownEditKeys = map[string]bool{
	"cleanup.policy":      true,
	"retention.ms":        true,
	"retention.bytes":     true,
	"max.message.bytes":   true,
	"min.insync.replicas": true,
}

// handleShowSettingsEdit opens the edit-settings form overlay and fetches config.
func (k *Keys) handleShowSettingsEdit(model *Model) tea.Cmd {
	model.showSettingsEdit = true
	model.settingsForm = nil
	model.markRenderDirty()
	ds := model.dataSource
	topic := model.topicName
	return func() tea.Msg {
		entries, err := ds.GetTopicConfig(topic)
		return EditConfigLoadedMsg{Topic: topic, Entries: entries, Err: err}
	}
}

// buildSettingsForm builds an edit form prefilled from the loaded config.
func buildSettingsForm(entries []api.TopicConfigEntry) *formpkg.Form {
	byName := make(map[string]api.TopicConfigEntry, len(entries))
	for _, e := range entries {
		byName[e.Name] = e
	}

	fields := []formpkg.Field{}

	// cleanup.policy as a Select including the current value.
	policyOpts := []string{"delete", "compact", "compact,delete"}
	cur := "delete"
	if e, ok := byName["cleanup.policy"]; ok {
		cur = e.Value
	}
	if !contains2(policyOpts, cur) {
		policyOpts = append([]string{cur}, policyOpts...)
	}
	fields = append(fields, formpkg.Field{
		Name: "cleanup.policy", Label: "cleanup.policy", Type: formpkg.Select,
		Options: policyOpts, Default: cur,
	})

	for _, name := range []string{"retention.ms", "retention.bytes", "max.message.bytes", "min.insync.replicas"} {
		def := ""
		if e, ok := byName[name]; ok {
			def = e.Value
		}
		fields = append(fields, formpkg.Field{Name: name, Label: name, Type: formpkg.Numeric, Default: def})
	}

	// Custom (non-well-known, non-sensitive, non-read-only) overrides as text.
	for _, e := range entries {
		if wellKnownEditKeys[e.Name] || e.Sensitive || e.ReadOnly {
			continue
		}
		fields = append(fields, formpkg.Field{Name: e.Name, Label: e.Name, Type: formpkg.Text, Default: e.Value})
	}

	return formpkg.New(fields)
}

func contains2(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// diffConfigChanges builds the map of changed config entries to submit. Only
// entries whose submitted value differs from the loaded effective value are
// included; unchanged entries (still at their default) are excluded. (TP-25)
func diffConfigChanges(loaded []api.TopicConfigEntry, values map[string]string) map[string]*string {
	byName := make(map[string]api.TopicConfigEntry, len(loaded))
	for _, e := range loaded {
		byName[e.Name] = e
	}
	changes := map[string]*string{}
	for name, newVal := range values {
		e, known := byName[name]
		if known {
			if newVal == e.Value {
				continue // unchanged
			}
		} else if newVal == "" {
			continue // brand-new empty custom field: nothing to set
		}
		v := newVal
		changes[name] = &v
	}
	return changes
}

// handleEditConfigLoaded builds the form once config arrives.
func (h *Handlers) handleEditConfigLoaded(model *Model, msg EditConfigLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Topic != model.topicName {
		return model, nil
	}
	if msg.Err != nil {
		model.showSettingsEdit = false
		model.markRenderDirty()
		return model, core.NotifyError("Load config failed", msg.Err)
	}
	model.loadedConfig = msg.Entries
	model.settingsForm = buildSettingsForm(msg.Entries)
	if model.dimensions.Width > 0 {
		model.settingsForm.SetDimensions(model.dimensions.Width-4, model.dimensions.Height-6)
	}
	cmd := model.settingsForm.Focus()
	model.markRenderDirty()
	return model, cmd
}

// handleEditFormKey routes keys to the edit form while it is open.
func (k *Keys) handleEditFormKey(model *Model, msg tea.KeyMsg) tea.Cmd {
	if model.settingsForm == nil {
		if msg.String() == "esc" || msg.String() == "q" {
			model.showSettingsEdit = false
			model.markRenderDirty()
		}
		return nil
	}
	cmd, _ := model.settingsForm.Update(msg)
	model.markRenderDirty()
	return cmd
}

// handleSettingsFormSubmit diffs and applies the edit-form submission.
func (h *Handlers) handleSettingsFormSubmit(model *Model, values map[string]string) (tea.Model, tea.Cmd) {
	changes := diffConfigChanges(model.loadedConfig, values)
	model.showSettingsEdit = false
	model.settingsForm = nil
	model.markRenderDirty()
	if len(changes) == 0 {
		return model, core.NewNotification(core.StatusInfo, "No changes", "Topic settings unchanged")
	}
	ds := model.dataSource
	topic := model.topicName
	return model, func() tea.Msg {
		err := ds.UpdateTopicConfig(topic, changes)
		return settingsUpdatedMsg{Topic: topic, Count: len(changes), Err: err}
	}
}

// handleSettingsUpdated reports the outcome of the config update.
func (h *Handlers) handleSettingsUpdated(model *Model, msg settingsUpdatedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		return model, core.NotifyError("Update settings failed", msg.Err)
	}
	return model, core.NewNotification(core.StatusSuccess, "Settings updated",
		fmt.Sprintf("Applied %d change(s) to %s", msg.Count, msg.Topic))
}

// renderEditOverlay renders the edit-settings form (TP-25).
func (m *Model) renderEditOverlay(width int) string {
	muted := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)
	header := lipgloss.NewStyle().Foreground(stylesPkg.Primary).Bold(true)
	var b strings.Builder
	b.WriteString(header.Render("Edit settings: " + m.topicName))
	b.WriteString("\n")
	// Read-only header line: topic name + partition count are immutable here.
	b.WriteString(muted.Render(fmt.Sprintf("  topic: %s   partitions: %d (read-only)",
		m.topicName, m.topicDetails.NumPartitions)))
	b.WriteString("\n\n")
	if m.settingsForm == nil {
		b.WriteString(muted.Render("Loading configuration…"))
		return b.String()
	}
	b.WriteString(m.settingsForm.View())
	b.WriteString("\n")
	b.WriteString(muted.Render("tab: next field • enter: submit/next • esc: cancel"))
	return b.String()
}
