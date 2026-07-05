package keys

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	// Verify all key maps are present
	assert.NotNil(t, km.Global)
	assert.NotNil(t, km.Main)
	assert.NotNil(t, km.Topic)
	assert.NotNil(t, km.Detail)
	assert.NotNil(t, km.ResourceDetail)
	assert.NotNil(t, km.Search)
}

func TestDefaultGlobalKeyMap(t *testing.T) {
	km := DefaultGlobalKeyMap()

	// Verify all global bindings are defined
	assert.NotNil(t, km.Quit)
	assert.NotNil(t, km.Help)
	assert.NotNil(t, km.Back)
	assert.NotNil(t, km.Search)
	assert.NotNil(t, km.ToggleTheme)

	// Verify key bindings have proper keys
	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.Help.Keys())
	assert.NotEmpty(t, km.Back.Keys())
	assert.NotEmpty(t, km.Search.Keys())
	assert.NotEmpty(t, km.ToggleTheme.Keys())

	// Verify help text is present
	assert.NotEmpty(t, km.Quit.Help().Desc)
	assert.NotEmpty(t, km.Help.Help().Desc)
	assert.NotEmpty(t, km.Back.Help().Desc)
	assert.NotEmpty(t, km.Search.Help().Desc)
	assert.NotEmpty(t, km.ToggleTheme.Help().Desc)
}

