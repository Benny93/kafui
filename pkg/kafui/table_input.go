package kafui

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/api"

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
				r, _ := table.GetSelection()
				if r == 0 {
					return event
				}

				if currentResouce == Topic[0] {
					row, _ := table.GetSelection()
					topicName := table.GetCell(row, 0).Text

					currentTopic = lastFetchedTopics[topicName]
					msgChannel <- OnPageChange
					pages.SwitchToPage("topicPage")
					topicPage.PageConsumeTopic(topicName, currentTopic)
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
