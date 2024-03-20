package kafui

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/api"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var currentTopic string = ""
var tviewApp *tview.Application

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

	// Fetch context data from api.KafkaDataSource
	// show dialog that the requested resource could not be found
	flex := CreateMainPage(dataSource, pages, tviewApp, modal, msgChannel)

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
		// Check if the pressed key is Shift + :
		if event.Rune() == ':' {
			// Handle the Shift + : key combination
			msgChannel <- OnFocusSearch
			return nil // Return nil to indicate that the event has been handled
		}

		if event.Rune() == '/' {
			// Handle the Shift + : key combination
			msgChannel <- OnStartTableSearch
			return nil // Return nil to indicate that the event has been handled
		}

		if event.Key() == tcell.KeyEsc {
			frontPage, _ := pages.GetFrontPage()
			if frontPage != "main" {
				pages.SwitchToPage("main")
			}
		}

		// Return the event to continue processing other key events
		return event
	})

	topicPage := CreateTopicPage(dataSource, pages, tviewApp, msgChannel)

	pages.
		AddPage("main", flex, true, true).
		AddPage("modal", modal, true, false).
		AddPage("topicPage", topicPage, true, false)

	pages.SetChangedFunc(func() {
		msgChannel <- OnPageChange
	})

	// Recover from panics and handle gracefully
	defer func() {
		if r := recover(); r != nil {
			tviewApp.Stop()
			fmt.Println("An error occurred:", r)
			fmt.Println("Application stopped.")
		}
	}()

	if err := tviewApp.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		fmt.Println("Run ended in panic")
		panic(err)
	}
}
