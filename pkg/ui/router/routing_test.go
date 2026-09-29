package router

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePage records its lifecycle calls and the messages it receives.
type fakePage struct {
	id       string
	inits    int
	focuses  int
	received []tea.Msg
	disposed bool
	// reply, when set, is returned as the command from Update.
	reply tea.Cmd
}

type resultMsg struct{ n int }

func (p *fakePage) Init() tea.Cmd { p.inits++; return nil }
func (p *fakePage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	p.received = append(p.received, msg)
	return p, p.reply
}
func (p *fakePage) View() string                                  { return p.id }
func (p *fakePage) SetDimensions(int, int)                        {}
func (p *fakePage) GetID() string                                 { return p.id }
func (p *fakePage) GetTitle() string                              { return p.id }
func (p *fakePage) GetHelp() []key.Binding                        { return nil }
func (p *fakePage) HandleNavigation(tea.Msg) (core.Page, tea.Cmd) { return p, nil }
func (p *fakePage) OnFocus() tea.Cmd                              { p.focuses++; return nil }
func (p *fakePage) OnBlur() tea.Cmd                               { return nil }
func (p *fakePage) Dispose()                                      { p.disposed = true }

// newFakeRouter builds a router whose pages are fakes: main is showing, and
// a and b are cached behind it.
func newFakeRouter(t *testing.T) (*Router, map[string]*fakePage) {
	t.Helper()
	return newFakeRouterWith(t, &mockDataSource{})
}

func newFakeRouterWith(t *testing.T, ds api.KafkaDataSource) (*Router, map[string]*fakePage) {
	t.Helper()
	r := NewRouter(core.NewCommon(ds))
	pages := map[string]*fakePage{}
	for _, id := range []string{"main", "topic:a", "topic:b"} {
		p := &fakePage{id: id}
		pages[id] = p
		r.register(id, p)
	}
	r.currentPage = "main"
	return r, pages
}

func addressed(r *Router, pageID string, msg tea.Msg) core.PageMsg {
	return core.PageMsg{PageID: pageID, Gen: r.gens[pageID], Ctx: r.activeContext(), Msg: msg}
}

func TestTagMsg(t *testing.T) {
	inner := func() tea.Msg { return resultMsg{1} }
	tests := []struct {
		name  string
		msg   tea.Msg
		check func(t *testing.T, got tea.Msg)
	}{
		{"nil stays nil", nil, func(t *testing.T, got tea.Msg) { assert.Nil(t, got) }},
		{"own messages are addressed", resultMsg{1}, func(t *testing.T, got tea.Msg) {
			assert.Equal(t, core.PageMsg{PageID: "p", Gen: 7, Msg: resultMsg{1}}, got)
		}},
		{"runtime messages pass through", tea.QuitMsg{}, func(t *testing.T, got tea.Msg) {
			assert.Equal(t, tea.QuitMsg{}, got)
		}},
		{"synthetic keys pass through", tea.KeyMsg{Type: tea.KeyDown}, func(t *testing.T, got tea.Msg) {
			assert.Equal(t, tea.KeyMsg{Type: tea.KeyDown}, got)
		}},
		{"already addressed is kept", core.PageMsg{PageID: "other", Gen: 1}, func(t *testing.T, got tea.Msg) {
			assert.Equal(t, core.PageMsg{PageID: "other", Gen: 1}, got)
		}},
		{"batch members are addressed one by one", tea.BatchMsg{inner, nil}, func(t *testing.T, got tea.Msg) {
			batch, ok := got.(tea.BatchMsg)
			require.True(t, ok, "a batch must stay a batch so the runtime runs it")
			require.Len(t, batch, 2)
			assert.Equal(t, core.PageMsg{PageID: "p", Gen: 7, Msg: resultMsg{1}}, batch[0]())
			assert.Nil(t, batch[1])
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tagMsg(core.PageMsg{PageID: "p", Gen: 7}, tt.msg))
		})
	}
}

// A result for a page that is not showing must not reach the page that is,
// and must reach its own page once that page is shown again (CONC-4).
func TestPageResultWaitsForItsPage(t *testing.T) {
	r, pages := newFakeRouter(t)
	r.currentPage = "topic:b"

	r.Update(addressed(r, "topic:a", resultMsg{1}))
	assert.Empty(t, pages["topic:b"].received, "topic:a's result was delivered to topic:b")
	assert.Empty(t, pages["topic:a"].received, "a hidden page gets its results when shown")

	r.navigateToWithoutHistory("topic:a", nil)
	assert.Equal(t, []tea.Msg{resultMsg{1}}, pages["topic:a"].received)
	assert.Empty(t, r.pending["topic:a"])
}

