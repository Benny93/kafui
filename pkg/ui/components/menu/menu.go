// Package menu provides the filterable overlay list behind the two discovery
// surfaces the controls spec requires: the command palette (`:`) and the
// contextual actions menu (`a` / right-click). They differ only in what fills
// them, so they share one implementation.
package menu

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	zone "github.com/lrstanley/bubblezone"

	"github.com/Benny93/kafui/pkg/ui/keys"
	"github.com/Benny93/kafui/pkg/ui/styles"
)

// menuZoneID is the bubblezone id of the whole overlay, used for hit-testing
// hover and clicks against its rows.
const menuZoneID = "menu-overlay"

// Entry is one row of a menu. Disabled entries are still listed — the spec
// requires the user to see that an action exists and why they cannot use it.
type Entry struct {
	// Label is what the user reads, e.g. "Delete topic".
	Label string
	// Key is the direct binding shown alongside the label, e.g. "d". Empty when
	// the action has no direct binding, which is how most entries look.
	Key string
	// Desc is optional secondary text.
	Desc string
	// Disabled marks an action the user may not perform right now.
	Disabled bool
	// Reason explains a disabled entry: a permission denial, read-only mode, or
	// a missing capability. Required when Disabled is set.
	Reason string
	// Destructive renders the entry in the danger style.
	Destructive bool
	// Mnemonic is the letter that runs this entry directly while the menu is
	// open. Assigned automatically from the label when the menu opens; the
	// first unused letter wins.
	Mnemonic rune
	// Run builds the command to execute, and is called ONLY when the entry is
	// chosen. It is a builder rather than a tea.Cmd because building an entry
	// must never have a side effect: the menu is constructed in order to be
	// displayed, and an eager `Run: k.openCreateForm()` would open the form
	// behind the menu on every render.
	Run func() tea.Cmd
}

// Model is the overlay itself.
type Model struct {
	// mnemonics is on for the actions menu, where the user picks one of a
	// handful of named actions, and off for the command palette, where the
	// user types to search. A palette with mnemonics is unusable: the first
	// letter of what you are typing runs something.
	mnemonics bool
	title     string
	entries   []Entry
	// shown is entries surviving the filter, as indices into entries.
	shown  []int
	filter string
	cursor int
	active bool
	width  int
	height int
}

func New() *Model { return &Model{} }

// Open shows the contextual actions menu: entries carry mnemonics, so `a` then
// a letter runs an action in two keystrokes.
func (m *Model) Open(title string, entries []Entry) {
	m.mnemonics = true
	m.open(title, assignMnemonics(entries))
}

// OpenFilter shows a search-first menu — the command palette. Every printable
// key filters; there are no mnemonics, because you cannot type "ksqlDB" on a
// surface where `k` and `s` run things.
func (m *Model) OpenFilter(title string, entries []Entry) {
	m.mnemonics = false
	m.open(title, entries)
}

func (m *Model) open(title string, entries []Entry) {
	m.title = title
	m.entries = entries
	m.filter = ""
	m.cursor = 0
	m.active = true
	m.refilter()
}

// Close hides the menu without running anything.
func (m *Model) Close() { m.active = false }

// Active reports whether the menu is capturing input.
func (m *Model) Active() bool { return m.active }

// SetDimensions sizes the overlay.
func (m *Model) SetDimensions(w, h int) { m.width, m.height = w, h }

// Update handles input while the menu is open. It returns true when it consumed
// the message, which is how the caller honours the overlay's input precedence.
func (m *Model) Update(msg tea.Msg) (tea.Cmd, bool) {
	if !m.active {
		return nil, false
	}
	if mouse, isMouse := msg.(tea.MouseMsg); isMouse {
		// Overlays swallow every mouse event, so clicks cannot reach the screen
		// behind them. Within the menu, hover highlights and a click runs.
		if row, over := m.rowAt(mouse); over {
			m.cursor = row
			if mouse.Button == tea.MouseButtonLeft && mouse.Action == tea.MouseActionRelease {
				return m.run(), true
			}
		}
		return nil, true
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil, false
	}

	// Resolve against the TEXT-ENTRY scope, not the overlay scope. Text entry
	// binds only non-printable keys, so every letter reaches the filter. The
	// overlay scope binds k/j/y/n/d/e/r, which made it impossible to type any
	// word containing them — the palette closed halfway through "ksqlDB".
	action, bound := keys.Default.Resolve(keys.ScopeTextEntry, key.String())
	if bound {
		switch action {
		case keys.ActionCancel:
			m.Close()
			return nil, true
		case keys.ActionUp:
			m.move(-1)
			return nil, true
		case keys.ActionDown:
			m.move(1)
			return nil, true
		case keys.ActionCommit:
			return m.run(), true
		case keys.ActionClearField:
			m.filter = ""
			m.refilter()
			return nil, true
		}
	}

	switch key.Type {
	case tea.KeyBackspace:
		if m.filter != "" {
			m.filter = m.filter[:len(m.filter)-1]
			m.refilter()
		}
		return nil, true
	case tea.KeyRunes, tea.KeySpace:
		// In the actions menu, a letter typed first is a mnemonic: `a` then `d`
		// deletes. Once the user starts filtering, letters extend the filter —
		// otherwise typing a name would fire an action mid-word.
		if m.mnemonics && m.filter == "" && len(key.Runes) == 1 {
			if cmd, ok := m.runMnemonic(key.Runes[0]); ok {
				return cmd, true
			}
		}
		m.filter += key.String()
		m.refilter()
		return nil, true
	}
	return nil, true
}

