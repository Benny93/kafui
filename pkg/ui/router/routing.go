package router

import (
	"reflect"

	"github.com/Benny93/kafui/pkg/ui/core"
	tea "github.com/charmbracelet/bubbletea"
)

// maxPending bounds the results held for a page that is not showing. Pages keep
// at most a handful of commands in flight (each listener or timer re-arms only
// when its message is delivered), so hitting this means a page is misbehaving;
// dropping beyond it keeps one such page from growing without bound.
const maxPending = 256

// teaPkg is the package of bubbletea's own messages. The runtime acts on
// several of them (quit, batch, sequence, mouse and cursor modes) before the
// model sees anything, so they must reach it unwrapped.
var teaPkg = reflect.TypeOf(tea.QuitMsg{}).PkgPath()

// register stores page under pageID as a new instance. Results addressed to a
// previous instance with the same ID are dropped from here on.
func (r *Router) register(pageID string, page core.Page) {
	r.nextGen++
	r.pages[pageID] = page
	r.gens[pageID] = r.nextGen
	delete(r.pending, pageID)
}

// tag addresses the messages cmd produces to the page instance stored under
// pageID, so they reach that page even if another page is showing by then. The
// address also records the cluster context active now, when the page starts
// the command, so a result that arrives after a cluster switch can be told
// apart from one meant for the cluster showing.
func (r *Router) tag(pageID string, cmd tea.Cmd) tea.Cmd {
	return tagCmd(core.PageMsg{PageID: pageID, Gen: r.gens[pageID], Ctx: r.activeContext()}, cmd)
}

// activeContext is the cluster context the data source points at right now.
func (r *Router) activeContext() string {
	if r.com == nil || r.com.DataSource == nil {
		return ""
	}
	return r.com.DataSource.GetContext()
}

// tagCmd wraps cmd so its result is addressed to `to` (its Msg is ignored).
func tagCmd(to core.PageMsg, cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg { return tagMsg(to, cmd()) }
}

func tagMsg(to core.PageMsg, msg tea.Msg) tea.Msg {
	switch m := msg.(type) {
	case nil:
		return nil
	case core.PageMsg:
		return m
	case tea.BatchMsg:
		// The runtime runs each command of a batch on its own; tag them one by
		// one so their results are still addressed.
		out := make(tea.BatchMsg, len(m))
		for i, c := range m {
			out[i] = tagCmd(to, c)
		}
		return out
	}
	if isRuntimeMsg(msg) {
		// Includes tea.Sequence: its commands then run untagged and reach the
		// page that is showing, as every message did before tagging.
		return msg
	}
	to.Msg = msg
	return to
}

// IsLive reports whether pm comes from the page instance that is showing, on
// the cluster that is active. The shell asks before it acts on a payload the
// user must answer, such as a confirmation: one raised by a page the user has
// left, by an evicted or rebuilt instance, or before a cluster switch would
// otherwise pop over whatever is showing now and act on the wrong cluster.
func (r *Router) IsLive(pm core.PageMsg) bool {
	if _, ok := r.pages[pm.PageID]; !ok || r.gens[pm.PageID] != pm.Gen {
		return false
	}
	return pm.PageID == r.currentPage && pm.Ctx == r.activeContext()
}

func isRuntimeMsg(msg tea.Msg) bool {
	t := reflect.TypeOf(msg)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.PkgPath() == teaPkg
}

// routePageMsg delivers a page-addressed result. The page that is showing gets
// it at once, through the same path as any other message. A page that is
// cached but not showing gets it when it is shown again, so its listeners and
// timers resume where they stopped instead of dying or feeding another page.
// Results for an evicted or replaced page are dropped.
func (r *Router) routePageMsg(pm core.PageMsg) tea.Cmd {
	switch pm.Msg.(type) {
	case core.BackMsg, core.PageChangeMsg:
		// Navigation requests act on the router whichever page raised them.
		_, cmd := r.Update(pm.Msg)
		return cmd
	}
	if _, ok := r.pages[pm.PageID]; !ok || r.gens[pm.PageID] != pm.Gen {
		return nil
	}
	if pm.PageID == r.currentPage {
		_, cmd := r.Update(pm.Msg)
		return cmd
	}
	if q := r.pending[pm.PageID]; len(q) < maxPending {
		r.pending[pm.PageID] = append(q, pm.Msg)
	}
	return nil
}

// flushPending delivers the results that arrived for pageID while it was not
// showing.
func (r *Router) flushPending(pageID string) tea.Cmd {
	queued := r.pending[pageID]
	if len(queued) == 0 {
		return nil
	}
	delete(r.pending, pageID)
	cmds := make([]tea.Cmd, 0, len(queued))
	for _, msg := range queued {
		page, ok := r.pages[pageID]
		if !ok {
			break
		}
		updated, cmd := page.Update(msg)
		if p, ok := updated.(core.Page); ok && p != nil {
			r.pages[pageID] = p
		}
		cmds = append(cmds, r.tag(pageID, cmd))
	}
	return tea.Batch(cmds...)
}
