package core

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Page represents a UI page component
type Page interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (tea.Model, tea.Cmd)
	View() string
	SetDimensions(width, height int)
	GetID() string

	// Navigation methods for enhanced routing
	GetTitle() string
	GetHelp() []key.Binding
	HandleNavigation(msg tea.Msg) (Page, tea.Cmd)
	OnFocus() tea.Cmd
	OnBlur() tea.Cmd
}

// Disposer is implemented by pages that hold resources beyond their own memory,
// such as a running consumer. The router calls Dispose when it evicts the page
// (it left the navigation history, or the active cluster changed), after
// OnBlur. A disposed page is never shown again.
type Disposer interface {
	Dispose()
}

// StatefulPage extends Page with state management capabilities
type StatefulPage interface {
	Page
	GetState() UIState
	GetFocusState() FocusState
	SetState(UIState)
	SetFocusState(FocusState)
}

// KeyHandler handles keyboard input
type KeyHandler interface {
	HandleKey(key tea.KeyMsg) tea.Cmd
	GetKeyBindings() []key.Binding
}

// ViewRenderer handles view rendering
type ViewRenderer interface {
	Render(model interface{}) string
	SetTheme(theme Theme)
	SetDimensions(width, height int)
}

// DataLoader handles data loading operations
type DataLoader interface {
	LoadData() tea.Cmd
	RefreshData() tea.Cmd
}

// EventHandler handles different types of events
type EventHandler interface {
	HandleKeyEvent(msg tea.KeyMsg) tea.Cmd
	HandleDataEvent(msg DataLoadedMsg) tea.Cmd
	HandleErrorEvent(msg DataErrorMsg) tea.Cmd
	HandleTimerEvent(msg TimerTickMsg) tea.Cmd
}

// ResourceManager manages resource operations
type ResourceManager interface {
	LoadResources() tea.Cmd
	GetCurrentResourceType() string
	SwitchResource(resourceType string) tea.Cmd
}

// Theme represents visual styling configuration
type Theme struct {
	Primary   string
	Secondary string
	Accent    string
	Error     string
	Success   string
	Warning   string
	Info      string
}

// Dimensions represents width and height
type Dimensions struct {
	Width  int
	Height int
}

// KeyBinding represents a key binding with help text
type KeyBinding struct {
	Key  key.Binding
	Help string
}

// StatusType represents different status message types
type StatusType int

const (
	StatusInfo StatusType = iota
	StatusError
	StatusSuccess
	StatusWarning
)

// SearchMode represents different search modes
type SearchMode int

const (
	SearchModeSimple SearchMode = iota
	SearchModeAdvanced
	SearchModeRegex
	SearchModeFuzzy
)
