package kafui

import (
	"container/list"
	"context"
	"fmt"
	"strconv"
	"strings"
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
	topFlexElements    *list.List
	consumerTable      *tview.Table
	cancelConsumption  context.CancelFunc
	cancelRefresh      context.CancelFunc
	messageDetailPage  *DetailPage
	consumedMessages   map[string]api.Message
	newMessageConsumed bool
	notifyView         *tview.TextView
	topicName          string
	topicDetails       api.Topic
	consumeFlags       api.ConsumeFlags
	searchText         string
	tableSearch        *tview.Flex
	bottomFlex         *tview.Flex
	requireRefresh     bool
}

func NewTopicPage(dataSource api.KafkaDataSource, pages *tview.Pages, app *tview.Application, msgChannel chan UIEvent) *TopicPage {
	return &TopicPage{
		app:             app,
		dataSource:      dataSource,
		pages:           pages,
		msgChannel:      msgChannel,
		topFlexElements: list.New(),
	}
}

func (tp *TopicPage) getHandler() api.MessageHandlerFunc {

	tp.consumedMessages = make(map[string]api.Message)
	return func(msg api.Message) {
		key := tp.getMessageKey(fmt.Sprint(msg.Partition), fmt.Sprint(msg.Offset))
		tp.consumedMessages[key] = msg
		tp.newMessageConsumed = true
	}
}

