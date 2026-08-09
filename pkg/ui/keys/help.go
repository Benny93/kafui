package keys

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

// hintOrder is the order the hint bar prefers when it has room for only a few
// bindings: what the user needs most often, first.
var hintOrder = []Action{
	ActionHelp, ActionActionsMenu, ActionSearch, ActionPalette,
	ActionActivate, ActionCancel, ActionQuit,
}

// HintKeyMap renders a scope's bindings for the footer hint bar and the help
// overlay. It satisfies the bubbles help.KeyMap interface, so the footer stays
// registry-driven and cannot drift from behavior.
type HintKeyMap struct {
	scope Scope
	// extra are screen-specific bindings contributed at runtime, e.g. actions
	// the screen promotes that are not in the static registry.
	extra []key.Binding
}

// Hints builds a hint key map for a scope.
func Hints(scope Scope) HintKeyMap { return HintKeyMap{scope: scope} }

// With returns a copy carrying additional bindings shown after the standard ones.
func (h HintKeyMap) With(extra ...key.Binding) HintKeyMap {
	h.extra = append(append([]key.Binding{}, h.extra...), extra...)
	return h
}

// ShortHelp implements help.KeyMap — the single-line hint bar.
func (h HintKeyMap) ShortHelp() []key.Binding {
	var out []key.Binding
	for _, a := range hintOrder {
		if b, ok := Default.Lookup(a); ok && inScope(h.scope, b.Context) {
			out = append(out, b.KeyBinding())
		}
	}
	return append(out, h.extra...)
}

// FullHelp implements help.KeyMap — the expanded, category-grouped view.
func (h HintKeyMap) FullHelp() [][]key.Binding {
	order := []string{CatNavigation, CatView, CatSelection, CatActions, CatApp}
	byCat := map[string][]key.Binding{}
	for _, b := range Default.InScope(h.scope) {
		byCat[b.Category] = append(byCat[b.Category], b.KeyBinding())
	}
	var cols [][]key.Binding
	for _, c := range order {
		if len(byCat[c]) > 0 {
			cols = append(cols, byCat[c])
		}
	}
	if len(h.extra) > 0 {
		cols = append(cols, h.extra)
	}
	return cols
}

func inScope(s Scope, c Context) bool {
	for _, x := range s {
		if x == c {
			return true
		}
	}
	return false
}

// GlobalBindings returns every global binding, for the help overlay's shared
// section.
func GlobalBindings() []key.Binding {
	var out []key.Binding
	for _, b := range Default.InScope(Scope{CtxGlobal}) {
		out = append(out, b.KeyBinding())
	}
	return out
}

// Binding returns the bubbles binding for an action, for callers that still
// need key.Matches (the focus manager).
func BindingFor(a Action) key.Binding {
	if b, ok := Default.Lookup(a); ok {
		return b.KeyBinding()
	}
	return key.NewBinding()
}

var _ help.KeyMap = HintKeyMap{}

// Help returns every binding reachable in a scope, for a page's GetHelp. The
// hint bar shows only the handful in hintOrder; the help overlay shows all of
// them, so the two must not share one list.
func Help(scope Scope) []key.Binding {
	var out []key.Binding
	for _, b := range Default.InScope(scope) {
		out = append(out, b.KeyBinding())
	}
	return out
}

// KeyIn returns the key that triggers an action WITHIN a scope, which is not
// always the globally advertised one: refresh is `r` on a list but `F5` in a
// text editor, because `r` there is a character the user is typing. Returns ""
// when the action is unreachable in that scope.
func (r *Registry) KeyIn(scope Scope, a Action) string {
	for _, ctx := range scope {
		for _, b := range r.bindings {
			if b.Context == ctx && b.Action == a {
				return b.Primary()
			}
		}
	}
	return ""
}

// Hint renders a "key action • key action" strip for a scope, so an inline hint
// can never advertise a key the screen does not handle — nor the wrong key for
// the context it is shown in. Pairs are (Action, label).
func Hint(scope Scope, pairs ...any) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		action, _ := pairs[i].(Action)
		label, _ := pairs[i+1].(string)
		if k := Default.KeyIn(scope, action); k != "" {
			parts = append(parts, k+" "+label)
		}
	}
	return strings.Join(parts, " • ")
}
