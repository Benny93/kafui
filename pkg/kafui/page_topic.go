package kafui

import (
	"com/emptystate/kafui/pkg/api"
	"fmt"

	"github.com/rivo/tview"
)

func getHandler(app *tview.Application, textView *tview.TextView) api.MessageHandlerFunc {
	return func(msg api.Message) {
		app.QueueUpdateDraw(func() {
			text := textView.GetText(false)
			text += msg.Value + "\n"
			textView.SetText(text)
			textView.ScrollToEnd()
		})
	}
}

func receivingMessageTopicPage(app *tview.Application, pages *tview.Pages, flex *tview.Flex, textView *tview.TextView, dataSource api.KafkaDataSource, msgChannel chan UIEvent) {
	for {
		msg := <-msgChannel
		if msg == OnPageChange {
			frontPage, _ := pages.GetFrontPage()
			if frontPage == "topicPage" {
				app.QueueUpdateDraw(func() {
					flex.SetBorder(true).SetTitle(fmt.Sprintf("<%s>", currentTopic))
					textView.SetText("")
				})
				handlerFunc := getHandler(app, textView)
				err := dataSource.ConsumeTopic(currentTopic, handlerFunc)
				if err != nil {
					panic("Error consume messages!")
				}
			}

		}
	}
}

func CreateTopicPage(dataSource api.KafkaDataSource, pages *tview.Pages, app *tview.Application, msgChannel chan UIEvent) *tview.Flex {

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	topFlex := tview.NewFlex()

	topFlex.SetBorder(false)

	midFlex := tview.NewFlex().
		AddItem(textView, 0, 3, true)
	midFlex.SetBorder(true).SetTitle("Messages")

	centralFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topFlex, 0, 1, false).
		AddItem(midFlex, 0, 5, true).
		AddItem(tview.NewFlex().SetBorder(false), 0, 1, false)

	flex := tview.NewFlex().
		AddItem(centralFlex, 0, 2, true)

	go receivingMessageTopicPage(app, pages, midFlex, textView, dataSource, msgChannel)

	return flex
}
