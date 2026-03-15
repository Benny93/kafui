package topic

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Keys handles key bindings for the topic page using centralized key definitions
type Keys struct {
	bindings keys.TopicKeyMap
}

// NewKeys creates a new Keys instance using centralized key bindings
func NewKeys() *Keys {
	return &Keys{
		bindings: keys.DefaultKeyMap().Topic,
	}
}

// HandleKey processes key events using centralized key bindings
func (k *Keys) HandleKey(model *Model, msg tea.KeyMsg) tea.Cmd {
	var cmds []tea.Cmd

	// If in search mode, let the search input handle keys
	if model.searchMode {
		return k.handleSearchMode(model, msg)
	}

	// Handle navigation keys
	switch {
	case key.Matches(msg, k.bindings.Back):
		return k.handleBack(model)
	case key.Matches(msg, k.bindings.Quit):
		return k.handleQuit(model)
	case key.Matches(msg, k.bindings.Search):
		return k.handleSearch(model)
	case key.Matches(msg, k.bindings.Pause):
		return k.handlePauseResume(model)
	case key.Matches(msg, k.bindings.Refresh):
		return k.handleRefresh(model)
	case key.Matches(msg, k.bindings.Retry):
		return k.handleRetry(model)
	case key.Matches(msg, k.bindings.Select):
		return k.handleSelect(model)
	}

	// Handle display option keys
	switch {
	case key.Matches(msg, k.bindings.Format):
		return k.handleFormat(model)
	case key.Matches(msg, k.bindings.Headers):
		return k.handleHeaders(model)
	case key.Matches(msg, k.bindings.Metadata):
		return k.handleMetadata(model)
	}

	// Handle scrolling keys
	switch {
	case key.Matches(msg, k.bindings.ScrollUp):
		return k.handleNavigation(model, "up")
	case key.Matches(msg, k.bindings.ScrollDown):
		return k.handleNavigation(model, "down")
	case key.Matches(msg, k.bindings.PageUp):
		return k.handleNavigation(model, "pageup")
	case key.Matches(msg, k.bindings.PageDown):
		return k.handleNavigation(model, "pagedown")
	case key.Matches(msg, k.bindings.GotoStart):
		return k.handleNavigation(model, "home")
	case key.Matches(msg, k.bindings.GotoEnd):
		return k.handleNavigation(model, "end")
	}

	// Handle message operation keys
	switch {
	case key.Matches(msg, k.bindings.CopyKey):
		return k.handleCopyKey(model)
	case key.Matches(msg, k.bindings.CopyValue):
		return k.handleCopyValue(model)
	}

	return tea.Batch(cmds...)
}

// handleSearchMode handles keys when search input is focused
func (k *Keys) handleSearchMode(model *Model, msg tea.KeyMsg) tea.Cmd {
	// Handle Enter to confirm search
	if msg.String() == "enter" {
		model.searchMode = false
		model.searchInput.Blur()
		model.FilterMessages()
		return nil
	}

	// Handle Esc to cancel search
	if msg.String() == "esc" {
		model.searchMode = false
		model.searchInput.Blur()
		model.searchInput.SetValue("")
		model.FilterMessages()
		return nil
	}

	// Let the search input handle other keys
	var cmd tea.Cmd
	model.searchInput, cmd = model.searchInput.Update(msg)
	cmds := []tea.Cmd{cmd}
	model.FilterMessages()
	return tea.Batch(cmds...)
}

// Key handling functions

func (k *Keys) handleBack(model *Model) tea.Cmd {
	if model.searchMode {
		model.searchMode = false
		model.searchInput.Blur()
		model.FilterMessages()
		return nil
	}

	// Cancel consumption first
	if model.cancelConsumption != nil {
		model.cancelConsumption()
	}

	return func() tea.Msg {
		return core.BackMsg{}
	}
}

func (k *Keys) handleQuit(model *Model) tea.Cmd {
	if model.cancelConsumption != nil {
		model.cancelConsumption()
	}
	return tea.Quit
}

func (k *Keys) handleSearch(model *Model) tea.Cmd {
	model.searchMode = true
	model.searchInput.Focus()
	return nil
}

func (k *Keys) handlePauseResume(model *Model) tea.Cmd {
	model.TogglePause()
	return nil
}

func (k *Keys) handleRefresh(model *Model) tea.Cmd {
	const fetchCount = 60
	model.statusMessage = "Refreshing messages..."
	return model.consumption.FetchLatestMessages(fetchCount)
}

func (k *Keys) handleRetry(model *Model) tea.Cmd {
	if model.consumption != nil {
		return model.consumption.RetryConnection()
	}
	return nil
}

