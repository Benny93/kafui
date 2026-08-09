// Package keys is the single binding registry required by the controls
// specification (kafui-specification/controls). Every key the application acts
// on is declared here exactly once, with its label, category, context and
// destructive flag. No other package may compare a raw key string.
//
// Resolution: a screen declares the scope it runs in, and Resolve walks that
// scope's contexts from most to least specific. Validate enforces the spec's
// three structural rules — no duplicate key within a scope, no screen shadowing
// a reserved global, and no use of a key terminals cannot report distinctly.
package keys

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

// Action identifies what a binding does. Screens switch on these instead of on
// raw key strings.
type Action string

// Global actions — available on every screen, in every pane.
const (
	ActionHelp        Action = "help"
	ActionPalette     Action = "palette"
	ActionActionsMenu Action = "actions-menu"
	ActionSearch      Action = "search"
	ActionCancel      Action = "cancel"
	ActionActivate    Action = "activate"
	ActionQuit        Action = "quit"
	ActionForceQuit   Action = "force-quit"
	ActionFocusNext   Action = "focus-next"
	ActionFocusPrev   Action = "focus-prev"
	ActionSelectTab   Action = "select-tab"
	ActionToggleMark  Action = "toggle-mark"
	ActionMarkAll     Action = "mark-all"
	ActionRefresh     Action = "refresh"
	ActionCopy        Action = "copy"
	ActionExport      Action = "export"
	ActionSidebar     Action = "toggle-sidebar"
	ActionNew         Action = "new"
)

// Navigation actions — the arrow-key cluster. Nothing requires Fn.
const (
	ActionUp          Action = "up"
	ActionDown        Action = "down"
	ActionPageBack    Action = "page-back"
	ActionPageForward Action = "page-forward"
	ActionFirst       Action = "first"
	ActionLast        Action = "last"
	ActionNextMatch   Action = "next-match"
	ActionPrevMatch   Action = "prev-match"
)

// List-pane actions.
const (
	ActionSort           Action = "sort"
	ActionSortReverse    Action = "sort-reverse"
	ActionDelete         Action = "delete"
	ActionEdit           Action = "edit"
	ActionToggleInternal Action = "toggle-internal"
)

// Content-pane actions.
const (
	ActionFormat   Action = "format"
	ActionWrap     Action = "wrap"
	ActionMetadata Action = "metadata"
)

// Screen-promoted actions.
const (
	ActionPause Action = "pause"
)

// Text-entry actions.
const (
	ActionCommit     Action = "commit"
	ActionCommitSave Action = "commit-save"
	ActionClearField Action = "clear-field"
	ActionDeleteWord Action = "delete-word"
)

// Overlay actions.
const (
	ActionConfirm Action = "confirm"
	ActionDismiss Action = "dismiss"
)

// Debug-build actions.
const (
	ActionScreenshot         Action = "screenshot"
	ActionScreenshotRedacted Action = "screenshot-redacted"
	ActionDebugOverlay       Action = "debug-overlay"
)

// Context groups bindings that apply together.
type Context string

const (
	CtxGlobal    Context = "global"
	CtxList      Context = "list"
	CtxContent   Context = "content"
	CtxTopic     Context = "topic"
	CtxConnector Context = "connector"
	CtxTextEntry Context = "text-entry"
	CtxOverlay   Context = "overlay"
	CtxDebug     Context = "debug"
)

// Categories used to group entries in the help overlay.
const (
	CatNavigation = "Navigation"
	CatView       = "View"
	CatSelection  = "Selection"
	CatActions    = "Actions"
	CatApp        = "Application"
	CatDebug      = "Debug"
)

// Binding is one registry entry. Keys[0] is the advertised key; the rest are
// accepted but never shown in the hint bar.
type Binding struct {
	Action      Action
	Context     Context
	Keys        []string
	Label       string
	Category    string
	Destructive bool
}

// Primary returns the key the hint bar and help overlay display.
func (b Binding) Primary() string {
	if len(b.Keys) == 0 {
		return ""
	}
	return display(b.Keys[0])
}

