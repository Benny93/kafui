package kafui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func receivingMessage(app *tview.Application, table *tview.Table, searchInput *tview.InputField, msgChannel chan string) {
	for {
		msg := <-msgChannel
		if msg == "ModalClose" {
			app.SetFocus(table)
		}
		if msg == "FocusSearch" {
			searchInput.SetLabel("🧐>")
			app.SetFocus(searchInput)
		}
	}
}

func CreateMainPage(dataSource KafkaDataSource, pages *tview.Pages, app *tview.Application, modal *tview.Modal, msgChannel chan string) (*tview.Table, *tview.InputField, *tview.Flex) {

	table := tview.NewTable().SetBorders(false)
	table.SetSelectable(true, false)
	table.SetFixed(1, 1)
	defaultLabel := "😎|"
	searchInput := tview.NewInputField().
		SetLabel(defaultLabel).
		SetFieldWidth(20)
	searchText := ""

	searchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			searchText = searchInput.GetText()
			match := false
			if searchText == "context" {
				table.Clear()
				searchInput.SetLabel(defaultLabel)
				contexts := fetchContexts(dataSource)
				showContextsInTable(table, contexts)
				match = true
			}

			if searchText == "topics" {
				table.Clear()
				topics := fetchTopics(dataSource)
				showTopicsInTable(table, topics)
				match = true
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

	return table, searchInput, flex
}

func showContextsInTable(table *tview.Table, contexts []string) {
	table.SetCell(0, 0, tview.NewTableCell("Context").SetTextColor(tview.Styles.SecondaryTextColor))
	for i, context := range contexts {
		cell := tview.NewTableCell(context)
		cell.SetExpansion(1)
		table.SetCell(i+1, 0, cell)
	}
}

func fetchContexts(dataSource KafkaDataSource) []string {
	contexts, err := dataSource.GetContexts()
	if err != nil {
		fmt.Println("Error fetching contexts:", err)
	}
	return contexts
}

func fetchTopics(dataSource KafkaDataSource) []string {
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
