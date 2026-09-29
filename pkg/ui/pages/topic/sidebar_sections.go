package topic

import (
	"fmt"
	"github.com/Benny93/kafui/pkg/ui/keys"
	"sort"

	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	tea "github.com/charmbracelet/bubbletea"
)

// TopicInfoSection provides topic information for the sidebar
type TopicInfoSection struct {
	model *Model
}

func NewTopicInfoSection(model *Model) *TopicInfoSection {
	return &TopicInfoSection{
		model: model,
	}
}

func (t *TopicInfoSection) GetTitle() string {
	return "TOPIC INFO"
}

func (t *TopicInfoSection) RenderItems(maxItems, width int) []providers.SidebarItem {
	items := []providers.SidebarItem{
		{
			Icon:   "📝",
			Text:   "Name",
			Value:  t.model.topicName,
			Status: "info",
		},
		{
			Icon:   "🔢",
			Text:   "Partitions",
			Value:  fmt.Sprintf("%d", t.model.topicDetails.NumPartitions),
			Status: "info",
		},
		{
			Icon:   "🔄",
			Text:   "Replication",
			Value:  fmt.Sprintf("%d", t.model.topicDetails.ReplicationFactor),
			Status: "info",
		},
		{
			Icon:   "💬",
			Text:   "Messages",
			Value:  fmt.Sprintf("%d", len(t.model.messages)),
			Status: "success",
		},
	}

	// Add config entries in stable alphabetical order (map iteration is random).
	configKeys := make([]string, 0, len(t.model.topicDetails.ConfigEntries))
	for key := range t.model.topicDetails.ConfigEntries {
		configKeys = append(configKeys, key)
	}
	sort.Strings(configKeys)

	configCount := 0
	for _, key := range configKeys {
		value := t.model.topicDetails.ConfigEntries[key]
		if configCount >= maxItems-len(items) {
			break
		}
		valueStr := "<nil>"
		if value != nil {
			valueStr = *value
			if len(valueStr) > 15 {
				valueStr = valueStr[:12] + "..."
			}
		}
		items = append(items, providers.SidebarItem{
			Icon:   "⚙️",
			Text:   key,
			Value:  valueStr,
			Status: "muted",
		})
		configCount++
	}

	return items
}

func (t *TopicInfoSection) HandleSectionUpdate(msg tea.Msg) tea.Cmd {
	return nil
}

func (t *TopicInfoSection) InitSection() tea.Cmd {
	return nil
}

func (t *TopicInfoSection) RefreshSection() tea.Cmd {
	return nil
}

// MessageInfoSection provides information about the selected message
type MessageInfoSection struct {
	model *Model
}

func NewMessageInfoSection(model *Model) *MessageInfoSection {
	return &MessageInfoSection{
		model: model,
	}
}

func (t *MessageInfoSection) GetTitle() string {
	return "SELECTED MESSAGE"
}

func (t *MessageInfoSection) RenderItems(maxItems, width int) []providers.SidebarItem {
	selectedMsg := t.model.GetSelectedMessage()
	if selectedMsg == nil {
		return []providers.SidebarItem{
			{
				Icon:   "❌",
				Text:   "No message selected",
				Value:  "",
				Status: "muted",
			},
		}
	}

	items := []providers.SidebarItem{
		{
			Icon:   "🔢",
			Text:   "Partition",
			Value:  fmt.Sprintf("%d", selectedMsg.Partition),
			Status: "info",
		},
		{
			Icon:   "📍",
			Text:   "Offset",
			Value:  fmt.Sprintf("%d", selectedMsg.Offset),
			Status: "info",
		},
	}

	// Schema IDs only: resolving them is a registry round trip, which the
	// message detail page does asynchronously.
	if selectedMsg.KeySchemaID != "" || selectedMsg.ValueSchemaID != "" {
		if selectedMsg.KeySchemaID != "" {
			items = append(items, providers.SidebarItem{
				Icon:   "🔑",
				Text:   "Key Schema ID",
				Value:  selectedMsg.KeySchemaID,
				Status: "warning",
			})
		}
		if selectedMsg.ValueSchemaID != "" {
			items = append(items, providers.SidebarItem{
				Icon:   "💎",
				Text:   "Value Schema ID",
				Value:  selectedMsg.ValueSchemaID,
				Status: "warning",
			})
		}
	}

	return items
}

func (t *MessageInfoSection) HandleSectionUpdate(msg tea.Msg) tea.Cmd {
	return nil
}

