// Package connector implements the connector detail page (dynamic page ID
// "connector:<connect>:<name>"). It renders a summary strip plus four tabs —
// Overview, Tasks, Config and Topics — over the shared template shell. The page
// is created by the router; see NewModelWithCommon for the constructor the
// router wires to the "connector:<connect>:<name>" dynamic ID.
package connector

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components/editor"
	"github.com/Benny93/kafui/pkg/ui/components/tabstrip"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// Model is the connector detail page.
type Model struct {
	common      *core.Common
	reusableApp *templateui.ReusableApp
	dims        core.Dimensions

	connect string
	name    string

	details       api.ConnectorDetails
	detailsLoaded bool
	notFound      bool
	loadErr       error

	active tab
	// tabStrip owns the tab bar's click and hover zones.
	tabStrip *tabstrip.Model

	// Tasks tab
	tasksTable   table.Model
	expandedTask int // index of the expanded task, -1 when none

	// Config tab
	configEditor *editor.Editor
	editing      bool
	configText   string // the masked JSON currently displayed (edit baseline)
}

// NewModelWithCommon builds the connector detail page for the given Connect
// cluster and connector name. The router wires this to the
// "connector:<connect>:<name>" dynamic page ID.
func NewModelWithCommon(common *core.Common, connectCluster, connectorName string) core.Page {
	return newModel(common, connectCluster, connectorName)
}

func newModel(common *core.Common, connect, name string) *Model {
	m := &Model{
		common:       common,
		connect:      connect,
		name:         name,
		expandedTask: -1,
	}

	m.tasksTable = table.New(table.WithColumns(taskColumns()), table.WithFocused(true), table.WithHeight(10))
	m.configEditor = editor.NewEditor("")

	config := &providers.AppConfig{
		ContentProvider:      &contentProvider{model: m},
		ShowSidebarByDefault: false,
	}
	m.reusableApp = templateui.NewReusableApp(config)
	m.reusableApp.SetKeyMap(keys.Hints(pageScope()))
	return m
}

func (m *Model) Init() tea.Cmd { return m.reusableApp.Init() }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.reusableApp.Update(msg)
	if app, ok := updated.(*templateui.ReusableApp); ok {
		m.reusableApp = app
	}
	return m, cmd
}

func (m *Model) View() string { return m.reusableApp.View() }

func (m *Model) SetDimensions(width, height int) {
	m.dims = core.Dimensions{Width: width, Height: height}
	body := height - 12
	if body < 3 {
		body = 3
	}
	m.tasksTable.SetWidth(width)
	m.tasksTable.SetHeight(body)
	m.configEditor.SetDimensions(width, body)
	m.reusableApp.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

func (m *Model) GetID() string { return fmt.Sprintf("connector:%s:%s", m.connect, m.name) }

func (m *Model) GetTitle() string { return m.name }

func (m *Model) GetHelp() []key.Binding {
	return keys.Help(pageScope())
}

func (m *Model) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) { return m, nil }

func (m *Model) OnBlur() tea.Cmd { return nil }

// OnFocus kicks off the initial detail load.
func (m *Model) OnFocus() tea.Cmd { return m.loadDetails() }

// --- loads ---

func (m *Model) loadDetails() tea.Cmd {
	ds := m.common.DataSource
	connect, name := m.connect, m.name
	return func() tea.Msg {
		details, err := ds.GetConnectorDetails(connect, name)
		if err != nil {
			var nf api.ConnectorNotFoundError
			if asConnectorNotFound(err, &nf) {
				return detailsLoadedMsg{connect: connect, name: name, found: false}
			}
			return detailsLoadedMsg{connect: connect, name: name, err: err}
		}
		return detailsLoadedMsg{connect: connect, name: name, details: details, found: true}
	}
}

// --- message handling (via the content provider) ---

