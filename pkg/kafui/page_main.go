package kafui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/fvbommel/sortorder"

	"github.com/maruel/natural"
	"github.com/rivo/tview"
)

const refreshInterval = 5000 * time.Millisecond

var (
	currentResouce string = Topic[0] // Topic is the default

	currentContextName string

	notificationTextView *tview.TextView

	midFlex *tview.Flex

	contextInfo *tview.InputField

	currentSearchMode SearchMode = ResouceSearch

	currentSearchString string = ""

	lastFetchedTopics map[string]api.Topic
)

func receivingMessage(app *tview.Application, table *tview.Table, searchInput *tview.InputField, msgChannel chan UIEvent) {
	for {
		msg := <-msgChannel
		if msg == OnModalClose {
			app.SetFocus(table)
		}
		if msg == OnFocusSearch {
			searchInput.SetLabel("🧐>")
			searchInput.SetText("")
			app.SetFocus(searchInput)
			currentSearchMode = ResouceSearch
			currentSearchString = ""
		}
		if msg == OnStartTableSearch {
			searchInput.SetLabel("💡?")
			app.SetFocus(searchInput)
			currentSearchMode = TableSearch
			currentSearchString = ""
		}
	}
}

func currentTimeString() string {
	t := time.Now()
	return fmt.Sprintf(t.Format("Current time is 15:04"))
}

func updateTableRoutine(app *tview.Application, table *tview.Table, timerView *tview.TextView, dataSource api.KafkaDataSource) {
	for {
		app.QueueUpdateDraw(func() {
			timerView.SetText(currentTimeString())
			UpdateTable(table, dataSource)

		})
		time.Sleep(refreshInterval)
	}
}

func UpdateTable(table *tview.Table, dataSource api.KafkaDataSource) {
	if currentResouce == Topic[0] {
		table.Clear()
		topics := fetchTopics(dataSource)
		showTopicsInTable(table, topics)
		currentResouce = Topic[0]
		//ShowNotification("Fetched Topics ...")
		updateMidFlexTitle(currentResouce, table.GetRowCount())
	} else if currentResouce == Context[0] {
		table.Clear()
		contexts := fetchContexts(dataSource)
		showContextsInTable(table, contexts)
		currentResouce = Context[0]
		//ShowNotification("Fetched Contexts ...")
		updateMidFlexTitle(currentResouce, table.GetRowCount())
	} else if currentResouce == ConsumerGroup[0] {
		table.Clear()
		cgs := fetchConsumerGroups(dataSource)
		showConsumerGroups(table, cgs)
		currentResouce = ConsumerGroup[0]
		//ShowNotification("Fetched Consumer Groups ...")
		updateMidFlexTitle(currentResouce, table.GetRowCount())
	}
}

func CreateMainPage(dataSource api.KafkaDataSource, pages *tview.Pages, app *tview.Application, modal *tview.Modal, msgChannel chan UIEvent) *tview.Flex {

	table := tview.NewTable().SetBorders(false)
	table.SetSelectable(true, false)
	table.SetFixed(1, 1)

	SetupTableInput(table, app, pages, dataSource, msgChannel)

	searchInput := CreateSearchInput(table, dataSource, pages, app, modal)
	contextInfo = createContextInfo()
	topics := fetchTopics(dataSource)

	showTopicsInTable(table, topics)

	searchFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(searchInput, 0, 1, true)

	infoFlex := tview.NewFlex()
	infoFlex.AddItem(contextInfo, 0, 1, false)
	infoFlex.AddItem(CreateMainInputLegend(), 0, 1, false)
	topFlex := tview.NewFlex().
		AddItem(infoFlex, 0, 2, false).
		AddItem(searchFlex, 3, 1, true).SetDirection(tview.FlexRow)

	//topFlex.SetBorder(false).SetTitle("Top")

	midFlex = tview.NewFlex().
		AddItem(table, 0, 3, true)
	midFlex.SetBorder(true)

	updateMidFlexTitle(currentResouce, table.GetRowCount())

	notificationTextView = createNotificationTextView()
	timerView := tview.NewTextView().SetText("00:00")
	timerView.SetBorder(false)

	bottomFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(notificationTextView, 0, 3, false).
		AddItem(timerView, 0, 1, false)

	centralFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topFlex, 8, 1, false).
		AddItem(midFlex, 0, 3, true).
		AddItem(bottomFlex, 5, 1, false)

	flex := tview.NewFlex().
		AddItem(centralFlex, 0, 2, true)

	go receivingMessage(app, table, searchInput, msgChannel)

	go updateTableRoutine(app, table, timerView, dataSource)

	ShowNotification("Fetched topics...")

	currentContextName = dataSource.GetContext()
	go func() {
		app.QueueUpdateDraw(func() {
			contextInfo.SetText(currentContextName)
		})
	}()

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
		SetFieldWidth(0).
		SetText("n/a")

	inputField.SetDisabled(true)
	return inputField
}

