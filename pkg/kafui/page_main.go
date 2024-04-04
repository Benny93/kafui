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

type MainPage struct {
	CurrentResource      string
	CurrentContextName   string
	NotificationTextView *tview.TextView
	MidFlex              *tview.Flex
	ContextInfo          *tview.InputField
	CurrentSearchString  string
	LastFetchedTopics    map[string]api.Topic
	SearchBar            SearchBar
}

func NewMainPage() *MainPage {
	return &MainPage{
		CurrentResource:    Topic[0],
		CurrentContextName: "",
		LastFetchedTopics:  make(map[string]api.Topic),
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
		time.Sleep(refreshInterval)
	}
}

func (m *MainPage) UpdateTable(table *tview.Table, dataSource api.KafkaDataSource) {
	//m.ShowNotification("Update Table " + m.CurrentSearchString)
	if m.CurrentResource == Topic[0] {
		table.Clear()
		topics := m.FetchTopics(dataSource)
		m.ShowTopicsInTable(table, topics)
		m.CurrentResource = Topic[0]
		//ShowNotification("Fetched Topics ...")
		m.UpdateMidFlexTitle(m.CurrentResource, table.GetRowCount())
	} else if m.CurrentResource == Context[0] {
		table.Clear()
		contexts := m.FetchContexts(dataSource)
		m.ShowContextsInTable(table, contexts)
		m.CurrentResource = Context[0]
		//ShowNotification("Fetched Contexts ...")
		m.UpdateMidFlexTitle(m.CurrentResource, table.GetRowCount())
	} else if m.CurrentResource == ConsumerGroup[0] {
		table.Clear()
		cgs := m.FetchConsumerGroups(dataSource)
		m.ShowConsumerGroups(table, cgs)
		m.CurrentResource = ConsumerGroup[0]
		//ShowNotification("Fetched Consumer Groups ...")
		m.UpdateMidFlexTitle(m.CurrentResource, table.GetRowCount())
	}
}

func (m *MainPage) CreateMainPage(dataSource api.KafkaDataSource, pages *tview.Pages, app *tview.Application, modal *tview.Modal, msgChannel chan UIEvent) *tview.Flex {

	table := tview.NewTable().SetBorders(false)
	table.SetSelectable(true, false)
	table.SetFixed(1, 1)

	m.SetupTableInput(table, app, pages, dataSource, msgChannel)

	m.SearchBar = *NewSearchBar(table, dataSource, pages, app, modal, func(resouceName, searchText string) {
		m.CurrentResource = resouceName
		m.CurrentSearchString = searchText
		m.UpdateTable(table, dataSource)
	})
	searchInput := m.SearchBar.CreateSearchInput(msgChannel)
	m.ContextInfo = createContextInfo()
	topics := m.FetchTopics(dataSource)

	m.ShowTopicsInTable(table, topics)

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

	m.UpdateMidFlexTitle(m.CurrentResource, table.GetRowCount())

	m.NotificationTextView = createNotificationTextView()
	timerView := tview.NewTextView().SetText("00:00")
	timerView.SetBorder(false)

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

	m.ShowNotification("Fetched topics...")

	m.CurrentContextName = dataSource.GetContext()
	go func() {
		defer RecoverAndExit(app)
		app.QueueUpdateDraw(func() {
			m.ContextInfo.SetText(m.CurrentContextName)
		})
	}()

	return flex
}

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
	table.Clear()
	topics := m.FetchTopics(dataSource)
	m.ShowTopicsInTable(table, topics)
	m.CurrentResource = Topic[0]
	m.ShowNotification("Fetched Topics ...")
	m.UpdateMidFlexTitle(m.CurrentResource, table.GetRowCount())
	app.SetFocus(table)
}

func (m *MainPage) UpdateMidFlexTitle(currentResouce string, amount int) {
	m.MidFlex.SetTitle(fmt.Sprintf("<%s (%d)>", currentResouce, amount-1))
}

func (m *MainPage) ShowConsumerGroups(table *tview.Table, cgs []api.ConsumerGroup) {
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

func (m *MainPage) FetchConsumerGroups(dataSource api.KafkaDataSource) []api.ConsumerGroup {
	cgs, err := dataSource.GetConsumerGroups()
	if err != nil {
		m.ShowNotification(fmt.Sprintf("Error fetching GetConsumerGroups:", err))
		return []api.ConsumerGroup{}
	}
	return cgs
}

func (m *MainPage) ShowContextsInTable(table *tview.Table, contexts []string) {
	contexts = sortorder.Natural(contexts)
	table.SetCell(0, 0, tview.NewTableCell("Context").SetTextColor(tview.Styles.SecondaryTextColor))
	for i, context := range contexts {
		cell := tview.NewTableCell(context)
		cell.SetExpansion(1)
		table.SetCell(i+1, 0, cell)
	}
	table.SetTitle(m.CurrentResource)
}

func (m *MainPage) FetchContexts(dataSource api.KafkaDataSource) []string {
	contexts, err := dataSource.GetContexts()
	if err != nil {
		m.ShowNotification(fmt.Sprintf("Error fetching contexts:", err))
		return []string{}
	}
	return contexts
}

func (m *MainPage) FetchTopics(dataSource api.KafkaDataSource) map[string]api.Topic {
	topics, err := dataSource.GetTopics()
	if err != nil {
		m.ShowNotification(fmt.Sprintf("Error reading topics:", err))
		return make(map[string]api.Topic)
	}
	m.LastFetchedTopics = topics
	return topics
}

func (m *MainPage) ShowTopicsInTable(table *tview.Table, topics map[string]api.Topic) {
	table.Clear()
	table.SetCell(0, 0, tview.NewTableCell("Topic").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetCell(0, 1, tview.NewTableCell("Num Messages").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetCell(0, 2, tview.NewTableCell("Num Partitions").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetCell(0, 3, tview.NewTableCell("Replication Factor").SetTextColor(tview.Styles.SecondaryTextColor))

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
	table.SetTitle(m.CurrentResource)
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