func (k *Keys) handleSelect(model *Model) tea.Cmd {
	if model.searchMode {
		model.FilterMessages()
		return nil
	}

	// Navigate to message detail page
	if selectedMsg := model.GetSelectedMessage(); selectedMsg != nil {
		model.selectedMessage = selectedMsg
		return func() tea.Msg {
			pageID := fmt.Sprintf("detail:%s:%d:%d", model.topicName, selectedMsg.Partition, selectedMsg.Offset)
			return core.PageChangeMsg{PageID: pageID, Data: *selectedMsg}
		}
	}

	return nil
}

func (k *Keys) handleFormat(model *Model) tea.Cmd {
	// Toggle message format (placeholder - topic page may not support this)
	model.statusMessage = "Format toggle not implemented"
	return nil
}

func (k *Keys) handleHeaders(model *Model) tea.Cmd {
	// Toggle headers display (placeholder)
	model.statusMessage = "Headers toggle not implemented"
	return nil
}

func (k *Keys) handleMetadata(model *Model) tea.Cmd {
	// Toggle metadata display (placeholder)
	model.statusMessage = "Metadata toggle not implemented"
	return nil
}

func (k *Keys) handleCopyKey(model *Model) tea.Cmd {
	if selectedMsg := model.GetSelectedMessage(); selectedMsg != nil && selectedMsg.Key != "" {
		model.statusMessage = "Message key copied to clipboard"
		// TODO: Implement actual clipboard copy
	}
	return nil
}

func (k *Keys) handleCopyValue(model *Model) tea.Cmd {
	if selectedMsg := model.GetSelectedMessage(); selectedMsg != nil && selectedMsg.Value != "" {
		model.statusMessage = "Message value copied to clipboard"
		// TODO: Implement actual clipboard copy
	}
	return nil
}

func (k *Keys) handleNavigation(model *Model, direction string) tea.Cmd {
	switch direction {
	case "up":
		currentRow := model.messageTable.GetHighlightedRowIndex()
		if currentRow > 0 {
			model.messageTable = model.messageTable.WithHighlightedRow(currentRow - 1)
		}
		model.markRenderDirty()
		return nil
	case "down":
		currentRow := model.messageTable.GetHighlightedRowIndex()
		pageSize := model.messageTable.PageSize()
		visibleCount := len(model.pagination.GetVisibleMessages(model.filteredMessages))
		if visibleCount == 0 {
			visibleCount = pageSize
		}
		if currentRow < visibleCount-1 {
			model.messageTable = model.messageTable.WithHighlightedRow(currentRow + 1)
		}
		model.markRenderDirty()
		return nil
	case "pageup":
		if model.pagination.PrevPage() {
			model.messageTable = model.messageTable.WithCurrentPage(model.pagination.Page + 1)
			model.markRenderDirty()
		}
		return nil
	case "pagedown":
		if model.pagination.NextPage() {
			model.messageTable = model.messageTable.WithCurrentPage(model.pagination.Page + 1)
			model.markRenderDirty()
		}
		return nil
	case "home":
		model.pagination.FirstPage()
		model.messageTable = model.messageTable.PageFirst()
		model.markRenderDirty()
		return nil
	case "end":
		model.pagination.LastPage()
		model.messageTable = model.messageTable.PageLast()
		model.markRenderDirty()
		return nil
	}
	return nil
}

// GetKeyBindings returns the centralized key bindings for help display
func (k *Keys) GetKeyBindings() []key.Binding {
	return []key.Binding{
		k.bindings.Search,
		k.bindings.Back,
		k.bindings.Quit,
		k.bindings.Select,
		k.bindings.Pause,
		k.bindings.Refresh,
		k.bindings.Retry,
		k.bindings.ScrollUp,
		k.bindings.ScrollDown,
		k.bindings.GotoStart,
		k.bindings.GotoEnd,
		k.bindings.CopyKey,
		k.bindings.CopyValue,
	}
}

// GetCentralizedKeyMap returns the centralized key map for footer display
func GetCentralizedKeyMap() keys.TopicKeyMap {
	return keys.DefaultKeyMap().Topic
}

// GetShortcuts returns formatted shortcut descriptions
func (k *Keys) GetShortcuts() []string {
	return []string{
		"↑/↓   Select row",
		"Enter View details",
		"Space Pause/resume",
		"R     Refresh messages",
		"g/G   First/Last page",
		"/     Search messages",
		"r     Retry connection",
		"c     Copy key",
		"v     Copy value",
		"f     Toggle format",
		"h     Toggle headers",
		"m     Toggle metadata",
		"Esc   Exit search",
		"q/Esc Back to topics",
	}
}