func (tp *TopicPage) getMessageKey(partition string, offset string) string {
	return fmt.Sprintf("%s:%s", partition, offset)
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
			if !tp.newMessageConsumed && !tp.requireRefresh {
				continue
			}
			tp.requireRefresh = false
			tp.newMessageConsumed = false
			//tp.ShowNotification(fmt.Sprintf("Consumed messages %d", len(tp.consumedMessages)))
			tp.app.QueueUpdateDraw(func() {
				// Clear the table before updating it
				tp.consumerTable.Clear()
				tp.createFirstRowTopicTable(tp.topicName)

				// Iterate over the consumedMessages slice using range
				for _, msg := range tp.consumedMessages {
					if tp.searchText != "" {
						// skip message if none field fuzzy matches search text
						if !fieldFuzzyMatchesSearchText(msg, tp.searchText) {
							continue // Skip this message
						}
					}
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

func fieldFuzzyMatchesSearchText(msg api.Message, searchText string) bool {

	// Convert all fields to lowercase for case-insensitive comparison
	searchText = strings.ToLower(searchText)

	// Check if any field contains the search text in a fuzzy manner
	if strings.Contains(strings.ToLower(strconv.FormatInt(msg.Offset, 10)), searchText) ||
		strings.Contains(strings.ToLower(fmt.Sprint(msg.Partition)), searchText) ||
		strings.Contains(strings.ToLower(fmt.Sprint(msg.KeySchemaID)), searchText) ||
		strings.Contains(strings.ToLower(fmt.Sprint(msg.ValueSchemaID)), searchText) ||
		strings.Contains(strings.ToLower(msg.Key), searchText) ||
		strings.Contains(strings.ToLower(msg.Value), searchText) {
		return true
	}

	return false
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

func (tp *TopicPage) PageConsumeTopic(topicName string, currentTopic api.Topic, flags api.ConsumeFlags) {
	tp.topicName = topicName
	tp.topicDetails = currentTopic
	tp.consumeFlags = flags
	// topic info
	topicInfoFlex := tp.CreateTopicInfoSection(topicName, currentTopic)
	tp.topFlexElements.PushBack(topicInfoFlex)
	tp.topFlex.AddItem(topicInfoFlex, 0, 1, false)
	// consumer flags
	consumerFlagsFlex := tp.CreateConsumeFlagsSection()
	tp.topFlexElements.PushBack(consumerFlagsFlex)
	tp.topFlex.AddItem(consumerFlagsFlex, 0, 1, false)
	// input legend
	inputLegend := tp.CreateInputLegend()
	tp.topFlexElements.PushBack(inputLegend)
	tp.topFlex.AddItem(inputLegend, 0, 1, false)

	tp.ShowNotification("Consuming messages...")

	tp.consumedMessages = make(map[string]api.Message)
	go func() {
		tp.app.QueueUpdateDraw(func() {
			tp.createFirstRowTopicTable(topicName)
		})
		handlerFunc := tp.getHandler()
		ctx, cancel := context.WithCancel(context.Background())
		tp.cancelConsumption = cancel
		err := tp.dataSource.ConsumeTopic(ctx, topicName, tp.consumeFlags, handlerFunc)
		if err != nil {
			panic("Error consume messages!")
		}
	}()
	ctx, c := context.WithCancel(context.Background())
	tp.cancelRefresh = c
	go tp.refreshTopicTable(ctx)
}

func (tp *TopicPage) createFirstRowTopicTable(topicName string) {
	title := fmt.Sprintf("<%s>", topicName)
	if tp.searchText != "" {
		title = fmt.Sprintf("<%s/%s>", topicName, tp.searchText)
	}
	tp.messagesFlex.SetBorder(true).SetTitle(title)
	tp.consumerTable.SetCell(0, 0, tview.NewTableCell("Offset").SetTextColor(tview.Styles.SecondaryTextColor))
	tp.consumerTable.SetCell(0, 1, tview.NewTableCell("Partition").SetTextColor(tview.Styles.SecondaryTextColor))
	tp.consumerTable.SetCell(0, 2, tview.NewTableCell("KeySID").SetTextColor(tview.Styles.SecondaryTextColor))
	tp.consumerTable.SetCell(0, 3, tview.NewTableCell("ValueSID").SetTextColor(tview.Styles.SecondaryTextColor))
	tp.consumerTable.SetCell(0, 4, tview.NewTableCell("Key").SetTextColor(tview.Styles.SecondaryTextColor))
	tp.consumerTable.SetCell(0, 5, tview.NewTableCell("Value").SetTextColor(tview.Styles.SecondaryTextColor).SetExpansion(1))

}

func (tp *TopicPage) inputCapture() func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			// Get the selected row index
			row, _ := tp.consumerTable.GetSelection()
			off := tp.consumerTable.GetCell(row, 0)
			part := tp.consumerTable.GetCell(row, 1)
			key := tp.getMessageKey(part.Text, off.Text)
			// Display the value content in a new page
			if row > 0 {
				msgv := tp.consumedMessages[key].Value
				tp.messageDetailPage = NewDetailPage(tp.app, tp.pages, msgv)
				tp.messageDetailPage.Show()
			}
		}
		if event.Rune() == 'g' {
			// Handle 'g' key event
			tp.consumerTable.ScrollToBeginning()
		}
		if event.Rune() == 'G' {
			tp.consumerTable.ScrollToEnd()
		}
		if event.Rune() == '/' {
			if tp.tableSearch == nil {
				tp.tableSearch = tp.CreateInputSearch(func() {
					tp.bottomFlex.RemoveItem(tp.tableSearch)
					tp.app.SetFocus(tp.consumerTable)
					tp.tableSearch = nil
				})
				tp.bottomFlex.AddItem(tp.tableSearch, 0, 1, false)
			}
			tp.app.SetFocus(tp.tableSearch)
		}
		if event.Rune() == 'o' {
			// Toggle between "newest" and "oldest" values
			if tp.consumeFlags.OffsetFlag == "latest" {
				tp.consumeFlags.OffsetFlag = "oldest"
				tp.consumeFlags.Tail = 0
			} else {
				tp.consumeFlags.OffsetFlag = "latest"
				tp.consumeFlags.Tail = 50
			}
			tp.RestartConsumer()
		}
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'c':
				CopySelectedRowToClipboard(tp.consumerTable, tp.ShowNotification)
			}
		}
		return event
	}
}

func (tp *TopicPage) RestartConsumer() {
	tp.clearConsumedData()
	tp.PageConsumeTopic(tp.topicName, tp.topicDetails, tp.consumeFlags)
}

func (tp *TopicPage) CreateTopicPage(currentTopic string) *tview.Flex {
	tp.consumerTable = tview.NewTable()
	tp.consumerTable.SetSelectable(true, false)
	tp.consumerTable.SetFixed(1, 1)
	tp.consumerTable.SetInputCapture(tp.inputCapture())

	tp.topFlex = tview.NewFlex()
	tp.topFlex.SetBorder(false)

	tp.messagesFlex = tview.NewFlex().
		AddItem(tp.consumerTable, 0, 3, true)
	tp.messagesFlex.SetBorder(true).SetTitle("Messages")

	tp.bottomFlex = tview.NewFlex().SetDirection(tview.FlexRow)
	tp.notifyView = tview.NewTextView().SetText("Notification...")
	tp.notifyView.SetBorder(false)
	tp.bottomFlex.AddItem(tp.notifyView, 0, 1, false)

	centralFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tp.topFlex, 5, 1, false).
		AddItem(tp.messagesFlex, 0, 5, true).
		AddItem(tp.bottomFlex, 5, 1, false)

	flex := tview.NewFlex().
		AddItem(centralFlex, 0, 2, true)

	return flex
}

