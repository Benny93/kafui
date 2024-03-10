package kafui

import (
	"com/emptystate/kafui/pkg/api"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var currentResouce string = Topic[0] // Topic is the default

var notificationTextView *tview.TextView

func receivingMessage(app *tview.Application, table *tview.Table, searchInput *tview.InputField, msgChannel chan UIEvent) {
	for {
		msg := <-msgChannel
		if msg == OnModalClose {
			app.SetFocus(table)
		}
		if msg == OnFocusSearch {
			searchInput.SetLabel("🧐>")
			app.SetFocus(searchInput)
		}
	}
}

func CreateMainPage(dataSource api.KafkaDataSource, pages *tview.Pages, app *tview.Application, modal *tview.Modal, msgChannel chan UIEvent) *tview.Flex {

	table := tview.NewTable().SetBorders(false)
	table.SetSelectable(true, false)
	table.SetFixed(1, 1)

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			// Check if the table has focus
			if app.GetFocus() == table {
				if currentResouce == Topic[0] {
					row, _ := table.GetSelection()
					text := table.GetCell(row, 0).Text
					currentTopic = text
					msgChannel <- OnPageChange
					pages.SwitchToPage("topicPage")
				}
			}
		}
		return event
	})

	defaultLabel := "😎|"
	searchInput := createSearchInput(defaultLabel, table, dataSource, pages, app, modal)
	contextInfo := createContextInfo()
	topics := fetchTopics(dataSource)

	showTopicsInTable(table, topics)

	topFlex := tview.NewFlex().
		AddItem(contextInfo, 0, 1, false).
		AddItem(searchInput, 0, 1, true).SetDirection(tview.FlexRow)

	//topFlex.SetBorder(false).SetTitle("Top")

	midFlex := tview.NewFlex().
		AddItem(table, 0, 3, true)
	midFlex.SetBorder(true).SetTitle(currentResouce)

	notificationTextView = createNotificationTextView()

	bottomFlex := tview.NewFlex().
		AddItem(notificationTextView, 0, 3, false)

	centralFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topFlex, 0, 1, false).
		AddItem(midFlex, 0, 3, true).
		AddItem(bottomFlex, 5, 1, false)

	flex := tview.NewFlex().
		AddItem(centralFlex, 0, 2, true)

	go receivingMessage(app, table, searchInput, msgChannel)

	ShowNotification("Fetched topics...")
	return flex
}

func ShowNotification(message string) {
	go func() {
		tviewApp.QueueUpdateDraw(func() {
			notificationTextView.SetText(message)
		})
		// Schedule hiding TextView after 2 seconds

		time.Sleep(2 * time.Second)
		tviewApp.QueueUpdateDraw(func() {
			notificationTextView.SetText("")
		})
	}()
}

func createNotificationTextView() *tview.TextView {
	textView := tview.NewTextView().SetText("Notification...")
	textView.SetBorder(false)
	return textView
}

func createContextInfo() *tview.InputField {
	inputField := tview.NewInputField().
		SetLabel("Current Context: ").
		SetFieldWidth(10).
		SetText("n/a")

	inputField.SetDisabled(true)
	return inputField
}

