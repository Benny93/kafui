package components

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// testKeyMap implements help.KeyMap for testing
type testKeyMap struct {
	Up   key.Binding
	Down key.Binding
	Quit key.Binding
}

func (t testKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{t.Up, t.Down, t.Quit}
}

func (t testKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{t.Up, t.Down},
		{t.Quit},
	}
}

func TestFooterComponent(t *testing.T) {
	// Create a new footer
	footer := NewFooter()
	
	// Test initial state
	assert.Equal(t, 0, footer.width)
	assert.Equal(t, 0, footer.height)
	
	// Test Init
	cmd := footer.Init()
	assert.Nil(t, cmd)
	
	// Test SetSize
	cmd = footer.SetSize(80, 1)
	assert.Nil(t, cmd)
	assert.Equal(t, 80, footer.width)
	assert.Equal(t, 1, footer.height)
	
	// Test GetSize
	width, height := footer.GetSize()
	assert.Equal(t, 80, width)
	assert.Equal(t, 1, height)
	
	// Test View without keymap (should be empty)
	view := footer.View()
	assert.Equal(t, "", view)
	
	// Create a test keymap
	testMap := testKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
	
	// Set the keymap
	footer.SetKeyMap(testMap)
	
	// Test View with keymap (should show help)
	view = footer.View()
	assert.NotEqual(t, "", view)
	assert.Contains(t, view, "↑/k")
	assert.Contains(t, view, "move up")
	assert.Contains(t, view, "↓/j")
	assert.Contains(t, view, "move down")
	assert.Contains(t, view, "q")
	assert.Contains(t, view, "quit")
	
	// Test Update with WindowSizeMsg
	newFooter, cmd := footer.Update(tea.WindowSizeMsg{Width: 100, Height: 2})
	updatedFooter := newFooter.(*Footer)
	assert.Nil(t, cmd)
	assert.Equal(t, 100, updatedFooter.width)
	assert.Equal(t, 2, updatedFooter.height)
	
	// Test SetCompactMode
	cmd = footer.SetCompactMode(true)
	assert.Nil(t, cmd)
	
	// Test ToggleShowAll
	originalShowAll := footer.help.ShowAll
	footer.ToggleShowAll()
	assert.Equal(t, !originalShowAll, footer.help.ShowAll)
	footer.ToggleShowAll()
	assert.Equal(t, originalShowAll, footer.help.ShowAll)
}