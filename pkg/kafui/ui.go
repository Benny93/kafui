package kafui

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/api"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var currentTopic api.Topic
var tviewApp *tview.Application
var topicPage *TopicPage

func OpenUI(dataSource api.KafkaDataSource) {
	tview.Styles = tview.Theme{
		PrimitiveBackgroundColor:    tcell.ColorBlack.TrueColor(),
		ContrastBackgroundColor:     tcell.ColorBlack.TrueColor(),
		MoreContrastBackgroundColor: tcell.ColorGreen.TrueColor(),
		BorderColor:                 tcell.ColorWhite.TrueColor(),
		TitleColor:                  tcell.ColorWhite.TrueColor(),
		GraphicsColor:               tcell.ColorBlack.TrueColor(),
		PrimaryTextColor:            tcell.ColorDarkCyan.TrueColor(),
		SecondaryTextColor:          tcell.ColorWhite.TrueColor(),
		TertiaryTextColor:           tcell.ColorGreen.TrueColor(),
		InverseTextColor:            tcell.ColorGreen.TrueColor(),
		ContrastSecondaryTextColor:  tcell.ColorWhite.TrueColor(),
	}

	// Create the application
	tviewApp = tview.NewApplication()

	pages := tview.NewPages()
	modal := tview.NewModal().
		SetText("Resource Not Found").
		AddButtons([]string{"OK"})

	// channel to publish messages to
	msgChannel := make(chan UIEvent)

	mainPage := NewMainPage()
	flex := mainPage.CreateMainPage(dataSource, pages, tviewApp, modal, msgChannel)

	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		pages.HidePage("modal")
		msgChannel <- OnModalClose
	})

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Hide the modal when any key is pressed
		pages.HidePage("modal")
		return event // Return the event to continue processing other key events
	})
	// input

	// Set the input capture to capture key events
	tviewApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		frontPage, _ := pages.GetFrontPage()
		// Check if the pressed key is Shift + :
		if event.Rune() == ':' {
			// Handle the Shift + : key combination
			msgChannel <- OnFocusSearch
			return nil // Return nil to indicate that the event has been handled
		}

		if event.Rune() == '/' && frontPage == "main" {
			// Handle the Shift + : key combination
			msgChannel <- OnStartTableSearch
			return nil // Return nil to indicate that the event has been handled
		}

		if event.Key() == tcell.KeyEsc {

			if frontPage == "topicPage" {
				topicPage.CloseTopicPage()
			}
			if frontPage == "DetailPage" {
				topicPage.messageDetailPage.Hide()
				//pages.SwitchToPage("topicPage")
				return event
			}

			if frontPage != "main" {
				pages.SwitchToPage("main")
			}
		}

		// Return the event to continue processing other key events
		return event
	})

	topicPage = NewTopicPage(dataSource, pages, tviewApp, msgChannel)
	topicPageFlex := topicPage.CreateTopicPage("Current Topic")

	pages.
		AddPage("main", flex, true, true).
		AddPage("modal", modal, true, false).
		AddPage("topicPage", topicPageFlex, true, false)

	pages.SetChangedFunc(func() {
		msgChannel <- OnPageChange
	})

	// Recover from panics and handle gracefully
	defer RecoverAndExit(tviewApp)

	if err := tviewApp.SetRoot(pages, true).EnableMouse(false).Run(); err != nil {
		fmt.Println("Run ended in panic")
		panic(err)
	}
}

func CreatePropertyInfo(propertyName string, propertyValue string) *tview.InputField {
	inputField := tview.NewInputField().
		SetLabel(fmt.Sprintf("%s: ", propertyName)).
		SetFieldWidth(0).
		SetText(propertyValue)
	inputField.SetDisabled(true)
	return inputField
}

func CreateRunInfo(runeName string, info string) *tview.InputField {
	inputField := tview.NewInputField().
		SetLabel(fmt.Sprintf("<%s>: ", runeName)).
		SetFieldWidth(0).
		SetText(info)
	inputField.SetDisabled(true)
	inputField.SetLabelColor(tcell.ColorBlue)
	inputField.SetFieldTextColor(tcell.ColorBlue)
	return inputField
}
