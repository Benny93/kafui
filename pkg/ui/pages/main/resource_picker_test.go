package mainpage

import (
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/keys"
	tea "github.com/charmbracelet/bubbletea"
)

func newPickerProvider(t *testing.T) *KafuiContentProvider {
	t.Helper()
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	return NewKafuiContentProvider(ds)
}

// The bespoke `:` resource picker is superseded by the shell's command palette.
// This screen exports the resource vocabulary; the SHELL turns it into palette
// entries, so switching resource works from every screen rather than only from
// the list you are already on.
func TestPaletteResourcesCoverEveryResource(t *testing.T) {
	names := map[string]bool{}
	for _, rt := range PaletteResources() {
		names[rt.String()] = true
	}
	for _, want := range []string{"topics", "consumer-groups", "brokers", "acls", "quotas", "schemas"} {
		if !names[want] {
			t.Errorf("the palette is missing a destination for %q", want)
		}
	}
}

func TestResourceChoiceFilter(t *testing.T) {
	k := newPickerProvider(t)
	all := k.matchedResourceChoices("")
	if len(all) == 0 {
		t.Fatal("expected resource choices")
	}
	if got := k.matchedResourceChoices("top"); len(got) == 0 || len(got) >= len(all) {
		t.Fatalf("expected 'top' to narrow %d choices, got %d", len(all), len(got))
	}
}

// The screen must not treat `:` itself as input any more — the palette is the
// shell's overlay, so the page stays in Normal mode while it is open.
func TestColonIsNoLongerPageInputMode(t *testing.T) {
	k := newPickerProvider(t)
	k.HandleContentUpdate(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	if k.IsInputMode() {
		t.Fatal("the page must not enter input mode for the palette key")
	}
}

// Regression for the defect the controls spec was written against: a key that
// is not in the registry must do nothing rather than fall into a screen-local
// switch. `t` used to toggle the sidebar and open topic analysis at once.
func TestUnboundKeysAreInert(t *testing.T) {
	for _, k := range []string{"t", "T", "C", "K", "v", "z"} {
		if _, bound := keys.Default.Resolve(keys.ScopeList, k); bound {
			t.Errorf("key %q should not be bound on a list screen", k)
		}
	}
}
