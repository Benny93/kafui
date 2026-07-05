// Package consumergroup implements the consumer-group detail page (dynamic page
// ID "consumer_group:<groupID>"). It renders a summary strip plus a topic-grouped
// partition table over the shared template shell, with expand/collapse, topic
// filtering, sorting, refresh + auto-refresh, lag trend indicators, CSV export,
// and destructive operations (delete group, delete a topic's offsets, reset
// offsets). The page is created by the router; see NewModelWithCommon.
package consumergroup

import (
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Model is the consumer-group detail page.
type Model struct {
	common      *core.Common
	keys        pageKeys
	reusableApp *templateui.ReusableApp
	dims        core.Dimensions

	groupID  string
	detail   api.ConsumerGroupDetail
	loaded   bool
	notFound bool
	loadErr  error

	// Topic-grouped view.
	topicRows  []topicRow
	topicTable table.Model
	partTable  table.Model
	expanded   int // index into topicRows of the expanded topic, -1 when none

	// Topic filter.
	topicFilter string
	searching   bool
	searchInput textinput.Model

	// Topic-column sort.
	sortCol  int
	sortDesc bool

	// Refresh / auto-refresh (CG-15).
	autoInterval time.Duration // 0 = off
	// trendBaseline holds the previous per-topic aggregate lag captured at the
	// last auto-refresh tick; nil means no baseline (no trend arrows shown).
	trendBaseline map[string]int64

	// Reset-offsets form (CG-19/20).
	resetForm *resetForm
}

// NewModelWithCommon builds the consumer-group detail page for the given group.
// The router wires this to the "consumer_group:<groupID>" dynamic page ID.
func NewModelWithCommon(common *core.Common, groupID string) core.Page {
	m := &Model{
		common:   common,
		keys:     defaultKeys(),
		groupID:  groupID,
		expanded: -1,
	}
	if common != nil && common.Config != nil {
		m.autoInterval = common.Config.ConsumerGroupRefreshInterval
	}

	si := textinput.New()
	si.Prompt = "/"
	m.searchInput = si

	m.topicTable = table.New(table.WithColumns(topicColumns()), table.WithFocused(true), table.WithHeight(10))
	m.partTable = table.New(table.WithColumns(partitionColumns()), table.WithFocused(true), table.WithHeight(8))

	config := &providers.AppConfig{
		ContentProvider:      &contentProvider{model: m},
		ShowSidebarByDefault: false,
	}
	m.reusableApp = templateui.NewReusableApp(config)
	m.reusableApp.SetKeyMap(helpKeyMap{keys: m.keys})
	return m
}

func topicColumns() []table.Column {
	return []table.Column{
		{Title: "Topic", Width: 40},
		{Title: "Partitions", Width: 12},
		{Title: "Lag", Width: 14},
		{Title: "Trend", Width: 8},
	}
}

func partitionColumns() []table.Column {
	return []table.Column{
		{Title: "Partition", Width: 10},
		{Title: "Consumer", Width: 22},
		{Title: "Host", Width: 16},
		{Title: "Committed", Width: 12},
		{Title: "End", Width: 12},
		{Title: "Lag", Width: 10},
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
	body := height - 12
	if body < 3 {
		body = 3
	}
	m.topicTable.SetWidth(width)
	m.topicTable.SetHeight(body)
	m.partTable.SetWidth(width)
	m.partTable.SetHeight(body / 2)
	if m.resetForm != nil {
		m.resetForm.SetDimensions(width, height)
	}
	m.reusableApp.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

func (m *Model) GetID() string    { return "consumer_group:" + m.groupID }
func (m *Model) GetTitle() string { return "Group " + m.groupID }

func (m *Model) GetHelp() []key.Binding {
	return []key.Binding{
		m.keys.Expand, m.keys.Filter, m.keys.Sort, m.keys.Refresh, m.keys.AutoRefresh,
		m.keys.GotoTopic, m.keys.Reset, m.keys.DeleteOff, m.keys.Delete, m.keys.Export, m.keys.Back,
	}
}

func (m *Model) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) { return m, nil }
func (m *Model) OnBlur() tea.Cmd                                   { return nil }

// OnFocus kicks off the initial detail load (and re-arms auto-refresh if it was
// previously enabled and persisted).
func (m *Model) OnFocus() tea.Cmd {
	cmds := []tea.Cmd{m.loadDetail()}
	if m.autoInterval > 0 {
		cmds = append(cmds, m.scheduleTick())
	}
	return tea.Batch(cmds...)
}

// --- loads ---

func (m *Model) loadDetail() tea.Cmd {
	ds := m.common.DataSource
	id := m.groupID
	return func() tea.Msg {
		detail, err := ds.GetConsumerGroupDetail(id)
		if err != nil {
			var nf api.GroupNotFoundError
			if asGroupNotFound(err, &nf) {
				return detailLoadedMsg{groupID: id, notFound: true}
			}
			return detailLoadedMsg{groupID: id, err: err}
		}
		return detailLoadedMsg{groupID: id, detail: detail}
	}
}

// --- message handling (via the content provider) ---

func (m *Model) handle(msg tea.Msg) tea.Cmd {
	switch v := msg.(type) {
	case detailLoadedMsg:
		return m.handleDetailLoaded(v)
	case autoRefreshTickMsg:
		return m.handleAutoTick(v)
	case groupDeletedMsg:
		return m.handleGroupDeleted(v)
	case offsetsDeletedMsg:
		return m.handleOffsetsDeleted(v)
	case offsetsResetMsg:
		return m.handleOffsetsReset(v)
	case resetFormSubmitMsg:
		return m.handleResetSubmit(v)
	case resetFormCancelMsg:
		m.resetForm = nil
		return nil
	case tea.KeyMsg:
		return m.handleKey(v)
	}
	return m.forwardToActive(msg)
}

func (m *Model) handleDetailLoaded(v detailLoadedMsg) tea.Cmd {
	if v.groupID != m.groupID {
		return nil
	}
	m.loaded = true
	if v.notFound {
		m.notFound = true
		return nil
	}
	if v.err != nil {
		m.loadErr = v.err
		return nil
	}
	m.notFound = false
	m.loadErr = nil
	m.rebuildTopicRows(v.detail)
	m.detail = v.detail
	return nil
}

func (m *Model) forwardToActive(msg tea.Msg) tea.Cmd {
	if m.resetForm != nil {
		cmd, _ := m.resetForm.Update(msg)
		return cmd
	}
	var cmd tea.Cmd
	if m.expanded >= 0 {
		m.partTable, cmd = m.partTable.Update(msg)
	} else {
		m.topicTable, cmd = m.topicTable.Update(msg)
	}
	return cmd
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Sub-states first.
	if m.resetForm != nil {
		cmd, _ := m.resetForm.Update(msg)
		return cmd
	}
	if m.searching {
		return m.handleSearchKey(msg)
	}
	if m.notFound {
		if key.Matches(msg, m.keys.Retry) {
			m.notFound = false
			return m.loadDetail()
		}
		return nil
	}

	switch {
	case key.Matches(msg, m.keys.Expand):
		return m.toggleExpand()
	case key.Matches(msg, m.keys.Filter):
		m.searching = true
		m.searchInput.SetValue(m.topicFilter)
		return m.searchInput.Focus()
	case key.Matches(msg, m.keys.Sort):
		m.cycleSort()
		return nil
	case key.Matches(msg, m.keys.Refresh):
		return m.loadDetail()
	case key.Matches(msg, m.keys.AutoRefresh):
		return m.cycleAutoRefresh()
	case key.Matches(msg, m.keys.GotoTopic):
		return m.gotoSelectedTopic()
	case key.Matches(msg, m.keys.Reset):
		return m.openResetForm()
	case key.Matches(msg, m.keys.DeleteOff):
		return m.deleteSelectedTopicOffsets()
	case key.Matches(msg, m.keys.Delete):
		return m.deleteGroup()
	case key.Matches(msg, m.keys.Export):
		return m.exportCSV()
	}
	return m.forwardToActive(msg)
}

func (m *Model) handleSearchKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		m.searching = false
		m.topicFilter = m.searchInput.Value()
		m.searchInput.Blur()
		m.rebuildTopicRows(m.detail)
		return nil
	case "esc":
		m.searching = false
		m.searchInput.Blur()
		return nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return cmd
}

func (m *Model) toggleExpand() tea.Cmd {
	if m.expanded >= 0 {
		m.expanded = -1
		return nil
	}
	i := m.topicTable.Cursor()
	if i >= 0 && i < len(m.topicRows) {
		m.expanded = i
		m.rebuildPartTable()
	}
	return nil
}

func (m *Model) gotoSelectedTopic() tea.Cmd {
	tr, ok := m.selectedTopicRow()
	if !ok {
		return nil
	}
	topic := tr.topic
	return func() tea.Msg {
		return core.PageChangeMsg{PageID: "topic:" + topic, Data: map[string]interface{}{"name": topic}}
	}
}

func (m *Model) selectedTopicRow() (topicRow, bool) {
	i := m.topicTable.Cursor()
	if i < 0 || i >= len(m.topicRows) {
		return topicRow{}, false
	}
	return m.topicRows[i], true
}

// --- rendering ---

func (m *Model) render(width, height int) string {
	m.topicTable.SetWidth(width - 2) // -2 leaves room for the FrameTable border
	m.partTable.SetWidth(width - 2)
	var b strings.Builder

	if m.resetForm != nil {
		b.WriteString(m.common.Styles.Header.Render("Reset offsets — " + m.groupID))
		b.WriteString("\n\n")
		b.WriteString(m.resetForm.View())
		return b.String()
	}

	if !m.loaded {
		return m.common.Styles.Muted.Render("Loading consumer group…")
	}
	if m.notFound {
		return m.common.Styles.Error.Render(fmt.Sprintf("Consumer group %q not found.", m.groupID)) +
			"\n" + m.common.Styles.Muted.Render("Press esc to go back, r to retry.")
	}
	if m.loadErr != nil {
		return m.common.Styles.Error.Render("Error: "+m.loadErr.Error()) +
			"\n" + m.common.Styles.Muted.Render("Press r to retry.")
	}

	b.WriteString(m.summaryStrip())
	b.WriteString("\n\n")

	if len(m.topicRows) == 0 {
		b.WriteString(m.common.Styles.Muted.Render("This group has no associated topics or committed offsets."))
		return b.String()
	}

	b.WriteString(stylesPkg.FrameTable(m.topicTable.View()))
	if m.expanded >= 0 && m.expanded < len(m.topicRows) {
		b.WriteString("\n\n")
		b.WriteString(m.common.Styles.Header.Render("Partitions of " + m.topicRows[m.expanded].topic))
		b.WriteString("\n")
		b.WriteString(stylesPkg.FrameTable(m.partTable.View()))
		b.WriteString("\n")
		b.WriteString(m.common.Styles.Muted.Render("enter/esc: collapse • d: delete offsets • t: go to topic"))
	} else {
		b.WriteString("\n")
		b.WriteString(m.footerHint())
	}
	if m.searching {
		b.WriteString("\n")
		b.WriteString(m.searchInput.View())
	}
	return b.String()
}

func (m *Model) footerHint() string {
	auto := "off"
	if m.autoInterval > 0 {
		auto = m.autoInterval.String()
	}
	return m.common.Styles.Muted.Render(fmt.Sprintf(
		"enter: expand • /: filter • s: sort • r: refresh • a: auto-refresh (%s) • R: reset • ctrl+d: delete", auto))
}

func (m *Model) summaryStrip() string {
	assignedTopics := len(m.topicRows)
	assignedParts := 0
	for _, tr := range m.topicRows {
		assignedParts += len(tr.partitions)
	}
	lag := formatLagPtr(groupTotalLag(m.detail))
	parts := []string{
		m.common.Styles.Header.Render(m.groupID),
		"State: " + groupStateStyled(m.common, m.detail.State),
		fmt.Sprintf("Members: %d", len(m.detail.Members)),
		fmt.Sprintf("Topics: %d", assignedTopics),
		fmt.Sprintf("Partitions: %d", assignedParts),
		fmt.Sprintf("Coordinator: %s", coordString(m.detail.CoordinatorID)),
		"Lag: " + lag,
	}
	strip := strings.Join(parts, "   ")
	return strip + "\n" + m.common.Styles.Muted.Render(stateExplanation(m.detail.State))
}

func coordString(id int32) string {
	if id < 0 {
		return "—"
	}
	return fmt.Sprintf("%d", id)
}

// stateExplanation is the TUI adaptation of the spec's per-state tooltip.
func stateExplanation(state string) string {
	switch state {
	case api.GroupStateStable:
		return "Stable: all members have joined and partitions are assigned."
	case api.GroupStatePreparingRebalance:
		return "PreparingRebalance: members are (re)joining; assignments are being revoked."
	case api.GroupStateCompletingRebalance:
		return "CompletingRebalance: the leader is assigning partitions to members."
	case api.GroupStateEmpty:
		return "Empty: no active members; offsets may still be committed."
	case api.GroupStateDead:
		return "Dead: the group has no members and no committed offsets."
	default:
		return "Unknown: the group's state could not be determined."
	}
}
