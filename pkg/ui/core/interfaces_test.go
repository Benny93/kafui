package core

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// mockPage is a mock implementation of the Page interface for testing
type mockPage struct {
	id          string
	title       string
	help        []key.Binding
	focusCmd    tea.Cmd
	blurCmd     tea.Cmd
	handleNavFn func(tea.Msg) (Page, tea.Cmd)
}

func (m *mockPage) Init() tea.Cmd {
	return nil
}

func (m *mockPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *mockPage) View() string {
	return "mock page view"
}

func (m *mockPage) SetDimensions(width, height int) {
	// Mock implementation
}

func (m *mockPage) GetID() string {
	return m.id
}

// Enhanced Page interface methods
func (m *mockPage) GetTitle() string {
	return m.title
}

func (m *mockPage) GetHelp() []key.Binding {
	return m.help
}

func (m *mockPage) HandleNavigation(msg tea.Msg) (Page, tea.Cmd) {
	if m.handleNavFn != nil {
		return m.handleNavFn(msg)
	}
	return m, nil
}

func (m *mockPage) OnFocus() tea.Cmd {
	return m.focusCmd
}

func (m *mockPage) OnBlur() tea.Cmd {
	return m.blurCmd
}

func TestPageInterfaceImplementation(t *testing.T) {
	// Test that mockPage implements the enhanced Page interface
	var _ Page = &mockPage{}

	// Create a mock page instance
	page := &mockPage{
		id:    "test",
		title: "Test Page",
		help: []key.Binding{
			key.NewBinding(
				key.WithKeys("q"),
				key.WithHelp("q", "quit"),
			),
		},
	}

	// Test all interface methods
	if page.GetID() != "test" {
		t.Errorf("Expected ID 'test', got '%s'", page.GetID())
	}

	if page.GetTitle() != "Test Page" {
		t.Errorf("Expected title 'Test Page', got '%s'", page.GetTitle())
	}

	help := page.GetHelp()
	if len(help) != 1 {
		t.Errorf("Expected 1 help binding, got %d", len(help))
	}

	// Test navigation handling
	navPage, navCmd := page.HandleNavigation(nil)
	if navPage != page {
		t.Error("Expected same page from HandleNavigation")
	}
	if navCmd != nil {
		t.Error("Expected nil command from HandleNavigation")
	}

	// Test focus/blur
	focusCmd := page.OnFocus()
	if focusCmd != nil {
		t.Error("Expected nil command from OnFocus")
	}

	blurCmd := page.OnBlur()
	if blurCmd != nil {
		t.Error("Expected nil command from OnBlur")
	}
}