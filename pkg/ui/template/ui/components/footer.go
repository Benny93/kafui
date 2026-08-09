package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"github.com/Benny93/kafui/pkg/ui/styles"
)

// HintZoneID is the bubblezone id of the nth hint in the bar. Every hint is a
// click target: the controls spec requires that clicking a hint runs its
// action, exactly as pressing its key would.
func HintZoneID(i int) string { return fmt.Sprintf("hint-%d", i) }

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

var (
	hintKeyStyle  = lipgloss.NewStyle().Foreground(styles.FgMuted)
	hintDescStyle = lipgloss.NewStyle().Foreground(styles.FgSubtle)
	hintSepStyle  = lipgloss.NewStyle().Foreground(styles.FgSubtle)
)

// visibleHints is the binding list currently rendered, in order, so a click can
// be mapped back to the binding it landed on.
func (f *Footer) visibleHints() []key.Binding {
	if f.keyMap == nil {
		return nil
	}
	if f.help.ShowAll {
		var out []key.Binding
		for _, col := range f.keyMap.FullHelp() {
			out = append(out, col...)
		}
		return out
	}
	return f.keyMap.ShortHelp()
}

// View implements the Component interface. It renders the hints itself rather
// than delegating to bubbles/help so each one can be zone-marked.
func (f *Footer) View() string {
	hints := f.visibleHints()
	if len(hints) == 0 {
		return ""
	}

	var parts []string
	for i, b := range hints {
		h := b.Help()
		if h.Key == "" {
			continue
		}
		label := hintKeyStyle.Render(h.Key) + " " + hintDescStyle.Render(h.Desc)
		parts = append(parts, mark(HintZoneID(i), label))
	}
	return strings.Join(parts, hintSepStyle.Render(" • "))
}

// ClickedHint returns the binding a mouse event landed on, or false when the
// event is not over a hint.
func (f *Footer) ClickedHint(msg tea.MouseMsg) (key.Binding, bool) {
	hints := f.visibleHints()
	for i := range hints {
		if z := zone.Get(HintZoneID(i)); z != nil && z.InBounds(msg) {
			return hints[i], true
		}
	}
	return key.Binding{}, false
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
