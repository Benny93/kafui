package ksql

import (
	"context"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components/editor"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// QueryModel is the ksqlDB query editor page.
type QueryModel struct {
	common      *core.Common
	reusableApp *templateui.ReusableApp
	dims        core.Dimensions

	editor *editor.Editor
	props  []propRow

	// focusIdx: 0 = editor; 1..2N = property inputs (row r field f → 1+2r+f).
	focusIdx int

	// streaming state
	running   bool
	aborted   bool
	ch        <-chan api.KsqlResultTable
	cancel    context.CancelFunc
	wasSelect bool
	// gen identifies the current query. Every async message carries the gen
	// it was started under, so messages from a query that was stopped or
	// replaced are dropped instead of driving the current one.
	gen int

	// result display
	resTable    table.Model
	resCols     []string
	resRows     [][]string
	resTitle    string
	hasResult   bool
	placeholder bool
	truncated   bool
	errPanel    string
}

// NewQueryModelWithCommon builds the ksqlDB query editor page. The intended
// router page ID is "ksql_query".
func NewQueryModelWithCommon(common *core.Common) core.Page {
	return newQueryModel(common, "")
}

// NewQueryModelWithSeed builds the query page with the editor pre-seeded (used
// when opened from a stream/table row on the overview page). The router passes
// the seed via the navigation "name" field.
func NewQueryModelWithSeed(common *core.Common, seed string) core.Page {
	return newQueryModel(common, seed)
}

func newQueryModel(common *core.Common, seed string) *QueryModel {
	m := &QueryModel{
		common: common,
		editor: editor.NewEditor(seed),
	}
	m.resTable = table.New(table.WithFocused(false), table.WithHeight(10))

	config := &providers.AppConfig{
		ContentProvider:      &queryContentProvider{model: m},
		ShowSidebarByDefault: false,
	}
	m.reusableApp = templateui.NewReusableApp(config)
	m.reusableApp.SetKeyMap(keys.Hints(queryScope()))
	return m
}

// --- core.Page ---

func (m *QueryModel) Init() tea.Cmd { return m.reusableApp.Init() }

func (m *QueryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.reusableApp.Update(msg)
	if app, ok := updated.(*templateui.ReusableApp); ok {
		m.reusableApp = app
	}
	return m, cmd
}

func (m *QueryModel) View() string { return m.reusableApp.View() }

func (m *QueryModel) SetDimensions(width, height int) {
	m.dims = core.Dimensions{Width: width, Height: height}
	edH := 6
	m.editor.SetDimensions(width, edH)
	m.resTable.SetWidth(width)
	body := height - 22
	if body < 3 {
		body = 3
	}
	m.resTable.SetHeight(body)
	m.reusableApp.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

func (m *QueryModel) GetID() string { return "ksql_query" }

func (m *QueryModel) GetTitle() string { return "ksqlDB Query" }

func (m *QueryModel) GetHelp() []key.Binding {
	return keys.Help(queryScope())
}

func (m *QueryModel) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) { return m, nil }

func (m *QueryModel) OnFocus() tea.Cmd { return m.editor.Focus() }

// OnBlur cancels any in-flight query when navigating away (KS-15). Leaving the
// page mid-stream surfaces the same "cancelled" notice as an explicit abort.
func (m *QueryModel) OnBlur() tea.Cmd {
	if m.running {
		m.stopQuery()
		return core.NewNotification(core.StatusInfo, "ksqlDB", "consumption cancelled")
	}
	m.stopQuery()
	return nil
}

// IsInputMode reports whether the shell should let keystrokes reach the editor
// unmodified. True while editing (not running) so SQL text (including 'q') is
// typed rather than triggering global hotkeys.
func (m *QueryModel) IsInputMode() bool { return !m.running }

// --- message handling ---

func (m *QueryModel) handle(msg tea.Msg) tea.Cmd {
	switch v := msg.(type) {
	case queryStartedMsg:
		if v.gen != m.gen || !m.running {
			// The query was stopped (page left) or replaced before its
			// stream opened: close it rather than let it stream unobserved.
			v.cancel()
			return nil
		}
		if m.aborted {
			return m.finish()
		}
		m.ch = v.ch
		return listenForResults(v.ch, v.gen)
	case queryTickMsg:
		if v.gen == m.gen && m.running && m.ch != nil {
			return listenForResults(m.ch, m.gen)
		}
		return nil
	case ksqlResultMsg:
		if v.gen != m.gen || !m.running {
			return nil
		}
		return m.handleResult(v)
	case tea.KeyMsg:
		return m.handleKey(v)
	}
	return m.forwardToFocused(msg)
}

func (m *QueryModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Text-entry scope: everything printable is typed. Statement execution is
	// F5, the one action key that is not a printable character. Clearing the
	// editor, clearing results and the property rows are actions-menu entries.
	action, bound := keys.Default.Resolve(queryScope(), msg.String())
	if !bound {
		return m.forwardToFocused(msg)
	}
	switch action {
	case keys.ActionRefresh:
		return m.execute()
	case keys.ActionFocusNext:
		m.advanceFocus()
		return nil
	case keys.ActionClearField:
		if !m.running {
			m.editor.SetValue("")
		}
		return nil
	}
	return m.forwardToFocused(msg)
}

// forwardToFocused routes the key to the currently focused input.
func (m *QueryModel) forwardToFocused(msg tea.Msg) tea.Cmd {
	if m.running {
		return nil
	}
	if m.focusIdx == 0 {
		_, cmd := m.editor.Update(msg)
		return cmd
	}
	idx := m.focusIdx - 1
	row := idx / 2
	if row >= len(m.props) {
		return nil
	}
	var cmd tea.Cmd
	if idx%2 == 0 {
		m.props[row].keyIn, cmd = m.props[row].keyIn.Update(msg)
	} else {
		m.props[row].valIn, cmd = m.props[row].valIn.Update(msg)
	}
	return cmd
}

// advanceFocus cycles editor → each property input → back to editor.
func (m *QueryModel) advanceFocus() {
	total := 1 + 2*len(m.props)
	m.focusIdx = (m.focusIdx + 1) % total
	m.syncFocus()
}

func (m *QueryModel) syncFocus() {
	if m.focusIdx == 0 {
		m.editor.Focus()
	} else {
		m.editor.Blur()
	}
	for i := range m.props {
		m.props[i].keyIn.Blur()
		m.props[i].valIn.Blur()
	}
	if m.focusIdx > 0 {
		idx := m.focusIdx - 1
		row := idx / 2
		if row < len(m.props) {
			if idx%2 == 0 {
				m.props[row].keyIn.Focus()
			} else {
				m.props[row].valIn.Focus()
			}
		}
	}
}
