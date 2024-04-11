package kafui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"

	"github.com/maruel/natural"
	"github.com/rivo/tview"
)

const refreshInterval = 5000 * time.Millisecond
const refreshIntervalTable = 500 * time.Millisecond

type MainPage struct {
	CurrentContextName   string
	NotificationTextView *tview.TextView
	MidFlex              *tview.Flex
	ContextInfo          *tview.InputField
	CurrentSearchString  string
	CurrentResource      *Resource
	SearchBar            SearchBar
	cancelFetch          func()
}

func NewMainPage() *MainPage {
	return &MainPage{
		CurrentContextName: "",
	}
}

func (m *MainPage) CurrentTimeString() string {
	t := time.Now()
	return fmt.Sprintf(t.Format("Current time is 15:04"))
}

func (m *MainPage) UpdateTableRoutine(app *tview.Application, table *tview.Table, timerView *tview.TextView, dataSource api.KafkaDataSource) {
	defer RecoverAndExit(app)
	for {

		app.QueueUpdateDraw(func() {

			timerView.SetText(m.CurrentTimeString())
			m.UpdateTable(table, dataSource)

		})

		time.Sleep(refreshIntervalTable)
	}
}

func (m *MainPage) UpdateTable(table *tview.Table, dataSource api.KafkaDataSource) {
	resource := *m.CurrentResource
	switch r := (resource).(type) {
	case ResouceTopic:
		m.ShowTopicsInTable(table, r.LastFetchedTopics)
		//m.ShowNotification("Fetched Topics ...")
		m.UpdateMidFlexTitle(m.SearchBar.CurrentResource.GetName(), table.GetRowCount())
	case ResourceContext:
		m.ShowContextsInTable(table, r.FetchedContexts)
		//m.ShowNotification("Fetched Contexts ...")
		m.UpdateMidFlexTitle(m.SearchBar.CurrentResource.GetName(), table.GetRowCount())
	case ResourceGroup:
		m.ShowConsumerGroups(table, r.FetchedConsumerGroups)
		//m.ShowNotification("Fetched Consumer Groups ...")
		m.UpdateMidFlexTitle(m.SearchBar.CurrentResource.GetName(), table.GetRowCount())
	default:
		//TODO: handle
	}

}

func (m *MainPage) CreateMainPage(dataSource api.KafkaDataSource, pages *tview.Pages, app *tview.Application, modal *tview.Modal, msgChannel chan UIEvent) *tview.Flex {

	timerView := tview.NewTextView().SetText("00:00")
	timerView.SetBorder(false)

	table := tview.NewTable().SetBorders(false)
	table.SetSelectable(true, false)
	table.SetFixed(1, 1)

	m.SetupTableInput(table, app, pages, dataSource, msgChannel)

	errorFunc := func(err error) {
		m.ShowNotification("Error: " + err.Error())
	}
	onSearchBarEnterFunc := func(newResouce Resource, searchText string) {
		m.CurrentSearchString = searchText
		(*m.CurrentResource).StopFetching()
		m.CurrentResource = &newResouce
		//startUpdatingData(m, app, dataSource)
		(*m.CurrentResource).StartFetchingData()
		m.UpdateTable(table, dataSource)
	}

	m.SearchBar = *NewSearchBar(table, dataSource, pages, app, modal, onSearchBarEnterFunc, errorFunc)
	searchInput := m.SearchBar.CreateSearchInput(msgChannel)
	m.ContextInfo = createContextInfo()
	//topics := m.FetchTopics(dataSource)

	//m.ShowTopicsInTable(table, topics)

	searchFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(searchInput, 0, 1, true)

	infoFlex := tview.NewFlex()
	infoFlex.AddItem(m.ContextInfo, 0, 1, false)
	infoFlex.AddItem(CreateMainInputLegend(), 0, 1, false)
	topFlex := tview.NewFlex().
		AddItem(infoFlex, 0, 2, false).
		AddItem(searchFlex, 3, 1, true).SetDirection(tview.FlexRow)

	//topFlex.SetBorder(false).SetTitle("Top")

	m.MidFlex = tview.NewFlex().
		AddItem(table, 0, 3, true)
	m.MidFlex.SetBorder(true)

	m.UpdateMidFlexTitle(m.SearchBar.CurrentResource.GetName(), table.GetRowCount())

	m.NotificationTextView = createNotificationTextView()

	bottomFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(m.NotificationTextView, 0, 3, false).
		AddItem(timerView, 0, 1, false)

	centralFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topFlex, 8, 1, false).
		AddItem(m.MidFlex, 0, 3, true).
		AddItem(bottomFlex, 5, 1, false)

	flex := tview.NewFlex().
		AddItem(centralFlex, 0, 2, true)

	go m.UpdateTableRoutine(app, table, timerView, dataSource)
	// Create a context with cancellation
	//startUpdatingData(m, app, dataSource)
	var tr Resource
	tr = NewResouceTopic(dataSource, errorFunc, func() { RecoverAndExit(app) })
	m.CurrentResource = &tr
	(*m.CurrentResource).StartFetchingData()

	m.CurrentContextName = dataSource.GetContext()
	go func() {
		defer RecoverAndExit(app)
		app.QueueUpdateDraw(func() {
			m.ContextInfo.SetText(m.CurrentContextName)
		})
	}()

	return flex
}

