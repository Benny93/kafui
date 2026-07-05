package schemadetail

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

// ─── Key bindings ────────────────────────────────────────────────────────────

// SchemaDetailKeyMap implements help.KeyMap for the schema detail page.
type SchemaDetailKeyMap struct {
	Versions      key.Binding
	Diff          key.Binding
	Register      key.Binding
	CheckCompat   key.Binding
	Compatibility key.Binding
	DeleteSubject key.Binding
	DeleteVersion key.Binding
	Topic         key.Binding
	Copy          key.Binding
	Back          key.Binding
	Quit          key.Binding
}

func NewSchemaDetailKeyMap() SchemaDetailKeyMap {
	return SchemaDetailKeyMap{
		Versions:      key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "versions")),
		Diff:          key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "diff")),
		Register:      key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "register version")),
		CheckCompat:   key.NewBinding(key.WithKeys("ctrl+k"), key.WithHelp("ctrl+k", "check compat")),
		Compatibility: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "compatibility")),
		DeleteSubject: key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "delete subject")),
		DeleteVersion: key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete version")),
		Topic:         key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "open topic")),
		Copy:          key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy")),
		Back:          key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
		Quit:          key.NewBinding(key.WithKeys("ctrl+c", "q"), key.WithHelp("q", "quit")),
	}
}

func (k SchemaDetailKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Versions, k.Diff, k.Register, k.Compatibility, k.Back, k.Quit}
}

func (k SchemaDetailKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Versions, k.Diff, k.Register, k.CheckCompat},
		{k.Compatibility, k.DeleteSubject, k.DeleteVersion, k.Topic},
		{k.Copy, k.Back, k.Quit},
	}
}

// ─── Content Provider ────────────────────────────────────────────────────────

// SchemaDetailContentProvider renders whichever sub-view the model is in.
type SchemaDetailContentProvider struct {
	model   *Model
	spinner spinner.Model
	width   int
	height  int
}

func NewSchemaDetailContentProvider(m *Model) *SchemaDetailContentProvider {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return &SchemaDetailContentProvider{model: m, spinner: sp}
}

func (p *SchemaDetailContentProvider) RenderContent(width, height int) string {
	p.width = width
	p.height = height
	m := p.model
	m.width = width
	m.height = height
	m.ClearExpiredStatus()

	if m.IsLoading() && m.mode == modeContent {
		return lipgloss.NewStyle().
			Foreground(stylesPkg.FgMuted).
			Render(fmt.Sprintf("\n  %s Loading schema…", p.spinner.View()))
	}

	var body string
	switch m.mode {
	case modeVersions:
		body = renderVersionList(m, width, height)
	case modeDiff:
		body = renderDiff(m, width, height)
	case modeRegister:
		body = renderRegister(m, width, height)
	case modePicker:
		body = renderPicker(m, width, height)
	default:
		m.viewer.SetDimensions(width, height-1)
		body = zone.Mark("schema-content", m.viewer.View())
	}

	if m.statusMsg != "" {
		statusStyle := lipgloss.NewStyle().Foreground(stylesPkg.Success).Italic(true)
		return body + "\n" + statusStyle.Render("📋 "+m.statusMsg)
	}
	return body
}

func (p *SchemaDetailContentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd {
	m := p.model
	if m == nil {
		return nil
	}
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.IsLoading() {
			var cmd tea.Cmd
			p.spinner, cmd = p.spinner.Update(msg)
			return cmd
		}
		return nil
	case tea.KeyMsg:
		switch m.mode {
		case modeVersions:
			return handleVersionsKey(m, msg)
		case modeDiff:
			return handleDiffKey(m, msg)
		case modeRegister:
			return handleRegisterKey(m, msg)
		case modePicker:
			return handlePickerKey(m, msg)
		default:
			return handleContentKey(m, msg)
		}
	}
	return nil
}

// handleContentKey handles keys while viewing the schema (default mode).
func handleContentKey(m *Model, msg tea.KeyMsg) tea.Cmd {
	// While the viewer's in-content search prompt is open, every keystroke must
	// reach the search field — do not intercept single-letter action hotkeys.
	if m.viewer.Searching() {
		_, cmd := m.viewer.Update(msg)
		return cmd
	}
	switch msg.String() {
	case "y":
		m.CopyToClipboard()
		return nil
	case "v":
		return m.enterVersions()
	case "d":
		return m.enterDiffFromContent()
	case "r":
		m.enterRegister()
		return nil
	case "c":
		m.enterPicker()
		return nil
	case "ctrl+k":
		m.enterRegister()
		return m.checkOnlyCmd()
	case "D":
		return m.confirmDeleteSubjectCmd()
	default:
		// Delegate scrolling / in-content search to the viewer.
		_, cmd := m.viewer.Update(msg)
		return cmd
	}
}

func (p *SchemaDetailContentProvider) InitContent() tea.Cmd {
	return p.spinner.Tick
}

// IsInputMode reports true when the page owns a text/selection sub-view so the
// framework must not intercept keystrokes as app-level hotkeys.
func (p *SchemaDetailContentProvider) IsInputMode() bool {
	return p.model.mode == modeRegister || p.model.mode == modePicker ||
		p.model.mode == modeVersions || p.model.mode == modeDiff
}

func (p *SchemaDetailContentProvider) GetContentSize(width int) int {
	if p.model.content == "" {
		return 5
	}
	return strings.Count(p.model.content, "\n") + 5
}

// ─── Header Provider ─────────────────────────────────────────────────────────

type SchemaDetailHeaderProvider struct {
	model *Model
}

