package mainpage

import (
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	tea "github.com/charmbracelet/bubbletea"
)

func newPickerProvider(t *testing.T) *KafuiContentProvider {
	t.Helper()
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	return NewKafuiContentProvider(ds)
}

func TestResourcePicker_OpenAndCancel(t *testing.T) {
	k := newPickerProvider(t)
	k.HandleContentUpdate(StartResourceSwitchingMsg{})
	if !k.resourcePickerMode {
		t.Fatal("expected picker to open on StartResourceSwitchingMsg")
	}
	if !k.IsInputMode() {
		t.Fatal("picker should count as input mode")
	}
	// Esc cancels.
	k.HandleContentUpdate(tea.KeyMsg{Type: tea.KeyEsc})
	if k.resourcePickerMode {
		t.Fatal("expected picker closed after esc")
	}
}

func TestResourcePicker_SuggestionFilter(t *testing.T) {
	k := newPickerProvider(t)
	all := k.matchedResourceChoices("")
	if len(all) == 0 {
		t.Fatal("expected non-empty resource choices")
	}
	m := k.matchedResourceChoices("cons")
	if len(m) != 1 || m[0].rt != ConsumerGroupResourceType {
		t.Fatalf("expected only consumer-groups to match 'cons', got %v", m)
	}
}

func TestResourcePicker_EnterSwitches(t *testing.T) {
	k := newPickerProvider(t)
	k.switchResource(SwitchResourceMsg(TopicResourceType))
	k.HandleContentUpdate(StartResourceSwitchingMsg{})
	// Type a partial name and confirm; picker resolves the first match.
	for _, r := range "consumer-groups" {
		k.HandleContentUpdate(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	k.HandleContentUpdate(tea.KeyMsg{Type: tea.KeyEnter})
	if k.resourcePickerMode {
		t.Fatal("expected picker closed after enter")
	}
	if k.currentResource.GetType() != ConsumerGroupResourceType {
		t.Fatalf("expected switch to consumer-groups, got %v", k.currentResource.GetType())
	}
}

func TestResourcePicker_TabComplete(t *testing.T) {
	k := newPickerProvider(t)
	k.HandleContentUpdate(StartResourceSwitchingMsg{})
	k.resourcePickerInput = "top"
	k.HandleContentUpdate(tea.KeyMsg{Type: tea.KeyTab})
	if k.resourcePickerInput != "topics" {
		t.Fatalf("expected tab to complete to 'topics', got %q", k.resourcePickerInput)
	}
}
