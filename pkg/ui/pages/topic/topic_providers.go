package topic

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/ui/components"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

// TopicContentProvider provides the main content for the topic page (message table and search)
type TopicContentProvider struct {
	model *Model
}

func NewTopicContentProvider(model *Model) *TopicContentProvider {
	return &TopicContentProvider{
		model: model,
	}
}

func (t *TopicContentProvider) RenderContent(width, height int) (result string) {
	// Catch rendering panics so a bad message never crashes the TUI.
	defer func() {
		if r := recover(); r != nil {
			shared.Log.Error("panic in RenderContent", "topic", t.model.topicName, "panic", r,
				"messages", len(t.model.messages), "width", width, "height", height)
			result = fmt.Sprintf("Render error — see ~/.kafui/kafui.log\n%v", r)
		}
	}()

	tableWidth := width - 2
	tableHeight := height

	if t.model.common != nil && t.model.common.Layout != nil {
		tableWidth = t.model.common.Layout.GetAvailableWidth() - 2
		tableHeight = t.model.common.Layout.GetAvailableHeight() - 4
	}

	if tableHeight < 5 {
		tableHeight = 5
	}
	if tableWidth < 20 {
		tableWidth = 20
	}

	if tableWidth > 0 && tableHeight > 0 {
		t.model.updateTableDimensions(tableWidth, tableHeight)
	}

	// Overlays take over the content area when open.
	if t.model.showGroups {
		return t.model.renderGroupsOverlay(width)
	}
	if t.model.showOverview {
		return t.model.renderOverviewOverlay(width)
	}
	if t.model.showSettings {
		return t.model.renderSettingsOverlay(width)
	}
	if t.model.showSettingsEdit {
		return t.model.renderEditOverlay(width)
	}
	if t.model.showMutationForm {
		return t.model.renderMutationOverlay(width)
	}
	if t.model.showAnalysis {
		return t.model.renderAnalysisOverlay(width)
	}
	if t.model.showSeek {
		return t.model.renderSeekOverlay(width)
	}
	if t.model.showPartitions {
		return t.model.renderPartitionsOverlay(width)
	}
	if t.model.showProduce {
		return t.model.renderProduceOverlay(width)
	}
	if t.model.showProjections {
		return t.model.renderProjectionsOverlay(width)
	}
	if t.model.showSavedFilters {
		return t.model.renderSavedFiltersOverlay(width)
	}

	if t.model.error != nil {
		return t.renderError()
	}

	if t.model.loading && len(t.model.messages) == 0 {
		return t.renderLoading(tableWidth)
	}

	if len(t.model.messages) == 0 && !t.model.loading {
		return t.renderEmpty()
	}

	var contentBuilder strings.Builder

	if t.model.searchMode {
		contentBuilder.WriteString(t.renderSearchBar(width))
		contentBuilder.WriteString("\n\n")
	}

	// renderTableCustom reuses cached row strings on cursor-only moves.
	tableView := t.model.renderTableCustom(width, height)
	tableView = zone.Mark("message-table", tableView)
	contentBuilder.WriteString(tableView)

	return strings.TrimSpace(contentBuilder.String())
}

// renderCustomTable delegates to renderTableCustom (kept for compatibility).
func (t *TopicContentProvider) renderCustomTable(width, height int) string {
	return t.model.renderTableCustom(width, height)
}

func (t *TopicContentProvider) renderSearchBar(width int) string {
	searchStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	// Create search prompt
	prompt := searchStyle.Render("🔍 Search: ")
	searchValue := t.model.searchInput.Value()
	if searchValue == "" {
		searchValue = promptStyle.Render("(type to filter messages...)")
	}

	// Add cursor if in search mode
	cursor := ""
	if t.model.searchMode {
		cursor = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Render("█")
	}

	searchLine := prompt + searchValue + cursor

	// Add help text
	helpText := promptStyle.Render("ESC to cancel • Enter to search")

	return searchLine + "\n" + helpText
}

func (t *TopicContentProvider) renderError() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Padding(1)
	return style.Render(fmt.Sprintf("Error: %v", t.model.error))
}

func (t *TopicContentProvider) renderLoading(width int) string {
	spinnerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	if t.model.fetchProgressBar.IsActive() {
		label := spinnerStyle.Render(t.model.spinner.View()) + " " +
			labelStyle.Render(fmt.Sprintf(
				"Fetching messages… %d / %d",
				t.model.fetchProgressBar.Current(),
				t.model.fetchProgressBar.Total(),
			))
		bar := t.model.fetchProgressBar.View(width - 4)
		return lipgloss.NewStyle().Padding(1, 0).Render(label + "\n" + bar)
	}

	// Shared loading-indicator mechanism (UI-12): a centered spinner + label.
	frame := spinnerStyle.Render(t.model.spinner.View())
	return components.CenteredLoading(frame, labelStyle.Render("Loading messages…"), width, 0)
}

