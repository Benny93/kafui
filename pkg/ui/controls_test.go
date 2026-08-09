package ui

import (
	"strings"
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	tea "github.com/charmbracelet/bubbletea"
)

func newShell(t *testing.T) *Model {
	t.Helper()
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	m := initialModelWithRouter(ds)
	m.Init()
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 44})
	return m
}

func typeInto(m *Model, s string) {
	for _, r := range s {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

// The bug the ksqlDB demo caught: the palette resolved keys against the OVERLAY
// scope, which binds k/j/y/n/d/e/r. Typing "ksqlDB" hit `k` (up-alias, so the
// filter stayed empty) and then `s` fired a mnemonic, closing the palette — and
// the remaining "q" reached the shell and quit the application.
func TestPaletteAcceptsEveryLetterIncludingNavigationAliases(t *testing.T) {
	for _, query := range []string{"ksqlDB", "brokers", "quotas", "consumer-groups", "dead"} {
		m := newShell(t)
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
		if !m.palette.Active() {
			t.Fatal("palette did not open on ':'")
		}

		typeInto(m, query)

		if !m.palette.Active() {
			t.Fatalf("palette closed while typing %q — a letter was treated as a command", query)
		}
		if v := m.palette.View(); !strings.Contains(v, query) {
			t.Errorf("typing %q did not reach the filter", query)
		}
	}
}

// A `q` typed into the palette must never quit. That is what turned a demo
// recording into a terminated process.
func TestTypingQIntoThePaletteDoesNotQuit(t *testing.T) {
	m := newShell(t)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil {
		if msg := cmd(); msg != nil {
			if _, isQuit := msg.(tea.QuitMsg); isQuit {
				t.Fatal("typing 'q' into the palette quit the application")
			}
		}
	}
	if !m.palette.Active() {
		t.Fatal("palette closed on 'q'")
	}
}

// The actions menu keeps its mnemonics: it is a pick-one-of-N surface, not a
// search box, so a first letter runs an entry.
func TestActionsMenuKeepsMnemonics(t *testing.T) {
	m := newShell(t)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if !m.actions.Active() {
		t.Fatal("actions menu did not open on 'a'")
	}
	entry, ok := m.actions.SelectedEntry()
	if !ok {
		t.Fatal("actions menu is empty")
	}
	if entry.Mnemonic == 0 {
		t.Error("actions-menu entries must carry mnemonics")
	}
}

// Esc closes either surface without acting.
func TestEscClosesTheDiscoverySurfaces(t *testing.T) {
	m := newShell(t)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.palette.Active() {
		t.Error("esc did not close the palette")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.actions.Active() {
		t.Error("esc did not close the actions menu")
	}
}

// Bug 10 from the demo review: the palette asked only the CURRENT page for
// destinations, so once you navigated to metrics or ksqlDB there was no
// palette route back to a resource list.
func TestPaletteOffersResourcesFromEveryScreen(t *testing.T) {
	m := newShell(t)
	m.Router.NavigateTo("metrics", nil)

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	typeInto(m, "topics")

	if !m.palette.Active() {
		t.Fatal("palette closed while typing")
	}
	if v := m.palette.View(); !strings.Contains(v, "topics") {
		t.Errorf("no route back to the topics list from the metrics page; view=%q", v)
	}
}
