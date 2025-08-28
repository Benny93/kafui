package components

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
)

// Footer represents a reusable footer component
type Footer struct {
	width       int
	height      int
	help        help.Model
	keyMap      help.KeyMap
	compactMode bool
}

// NewFooter creates a new footer component
func NewFooter() *Footer {
	return &Footer{
		help: help.New(),
	}
}

// Init implements the Component interface
func (f *Footer) Init() tea.Cmd {
	return nil
}

// Update implements the Component interface
func (f *Footer) Update(msg tea.Msg) (Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width = msg.Width
		f.height = msg.Height
		f.help.Width = msg.Width
	}

	return f, nil
}

// View implements the Component interface
func (f *Footer) View() string {
	if f.keyMap == nil {
		return ""
	}

	return f.help.View(f.keyMap)
}

// SetSize implements the Sizeable interface
func (f *Footer) SetSize(width, height int) tea.Cmd {
	f.width = width
	f.height = height
	f.help.Width = width
	return nil
}

// GetSize implements the Sizeable interface
func (f *Footer) GetSize() (int, int) {
	return f.width, f.height
}

// SetCompactMode implements the CompactModeToggleable interface
func (f *Footer) SetCompactMode(compact bool) tea.Cmd {
	f.compactMode = compact
	return nil
}

// SetKeyMap sets the key map for the footer help
func (f *Footer) SetKeyMap(keyMap help.KeyMap) {
	f.keyMap = keyMap
}

// SetShowAll toggles between short and full help
func (f *Footer) SetShowAll(showAll bool) {
	f.help.ShowAll = showAll
}

// ToggleShowAll toggles the help view between short and full
func (f *Footer) ToggleShowAll() {
	f.help.ShowAll = !f.help.ShowAll
}
