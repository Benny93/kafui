// Package ksql implements the ksqlDB UI: an overview page (page ID "ksql")
// listing the cluster's streams and tables in two tabs, and a query editor page
// (page ID "ksql_query") for executing statements and streaming SELECT results.
//
// Neither page self-registers. The router registers them under "ksql" and
// "ksql_query" (see pkg/ui/router/router.go); the overview is reached from the
// main page via the global 'K' key, gated on api.CapKsqlDB.
package ksql

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model is the ksqlDB overview page.
type Model struct {
	common      *core.Common
	keys        overviewKeys
	reusableApp *templateui.ReusableApp
	dims        core.Dimensions

	active tab

	streams       []api.KsqlStream
	streamsLoaded bool
	streamsErr    error

	tables       []api.KsqlTable
	tablesLoaded bool
	tablesErr    error

	tblTable    table.Model
	streamTable table.Model

	// sortCol/sortAsc drive the shared name→topic→formats→windowed sort. The
	// single Sort key advances the column and flips direction on wrap.
	sortCol int
	sortAsc bool
}

// NewModelWithCommon builds the ksqlDB overview page. The intended router page
// ID is "ksql".
func NewModelWithCommon(common *core.Common) core.Page {
	m := &Model{
		common:  common,
		keys:    defaultOverviewKeys(),
		active:  tabTables,
		sortAsc: true,
	}
	m.tblTable = table.New(table.WithColumns(tableColumns()), table.WithFocused(true), table.WithHeight(12))
	m.streamTable = table.New(table.WithColumns(streamColumns()), table.WithFocused(true), table.WithHeight(12))

	config := &providers.AppConfig{
		ContentProvider:      &overviewContentProvider{model: m},
		ShowSidebarByDefault: false,
	}
	m.reusableApp = templateui.NewReusableApp(config)
	m.reusableApp.SetKeyMap(overviewHelpKeyMap{keys: m.keys})
	return m
}

func streamColumns() []table.Column {
	return []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Topic", Width: 24},
		{Title: "Key Format", Width: 14},
		{Title: "Value Format", Width: 14},
	}
}

func tableColumns() []table.Column {
	return []table.Column{
		{Title: "Name", Width: 28},
		{Title: "Topic", Width: 22},
		{Title: "Key Format", Width: 12},
		{Title: "Value Format", Width: 12},
		{Title: "Windowed", Width: 10},
	}
}

// --- core.Page ---

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
	body := height - 10
	if body < 3 {
		body = 3
	}
	m.tblTable.SetWidth(width)
	m.tblTable.SetHeight(body)
	m.streamTable.SetWidth(width)
	m.streamTable.SetHeight(body)
	m.reusableApp.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

func (m *Model) GetID() string    { return "ksql" }
func (m *Model) GetTitle() string { return "ksqlDB" }

func (m *Model) GetHelp() []key.Binding {
	return []key.Binding{m.keys.NextTab, m.keys.Sort, m.keys.Query, m.keys.Seed, m.keys.Retry, m.keys.Back}
}

func (m *Model) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) { return m, nil }
func (m *Model) OnBlur() tea.Cmd                                   { return nil }

// OnFocus kicks off both listings in parallel.
func (m *Model) OnFocus() tea.Cmd {
	return tea.Batch(m.loadStreams(), m.loadTables())
}

// --- loads ---

func (m *Model) loadStreams() tea.Cmd {
	ds := m.common.DataSource
	return func() tea.Msg {
		s, err := ds.ListKsqlStreams()
		return streamsLoadedMsg{streams: s, err: err}
	}
}

func (m *Model) loadTables() tea.Cmd {
	ds := m.common.DataSource
	return func() tea.Msg {
		t, err := ds.ListKsqlTables()
		return tablesLoadedMsg{tables: t, err: err}
	}
}

// --- message handling ---

