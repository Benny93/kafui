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
				handleTableSearch(searchText, app, table, dataSource)
			}

		}
	})

	searchInput.SetChangedFunc(func(text string) {
		if currentSearchMode == TableSearch {
			currentSearchString = text
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

func handleTableSearch(searchText string, app *tview.Application, table *tview.Table, dataSource api.KafkaDataSource) {
	// filter table by given searchText
	currentSearchString = searchText
	UpdateTable(table, dataSource)
	app.SetFocus(table)
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
