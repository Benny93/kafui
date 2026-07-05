package appconfig_view

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/Benny93/kafui/pkg/version"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// pageID is the intended router page ID. Registration is done in the router
// (pkg/ui/router/router.go), not here.
const pageID = "appconfig"

// Model is the read-only Application Config page. It renders the effective
// merged configuration as a scrollable, structured document with all secrets
// redacted.
type Model struct {
	common      *core.Common
	dimensions  core.Dimensions
	reusableApp *templateui.ReusableApp
}

// NewModelWithCommon creates the Application Config page from the shared context.
func NewModelWithCommon(common *core.Common) *Model {
	m := &Model{common: common}

	contentProvider := newContentProvider(buildDocument(common))

	config := &providers.AppConfig{
		ContentProvider:      contentProvider,
		ShowSidebarByDefault: false,
	}
	m.reusableApp = templateui.NewReusableApp(config)
	m.reusableApp.SetKeyMap(keys.DefaultKeyMap().Detail)

	return m
}

// buildDocument renders the whole effective configuration to a styled string.
func buildDocument(common *core.Common) string {
	s := common.Styles
	var b strings.Builder

	section := func(title string) {
		b.WriteString("\n")
		b.WriteString(s.Header.Render(title))
		b.WriteString("\n")
	}
	kv := func(label, value string) {
		b.WriteString(s.Muted.Render(fmt.Sprintf("  %-22s", label)))
		b.WriteString(value)
		b.WriteString("\n")
	}

	// Build info
	info := version.Get()
	section("Build")
	kv("Version", info.Version)
	kv("Commit", info.Commit)
	kv("Build Time", info.BuildTime)
	kv("Go Version", info.GoVersion)
	kv("Platform", info.Platform)

	// App settings
	cfg := common.AppConfig
	if cfg != nil {
		section("Application Settings")
		kv("Dynamic Config", fmt.Sprintf("%t", cfg.DynamicConfigEnabled))
		kv("Refresh Interval", cfg.RefreshInterval.String())
		kv("Release Check", fmt.Sprintf("%t", cfg.ReleaseCheck.Enabled))
		kv("UI Theme", cfg.UI.Theme)
		kv("UI Show Sidebar", fmt.Sprintf("%t", cfg.UI.ShowSidebar))
		kv("UI Compact Mode", fmt.Sprintf("%t", cfg.UI.CompactMode))
		kv("UI Timezone", cfg.UI.Timezone)
	}

	// Effective permissions ("whoami") — the resolved authorization posture for
	// the active cluster (AA-12). Only shown when authorization is enabled: with
	// authz off (the default single-user mode) it would just echo the OS
	// username and "all permitted", which adds no information and is PII we
	// shouldn't surface. It appears once RBAC profiles are configured.
	if common.AuthzEnabled() {
		writePermissions(&b, common, section, kv)
	}

	// Per-cluster sections
	writeClusters(&b, common, s, section, kv)

	// Read-only notice
	b.WriteString("\n")
	b.WriteString(s.Muted.Render(
		"~/.kaf/config is read-only and is never rewritten by kafui.\n" +
			"Edit cluster connection settings via the kafui config wizard."))
	b.WriteString("\n")

	return b.String()
}

// writePermissions renders the effective-permissions ("whoami") view: identity,
// authz enabled flag, active profile, read-only flag, and the flattened
// permission set (implied actions already expanded by the Gate).
func writePermissions(b *strings.Builder, common *core.Common, section func(string), kv func(string, string)) {
	section("Permissions (whoami)")
	kv("Identity", fallback(common.Identity, "Unknown"))
	kv("Authorization", fmt.Sprintf("%t", common.AuthzEnabled()))
	kv("Active Profile", fallback(common.ActiveProfileName(), "(none)"))
	kv("Read Only", fmt.Sprintf("%t", common.IsReadOnly()))

	if common.Gate == nil || !common.AuthzEnabled() {
		kv("Effective", "all actions permitted (authz disabled)")
		return
	}
	perms := common.Gate.EffectivePermissions()
	if len(perms) == 0 {
		kv("Effective", "no permissions (no profile covers this cluster)")
		return
	}
	for _, p := range perms {
		name := p.Pattern
		if name == "" {
			name = "*"
		}
		kv(fmt.Sprintf("  %s [%s]", p.Resource, name), string(p.Action))
	}
}

