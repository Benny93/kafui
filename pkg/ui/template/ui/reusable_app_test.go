package ui

import (
	"testing"

	"github.com/Benny93/kafui/pkg/ui/core"
	tea "github.com/charmbracelet/bubbletea"
)

// drain executes a (possibly batched) command and reports whether any resulting
// message is a SidebarToggledMsg.
func containsSidebarToggle(cmd tea.Cmd) (core.SidebarToggledMsg, bool) {
	if cmd == nil {
		return core.SidebarToggledMsg{}, false
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if m, ok := containsSidebarToggle(c); ok {
				return m, true
			}
		}
		return core.SidebarToggledMsg{}, false
	}
	if m, ok := msg.(core.SidebarToggledMsg); ok {
		return m, true
	}
	return core.SidebarToggledMsg{}, false
}

func TestSidebarToggleFlipsAndEmits(t *testing.T) {
	app := NewDefaultApp()
	// Large size so the sidebar toggle is permitted.
	app.Update(tea.WindowSizeMsg{Width: 200, Height: 60})
	if !app.showSidebar {
		t.Fatal("expected sidebar visible at large size by default")
	}

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	if app.showSidebar {
		t.Fatal("expected sidebar hidden after toggle")
	}
	msg, ok := containsSidebarToggle(cmd)
	if !ok {
		t.Fatal("expected SidebarToggledMsg to be emitted on explicit toggle")
	}
	if msg.Visible {
		t.Fatal("expected emitted SidebarToggledMsg.Visible=false after hiding")
	}
}

func TestSidebarAutoCollapseWinsAtSmallSize(t *testing.T) {
	app := NewDefaultApp()
	app.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	if app.showSidebar {
		t.Fatal("expected sidebar auto-collapsed at small size")
	}
	// Toggle must be a no-op at small size (auto-collapse wins).
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	if app.showSidebar {
		t.Fatal("expected sidebar to stay collapsed at small size")
	}
}
