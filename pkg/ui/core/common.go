package core

import (
	"slices"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/authz"
	"github.com/Benny93/kafui/pkg/cluster"
	"github.com/Benny93/kafui/pkg/metrics"
	"github.com/Benny93/kafui/pkg/ui/layout"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
)

// Common provides shared context and dependencies across all UI components.
// This pattern ensures consistent dependency injection and makes testing easier.
type Common struct {
	// DataSource is the Kafka data source
	DataSource api.KafkaDataSource

	// Styles contains all application styles
	Styles *stylesPkg.Styles

	// Layout contains the current layout configuration
	Layout *layout.Layout

	// LayoutConfig contains layout configuration options
	LayoutConfig *layout.LayoutConfig

	// Config contains UI configuration
	Config *UIConfig

	// AppConfig is the effective kafui-owned configuration (read-only flags,
	// optional integrations, UI prefs). Never nil.
	AppConfig *appconfig.Config

	// Redactor masks secrets when displaying configuration.
	Redactor *appconfig.Redactor

	// Collector is the background cluster statistics collector (may be nil in
	// lightweight/test contexts; callers must nil-check).
	Collector *cluster.Collector

	// MetricsCollector is the background metrics collector feeding the metrics
	// page (may be nil in lightweight/test contexts; callers must nil-check).
	MetricsCollector *metrics.Collector

	// Gate is the local authorization gate. Nil in lightweight/test contexts,
	// which means allow-all (see Can). Pages must query permissions through the
	// Can/AuthzEnabled/ActiveProfileName helpers, never a global.
	Gate *authz.Gate

	// Identity is the acting local user (OS user), shown in the header and
	// whoami view and recorded in the audit log.
	Identity string

	// InitialResource is the CLI --resource deep-link (UI-9), consumed once by
	// the main page at construction time so the sidebar/breadcrumb reflect it
	// from the start (BUG-7) instead of racing an async switch against the
	// page's own default-resource Init().
	InitialResource string
}

// Can reports whether the active profile permits action on the named resource.
// A nil Gate (tests / authz disabled) is allow-all. This is the single helper
// pages use to hide/disable mutating keys; blocked attempts still route the
// guard's typed error to the status bar. Use "" as name for create/unnamed
// checks (name patterns are ignored for those).
func (c *Common) Can(action authz.Action, rt authz.ResourceType, name string) bool {
	if c.Gate == nil {
		return true
	}
	return c.Gate.Allowed(action, rt, name)
}

// AuthzEnabled reports whether a permission profile is active.
func (c *Common) AuthzEnabled() bool {
	return c.Gate != nil && c.Gate.Enabled()
}

// ActiveProfileName returns the resolved active profile name (empty when authz
// is disabled or no profile covers the current cluster).
func (c *Common) ActiveProfileName() string {
	if c.Gate == nil {
		return ""
	}
	return c.Gate.ActiveProfileName()
}

// ActiveCapabilities returns the capability set of the active cluster from the
// collector cache, or nil if the collector is unavailable / has no data yet.
func (c *Common) ActiveCapabilities() []api.Capability {
	if c.Collector == nil || c.DataSource == nil {
		return nil
	}
	active := c.DataSource.GetContext()
	for _, ov := range c.Collector.ListClusters() {
		if ov.Name == active {
			return ov.Capabilities
		}
	}
	return nil
}

// HasCapability reports whether the active cluster advertises the given capability.
// Returns true when capabilities are unknown (collector not ready) so features are
// not hidden before the first collection cycle.
func (c *Common) HasCapability(cap api.Capability) bool {
	caps := c.ActiveCapabilities()
	if caps == nil {
		return true
	}
	return slices.Contains(caps, cap)
}

