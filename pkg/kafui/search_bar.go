package kafui

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/api"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SearchBar struct {
	Table           *tview.Table
	DataSource      api.KafkaDataSource
	Pages           *tview.Pages
	App             *tview.Application
	Modal           *tview.Modal
	DefaultLabel    string
	SearchInput     *tview.InputField
	CurrentMode     SearchMode
	CurrentString   string
	CurrentResource Resource
	UpdateTable     func(newResource Resource, searchText string)
	onError         func(err error)
}

func NewSearchBar(table *tview.Table, dataSource api.KafkaDataSource, pages *tview.Pages, app *tview.Application, modal *tview.Modal, updateTable func(newResource Resource, searchText string), onError func(err error)) *SearchBar {
	return &SearchBar{
		Table:           table,
		DataSource:      dataSource,
		Pages:           pages,
		App:             app,
		Modal:           modal,
		DefaultLabel:    "😎|",
		SearchInput:     nil,
		CurrentMode:     ResouceSearch,
		CurrentString:   "",
		CurrentResource: NewResouceTopic(dataSource, onError, func() { RecoverAndExit(app) }),
		UpdateTable:     updateTable,
		onError:         onError,
	}
}

func (s *SearchBar) CreateSearchInput(msgChannel chan UIEvent) *tview.InputField {
	s.SearchInput = tview.NewInputField().
		SetLabel(s.DefaultLabel).
		SetFieldWidth(0)
	s.SearchInput.SetBorder(true).SetBorderColor(tcell.ColorDarkCyan.TrueColor())
	//searchInput.SetFieldBackgroundColor(tcell.ColorBlack)
	selectedStyle := tcell.Style{}
	selectedStyle.Background(tcell.ColorWhite)
	s.SearchInput.SetAutocompleteStyles(tcell.ColorBlue, tcell.Style{}, selectedStyle)
	searchText := ""

	s.SearchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			searchText = s.SearchInput.GetText()
			// Check if search text is "q" or "exit"
			if searchText == "q" || searchText == "exit" {
				s.App.Stop()
				fmt.Println("Goodbye!")
				return
			}
			if s.CurrentMode == ResouceSearch {
				s.handleResouceSearch(searchText)
			} else {
				s.handleTableSearch(searchText)
			}

		}
	})

	s.SearchInput.SetChangedFunc(func(text string) {
		if s.CurrentMode == TableSearch {
			s.CurrentString = text
			s.UpdateTable(s.CurrentResource, s.CurrentString)
		}
	})

	s.SearchInput.SetAutocompleteFunc(func(currentText string) (entries []string) {
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

	go s.ReceivingMessage(s.App, s.Table, s.SearchInput, msgChannel)

	return s.SearchInput
}

func (s *SearchBar) handleTableSearch(searchText string) {
	// filter table by given searchText
	s.CurrentString = searchText
	s.UpdateTable(s.CurrentResource, s.CurrentString)
	s.App.SetFocus(s.Table)
}

func (s *SearchBar) handleResouceSearch(searchText string) {
	match := false
	if Contains(Context, searchText) {
		match = true
		s.CurrentResource = NewResourceContext(s.onError)
	}

	if Contains(Topic, searchText) {
		s.CurrentResource = NewResouceTopic(s.DataSource, s.onError, func() { RecoverAndExit(s.App) })
		match = true
	}

	if Contains(ConsumerGroup, searchText) {
		s.CurrentResource = NewResouceConsumerGroup(s.onError)
		match = true
	}
	if !match {
		s.Pages.ShowPage("modal")
		s.App.SetFocus(s.Modal)
	} else {
		s.UpdateTable(s.CurrentResource, s.CurrentString)
		s.App.SetFocus(s.Table)
	}
	s.SearchInput.SetLabel(s.DefaultLabel)
	s.SearchInput.SetText("")
}

func (s *SearchBar) ReceivingMessage(app *tview.Application, table *tview.Table, searchInput *tview.InputField, msgChannel chan UIEvent) {
	defer RecoverAndExit(s.App)
	for {
		msg := <-msgChannel
		if msg == OnModalClose {
			app.SetFocus(table)
		}
		if msg == OnFocusSearch {
			searchInput.SetLabel("🧐>")
			searchInput.SetText("")
			app.SetFocus(searchInput)
			s.CurrentMode = ResouceSearch
			s.CurrentString = ""
		}
		if msg == OnStartTableSearch {
			searchInput.SetLabel("💡?")
			app.SetFocus(searchInput)
			s.CurrentMode = TableSearch
			s.CurrentString = ""
		}
	}
}
