package components

import tea "github.com/charmbracelet/bubbletea"

// Component defines the basic interface all UI components must implement
type Component interface {
	Init() tea.Cmd
	Update(tea.Msg) (Component, tea.Cmd)
	View() string
}

// Sizeable components can be resized
type Sizeable interface {
	SetSize(width, height int) tea.Cmd
	GetSize() (int, int)
}

// Focusable components can receive focus
type Focusable interface {
	Focus() tea.Cmd
	Blur() tea.Cmd
	IsFocused() bool
}

// Refreshable components can refresh their data
type Refreshable interface {
	Refresh() tea.Cmd
}

// CompactModeToggleable components can switch between compact and full modes
type CompactModeToggleable interface {
	SetCompactMode(compact bool) tea.Cmd
}