package kafui

import (
	"strings"

	"github.com/Benny93/kafui/pkg/api"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func CreateSearchInput(table *tview.Table, dataSource api.KafkaDataSource, pages *tview.Pages, app *tview.Application, modal *tview.Modal) *tview.InputField {
	defaultLabel := "😎|"
	searchInput := tview.NewInputField().
		SetLabel(defaultLabel).
		SetFieldWidth(0)
	searchInput.SetBorder(true).SetBorderColor(tcell.ColorDarkCyan.TrueColor())
	//searchInput.SetFieldBackgroundColor(tcell.ColorBlack)
	selectedStyle := tcell.Style{}
	selectedStyle.Background(tcell.ColorWhite)
	searchInput.SetAutocompleteStyles(tcell.ColorBlue, tcell.Style{}, selectedStyle)
	searchText := ""

	searchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			searchText = searchInput.GetText()
			if currentSearchMode == ResouceSearch {
				handleResouceSearch(searchText, table, searchInput, defaultLabel, dataSource, app, pages, modal)
			} else {
				handleTableSearch(searchText, table)
			}

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

func handleTableSearch(searchText string, table *tview.Table) {
	// filter table by given searchText
	// Store all rows
	var visibleRows [][]*tview.TableCell
	for row := 0; row < table.GetRowCount(); row++ {
		var visibleCells []*tview.TableCell
		for column := 0; column < table.GetColumnCount(); column++ {
			cell := table.GetCell(row, column)
			if cell == nil {
				continue
			}
			cellText := cell.Text
			// Check if the cell content contains the search text
			if strings.Contains(strings.ToLower(cellText), strings.ToLower(searchText)) {
				visibleCells = append(visibleCells, cell)
			}
		}
		// If any cell in the row matches the search text, add the row to visibleRows
		if len(visibleCells) > 0 {
			visibleRows = append(visibleRows, visibleCells)
		}
	}

	// Clear the table
	table.Clear()

	// Add filtered rows to the table
	for _, row := range visibleRows {
		for _, cell := range row {
			table.SetCell(table.GetRowCount(), table.GetColumnCount(), cell)
		}
	}
}

func handleResouceSearch(searchText string, table *tview.Table, searchInput *tview.InputField, defaultLabel string, dataSource api.KafkaDataSource, app *tview.Application, pages *tview.Pages, modal *tview.Modal) {
	match := false
	if Contains(Context, searchText) {
		match = true
		currentResouce = Context[0]
	}

	if Contains(Topic, searchText) {
		currentResouce = Topic[0]
		match = true
	}

	if Contains(ConsumerGroup, searchText) {
		currentResouce = ConsumerGroup[0]
		match = true
	}
	if !match {
		pages.ShowPage("modal")
		app.SetFocus(modal)
	} else {
		UpdateTable(table, dataSource)
		app.SetFocus(table)
	}
	searchInput.SetLabel(defaultLabel)
	searchInput.SetText("")
}
