package kafui

import (
	"encoding/json"
	"time"

	"github.com/TylerBrock/colorjson"
	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type DetailPage struct {
	app           *tview.Application
	pages         *tview.Pages
	value         string
	valueTextView *tview.TextView
}

func NewDetailPage(app *tview.Application, pages *tview.Pages, value string) *DetailPage {

	// Format and colorize JSON
	var obj map[string]interface{}
	json.Unmarshal([]byte(value), &obj)
	f := colorjson.NewFormatter()
	f.Indent = 2
	s, _ := f.Marshal(obj)

	valueTextView := tview.NewTextView().
		//SetText("Placeholder :)").
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true).
		SetWordWrap(false)
	valueTextView.SetTextColor(tcell.ColorWhite)
	valueTextView.SetBorder(true)

	writer := tview.ANSIWriter(valueTextView)
	writer.Write(s)

	return &DetailPage{
		app:           app,
		pages:         pages,
		value:         value,
		valueTextView: valueTextView,
	}
}

func (vp *DetailPage) Show() {
	// Create a new flex layout for the value page
	valueFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText("Message Value").SetTextAlign(tview.AlignCenter), 1, 0, false).
		AddItem(tview.NewTextView(), 2, 1, false)

	// Add the TextView to the flex layout
	valueFlex.AddItem(vp.valueTextView, 0, 1, true)

	// Add the value page to the pages container
	vp.pages.AddPage("DetailPage", valueFlex, true, true)

	vp.valueTextView.SetInputCapture(vp.handleInput)

}

// Hide hides the page.
func (vp *DetailPage) Hide() {
	vp.pages.RemovePage("DetailPage")
}

func (vp *DetailPage) handleInput(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyRune && event.Rune() == 'c' && vp.valueTextView.HasFocus() {
		// Copy the content of valueTextView to the clipboard

		clipboard.WriteAll(vp.valueTextView.GetText(true))
		// Show a notification that the content has been copied
		vp.showCopiedNotification()
		return nil
	}
	return event
}

// TODO: HandleKey handles key events for the value page.
//
//	func (vp *DetailPage) HandleKey(event *tcell.EventKey) *tcell.EventKey {
//		if event.Key() == tcell.KeyEsc {
//			// Switch back to the original page when Escape is pressed
//			//vp.pages.SwitchToPage("TopicPage")
//			vp.Hide()
//			return nil
//		}
//		return event
//	}
func (vp *DetailPage) showCopiedNotification() {
	go func() {
		_, page := vp.pages.GetFrontPage()
		item := tview.NewTextView().SetText("😎 Content copied to clipboard ...").SetTextAlign(tview.AlignCenter)
		vp.app.QueueUpdateDraw(func() {

			page.(*tview.Flex).AddItem(item, 1, 0, false)
		})
		// Hide the notification after 2 seconds
		time.Sleep(2 * time.Second)
		vp.app.QueueUpdateDraw(func() {
			page.(*tview.Flex).RemoveItem(item)
		})
	}()
}
