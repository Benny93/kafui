package ui

import (
	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
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

// Key mappings for legacy compatibility (unused, kept for API compatibility)
type keyMap struct {
	Search    key.Binding
	TopicMode key.Binding
	Back      key.Binding
	Quit      key.Binding
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

// setFocusState updates the focus state
func (m *Model) setFocusState(focus core.FocusState) {
	m.focusState = focus
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