// Aliases returns the accepted-but-unadvertised keys.
func (b Binding) Aliases() []string {
	if len(b.Keys) < 2 {
		return nil
	}
	out := make([]string, 0, len(b.Keys)-1)
	for _, k := range b.Keys[1:] {
		out = append(out, display(k))
	}
	return out
}

// KeyBinding adapts a registry entry to the bubbles help/key API so the footer
// and help overlay render from the registry rather than from a parallel list.
func (b Binding) KeyBinding() key.Binding {
	return key.NewBinding(key.WithKeys(b.Keys...), key.WithHelp(b.Primary(), b.Label))
}

// display renders a key name the way the user sees it on the keyboard.
var displayNames = map[string]string{
	"up": "↑", "down": "↓", "left": "←", "right": "→",
	"shift+left": "⇧←", "shift+right": "⇧→",
	"pgup": "PgUp", "pgdown": "PgDn", "home": "Home", "end": "End",
	"enter": "enter", "esc": "esc", "tab": "tab", "shift+tab": "⇧tab",
	" ": "space", "backspace": "⌫",
}

func display(k string) string {
	if d, ok := displayNames[k]; ok {
		return d
	}
	return k
}

// Display renders a key name the way the user sees it on the keyboard. It is
// exported so nothing else needs its own copy of this mapping — a second copy
// is how a hint bar ends up disagreeing with a help overlay about whether the
// key is "left" or "←".
func Display(k string) string { return display(k) }

// Scope is the set of contexts active on a screen, most specific first.
type Scope []Context

// The scopes screens run in. Validation checks every one of them.
var (
	ScopeList        = Scope{CtxList, CtxGlobal}
	ScopeContent     = Scope{CtxContent, CtxGlobal}
	ScopeListContent = Scope{CtxList, CtxContent, CtxGlobal}
	ScopeTopic       = Scope{CtxTopic, CtxList, CtxContent, CtxGlobal}
	ScopeConnector   = Scope{CtxConnector, CtxList, CtxContent, CtxGlobal}
	ScopeTextEntry   = Scope{CtxTextEntry}
	ScopeOverlay     = Scope{CtxOverlay}
	ScopeDebug       = Scope{CtxDebug, CtxGlobal}
)

// AllScopes is what Validate checks. A scope absent from this list is not
// validated, so every screen scope must appear here.
var AllScopes = map[string]Scope{
	"list":         ScopeList,
	"content":      ScopeContent,
	"list+content": ScopeListContent,
	"topic":        ScopeTopic,
	"connector":    ScopeConnector,
	"text-entry":   ScopeTextEntry,
	"overlay":      ScopeOverlay,
	"debug":        ScopeDebug,
}

// bannedKeys are keys the spec forbids, with the reason shown when validation
// rejects one. Terminals cannot report these distinctly, or the terminal itself
// consumes them.
var bannedKeys = map[string]string{
	"ctrl+i": "terminals deliver it as tab",
	"ctrl+m": "terminals deliver it as enter",
	"ctrl+j": "terminals deliver it as enter",
	"ctrl+h": "terminals deliver it as backspace",
	"ctrl+[": "terminals deliver it as esc",
	"ctrl+s": "software flow control (XOFF); can freeze the terminal",
	"ctrl+q": "software flow control (XON)",
	"ctrl+z": "job control; the shell suspends the process",
}

// Registry holds every declared binding.
type Registry struct {
	bindings []Binding
	byCtx    map[Context]map[string]Binding
	byAction map[Action]Binding
}

// Default is the application's registry. It is built once and is the only
// source of truth for key handling, hint bars and the help overlay.
var Default = newRegistry(defaultBindings())

func newRegistry(bs []Binding) *Registry {
	r := &Registry{
		bindings: bs,
		byCtx:    map[Context]map[string]Binding{},
		byAction: map[Action]Binding{},
	}
	for _, b := range bs {
		if r.byCtx[b.Context] == nil {
			r.byCtx[b.Context] = map[string]Binding{}
		}
		for _, k := range b.Keys {
			r.byCtx[b.Context][k] = b
		}
		if _, seen := r.byAction[b.Action]; !seen {
			r.byAction[b.Action] = b
		}
	}
	return r
}

