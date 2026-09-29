package ui

import (
	"context"
	"fmt"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/cluster"
	"github.com/Benny93/kafui/pkg/metrics"
	"github.com/Benny93/kafui/pkg/ui/components/menu"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/debug"
	"github.com/Benny93/kafui/pkg/ui/dialog"
	keys "github.com/Benny93/kafui/pkg/ui/keys"
	"github.com/Benny93/kafui/pkg/ui/notify"
	"github.com/Benny93/kafui/pkg/ui/router"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	templatestyles "github.com/Benny93/kafui/pkg/ui/template/ui/styles"
	"github.com/Benny93/kafui/pkg/version"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

// Model represents the main application state
type Model struct {
	common       *core.Common     // Shared context (replaces direct dataSource)
	Router       *router.Router   // Exported for testing
	state        core.UIState     // Application state (replaces ShowHelp bool)
	focusState   core.FocusState  // Focus state
	HelpSystem   *core.HelpSystem // Help system
	FocusManager *core.FocusManager
	confirm      *dialog.Confirm // Root-owned confirmation modal
	notifier     *notify.Manager // Shell-owned notification/status line
	palette      *menu.Model     // Command palette (`:` / ctrl+p)
	actions      *menu.Model     // Contextual actions menu (`a` / right-click)
	mouseOn      bool            // Mouse reporting; off releases the terminal's own selection
	keycast      debug.Keycast   // Debug builds: the recent-keys strip
	width        int
	height       int

	// metricsExposed keeps metrics collection running for the Prometheus
	// exposition endpoint; otherwise it runs only while the metrics page shows.
	metricsExposed bool
	// metricsTickArmed is true while a metrics CollectTickMsg is pending, so
	// at most one tick chain exists.
	metricsTickArmed bool
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
		palette:      menu.New(),
		actions:      menu.New(),
		mouseOn:      true,
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
	// Metrics collection starts here only for the exposition endpoint; the
	// metrics page starts it when it opens. Its tick is armed when the cycle
	// reports back.
	if mc := m.common.MetricsCollector; mc != nil && m.metricsExposed {
		cmds = append(cmds, mc.CollectCmd())
	}
	if cmd := m.releaseCheckCmd(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

// metricsWanted reports whether anyone reads metrics right now: the metrics
// page is showing, or the exposition endpoint is serving.
func (m *Model) metricsWanted() bool {
	return m.metricsExposed || m.Router.GetCurrentPageID() == "metrics"
}

// armMetricsTick schedules the next metrics cycle, unless one is already
// scheduled or nobody reads metrics. The chain then stops, and the metrics
// page restarts it by collecting when it opens.
func (m *Model) armMetricsTick() tea.Cmd {
	mc := m.common.MetricsCollector
	if mc == nil || m.metricsTickArmed || !m.metricsWanted() {
		return nil
	}
	m.metricsTickArmed = true
	return mc.TickCmd()
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

	// A page's own results arrive addressed to that page (CONC-4). The shell
	// reacts to the payload like any other message, and hands the envelope to
	// the router so the payload reaches the page that asked for it.
	routed := msg
	pm, addressed := msg.(core.PageMsg)
	if addressed {
		msg = pm.Msg
	}

	// Confirmation dialog: intercept the request to open it, and while it is
	// open trap all key/mouse input so the page underneath is frozen. A page
	// raises it from a command that may finish after the user left the page or
	// switched clusters; opening it then would ask about the old page over the
	// new one, and confirming would act on the cluster active now.
	if showMsg, ok := msg.(core.ShowConfirmMsg); ok {
		if addressed && !m.Router.IsLive(pm) {
			return m, nil
		}
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

	// Shell-internal messages raised by palette entries.
	if cmd, handled := m.handleControlMsg(msg); handled {
		return m, cmd
	}

	// Periodic collection tick: run a cycle and reschedule.
	if _, ok := msg.(cluster.CollectTickMsg); ok {
		if c := m.common.Collector; c != nil {
			return m, tea.Batch(c.CollectCmd(), c.TickCmd())
		}
		return m, nil
	}

	// Periodic metrics collection tick: run a cycle if anyone reads the
	// result. The next tick is armed when the cycle reports back, below, so
	// a cycle slower than the interval never overlaps the next (PERF-2).
	if _, ok := msg.(metrics.CollectTickMsg); ok {
		m.metricsTickArmed = false
		if mc := m.common.MetricsCollector; mc != nil && m.metricsWanted() {
			return m, mc.CollectCmd()
		}
		return m, nil
	}
	if _, ok := msg.(metrics.MetricsUpdatedMsg); ok {
		// Not consumed: the metrics page renders from it.
		cmds = append(cmds, m.armMetricsTick())
	}

	// Config hot-reload (AC-16): apply reloadable settings (UI prefs, cluster
	// extensions) without reconnecting the active cluster, and surface a notice.
	if reload, ok := msg.(core.ConfigReloadedMsg); ok {
		if cfg, ok := reload.Config.(*appconfig.Config); ok && cfg != nil {
			m.common.ApplyAppConfig(*cfg)
			m.applyThemeMode(cfg.UI.Theme)
		}
		return m, core.NewNotification(core.StatusInfo, "Config changed",
			"reloaded settings — open the command palette to review the active cluster")
	}

	// Sidebar toggle: persist the user's explicit choice (UI-15). The template
	// already flipped its own visibility; the shell just writes it back.
	if tog, ok := msg.(core.SidebarToggledMsg); ok {
		cmd := m.persistSidebarCmd(tog.Visible)
		// Continue delegating so the page still processes any batched work.
		_, rcmd := m.Router.Update(routed)
		return m, tea.Batch(cmd, rcmd)
	}

	// Notification/status messages are consumed by the shell-owned notifier.
	if cmd, consumed := m.notifier.HandleMsg(msg); consumed {
		return m, cmd
	}

	if _, isMouse := msg.(tea.MouseMsg); isMouse {
		if cmd, consumed := m.palette.Update(msg); consumed {
			return m, cmd
		}
		if cmd, consumed := m.actions.Update(msg); consumed {
			return m, cmd
		}
		// Right-click anywhere opens the actions menu for the focused item.
		if mm, ok := msg.(tea.MouseMsg); ok && mm.Button == tea.MouseButtonRight && mm.Action == tea.MouseActionRelease {
			m.openActions()
			return m, nil
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.palette.SetDimensions(msg.Width, msg.Height)
		m.actions.SetDimensions(msg.Width, msg.Height)
		// Update layout through Common context
		m.common.UpdateLayout(msg.Width, msg.Height)
		// Propagate dimensions to router and help system
		m.Router.SetDimensions(msg.Width, msg.Height)
		m.HelpSystem.SetDimensions(msg.Width, msg.Height)
		m.confirm.SetDimensions(msg.Width, msg.Height)

	case tea.KeyMsg:
		// Record the press and what the registry made of it, before any layer
		// consumes it. An unbound key and a bound key whose action did nothing
		// look identical on screen; this tells them apart.
		if debug.KeycastEnabled {
			m.recordKey(msg)
		}

		// Overlay precedence: while the palette or the actions menu is open it
		// owns every key, so nothing reaches the screen behind it.
		if cmd, consumed := m.palette.Update(msg); consumed {
			return m, cmd
		}
		if cmd, consumed := m.actions.Update(msg); consumed {
			return m, cmd
		}

		// Debug-build keys resolve before focus handling so they work anywhere.
		if a, ok := keys.Default.Resolve(keys.ScopeDebug, msg.String()); ok {
			switch a {
			case keys.ActionScreenshot:
				return m, m.takeScreenshot(false)
			case keys.ActionScreenshotRedacted:
				return m, m.takeScreenshot(true)
			}
		}

		// Focus cycling (tab / shift+tab) when not in help.
		if m.state != core.StateHelp {
			if cmd := m.FocusManager.HandleKeyMsg(msg); cmd != nil {
				return m, cmd
			}
		}

		// Text-entry precedence: a page holding a focused field consumes every
		// key but the emergency exit, so typed characters are never actions.
		if m.inInputMode() {
			// ctrl+c is the single binding that escapes text entry, and the
			// text-entry context is where that exception is declared.
			if a, ok := keys.Default.Resolve(keys.ScopeTextEntry, msg.String()); ok && a == keys.ActionForceQuit {
				return m, tea.Quit
			}
			break
		}

		// The shell claims a fixed handful of global actions and forwards
		// everything else. It never does both.
		action, bound := keys.Default.Resolve(keys.ScopeDebug, msg.String())
		if !bound {
			break
		}
		switch action {
		case keys.ActionHelp:
			if m.state == core.StateHelp {
				m.setState(core.StateNormal)
				m.HelpSystem.Hide()
			} else {
				m.setState(core.StateHelp)
				m.HelpSystem.Toggle()
				if currentPage := m.Router.GetCurrentPage(); currentPage != nil {
					m.HelpSystem.SetCurrentPage(currentPage)
				}
			}
			return m, nil

		case keys.ActionPalette:
			m.openPalette()
			return m, nil

		case keys.ActionActionsMenu:
			m.openActions()
			return m, nil

		case keys.ActionForceQuit:
			return m, tea.Quit

		case keys.ActionQuit:
			if m.state == core.StateHelp {
				m.setState(core.StateNormal)
				m.HelpSystem.Hide()
				return m, nil
			}
			return m, tea.Quit

		case keys.ActionCancel:
			// Esc unwinds exactly one level. Help first, then whatever the page
			// has open, and only then the parent screen.
			if m.state == core.StateHelp {
				m.setState(core.StateNormal)
				m.HelpSystem.Hide()
				return m, nil
			}
			if u, ok := m.Router.GetCurrentPage().(core.Unwinder); ok {
				if cmd, consumed := u.Unwind(); consumed {
					return m, cmd
				}
			}
			return m, m.Router.Back()
		}
	}

	// The help overlay holds the keyboard and mouse. Everything else still
	// reaches the pages underneath: dropping a fetch result or a listener's
	// next message there would leave a page loading forever, or stop a live
	// stream, once help closes (CONC-4).
	if m.state == core.StateHelp {
		switch msg.(type) {
		case tea.KeyMsg, tea.MouseMsg:
			return m, tea.Batch(cmds...)
		}
	}
	_, cmd := m.Router.Update(routed)
	cmds = append(cmds, cmd)

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
	// Debug builds show the recent-keys strip above the status line.
	if v := m.keycast.View(); v != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, v)
	}
	// Overlay the discovery surfaces, then the confirmation modal, so a
	// confirmation raised from a menu entry renders above the menu.
	// Centre the overlays over the CONTENT pane, not the whole terminal: a box
	// centred on the terminal reaches into the sidebar and covers its border
	// and bullet column, which reads as corrupted text rather than as something
	// drawn on top.
	if v := m.palette.View(); v != "" {
		content = composite(content, v, m.contentWidth(), m.height)
	}
	if v := m.actions.View(); v != "" {
		content = composite(content, v, m.contentWidth(), m.height)
	}
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
		palette:      menu.New(),
		actions:      menu.New(),
		mouseOn:      true,
	}
}
