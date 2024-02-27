package kafui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func OpenUI(dataSource KafkaDataSource) {
	// Create the application
	app := tview.NewApplication()
	table := tview.NewTable().SetBorders(false)
	table.SetSelectable(true, false)
	table.SetFixed(1, 1)

	searchInput := tview.NewInputField().
		SetLabel("Search: ").
		SetFieldWidth(20)
	searchText := ""

	searchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			searchText = searchInput.GetText()

			if searchText == "context" {
				table.Clear()
				// Fetch context data from KafkaDataSource
				contexts := fetchContexts(dataSource)
				showContextsInTable(table, contexts)
			}

			if searchText == "topics" {
				table.Clear()
				topics := fetchTopics(dataSource)
				showTopicsInTable(table, topics)
			}
			searchInput.SetText("")

			app.SetFocus(table)

		}
	})

	/*contexts, err := dataSource.GetContexts()
	if err != nil {
		fmt.Println("Error fetching contexts:", err)
		return
	}*/

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

	// input

	// Set the input capture to capture key events
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Check if the pressed key is Shift + :
		if event.Key() == tcell.KeyRune && event.Modifiers() == tcell.ModShift && event.Rune() == ':' {
			// Handle the Shift + : key combination
			app.SetFocus(searchInput)
			return nil // Return nil to indicate that the event has been handled
		}
		// Return the event to continue processing other key events
		return event
	})

	if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func showContextsInTable(table *tview.Table, contexts []string) {
	table.SetCell(0, 0, tview.NewTableCell("Context").SetTextColor(tview.Styles.SecondaryTextColor))
	for i, context := range contexts {
		table.SetCell(i+1, 0, tview.NewTableCell(context))
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