// Resolve maps a pressed key to an action within a scope, walking the scope's
// contexts from most to least specific. The second result is false when the
// scope has no binding for the key, in which case the caller does nothing.
func (r *Registry) Resolve(scope Scope, pressed string) (Action, bool) {
	for _, ctx := range scope {
		if b, ok := r.byCtx[ctx][pressed]; ok {
			return b.Action, true
		}
	}
	return "", false
}

// Lookup returns the binding for an action, for rendering its key in menus.
func (r *Registry) Lookup(a Action) (Binding, bool) {
	b, ok := r.byAction[a]
	return b, ok
}

// KeyFor returns the advertised key for an action, or "" when unbound.
func (r *Registry) KeyFor(a Action) string {
	if b, ok := r.byAction[a]; ok {
		return b.Primary()
	}
	return ""
}

// InScope returns every binding reachable in a scope, ordered by context
// specificity then by category, for the help overlay.
func (r *Registry) InScope(scope Scope) []Binding {
	var out []Binding
	seen := map[Action]bool{}
	for _, ctx := range scope {
		for _, b := range r.bindings {
			if b.Context == ctx && !seen[b.Action] {
				seen[b.Action] = true
				out = append(out, b)
			}
		}
	}
	return out
}

// Destructive reports whether an action must route through the confirmation
// dialog.
func (r *Registry) Destructive(a Action) bool {
	b, ok := r.byAction[a]
	return ok && b.Destructive
}

// Validate enforces the spec's structural rules across every declared scope.
// It returns every violation found, not just the first, so a broken binding set
// can be fixed in one pass.
func (r *Registry) Validate() []error {
	var errs []error

	// Rule 3: no key the terminal cannot report distinctly.
	for _, b := range r.bindings {
		for _, k := range b.Keys {
			if reason, banned := bannedKeys[k]; banned {
				errs = append(errs, fmt.Errorf("binding %q uses forbidden key %q: %s", b.Action, k, reason))
			}
			if strings.HasPrefix(k, "ctrl+shift+") {
				errs = append(errs, fmt.Errorf("binding %q uses %q: ctrl+shift is not reportable by many terminals", b.Action, k))
			}
		}
		if b.Label == "" {
			errs = append(errs, fmt.Errorf("binding %q has no help label", b.Action))
		}
		if b.Category == "" {
			errs = append(errs, fmt.Errorf("binding %q has no category", b.Action))
		}
	}

	// Rules 1 and 2: within every scope a key resolves to exactly one action.
	// Because global is the last context of every screen scope, a screen that
	// rebinds a reserved global surfaces here as a duplicate.
	names := make([]string, 0, len(AllScopes))
	for name := range AllScopes {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		owner := map[string]Binding{}
		for _, ctx := range AllScopes[name] {
			for _, b := range r.bindings {
				if b.Context != ctx {
					continue
				}
				for _, k := range b.Keys {
					if prev, dup := owner[k]; dup && prev.Action != b.Action {
						errs = append(errs, fmt.Errorf(
							"scope %q: key %q is bound to both %q (%s) and %q (%s)",
							name, k, prev.Action, prev.Context, b.Action, b.Context))
						continue
					}
					owner[k] = b
				}
			}
		}
	}

	return errs
}

