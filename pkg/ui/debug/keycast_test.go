//go:build debug

package debug

import (
	"strings"
	"testing"
)

func TestKeycastDistinguishesUnboundKeys(t *testing.T) {
	var k Keycast
	k.Record("s", "sort")
	k.Record("z", "")

	out := k.View()
	if !strings.Contains(out, "sort") {
		t.Error("a bound key must show the action it resolved to")
	}
	if !strings.Contains(out, "unbound") {
		t.Error("an unbound key must say so — that is the whole point of the strip")
	}
}

func TestKeycastKeepsOnlyTheRecentPresses(t *testing.T) {
	var k Keycast
	for _, key := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		k.Record(key, "act")
	}
	if len(k.entries) != maxKeys {
		t.Errorf("expected the strip to hold %d presses, got %d", maxKeys, len(k.entries))
	}
	if k.entries[len(k.entries)-1].key != "h" {
		t.Error("the newest press must survive")
	}
}

func TestKeycastIgnoresEmptyPresses(t *testing.T) {
	var k Keycast
	k.Record("", "")
	if k.View() != "" {
		t.Error("an empty key name must not produce a strip entry")
	}
}

func TestKeycastRendersTheRegistrySpelling(t *testing.T) {
	var k Keycast
	k.Record("shift+left", "first")
	if !strings.Contains(k.View(), "⇧←") {
		t.Error("the strip must use the same key spelling as the hint bar and help")
	}
}
