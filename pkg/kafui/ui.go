package kafui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func OpenUI(dataSource KafkaDataSource) {
	// Create the application
	app := tview.NewApplication()

	searchInput := tview.NewInputField().
		SetLabel("Search: ").
		SetFieldWidth(20)

	searchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			searchText := searchInput.GetText()
			// Handle search logic here
			fmt.Println("Search text:", searchText)
		}
	})

	/*contexts, err := dataSource.GetContexts()
	if err != nil {
		fmt.Println("Error fetching contexts:", err)
		return
	}*/

	topics, err := dataSource.GetTopics()
	if err != nil {
		fmt.Println("Error reading topics")
		return
	}

	table := tview.NewTable().SetBorders(false)
	// Add header row
	table.SetCell(0, 0, tview.NewTableCell("Topics").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetSelectable(true, false)
	table.SetFixed(1, 1)

	// Add data rows
	for i, topic := range topics {
		cell := tview.NewTableCell(topic)
		cell.SetExpansion(1)
		table.SetCell(i+1, 0, cell)
	}

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

	if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