// defaultBindings is the normative key map from
// kafui-specification/controls/keymap.md, transcribed once.
func defaultBindings() []Binding {
	return []Binding{
		// ---- Global ----
		{ActionHelp, CtxGlobal, []string{"?", "f1"}, "help", CatApp, false},
		{ActionPalette, CtxGlobal, []string{":", "ctrl+p"}, "commands", CatApp, false},
		{ActionActionsMenu, CtxGlobal, []string{"a"}, "actions", CatActions, false},
		{ActionSearch, CtxGlobal, []string{"/"}, "search", CatView, false},
		{ActionCancel, CtxGlobal, []string{"esc"}, "back", CatNavigation, false},
		{ActionActivate, CtxGlobal, []string{"enter"}, "open", CatNavigation, false},
		{ActionQuit, CtxGlobal, []string{"q"}, "quit", CatApp, false},
		{ActionForceQuit, CtxGlobal, []string{"ctrl+c"}, "quit now", CatApp, false},
		{ActionFocusNext, CtxGlobal, []string{"tab"}, "next pane", CatNavigation, false},
		{ActionFocusPrev, CtxGlobal, []string{"shift+tab"}, "prev pane", CatNavigation, false},
		{ActionSelectTab, CtxGlobal, []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}, "go to tab", CatNavigation, false},
		{ActionToggleMark, CtxGlobal, []string{" "}, "mark row", CatSelection, false},
		{ActionMarkAll, CtxGlobal, []string{"ctrl+a"}, "mark all", CatSelection, false},
		{ActionRefresh, CtxGlobal, []string{"r", "f5"}, "refresh", CatView, false},
		{ActionCopy, CtxGlobal, []string{"c", "y"}, "copy", CatActions, false},
		{ActionExport, CtxGlobal, []string{"ctrl+e"}, "export CSV", CatActions, false},
		{ActionSidebar, CtxGlobal, []string{"ctrl+b"}, "sidebar", CatApp, false},
		{ActionNew, CtxGlobal, []string{"ctrl+n"}, "new", CatActions, false},

		// ---- Global: navigation, arrows only ----
		{ActionUp, CtxGlobal, []string{"up", "k"}, "up", CatNavigation, false},
		{ActionDown, CtxGlobal, []string{"down", "j"}, "down", CatNavigation, false},
		{ActionPageBack, CtxGlobal, []string{"left", "h", "pgup"}, "page back", CatNavigation, false},
		{ActionPageForward, CtxGlobal, []string{"right", "l", "pgdown"}, "page fwd", CatNavigation, false},
		{ActionFirst, CtxGlobal, []string{"shift+left", "g", "home"}, "first", CatNavigation, false},
		{ActionLast, CtxGlobal, []string{"shift+right", "G", "end"}, "last", CatNavigation, false},
		{ActionNextMatch, CtxGlobal, []string{"n"}, "next match", CatView, false},
		{ActionPrevMatch, CtxGlobal, []string{"N"}, "prev match", CatView, false},

		// ---- List panes ----
		{ActionSort, CtxList, []string{"s"}, "sort", CatView, false},
		{ActionSortReverse, CtxList, []string{"S"}, "sort dir", CatView, false},
		{ActionDelete, CtxList, []string{"d"}, "delete", CatActions, true},
		{ActionEdit, CtxList, []string{"e"}, "edit", CatActions, false},
		{ActionToggleInternal, CtxList, []string{"i"}, "internal", CatView, false},

		// ---- Content panes ----
		{ActionFormat, CtxContent, []string{"f"}, "format", CatView, false},
		{ActionWrap, CtxContent, []string{"w"}, "wrap", CatView, false},
		{ActionMetadata, CtxContent, []string{"m"}, "metadata", CatView, false},

		// ---- Screen-promoted ----
		{ActionPause, CtxTopic, []string{"p"}, "pause", CatActions, false},
		{ActionPause, CtxConnector, []string{"p"}, "pause", CatActions, false},

		// ---- Text entry ----
		{ActionCommit, CtxTextEntry, []string{"enter"}, "confirm", CatApp, false},
		{ActionCancel, CtxTextEntry, []string{"esc"}, "cancel", CatApp, false},
		{ActionFocusNext, CtxTextEntry, []string{"tab"}, "complete", CatApp, false},
		{ActionFocusPrev, CtxTextEntry, []string{"shift+tab"}, "prev field", CatApp, false},
		{ActionUp, CtxTextEntry, []string{"up"}, "prev entry", CatNavigation, false},
		{ActionDown, CtxTextEntry, []string{"down"}, "next entry", CatNavigation, false},
		{ActionClearField, CtxTextEntry, []string{"ctrl+u"}, "clear", CatApp, false},
		{ActionDeleteWord, CtxTextEntry, []string{"ctrl+w"}, "delete word", CatApp, false},
		{ActionForceQuit, CtxTextEntry, []string{"ctrl+c"}, "quit now", CatApp, false},
		// F5 runs the statement being edited. It is the one action key that
		// survives text entry, because it is not a printable character.
		{ActionRefresh, CtxTextEntry, []string{"f5"}, "run", CatActions, false},
		// F2 saves a form or an editor's contents. It joins F5 as an action key
		// that survives text entry because it is not a printable character.
		{ActionCommitSave, CtxTextEntry, []string{"f2"}, "save", CatActions, false},

		// ---- Overlays ----
		{ActionUp, CtxOverlay, []string{"up", "k"}, "up", CatNavigation, false},
		{ActionDown, CtxOverlay, []string{"down", "j"}, "down", CatNavigation, false},
		{ActionFocusNext, CtxOverlay, []string{"tab"}, "next", CatNavigation, false},
		{ActionFocusPrev, CtxOverlay, []string{"shift+tab"}, "prev", CatNavigation, false},
		{ActionActivate, CtxOverlay, []string{"enter"}, "confirm", CatApp, false},
		{ActionCancel, CtxOverlay, []string{"esc"}, "dismiss", CatApp, false},
		{ActionConfirm, CtxOverlay, []string{"y"}, "yes", CatApp, false},
		{ActionDismiss, CtxOverlay, []string{"n"}, "no", CatApp, false},
		{ActionSearch, CtxOverlay, []string{"/"}, "filter", CatView, false},
		// Overlays that show live data or a list of saved things need the same
		// refresh, edit and delete vocabulary as the screens behind them.
		{ActionRefresh, CtxOverlay, []string{"r"}, "refresh", CatView, false},
		{ActionEdit, CtxOverlay, []string{"e"}, "edit", CatActions, false},
		{ActionDelete, CtxOverlay, []string{"d"}, "delete", CatActions, true},

		// ---- Debug builds ----
		{ActionScreenshot, CtxDebug, []string{"f3"}, "screenshot", CatDebug, false},
		{ActionScreenshotRedacted, CtxDebug, []string{"shift+f3"}, "redacted shot", CatDebug, false},
		{ActionDebugOverlay, CtxDebug, []string{"f12"}, "debug overlay", CatDebug, false},
	}
}

