package topic

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/ui/components"
	"github.com/Benny93/kafui/pkg/ui/shared"
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
	// Page size follows the rows renderTableCustom draws at this height, so
	// the cursor can never sit on a row that is not drawn.
	t.model.syncPageSize(height)

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

	// The router calls Init once, when the page is created; OnFocus resumes a
	// live stream on later activations. A page created in Live mode streams.
	if t.model.consumeMode == ModeLive {
		t.model.retryCount = 0
		return tea.Batch(t.model.startLive(), t.model.spinner.Tick)
	}

	// Only callers that loaded the topic pass its partitions. Without them the
	// MessageCount is not known (a zero value, not a real count), so fetch the
	// metadata for the sidebar and dialogs, and fetch messages regardless.
	var metaCmd tea.Cmd
	if t.model.topicDetails.NumPartitions <= 0 {
		metaCmd = fetchTopicMeta(t.model.dataSource, t.model.topicName)
	}

	// If the main page already knew the topic is empty, skip the fetch
	// entirely — no loading screen, show empty state immediately.
	if t.model.topicDetails.MessageCount == 0 && t.model.topicDetails.NumPartitions > 0 && len(t.model.messages) == 0 {
		t.model.beginGeneration()
		t.model.loading = false
		t.model.statusMessage = "Topic is empty — no messages found"
		return nil
	}

	t.model.beginGeneration()
	t.model.retryCount = 0
	t.model.loading = true
	const fetchCount = 60
	return tea.Batch(
		t.model.consumption.FetchLatestMessages(fetchCount),
		t.model.spinner.Tick,
		metaCmd,
	)
}

func (t *TopicContentProvider) IsInputMode() bool {
	return t.model.IsInputMode()
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
