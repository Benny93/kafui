package topic

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/Benny93/kafui/pkg/ui/core"
)

// mouseTarget maps a mouse event to the page row under it (-1 for none), or
// reports that it is over the expanded panel, using the last render's layout.
func (m *Model) mouseTarget(msg tea.MouseMsg) (row int, inPanel bool) {
	z := zone.Get("message-table")
	if z == nil || !z.InBounds(msg) {
		return -1, false
	}
	_, relY := z.Pos(msg)
	row, inPanel = m.tableLayout.lineTarget(relY)
	if row >= len(m.pagination.GetVisibleMessages(m.filteredMessages)) {
		return -1, false
	}
	return row, inPanel
}

func (h *Handlers) handleMouseMsg(model *Model, msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Same gesture vocabulary as every other list: the wheel moves the view and
	// never the cursor, a click selects, and a click on the already-selected
	// row activates. This screen used to open a message on the first click,
	// while the resource list only selected — one of the two had to be wrong.
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		// Over the expanded panel the wheel scrolls its content; elsewhere it
		// pages the table.
		if _, inPanel := model.mouseTarget(msg); inPanel {
			step := 3
			if msg.Button == tea.MouseButtonWheelUp {
				step = -3
			}
			model.scrollExpanded(step)
			break
		}
		if msg.Button == tea.MouseButtonWheelUp {
			model.pagination.PrevPage()
		} else {
			model.pagination.NextPage()
		}
		model.markRenderDirty()

	case tea.MouseButtonNone:
		// Hover feedback, same as every other list.
		if !core.IsHover(msg) {
			break
		}
		if row, _ := model.mouseTarget(msg); row >= 0 && row != model.cursorRow {
			model.cursorRow = row
			model.markRenderDirty()
		}

	case tea.MouseButtonLeft:
		if !core.IsLeftRelease(msg) {
			break
		}
		row, _ := model.mouseTarget(msg)
		if row < 0 {
			break
		}
		// Capture the previous cursor before moving it, or the
		// already-selected test would always be true.
		wasSelected := row == model.cursorRow
		double := model.clicks.Click(msg)
		model.cursorRow = row
		model.markRenderDirty()
		if double || wasSelected {
			return model, model.keys.handleSelect(model)
		}
	}

	return model, nil
}