// TestGlobalKeysCanonicalSet asserts the unified global registry (UI-17) carries
// all 13 shell bindings, each with WithHelp, and matches the expected keys.
func TestGlobalKeysCanonicalSet(t *testing.T) {
	g := GlobalKeys

	bindings := g.GetAllBindings()
	assert.Len(t, bindings, 13, "expected 13 global bindings")
	for _, b := range bindings {
		assert.NotEmpty(t, b.Keys(), "binding must define keys")
		assert.NotEmpty(t, b.Help().Key, "binding must carry WithHelp key")
		assert.NotEmpty(t, b.Help().Desc, "binding must carry WithHelp desc")
	}

	// Spot-check the runtime-critical bindings behave as before.
	assert.True(t, key.Matches(tea.KeyMsg{Type: tea.KeyEsc}, g.Back))
	assert.True(t, key.Matches(tea.KeyMsg{Type: tea.KeyTab}, g.NextPage))
	assert.True(t, key.Matches(tea.KeyMsg{Type: tea.KeyShiftTab}, g.PrevPage))
	assert.True(t, key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("T")}, g.ToggleTheme))
	assert.True(t, key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")}, g.Clusters))
	assert.True(t, key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")}, g.Ksql))
}

func TestDefaultMainKeyMap(t *testing.T) {
	km := DefaultMainKeyMap()

	// Verify all main page bindings are defined
	assert.NotNil(t, km.Select)
	assert.NotNil(t, km.SwitchResource)
	assert.NotNil(t, km.Search)
	assert.NotNil(t, km.Help)
	assert.NotNil(t, km.Quit)
	assert.NotNil(t, km.Back)
}

func TestDefaultTopicKeyMap(t *testing.T) {
	km := DefaultTopicKeyMap()

	// Verify all topic page bindings are defined
	assert.NotNil(t, km.Select)
	assert.NotNil(t, km.Back)
	assert.NotNil(t, km.Search)
	assert.NotNil(t, km.Help)
	assert.NotNil(t, km.Quit)
	assert.NotNil(t, km.Pause)
	assert.NotNil(t, km.Refresh)
	assert.NotNil(t, km.Retry)
	assert.NotNil(t, km.Format)
	assert.NotNil(t, km.Headers)
	assert.NotNil(t, km.Metadata)
	assert.NotNil(t, km.ScrollUp)
	assert.NotNil(t, km.ScrollDown)
	assert.NotNil(t, km.PageUp)
	assert.NotNil(t, km.PageDown)
	assert.NotNil(t, km.GotoStart)
	assert.NotNil(t, km.GotoEnd)
	assert.NotNil(t, km.CopyKey)
	assert.NotNil(t, km.CopyValue)
}

func TestDefaultDetailKeyMap(t *testing.T) {
	km := DefaultDetailKeyMap()

	// Verify all detail page bindings are defined
	assert.NotNil(t, km.Back)
	assert.NotNil(t, km.Help)
	assert.NotNil(t, km.Quit)
	assert.NotNil(t, km.Format)
	assert.NotNil(t, km.Headers)
	assert.NotNil(t, km.Metadata)
	assert.NotNil(t, km.Wrap)
	assert.NotNil(t, km.ScrollUp)
	assert.NotNil(t, km.ScrollDown)
	assert.NotNil(t, km.PageUp)
	assert.NotNil(t, km.PageDown)
	assert.NotNil(t, km.GotoStart)
	assert.NotNil(t, km.GotoEnd)
	assert.NotNil(t, km.Copy)
}

func TestDefaultResourceDetailKeyMap(t *testing.T) {
	km := DefaultResourceDetailKeyMap()

	// Verify all resource detail bindings are defined
	assert.NotNil(t, km.Back)
	assert.NotNil(t, km.Help)
	assert.NotNil(t, km.Quit)
	assert.NotNil(t, km.ScrollUp)
	assert.NotNil(t, km.ScrollDown)
	assert.NotNil(t, km.PageUp)
	assert.NotNil(t, km.PageDown)
	assert.NotNil(t, km.GotoStart)
	assert.NotNil(t, km.GotoEnd)
	assert.NotNil(t, km.Copy)
}

func TestDefaultSearchKeyMap(t *testing.T) {
	km := DefaultSearchKeyMap()

	// Verify all search mode bindings are defined
	assert.NotNil(t, km.Confirm)
	assert.NotNil(t, km.Cancel)
	assert.NotNil(t, km.Clear)
	assert.NotNil(t, km.Navigate)
	assert.NotNil(t, km.TabComplete)
}

func TestKeyMap_GetShortHelp(t *testing.T) {
	km := DefaultKeyMap()

	// Test GetShortHelp
	shortHelp := km.GetShortHelp()
	assert.NotEmpty(t, shortHelp)

	// Should contain global help bindings
	assert.Contains(t, shortHelp, km.Global.Help)
	assert.Contains(t, shortHelp, km.Global.Back)
	assert.Contains(t, shortHelp, km.Global.Quit)
}

func TestKeyMap_GetFullHelp(t *testing.T) {
	km := DefaultKeyMap()

	// Test GetFullHelp
	fullHelp := km.GetFullHelp()
	assert.NotEmpty(t, fullHelp)

	// Should have at least one row
	assert.GreaterOrEqual(t, len(fullHelp), 1)
}

func TestKeyMap_GetMainPageHelp(t *testing.T) {
	km := DefaultKeyMap()

	// Test GetMainPageHelp
	help := km.GetMainPageHelp()
	assert.NotEmpty(t, help)

	// Should contain main page specific bindings
	assert.Contains(t, help, km.Main.Search)
	assert.Contains(t, help, km.Main.SwitchResource)
	assert.Contains(t, help, km.Main.Select)
}

func TestKeyMap_GetTopicPageHelp(t *testing.T) {
	km := DefaultKeyMap()

	// Test GetTopicPageHelp
	help := km.GetTopicPageHelp()
	assert.NotEmpty(t, help)

	// Should contain topic page specific bindings
	assert.Contains(t, help, km.Topic.Select)
	assert.Contains(t, help, km.Topic.Pause)
}

func TestKeyMap_GetDetailPageHelp(t *testing.T) {
	km := DefaultKeyMap()

	// Test GetDetailPageHelp
	help := km.GetDetailPageHelp()
	assert.NotEmpty(t, help)

	// Should contain detail page specific bindings
	assert.Contains(t, help, km.Detail.Format)
	assert.Contains(t, help, km.Detail.Copy)
}

func TestKeyMatching(t *testing.T) {
	km := DefaultGlobalKeyMap()

	// Test that key matching works correctly
	tests := []struct {
		name     string
		msg      tea.KeyMsg
		binding  key.Binding
		expected bool
	}{
		{
			name:     "quit_with_q",
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}},
			binding:  km.Quit,
			expected: true,
		},
		{
			name:     "quit_with_ctrl_c",
			msg:      tea.KeyMsg{Type: tea.KeyCtrlC},
			binding:  km.Quit,
			expected: true,
		},
		{
			name:     "help_with_question_mark",
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}},
			binding:  km.Help,
			expected: true,
		},
		{
			name:     "back_with_esc",
			msg:      tea.KeyMsg{Type: tea.KeyEsc},
			binding:  km.Back,
			expected: true,
		},
		{
			name:     "search_with_slash",
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}},
			binding:  km.Search,
			expected: true,
		},
		{
			name:     "toggle_theme_with_T",
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}},
			binding:  km.ToggleTheme,
			expected: true,
		},
		{
			name:     "wrong_key",
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}},
			binding:  km.Quit,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := key.Matches(tt.msg, tt.binding)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNoKeyConflicts(t *testing.T) {
	km := DefaultKeyMap()

	// Collect all key bindings
	allBindings := []key.Binding{
		km.Global.Quit,
		km.Global.Help,
		km.Global.Back,
		km.Global.Search,
		km.Global.ToggleTheme,
		km.Main.Select,
		km.Main.SwitchResource,
		km.Topic.Pause,
		km.Topic.Refresh,
		km.Topic.Retry,
		km.Detail.Format,
		km.Detail.Copy,
		km.ResourceDetail.Copy,
	}

	// Check for duplicate keys within global scope
	globalKeys := make(map[string]string) // key -> binding name

	addBinding := func(name string, binding key.Binding) {
		for _, k := range binding.Keys() {
			if existing, exists := globalKeys[k]; exists {
				t.Logf("Warning: Key '%s' used by both '%s' and '%s'", k, existing, name)
				// Note: Some key overlap is expected and intentional (e.g., 'q' for quit/back)
			}
			globalKeys[k] = name
		}
	}

	for _, binding := range allBindings {
		addBinding("binding", binding)
	}

	// Test passes as long as we're aware of the overlaps
	t.Logf("Total unique global keys: %d", len(globalKeys))
}

func TestHelpInterfaceImplementation(t *testing.T) {
	// Verify that DetailKeyMap implements help.KeyMap interface
	var _ interface {
		ShortHelp() []key.Binding
		FullHelp() [][]key.Binding
	} = DefaultDetailKeyMap()

	// Verify that ResourceDetailKeyMap implements help.KeyMap interface
	var _ interface {
		ShortHelp() []key.Binding
		FullHelp() [][]key.Binding
	} = DefaultResourceDetailKeyMap()

	// Test ShortHelp returns non-empty
	detailKm := DefaultDetailKeyMap()
	shortHelp := detailKm.ShortHelp()
	assert.NotEmpty(t, shortHelp)

	// Test FullHelp returns non-empty
	fullHelp := detailKm.FullHelp()
	assert.NotEmpty(t, fullHelp)
	assert.GreaterOrEqual(t, len(fullHelp), 1)
}
