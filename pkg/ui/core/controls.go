package core

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/ui/components/menu"
	"github.com/Benny93/kafui/pkg/ui/keys"
)

// The controls spec requires every action to be reachable through a visible
// surface. These three optional interfaces are how a page contributes to them.
// A page that implements none still works — it simply offers nothing beyond the
// reserved global vocabulary.

// ActionProvider supplies the contextual actions menu (`a`, right-click) for
// whatever the page currently has focused or selected. Entries the user may not
// perform must be returned Disabled with a Reason rather than omitted, so the
// action's existence stays discoverable.
type ActionProvider interface {
	ContextActions() []menu.Entry
}

// PaletteProvider supplies page-specific destinations and commands to the
// command palette, on top of the application-wide entries the shell adds.
type PaletteProvider interface {
	PaletteEntries() []menu.Entry
}

// Unwinder lets a page consume one level of Esc before the shell navigates
// back: closing its own overlay, leaving text entry, or clearing an active
// filter. Returning false means the page has nothing left to unwind and the
// shell should go to the parent screen.
type Unwinder interface {
	Unwind() (tea.Cmd, bool)
}

// InputModeReporter is implemented by pages that can hold a focused text field.
// While it reports true, only ctrl+c escapes — every other key is typed.
type InputModeReporter interface {
	IsInputMode() bool
}

// KeyScoper lets a page declare the key scope it resolves against. Pages that
// do not implement it are treated as list screens.
type KeyScoper interface {
	KeyScope() keys.Scope
}

// ScopeOf returns the key scope for a page, defaulting to a list screen.
func ScopeOf(page any) keys.Scope {
	if s, ok := page.(KeyScoper); ok {
		return s.KeyScope()
	}
	return keys.ScopeList
}
