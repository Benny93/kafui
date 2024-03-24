package kafui

import (
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
	valueTextView := tview.NewTextView().
		SetText(value).
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true).
		SetWordWrap(true)
	valueTextView.SetTextColor(tcell.ColorWhite)
	valueTextView.SetBorder(true)

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
}

// Hide hides the page.
func (vp *DetailPage) Hide() {
	vp.pages.RemovePage("DetailPage")
}

// TODO: HandleKey handles key events for the value page.
// func (vp *DetailPage) HandleKey(event *tcell.EventKey) *tcell.EventKey {
// 	if event.Key() == tcell.KeyEsc {
// 		// Switch back to the original page when Escape is pressed
// 		//vp.pages.SwitchToPage("TopicPage")
// 		vp.Hide()
// 		return nil
// 	}
// 	return event
// }