func switchToTopicTable(table *tview.Table, dataSource api.KafkaDataSource, app *tview.Application) {
	table.Clear()
	topics := fetchTopics(dataSource)
	showTopicsInTable(table, topics)
	currentResouce = Topic[0]
	ShowNotification("Fetched Topics ...")
	updateMidFlexTitle(currentResouce, table.GetRowCount())
	app.SetFocus(table)
}

func updateMidFlexTitle(currentResouce string, amount int) {
	midFlex.SetTitle(fmt.Sprintf("<%s (%d)>", currentResouce, amount-1))
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
	contexts = sortorder.Natural(contexts)
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

func fetchTopics(dataSource api.KafkaDataSource) map[string]api.Topic {
	topics, err := dataSource.GetTopics()
	if err != nil {
		ShowNotification(fmt.Sprintf("Error reading topics:", err))
		return make(map[string]api.Topic)
	}
	lastFetchedTopics = topics
	return topics
}

func showTopicsInTable(table *tview.Table, topics map[string]api.Topic) {
	table.Clear()
	table.SetCell(0, 0, tview.NewTableCell("Topic").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetCell(0, 1, tview.NewTableCell("Num Messages").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetCell(0, 2, tview.NewTableCell("Num Partitions").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetCell(0, 3, tview.NewTableCell("Replication Factor").SetTextColor(tview.Styles.SecondaryTextColor))

	keys := make([]string, 0, len(topics))
	for key := range topics {
		if currentSearchString == "" || strings.Contains(key, currentSearchString) {
			keys = append(keys, key)
		}
	}

	sort.Sort(natural.StringSlice(keys))

	for i, key := range keys {
		value := topics[key]

		cell := tview.NewTableCell(key)
		cell.SetExpansion(1)
		table.SetCell(i+1, 0, cell)
		table.SetCell(i+1, 1, tview.NewTableCell(fmt.Sprint(value.MessageCount)))
		table.SetCell(i+1, 2, tview.NewTableCell(fmt.Sprint(value.NumPartitions)))
		table.SetCell(i+1, 3, tview.NewTableCell(fmt.Sprint(value.ReplicationFactor)))

	}
	table.SetTitle(currentResouce)
}

func CreateMainInputLegend() *tview.Flex {
	flex := tview.NewFlex()
	flex.SetBorderPadding(0, 0, 1, 0)
	left := tview.NewFlex().SetDirection(tview.FlexRow)
	right := tview.NewFlex().SetDirection(tview.FlexRow)
	right.SetBorderPadding(0, 1, 0, 0)

	left.AddItem(CreateRunInfo("↑", "Move up"), 0, 1, false)
	left.AddItem(CreateRunInfo("↓", "Move down"), 0, 1, false)
	left.AddItem(CreateRunInfo(":", "Search resource"), 0, 1, false)
	left.AddItem(CreateRunInfo("/", "Search in table"), 0, 1, false)
	right.AddItem(CreateRunInfo("g", "Scroll to top"), 0, 1, false)
	right.AddItem(CreateRunInfo("G", "Scroll to bottom"), 0, 1, false)
	right.AddItem(CreateRunInfo("c", "Copy current line"), 0, 1, false)
	right.AddItem(CreateRunInfo("Enter", "Show details"), 0, 1, false)

	flex.AddItem(left, 0, 1, false)
	flex.AddItem(right, 0, 1, false)

	return flex
}
