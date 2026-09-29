package mainpage

import (
	"github.com/Benny93/kafui/pkg/ui/keys"
	tea "github.com/charmbracelet/bubbletea"
)

// handleKey routes a key press: open forms first, then the filter input,
// then the binding registry; unhandled bound keys go to the table.
func (k *KafuiContentProvider) handleKey(msg tea.KeyMsg) tea.Cmd {
	var cmds []tea.Cmd

	// The create/clone form swallows all key input while visible.
	if k.showTopicForm && k.topicForm != nil {
		cmd, _ := k.topicForm.Update(msg)
		return cmd
	}
	// ACL / quota overlay forms swallow all key input while visible.
	if f := k.activeOverlayForm(); f != nil {
		cmd, _ := f.Update(msg)
		return cmd
	}

	// Text-entry precedence: while the filter is open every printable key is
	// typed, so nothing bound in Normal mode fires.
	if k.searchMode {
		switch msg.String() {
		case "esc":
			k.searchMode = false
			k.clearSearch()
			return k.reloadPageDetails()
		case "enter":
			k.searchMode = false
			return nil
		case "backspace":
			if len(k.currentFilter) > 0 {
				k.currentFilter = k.currentFilter[:len(k.currentFilter)-1]
				k.handleSearch(k.currentFilter)
				return k.pageDetailsWhenSearchSettles()
			}
			return nil
		default:
			if len(msg.Runes) > 0 {
				k.currentFilter += string(msg.Runes)
				k.handleSearch(k.currentFilter)
				return k.pageDetailsWhenSearchSettles()
			}
			return nil
		}
	}

	// Every remaining key resolves through the single binding registry.
	// An unbound key does nothing rather than falling into a screen-local
	// switch, which is what let the same keystroke mean two things.
	action, bound := keys.Default.Resolve(keys.ScopeList, msg.String())
	if !bound {
		return nil
	}
	switch action {
	case keys.ActionSearch:
		k.searchMode = true
		k.currentFilter = ""
		return nil

	case keys.ActionActivate:
		return k.handleResourceSelection()

	// Logical page navigation — handled here so bubble-table doesn't treat
	// these as visual page changes within the current 50-item slice.
	case keys.ActionPageBack:
		if k.pagination.PrevPage() {
			k.updateTableForCurrentPageAndReset()
			cmds = append(cmds, k.loadPageDetails())
		}
		return tea.Batch(cmds...)
	case keys.ActionPageForward:
		if k.pagination.NextPage() {
			k.updateTableForCurrentPageAndReset()
			cmds = append(cmds, k.loadPageDetails())
		}
		return tea.Batch(cmds...)
	case keys.ActionFirst:
		if !k.pagination.OnFirstPage() {
			k.pagination.FirstPage()
			k.updateTableForCurrentPageAndReset()
			cmds = append(cmds, k.loadPageDetails())
		}
		return tea.Batch(cmds...)
	case keys.ActionLast:
		if !k.pagination.OnLastPage() {
			k.pagination.LastPage()
			k.updateTableForCurrentPageAndReset()
			cmds = append(cmds, k.loadPageDetails())
		}
		return tea.Batch(cmds...)

	case keys.ActionCopy:
		return k.handleCopyRow()

	case keys.ActionSort:
		k.cycleSortColumn()
		return k.reloadPageDetails()
	case keys.ActionSortReverse:
		k.toggleSortDir()
		return k.reloadPageDetails()

	case keys.ActionToggleInternal:
		if k.isTopicResource() {
			return k.toggleHideInternal()
		}
		return nil

	case keys.ActionExport:
		return k.exportCurrentResourceCSV()

	case keys.ActionNew:
		return k.createForCurrentResource()

	case keys.ActionDelete:
		return k.deleteForCurrentResource()

	case keys.ActionEdit:
		if k.isQuotaResource() {
			return k.openQuotaForm(true)
		}
		return nil

	case keys.ActionToggleMark:
		if k.isTopicResource() {
			k.toggleTopicSelection()
		}
		return nil
	case keys.ActionMarkAll:
		if k.isTopicResource() {
			k.selectAllVisibleTopics()
		}
		return nil

	case keys.ActionCancel:
		// The shell unwinds esc, but a live selection is this pane's own
		// level to clear, so it is consumed here first (see Unwind).
		if len(k.selected) > 0 {
			k.clearTopicSelection()
			return nil
		}
		return nil
	}

	// Delegate remaining keys (↑/↓ row navigation) to bubble-table
	var cmd tea.Cmd
	k.resourcesTable, cmd = k.resourcesTable.Update(msg)
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}
