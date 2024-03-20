package kafui

import (
	"fmt"
	"time"

	"github.com/Benny93/kafui/pkg/api"

	"github.com/rivo/tview"
)

var (
	messagesFlex     *tview.Flex
	messagesTextView *tview.TextView
	reportTextView   *tview.TextView
)

func getHandler(app *tview.Application, textView *tview.TextView, reportV *tview.TextView) api.MessageHandlerFunc {
	return func(msg api.Message) {
		app.QueueUpdateDraw(func() {
			text := textView.GetText(false)
			text += msg.Value + "\n"
			textView.SetText(text)
			textView.ScrollToEnd()
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
			messagesTextView.SetText("")
		})
		handlerFunc := getHandler(app, messagesTextView, reportTextView)
		err := dataSource.ConsumeTopic(currentTopic, handlerFunc)
		if err != nil {
			panic("Error consume messages!")
		}
	}()
}

func CreateTopicPage(dataSource api.KafkaDataSource, pages *tview.Pages, app *tview.Application, msgChannel chan UIEvent) *tview.Flex {

	messagesTextView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	topFlex := tview.NewFlex()

	topFlex.SetBorder(false)

	messagesFlex = tview.NewFlex().
		AddItem(messagesTextView, 0, 3, true)
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
