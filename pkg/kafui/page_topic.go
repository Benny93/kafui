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

type TopicPage struct {
	app                *tview.Application
	dataSource         api.KafkaDataSource
	pages              *tview.Pages
	msgChannel         chan UIEvent
	messagesFlex       *tview.Flex
	topFlex            *tview.Flex
	topicInfoFlex      *tview.Flex
	consumerTable      *tview.Table
	cancelConsumption  context.CancelFunc
	cancelRefresh      context.CancelFunc
	messageDetailPage  *DetailPage
	consumedMessages   []api.Message
	newMessageConsumed bool
	notifyView         *tview.TextView
	topicName          string
}

func NewTopicPage(dataSource api.KafkaDataSource, pages *tview.Pages, app *tview.Application, msgChannel chan UIEvent) *TopicPage {
	return &TopicPage{
		app:        app,
		dataSource: dataSource,
		pages:      pages,
		msgChannel: msgChannel,
	}
}

func (tp *TopicPage) getHandler() api.MessageHandlerFunc {
	var empty []api.Message
	tp.consumedMessages = empty
	return func(msg api.Message) {
		tp.consumedMessages = append(tp.consumedMessages, msg)
		tp.newMessageConsumed = true
	}
}

func (tp *TopicPage) refreshTopicTable(ctx context.Context) {
	refreshTicker := time.NewTicker(100 * time.Millisecond)
	defer refreshTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Exit the function if the context is done
			return
		case <-refreshTicker.C:
			if !tp.newMessageConsumed {
				continue
			}
			tp.newMessageConsumed = false
			tp.ShowNotification(fmt.Sprintf("Consumed messages %d", len(tp.consumedMessages)))
			tp.app.QueueUpdateDraw(func() {
				// Clear the table before updating it
				tp.consumerTable.Clear()
				tp.createFirstRowTopicTable(tp.topicName)

				// Iterate over the consumedMessages slice using range
				for _, msg := range tp.consumedMessages {
					rowIndex := tp.consumerTable.GetRowCount() // Get the current row index
					tp.consumerTable.SetCell(rowIndex, 0, tview.NewTableCell(strconv.FormatInt(msg.Offset, 10)))
					tp.consumerTable.SetCell(rowIndex, 1, tview.NewTableCell(fmt.Sprint(msg.Partition)))
					tp.consumerTable.SetCell(rowIndex, 2, tview.NewTableCell(fmt.Sprint(msg.KeySchemaID)))
					tp.consumerTable.SetCell(rowIndex, 3, tview.NewTableCell(fmt.Sprint(msg.ValueSchemaID)))
					tp.consumerTable.SetCell(rowIndex, 4, tview.NewTableCell(msg.Key))
					shortenedText := tp.shortValue(msg)
					cell := tview.NewTableCell(shortenedText)
					cell.SetExpansion(1)
					tp.consumerTable.SetCell(rowIndex, 5, cell)
				}
				tp.consumerTable.ScrollToEnd()
				tp.consumerTable.Select(tp.consumerTable.GetRowCount()-1, 0) // Select the last row
			})
		}
	}
}

func (*TopicPage) shortValue(msg api.Message) string {
	if len(msg.Value) <= 100 {
		return msg.Value
	}
	shortenedText := msg.Value[:100]
	if len(shortenedText) < len(msg.Value) {
		shortenedText = shortenedText + "..."
	}
	return shortenedText
}

func (tp *TopicPage) PageConsumeTopic(topicName string, currentTopic api.Topic) {
	tp.topicName = topicName
	tp.topicInfoFlex = tp.CreateTopicInfoSection(topicName, currentTopic)
	tp.topFlex.AddItem(tp.topicInfoFlex, 0, 1, false)
	tp.ShowNotification("Consuming messages...")
	var emptyArray []api.Message
	tp.consumedMessages = emptyArray
	go func() {
		tp.app.QueueUpdateDraw(func() {
			tp.createFirstRowTopicTable(topicName)
		})
		handlerFunc := tp.getHandler()
		ctx, cancel := context.WithCancel(context.Background())
		tp.cancelConsumption = cancel
		flags := api.DefaultConsumeFlags()
		err := tp.dataSource.ConsumeTopic(ctx, topicName, flags, handlerFunc)
		if err != nil {
			panic("Error consume messages!")
		}
	}()
	ctx, c := context.WithCancel(context.Background())
	tp.cancelRefresh = c
	go tp.refreshTopicTable(ctx)
}