func (m *Model) move(d int) {
	if len(m.shown) == 0 {
		return
	}
	m.cursor = (m.cursor + d + len(m.shown)) % len(m.shown)
}

func (m *Model) run() tea.Cmd {
	if len(m.shown) == 0 {
		return nil
	}
	e := m.entries[m.shown[m.cursor]]
	if e.Disabled {
		// Refusing silently would be a no-op the user cannot explain, so the
		// menu stays open with the reason visible.
		return nil
	}
	m.Close()
	if e.Run == nil {
		return nil
	}
	return e.Run()
}

// assignMnemonics gives each entry the first letter of its label that no
// earlier entry claimed, so an entry can be run with two keystrokes (`a` then
// the letter) without anyone having to memorise a table.
func assignMnemonics(entries []Entry) []Entry {
	taken := map[rune]bool{}
	out := make([]Entry, len(entries))
	copy(out, entries)
	for i := range out {
		if out[i].Disabled {
			continue
		}
		for _, r := range strings.ToLower(out[i].Label) {
			if r < 'a' || r > 'z' || taken[r] {
				continue
			}
			taken[r] = true
			out[i].Mnemonic = r
			break
		}
	}
	return out
}

// runMnemonic runs the entry whose mnemonic is r, reporting whether one matched.
func (m *Model) runMnemonic(r rune) (tea.Cmd, bool) {
	for _, i := range m.shown {
		e := m.entries[i]
		if e.Mnemonic == r && !e.Disabled {
			m.Close()
			if e.Run == nil {
				return nil, true
			}
			return e.Run(), true
		}
	}
	return nil, false
}

// refilter recomputes the visible set. Matching is substring, case-insensitive,
// over the label — enough to find an entry by typing part of its name.
func (m *Model) refilter() {
	m.shown = m.shown[:0]
	q := strings.ToLower(m.filter)
	for i, e := range m.entries {
		if q == "" || strings.Contains(strings.ToLower(e.Label), q) {
			m.shown = append(m.shown, i)
		}
	}
	if m.cursor >= len(m.shown) {
		m.cursor = 0
	}
}

// markZone is zone.Mark guarded against there being no global manager, which is
// the case in unit tests and before the program starts. An unmarked overlay is
// simply not click-testable, which is the right degradation.
func markZone(id, s string) (out string) {
	defer func() {
		if recover() != nil {
			out = s
		}
	}()
	return zone.Mark(id, s)
}

// rowAt maps a mouse position to a visible row index.
func (m *Model) rowAt(msg tea.MouseMsg) (int, bool) {
	z := zone.Get(menuZoneID)
	if z == nil || !z.InBounds(msg) {
		return 0, false
	}

	_, relY := z.Pos(msg)
	// Frame border + title + blank line precede the first entry.
	row := relY - 3
	if row < 0 || row >= len(m.shown) {
		return 0, false
	}
	return row, true
}

// SelectedEntry exposes the highlighted entry, for tests.
func (m *Model) SelectedEntry() (Entry, bool) {
	if !m.active || len(m.shown) == 0 {
		return Entry{}, false
	}
	return m.entries[m.shown[m.cursor]], true
}