/*
func startUpdatingData(m *MainPage, app *tview.Application, dataSource api.KafkaDataSource) {
	m.FetchedConsumerGroups = make(map[string]api.ConsumerGroup)
	m.FetchedContexts = make(map[string]string)
	m.LastFetchedTopics = make(map[string]api.Topic)

	ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()
	m.cancelFetch = cancel
	go m.UpdateTableDataRoutine(ctx, app, dataSource)

}*/

func (m *MainPage) ShowNotification(message string) {
	go func() {
		defer RecoverAndExit(tviewApp)
		tviewApp.QueueUpdateDraw(func() {
			m.NotificationTextView.SetText(message)
		})
		// Schedule hiding TextView after 2 seconds

		time.Sleep(2 * time.Second)
		tviewApp.QueueUpdateDraw(func() {
			m.NotificationTextView.SetText("")
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

func (m *MainPage) switchToTopicTable(table *tview.Table, dataSource api.KafkaDataSource, app *tview.Application) {
	//table.Clear()
	//topics := m.FetchTopics(dataSource)
	//m.ShowTopicsInTable(table, topics)
	m.SearchBar.CurrentResource = NewResouceTopic(dataSource, m.SearchBar.onError, func() { RecoverAndExit(app) })
	m.ShowNotification("Fetched Topics ...")
	m.UpdateMidFlexTitle(m.SearchBar.CurrentResource.GetName(), table.GetRowCount())
	app.SetFocus(table)
}

func (m *MainPage) UpdateMidFlexTitle(currentResouce string, amount int) {
	m.MidFlex.SetTitle(fmt.Sprintf("<%s (%d)>", currentResouce, amount-1))
}

func (m *MainPage) ShowConsumerGroups(table *tview.Table, cgs map[string]api.ConsumerGroup) {
	table.Clear()
	// Define table headers
	table.SetCell(0, 0, tview.NewTableCell("Name").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetCell(0, 1, tview.NewTableCell("State").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetCell(0, 2, tview.NewTableCell("Consumers").SetTextColor(tview.Styles.SecondaryTextColor).SetExpansion(1))

	keys := make([]string, 0, len(cgs))
	for key := range cgs {
		if m.CurrentSearchString == "" || strings.Contains(strings.ToLower(key), strings.ToLower(m.CurrentSearchString)) {
			keys = append(keys, key)
		}
	}

	sort.Sort(natural.StringSlice(keys))

	for i, key := range keys {
		cg := cgs[key]
		// Add data to the table
		cell := tview.NewTableCell(cg.Name)
		table.SetCell(i+1, 0, cell)
		table.SetCell(i+1, 1, tview.NewTableCell(cg.State))
		table.SetCell(i+1, 2, tview.NewTableCell(strconv.Itoa(cg.Consumers)).SetExpansion(1))
	}
}

func (m *MainPage) FetchConsumerGroups(dataSource api.KafkaDataSource) []api.ConsumerGroup {
	cgs, err := dataSource.GetConsumerGroups()
	if err != nil {
		m.ShowNotification(fmt.Sprintf("Error fetching GetConsumerGroups:", err))
		return []api.ConsumerGroup{}
	}
	return cgs
}

func (m *MainPage) ShowContextsInTable(table *tview.Table, contexts map[string]string) {
	table.Clear()

	table.SetCell(0, 0, tview.NewTableCell("Context").SetTextColor(tview.Styles.SecondaryTextColor).SetExpansion(1))

	keys := make([]string, 0, len(contexts))
	for key := range contexts {
		if m.CurrentSearchString == "" || strings.Contains(strings.ToLower(key), strings.ToLower(m.CurrentSearchString)) {
			keys = append(keys, key)
		}
	}

	sort.Sort(natural.StringSlice(keys))

	for i, key := range keys {
		context := contexts[key]
		cell := tview.NewTableCell(context)
		cell.SetExpansion(1)
		table.SetCell(i+1, 0, cell)
	}
	table.SetTitle(m.SearchBar.CurrentResource.GetName())
}

func (m *MainPage) FetchContexts(dataSource api.KafkaDataSource) []string {
	contexts, err := dataSource.GetContexts()
	if err != nil {
		m.ShowNotification(fmt.Sprintf("Error fetching contexts:", err))
		return []string{}
	}
	return contexts
}

func (m *MainPage) ShowTopicsInTable(table *tview.Table, topics map[string]api.Topic) {
	table.Clear()
	table.SetCell(0, 0, tview.NewTableCell("Topic").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetCell(0, 1, tview.NewTableCell("Num Messages").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetCell(0, 2, tview.NewTableCell("Num Partitions").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetCell(0, 3, tview.NewTableCell("Replication Factor").SetTextColor(tview.Styles.SecondaryTextColor).SetExpansion(1))

	keys := make([]string, 0, len(topics))
	for key := range topics {
		if m.CurrentSearchString == "" || strings.Contains(strings.ToLower(key), strings.ToLower(m.CurrentSearchString)) {
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
	table.SetTitle(m.SearchBar.CurrentResource.GetName())
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
