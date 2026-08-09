package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components/menu"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	mainpage "github.com/Benny93/kafui/pkg/ui/pages/main"
)

// This file holds the shell's two discovery surfaces. Between them they satisfy
// the controls spec's coverage rule: every action is reachable without knowing
// a key, and — because right-click opens the same actions menu — without a
// keyboard either.

// openPalette builds and shows the command palette: every navigable destination
// plus every application-level command, with the page's own contributions
// merged in. Destinations that used to have dedicated global keys (C, K,
// ctrl+t, ctrl+g, ctrl+w) live here now.
func (m *Model) openPalette() {
	entries := m.paletteEntries()
	m.palette.SetDimensions(m.contentWidth(), m.height)
	m.palette.OpenFilter("Commands", entries)
}

func (m *Model) paletteEntries() []menu.Entry {
	nav := func(pageID string) func() tea.Cmd {
		return func() tea.Cmd { return core.NewPageChangeMsg(pageID, nil) }
	}

	entries := []menu.Entry{}

	// Resource destinations come from the main screen. They must be offered
	// from EVERY screen — asking only the current page meant that once you
	// navigated to metrics or ksqlDB there was no palette route back to a
	// resource list, which is the opposite of what the palette promises.
	for _, rt := range mainpage.PaletteResources() {
		rt := rt
		entries = append(entries, menu.Entry{
			Label: "Show: " + rt.String(),
			Run: func() tea.Cmd {
				return tea.Sequence(
					core.NewPageChangeMsg("main", nil),
					func() tea.Msg { return mainpage.SwitchResourceMsg(rt) },
				)
			},
		})
	}

	// Anything else the current page wants to contribute.
	if p, ok := m.Router.GetCurrentPage().(core.PaletteProvider); ok {
		entries = append(entries, p.PaletteEntries()...)
	}

	entries = append(entries,
		menu.Entry{Label: "Go to: cluster dashboard", Run: nav("clusters")},
		menu.Entry{Label: "Go to: metrics", Run: nav("metrics")},
		menu.Entry{Label: "Go to: application config", Run: nav("appconfig")},
	)

	// ksqlDB is capability-gated. Listed-but-disabled rather than absent, so the
	// user learns the feature exists and why it is unavailable.
	ksql := menu.Entry{Label: "Go to: ksqlDB", Run: nav("ksql")}
	if !m.common.HasCapability(api.CapKsqlDB) {
		ksql.Disabled = true
		ksql.Reason = "the active cluster does not advertise ksqlDB"
	}
	entries = append(entries, ksql)

	wizard := menu.Entry{Label: "Go to: cluster setup wizard", Run: nav("cluster_form")}
	if m.common.AppConfig == nil || !m.common.AppConfig.DynamicConfigEnabled {
		wizard.Disabled = true
		wizard.Reason = "set dynamicConfigEnabled: true to edit clusters in-app"
	}
	entries = append(entries, wizard)

	entries = append(entries,
		menu.Entry{
			Label: "Toggle theme (auto / dark / light)",
			Run: func() tea.Cmd {
				return func() tea.Msg { return toggleThemeMsg{} }
			},
		},
		menu.Entry{
			Label: "Toggle sidebar",
			Key:   keys.Default.KeyFor(keys.ActionSidebar),
			Run: func() tea.Cmd {
				return func() tea.Msg { return toggleSidebarMsg{} }
			},
		},
		menu.Entry{
			Label: mouseToggleLabel(m.mouseOn),
			Desc:  "release the mouse to use the terminal's own text selection",
			Run: func() tea.Cmd {
				return func() tea.Msg { return toggleMouseMsg{} }
			},
		},
		menu.Entry{
			Label: "Help",
			Key:   keys.Default.KeyFor(keys.ActionHelp),
			Run:   func() tea.Cmd { return func() tea.Msg { return showHelpMsg{} } },
		},
		menu.Entry{
			Label: "Quit",
			Key:   keys.Default.KeyFor(keys.ActionQuit),
			Run:   func() tea.Cmd { return tea.Quit },
		},
	)
	return entries
}

