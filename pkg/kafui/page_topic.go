package kafui

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/gdamore/tcell/v2"

	"github.com/rivo/tview"
)

var (
	messagesFlex         *tview.Flex
	consumerTable        *tview.Table
	consumerTableNextRow int
	reportTextView       *tview.TextView
	cancelConsumption    context.CancelFunc
	messageDetailPage    *DetailPage
)

func getHandler(app *tview.Application, table *tview.Table, reportV *tview.TextView) api.MessageHandlerFunc {
	return func(msg api.Message) {
		app.QueueUpdateDraw(func() {

			table.SetCell(consumerTableNextRow, 0, tview.NewTableCell(strconv.FormatInt(msg.Offset, 10)))
			table.SetCell(consumerTableNextRow, 1, tview.NewTableCell(msg.Key))
			cell := tview.NewTableCell(msg.Value)
			cell.SetExpansion(1)
			table.SetCell(consumerTableNextRow, 2, cell)
			consumerTableNextRow++
			table.ScrollToEnd()

		})
		ReportConsumption(fmt.Sprintf("Consumed message at offset %d", msg.Offset), reportV)
	}
}

func ReportConsumption(message string, textView *tview.TextView) {
	go func() {
		tviewApp.QueueUpdateDraw(func() {
			textView.SetText(message)
		})
		// Schedule hiding TextView after 2 seconds

		time.Sleep(1 * time.Second)
		tviewApp.QueueUpdateDraw(func() {
			textView.SetText("")
		})
	}()
}

func PageConsumeTopic(app *tview.Application, dataSource api.KafkaDataSource) {
	go func() {
		app.QueueUpdateDraw(func() {
			messagesFlex.SetBorder(true).SetTitle(fmt.Sprintf("<%s>", currentTopic))
			consumerTable.SetCell(0, 0, tview.NewTableCell("Offset").SetTextColor(tview.Styles.SecondaryTextColor))
			consumerTable.SetCell(0, 1, tview.NewTableCell("Key").SetTextColor(tview.Styles.SecondaryTextColor))
			consumerTable.SetCell(0, 2, tview.NewTableCell("Value").SetTextColor(tview.Styles.SecondaryTextColor).SetExpansion(1))
			consumerTableNextRow = 1
		})
		handlerFunc := getHandler(app, consumerTable, reportTextView)
		ctx, cancel := context.WithCancel(context.Background())
		cancelConsumption = cancel
		err := dataSource.ConsumeTopic(ctx, currentTopic, handlerFunc)
		if err != nil {
			panic("Error consume messages!")
		}
	}()
}
func handleEnter(table *tview.Table, app *tview.Application, pages *tview.Pages) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			// Get the selected row index
			row, _ := table.GetSelection()
			valueCell := table.GetCell(row, 2)
			// Display the value content in a new page
			if row > 0 {
				messageDetailPage = NewDetailPage(app, pages, valueCell.Text)
				messageDetailPage.Show()
			}
		}
		return event
	}
}

func CreateTopicPage(dataSource api.KafkaDataSource, pages *tview.Pages, app *tview.Application, msgChannel chan UIEvent) *tview.Flex {

	// consumerTable = tview.NewTextView().
	// SetDynamicColors(true).
	// SetRegions(true).
	// SetChangedFunc(func() {
	// 	app.Draw()
	// })

	consumerTable = tview.NewTable()
	consumerTable.SetSelectable(true, false)
	consumerTable.SetFixed(1, 1)
	consumerTable.SetInputCapture(handleEnter(consumerTable, app, pages))

	topFlex := tview.NewFlex()

	topFlex.SetBorder(false)

	messagesFlex = tview.NewFlex().
		AddItem(consumerTable, 0, 3, true)
	messagesFlex.SetBorder(true).SetTitle("Messages")

	reportTextView = createNotificationTextView()

	bottomFlex := tview.NewFlex().
		AddItem(reportTextView, 0, 1, false)

	centralFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topFlex, 5, 1, false).
		AddItem(messagesFlex, 0, 5, true).
		AddItem(bottomFlex, 5, 1, false)

	flex := tview.NewFlex().
		AddItem(centralFlex, 0, 2, true)

	return flex
}

func CloseTopicPage() {

	go func() {
		consumerTable.Clear()
		consumerTableNextRow = 0
		cancelConsumption()
	}()
}
