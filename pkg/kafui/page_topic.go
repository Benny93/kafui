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
	consumerTable      *tview.Table
	cancelConsumption  context.CancelFunc
	cancelRefresh      context.CancelFunc
	messageDetailPage  *DetailPage
	consumedMessages   []api.Message
	newMessageConsumed bool
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
			tp.app.QueueUpdateDraw(func() {
				// Clear the table before updating it
				tp.consumerTable.Clear()
				tp.createFirstRowTopicTable()

				// Iterate over the consumedMessages slice using range
				for _, msg := range tp.consumedMessages {
					rowIndex := tp.consumerTable.GetRowCount() // Get the current row index
					tp.consumerTable.SetCell(rowIndex, 0, tview.NewTableCell(strconv.FormatInt(msg.Offset, 10)))
					tp.consumerTable.SetCell(rowIndex, 1, tview.NewTableCell(msg.Key))
					cell := tview.NewTableCell(msg.Value)
					cell.SetExpansion(1)
					tp.consumerTable.SetCell(rowIndex, 2, cell)
				}
				tp.consumerTable.ScrollToEnd()
				tp.consumerTable.Select(tp.consumerTable.GetRowCount()-1, 0) // Select the last row
			})
		}
	}
}

func (tp *TopicPage) PageConsumeTopic(currentTopic string) {
	var emptyArray []api.Message
	tp.consumedMessages = emptyArray
	go func() {
		tp.app.QueueUpdateDraw(func() {
			tp.createFirstRowTopicTable()
		})
		handlerFunc := tp.getHandler()
		ctx, cancel := context.WithCancel(context.Background())
		tp.cancelConsumption = cancel
		flags := api.DefaultConsumeFlags()
		err := tp.dataSource.ConsumeTopic(ctx, currentTopic, flags, handlerFunc)
		if err != nil {
			panic("Error consume messages!")
		}
	}()
	ctx, c := context.WithCancel(context.Background())
	tp.cancelRefresh = c
	go tp.refreshTopicTable(ctx)
}

func (tp *TopicPage) createFirstRowTopicTable() {
	tp.messagesFlex.SetBorder(true).SetTitle(fmt.Sprintf("<%s>", currentTopic))
	tp.consumerTable.SetCell(0, 0, tview.NewTableCell("Offset").SetTextColor(tview.Styles.SecondaryTextColor))
	tp.consumerTable.SetCell(0, 1, tview.NewTableCell("Key").SetTextColor(tview.Styles.SecondaryTextColor))
	tp.consumerTable.SetCell(0, 2, tview.NewTableCell("Value").SetTextColor(tview.Styles.SecondaryTextColor).SetExpansion(1))

}

func (tp *TopicPage) handleEnter() func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			// Get the selected row index
			row, _ := tp.consumerTable.GetSelection()
			valueCell := tp.consumerTable.GetCell(row, 2)
			// Display the value content in a new page
			if row > 0 {
				tp.messageDetailPage = NewDetailPage(tp.app, tp.pages, valueCell.Text)
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

	topFlex := tview.NewFlex()
	topFlex.SetBorder(false)

	tp.messagesFlex = tview.NewFlex().
		AddItem(tp.consumerTable, 0, 3, true)
	tp.messagesFlex.SetBorder(true).SetTitle("Messages")

	bottomFlex := tview.NewFlex()

	centralFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topFlex, 5, 1, false).
		AddItem(tp.messagesFlex, 0, 5, true).
		AddItem(bottomFlex, 5, 1, false)

	flex := tview.NewFlex().
		AddItem(centralFlex, 0, 2, true)

	return flex
}

func (tp *TopicPage) CloseTopicPage() {
	go func() {
		tp.consumerTable.Clear()

		tp.cancelConsumption()
		tp.cancelRefresh()
	}()
}
