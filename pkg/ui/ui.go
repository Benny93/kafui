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
	dataSource   api.KafkaDataSource
	Router       *router.Router   // Exported for testing
	ShowHelp     bool             // Exported for testing
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
func initialModelWithRouter(dataSource api.KafkaDataSource) Model {
	r := router.NewRouter(dataSource)
	helpSystem := core.NewHelpSystem()
	focusManager := core.NewFocusManager()

	return Model{
		dataSource:   dataSource,
		Router:       r,
		ShowHelp:     false,
		HelpSystem:   helpSystem,
		FocusManager: focusManager,
	}
}

func (m Model) Init() tea.Cmd {
	return m.Router.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.Router.SetDimensions(msg.Width, msg.Height)
		m.HelpSystem.SetDimensions(msg.Width, msg.Height)

	case tea.KeyMsg:
		// Handle focus management first (if not in help mode)
		if !m.ShowHelp {
			if cmd := m.FocusManager.HandleKeyMsg(msg); cmd != nil {
				return m, cmd
			}
		}

		// Handle global key bindings
		switch {
		case key.Matches(msg, core.DefaultGlobalKeys.Help):
			m.ShowHelp = !m.ShowHelp
			m.HelpSystem.Toggle()
			// Update help system with current page
			if currentPage := m.Router.GetCurrentPage(); currentPage != nil {
				m.HelpSystem.SetCurrentPage(currentPage)
			}
			return m, nil
		case key.Matches(msg, core.DefaultGlobalKeys.Quit):
			return m, tea.Quit
		case key.Matches(msg, core.DefaultGlobalKeys.Back):
			if !m.ShowHelp {
				return m, m.Router.Back()
			}
			// Close help if it's open
			m.ShowHelp = false
			m.HelpSystem.Hide()
			return m, nil
		}
	}

	// Handle router updates if not in help mode
	if !m.ShowHelp {
		_, cmd := m.Router.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.ShowHelp {
		return m.HelpSystem.Render()
	}

	return m.Router.View()
}

// NewUIModel creates a new UI model using router-based navigation
func NewUIModel(dataSource api.KafkaDataSource) Model {
	return initialModelWithRouter(dataSource)
}

// NewUIModelWithRouter creates a new UI model using router-based navigation
func NewUIModelWithRouter(dataSource api.KafkaDataSource) Model {
	return initialModelWithRouter(dataSource)
}
