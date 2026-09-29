package ui

import (
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/core"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// While help is open the shell keeps the keyboard, but other messages must
// still reach the router. Dropping them there lost fetch results and stopped
// listener chains for good (CONC-4).
func TestHelpOverlayStillForwardsNonKeyMessages(t *testing.T) {
	m := newShell(t)
	m.Update(showHelpMsg{})
	assert.Equal(t, core.StateHelp, m.GetState())

	m.Update(core.PageMsg{PageID: "main", Gen: 1, Msg: core.PageChangeMsg{PageID: "metrics"}})
	assert.Equal(t, "metrics", m.Router.GetCurrentPageID(), "a message raised behind the help overlay was dropped")
	assert.Equal(t, core.StateHelp, m.GetState())
}

// The shell contract for text entry: while the current page reports
// core.InputModeReporter.IsInputMode, shell shortcuts are typed text. A page
// that holds a search box or form must implement it on the page model the
// router holds, not only on its content provider.
func TestShellShortcutsAreTypedWhilePageTakesInput(t *testing.T) {
	m := newShell(t)
	m.Router.NavigateTo("ksql_query", nil)
	p, ok := m.Router.GetCurrentPage().(core.InputModeReporter)
	require.True(t, ok)
	require.True(t, p.IsInputMode(), "the ksql editor takes input while idle")

	for _, k := range []string{"q", "a", ":", "?"} {
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		if k == "q" && cmd != nil {
			_, quit := cmd().(tea.QuitMsg)
			assert.False(t, quit, "%q quit the application while typing", k)
		}
		assert.False(t, m.palette.Active(), "%q opened the palette while typing", k)
		assert.False(t, m.actions.Active(), "%q opened the actions menu while typing", k)
		assert.Equal(t, core.StateNormal, m.GetState(), "%q opened help while typing", k)
	}
}

// ctxDS is the mock data source with a cluster context the test can switch
// without touching the mock's package-level state.
type ctxDS struct {
	*mock.KafkaDataSourceMock
	ctx string
}

func (d *ctxDS) GetContext() string { return d.ctx }

// liveEnvelope addresses msg to the showing page the way the router does when
// that page starts a command: it finds the instance number the router
// accepts for it right now.
func liveEnvelope(t *testing.T, m *Model, ctx string, msg tea.Msg) core.PageMsg {
	t.Helper()
	pm := core.PageMsg{PageID: m.Router.GetCurrentPageID(), Ctx: ctx, Msg: msg}
	for pm.Gen = 1; pm.Gen < 1000; pm.Gen++ {
		if m.Router.IsLive(pm) {
			return pm
		}
	}
	t.Fatal("no live instance for the showing page")
	return pm
}

// A page raises a confirmation from a command that may finish after the user
// left it or switched clusters. The shell must not open it then: confirming
// would run the page's action against the cluster active now (router-1).
func TestShellDropsConfirmationFromStalePage(t *testing.T) {
	tests := []struct {
		name  string
		stale func(m *Model, ds *ctxDS)
		shown bool
	}{
		{"live page", func(*Model, *ctxDS) {}, true},
		{"cluster switched", func(_ *Model, ds *ctxDS) { ds.ctx = "B" }, false},
		{"user left the page", func(m *Model, _ *ctxDS) { m.Router.NavigateTo("metrics", nil) }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := &mock.KafkaDataSourceMock{}
			base.Init("")
			ds := &ctxDS{KafkaDataSourceMock: base, ctx: "A"}
			m := initialModelWithRouter(ds)
			m.Init()
			m.Update(tea.WindowSizeMsg{Width: 140, Height: 44})
			require.Equal(t, "main", m.Router.GetCurrentPageID())

			pm := liveEnvelope(t, m, "A", core.ShowConfirmMsg{Title: "Delete topic \"orders\"?"})
			tt.stale(m, ds)
			m.Update(pm)
			assert.Equal(t, tt.shown, m.confirm.Active())
		})
	}
}
