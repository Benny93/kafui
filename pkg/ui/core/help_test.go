package core

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// mockHelpPage for testing help system
type mockHelpPage struct {
	id    string
	title string
	help  []key.Binding
}

func (m *mockHelpPage) Init() tea.Cmd                                    { return nil }
func (m *mockHelpPage) Update(msg tea.Msg) (tea.Model, tea.Cmd)          { return m, nil }
func (m *mockHelpPage) View() string                                     { return "mock view" }
func (m *mockHelpPage) SetDimensions(width, height int)                  {}
func (m *mockHelpPage) GetID() string                                    { return m.id }
func (m *mockHelpPage) GetTitle() string                                 { return m.title }
func (m *mockHelpPage) GetHelp() []key.Binding                           { return m.help }
func (m *mockHelpPage) HandleNavigation(msg tea.Msg) (Page, tea.Cmd)     { return m, nil }
func (m *mockHelpPage) OnFocus() tea.Cmd                                 { return nil }
func (m *mockHelpPage) OnBlur() tea.Cmd                                  { return nil }

func newMockHelpPage(id, title string, helpBindings []key.Binding) *mockHelpPage {
	return &mockHelpPage{
		id:    id,
		title: title,
		help:  helpBindings,
	}
}

func TestNewHelpSystem(t *testing.T) {
	help := NewHelpSystem()
	
	if help == nil {
		t.Fatal("NewHelpSystem returned nil")
	}
	
	if help.IsVisible() {
		t.Error("Expected help system to be hidden by default")
	}
}

func TestHelpSystemToggle(t *testing.T) {
	help := NewHelpSystem()
	
	// Initially hidden
	if help.IsVisible() {
		t.Error("Expected help system to be hidden initially")
	}
	
	// Toggle to show
	help.Toggle()
	if !help.IsVisible() {
		t.Error("Expected help system to be visible after toggle")
	}
	
	// Toggle to hide
	help.Toggle()
	if help.IsVisible() {
		t.Error("Expected help system to be hidden after second toggle")
	}
}

func TestHelpSystemShowHide(t *testing.T) {
	help := NewHelpSystem()
	
	// Show
	help.Show()
	if !help.IsVisible() {
		t.Error("Expected help system to be visible after Show()")
	}
	
	// Hide
	help.Hide()
	if help.IsVisible() {
		t.Error("Expected help system to be hidden after Hide()")
	}
}

func TestHelpSystemSetCurrentPage(t *testing.T) {
	help := NewHelpSystem()
	
	mockBindings := []key.Binding{
		key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "test function"),
		),
	}
	
	page := newMockHelpPage("test", "Test Page", mockBindings)
	help.SetCurrentPage(page)
	
	// The page should be set (we can't directly access it, but we can test through rendering)
	help.Show()
	help.SetDimensions(100, 30)
	rendered := help.Render()
	
	if !strings.Contains(rendered, "Test Page") {
		t.Error("Expected rendered help to contain page title")
	}
	
	if !strings.Contains(rendered, "test function") {
		t.Error("Expected rendered help to contain page-specific help")
	}
}

func TestHelpSystemRender(t *testing.T) {
	help := NewHelpSystem()
	help.SetDimensions(100, 30)
	
	// When hidden, should return empty string
	rendered := help.Render()
	if rendered != "" {
		t.Error("Expected empty string when help is hidden")
	}
	
	// When shown, should return help content
	help.Show()
	rendered = help.Render()
	if rendered == "" {
		t.Error("Expected non-empty string when help is shown")
	}
	
	// Should contain global help
	if !strings.Contains(rendered, "Global Keys") {
		t.Error("Expected rendered help to contain 'Global Keys' section")
	}
	
	if !strings.Contains(rendered, "quit") {
		t.Error("Expected rendered help to contain quit binding")
	}
}

