package kafui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func OpenUI(dataSource KafkaDataSource) {
	// Create the application
	app := tview.NewApplication()
	pages := tview.NewPages()
	modal := tview.NewModal().
		SetText("Resource Not Found").
		AddButtons([]string{"OK"})

	// channel to publish messages to
	msgChannel := make(chan string)

	// Fetch context data from KafkaDataSource
	// show dialog that the requested resource could not be found
	_, _, flex := CreateMainPage(dataSource, pages, app, modal, msgChannel)

	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		pages.HidePage("modal")
		msgChannel <- "ModalClose"
	})

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Hide the modal when any key is pressed
		pages.HidePage("modal")
		return event // Return the event to continue processing other key events
	})
	// input

	// Set the input capture to capture key events
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Check if the pressed key is Shift + :
		if event.Key() == tcell.KeyRune && event.Modifiers() == tcell.ModShift && event.Rune() == ':' {
			// Handle the Shift + : key combination
			msgChannel <- "FocusSearch"
			return nil // Return nil to indicate that the event has been handled
		}

		// Return the event to continue processing other key events
		return event
	})

	pages.
		AddPage("main", flex, true, true).
		AddPage("modal", modal, true, false)

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
