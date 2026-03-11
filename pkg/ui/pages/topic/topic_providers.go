package topic

import (
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func (t *TopicContentProvider) RenderContent(width, height int) string {
	// PERFORMANCE: Check render cache first
	if cached, ok := t.model.getRenderCache(); ok {
		return cached
	}
	
	// Update table dimensions based on actual content area size
	// width/height here are the inner content dimensions (after border and padding)
	// We need to add back the border (2) and padding (2) to get the full cell width
	if width > 0 && height > 0 {
		cellWidth := width + 4 // Account for border and padding
		cellHeight := height + 4
		t.model.updateTableDimensions(cellWidth, cellHeight)
	}

	if t.model.error != nil {
		return t.renderError()
	}

	if t.model.loading && len(t.model.messages) == 0 {
		return t.renderLoading()
	}

	if len(t.model.messages) == 0 && !t.model.loading {
		return t.renderEmpty()
	}

	var content string

	// PERFORMANCE: For large datasets, use custom renderer (bypasses bubbles table overhead)
	if len(t.model.filteredMessages) > UseCustomRenderer {
		content = t.renderCustomTable(width, height)
	} else {
		// For small datasets, use standard table rendering
		t.model.updateMessageTable()

		var contentBuilder strings.Builder

		// Add search bar if in search mode
		if t.model.searchMode {
			searchBar := t.renderSearchBar(width)
			contentBuilder.WriteString(searchBar)
			contentBuilder.WriteString("\n\n")
		}

		// Render the main table with max width constraint to prevent overflow
		tableView := t.model.messageTable.View()
		// Use MaxWidth to ensure table doesn't exceed available width
		tableView = lipgloss.NewStyle().MaxWidth(width).Render(tableView)
		contentBuilder.WriteString(tableView)
		
		content = contentBuilder.String()
	}
	
	// Cache the render result
	t.model.setRenderCache(content)
	
	return content
}

// renderCustomTable renders a custom table for large datasets (bypasses bubbles table)
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

func (t *TopicContentProvider) renderLoading() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Padding(1)
	return style.Render(fmt.Sprintf("%s Loading messages...", t.model.spinner.View()))
}

func (t *TopicContentProvider) renderEmpty() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Padding(1)

	if t.model.consuming {
		return style.Render(fmt.Sprintf("%s Waiting for messages...", t.model.spinner.View()))
	}
	return style.Render("No messages available. Press 'r' to start consumption or check connection.")
}

func (t *TopicContentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd {
	// Delegate to the model's handlers
	_, cmd := t.model.handlers.Handle(t.model, msg)
	return cmd
}

func (t *TopicContentProvider) InitContent() tea.Cmd {
	// Start consuming messages
	return t.model.consumption.StartConsuming()
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
	}
}

func (t *TopicHeaderDataProvider) HandleHeaderUpdate(msg tea.Msg) tea.Cmd {
	// Handle timer ticks for header updates
	switch msg := msg.(type) {
	case TimerTickMsg:
		t.model.lastUpdate = time.Time(msg)
		return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
			return TimerTickMsg(t)
		})
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

	// Add config entries (limited to fit)
	configCount := 0
	for key, value := range t.model.topicDetails.ConfigEntries {
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

	// Follow mode
	followIcon := "📍"
	followStatus := "muted"
	if t.model.consumeFlags.Follow {
		followIcon = "🔄"
		followStatus = "info"
	}

	items = append(items, providers.SidebarItem{
		Icon:   followIcon,
		Text:   "Follow",
		Value:  fmt.Sprintf("%t", t.model.consumeFlags.Follow),
		Status: followStatus,
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