func (t *MessageInfoSection) InitSection() tea.Cmd {
	return nil
}

func (t *MessageInfoSection) RefreshSection() tea.Cmd {
	return nil
}

// ConsumptionControlSection provides consumption control information
type ConsumptionControlSection struct {
	model *Model
}

func NewConsumptionControlSection(model *Model) *ConsumptionControlSection {
	return &ConsumptionControlSection{
		model: model,
	}
}

func (t *ConsumptionControlSection) GetTitle() string {
	return "CONSUMPTION"
}

func (t *ConsumptionControlSection) RenderItems(maxItems, width int) []providers.SidebarItem {
	items := []providers.SidebarItem{}

	// Connection status
	statusIcon := "❌"
	statusColor := "error"
	switch t.model.connectionStatus {
	case "connected":
		statusIcon = "✅"
		statusColor = "success"
	case "connecting":
		statusIcon = "🔄"
		statusColor = "warning"
	case "retrying":
		statusIcon = "⚠️"
		statusColor = "warning"
	}

	items = append(items, providers.SidebarItem{
		Icon:   statusIcon,
		Text:   "Status",
		Value:  t.model.connectionStatus,
		Status: statusColor,
	})

	// Consumption state
	consumingIcon := "⏸️"
	consumingStatus := "muted"
	consumingText := "Stopped"
	if t.model.consuming {
		if t.model.paused {
			consumingIcon = "⏸️"
			consumingStatus = "warning"
			consumingText = "Paused"
		} else {
			consumingIcon = "▶️"
			consumingStatus = "success"
			consumingText = "Active"
		}
	}

	items = append(items, providers.SidebarItem{
		Icon:   consumingIcon,
		Text:   "Consuming",
		Value:  consumingText,
		Status: consumingStatus,
	})

	// Mode indicator
	modeIcon := "📋"
	modeStatus := "info"
	if t.model.consumeMode == ModeLive {
		modeIcon = "📡"
		modeStatus = "success"
	} else if t.model.consumeMode == ModeOldest {
		modeIcon = "📜"
		modeStatus = "muted"
	}

	items = append(items, providers.SidebarItem{
		Icon:   modeIcon,
		Text:   "Mode",
		Value:  t.model.consumeMode.String(),
		Status: modeStatus,
	})

	return items
}

func (t *ConsumptionControlSection) HandleSectionUpdate(msg tea.Msg) tea.Cmd {
	return nil
}

func (t *ConsumptionControlSection) InitSection() tea.Cmd {
	return nil
}

func (t *ConsumptionControlSection) RefreshSection() tea.Cmd {
	return nil
}

// TopicShortcutsSection provides keyboard shortcuts for the topic page
type TopicShortcutsSection struct {
	model *Model
}

func NewTopicShortcutsSection(model *Model) *TopicShortcutsSection {
	return &TopicShortcutsSection{
		model: model,
	}
}

func (t *TopicShortcutsSection) GetTitle() string {
	return "SHORTCUTS"
}

func (t *TopicShortcutsSection) RenderItems(maxItems, width int) []providers.SidebarItem {
	// Keys come from the registry, so this list cannot drift from what the
	// screen actually binds (it used to advertise space for pause).
	key := keys.Default.KeyFor
	shortcuts := []providers.SidebarItem{
		{Icon: "⌨️", Text: key(keys.ActionUp) + "/" + key(keys.ActionDown), Value: "navigate", Status: "info"},
		{Icon: "🔍", Text: key(keys.ActionSearch), Value: "search", Status: "info"},
		{Icon: "↕️", Text: key(keys.ActionExpand), Value: "expand row", Status: "info"},
		{Icon: "↩️", Text: key(keys.ActionActivate), Value: "view details", Status: "info"},
		{Icon: "⏯️", Text: key(keys.ActionPause), Value: "pause/resume", Status: "info"},
		{Icon: "🔄", Text: key(keys.ActionRefresh), Value: "refresh", Status: "info"},
		{Icon: "🚪", Text: key(keys.ActionCancel), Value: "back", Status: "info"},
	}

	// Limit to maxItems
	if len(shortcuts) > maxItems {
		shortcuts = shortcuts[:maxItems]
	}

	return shortcuts
}

func (t *TopicShortcutsSection) HandleSectionUpdate(msg tea.Msg) tea.Cmd {
	return nil
}

func (t *TopicShortcutsSection) InitSection() tea.Cmd {
	return nil
}

func (t *TopicShortcutsSection) RefreshSection() tea.Cmd {
	return nil
}
