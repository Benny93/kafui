package ui

import (
	"context"
	"fmt"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/cluster"
	"github.com/Benny93/kafui/pkg/metrics"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/debug"
	"github.com/Benny93/kafui/pkg/ui/dialog"
	keys "github.com/Benny93/kafui/pkg/ui/keys"
	"github.com/Benny93/kafui/pkg/ui/notify"
	"github.com/Benny93/kafui/pkg/ui/router"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	templatestyles "github.com/Benny93/kafui/pkg/ui/template/ui/styles"
	"github.com/Benny93/kafui/pkg/version"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

// Model represents the main application state
type Model struct {
	common          *core.Common     // Shared context (replaces direct dataSource)
	Router          *router.Router   // Exported for testing
	state           core.UIState     // Application state (replaces ShowHelp bool)
	focusState      core.FocusState  // Focus state
	HelpSystem      *core.HelpSystem // Help system
	FocusManager    *core.FocusManager
	confirm         *dialog.Confirm // Root-owned confirmation modal
	notifier        *notify.Manager // Shell-owned notification/status line
	width           int
	height          int
}

// initialModelWithRouter creates a new Model using the router-based navigation
func initialModelWithRouter(dataSource api.KafkaDataSource) *Model {
	// Create Common context with data source, styles, and config
	common := core.NewCommon(dataSource)

	r := router.NewRouter(common)
	helpSystem := core.NewHelpSystem()
	focusManager := core.NewFocusManager()

	return &Model{
		common:       common,
		Router:       r,
		state:        core.StateNormal,
		focusState:   core.FocusMain,
		HelpSystem:   helpSystem,
		FocusManager: focusManager,
		confirm:      dialog.New(common.Styles),
		notifier:     notify.New(common.Styles),
	}
}

// GetCommon returns the shared context
func (m *Model) GetCommon() *core.Common {
	return m.common
}

// GetState returns the current UI state
func (m *Model) GetState() core.UIState {
	return m.state
}

// GetFocusState returns the current focus state
func (m *Model) GetFocusState() core.FocusState {
	return m.focusState
}

// setState updates the UI state and handles side effects
func (m *Model) setState(state core.UIState) {
	m.state = state
}

func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.Router.Init()}
	// Kick off background cluster statistics collection and its periodic tick.
	if c := m.common.Collector; c != nil {
		cmds = append(cmds, c.CollectCmd(), c.TickCmd())
	}
	// Kick off background metrics collection and its periodic tick.
	if mc := m.common.MetricsCollector; mc != nil {
		cmds = append(cmds, mc.CollectCmd(), mc.TickCmd())
	}
	if cmd := m.releaseCheckCmd(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

// currentThemeMode returns the persisted theme mode ("auto", "dark", "light").
func (m *Model) currentThemeMode() string {
	if m.common.AppConfig != nil && m.common.AppConfig.UI.Theme != "" {
		return m.common.AppConfig.UI.Theme
	}
	if m.common.Config != nil && m.common.Config.Theme != "" {
		return m.common.Config.Theme
	}
	return "auto"
}

// nextThemeMode cycles auto → dark → light → auto (UI-3).
func nextThemeMode(cur string) string {
	switch cur {
	case "auto":
		return "dark"
	case "dark":
		return "light"
	default: // "light" or unknown
		return "auto"
	}
}

// applyThemeMode resolves the mode to a concrete dark/light palette (auto uses
// terminal-background detection) and applies it to BOTH the core styles and the
// template chrome so the whole UI follows the selection (UI-3).
func (m *Model) applyThemeMode(mode string) {
	isDark := true
	switch mode {
	case "light":
		isDark = false
	case "dark":
		isDark = true
	default: // "auto"
		isDark = lipgloss.HasDarkBackground()
	}
	if m.common.Styles != nil {
		if isDark {
			m.common.Styles.SetTheme(stylesPkg.DarkTheme)
		} else {
			m.common.Styles.SetTheme(stylesPkg.LightTheme)
		}
	}
	templatestyles.SetTheme(isDark)
	if m.common.Config != nil {
		m.common.Config.Theme = mode
	}
	if m.common.AppConfig != nil {
		m.common.AppConfig.UI.Theme = mode
	}
}

// persistThemeCmd writes the chosen theme back to the kafui config file so it
// survives restarts (AC-15). Persistence failures surface as a warning
// notification but never block the toggle.
func (m *Model) persistThemeCmd(theme string) tea.Cmd {
	if m.common.AppConfig == nil {
		return nil
	}
	m.common.AppConfig.UI.Theme = theme
	cfg := *m.common.AppConfig
	return func() tea.Msg {
		if err := appconfig.Save(appconfig.DefaultPath(), cfg); err != nil {
			return core.NotificationMsg{Severity: core.StatusWarning, Title: "Config", Message: "could not save theme: " + err.Error()}
		}
		return nil
	}
}

// persistSidebarCmd writes the sidebar visibility preference back to the kafui
// config (UI-15). Failures surface as a warning notification, never blocking.
func (m *Model) persistSidebarCmd(visible bool) tea.Cmd {
	if m.common.AppConfig == nil {
		return nil
	}
	m.common.AppConfig.UI.ShowSidebar = visible
	if m.common.Config != nil {
		m.common.Config.ShowSidebar = visible
	}
	cfg := *m.common.AppConfig
	return func() tea.Msg {
		if err := appconfig.Save(appconfig.DefaultPath(), cfg); err != nil {
			return core.NotificationMsg{Severity: core.StatusWarning, Title: "Config", Message: "could not save sidebar preference: " + err.Error()}
		}
		return nil
	}
}

// releaseCheckCmd performs the opt-in latest-release check once at startup and
// emits a warning notification when the running build is outdated. Disabled via
// config; all failures are silent (air-gapped friendly).
func (m *Model) releaseCheckCmd() tea.Cmd {
	if m.common.AppConfig == nil || !m.common.AppConfig.ReleaseCheck.Enabled {
		return nil
	}
	timeout := m.common.AppConfig.ReleaseCheck.Timeout
	return func() tea.Msg {
		rel, err := version.CheckLatest(context.Background(), timeout)
		if err != nil || rel == nil {
			return nil
		}
		if version.IsOutdated(version.Version, rel.TagName) {
			return core.NotificationMsg{
				Severity: core.StatusWarning,
				Title:    "Update available",
				Message:  "kafui " + rel.TagName + " is available (running " + version.Version + ")",
			}
		}
		return nil
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Confirmation dialog: intercept the request to open it, and while it is
	// open trap all key/mouse input so the page underneath is frozen.
	if showMsg, ok := msg.(core.ShowConfirmMsg); ok {
		m.confirm.Show(showMsg)
		m.confirm.SetDimensions(m.width, m.height)
		return m, nil
	}
	if m.confirm.Active() {
		switch msg.(type) {
		case tea.KeyMsg, tea.MouseMsg:
			cmd, _ := m.confirm.Update(msg)
			return m, cmd
		}
	}

	// Periodic collection tick: run a cycle and reschedule.
	if _, ok := msg.(cluster.CollectTickMsg); ok {
		if c := m.common.Collector; c != nil {
			return m, tea.Batch(c.CollectCmd(), c.TickCmd())
		}
		return m, nil
	}

	// Periodic metrics collection tick: run a cycle and reschedule.
	if _, ok := msg.(metrics.CollectTickMsg); ok {
		if mc := m.common.MetricsCollector; mc != nil {
			return m, tea.Batch(mc.CollectCmd(), mc.TickCmd())
		}
		return m, nil
	}

	// Config hot-reload (AC-16): apply reloadable settings (UI prefs, cluster
	// extensions) without reconnecting the active cluster, and surface a notice.
	if reload, ok := msg.(core.ConfigReloadedMsg); ok {
		if cfg, ok := reload.Config.(*appconfig.Config); ok && cfg != nil {
			m.common.ApplyAppConfig(*cfg)
			m.applyThemeMode(cfg.UI.Theme)
		}
		return m, core.NewNotification(core.StatusInfo, "Config changed",
			"reloaded settings — press ctrl+g to review the active cluster")
	}

	// Sidebar toggle: persist the user's explicit choice (UI-15). The template
	// already flipped its own visibility; the shell just writes it back.
	if tog, ok := msg.(core.SidebarToggledMsg); ok {
		cmd := m.persistSidebarCmd(tog.Visible)
		// Continue delegating so the page still processes any batched work.
		if m.state != core.StateHelp {
			_, rcmd := m.Router.Update(msg)
			return m, tea.Batch(cmd, rcmd)
		}
		return m, cmd
	}

	// Notification/status messages are consumed by the shell-owned notifier.
	if cmd, consumed := m.notifier.HandleMsg(msg); consumed {
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update layout through Common context
		m.common.UpdateLayout(msg.Width, msg.Height)
		// Propagate dimensions to router and help system
		m.Router.SetDimensions(msg.Width, msg.Height)
		m.HelpSystem.SetDimensions(msg.Width, msg.Height)
		m.confirm.SetDimensions(msg.Width, msg.Height)

	case tea.KeyMsg:
		// Handle debug screenshot keys first (before focus manager)
		switch msg.String() {
		case "f3":
			return m, m.takeScreenshot(false)
		case "shift+f3":
			return m, m.takeScreenshot(true)
		}

		// Handle focus management first (if not in help mode)
		if m.state != core.StateHelp {
			if cmd := m.FocusManager.HandleKeyMsg(msg); cmd != nil {
				return m, cmd
			}
		}

		// A page (e.g. the ksqlDB query editor) may hold a focused text input.
		// While it does, single-key global hotkeys must not fire so typed
		// characters (including 'q', 'C', 'K', 'T', '?') reach the input; esc
		// (back) and ctrl+c (quit) still work as escapes.
		inputMode := false
		if p, ok := m.Router.GetCurrentPage().(interface{ IsInputMode() bool }); ok {
			inputMode = p.IsInputMode()
		}

		// Handle global key bindings
		switch {
		case key.Matches(msg, keys.GlobalKeys.ToggleTheme) && !inputMode:
			// Cycle theme auto → dark → light and sync both style systems (UI-3).
			next := nextThemeMode(m.currentThemeMode())
			m.applyThemeMode(next)
			return m, m.persistThemeCmd(next)
		case key.Matches(msg, keys.GlobalKeys.Help) && !inputMode:
			// Toggle help state
			if m.state == core.StateHelp {
				m.setState(core.StateNormal)
				m.HelpSystem.Hide()
			} else {
				m.setState(core.StateHelp)
				m.HelpSystem.Toggle()
				// Update help system with current page
				if currentPage := m.Router.GetCurrentPage(); currentPage != nil {
					m.HelpSystem.SetCurrentPage(currentPage)
				}
			}
			return m, nil
		case key.Matches(msg, keys.GlobalKeys.Clusters) && !inputMode:
			if m.state != core.StateHelp {
				return m, m.Router.NavigateTo("clusters", nil)
			}
		case key.Matches(msg, keys.GlobalKeys.Ksql) && !inputMode:
			// ponytail: ksqlDB is reached via this global key (consistent with the
			// clusters 'C' and appconfig 'ctrl+g' dashboards, which likewise have no
			// sidebar item). A capability-gated main-page sidebar entry is deferred.
			// Gated on the active cluster advertising ksqlDB support.
			if m.state != core.StateHelp && m.common.HasCapability(api.CapKsqlDB) {
				return m, m.Router.NavigateTo("ksql", nil)
			}
		case key.Matches(msg, keys.GlobalKeys.Metrics) && !inputMode:
			// Metrics are always available (offset-delta collection needs no
			// config), so this global key is not capability-gated.
			if m.state != core.StateHelp {
				return m, m.Router.NavigateTo("metrics", nil)
			}
		case key.Matches(msg, keys.GlobalKeys.Config) && !inputMode:
			if m.state != core.StateHelp {
				return m, m.Router.NavigateTo("appconfig", nil)
			}
		case key.Matches(msg, keys.GlobalKeys.ClusterWizard) && !inputMode:
			// Cluster setup wizard, gated on the dynamic-config toggle (AC-12).
			if m.state != core.StateHelp {
				if m.common.AppConfig != nil && m.common.AppConfig.DynamicConfigEnabled {
					return m, m.Router.NavigateTo("cluster_form", nil)
				}
				return m, core.NewNotification(core.StatusInfo, "Cluster wizard disabled",
					"set dynamicConfigEnabled: true in the kafui config to enable in-app cluster editing")
			}
		case key.Matches(msg, keys.GlobalKeys.Quit) && (!inputMode || msg.String() == "ctrl+c"):
			return m, tea.Quit
		case key.Matches(msg, keys.GlobalKeys.Back):
			if m.state != core.StateHelp {
				return m, m.Router.Back()
			}
			// Close help if it's open
			m.setState(core.StateNormal)
			m.HelpSystem.Hide()
			return m, nil
		}
	}

	// Handle router updates if not in help mode
	if m.state != core.StateHelp {
		_, cmd := m.Router.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	var content string
	if m.state == core.StateHelp {
		content = m.HelpSystem.Render()
	} else {
		content = m.Router.View()
	}
	// Append the transient notification/status line at the bottom.
	if !m.notifier.Empty() {
		content = lipgloss.JoinVertical(lipgloss.Left, content, m.notifier.View(m.width))
	}
	// Overlay the confirmation modal when active.
	if m.confirm.Active() {
		content = m.confirm.View(content)
	}
	// zone.Scan must only be called once at the root model so that bubblezone
	// can register the offsets of all child zone.Mark() calls before returning
	// the final rendered string to Bubble Tea.
	return zone.Scan(content)
}

// takeScreenshot captures the current TUI screen to a file
func (m *Model) takeScreenshot(redact bool) tea.Cmd {
	return func() tea.Msg {
		// Get current view
		view := m.View()

		// Get current page info
		currentPage := m.Router.GetCurrentPage()
		pageID := "unknown"
		pageContext := ""
		if currentPage != nil {
			pageID = currentPage.GetID()
			pageContext = fmt.Sprintf("state=%s, focus=%s", m.state, m.focusState)
		}

		// Capture screenshot
		options := debug.CaptureOptions{
			Format:         debug.FormatPlainText,
			Redact:         redact,
			OutputDir:      m.common.Config.ScreenshotDir,
			Version:        version.Version,
			CurrentPage:    pageID,
			PageContext:    pageContext,
			TerminalWidth:  m.width,
			TerminalHeight: m.height,
		}

		filepath, err := debug.Capture(view, options)
		if err != nil {
			return core.StatusMsg{
				Message: fmt.Sprintf("Screenshot failed: %v", err),
				Type:    core.StatusError,
			}
		}

		// Return success message
		msg := fmt.Sprintf("Screenshot saved: %s", filepath)
		if redact {
			msg = fmt.Sprintf("Redacted screenshot saved: %s", filepath)
		}

		return core.StatusMsg{
			Message: msg,
			Type:    core.StatusSuccess,
		}
	}
}

// NewUIModel creates a new UI model using router-based navigation
func NewUIModel(dataSource api.KafkaDataSource) *Model {
	return initialModelWithRouter(dataSource)
}

// NewUIModelWithRouter creates a new UI model using router-based navigation
func NewUIModelWithRouter(dataSource api.KafkaDataSource) *Model {
	return initialModelWithRouter(dataSource)
}

// NewUIModelWithCommon creates a new UI model with a pre-configured Common context
func NewUIModelWithCommon(common *core.Common) *Model {
	r := router.NewRouter(common)
	helpSystem := core.NewHelpSystem()
	focusManager := core.NewFocusManager()

	return &Model{
		common:       common,
		Router:       r,
		state:        core.StateNormal,
		focusState:   core.FocusMain,
		HelpSystem:   helpSystem,
		FocusManager: focusManager,
		confirm:      dialog.New(common.Styles),
		notifier:     notify.New(common.Styles),
	}
}