func createSearchInput(defaultLabel string, table *tview.Table, dataSource api.KafkaDataSource, pages *tview.Pages, app *tview.Application, modal *tview.Modal) *tview.InputField {
	searchInput := tview.NewInputField().
		SetLabel(defaultLabel).
		SetFieldWidth(0)
	searchInput.SetBorder(true).SetBackgroundColor(tcell.ColorBlack).SetBorderColor(tcell.ColorDarkCyan.TrueColor())
	searchInput.SetFieldBackgroundColor(tcell.ColorBlack)
	selectedStyle := tcell.Style{}
	selectedStyle.Background(tcell.ColorWhite)
	searchInput.SetAutocompleteStyles(tcell.ColorBlue, tcell.Style{}, selectedStyle)
	searchText := ""

	searchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			searchText = searchInput.GetText()
			match := false
			if Contains(Context, searchText) {
				table.Clear()
				searchInput.SetLabel(defaultLabel)
				contexts := fetchContexts(dataSource)
				showContextsInTable(table, contexts)
				match = true
				currentResouce = Context[0]
				ShowNotification("Fetched Contexts ...")
			}

			if Contains(Topic, searchText) {
				table.Clear()
				topics := fetchTopics(dataSource)
				showTopicsInTable(table, topics)
				match = true
				currentResouce = Topic[0]
				ShowNotification("Fetched Topics ...")
			}

			if Contains(ConsumerGroup, searchText) {
				table.Clear()
				cgs := fetchConsumerGroups(dataSource)
				showConsumerGroups(table, cgs)
				match = true
				currentResouce = ConsumerGroup[0]
				ShowNotification("Fetched Consumer Groups ...")
			}
			if !match {
				pages.ShowPage("modal")
				app.SetFocus(modal)
			} else {
				app.SetFocus(table)
			}
			searchInput.SetLabel(defaultLabel)
			searchInput.SetText("")

		}
	})

	searchInput.SetAutocompleteFunc(func(currentText string) (entries []string) {
		if len(currentText) == 0 {
			return
		}
		words := append(append(Context, Topic...), ConsumerGroup...)
		for _, word := range words {
			if strings.HasPrefix(strings.ToLower(word), strings.ToLower(currentText)) {
				entries = append(entries, word)
			}
		}
		if len(entries) <= 1 {
			entries = nil
		}
		return
	})

	return searchInput
}

func showConsumerGroups(table *tview.Table, cgs []api.ConsumerGroup) {
	// Define table headers
	table.SetCell(0, 0, tview.NewTableCell("Name").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetCell(0, 1, tview.NewTableCell("State").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetCell(0, 2, tview.NewTableCell("Consumers").SetTextColor(tview.Styles.SecondaryTextColor))

	// Add data for each consumer group
	for i, cg := range cgs {
		// Add data to the table
		cell := tview.NewTableCell(cg.Name)
		table.SetCell(i+1, 0, cell)
		table.SetCell(i+1, 1, tview.NewTableCell(cg.State))
		table.SetCell(i+1, 2, tview.NewTableCell(strconv.Itoa(cg.Consumers)).SetExpansion(1))
	}
}

func fetchConsumerGroups(dataSource api.KafkaDataSource) []api.ConsumerGroup {
	cgs, err := dataSource.GetConsumerGroups()
	if err != nil {
		ShowNotification(fmt.Sprintf("Error fetching GetConsumerGroups:", err))
		return []api.ConsumerGroup{}
	}
	return cgs
}

func showContextsInTable(table *tview.Table, contexts []string) {
	table.SetCell(0, 0, tview.NewTableCell("Context").SetTextColor(tview.Styles.SecondaryTextColor))
	for i, context := range contexts {
		cell := tview.NewTableCell(context)
		cell.SetExpansion(1)
		table.SetCell(i+1, 0, cell)
	}
	table.SetTitle(currentResouce)
}

func fetchContexts(dataSource api.KafkaDataSource) []string {
	contexts, err := dataSource.GetContexts()
	if err != nil {
		ShowNotification(fmt.Sprintf("Error fetching contexts:", err))
		return []string{}
	}
	return contexts
}

func fetchTopics(dataSource api.KafkaDataSource) []string {
	topics, err := dataSource.GetTopics()
	if err != nil {
		ShowNotification(fmt.Sprintf("Error reading topics:", err))
		return []string{}
	}
	return topics
}

func showTopicsInTable(table *tview.Table, topics []string) {
	table.SetCell(0, 0, tview.NewTableCell("Topics").SetTextColor(tview.Styles.SecondaryTextColor))

	for i, topic := range topics {
		cell := tview.NewTableCell(topic)
		cell.SetExpansion(1)
		table.SetCell(i+1, 0, cell)
	}
	table.SetTitle(currentResouce)
}