func TestHelpSystemCompactMode(t *testing.T) {
	help := NewHelpSystem()
	help.Show()
	
	// Set very small dimensions to trigger compact mode
	help.SetDimensions(30, 5)
	rendered := help.Render()
	
	if rendered == "" {
		t.Error("Expected non-empty string in compact mode")
	}
	
	// Compact mode should be much shorter
	lines := strings.Split(rendered, "\n")
	if len(lines) > 5 {
		t.Errorf("Expected compact help to be short, got %d lines", len(lines))
	}
}

func TestHelpSystemWithPageSpecificBindings(t *testing.T) {
	help := NewHelpSystem()
	help.SetDimensions(100, 30)
	help.Show()
	
	// Create a page with specific bindings
	pageBindings := []key.Binding{
		key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "format message"),
		),
		key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy content"),
		),
	}
	
	page := newMockHelpPage("message_detail", "Message Detail", pageBindings)
	help.SetCurrentPage(page)
	
	rendered := help.Render()
	
	// Should contain page-specific section
	if !strings.Contains(rendered, "Message Detail Keys") {
		t.Error("Expected page-specific section in help")
	}
	
	// Should contain page-specific bindings
	if !strings.Contains(rendered, "format message") {
		t.Error("Expected page-specific binding 'format message'")
	}
	
	if !strings.Contains(rendered, "copy content") {
		t.Error("Expected page-specific binding 'copy content'")
	}
}

func TestHelpSystemSections(t *testing.T) {
	help := NewHelpSystem()
	help.SetDimensions(100, 40)
	help.Show()
	
	rendered := help.Render()
	
	// Should contain all expected sections
	expectedSections := []string{
		"Global Keys",
		"Navigation", 
		"Focus Management",
	}
	
	for _, section := range expectedSections {
		if !strings.Contains(rendered, section) {
			t.Errorf("Expected help to contain section '%s'", section)
		}
	}
	
	// Should contain navigation bindings
	if !strings.Contains(rendered, "move up") {
		t.Error("Expected navigation bindings")
	}
	
	// Should contain focus bindings
	if !strings.Contains(rendered, "next component") {
		t.Error("Expected focus management bindings")
	}
}

func TestGetKeyBindingHelp(t *testing.T) {
	help := NewHelpSystem()
	
	binding := key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "save file"),
	)
	
	helpText := help.GetKeyBindingHelp(binding)
	expected := "ctrl+s: save file"
	
	if helpText != expected {
		t.Errorf("Expected '%s', got '%s'", expected, helpText)
	}
}

func TestGetQuickHelp(t *testing.T) {
	help := NewHelpSystem()
	
	quickHelp := help.GetQuickHelp()
	
	// Should contain essential bindings
	if !strings.Contains(quickHelp, "help") {
		t.Error("Expected quick help to contain 'help'")
	}
	
	if !strings.Contains(quickHelp, "quit") {
		t.Error("Expected quick help to contain 'quit'")
	}
	
	if !strings.Contains(quickHelp, "back") {
		t.Error("Expected quick help to contain 'back'")
	}
	
	// Should use bullet separator
	if !strings.Contains(quickHelp, "•") {
		t.Error("Expected quick help to use bullet separator")
	}
}

func TestGetQuickHelpWithPage(t *testing.T) {
	help := NewHelpSystem()
	
	pageBindings := []key.Binding{
		key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "format"),
		),
	}
	
	page := newMockHelpPage("test", "Test Page", pageBindings)
	help.SetCurrentPage(page)
	
	quickHelp := help.GetQuickHelp()
	
	// Should include page-specific binding
	if !strings.Contains(quickHelp, "format") {
		t.Error("Expected quick help to contain page-specific binding")
	}
}

func TestHelpSystemSetDimensions(t *testing.T) {
	help := NewHelpSystem()
	
	// Test that SetDimensions doesn't panic
	help.SetDimensions(80, 24)
	help.SetDimensions(120, 40)
	help.SetDimensions(0, 0)
	
	// Test rendering with different dimensions
	help.Show()
	
	help.SetDimensions(100, 30)
	rendered1 := help.Render()
	
	help.SetDimensions(50, 15)
	rendered2 := help.Render()
	
	// Different dimensions should produce different output
	if rendered1 == rendered2 {
		t.Error("Expected different output for different dimensions")
	}
}