// Package broker implements the broker detail page (dynamic page ID
// "broker:<id>"). It renders a summary strip plus three tabs — Log Dirs,
// Configs and Metrics — over the shared template shell. The page is created by
// the router; see NewModelWithCommon / NewModelWithInfo for the constructors the
// router wires to the "broker:<id>" dynamic ID.
package broker

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components/editor"
	"github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/components/tabstrip"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Model is the broker detail page.
type Model struct {
	common      *core.Common
	reusableApp *templateui.ReusableApp
	dims        core.Dimensions

	brokerID   int32
	info       api.BrokerInfo
	infoLoaded bool
	notFound   bool

	active tab
	// tabStrip owns the tab bar's click and hover zones.
	tabStrip *tabstrip.Model

	// Log Dirs tab
	logDirs       []api.BrokerLogDir
	logDirsLoaded bool
	logDirsErr    error
	logTable      table.Model
	expanded      int // index of the expanded dir, -1 when none
	partTable     table.Model

	// Configs tab
	configs       []api.BrokerConfigEntry
	configsLoaded bool
	configsErr    error
	cfgTable      table.Model
	cfgVisible    []api.BrokerConfigEntry // entries backing the current cfgTable rows
	cfgFilter     string
	searching     bool
	searchInput   textinput.Model
	editing       bool
	editKey       string
	editOld       string
	editInput     textinput.Model

	// Metrics tab
	metricsViewer *editor.Viewer
	metricsLoaded bool
	metricsErr    error

	// Reassignment form
	moveForm *form.Form
}

// NewModelWithCommon builds the broker detail page for the given broker ID.
// The router wires this to the "broker:<id>" dynamic page ID.
func NewModelWithCommon(common *core.Common, brokerID int32) core.Page {
	return newModel(common, brokerID, api.BrokerInfo{}, false)
}

// NewModelWithInfo builds the page with broker metadata already known (passed via
// NavigationData from the list row), avoiding a refetch for the summary strip.
func NewModelWithInfo(common *core.Common, brokerID int32, info api.BrokerInfo) core.Page {
	return newModel(common, brokerID, info, true)
}

func newModel(common *core.Common, brokerID int32, info api.BrokerInfo, haveInfo bool) *Model {
	m := &Model{
		common:   common,
		brokerID: brokerID,
		expanded: -1,
	}
	if haveInfo {
		m.info = info
		m.infoLoaded = true
	}

	si := textinput.New()
	si.Prompt = "/"
	m.searchInput = si
	ei := textinput.New()
	m.editInput = ei

	m.logTable = table.New(table.WithColumns(logDirColumns()), table.WithFocused(true), table.WithHeight(10))
	m.partTable = table.New(table.WithColumns(partitionColumns()), table.WithFocused(true), table.WithHeight(8))
	m.cfgTable = table.New(table.WithColumns(configColumns()), table.WithFocused(true), table.WithHeight(12))
	m.metricsViewer = editor.NewViewer("")
	m.metricsViewer.SetHighlight(true)

	config := &providers.AppConfig{
		ContentProvider:      &contentProvider{model: m},
		ShowSidebarByDefault: false,
	}
	m.reusableApp = templateui.NewReusableApp(config)
	m.reusableApp.SetKeyMap(keys.Hints(pageScope()))
	return m
}

// --- column definitions ---

// Init runs once, when the router creates the page, and kicks off the initial
// data load: broker list (for found/not-found + summary) and the default tab's
// data. The summary strip's disk usage is derived from the log dirs, so no
// separate GetBrokerStats (a second DescribeLogDirs, cluster-wide) is issued.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.reusableApp.Init(), m.initialLoad())
}

func (m *Model) initialLoad() tea.Cmd {
	return tea.Batch(m.loadInfo(), m.loadTab(tabLogDirs))
}

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
	m.logTable.SetWidth(width)
	m.logTable.SetHeight(body)
	m.partTable.SetWidth(width)
	m.partTable.SetHeight(body / 2)
	m.cfgTable.SetWidth(width)
	m.cfgTable.SetHeight(body)
	m.metricsViewer.SetDimensions(width, body)
	m.reusableApp.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

func (m *Model) GetID() string { return fmt.Sprintf("broker:%d", m.brokerID) }

func (m *Model) GetTitle() string { return fmt.Sprintf("Broker %d", m.brokerID) }

func (m *Model) GetHelp() []key.Binding {
	return keys.Help(pageScope())
}

func (m *Model) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) { return m, nil }

func (m *Model) OnBlur() tea.Cmd { return nil }

// OnFocus loads nothing: Init did the initial load, and a return keeps the
// loaded state (r refreshes the active tab).
func (m *Model) OnFocus() tea.Cmd { return nil }

// --- loads ---

func (m *Model) loadInfo() tea.Cmd {
	ds := m.common.DataSource
	id := m.brokerID
	return func() tea.Msg {
		brokers, err := ds.GetBrokers()
		if err != nil {
			return brokerInfoLoadedMsg{brokerID: id, err: err}
		}
		for _, b := range brokers {
			if b.ID == id {
				return brokerInfoLoadedMsg{brokerID: id, info: b, found: true}
			}
		}
		return brokerInfoLoadedMsg{brokerID: id, found: false}
	}
}