func (tp *TopicPage) CreateTopicInfoSection(topicName string, topicDetail api.Topic) *tview.Flex {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBorderPadding(0, 0, 1, 0)
	//flex.SetBorder(true)
	flex.
		AddItem(CreatePropertyInfo("Topic Name", topicName), 0, 1, false).
		AddItem(CreatePropertyInfo("Number of Messages", fmt.Sprint(topicDetail.MessageCount)), 0, 1, false).
		AddItem(CreatePropertyInfo("Number of Partitions", fmt.Sprint(topicDetail.NumPartitions)), 0, 1, false).
		AddItem(CreatePropertyInfo("Replication Factor", fmt.Sprint(topicDetail.ReplicationFactor)), 0, 1, false)

	return flex
}

func (tp *TopicPage) CreateConsumeFlagsSection() *tview.Flex {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBorderPadding(0, 0, 1, 0)
	//flex.SetBorder(true)
	flex.
		AddItem(CreatePropertyInfo("From Offset", tp.consumeFlags.OffsetFlag), 0, 1, false).
		AddItem(CreatePropertyInfo("Follow", fmt.Sprint(tp.consumeFlags.Follow)), 0, 1, false).
		AddItem(CreatePropertyInfo("Tail", fmt.Sprint(tp.consumeFlags.Tail)), 0, 1, false)

	return flex
}

func (tp *TopicPage) CloseTopicPage() {
	go func() {
		tp.clearConsumedData()

	}()
}

func (tp *TopicPage) clearConsumedData() {
	tp.consumerTable.Clear()
	tp.searchText = ""
	if tp.tableSearch != nil {
		tp.bottomFlex.RemoveItem(tp.tableSearch)
		tp.tableSearch = nil
	}

	tp.cancelConsumption()
	tp.cancelRefresh()

	for e := tp.topFlexElements.Front(); e != nil; e = e.Next() {
		if value, ok := e.Value.(tview.Primitive); ok {
			tp.topFlex.RemoveItem(value)
		}
	}
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

func (tp *TopicPage) CreateInputLegend() *tview.Flex {
	flex := tview.NewFlex()
	flex.SetBorderPadding(0, 0, 1, 0)
	left := tview.NewFlex().SetDirection(tview.FlexRow)
	right := tview.NewFlex().SetDirection(tview.FlexRow)
	right.SetBorderPadding(0, 1, 0, 0)

	left.AddItem(CreateRunInfo("↑", "Move up"), 0, 1, false)
	left.AddItem(CreateRunInfo("↓", "Move down"), 0, 1, false)
	left.AddItem(CreateRunInfo("g", "Scroll to top"), 0, 1, false)
	left.AddItem(CreateRunInfo("G", "Scroll to bottom"), 0, 1, false)
	left.AddItem(CreateRunInfo("c", "Copy current line"), 0, 1, false)
	right.AddItem(CreateRunInfo("Enter", "Show value"), 0, 1, false)
	right.AddItem(CreateRunInfo("Esc", "Go Back"), 0, 1, false)
	right.AddItem(CreateRunInfo("o", "Toggle start offset"), 0, 1, false)
	right.AddItem(CreateRunInfo("/", "Search Table"), 0, 1, false)

	flex.AddItem(left, 0, 1, false)
	flex.AddItem(right, 0, 1, false)

	return flex
}

func (tp *TopicPage) CreateInputSearch(onDone func()) *tview.Flex {
	i := tview.NewInputField().
		SetLabel("💡?").
		SetFieldWidth(0)
	i.SetText(tp.searchText)
	i.SetBorder(true).SetBorderColor(tcell.ColorDarkCyan.TrueColor())
	i.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			tp.searchText = i.GetText()
			tp.requireRefresh = true
			onDone()
		}
	})

	i.SetChangedFunc(func(text string) {
		tp.searchText = text
		tp.requireRefresh = true
	})

	f := tview.NewFlex()
	f.SetBorder(false)
	f.AddItem(i, 0, 1, true)
	return f
}