func (m *Model) handle(msg tea.Msg) tea.Cmd {
	switch v := msg.(type) {
	case streamsLoadedMsg:
		m.streams = v.streams
		m.streamsErr = v.err
		m.streamsLoaded = true
		m.rebuild()
		return nil
	case tablesLoadedMsg:
		m.tables = v.tables
		m.tablesErr = v.err
		m.tablesLoaded = true
		m.rebuild()
		return nil
	case tea.KeyMsg:
		return m.handleKey(v)
	}
	return m.forwardToActive(msg)
}

func (m *Model) forwardToActive(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	if m.active == tabStreams {
		m.streamTable, cmd = m.streamTable.Update(msg)
	} else {
		m.tblTable, cmd = m.tblTable.Update(msg)
	}
	return cmd
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.NextTab):
		if m.active == tabTables {
			m.active = tabStreams
		} else {
			m.active = tabTables
		}
		m.sortCol, m.sortAsc = 0, true
		m.rebuild()
		return nil
	case key.Matches(msg, m.keys.Sort):
		m.advanceSort()
		m.rebuild()
		return nil
	case key.Matches(msg, m.keys.Retry):
		m.streamsLoaded, m.tablesLoaded = false, false
		return tea.Batch(m.loadStreams(), m.loadTables())
	case key.Matches(msg, m.keys.Query):
		return core.NewPageChangeMsg("ksql_query", nil)
	case key.Matches(msg, m.keys.Seed):
		return m.seedQuery()
	}
	return m.forwardToActive(msg)
}

// advanceSort walks the sort column forward, flipping direction when it wraps.
func (m *Model) advanceSort() {
	cols := m.numColumns()
	m.sortCol++
	if m.sortCol >= cols {
		m.sortCol = 0
		m.sortAsc = !m.sortAsc
	}
}

func (m *Model) numColumns() int {
	if m.active == tabStreams {
		return len(streamColumns())
	}
	return len(tableColumns())
}

// seedQuery (KS-16 cross-feature integration) opens the query editor pre-seeded
// with a push query for the selected stream/table. The seed travels as the
// navigation "name" field; the router passes it to NewQueryModelWithSeed.
func (m *Model) seedQuery() tea.Cmd {
	name := m.selectedName()
	if name == "" {
		return nil
	}
	seed := fmt.Sprintf("SELECT * FROM %s EMIT CHANGES;", name)
	return core.NewPageChangeMsg("ksql_query", map[string]interface{}{"name": seed})
}

// selectedName returns the highlighted row's source name in the active tab.
func (m *Model) selectedName() string {
	if m.active == tabStreams {
		s := m.sortedStreams()
		i := m.streamTable.Cursor()
		if i >= 0 && i < len(s) {
			return s[i].Name
		}
		return ""
	}
	t := m.sortedTables()
	i := m.tblTable.Cursor()
	if i >= 0 && i < len(t) {
		return t[i].Name
	}
	return ""
}

// --- sorting (pure) ---

func (m *Model) sortedStreams() []api.KsqlStream {
	out := append([]api.KsqlStream(nil), m.streams...)
	key := func(s api.KsqlStream) string {
		switch m.sortCol {
		case 1:
			return s.Topic
		case 2:
			return s.KeyFormat
		case 3:
			return s.ValueFormat
		default:
			return s.Name
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if m.sortAsc {
			return key(out[i]) < key(out[j])
		}
		return key(out[i]) > key(out[j])
	})
	return out
}

func (m *Model) sortedTables() []api.KsqlTable {
	out := append([]api.KsqlTable(nil), m.tables...)
	key := func(t api.KsqlTable) string {
		switch m.sortCol {
		case 1:
			return t.Topic
		case 2:
			return t.KeyFormat
		case 3:
			return t.ValueFormat
		case 4:
			return strconv.FormatBool(t.Windowed)
		default:
			return t.Name
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if m.sortAsc {
			return key(out[i]) < key(out[j])
		}
		return key(out[i]) > key(out[j])
	})
	return out
}

// --- rendering ---