func (tp *TopicPage) createFirstRowTopicTable(topicName string) {
	tp.messagesFlex.SetBorder(true).SetTitle(fmt.Sprintf("<%s>", topicName))
	tp.consumerTable.SetCell(0, 0, tview.NewTableCell("Offset").SetTextColor(tview.Styles.SecondaryTextColor))
	tp.consumerTable.SetCell(0, 1, tview.NewTableCell("Partition").SetTextColor(tview.Styles.SecondaryTextColor))
	tp.consumerTable.SetCell(0, 2, tview.NewTableCell("KeySchemaID").SetTextColor(tview.Styles.SecondaryTextColor))
	tp.consumerTable.SetCell(0, 3, tview.NewTableCell("ValueSchemaID").SetTextColor(tview.Styles.SecondaryTextColor))
	tp.consumerTable.SetCell(0, 4, tview.NewTableCell("Key").SetTextColor(tview.Styles.SecondaryTextColor))
	tp.consumerTable.SetCell(0, 5, tview.NewTableCell("Value").SetTextColor(tview.Styles.SecondaryTextColor).SetExpansion(1))

}

func (tp *TopicPage) handleEnter() func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			// Get the selected row index
			row, _ := tp.consumerTable.GetSelection()
			// Display the value content in a new page
			if row > 0 {
				msgv := tp.consumedMessages[row-1].Value
				tp.messageDetailPage = NewDetailPage(tp.app, tp.pages, msgv)
				tp.messageDetailPage.Show()
			}
		}
		return event
	}
}

func (tp *TopicPage) CreateTopicPage(currentTopic string) *tview.Flex {
	tp.consumerTable = tview.NewTable()
	tp.consumerTable.SetSelectable(true, false)
	tp.consumerTable.SetFixed(1, 1)
	tp.consumerTable.SetInputCapture(tp.handleEnter())

	tp.topFlex = tview.NewFlex()
	tp.topFlex.SetBorder(false)

	tp.messagesFlex = tview.NewFlex().
		AddItem(tp.consumerTable, 0, 3, true)
	tp.messagesFlex.SetBorder(true).SetTitle("Messages")

	bottomFlex := tview.NewFlex()
	tp.notifyView = tview.NewTextView().SetText("Notification...")
	tp.notifyView.SetBorder(false)
	bottomFlex.AddItem(tp.notifyView, 0, 1, false)

	centralFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tp.topFlex, 5, 1, false).
		AddItem(tp.messagesFlex, 0, 5, true).
		AddItem(bottomFlex, 5, 1, false)

	flex := tview.NewFlex().
		AddItem(centralFlex, 0, 2, true)

	return flex
}

func (tp *TopicPage) CreateTopicInfoSection(topicName string, topicDetail api.Topic) *tview.Flex {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBorderPadding(0, 0, 1, 0)
	//flex.SetBorder(true)
	flex.
		AddItem(CreatePropertyInfo("Name", topicName), 0, 1, false).
		AddItem(CreatePropertyInfo("MessageCount", fmt.Sprint(topicDetail.MessageCount)), 0, 1, false).
		AddItem(CreatePropertyInfo("Number of Partitions", fmt.Sprint(topicDetail.NumPartitions)), 0, 1, false).
		AddItem(CreatePropertyInfo("Replication Factor", fmt.Sprint(topicDetail.ReplicationFactor)), 0, 1, false)

	return flex
}

func (tp *TopicPage) CloseTopicPage() {
	go func() {
		tp.consumerTable.Clear()

		tp.cancelConsumption()
		tp.cancelRefresh()
		if tp.topicInfoFlex != nil {
			tp.topFlex.RemoveItem(tp.topicInfoFlex)
		}

	}()
}

func (tp *TopicPage) ShowNotification(message string) {
	go func() {
		tp.app.QueueUpdateDraw(func() {
			tp.notifyView.SetText(message)
		})
		// Schedule hiding TextView after 2 seconds

		time.Sleep(2 * time.Second)
		tp.app.QueueUpdateDraw(func() {
			tp.notifyView.SetText("")
		})
	}()
}