func (m *Model) loadTab(t tab) tea.Cmd {
	switch t {
	case tabConfigs:
		if m.configsLoaded {
			return nil
		}
		return m.loadConfigs()
	case tabMetrics:
		if m.metricsLoaded {
			return nil
		}
		return m.loadMetrics()
	default:
		if m.logDirsLoaded {
			return nil
		}
		return m.loadLogDirs()
	}
}

func (m *Model) loadLogDirs() tea.Cmd {
	ds := m.common.DataSource
	id := m.brokerID
	return func() tea.Msg {
		dirs, err := ds.GetBrokerLogDirs([]int32{id})
		if err != nil {
			return logDirsLoadedMsg{brokerID: id, err: err}
		}
		return logDirsLoadedMsg{brokerID: id, dirs: dirs[id]}
	}
}

func (m *Model) loadConfigs() tea.Cmd {
	ds := m.common.DataSource
	id := m.brokerID
	return func() tea.Msg {
		entries, err := ds.GetBrokerConfig(id)
		return configsLoadedMsg{brokerID: id, entries: entries, err: err}
	}
}

func (m *Model) loadMetrics() tea.Cmd {
	ds := m.common.DataSource
	id := m.brokerID
	return func() tea.Msg {
		data, err := ds.GetBrokerMetrics(id)
		return metricsLoadedMsg{brokerID: id, data: data, err: err}
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
	case brokerInfoLoadedMsg:
		if v.brokerID != m.brokerID {
			return nil
		}
		if v.err != nil {
			m.notFound = true
			return nil
		}
		m.notFound = !v.found
		if v.found {
			m.info = v.info
			m.infoLoaded = true
		}
		return nil
	case logDirsLoadedMsg:
		if v.brokerID == m.brokerID {
			m.logDirs = v.dirs
			m.logDirsErr = v.err
			m.logDirsLoaded = true
			m.rebuildLogTable()
		}
		return nil
	case configsLoadedMsg:
		if v.brokerID == m.brokerID {
			m.configs = v.entries
			m.configsErr = v.err
			m.configsLoaded = true
			m.rebuildConfigTable()
		}
		return nil
	case metricsLoadedMsg:
		if v.brokerID == m.brokerID {
			m.metricsLoaded = true
			m.metricsErr = v.err
			if v.err == nil {
				m.metricsViewer.SetContent(v.data)
			}
		}
		return nil
	case configAlteredMsg:
		return m.handleConfigAltered(v)
	case replicaMovedMsg:
		return m.handleReplicaMoved(v)
	case form.FormSubmitMsg:
		return m.handleMoveSubmit(v)
	case form.FormCancelMsg:
		m.moveForm = nil
		return nil
	case tea.KeyMsg:
		return m.handleKey(v)
	}
	// Forward other messages (mouse, viewport) to the active component.
	return m.forwardToActive(msg)
}

func (m *Model) forwardToActive(msg tea.Msg) tea.Cmd {
	switch m.active {
	case tabConfigs:
		var cmd tea.Cmd
		m.cfgTable, cmd = m.cfgTable.Update(msg)
		return cmd
	case tabMetrics:
		_, cmd := m.metricsViewer.Update(msg)
		return cmd
	default:
		var cmd tea.Cmd
		if m.expanded >= 0 {
			m.partTable, cmd = m.partTable.Update(msg)
		} else {
			m.logTable, cmd = m.logTable.Update(msg)
		}
		return cmd
	}
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Modal-ish sub-states first.
	if m.moveForm != nil {
		cmd, _ := m.moveForm.Update(msg)
		return cmd
	}
	if m.editing {
		return m.handleEditKey(msg)
	}
	if m.searching {
		return m.handleSearchKey(msg)
	}

	// Resolved through the single binding registry: tab cycles focus, 1..9
	// select a tab directly, r refreshes.
	if action, bound := keys.Default.Resolve(keys.ScopeListContent, msg.String()); bound {
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
		case keys.ActionSearch:
			if m.active == tabConfigs {
				m.searching = true
				m.searchInput.SetValue(m.cfgFilter)
				return m.searchInput.Focus()
			}
			return nil
		case keys.ActionEdit:
			if m.active == tabConfigs {
				return m.beginEdit()
			}
			return nil
		}
	}

	switch m.active {
	case tabConfigs:
		return m.handleConfigsKey(msg)
	case tabLogDirs:
		return m.handleLogDirsKey(msg)
	}
	return m.forwardToActive(msg)
}

func (m *Model) switchTab(t tab) tea.Cmd {
	m.active = t
	return m.loadTab(t)
}

func (m *Model) retry() tea.Cmd {
	if m.notFound {
		m.notFound = false
		return m.loadInfo()
	}
	switch m.active {
	case tabConfigs:
		m.configsLoaded = false
		return m.loadConfigs()
	case tabMetrics:
		m.metricsLoaded = false
		return m.loadMetrics()
	default:
		m.logDirsLoaded = false
		return m.loadLogDirs()
	}
}

// --- Log Dirs tab ---