// fallback returns v, or def when v is empty.
func fallback(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func writeClusters(b *strings.Builder, common *core.Common, s *stylesPkg.Styles, section func(string), kv func(string, string)) {
	if common.DataSource == nil {
		return
	}
	contexts, err := common.DataSource.GetContexts()
	if err != nil {
		section("Clusters")
		b.WriteString(s.Error.Render("  failed to list clusters: " + err.Error()))
		b.WriteString("\n")
		return
	}
	sort.Strings(contexts)

	red := common.Redactor
	for _, name := range contexts {
		section("Cluster: " + name)

		details, err := common.DataSource.GetClusterDetails(name)
		if err != nil {
			b.WriteString(s.Error.Render("  " + err.Error()))
			b.WriteString("\n")
			continue
		}
		kv("Brokers", strings.Join(details.Brokers, ", "))
		if details.SchemaRegistryURL != "" {
			kv("Schema Registry", red.Redact("schema.registry.url", details.SchemaRegistryURL))
		}
		kv("Read Only", fmt.Sprintf("%t", details.ReadOnly))
		kv("Current", fmt.Sprintf("%t", details.IsCurrent))

		if common.AppConfig == nil {
			continue
		}
		ext, ok := common.AppConfig.Clusters[name]
		if !ok {
			continue
		}
		writeExtension(b, ext, s, kv, red)
	}
}

func writeExtension(b *strings.Builder, ext appconfig.ClusterExtension, s *stylesPkg.Styles, kv func(string, string), red *appconfig.Redactor) {
	kv("Extension Read Only", fmt.Sprintf("%t", ext.ReadOnly))
	if ext.PollingThrottle > 0 {
		kv("Polling Throttle", ext.PollingThrottle.String())
	}
	for _, c := range ext.Connect {
		b.WriteString(s.Muted.Render("  Connect: " + c.Name))
		b.WriteString("\n")
		kv("  Address", c.Address)
		if c.Username != "" {
			kv("  Username", c.Username)
			kv("  Password", red.Redact("password", c.Password))
		}
	}
	if ext.Ksql != nil {
		kv("ksqlDB URL", ext.Ksql.URL)
		if ext.Ksql.Username != "" {
			kv("ksqlDB Username", ext.Ksql.Username)
			kv("ksqlDB Password", red.Redact("password", ext.Ksql.Password))
		}
	}
	writeProps(b, "Properties", ext.Properties, kv, red)
	writeProps(b, "Consumer Properties", ext.ConsumerProperties, kv, red)
	writeProps(b, "Producer Properties", ext.ProducerProperties, kv, red)
}

func writeProps(b *strings.Builder, label string, props map[string]any, kv func(string, string), red *appconfig.Redactor) {
	if len(props) == 0 {
		return
	}
	keysSorted := make([]string, 0, len(props))
	for k := range props {
		keysSorted = append(keysSorted, k)
	}
	sort.Strings(keysSorted)
	for _, k := range keysSorted {
		val := fmt.Sprintf("%v", props[k])
		kv("  "+k, red.Redact(k, val))
	}
}

// Init implements the Page interface.
func (m *Model) Init() tea.Cmd {
	return m.reusableApp.Init()
}

// Update implements the Page interface.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updatedApp, cmd := m.reusableApp.Update(msg)
	if updated, ok := updatedApp.(*templateui.ReusableApp); ok {
		m.reusableApp = updated
	}
	return m, cmd
}

// View implements the Page interface.
func (m *Model) View() string {
	return m.reusableApp.View()
}

// SetDimensions implements the Page interface.
func (m *Model) SetDimensions(width, height int) {
	m.dimensions = core.Dimensions{Width: width, Height: height}
	m.reusableApp.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

// GetID implements the Page interface.
func (m *Model) GetID() string {
	return pageID
}

// GetTitle implements the Page interface.
func (m *Model) GetTitle() string {
	return "Config"
}

// GetHelp implements the Page interface.
func (m *Model) GetHelp() []key.Binding {
	km := keys.DefaultKeyMap().Detail
	return []key.Binding{km.ScrollUp, km.ScrollDown, km.PageUp, km.PageDown, km.Back, km.Quit}
}

// HandleNavigation implements the Page interface.
func (m *Model) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
		return m, func() tea.Msg { return core.BackMsg{} }
	}
	return m, nil
}

// OnFocus implements the Page interface.
func (m *Model) OnFocus() tea.Cmd { return nil }

// OnBlur implements the Page interface.
func (m *Model) OnBlur() tea.Cmd { return nil }

// GetCommon returns the shared context.
func (m *Model) GetCommon() *core.Common { return m.common }
