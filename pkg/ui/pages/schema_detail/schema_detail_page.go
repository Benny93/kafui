package schemadetail

import (
	"fmt"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components/editor"
	"github.com/Benny93/kafui/pkg/ui/core"
	mainpage "github.com/Benny93/kafui/pkg/ui/pages/main"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// ponytail: deferred, tracked cross-page schema actions —
//   - SR-15: sortable columns on the main-page schemas table (subject/id/type/
//     compat) are not implemented; the table shows natural registry order.
//   - SR-16: the main-page "new subject" entry point (ctrl+n) is deferred;
//     registering a NEW VERSION of an existing subject works here via 'r'.
//   - SR-19: the main-page GLOBAL compatibility editor (ctrl+g) is deferred;
//     per-subject compatibility editing works here via 'c'.
// The datasource fully supports all of the above (RegisterSchema /
// SetGlobalCompatibility / sortable data), so these are UI-entry-point-only gaps.

// viewMode selects which sub-view the detail page shows.
type viewMode int

const (
	modeContent  viewMode = iota // read-only syntax-highlighted schema (default)
	modeVersions                 // version list (SR-11)
	modeDiff                     // side-by-side version diff (SR-12)
	modeRegister                 // editable new-version editor (SR-16)
	modePicker                   // compatibility-level picker (SR-19)
)

// ─── Async messages ──────────────────────────────────────────────────────────

// SchemaContentLoadedMsg is sent when the async schema fetch completes.
type SchemaContentLoadedMsg struct {
	Version int
	Content string
	Err     error
}

// SchemaVersionsLoadedMsg carries the version list of the current subject (SR-11).
type SchemaVersionsLoadedMsg struct {
	Versions []api.SchemaVersion
	Err      error
}

// SchemaMetaLoadedMsg carries the effective compatibility level and associated
// topic resolved for the sidebar (SR-13/SR-14).
type SchemaMetaLoadedMsg struct {
	Level    api.CompatibilityLevel
	Specific bool
	Topic    string
}

// schemaVersionContentMsg carries one version's text for the diff view (SR-12).
type schemaVersionContentMsg struct {
	Version int
	Content string
	Err     error
}

// SchemaCheckResultMsg is the result of a standalone compatibility check (SR-17).
type SchemaCheckResultMsg struct {
	Compatible bool
	Messages   []string
	Err        error
}

// SchemaRegisterResultMsg is the result of a register-schema attempt (SR-16).
type SchemaRegisterResultMsg struct {
	Schema       api.Schema
	Incompatible bool
	Messages     []string
	Err          error
}

// SchemaDeleteResultMsg is the result of a delete (subject or version) (SR-18).
type SchemaDeleteResultMsg struct {
	Err        error
	BackToList bool // true → navigate back to the schemas list (subject deleted)
}

// SchemaCompatSetResultMsg is the result of a compatibility-level update (SR-19).
type SchemaCompatSetResultMsg struct {
	Level api.CompatibilityLevel
	Err   error
}

// Model holds the state for the schema detail page.
type Model struct {
	common     *core.Common
	dataSource api.KafkaDataSource

	// Schema identity — populated from the SchemaResourceItem selected on the main page.
	subject    string
	version    int
	schemaID   int
	schemaType string

	// Loaded content
	content   string
	loading   bool
	loadedAt  time.Time
	statusMsg string
	statusAt  time.Time

	// View state
	mode          viewMode
	spinner       spinner.Model
	width, height int
	viewer        *editor.Viewer

	// Versions (SR-11)
	versions       []api.SchemaVersion
	versionsLoaded bool
	versionCursor  int

	// Diff (SR-12)
	diffView     *editor.DiffView
	diffLeft     int // left pane version number
	diffRight    int // right pane version number
	diffActive   int // 0 = left, 1 = right
	contentCache map[int]string

	// Register (SR-16)
	editor       *editor.Editor
	registerSeed string

	// Compatibility (SR-13/SR-19)
	compat         api.CompatibilityLevel
	compatSpecific bool
	compatLoaded   bool
	pickerCursor   int

	// Associated topic (SR-14)
	topic string
}

// NewModel creates the inner model.
func NewModel(ds api.KafkaDataSource, item *mainpage.SchemaResourceItem) *Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	m := &Model{
		dataSource:   ds,
		subject:      item.Subject(),
		version:      item.Version(),
		schemaID:     item.SchemaID(),
		schemaType:   item.SchemaType(),
		loading:      true,
		spinner:      sp,
		viewer:       editor.NewViewer(""),
		contentCache: map[int]string{},
	}
	return m
}