// ApplyAppConfig installs the loaded kafui config and syncs derived UI settings.
func (c *Common) ApplyAppConfig(cfg appconfig.Config) {
	c.AppConfig = &cfg
	c.Redactor = appconfig.NewRedactor(cfg.Redaction)
	if c.Config != nil {
		if cfg.UI.Theme != "" {
			c.Config.Theme = cfg.UI.Theme
		}
		c.Config.ShowSidebar = cfg.UI.ShowSidebar
		c.Config.CompactMode = cfg.UI.CompactMode
	}
	// Apply the persisted theme to the live styles. "auto" keeps the default
	// (dark) until terminal-background detection lands (UI-3).
	if c.Styles != nil {
		switch cfg.UI.Theme {
		case "light":
			c.Styles.SetTheme(stylesPkg.LightTheme)
		case "dark":
			c.Styles.SetTheme(stylesPkg.DarkTheme)
		}
	}
	// Apply the configured display timezone for timestamp formatting (UI-14).
	_ = shared.SetTimezone(cfg.UI.Timezone)
}

// IsReadOnly reports whether the active cluster is read-only, honoring both the
// per-cluster config flag and the global --read-only CLI flag (via the Gate).
func (c *Common) IsReadOnly() bool {
	if c.Gate != nil && c.Gate.ReadOnly() {
		return true
	}
	if c.AppConfig == nil || c.DataSource == nil {
		return false
	}
	ext, ok := c.AppConfig.Clusters[c.DataSource.GetContext()]
	return ok && ext.ReadOnly
}

// UIConfig contains UI-specific configuration
type UIConfig struct {
	// ShowSidebar indicates whether sidebar should be shown by default
	ShowSidebar bool

	// CompactMode indicates whether compact layout mode is enabled
	CompactMode bool

	// Theme name (e.g., "dark", "light")
	Theme string

	// ScreenshotDir is the directory for debug screenshots (defaults to temp dir)
	ScreenshotDir string

	// ConsumerGroupRefreshInterval is the auto-refresh interval selected on the
	// consumer-group detail page (0 = off). Persisted for the session so the
	// choice survives re-opening the page.
	// ponytail: session-scoped only — writing it back to the on-disk kafui config
	// is deferred to the application-config feature.
	ConsumerGroupRefreshInterval time.Duration
}

// DefaultUIConfig returns the default UI configuration
func DefaultUIConfig() *UIConfig {
	return &UIConfig{
		ShowSidebar: true,
		CompactMode: false,
		Theme:       "dark",
	}
}

// NewCommon creates a new Common context with the given data source.
// It initializes default styles and configuration.
func NewCommon(dataSource api.KafkaDataSource) *Common {
	layoutConfig := layout.DefaultLayoutConfig()
	defCfg := appconfig.Default()
	return &Common{
		DataSource:   dataSource,
		Styles:       stylesPkg.DefaultStyles(),
		Layout:       layout.CalculateLayout(80, 24, layoutConfig), // Default size
		LayoutConfig: layoutConfig,
		Config:       DefaultUIConfig(),
		AppConfig:    &defCfg,
		Redactor:     appconfig.NewRedactor(defCfg.Redaction),
	}
}

// NewCommonWithConfig creates a new Common context with custom configuration
func NewCommonWithConfig(dataSource api.KafkaDataSource, config *UIConfig) *Common {
	layoutConfig := layout.DefaultLayoutConfig()
	defCfg := appconfig.Default()
	return &Common{
		DataSource:   dataSource,
		Styles:       stylesPkg.DefaultStyles(),
		Layout:       layout.CalculateLayout(80, 24, layoutConfig), // Default size
		LayoutConfig: layoutConfig,
		Config:       config,
		AppConfig:    &defCfg,
		Redactor:     appconfig.NewRedactor(defCfg.Redaction),
	}
}

// UpdateLayout recalculates the layout based on new dimensions
func (c *Common) UpdateLayout(width, height int) {
	c.Layout = layout.CalculateLayout(width, height, c.LayoutConfig)
}

// GetLayout returns the current layout, calculating it if necessary
func (c *Common) GetLayout(width, height int) *layout.Layout {
	if c.Layout == nil || c.Layout.Width != width || c.Layout.Height != height {
		c.UpdateLayout(width, height)
	}
	return c.Layout
}
