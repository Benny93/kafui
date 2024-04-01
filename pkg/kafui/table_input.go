package kafui

import (
	"fmt"
	"strings"

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
					consumeFlags := api.DefaultConsumeFlags()
					topicPage.PageConsumeTopic(topicName, currentTopic, consumeFlags)
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
				CopySelectedRowToClipboard(table, ShowNotification)
			}
		}
		return event
	})
}

// Function to copy the selected row of the table to the clipboard in CSV format
func CopySelectedRowToClipboard(table *tview.Table, ConsumeMessage func(message string)) {
	// Get the selected row index
	row, _ := table.GetSelection()

	// Check if the row index is valid
	if row < 1 || row >= table.GetRowCount() {
		ConsumeMessage("Copy: Invalid row selection")
		return
	}

	// Initialize a slice to hold column values
	var rowValues []string

	// Iterate over each column in the selected row
	for column := 0; column < table.GetColumnCount(); column++ {
		cell := table.GetCell(row, column)
		if cell != nil {
			// Append the cell text to the rowValues slice
			rowValues = append(rowValues, cell.Text)
		}
	}

	// Create a CSV string by joining rowValues with commas
	csvString := strings.Join(rowValues, ",")

	// Copy the CSV string to the clipboard
	err := clipboard.WriteAll(csvString)
	if err != nil {
		// Handle error
		ConsumeMessage("Copy: Error copying CSV string to clipboard")
		return
	}

	// Show notification
	ConsumeMessage("😎 Copied selection to clipboard ...")
}