func (m *Model) handle(msg tea.Msg) tea.Cmd {
	// Tab strip mouse handling: hovering a tab highlights it, clicking one
	// activates it. Handled before anything else so a click on the bar never
	// reaches the pane behind it.
	if mouse, ok := msg.(tea.MouseMsg); ok {
		if clicked, hit := m.tabs().HandleMouse(mouse); hit {
			return m.switchTab(tab(clicked))
		}
	}

	switch v := msg.(type) {
	case detailsLoadedMsg:
		if v.connect != m.connect || v.name != m.name {
			return nil
		}
		if v.err != nil {
			m.loadErr = v.err
			return nil
		}
		m.notFound = !v.found
		m.loadErr = nil
		if v.found {
			m.details = v.details
			m.detailsLoaded = true
			m.rebuildTaskTable()
		}
		return nil
	case lifecycleResultMsg:
		return m.handleLifecycleResult(v)
	case taskRestartResultMsg:
		return m.handleTaskRestartResult(v)
	case configUpdatedMsg:
		return m.handleConfigUpdated(v)
	case tea.KeyMsg:
		return m.handleKey(v)
	}
	return m.forwardToActive(msg)
}

func (m *Model) forwardToActive(msg tea.Msg) tea.Cmd {
	switch m.active {
	case tabConfig:
		if m.editing {
			_, cmd := m.configEditor.Update(msg)
			return cmd
		}
	case tabTasks:
		var cmd tea.Cmd
		m.tasksTable, cmd = m.tasksTable.Update(msg)
		return cmd
	}
	return nil
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Config edit sub-state swallows keys except save/cancel.
	if m.editing {
		switch msg.String() {
		case "esc":
			m.editing = false
			return nil
		case "f2":
			// ctrl+s used to save here. It is XOFF and can freeze the terminal,
			// so the registry forbids it.
			return m.commitConfigEdit()
		}
		_, cmd := m.configEditor.Update(msg)
		return cmd
	}

	// Every key resolves through the single binding registry. The nine bare
	// letters this screen used to claim (r p u s R z t T f) appeared in no
	// help text; the lifecycle ones now live in the actions menu.
	action, bound := keys.Default.Resolve(keys.ScopeConnector, msg.String())
	if bound {
		switch action {
		case keys.ActionSelectTab:
			if n := tab(msg.String()[0] - '1'); int(n) < len(tabTitles) {
				return m.switchTab(n)
			}
			return nil
		case keys.ActionFocusNext:
			return m.switchTab((m.active + 1) % tab(len(tabTitles)))
		case keys.ActionFocusPrev:
			return m.switchTab((m.active + tab(len(tabTitles)) - 1) % tab(len(tabTitles)))
		case keys.ActionRefresh:
			return m.retry()
		case keys.ActionPause:
			// One key for the pause/resume pair, chosen by current state, in
			// place of p for pause and u for resume.
			if strings.EqualFold(m.details.State, api.ConnectorStatePaused) {
				return m.lifecycle("resume", m.common.DataSource.ResumeConnector)
			}
			return m.lifecycle("pause", m.common.DataSource.PauseConnector)
		case keys.ActionDelete:
			return m.deleteConnector()
		case keys.ActionEdit:
			if m.active == tabConfig {
				return m.beginConfigEdit()
			}
			return nil
		}
	}

	// Tab-specific keys.
	switch m.active {
	case tabTasks:
		return m.handleTasksKey(msg)
	case tabTopics:
		return m.handleTopicsKey(msg)
	}
	return m.forwardToActive(msg)
}

func (m *Model) switchTab(t tab) tea.Cmd {
	m.active = t
	return nil
}

func (m *Model) retry() tea.Cmd {
	m.notFound = false
	m.loadErr = nil
	m.detailsLoaded = false
	return m.loadDetails()
}

// --- lifecycle actions (KC-14/KC-16) ---

func (m *Model) handleTopicsKey(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "enter" && len(m.details.Topics) > 0 {
		// Navigate to the first topic (single-list, no cursor state kept here).
		// ponytail: per-row topic cursor deferred; opens the first topic.
		topic := m.details.Topics[0]
		return core.NewPageChangeMsg("topic:"+topic, map[string]interface{}{"name": topic})
	}
	return nil
}

// --- Config tab (KC-18) ---

// asConnectorNotFound reports whether err is (or wraps) an api.ConnectorNotFoundError.
func asConnectorNotFound(err error, dst *api.ConnectorNotFoundError) bool {
	if nf, ok := err.(api.ConnectorNotFoundError); ok {
		*dst = nf
		return true
	}
	return false
}