func TestPageResultForShowingPageIsDeliveredAndReaddressed(t *testing.T) {
	r, pages := newFakeRouter(t)
	pages["main"].reply = func() tea.Msg { return resultMsg{2} }

	_, cmd := r.Update(addressed(r, "main", resultMsg{1}))
	assert.Equal(t, []tea.Msg{resultMsg{1}}, pages["main"].received)
	require.NotNil(t, cmd)
	assert.Equal(t, addressed(r, "main", resultMsg{2}), cmd(), "the follow-up command must stay addressed")
}

func TestPageResultForReplacedInstanceIsDropped(t *testing.T) {
	r, pages := newFakeRouter(t)
	stale := addressed(r, "main", resultMsg{1})

	fresh := &fakePage{id: "main"}
	r.register("main", fresh)
	r.Update(stale)

	assert.Empty(t, pages["main"].received)
	assert.Empty(t, fresh.received, "a rebuilt page must not receive its predecessor's results")
}

func TestNavigationFromAddressedMessageStillNavigates(t *testing.T) {
	r, _ := newFakeRouter(t)
	r.Update(addressed(r, "topic:a", core.PageChangeMsg{PageID: "topic:b"}))
	assert.Equal(t, "topic:b", r.GetCurrentPageID())
}

// Init runs once per page instance; returning to a cached page only focuses
// it. Re-running Init on Back made the topic page refetch and drop its loaded
// batches and position (CONC-5, ARCH-4).
func TestReturningToCachedPageFocusesWithoutReinit(t *testing.T) {
	r, pages := newFakeRouter(t)
	a := pages["topic:a"]

	r.NavigateTo("topic:a", nil)
	r.NavigateTo("topic:b", nil)
	r.Back()

	assert.Equal(t, "topic:a", r.GetCurrentPageID())
	assert.Equal(t, 0, a.inits, "a cached page must not be re-initialised")
	assert.Equal(t, 2, a.focuses, "every activation focuses the page")
}

// The shell acts on some payloads itself (a confirmation dialog) before the
// router sees them. IsLive tells it whether the page that raised one is still
// the instance showing, on the cluster that is active.
func TestIsLive(t *testing.T) {
	tests := []struct {
		name  string
		setup func(r *Router, ds *switchableDataSource) core.PageMsg
		want  bool
	}{
		{"showing page, same cluster", func(r *Router, _ *switchableDataSource) core.PageMsg {
			return addressed(r, "main", resultMsg{1})
		}, true},
		{"cached page not showing", func(r *Router, _ *switchableDataSource) core.PageMsg {
			return addressed(r, "topic:a", resultMsg{1})
		}, false},
		{"evicted page", func(r *Router, _ *switchableDataSource) core.PageMsg {
			pm := addressed(r, "topic:a", resultMsg{1})
			r.evict("topic:a")
			return pm
		}, false},
		{"rebuilt instance", func(r *Router, _ *switchableDataSource) core.PageMsg {
			pm := addressed(r, "main", resultMsg{1})
			r.register("main", &fakePage{id: "main"})
			return pm
		}, false},
		{"cluster switched since the command started", func(r *Router, ds *switchableDataSource) core.PageMsg {
			pm := addressed(r, "main", resultMsg{1})
			_ = ds.SetContext("B")
			return pm
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &switchableDataSource{ctx: "A"}
			r, _ := newFakeRouterWith(t, ds)
			assert.Equal(t, tt.want, r.IsLive(tt.setup(r, ds)))
		})
	}
}

// tag stamps the cluster active when the page starts the command, not when
// the command finishes.
func TestTagRecordsContextAtStart(t *testing.T) {
	ds := &switchableDataSource{ctx: "A"}
	r, _ := newFakeRouterWith(t, ds)
	cmd := r.tag("main", func() tea.Msg { return resultMsg{1} })
	_ = ds.SetContext("B")
	pm, ok := cmd().(core.PageMsg)
	require.True(t, ok)
	assert.Equal(t, "A", pm.Ctx)
}
