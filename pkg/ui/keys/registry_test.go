package keys

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The controls spec's structural rules. This is the test that keeps the binding
// set honest: it fails on a duplicate key within a scope, on a screen shadowing
// a reserved global, and on any key terminals cannot report distinctly.
func TestRegistryValidates(t *testing.T) {
	errs := Default.Validate()
	for _, err := range errs {
		t.Error(err)
	}
	require.Empty(t, errs, "the default binding set must satisfy the controls spec")
}

func TestNoActionIsReachableOnlyByAFnKey(t *testing.T) {
	// PgUp/PgDn/Home/End are aliases, never the only way to reach an action —
	// laptop and 60% keyboards put them behind Fn.
	fnOnly := map[string]bool{"pgup": true, "pgdown": true, "home": true, "end": true}
	for _, b := range Default.bindings {
		reachable := false
		for _, k := range b.Keys {
			if !fnOnly[k] {
				reachable = true
				break
			}
		}
		assert.True(t, reachable, "action %q is reachable only through an Fn key", b.Action)
	}
}

func TestNavigationIsOnTheArrowKeys(t *testing.T) {
	for _, tc := range []struct {
		action Action
		key    string
	}{
		{ActionUp, "up"},
		{ActionDown, "down"},
		{ActionPageBack, "left"},
		{ActionPageForward, "right"},
		{ActionFirst, "shift+left"},
		{ActionLast, "shift+right"},
	} {
		b, ok := Default.Lookup(tc.action)
		require.True(t, ok, "%s must be bound", tc.action)
		assert.Equal(t, tc.key, b.Keys[0], "%s must advertise the arrow key", tc.action)
	}
}

func TestResolveWalksScopeFromSpecificToGeneral(t *testing.T) {
	// 'p' is a topic-screen binding and unbound elsewhere.
	a, ok := Default.Resolve(ScopeTopic, "p")
	assert.True(t, ok)
	assert.Equal(t, ActionPause, a)

	_, ok = Default.Resolve(ScopeList, "p")
	assert.False(t, ok, "a screen binding must not leak into other scopes")

	// Globals resolve from every screen scope.
	for name, scope := range map[string]Scope{"list": ScopeList, "topic": ScopeTopic} {
		a, ok := Default.Resolve(scope, "?")
		assert.True(t, ok, "help must resolve in %s", name)
		assert.Equal(t, ActionHelp, a)
	}
}

func TestTextEntryDoesNotInheritGlobals(t *testing.T) {
	// The whole point of the text-entry scope: printable characters bound in
	// Normal mode must reach the field instead of running an action.
	for _, k := range []string{"a", "c", "d", "e", "f", "n", "q", "r", "s", "w", "y", "/", ":", "?"} {
		_, ok := Default.Resolve(ScopeTextEntry, k)
		assert.False(t, ok, "key %q must be typed, not acted on, during text entry", k)
	}
	// Except the emergency exit.
	a, ok := Default.Resolve(ScopeTextEntry, "ctrl+c")
	assert.True(t, ok)
	assert.Equal(t, ActionForceQuit, a)
}

func TestEscIsCancelAndQIsQuit(t *testing.T) {
	a, _ := Default.Resolve(ScopeList, "esc")
	assert.Equal(t, ActionCancel, a, "esc goes back, never quits")
	a, _ = Default.Resolve(ScopeList, "q")
	assert.Equal(t, ActionQuit, a, "q quits, never goes back")
}

func TestTabIsFocusOnly(t *testing.T) {
	a, _ := Default.Resolve(ScopeTopic, "tab")
	assert.Equal(t, ActionFocusNext, a)
	a, _ = Default.Resolve(ScopeTopic, "shift+tab")
	assert.Equal(t, ActionFocusPrev, a, "shift+tab is the exact inverse of tab")
}

func TestSpaceMarksRatherThanPages(t *testing.T) {
	a, _ := Default.Resolve(ScopeTopic, " ")
	assert.Equal(t, ActionToggleMark, a)
}

func TestForbiddenKeysAreRejected(t *testing.T) {
	for key, reason := range map[string]string{
		"ctrl+i":       "tab",
		"ctrl+s":       "flow control",
		"ctrl+shift+x": "not reportable",
	} {
		r := newRegistry(append(defaultBindings(), Binding{
			Action: "bogus", Context: CtxList, Keys: []string{key},
			Label: "bogus", Category: CatActions,
		}))
		assert.NotEmpty(t, r.Validate(), "%s (%s) must be rejected", key, reason)
	}
}

func TestDuplicateWithinAScopeIsRejected(t *testing.T) {
	r := newRegistry(append(defaultBindings(), Binding{
		Action: "bogus", Context: CtxList, Keys: []string{"?"},
		Label: "bogus", Category: CatActions,
	}))
	assert.NotEmpty(t, r.Validate(), "a screen may not shadow a reserved global")
}

func TestDeleteIsMarkedDestructive(t *testing.T) {
	assert.True(t, Default.Destructive(ActionDelete))
	assert.False(t, Default.Destructive(ActionCopy))
}

func TestRemovedGlobalsAreGone(t *testing.T) {
	// The hard break: screen-jump keys moved into the command palette, and the
	// shell's sidebar/debug chords moved off keys that pages also used.
	for _, k := range []string{"T", "C", "K", "ctrl+t", "ctrl+g", "ctrl+w", "ctrl+r", "ctrl+d"} {
		_, ok := Default.Resolve(ScopeTopic, k)
		assert.False(t, ok, "%q must no longer be bound", k)
	}
}

func TestUserOverrideRebindsAnAction(t *testing.T) {
	// `z` is in the free pool; `m` would collide with the content pane's
	// metadata key, which the next test covers.
	r, errs := ApplyOverrides([]Override{{Action: string(ActionActionsMenu), Keys: []string{"z"}}})
	assert.Empty(t, errs)

	a, ok := r.Resolve(ScopeList, "z")
	assert.True(t, ok)
	assert.Equal(t, ActionActionsMenu, a)

	_, ok = r.Resolve(ScopeList, "a")
	assert.False(t, ok, "the default key no longer triggers the action")
}

func TestOverrideOntoAnotherContextsKeyIsRejected(t *testing.T) {
	// `m` is the content pane's metadata key, so binding a global to it makes
	// the two ambiguous on any screen that has both.
	_, errs := ApplyOverrides([]Override{{Action: string(ActionActionsMenu), Keys: []string{"m"}}})
	assert.NotEmpty(t, errs)
}

func TestConflictingOverrideIsRejectedWithoutBreakingStartup(t *testing.T) {
	// Rebinding delete onto the help key collides with a reserved global.
	r, errs := ApplyOverrides([]Override{{Action: string(ActionDelete), Keys: []string{"?"}}})
	assert.NotEmpty(t, errs, "the conflict must be reported")

	a, ok := r.Resolve(ScopeList, "?")
	assert.True(t, ok)
	assert.Equal(t, ActionHelp, a, "the defaults are kept when an override is rejected")
	a, _ = r.Resolve(ScopeList, "d")
	assert.Equal(t, ActionDelete, a)
}

func TestOverrideOfAnUnknownActionIsReported(t *testing.T) {
	_, errs := ApplyOverrides([]Override{{Action: "not-an-action", Keys: []string{"z"}}})
	assert.NotEmpty(t, errs)
}

func TestOverrideCannotUseAForbiddenKey(t *testing.T) {
	_, errs := ApplyOverrides([]Override{{Action: string(ActionDelete), Keys: []string{"ctrl+s"}}})
	assert.NotEmpty(t, errs, "flow-control keys stay forbidden for user overrides too")
}
