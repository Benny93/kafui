package topic

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TopicConfigLoadedMsg carries the result of GetTopicConfig (TP-24).
type TopicConfigLoadedMsg struct {
	Topic   string
	Entries []api.TopicConfigEntry
	Err     error
}

// fetchTopicConfig loads the effective config for a topic.
func fetchTopicConfig(ds api.KafkaDataSource, topic string) tea.Cmd {
	return func() tea.Msg {
		entries, err := ds.GetTopicConfig(topic)
		return TopicConfigLoadedMsg{Topic: topic, Entries: entries, Err: err}
	}
}

// handleShowSettings opens the settings/config overlay and fetches the config.
func (k *Keys) handleShowSettings(model *Model) tea.Cmd {
	model.showSettings = true
	model.settingsLoading = true
	model.settingsErr = nil
	model.markRenderDirty()
	return fetchTopicConfig(model.dataSource, model.topicName)
}

// handleSettingsKey handles keys while the settings overlay is open.
func (k *Keys) handleSettingsKey(model *Model, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "q":
		model.showSettings = false
		model.markRenderDirty()
		return nil
	case "r":
		return k.handleShowSettings(model)
	case "E":
		// Jump straight to the edit form.
		model.showSettings = false
		return k.handleShowSettingsEdit(model)
	}
	return nil
}

// handleTopicConfigLoaded stores the fetched config.
func (h *Handlers) handleTopicConfigLoaded(model *Model, msg TopicConfigLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Topic != model.topicName {
		return model, nil
	}
	model.settingsLoading = false
	if msg.Err != nil {
		model.settingsConfig = nil
		model.settingsErr = msg.Err
	} else {
		model.settingsConfig = msg.Entries
		model.settingsErr = nil
	}
	model.markRenderDirty()
	return model, nil
}

// isOverride reports whether an entry overrides its default value.
func isOverride(e api.TopicConfigEntry) bool {
	if e.Default == "" {
		return false
	}
	return e.Value != e.Default
}

// renderSettingsOverlay renders the topic config rows (TP-24).
func (m *Model) renderSettingsOverlay(width int) string {
	muted := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)
	header := lipgloss.NewStyle().Foreground(stylesPkg.Primary).Bold(true)
	override := lipgloss.NewStyle().Foreground(stylesPkg.Warning).Bold(true)
	base := lipgloss.NewStyle().Foreground(stylesPkg.FgBase)
	var b strings.Builder

	b.WriteString(header.Render("Settings: " + m.topicName))
	b.WriteString("\n\n")

	if m.settingsLoading {
		b.WriteString(muted.Render("Loading configuration…"))
		return b.String()
	}
	if m.settingsErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(stylesPkg.Error).Bold(true)
		b.WriteString(errStyle.Render("Failed to load config: " + m.settingsErr.Error()))
		b.WriteString("\n\n")
		b.WriteString(muted.Render("r: retry • esc: close"))
		return b.String()
	}
	if len(m.settingsConfig) == 0 {
		// Empty list is a permission case, not an error.
		b.WriteString(muted.Render("No configuration entries are visible (insufficient permissions?)."))
		b.WriteString("\n\n")
		b.WriteString(muted.Render("esc: close"))
		return b.String()
	}

	b.WriteString(muted.Render(configRow("NAME", "VALUE", "DEFAULT")))
	b.WriteString("\n")
	for _, e := range m.settingsConfig {
		val := formatSettingValue(e.Name, e.Value, e.Sensitive)
		def := e.Default
		if e.Sensitive {
			def = shared.ConfigValueMask
		} else if def != "" {
			def = formatSettingValue(e.Name, e.Default, false)
		} else {
			def = "—"
		}
		row := configRow(e.Name, val, def)
		if isOverride(e) {
			b.WriteString(override.Render(row))
		} else {
			b.WriteString(base.Render(row))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(muted.Render("E: edit • r: refresh • esc: close"))
	return b.String()
}

// formatSettingValue renders a config value: sensitive masked, ms/bytes as a
// human-readable equivalent, and non-positive ms/bytes shown as "unbounded".
func formatSettingValue(name, value string, sensitive bool) string {
	if sensitive {
		return shared.ConfigValueMask
	}
	if strings.HasSuffix(name, ".ms") || strings.HasSuffix(name, ".bytes") {
		if n, err := strconv.ParseInt(value, 10, 64); err == nil && n <= 0 {
			return "unbounded"
		}
	}
	return shared.FormatConfigValue(name, value, false)
}

func configRow(name, value, def string) string {
	return fmt.Sprintf("  %-34s %-26s %-24s", truncate(name, 34), truncate(value, 26), truncate(def, 24))
}
