package kafui

import (
	"com/emptystate/kafui/pkg/api"
	"fmt"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func SetupTableInput(table *tview.Table, app *tview.Application, pages *tview.Pages, dataSource api.KafkaDataSource, msgChannel chan UIEvent) {

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			frontPage, _ := pages.GetFrontPage()
			if frontPage != "main" {
				return event
			}
			// Check if the table has focus
			if app.GetFocus() == table {
				if currentResouce == Topic[0] {
					row, _ := table.GetSelection()
					text := table.GetCell(row, 0).Text
					currentTopic = text
					msgChannel <- OnPageChange
					pages.SwitchToPage("topicPage")
				}
				if currentResouce == Context[0] {
					row, _ := table.GetSelection()
					text := table.GetCell(row, 0).Text
					currentContextName = text
					err := dataSource.SetContext(currentContextName)
					if err != nil {
						ShowNotification(fmt.Sprintf("Failed to swtich context %s", err))
						return event
					}
					ShowNotification(fmt.Sprintf("Switched to context %s", currentContextName))
					go app.QueueUpdateDraw(func() {
						contextInfo.SetText(currentContextName)
					})
					switchToTopicTable(table, dataSource, app)
				}
			}
		}
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'c':
				row, column := table.GetSelection()
				cell := table.GetCell(row, column)
				if cell != nil {
					content := cell.Text
					clipboard.WriteAll(content)
					ShowNotification("😎 Copied selection to clipboard ...")
				}
			}
		}
		return event
	})
}