func (m *Model) rebuild() {
	srows := make([]table.Row, 0, len(m.streams))
	for _, s := range m.sortedStreams() {
		srows = append(srows, table.Row{s.Name, s.Topic, s.KeyFormat, s.ValueFormat})
	}
	m.streamTable.SetRows(srows)

	trows := make([]table.Row, 0, len(m.tables))
	for _, t := range m.sortedTables() {
		trows = append(trows, table.Row{t.Name, t.Topic, t.KeyFormat, t.ValueFormat, strconv.FormatBool(t.Windowed)})
	}
	m.tblTable.SetRows(trows)
}

func (m *Model) renderContent(width, height int) string {
	m.tblTable.SetWidth(width - 2) // -2 leaves room for the FrameTable border
	m.streamTable.SetWidth(width - 2)
	var b strings.Builder
	b.WriteString(m.header())
	b.WriteString("\n")
	b.WriteString(m.tabBar())
	b.WriteString("\n\n")

	if !m.streamsLoaded || !m.tablesLoaded {
		b.WriteString(m.common.Styles.Muted.Render("Loading streams and tables…"))
		return b.String()
	}

	// Not-configured friendly state (KS-11 empty state): the whole integration
	// is unavailable, not just an empty listing.
	if notConfigured(m.streamsErr) || notConfigured(m.tablesErr) {
		b.WriteString(m.common.Styles.Muted.Render("ksqlDB is not configured for this cluster."))
		return b.String()
	}

	if err := m.activeErr(); err != nil {
		b.WriteString(m.common.Styles.Error.Render("Error: " + err.Error()))
		b.WriteString("\n")
		b.WriteString(m.common.Styles.Muted.Render("Press r to retry both listings."))
		return b.String()
	}

	if m.active == tabStreams {
		if len(m.streams) == 0 {
			b.WriteString(m.common.Styles.Muted.Render("No streams."))
		} else {
			b.WriteString(stylesPkg.FrameTable(m.streamTable.View()))
		}
	} else {
		if len(m.tables) == 0 {
			b.WriteString(m.common.Styles.Muted.Render("No tables."))
		} else {
			b.WriteString(stylesPkg.FrameTable(m.tblTable.View()))
		}
	}
	b.WriteString("\n")
	b.WriteString(m.footer())
	return b.String()
}

func (m *Model) activeErr() error {
	if m.active == tabStreams {
		return m.streamsErr
	}
	return m.tablesErr
}

func (m *Model) header() string {
	s := m.common.Styles.StatusStyle
	return fmt.Sprintf("Streams: %s   Tables: %s",
		s.Info.Render(strconv.Itoa(len(m.streams))),
		s.Info.Render(strconv.Itoa(len(m.tables))),
	)
}

func (m *Model) tabBar() string {
	active := lipgloss.NewStyle().Foreground(stylesPkg.BgBase).Background(stylesPkg.Primary).Bold(true).Padding(0, 1)
	inactive := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted).Padding(0, 1)
	cell := func(t tab) string {
		if t == m.active {
			return active.Render(t.String())
		}
		return inactive.Render(t.String())
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cell(tabTables), cell(tabStreams))
}

func (m *Model) footer() string {
	dir := "asc"
	if !m.sortAsc {
		dir = "desc"
	}
	sortName := "Name"
	if cols := m.activeColumnTitles(); m.sortCol < len(cols) {
		sortName = cols[m.sortCol]
	}
	return m.common.Styles.Muted.Render(fmt.Sprintf(
		"tab: switch • s: sort (%s %s) • enter: query selected • e: query editor", sortName, dir))
}

func (m *Model) activeColumnTitles() []string {
	var cols []table.Column
	if m.active == tabStreams {
		cols = streamColumns()
	} else {
		cols = tableColumns()
	}
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.Title
	}
	return out
}

// notConfigured reports whether err is (or wraps) an api.KsqlNotConfiguredError.
func notConfigured(err error) bool {
	var e api.KsqlNotConfiguredError
	return errors.As(err, &e)
}