func mouseToggleLabel(on bool) string {
	if on {
		return "Turn mouse reporting off"
	}
	return "Turn mouse reporting on"
}

// openActions shows the contextual actions menu for whatever the current page
// has focused. A page that contributes nothing still gets a menu listing the
// global actions, so `a` is never a dead key.
func (m *Model) openActions() {
	var entries []menu.Entry
	if p, ok := m.Router.GetCurrentPage().(core.ActionProvider); ok {
		entries = p.ContextActions()
	}
	if len(entries) == 0 {
		entries = []menu.Entry{{
			Label:    "No actions for the current selection",
			Disabled: true,
			Reason:   "select a row, or press : for application commands",
		}}
	}
	m.actions.SetDimensions(m.contentWidth(), m.height)
	m.actions.Open("Actions", entries)
}

// Shell-internal messages raised by palette entries. They exist so palette
// entries stay pure data (a tea.Cmd) instead of closing over the model.
type (
	toggleThemeMsg   struct{}
	toggleMouseMsg   struct{}
	toggleSidebarMsg struct{}
	showHelpMsg      struct{}
)

// handleControlMsg processes the shell-internal messages above. It returns
// false when msg is not one of them.
func (m *Model) handleControlMsg(msg tea.Msg) (tea.Cmd, bool) {
	switch msg.(type) {
	case toggleThemeMsg:
		next := nextThemeMode(m.currentThemeMode())
		m.applyThemeMode(next)
		return m.persistThemeCmd(next), true

	case toggleMouseMsg:
		m.mouseOn = !m.mouseOn
		if m.mouseOn {
			return tea.Batch(
				tea.EnableMouseCellMotion,
				core.NewNotification(core.StatusInfo, "Mouse on", "mouse reporting enabled"),
			), true
		}
		return tea.Batch(
			tea.DisableMouse,
			core.NewNotification(core.StatusInfo, "Mouse off",
				"the terminal's own text selection now works; press : to turn it back on"),
		), true

	case toggleSidebarMsg:
		// Re-raise as the key the shell template already understands, so the
		// palette entry and ctrl+b take exactly the same path.
		return func() tea.Msg {
			return tea.KeyMsg{Type: tea.KeyCtrlB}
		}, true

	case showHelpMsg:
		m.setState(core.StateHelp)
		m.HelpSystem.Toggle()
		if p := m.Router.GetCurrentPage(); p != nil {
			m.HelpSystem.SetCurrentPage(p)
		}
		return nil, true
	}
	return nil, false
}

// recordKey feeds the debug keycast strip. It resolves the press against the
// scope that is actually active — an overlay, a text field, or the current
// page — so the strip reports what the application really did with the key
// rather than what the global scope would have done.
func (m *Model) recordKey(msg tea.KeyMsg) {
	scope := keys.ScopeDebug
	switch {
	case m.palette.Active() || m.actions.Active() || m.confirm.Active():
		scope = keys.ScopeOverlay
	case m.inInputMode():
		scope = keys.ScopeTextEntry
	default:
		if p := m.Router.GetCurrentPage(); p != nil {
			scope = core.ScopeOf(p)
		}
	}

	action, bound := keys.Default.Resolve(scope, msg.String())
	if !bound {
		m.keycast.Record(msg.String(), "")
		return
	}
	m.keycast.Record(msg.String(), string(action))
}

// inInputMode reports whether the current page holds a focused text field.
func (m *Model) inInputMode() bool {
	p, ok := m.Router.GetCurrentPage().(core.InputModeReporter)
	return ok && p.IsInputMode()
}

// contentWidth is the width of the page area, excluding the sidebar. Overlays
// are sized and centred against it so they never graze the sidebar.
func (m *Model) contentWidth() int {
	if m.common != nil && m.common.Layout != nil {
		if w := m.common.Layout.GetAvailableWidth(); w > 20 {
			return w
		}
	}
	return m.width
}