// ─── Async command constructors ───────────────────────────────────────────────

// LoadContentAsync fetches the current version's schema content.
func (m *Model) LoadContentAsync() tea.Cmd {
	subject, version, ds := m.subject, m.version, m.dataSource
	return func() tea.Msg {
		content, err := ds.GetSchemaContent(subject, version)
		return SchemaContentLoadedMsg{Version: version, Content: content, Err: err}
	}
}

// loadVersionsCmd lists all versions of the subject (SR-11).
func (m *Model) loadVersionsCmd() tea.Cmd {
	subject, ds := m.subject, m.dataSource
	return func() tea.Msg {
		versions, err := ds.GetSchemaVersions(subject)
		return SchemaVersionsLoadedMsg{Versions: versions, Err: err}
	}
}

// loadMetaCmd resolves effective compatibility + associated topic (SR-13/SR-14).
func (m *Model) loadMetaCmd() tea.Cmd {
	subject, ds := m.subject, m.dataSource
	return func() tea.Msg {
		level, specific, err := ds.GetSubjectCompatibility(subject)
		if err != nil {
			level, specific = "", false
		}
		return SchemaMetaLoadedMsg{Level: level, Specific: specific, Topic: resolveTopic(subject, ds)}
	}
}

// loadVersionContentCmd fetches one version's text for the diff view (SR-12).
func (m *Model) loadVersionContentCmd(version int) tea.Cmd {
	subject, ds := m.subject, m.dataSource
	return func() tea.Msg {
		content, err := ds.GetSchemaContent(subject, version)
		return schemaVersionContentMsg{Version: version, Content: content, Err: err}
	}
}

// ─── Simple accessors ──────────────────────────────────────────────────────────

func (m *Model) GetContent() string { return m.content }
func (m *Model) IsLoading() bool    { return m.loading }
func (m *Model) GetSubject() string { return m.subject }
func (m *Model) GetVersion() int    { return m.version }
func (m *Model) GetSchemaID() int   { return m.schemaID }
func (m *Model) GetMode() viewMode  { return m.mode }
func (m *Model) GetSchemaType() string {
	if m.schemaType == "" {
		return "AVRO"
	}
	return m.schemaType
}

// EffectiveCompatibility returns the resolved level and whether it is
// subject-specific (false → inherited from the global level).
func (m *Model) EffectiveCompatibility() (api.CompatibilityLevel, bool, bool) {
	return m.compat, m.compatSpecific, m.compatLoaded
}

// AssociatedTopic returns the matched topic name ("" when none).
func (m *Model) AssociatedTopic() string { return m.topic }

// Versions returns the loaded version list (may be empty until loaded).
func (m *Model) Versions() []api.SchemaVersion { return m.versions }

func (m *Model) SetStatus(msg string) {
	m.statusMsg = msg
	m.statusAt = time.Now()
}

func (m *Model) ClearExpiredStatus() {
	if m.statusMsg != "" && time.Since(m.statusAt) > 3*time.Second {
		m.statusMsg = ""
	}
}

func (m *Model) CopyToClipboard() {
	if m.content == "" {
		m.SetStatus("Nothing to copy")
		return
	}
	if err := clipboard.WriteAll(m.content); err != nil {
		m.SetStatus("Failed to copy: " + err.Error())
		return
	}
	m.SetStatus("Schema copied to clipboard")
}

// setContent updates the current schema text and refreshes the viewer.
func (m *Model) setContent(content string) {
	m.content = content
	pretty := prettySchema(content, m.GetSchemaType())
	m.viewer.SetContent(pretty)
	m.viewer.SetHighlight(isJSONType(m.GetSchemaType()))
}

// ─── Page Model ─────────────────────────────────────────────────────────────

// SchemaDetailPageModel wraps Model and implements core.Page via ReusableApp.
type SchemaDetailPageModel struct {
	common          *core.Common
	model           *Model
	reusableApp     *templateui.ReusableApp
	contentProvider *SchemaDetailContentProvider
}

