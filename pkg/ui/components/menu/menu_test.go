package menu

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func entries() []Entry {
	ran := ""
	return []Entry{
		{Label: "Open topic", Run: func() tea.Cmd { ran = "open"; _ = ran; return nil }},
		{Label: "Delete topic", Destructive: true, Run: func() tea.Cmd { return tea.Quit }},
		{Label: "Purge messages", Disabled: true, Reason: "read-only"},
	}
}

func TestMnemonicsAreUniqueAndSkipDisabled(t *testing.T) {
	m := New()
	m.Open("Actions", entries())

	seen := map[rune]bool{}
	for _, e := range m.entries {
		if e.Mnemonic == 0 {
			continue
		}
		assert.False(t, seen[e.Mnemonic], "mnemonic %q assigned twice", string(e.Mnemonic))
		seen[e.Mnemonic] = true
	}
	assert.Equal(t, 'o', m.entries[0].Mnemonic, "first free letter of the label wins")
	assert.Equal(t, 'd', m.entries[1].Mnemonic)
	assert.Zero(t, m.entries[2].Mnemonic, "a disabled entry gets no mnemonic")
}

func TestMnemonicRunsTheEntry(t *testing.T) {
	m := New()
	m.Open("Actions", entries())

	cmd, consumed := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	require.True(t, consumed)
	require.NotNil(t, cmd, "pressing a mnemonic runs its entry")
	assert.False(t, m.Active(), "running an entry closes the menu")
}

// Once the user starts filtering, letters must extend the filter rather than
// fire an action — otherwise typing a name would delete something mid-word.
func TestLettersFilterOnceFilteringHasStarted(t *testing.T) {
	m := New()
	m.Open("Actions", entries())

	// 'x' matches no mnemonic, so it starts a filter.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	assert.Equal(t, "x", m.filter)

	cmd, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	assert.Nil(t, cmd, "a letter typed into an active filter must not run an entry")
	assert.Equal(t, "xd", m.filter)
	assert.True(t, m.Active())
}

func TestDisabledEntryDoesNotRunAndKeepsTheMenuOpen(t *testing.T) {
	m := New()
	m.Open("Actions", entries())
	m.cursor = 2 // the disabled entry

	cmd, consumed := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.True(t, consumed)
	assert.Nil(t, cmd)
	assert.True(t, m.Active(), "the menu stays open so the reason stays visible")
}

func TestBuildingEntriesNeverRuns(t *testing.T) {
	calls := 0
	m := New()
	m.Open("Actions", []Entry{{Label: "Danger", Run: func() tea.Cmd { calls++; return nil }}})
	m.View()
	m.View()
	assert.Zero(t, calls, "rendering a menu must not run its entries")
}

// Every row must be exactly the same width, selected or not. Sizing the
// selected row separately is what made the highlight wrap onto a second line
// and overrun the right border.
func TestEveryRowIsTheSameWidth(t *testing.T) {
	m := New()
	m.SetDimensions(120, 30)
	m.Open("Actions", []Entry{
		{Label: "Short", Key: "d"},
		{Label: "A considerably longer label than the others", Key: "ctrl+n"},
		{Label: "No key at all"},
		{Label: "Disabled", Disabled: true, Reason: "not permitted"},
	})

	inner := m.innerWidth()
	for i, e := range m.entries {
		for _, selected := range []bool{false, true} {
			row := m.renderRow(e, inner, selected)
			assert.Equal(t, inner, lipgloss.Width(row),
				"entry %d (selected=%v) is not %d cells wide", i, selected, inner)
			assert.NotContains(t, row, "\n", "a row must never wrap")
		}
	}
}

// A label longer than the row must be truncated, not allowed to push the row
// wider than the frame.
func TestOverlongLabelIsTruncated(t *testing.T) {
	m := New()
	m.SetDimensions(60, 20)
	m.Open("Actions", []Entry{{Label: strings.Repeat("x", 500), Key: "d"}})

	inner := m.innerWidth()
	row := m.renderRow(m.entries[0], inner, false)
	assert.Equal(t, inner, lipgloss.Width(row))
	assert.Contains(t, row, "…", "an over-long label must show it was cut")
	assert.Contains(t, row, "d", "the key must survive a long label")
}

// A disabled entry states its reason on every row, not only when the cursor is
// on it — the whole point of listing it is that the user learns why it is off.
func TestDisabledEntryAlwaysShowsItsReason(t *testing.T) {
	m := New()
	m.SetDimensions(120, 30)
	m.Open("Commands", []Entry{
		{Label: "Go to: ksqlDB", Disabled: true, Reason: "not advertised by this cluster"},
		{Label: "Quit", Key: "q"},
	})
	// Cursor is on the second entry, so the first is not the highlighted row.
	m.cursor = 1
	row := m.renderRow(m.entries[0], m.innerWidth(), false)
	assert.Contains(t, row, "not advertised", "the reason must be visible without hovering")
}

// The whole box must be rectangular: a shorter title or an empty list cannot
// make the frame narrower than its rows.
func TestBoxIsRectangular(t *testing.T) {
	m := New()
	m.SetDimensions(120, 30)
	m.Open("Commands", []Entry{{Label: "One", Key: "1"}, {Label: "Two"}})

	lines := strings.Split(m.View(), "\n")
	want := lipgloss.Width(lines[0])
	for i, l := range lines {
		assert.Equal(t, want, lipgloss.Width(l), "line %d differs in width", i)
	}
}

// A long disabled reason must not push the row past the frame — it used to be
// drawn straight over the box's own right border and into the sidebar.
func TestLongReasonNeverOverflowsTheRow(t *testing.T) {
	m := New()
	m.SetDimensions(120, 30)
	m.Open("Commands", []Entry{{
		Label:    "Go to: cluster setup wizard",
		Disabled: true,
		Reason:   strings.Repeat("set dynamicConfigEnabled: true to edit clusters in-app ", 5),
	}})

	inner := m.innerWidth()
	row := m.renderRow(m.entries[0], inner, false)
	assert.Equal(t, inner, lipgloss.Width(row))
	assert.Contains(t, row, "cluster setup", "the label must survive a long reason")
}
