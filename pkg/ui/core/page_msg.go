package core

import tea "github.com/charmbracelet/bubbletea"

// PageMsg carries the result of a command a page started, addressed to the
// page instance that started it. The router wraps every command a page returns
// from Init, OnFocus, OnBlur or Update, so a fetch that finishes after the user
// has moved on reaches the page that asked for it rather than whichever page
// happens to be showing.
//
// Gen identifies the page instance: when a page is evicted and later rebuilt
// under the same ID, results addressed to the old instance are dropped. Ctx is
// the cluster context that was active when the page started the command.
//
// Code outside the router rarely needs this type. The shell unwraps it to react
// to the payload (notifications, confirmations, navigation) and hands the
// envelope on to the router, which delivers the payload to its page. Before it
// acts on a payload the user must answer, it asks Router.IsLive.
type PageMsg struct {
	PageID string
	Gen    uint64
	Ctx    string
	Msg    tea.Msg
}
