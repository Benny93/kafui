package topic

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Keys routes the topic screen's keys through the single binding registry.
// The screen promotes exactly one action of its own (pause/resume); everything
// that used to sit on an undiscoverable letter is now an actions-menu entry.
type Keys struct {
	scope keys.Scope
}

// NewKeys creates a new Keys instance bound to the topic scope.
func NewKeys() *Keys {
	return &Keys{scope: keys.ScopeTopic}
}

// KeyScope reports the scope the shell resolves this page's keys against.
func (k *Keys) KeyScope() keys.Scope { return k.scope }

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

	// Every key resolves through the registry. The twelve bare letters this
	// screen used to claim (C o s E t + F S # P Y L X) are gone: they appeared
	// in no help text, and several shadowed shell chords. They are now entries
	// in the actions menu, which shows each one's name.
	action, bound := keys.Default.Resolve(k.scope, msg.String())
	if !bound {
		return tea.Batch(cmds...)
	}

	switch action {
	case keys.ActionSearch:
		return k.handleSearch(model)
	case keys.ActionPause:
		return k.handlePauseResume(model)
	case keys.ActionRefresh:
		// One refresh concept. `r`, `R` and ctrl+r used to be three.
		return k.handleRefresh(model)
	case keys.ActionActivate:
		return k.handleSelect(model)

	case keys.ActionFormat:
		return k.handleFormat(model)
	case keys.ActionMetadata:
		return k.handleMetadata(model)

	case keys.ActionUp:
		return k.handleNavigation(model, "up")
	case keys.ActionDown:
		return k.handleNavigation(model, "down")
	case keys.ActionPageBack:
		return k.handleNavigation(model, "pageup")
	case keys.ActionPageForward:
		return k.handleNavigation(model, "pagedown")
	case keys.ActionFirst:
		return k.handleNavigation(model, "home")
	case keys.ActionLast:
		return k.handleNavigation(model, "end")

	case keys.ActionCopy:
		// One copy key acting on the focused pane, rather than c for the key
		// and v for the value.
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

	// Saving the active smart filter (MSG-25). It was ctrl+s, which is XOFF and
	// can freeze the terminal; F2 is the save key on every other form.
	if action, bound := keys.Default.Resolve(keys.ScopeTextEntry, msg.String()); bound && action == keys.ActionCommitSave {
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

// GetKeyBindings returns this screen's bindings for the help overlay, read
// from the registry so help cannot list a key the screen does not handle — the
// previous list advertised twelve keys the help overlay never rendered.
func (k *Keys) GetKeyBindings() []key.Binding {
	var out []key.Binding
	for _, b := range keys.Default.InScope(k.scope) {
		out = append(out, b.KeyBinding())
	}
	return out
}

// GetCentralizedKeyMap returns the hint key map for the footer.
func GetCentralizedKeyMap() keys.HintKeyMap {
	return keys.Hints(keys.ScopeTopic)
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
