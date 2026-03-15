package ui

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/debug"
	"github.com/Benny93/kafui/pkg/ui/router"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the main application state
type Model struct {
	common       *core.Common     // Shared context (replaces direct dataSource)
	Router       *router.Router   // Exported for testing
	state        core.UIState     // Application state (replaces ShowHelp bool)
	focusState   core.FocusState  // Focus state
	HelpSystem   *core.HelpSystem // Help system
	FocusManager *core.FocusManager
	width        int
	height       int
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
	return m.Router.Init()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update layout through Common context
		m.common.UpdateLayout(msg.Width, msg.Height)
		// Propagate dimensions to router and help system
		m.Router.SetDimensions(msg.Width, msg.Height)
		m.HelpSystem.SetDimensions(msg.Width, msg.Height)

	case tea.KeyMsg:
		// Handle focus management first (if not in help mode)
		if m.state != core.StateHelp {
			if cmd := m.FocusManager.HandleKeyMsg(msg); cmd != nil {
				return m, cmd
			}
		}

		// Handle global key bindings
		switch {
		case key.Matches(msg, core.DefaultGlobalKeys.DebugScreenshot):
			// Take screenshot (full)
			return m, m.takeScreenshot(false)
		case key.Matches(msg, core.DefaultGlobalKeys.DebugScreenshotRedacted):
			// Take screenshot (redacted)
			return m, m.takeScreenshot(true)
		case key.Matches(msg, core.DefaultGlobalKeys.ToggleTheme):
			// Toggle theme between dark and light
			if m.common.Styles != nil {
				m.common.Styles.ToggleTheme()
				// Update config to reflect current theme
				m.common.Config.Theme = string(m.common.Styles.GetTheme())
			}
			return m, nil
		case key.Matches(msg, core.DefaultGlobalKeys.Help):
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
		case key.Matches(msg, core.DefaultGlobalKeys.Quit):
			return m, tea.Quit
		case key.Matches(msg, core.DefaultGlobalKeys.Back):
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
	if m.state == core.StateHelp {
		return m.HelpSystem.Render()
	}

	return m.Router.View()
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
			Format:          debug.FormatPlainText,
			Redact:          redact,
			OutputDir:       m.common.Config.ScreenshotDir,
			Version:         "dev",
			CurrentPage:     pageID,
			PageContext:     pageContext,
			TerminalWidth:   m.width,
			TerminalHeight:  m.height,
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
	}
}