// ---- user-defined bindings ----

// Override rebinds one action. Keys[0] becomes the advertised key.
type Override struct {
	Action string   `yaml:"action"`
	Keys   []string `yaml:"keys"`
}

// ApplyOverrides rebuilds the registry with the user's overrides applied,
// returning the new registry and one error per rejected override. Rejected
// overrides are dropped rather than fatal: the spec requires the application to
// start with the defaults for anything it could not honour, and to say which.
func ApplyOverrides(overrides []Override) (*Registry, []error) {
	base := defaultBindings()
	var errs []error

	byAction := map[Action]int{}
	for i, b := range base {
		if _, seen := byAction[b.Action]; !seen {
			byAction[b.Action] = i
		}
	}

	for _, o := range overrides {
		i, known := byAction[Action(o.Action)]
		if !known {
			errs = append(errs, fmt.Errorf("keybinding override for unknown action %q ignored", o.Action))
			continue
		}
		if len(o.Keys) == 0 {
			errs = append(errs, fmt.Errorf("keybinding override for %q lists no keys; ignored", o.Action))
			continue
		}
		base[i].Keys = append([]string{}, o.Keys...)
	}

	r := newRegistry(base)
	if problems := r.Validate(); len(problems) > 0 {
		// A conflicting override must not take the whole binding set down with
		// it, so fall back to the defaults and report why.
		errs = append(errs, problems...)
		errs = append(errs, fmt.Errorf("keybinding overrides rejected; using the default bindings"))
		return newRegistry(defaultBindings()), errs
	}
	return r, errs
}

// SetDefault installs a registry as the application's active one. Called once
// at start-up after configuration has been read.
func SetDefault(r *Registry) { Default = r }
