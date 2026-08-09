package mainpage

import (
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/keys"
)

func newControlsProvider(t *testing.T) *KafuiContentProvider {
	t.Helper()
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	k := NewKafuiContentProvider(ds)
	k.HandleContentUpdate(SwitchResourceMsg(TopicResourceType))
	return k
}

// Regression: menu entries are built in order to be DISPLAYED. An eager
// `Run: k.openCreateTopicForm()` opened the create form behind the menu every
// time the menu was rendered.
func TestBuildingActionsMenuHasNoSideEffects(t *testing.T) {
	k := newControlsProvider(t)
	for range 3 {
		k.ContextActions()
	}
	if k.showTopicForm {
		t.Error("building the actions menu opened the create-topic form")
	}
	if k.activeOverlayForm() != nil {
		t.Error("building the actions menu opened an overlay form")
	}
	if k.searchMode {
		t.Error("building the actions menu entered search mode")
	}
}

// Coverage rule: every action the screen can perform is listed somewhere the
// user can see it. Destructive entries must be flagged so they render in the
// danger style and route through confirmation.
func TestActionsMenuListsDestructiveWork(t *testing.T) {
	k := newControlsProvider(t)
	entries := k.ContextActions()
	if len(entries) == 0 {
		t.Fatal("no actions offered for a topic list")
	}
	var destructive int
	for _, e := range entries {
		if e.Destructive {
			destructive++
		}
		if e.Disabled && e.Reason == "" {
			t.Errorf("disabled entry %q must state a reason", e.Label)
		}
		if !e.Disabled && e.Run == nil && e.Key == "" {
			t.Errorf("entry %q is neither runnable nor a key hint", e.Label)
		}
	}
	if destructive == 0 {
		t.Error("expected delete/recreate/purge to be flagged destructive")
	}
}

// Every key an entry advertises must actually be bound in the registry, so the
// menu cannot teach a shortcut that does nothing.
func TestActionsMenuKeysExistInTheRegistry(t *testing.T) {
	k := newControlsProvider(t)
	bound := map[string]bool{}
	for _, scope := range []keys.Scope{keys.ScopeList, keys.ScopeTopic} {
		for _, b := range keys.Default.InScope(scope) {
			for _, key := range b.Keys {
				bound[b.Primary()] = true
				_ = key
			}
		}
	}
	for _, e := range k.ContextActions() {
		if e.Key != "" && !bound[e.Key] {
			t.Errorf("entry %q advertises key %q, which is not in the registry", e.Label, e.Key)
		}
	}
}

// Header clicks must land on the column the user actually pointed at, which
// means mirroring bubble-table's flex arithmetic rather than guessing.
func TestColumnAtXMatchesTheRenderedLayout(t *testing.T) {
	cols := createResourceTableColumns(TopicResourceType)
	const width = 135

	// Walk every X inside the table and check the column index only ever
	// increases — a boundary bug shows up as a column appearing twice.
	last := -1
	seen := map[int]bool{}
	for x := range width {
		col, ok := columnAtX(cols, width, x)
		if !ok {
			continue
		}
		if col != last {
			if seen[col] {
				t.Fatalf("column %d appears in two separate ranges", col)
			}
			seen[col] = true
			last = col
		}
	}
	if len(seen) != len(cols) {
		t.Errorf("resolved %d of %d columns", len(seen), len(cols))
	}

	// The first column is the flex Name column and must be the widest.
	first, ok := columnAtX(cols, width, 2)
	if !ok || first != 0 {
		t.Errorf("x=2 should be the Name column, got %d (ok=%v)", first, ok)
	}
}

func TestHeaderClickSortsThenReverses(t *testing.T) {
	k := newControlsProvider(t)
	k.topicSortCol = -1

	k.setSortColumn(2)
	if k.topicSortCol != 2 || k.topicSortDesc {
		t.Fatalf("first click sorts ascending by that column, got col=%d desc=%v", k.topicSortCol, k.topicSortDesc)
	}
	k.setSortColumn(2)
	if !k.topicSortDesc {
		t.Error("clicking the active sort column again reverses direction")
	}
	k.setSortColumn(3)
	if k.topicSortCol != 3 || k.topicSortDesc {
		t.Error("clicking a different column sorts by it, ascending")
	}
}