func (t *TopicContentProvider) renderEmpty() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Padding(1)

	if t.model.consuming {
		return style.Render(fmt.Sprintf("%s Waiting for messages...", t.model.spinner.View()))
	}
	return style.Render("This topic has no messages.")
}

func (t *TopicContentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd {
	// Delegate to the model's handlers
	_, cmd := t.model.handlers.Handle(t.model, msg)
	return cmd
}

func (t *TopicContentProvider) InitContent() tea.Cmd {
	shared.Log.Info("opening topic page", "topic", t.model.topicName,
		"partitions", t.model.topicDetails.NumPartitions,
		"replicationFactor", t.model.topicDetails.ReplicationFactor,
		"knownMessageCount", t.model.topicDetails.MessageCount)

	// If the message count was already loaded on the main page and is 0,
	// skip the fetch entirely — no loading screen, show empty state immediately.
	if t.model.topicDetails.MessageCount == 0 {
		t.model.loading = false
		t.model.statusMessage = "Topic is empty — no messages found"
		return nil
	}

	t.model.loading = true
	const fetchCount = 60
	return tea.Batch(
		t.model.consumption.FetchLatestMessages(fetchCount),
		t.model.spinner.Tick,
	)
}

func (t *TopicContentProvider) IsInputMode() bool {
	return false
}

// GetContentSize returns the estimated content size for scrollbar calculation
func (t *TopicContentProvider) GetContentSize(width int) int {
	// Estimate based on table rows plus header
	rowCount := len(t.model.messages)
	if rowCount == 0 {
		return 5 // Default for empty/loading states
	}
	// Add header lines and account for search bar
	return rowCount + 5
}

// TopicHeaderDataProvider provides header data for the topic page
type TopicHeaderDataProvider struct {
	model *Model
}

func NewTopicHeaderDataProvider(model *Model) *TopicHeaderDataProvider {
	return &TopicHeaderDataProvider{
		model: model,
	}
}

func (t *TopicHeaderDataProvider) GetBrandName() string {
	return "Kafui™"
}

func (t *TopicHeaderDataProvider) GetAppName() string {
	return fmt.Sprintf("Topic: %s", t.model.topicName)
}

func (t *TopicHeaderDataProvider) GetStatusData() map[string]interface{} {
	return map[string]interface{}{
		"time":      t.model.lastUpdate.Format("15:04:05"),
		"status":    t.model.connectionStatus,
		"topic":     t.model.topicName,
		"messages":  len(t.model.messages),
		"consuming": t.model.consuming,
		"paused":    t.model.paused,
		"mode":      t.model.consumeMode.String(),
	}
}

func (t *TopicHeaderDataProvider) HandleHeaderUpdate(msg tea.Msg) tea.Cmd {
	// Handle timer ticks for header updates — only when actively consuming.
	// Stopping the tick when idle prevents timer proliferation: if this
	// handler and the model handler both re-schedule on the same message, the
	// number of pending timers doubles every cycle.
	switch msg := msg.(type) {
	case TimerTickMsg:
		t.model.lastUpdate = time.Time(msg)
		if t.model.consuming {
			return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
				return TimerTickMsg(t)
			})
		}
		// Not consuming — let the timer stop.
		return nil
	}
	return nil
}

func (t *TopicHeaderDataProvider) InitHeader() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return TimerTickMsg(t)
	})
}

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

	// Add schema information if available
	if t.model.selectedMessageSchema != nil {
		if t.model.selectedMessageSchema.KeySchema != nil {
			items = append(items, providers.SidebarItem{
				Icon:   "🔑",
				Text:   "Key Schema",
				Value:  t.model.selectedMessageSchema.KeySchema.RecordName,
				Status: "success",
			})
		}
		if t.model.selectedMessageSchema.ValueSchema != nil {
			items = append(items, providers.SidebarItem{
				Icon:   "💎",
				Text:   "Value Schema",
				Value:  t.model.selectedMessageSchema.ValueSchema.RecordName,
				Status: "success",
			})
		}
	} else if selectedMsg.KeySchemaID != "" || selectedMsg.ValueSchemaID != "" {
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
	shortcuts := []providers.SidebarItem{
		{Icon: "⌨️", Text: "j/k", Value: "navigate", Status: "info"},
		{Icon: "🔍", Text: "/", Value: "search", Status: "info"},
		{Icon: "⏯️", Text: "space", Value: "pause/resume", Status: "info"},
		{Icon: "🔄", Text: "r", Value: "retry", Status: "info"},
		{Icon: "↩️", Text: "enter", Value: "view details", Status: "info"},
		{Icon: "🚪", Text: "esc", Value: "back", Status: "info"},
		{Icon: "❌", Text: "q", Value: "quit", Status: "error"},
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
