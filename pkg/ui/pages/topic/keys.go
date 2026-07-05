package topic

import (
	"fmt"
	"strings"

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

	// Overlays capture keys while open (each owns its own key routing).
	if model.showGroups {
		return k.handleGroupsOverlayKey(model, msg)
	}
	if model.showOverview {
		return k.handleOverviewKey(model, msg)
	}
	if model.showSettings {
		return k.handleSettingsKey(model, msg)
	}
	if model.showAnalysis {
		return k.handleAnalysisKey(model, msg)
	}
	if model.showSettingsEdit {
		return k.handleEditFormKey(model, msg)
	}
	if model.showMutationForm {
		return k.handleMutationFormKey(model, msg)
	}
	if model.showSeek {
		return k.handleSeekFormKey(model, msg)
	}
	if model.showPartitions {
		return k.handlePartitionFormKey(model, msg)
	}
	if model.showProduce {
		return k.handleProduceFormKey(model, msg)
	}
	if model.showProjections {
		return k.handleProjectionsKey(model, msg)
	}
	if model.showSavedFilters {
		return k.handleSavedFiltersKey(model, msg)
	}

	// Overlay-open and header-action keys (checked before the centralized
	// bindings so single-character actions don't collide with them).
	switch msg.String() {
	case "C":
		return k.handleShowGroups(model)
	case "o":
		return k.handleShowOverview(model)
	case "s":
		return k.handleShowSettings(model)
	case "E":
		return k.handleShowSettingsEdit(model)
	case "t":
		return k.handleShowAnalysis(model)
	case "+":
		return k.handleIncreasePartitionsDialog(model)
	case "F":
		return k.handleReplicationFactorDialog(model)
	case "ctrl+p":
		return k.handleClearAllMessages(model)
	case "ctrl+r":
		return k.handleRecreateTopic(model)
	case "ctrl+d":
		return k.handleDeleteTopic(model)
	case "S":
		return k.handleShowSeek(model)
	case "#":
		return k.handleShowPartitions(model)
	case "P":
		return k.handleShowProduce(model)
	case "Y":
		return k.handleReproduce(model)
	case "L":
		return k.handleShowSavedFilters(model)
	case "X":
		return k.handleShowProjections(model)
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
	case key.Matches(msg, k.bindings.SwitchMode):
		return k.handleSwitchMode(model)
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

// handleSearchMode handles keys when search input is focused.
// A value prefixed with "~" is a smart-filter expression (MSG-25), compiled on
// Enter; anything else is a substring search over key/value/headers (MSG-23).
func (k *Keys) handleSearchMode(model *Model, msg tea.KeyMsg) tea.Cmd {
	// Handle Enter to confirm search
	if msg.String() == "enter" {
		model.searchMode = false
		model.searchInput.Blur()
		var cmd tea.Cmd
		if val := model.searchInput.Value(); strings.HasPrefix(val, smartFilterPrefix) {
			cmd = model.setSmartFilter(strings.TrimPrefix(val, smartFilterPrefix))
		} else {
			model.smartFilter = nil
		}
		model.FilterMessages()
		return cmd
	}

	// Handle Esc to cancel search
	if msg.String() == "esc" {
		model.searchMode = false
		model.searchInput.Blur()
		model.searchInput.SetValue("")
		model.smartFilter = nil
		model.FilterMessages()
		return nil
	}

	// Ctrl+S saves the active smart filter (MSG-25).
	if msg.String() == "ctrl+s" {
		return model.saveCurrentFilter()
	}

	// Let the search input handle other keys
	var cmd tea.Cmd
	model.searchInput, cmd = model.searchInput.Update(msg)
	cmds := []tea.Cmd{cmd}
	// Live substring filtering only — smart-filter expressions compile on Enter.
	if !strings.HasPrefix(model.searchInput.Value(), smartFilterPrefix) {
		model.smartFilter = nil
		model.FilterMessages()
	}
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

func (k *Keys) handleSwitchMode(model *Model) tea.Cmd {
	// Stop any active live consumption before switching.
	if model.consuming {
		if model.cancelConsumption != nil {
			model.cancelConsumption()
			model.cancelConsumption = nil
		}
		model.consuming = false
	}
	model.consumeMode = model.consumeMode.Next()
	model.statusMessage = fmt.Sprintf("Mode: %s", model.consumeMode)
	return model.startForMode()
}

func (k *Keys) handleRefresh(model *Model) tea.Cmd {
	model.statusMessage = fmt.Sprintf("Refreshing… (mode: %s)", model.consumeMode)
	return model.startForMode()
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
		// Load schema info here (once, on explicit open) — not in the render path.
		model.loadSchemaInfoForMessage(selectedMsg)
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
		if model.cursorRow > 0 {
			model.cursorRow--
		}
		model.markRenderDirty()
		return nil
	case "down":
		visibleCount := len(model.pagination.GetVisibleMessages(model.filteredMessages))
		if visibleCount == 0 {
			visibleCount = model.messageTable.PageSize()
		}
		if model.cursorRow < visibleCount-1 {
			model.cursorRow++
		}
		model.markRenderDirty()
		return nil
	case "pagedown":
		if model.pagination.NextPage() {
			model.pendingReset = true
			model.updateMessageTable()
			model.markRenderDirty()
		} else if !model.loading {
			// Already on the last page and not currently fetching — load more.
			if flags := model.nextBatchFlags(); flags != nil {
				return model.consumption.FetchNextBatch(*flags)
			}
		}
		return nil
	case "pageup":
		if model.pagination.PrevPage() {
			model.pendingReset = true
			model.updateMessageTable()
			model.markRenderDirty()
		}
		return nil
	case "home":
		model.pagination.FirstPage()
		model.pendingReset = true
		model.updateMessageTable()
		model.markRenderDirty()
		return nil
	case "end":
		model.pagination.LastPage()
		model.pendingReset = true
		model.updateMessageTable()
		model.markRenderDirty()
		return nil
	}
	return nil
}

// GetKeyBindings returns the centralized key bindings for help display
func (k *Keys) GetKeyBindings() []key.Binding {
	return []key.Binding{
		k.bindings.Search,
		k.bindings.SwitchMode,
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
		key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "overview")),
		key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "settings")),
		key.NewBinding(key.WithKeys("E"), key.WithHelp("E", "edit settings")),
		key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "consumer groups")),
		key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "statistics")),
		key.NewBinding(key.WithKeys("+"), key.WithHelp("+", "add partitions")),
		key.NewBinding(key.WithKeys("F"), key.WithHelp("F", "replication factor")),
		key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "clear messages")),
		key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "recreate")),
		key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "delete")),
		key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "seek")),
		key.NewBinding(key.WithKeys("#"), key.WithHelp("#", "partitions/serde")),
		key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "produce")),
		key.NewBinding(key.WithKeys("Y"), key.WithHelp("Y", "reproduce")),
		key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "saved filters")),
		key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "projections")),
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
		"o     Overview + partitions",
		"s     Settings",
		"E     Edit settings",
		"C     Consumer groups",
		"t     Statistics",
		"+     Add partitions",
		"F     Replication factor",
		"^p    Clear messages",
		"^r    Recreate topic",
		"^d    Delete topic",
		"Esc   Exit search",
		"q/Esc Back to topics",
	}
}