var (
	frameStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Primary).
			Padding(0, 1)
	titleStyle    = lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	filterStyle   = lipgloss.NewStyle().Foreground(styles.FgMuted)
	keyStyle      = lipgloss.NewStyle().Foreground(styles.FgSubtle)
	selStyle      = lipgloss.NewStyle().Background(styles.Primary).Foreground(styles.BgBase).Bold(true)
	disabledStyle = lipgloss.NewStyle().Foreground(styles.FgSubtle).Italic(true)
	dangerStyle   = lipgloss.NewStyle().Foreground(styles.Error)
	reasonStyle   = lipgloss.NewStyle().Foreground(styles.Warning).Italic(true)
	mnemonicStyle = lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
)

// View renders the overlay. An empty string means it is closed.
//
// Every row is rendered to exactly innerWidth cells, selected or not. Sizing
// the selected row separately is what made the highlight wrap onto a second
// line, overrun the right border, and misalign the key column against the
// other rows.
func (m *Model) View() string {
	if !m.active {
		return ""
	}
	inner := m.innerWidth()
	maxRows := m.height - 8
	if maxRows < 3 {
		maxRows = 3
	}

	var b strings.Builder
	title := titleStyle.Render(m.title) + "  " + filterStyle.Render("/"+m.filter+"▏")
	b.WriteString(pad(title, inner))
	b.WriteString("\n\n")

	if len(m.shown) == 0 {
		b.WriteString(pad(disabledStyle.Render("no matches"), inner))
	}

	// Scroll the window so the cursor stays visible in a long list.
	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	for i := start; i < len(m.shown) && i < start+maxRows; i++ {
		b.WriteString(m.renderRow(m.entries[m.shown[i]], inner, i == m.cursor))
		b.WriteString("\n")
	}

	return markZone(menuZoneID, frameStyle.Render(strings.TrimRight(b.String(), "\n")))
}

// innerWidth is the number of cells a row may occupy inside the frame. The
// frame adds its own border and padding around it, so the box is never sized
// from a row that was built to a different width.
func (m *Model) innerWidth() int {
	w := m.width * 2 / 3
	if w < 40 {
		w = 40
	}
	// Border (2) + padding (2) live outside the row.
	if m.width > 0 && w > m.width-6 {
		w = m.width - 6
	}
	if w < 10 {
		w = 10
	}
	return w
}

// renderRow lays out one entry: a mnemonic gutter, the label, and — right
// aligned — either the entry's key or, when it is disabled, the reason. A
// disabled entry states its reason on every row rather than only when the
// cursor happens to be on it, which is what the controls spec asks for.
func (m *Model) renderRow(e Entry, inner int, selected bool) string {
	mnemonic := "  "
	if e.Mnemonic != 0 && !e.Disabled {
		mnemonic = string(e.Mnemonic) + " "
	}

	right := e.Key
	if e.Disabled && e.Reason != "" {
		right = e.Reason
	}
	// The right column takes at most half the row, and never so much that the
	// label is squeezed below minLabel. A long reason used to push the row past
	// the frame, which drew the entry over the box's own right border.
	const minLabel = 12
	budget := inner / 2
	if hard := inner - 1 - lipgloss.Width(mnemonic) - minLabel - 1; budget > hard {
		budget = hard
	}
	if budget < 0 {
		budget = 0
	}
	if lipgloss.Width(right) > budget {
		right = ansi.Truncate(right, budget, "…")
	}

	label := e.Label
	room := inner - 1 - lipgloss.Width(mnemonic) - lipgloss.Width(right) - 1
	if room < 1 {
		room = 1
	}
	if lipgloss.Width(label) > room {
		label = ansi.Truncate(label, room, "…")
	}

	gap := inner - 1 - lipgloss.Width(mnemonic) - lipgloss.Width(label) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	if selected {
		// One background across the whole row, so the highlight is a clean bar.
		return selStyle.Render(" " + mnemonic + label + strings.Repeat(" ", gap) + right)
	}

	styledMnemonic := mnemonic
	if e.Mnemonic != 0 && !e.Disabled {
		styledMnemonic = mnemonicStyle.Render(string(e.Mnemonic)) + " "
	}
	styledLabel := label
	switch {
	case e.Disabled:
		styledLabel = disabledStyle.Render(label)
	case e.Destructive:
		styledLabel = dangerStyle.Render(label)
	}
	styledRight := keyStyle.Render(right)
	if e.Disabled && e.Reason != "" {
		styledRight = reasonStyle.Render(right)
	}
	return " " + styledMnemonic + styledLabel + strings.Repeat(" ", gap) + styledRight
}

// pad extends a styled string to exactly n cells so every line of the box is
// the same width and the frame cannot size itself from the longest one.
func pad(s string, n int) string {
	if d := n - lipgloss.Width(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}