// NewSchemaDetailPageModel creates a page with a ReusableApp layout.
func NewSchemaDetailPageModel(common *core.Common, item *mainpage.SchemaResourceItem) *SchemaDetailPageModel {
	m := NewModel(common.DataSource, item)
	m.common = common

	contentProvider := NewSchemaDetailContentProvider(m)
	km := NewSchemaDetailKeyMap()

	config := &providers.AppConfig{
		ContentProvider:             contentProvider,
		HeaderDataProvider:          NewSchemaDetailHeaderProvider(m),
		SidebarSections:             []providers.SidebarSection{NewSchemaMetadataSidebarSection(m)},
		ShowSidebarByDefault:        true,
		CompactModeWidthBreakpoint:  100,
		CompactModeHeightBreakpoint: 30,
	}

	app := templateui.NewReusableApp(config)
	app.SetKeyMap(km)

	return &SchemaDetailPageModel{
		common:          common,
		model:           m,
		reusableApp:     app,
		contentProvider: contentProvider,
	}
}

func (p *SchemaDetailPageModel) Init() tea.Cmd {
	return tea.Batch(p.reusableApp.Init(), p.model.LoadContentAsync(), p.model.loadMetaCmd())
}

func (p *SchemaDetailPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m := p.model
	switch msg := msg.(type) {
	case SchemaContentLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			m.setContent(friendlySchemaError(msg.Err))
		} else {
			m.version = msg.Version
			m.setContent(msg.Content)
			m.loadedAt = time.Now()
			if msg.Version > 0 {
				m.contentCache[msg.Version] = msg.Content
			}
		}
		return p, nil

	case SchemaVersionsLoadedMsg:
		if msg.Err != nil {
			return p, core.NotifyError("Schema versions", msg.Err)
		}
		m.versions = msg.Versions
		m.versionsLoaded = true
		// Default the cursor to the newest version.
		m.versionCursor = 0
		return p, nil

	case SchemaMetaLoadedMsg:
		m.compat = msg.Level
		m.compatSpecific = msg.Specific
		m.compatLoaded = msg.Level != ""
		m.topic = msg.Topic
		return p, nil

	case schemaVersionContentMsg:
		if msg.Err == nil {
			m.contentCache[msg.Version] = msg.Content
			m.refreshDiff()
		}
		return p, nil

	case SchemaCheckResultMsg:
		return p, checkResultCmd(msg)

	case SchemaRegisterResultMsg:
		return p, p.handleRegisterResult(msg)

	case SchemaDeleteResultMsg:
		return p, p.handleDeleteResult(msg)

	case SchemaCompatSetResultMsg:
		if msg.Err != nil {
			return p, core.NotifyError("Set compatibility", msg.Err)
		}
		m.mode = modeContent
		return p, tea.Batch(
			m.loadMetaCmd(),
			core.NewNotification(core.StatusSuccess, "Compatibility", "Level set to "+string(msg.Level)),
		)

	case tea.KeyMsg:
		// 't' navigates to the associated topic (SR-14). Handle it at the page
		// level so it overrides the framework's sidebar-toggle default, but only
		// while viewing content and when a topic actually matched.
		if msg.String() == "t" && m.mode == modeContent && m.topic != "" {
			return p, core.NewPageChangeMsg("topic:"+m.topic, map[string]interface{}{"name": m.topic})
		}
	}
	app, cmd := p.reusableApp.Update(msg)
	p.reusableApp = app.(*templateui.ReusableApp)
	return p, cmd
}

func (p *SchemaDetailPageModel) View() string {
	return p.reusableApp.View()
}

func (p *SchemaDetailPageModel) SetDimensions(width, height int) {
	p.reusableApp.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

func (p *SchemaDetailPageModel) GetID() string {
	return fmt.Sprintf("schema_detail:%s", p.model.subject)
}

func (p *SchemaDetailPageModel) GetTitle() string {
	return fmt.Sprintf("Schema: %s", p.model.subject)
}

func (p *SchemaDetailPageModel) GetHelp() []key.Binding {
	km := NewSchemaDetailKeyMap()
	return []key.Binding{
		km.Versions, km.Diff, km.Register, km.CheckCompat,
		km.Compatibility, km.DeleteSubject, km.DeleteVersion,
		km.Topic, km.Copy, km.Back, km.Quit,
	}
}

func (p *SchemaDetailPageModel) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
	return p, nil
}

func (p *SchemaDetailPageModel) OnFocus() tea.Cmd { return nil }
func (p *SchemaDetailPageModel) OnBlur() tea.Cmd  { return nil }
