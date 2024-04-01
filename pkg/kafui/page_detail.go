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

	valueTextView := tview.NewTextView().
		//SetText("Placeholder :)").
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true).
		SetWordWrap(false)
	valueTextView.SetTextColor(tcell.ColorWhite)
	valueTextView.SetBorder(true)

	// Format and colorize JSON
	var obj map[string]interface{}
	merror := json.Unmarshal([]byte(value), &obj)
	f := colorjson.NewFormatter()
	f.Indent = 2
	s, err := f.Marshal(obj)
	if merror != nil || err != nil {
		valueTextView.SetText(value)
	} else {
		writer := tview.ANSIWriter(valueTextView)
		writer.Write(s)
	}

	return &DetailPage{
		app:           app,
		pages:         pages,
		value:         value,
		valueTextView: valueTextView,
	}
}

func (vp *DetailPage) Show() {
	// Create a new flex layout for the value page
	valueFlex := tview.NewFlex()
	valueFlex.SetDirection(tview.FlexRow)
	//SetDirection(tview.FlexRow).
	//AddItem(tview.NewTextView().SetText("Message Value").SetTextAlign(tview.AlignCenter), 1, 0, false).
	//AddItem(tview.NewTextView(), 2, 1, false)
	valueFlex.AddItem(vp.CreateInputLegend(), 5, 1, false)
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

func (vp *DetailPage) CreateInputLegend() *tview.Flex {
	flex := tview.NewFlex()
	flex.SetBorderPadding(0, 0, 1, 0)
	left := tview.NewFlex().SetDirection(tview.FlexRow)
	right := tview.NewFlex().SetDirection(tview.FlexRow)
	right.SetBorderPadding(0, 1, 0, 0)

	left.AddItem(CreateRunInfo("↑", "Move up"), 0, 1, false)
	left.AddItem(CreateRunInfo("↓", "Move down"), 0, 1, false)
	left.AddItem(CreateRunInfo("g", "Scroll to top"), 0, 1, false)
	left.AddItem(CreateRunInfo("G", "Scroll to bottom"), 0, 1, false)
	left.AddItem(CreateRunInfo("c", "Copy content"), 0, 1, false)
	//right.AddItem(CreateRunInfo("Enter", "Show value"), 0, 1, false)
	right.AddItem(CreateRunInfo("Esc", "Go Back"), 0, 1, false)

	flex.AddItem(left, 0, 1, false)
	flex.AddItem(right, 0, 1, false)

	return flex
}
