package kafui

import (
	"com/emptystate/kafui/pkg/api"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var currentResouce string = Topic[0] // Topic is the default

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

	topics := fetchTopics(dataSource)

	showTopicsInTable(table, topics)

	topFlex := tview.NewFlex().
		AddItem(searchInput, 0, 1, true)

	topFlex.SetBorder(true).SetTitle("Top")

	midFlex := tview.NewFlex().
		AddItem(table, 0, 3, true)
	midFlex.SetBorder(true).SetTitle("Middle (3 x height of Top)")

	centralFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topFlex, 0, 1, false).
		AddItem(midFlex, 0, 3, true).
		AddItem(tview.NewFlex().SetBorder(true).SetTitle("Bottom (5 rows)"), 5, 1, false)

	flex := tview.NewFlex().
		AddItem(centralFlex, 0, 2, true)

	go receivingMessage(app, table, searchInput, msgChannel)

	return flex
}

func createSearchInput(defaultLabel string, table *tview.Table, dataSource api.KafkaDataSource, pages *tview.Pages, app *tview.Application, modal *tview.Modal) *tview.InputField {
	searchInput := tview.NewInputField().
		SetLabel(defaultLabel).
		SetFieldWidth(0)

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
			}

			if Contains(Topic, searchText) {
				table.Clear()
				topics := fetchTopics(dataSource)
				showTopicsInTable(table, topics)
				match = true
				currentResouce = Topic[0]
			}

			if Contains(ConsumerGroup, searchText) {
				table.Clear()
				cgs := fetchConsumerGroups(dataSource)
				showConsumerGroups(table, cgs)
				match = true
				currentResouce = ConsumerGroup[0]
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

func showConsumerGroups(table *tview.Table, cgs []string) {
	table.SetCell(0, 0, tview.NewTableCell("Context").SetTextColor(tview.Styles.SecondaryTextColor))
	for i, consumer := range cgs {
		cell := tview.NewTableCell(consumer)
		cell.SetExpansion(1)
		table.SetCell(i+1, 0, cell)
	}
}

func fetchConsumerGroups(dataSource api.KafkaDataSource) []string {
	cgs, err := dataSource.GetConsumerGroups()
	if err != nil {
		fmt.Println("Error fetching GetConsumerGroups:", err)
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
}

func fetchContexts(dataSource api.KafkaDataSource) []string {
	contexts, err := dataSource.GetContexts()
	if err != nil {
		fmt.Println("Error fetching contexts:", err)
	}
	return contexts
}

func fetchTopics(dataSource api.KafkaDataSource) []string {
	topics, err := dataSource.GetTopics()
	if err != nil {
		fmt.Println("Error reading topics")
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
}