func NewSchemaDetailHeaderProvider(m *Model) *SchemaDetailHeaderProvider {
	return &SchemaDetailHeaderProvider{model: m}
}

func (h *SchemaDetailHeaderProvider) GetBrandName() string { return "Kafui™" }
func (h *SchemaDetailHeaderProvider) GetAppName() string   { return "Schema Detail" }

func (h *SchemaDetailHeaderProvider) GetStatusData() map[string]interface{} {
	ctx := ""
	if h.model.dataSource != nil {
		ctx = h.model.dataSource.GetContext()
	}
	return map[string]interface{}{
		"context": ctx,
		"status":  "schema-registry",
	}
}

func (h *SchemaDetailHeaderProvider) HandleHeaderUpdate(msg tea.Msg) tea.Cmd { return nil }
func (h *SchemaDetailHeaderProvider) InitHeader() tea.Cmd                    { return nil }

// ─── Sidebar Section ─────────────────────────────────────────────────────────

type SchemaMetadataSidebarSection struct {
	model *Model
}

func NewSchemaMetadataSidebarSection(m *Model) *SchemaMetadataSidebarSection {
	return &SchemaMetadataSidebarSection{model: m}
}

func (s *SchemaMetadataSidebarSection) GetTitle() string { return "Schema Info" }

func (s *SchemaMetadataSidebarSection) RenderItems(maxItems, width int) []providers.SidebarItem {
	m := s.model
	versionStr := "latest"
	if m.GetVersion() > 0 {
		versionStr = strconv.Itoa(m.GetVersion())
	}
	idStr := "–"
	if m.GetSchemaID() > 0 {
		idStr = strconv.Itoa(m.GetSchemaID())
	}

	items := []providers.SidebarItem{
		{Text: "Subject", Value: m.GetSubject()},
		{Text: "Version", Value: versionStr},
		{Text: "ID", Value: idStr},
		{Text: "Type", Value: m.GetSchemaType()},
	}

	// Effective compatibility level (SR-13): annotate "(global)" when inherited.
	if level, specific, loaded := m.EffectiveCompatibility(); loaded {
		val := string(level)
		if !specific {
			val += " (global)"
		}
		items = append(items, providers.SidebarItem{Text: "Compatibility", Value: val})
	}

	// Associated topic (SR-14): show only when a topic matched.
	if m.AssociatedTopic() != "" {
		items = append(items, providers.SidebarItem{
			Text: "Topic", Value: m.AssociatedTopic() + " (t)", Status: "info",
		})
	}

	if !m.IsLoading() && m.GetContent() != "" {
		items = append(items, providers.SidebarItem{
			Text:  "Size",
			Value: fmt.Sprintf("%d bytes", len(m.GetContent())),
		})
	}
	return items
}

func (s *SchemaMetadataSidebarSection) HandleSectionUpdate(msg tea.Msg) tea.Cmd { return nil }
func (s *SchemaMetadataSidebarSection) InitSection() tea.Cmd                    { return nil }
func (s *SchemaMetadataSidebarSection) RefreshSection() tea.Cmd                 { return nil }

// ─── Shared helpers ────────────────────────────────────────────────────────────

// prettyJSON attempts to unmarshal and re-marshal s with indentation.
// Returns s unchanged when s is not valid JSON.
func prettyJSON(s string) string {
	if s == "" {
		return s
	}
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return s
	}
	return string(b)
}

// isJSONType reports whether a schema type is JSON-based (AVRO and JSON schemas
// are JSON documents; PROTOBUF is IDL text).
func isJSONType(schemaType string) bool {
	switch strings.ToUpper(schemaType) {
	case "", "AVRO", "JSON":
		return true
	default:
		return false
	}
}

// prettySchema pretty-prints JSON-based schemas and returns PROTOBUF verbatim.
func prettySchema(text, schemaType string) string {
	if isJSONType(schemaType) {
		return prettyJSON(text)
	}
	return text
}

// friendlySchemaError maps schema-registry errors to reader-friendly text shown
// in the content pane (SR-22).
func friendlySchemaError(err error) string {
	var notConfigured api.SchemaRegistryNotConfiguredError
	if errors.As(err, &notConfigured) {
		return "No schema registry is configured for this cluster."
	}
	var subjectNotFound api.SubjectNotFoundError
	if errors.As(err, &subjectNotFound) {
		return fmt.Sprintf("Subject %q was not found in the registry.", subjectNotFound.Subject)
	}
	var versionNotFound api.SchemaVersionNotFoundError
	if errors.As(err, &versionNotFound) {
		return fmt.Sprintf("Version %d of %q was not found.", versionNotFound.Version, versionNotFound.Subject)
	}
	return fmt.Sprintf("Error loading schema: %v", err)
}

// resolveTopic strips the value/key suffixes from a subject and matches the
// remainder against the cluster's topic names (SR-14). Returns "" on no match.
func resolveTopic(subject string, ds api.KafkaDataSource) string {
	// ponytail: the suffix set is fixed to the TopicNameStrategy defaults;
	// making it configurable via core.Common.Config is deferred.
	suffixes := []string{"-value", "-key"}
	var candidate string
	for _, suffix := range suffixes {
		if strings.HasSuffix(subject, suffix) {
			candidate = strings.TrimSuffix(subject, suffix)
			break
		}
	}
	if candidate == "" {
		return ""
	}
	names, err := ds.GetTopicNames()
	if err != nil {
		return ""
	}
	for _, n := range names {
		if n == candidate {
			return candidate
		}
	}
	return ""
}
