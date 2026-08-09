// Package tabstrip renders a tab bar whose tabs are click and hover targets.
//
// It exists so every tabbed screen gets the same gesture behaviour from one
// place: the controls spec requires clicking a tab to activate it and hovering
// one to highlight it, and four screens were each drawing their own bar with no
// zones at all.
package tabstrip

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"github.com/Benny93/kafui/pkg/ui/styles"
)

var (
	activeStyle = lipgloss.NewStyle().
			Foreground(styles.BgBase).Background(styles.Primary).Bold(true).Padding(0, 1)
	inactiveStyle = lipgloss.NewStyle().
			Foreground(styles.FgMuted).Padding(0, 1)
	hoverStyle = lipgloss.NewStyle().
			Foreground(styles.FgBase).Background(styles.BgSubtle).Padding(0, 1)
)

// Model is a tab bar. The zero value is unusable; call New.
type Model struct {
	// id namespaces this strip's zones so two strips on one screen do not
	// answer for each other's clicks.
	id      string
	titles  []string
	active  int
	hovered int
}

// New creates a tab strip. id must be unique within a screen.
func New(id string, titles []string) *Model {
	return &Model{id: id, titles: titles, hovered: -1}
}

// SetTitles replaces the tab labels, keeping the active index in range.
func (m *Model) SetTitles(titles []string) {
	m.titles = titles
	if m.active >= len(titles) {
		m.active = 0
	}
}

// Active returns the active tab index.
func (m *Model) Active() int { return m.active }

// SetActive selects a tab, ignoring an out-of-range index.
func (m *Model) SetActive(i int) {
	if i >= 0 && i < len(m.titles) {
		m.active = i
	}
}

func (m *Model) zoneID(i int) string { return fmt.Sprintf("%s-tab-%d", m.id, i) }

// HandleMouse applies a mouse event to the strip. It returns the tab the user
// clicked and true when the event was a click on a tab; hover is applied as a
// side effect and reports false, because hovering activates nothing.
func (m *Model) HandleMouse(msg tea.MouseMsg) (clicked int, ok bool) {
	over := m.tabAt(msg)

	if msg.Button == tea.MouseButtonNone && msg.Action == tea.MouseActionMotion {
		m.hovered = over
		return 0, false
	}
	if over >= 0 && msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionRelease {
		m.active = over
		return over, true
	}
	return 0, false
}

func (m *Model) tabAt(msg tea.MouseMsg) int {
	for i := range m.titles {
		if z := zone.Get(m.zoneID(i)); z != nil && z.InBounds(msg) {
			return i
		}
	}
	return -1
}

// View renders the bar. Each tab carries its ordinal, which is also the key
// that selects it.
func (m *Model) View() string {
	cells := make([]string, 0, len(m.titles))
	for i, t := range m.titles {
		label := fmt.Sprintf("%d %s", i+1, t)
		var rendered string
		switch {
		case i == m.active:
			rendered = activeStyle.Render(label)
		case i == m.hovered:
			rendered = hoverStyle.Render(label)
		default:
			rendered = inactiveStyle.Render(label)
		}
		cells = append(cells, mark(m.zoneID(i), rendered))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

// mark is zone.Mark guarded against there being no global manager, so a View is
// safe to call from a unit test.
func mark(id, s string) (out string) {
	defer func() {
		if recover() != nil {
			out = s
		}
	}()
	return zone.Mark(id, s)
}
