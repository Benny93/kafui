package kafui

import "github.com/rivo/tview"

func receivingMessageTopicPage(app *tview.Application, topFlex *tview.Flex, textView *tview.TextView, dataSource KafkaDataSource, msgChannel chan UIEvent) {
	for {
		msg := <-msgChannel
		if msg == OnPageChange {

			app.QueueUpdateDraw(func() {
				topFlex.SetBorder(true).SetTitle("Topic " + currentTopic)
				msgs := consumeTopic(dataSource, currentTopic)
				text := ""
				for _, str := range msgs {
					text += str + "\n"
				}
				textView.SetText(text)

			})
		}
	}
}

func consumeTopic(dataSource KafkaDataSource, topicName string) []string {
	msgs, err := dataSource.ConsumeTopic(topicName)
	if err != nil {
		panic("Error consuming messages")
	}
	return msgs
}

func CreateTopicPage(dataSource KafkaDataSource, pages *tview.Pages, app *tview.Application, msgChannel chan UIEvent) *tview.Flex {

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	topFlex := tview.NewFlex()

	topFlex.SetBorder(true).SetTitle("Topic " + currentTopic)

	midFlex := tview.NewFlex().
		AddItem(textView, 0, 3, true)
	midFlex.SetBorder(true).SetTitle("Messages")

	centralFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topFlex, 0, 1, false).
		AddItem(midFlex, 0, 3, true).
		AddItem(tview.NewFlex().SetBorder(true).SetTitle("Bottom (5 rows)"), 5, 1, false)

	flex := tview.NewFlex().
		AddItem(centralFlex, 0, 2, true)

	go receivingMessageTopicPage(app, topFlex, textView, dataSource, msgChannel)

	return flex
}
