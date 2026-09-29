package topic

import (
	tea "github.com/charmbracelet/bubbletea"
)

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
