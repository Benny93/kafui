package kafui

import (
	"fmt"
	"time"

	"github.com/Benny93/kafui/pkg/api"

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
	SearchBar            *SearchBar
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
	defer func() {
		if app != nil {
			RecoverAndExit(app)
		}
	}()
	for {
		if app == nil {
			return
		}

		app.QueueUpdateDraw(func() {
			if timerView != nil {
				timerView.SetText(m.CurrentTimeString())
			}
			if table != nil && dataSource != nil {
				m.UpdateTable(table, dataSource)
			}
		})

		time.Sleep(refreshIntervalTable)
	}
}

func (m *MainPage) UpdateTable(table *tview.Table, dataSource api.KafkaDataSource) {
	//m.ShowNotification("Update Table..")
	if m.CurrentResource == nil {
		return
	}
	
	resource := *m.CurrentResource
	searchString := ""
	if m.SearchBar != nil {
		searchString = m.SearchBar.CurrentString
	}
	
	resource.UpdateTable(table, dataSource, searchString)
	
	if m.SearchBar != nil && m.SearchBar.CurrentResource != nil {
		m.UpdateMidFlexTitle(m.SearchBar.CurrentResource.GetName(), table.GetRowCount())
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

	m.SearchBar = NewSearchBar(table, dataSource, pages, app, modal, onSearchBarEnterFunc, errorFunc)
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
	if m.NotificationTextView == nil {
		return
	}
	
	go func() {
		defer RecoverAndExit(tviewApp)
		tviewApp.QueueUpdateDraw(func() {
			if m.NotificationTextView != nil {
				m.NotificationTextView.SetText(message)
			}
		})
		// Schedule hiding TextView after 2 seconds

		time.Sleep(2 * time.Second)
		tviewApp.QueueUpdateDraw(func() {
			if m.NotificationTextView != nil {
				m.NotificationTextView.SetText("")
			}
		})

	}()
}

func createNotificationTextView() *tview.TextView {
	textView := tview.NewTextView().SetText("")
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

func (m *MainPage) UpdateMidFlexTitle(currentResouce string, amount int) {
	if m.MidFlex != nil {
		m.MidFlex.SetTitle(fmt.Sprintf("<%s (%d)>", currentResouce, amount-1))
	}
}

func (m *MainPage) FetchConsumerGroups(dataSource api.KafkaDataSource) []api.ConsumerGroup {
	cgs, err := dataSource.GetConsumerGroups()
	if err != nil {
		m.ShowNotification(fmt.Sprintf("Error fetching GetConsumerGroups: %v", err))
		return []api.ConsumerGroup{}
	}
	return cgs
}

func (m *MainPage) FetchContexts(dataSource api.KafkaDataSource) []string {
	contexts, err := dataSource.GetContexts()
	if err != nil {
		m.ShowNotification(fmt.Sprintf("Error fetching contexts: %v", err))
		return []string{}
	}
	return contexts
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
