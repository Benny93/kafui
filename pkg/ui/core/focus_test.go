package core

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// mockFocusableComponent for testing
type mockFocusableComponent struct {
	*FocusableComponent
	focusCallCount int
	blurCallCount  int
}

func newMockFocusableComponent(id string) *mockFocusableComponent {
	return &mockFocusableComponent{
		FocusableComponent: NewFocusableComponent(id),
		focusCallCount:     0,
		blurCallCount:      0,
	}
}

func (m *mockFocusableComponent) Focus() tea.Cmd {
	m.focusCallCount++
	return m.FocusableComponent.Focus()
}

func (m *mockFocusableComponent) Blur() tea.Cmd {
	m.blurCallCount++
	return m.FocusableComponent.Blur()
}

func TestNewFocusManager(t *testing.T) {
	fm := NewFocusManager()
	
	if fm == nil {
		t.Fatal("NewFocusManager returned nil")
	}
	
	if !fm.IsEnabled() {
		t.Error("Expected focus manager to be enabled by default")
	}
	
	if fm.GetComponentCount() != 0 {
		t.Errorf("Expected 0 components, got %d", fm.GetComponentCount())
	}
	
	if fm.GetFocusedComponentID() != "" {
		t.Errorf("Expected no focused component, got %s", fm.GetFocusedComponentID())
	}
}

func TestFocusableComponent(t *testing.T) {
	component := NewFocusableComponent("test-component")
	
	if component.GetID() != "test-component" {
		t.Errorf("Expected ID 'test-component', got '%s'", component.GetID())
	}
	
	if component.IsFocused() {
		t.Error("Expected component to not be focused initially")
	}
	
	if !component.CanFocus() {
		t.Error("Expected component to be focusable by default")
	}
	
	// Test focus
	component.Focus()
	if !component.IsFocused() {
		t.Error("Expected component to be focused after Focus()")
	}
	
	// Test blur
	component.Blur()
	if component.IsFocused() {
		t.Error("Expected component to not be focused after Blur()")
	}
	
	// Test SetCanFocus
	component.SetCanFocus(false)
	if component.CanFocus() {
		t.Error("Expected component to not be focusable after SetCanFocus(false)")
	}
}

func TestAddComponent(t *testing.T) {
	fm := NewFocusManager()
	component1 := newMockFocusableComponent("component1")
	component2 := newMockFocusableComponent("component2")
	
	fm.AddComponent(component1)
	if fm.GetComponentCount() != 1 {
		t.Errorf("Expected 1 component, got %d", fm.GetComponentCount())
	}
	
	fm.AddComponent(component2)
	if fm.GetComponentCount() != 2 {
		t.Errorf("Expected 2 components, got %d", fm.GetComponentCount())
	}
	
	components := fm.GetFocusableComponents()
	if len(components) != 2 {
		t.Errorf("Expected 2 components from GetFocusableComponents, got %d", len(components))
	}
}

func TestRemoveComponent(t *testing.T) {
	fm := NewFocusManager()
	component1 := newMockFocusableComponent("component1")
	component2 := newMockFocusableComponent("component2")
	component3 := newMockFocusableComponent("component3")
	
	fm.AddComponent(component1)
	fm.AddComponent(component2)
	fm.AddComponent(component3)
	
	// Focus component2
	fm.FocusComponent("component2")
	if fm.GetFocusedComponentID() != "component2" {
		t.Error("Expected component2 to be focused")
	}
	
	// Remove component1 (before focused component)
	fm.RemoveComponent("component1")
	if fm.GetComponentCount() != 2 {
		t.Errorf("Expected 2 components after removal, got %d", fm.GetComponentCount())
	}
	
	// component2 should still be focused (but index adjusted)
	if fm.GetFocusedComponentID() != "component2" {
		t.Error("Expected component2 to still be focused after removing component1")
	}
	
	// Remove the currently focused component
	fm.RemoveComponent("component2")
	if fm.GetComponentCount() != 1 {
		t.Errorf("Expected 1 component after removal, got %d", fm.GetComponentCount())
	}
	
	// Should focus the remaining component
	if fm.GetFocusedComponentID() != "component3" {
		t.Error("Expected component3 to be focused after removing component2")
	}
}

func TestFocusNext(t *testing.T) {
	fm := NewFocusManager()
	component1 := newMockFocusableComponent("component1")
	component2 := newMockFocusableComponent("component2")
	component3 := newMockFocusableComponent("component3")
	
	fm.AddComponent(component1)
	fm.AddComponent(component2)
	fm.AddComponent(component3)
	
	// Focus next should focus first component
	fm.FocusNext()
	if fm.GetFocusedComponentID() != "component1" {
		t.Errorf("Expected component1 to be focused, got %s", fm.GetFocusedComponentID())
	}
	if component1.focusCallCount != 1 {
		t.Errorf("Expected component1.Focus() to be called once, got %d", component1.focusCallCount)
	}
	
	// Focus next should focus second component
	fm.FocusNext()
	if fm.GetFocusedComponentID() != "component2" {
		t.Errorf("Expected component2 to be focused, got %s", fm.GetFocusedComponentID())
	}
	if component1.blurCallCount != 1 {
		t.Errorf("Expected component1.Blur() to be called once, got %d", component1.blurCallCount)
	}
	if component2.focusCallCount != 1 {
		t.Errorf("Expected component2.Focus() to be called once, got %d", component2.focusCallCount)
	}
	
	// Focus next should focus third component
	fm.FocusNext()
	if fm.GetFocusedComponentID() != "component3" {
		t.Errorf("Expected component3 to be focused, got %s", fm.GetFocusedComponentID())
	}
	
	// Focus next should wrap around to first component
	fm.FocusNext()
	if fm.GetFocusedComponentID() != "component1" {
		t.Errorf("Expected component1 to be focused after wrap-around, got %s", fm.GetFocusedComponentID())
	}
}

func TestFocusPrevious(t *testing.T) {
	fm := NewFocusManager()
	component1 := newMockFocusableComponent("component1")
	component2 := newMockFocusableComponent("component2")
	component3 := newMockFocusableComponent("component3")
	
	fm.AddComponent(component1)
	fm.AddComponent(component2)
	fm.AddComponent(component3)
	
	// Focus previous should focus last component (wrap around)
	fm.FocusPrevious()
	if fm.GetFocusedComponentID() != "component3" {
		t.Errorf("Expected component3 to be focused, got %s", fm.GetFocusedComponentID())
	}
	
	// Focus previous should focus second component
	fm.FocusPrevious()
	if fm.GetFocusedComponentID() != "component2" {
		t.Errorf("Expected component2 to be focused, got %s", fm.GetFocusedComponentID())
	}
	
	// Focus previous should focus first component
	fm.FocusPrevious()
	if fm.GetFocusedComponentID() != "component1" {
		t.Errorf("Expected component1 to be focused, got %s", fm.GetFocusedComponentID())
	}
}

func TestFocusComponent(t *testing.T) {
	fm := NewFocusManager()
	component1 := newMockFocusableComponent("component1")
	component2 := newMockFocusableComponent("component2")
	component3 := newMockFocusableComponent("component3")
	
	fm.AddComponent(component1)
	fm.AddComponent(component2)
	fm.AddComponent(component3)
	
	// Focus specific component
	fm.FocusComponent("component2")
	if fm.GetFocusedComponentID() != "component2" {
		t.Errorf("Expected component2 to be focused, got %s", fm.GetFocusedComponentID())
	}
	
	// Focus another component
	fm.FocusComponent("component1")
	if fm.GetFocusedComponentID() != "component1" {
		t.Errorf("Expected component1 to be focused, got %s", fm.GetFocusedComponentID())
	}
	if component2.blurCallCount != 1 {
		t.Errorf("Expected component2.Blur() to be called once, got %d", component2.blurCallCount)
	}
	
	// Try to focus non-existent component
	fm.FocusComponent("nonexistent")
	if fm.GetFocusedComponentID() != "component1" {
		t.Error("Expected focus to remain on component1 when focusing non-existent component")
	}
}

func TestSkipNonFocusableComponents(t *testing.T) {
	fm := NewFocusManager()
	component1 := newMockFocusableComponent("component1")
	component2 := newMockFocusableComponent("component2")
	component3 := newMockFocusableComponent("component3")
	
	// Make component2 non-focusable
	component2.SetCanFocus(false)
	
	fm.AddComponent(component1)
	fm.AddComponent(component2)
	fm.AddComponent(component3)
	
	// Focus next should skip component2
	fm.FocusNext()
	if fm.GetFocusedComponentID() != "component1" {
		t.Errorf("Expected component1 to be focused, got %s", fm.GetFocusedComponentID())
	}
	
	fm.FocusNext()
	if fm.GetFocusedComponentID() != "component3" {
		t.Errorf("Expected component3 to be focused (skipping component2), got %s", fm.GetFocusedComponentID())
	}
	
	// component2 should not have been focused
	if component2.focusCallCount > 0 {
		t.Errorf("Expected component2.Focus() to never be called, got %d", component2.focusCallCount)
	}
}

func TestSetEnabled(t *testing.T) {
	fm := NewFocusManager()
	component1 := newMockFocusableComponent("component1")
	component2 := newMockFocusableComponent("component2")
	
	fm.AddComponent(component1)
	fm.AddComponent(component2)
	
	// Focus a component
	fm.FocusNext()
	if !component1.IsFocused() {
		t.Error("Expected component1 to be focused")
	}
	
	// Disable focus manager
	fm.SetEnabled(false)
	if fm.IsEnabled() {
		t.Error("Expected focus manager to be disabled")
	}
	
	// All components should be blurred
	if component1.IsFocused() {
		t.Error("Expected component1 to be blurred when focus manager is disabled")
	}
	
	// Focus operations should not work when disabled
	fm.FocusNext()
	if component2.focusCallCount > 0 {
		t.Error("Expected FocusNext to not work when disabled")
	}
	
	// Re-enable
	fm.SetEnabled(true)
	if !fm.IsEnabled() {
		t.Error("Expected focus manager to be enabled")
	}
}

func TestHandleKeyMsg(t *testing.T) {
	fm := NewFocusManager()
	component1 := newMockFocusableComponent("component1")
	component2 := newMockFocusableComponent("component2")
	
	fm.AddComponent(component1)
	fm.AddComponent(component2)
	
	// Test Tab key (next)
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	cmd := fm.HandleKeyMsg(tabMsg)
	if cmd == nil {
		t.Error("Expected command from HandleKeyMsg with Tab")
	}
	if fm.GetFocusedComponentID() != "component1" {
		t.Error("Expected Tab to focus first component")
	}
	
	// Test Shift+Tab key (previous)
	shiftTabMsg := tea.KeyMsg{Type: tea.KeyShiftTab}
	cmd = fm.HandleKeyMsg(shiftTabMsg)
	if cmd == nil {
		t.Error("Expected command from HandleKeyMsg with Shift+Tab")
	}
	if fm.GetFocusedComponentID() != "component2" {
		t.Error("Expected Shift+Tab to focus previous component")
	}
	
	// Test when disabled
	fm.SetEnabled(false)
	cmd = fm.HandleKeyMsg(tabMsg)
	if cmd != nil {
		t.Error("Expected no command when focus manager is disabled")
	}
}

func TestClear(t *testing.T) {
	fm := NewFocusManager()
	component1 := newMockFocusableComponent("component1")
	component2 := newMockFocusableComponent("component2")
	
	fm.AddComponent(component1)
	fm.AddComponent(component2)
	fm.FocusNext()
	
	// Verify setup
	if fm.GetComponentCount() != 2 {
		t.Error("Expected 2 components before clear")
	}
	if !component1.IsFocused() {
		t.Error("Expected component1 to be focused before clear")
	}
	
	// Clear
	fm.Clear()
	
	// Verify all components are removed and blurred
	if fm.GetComponentCount() != 0 {
		t.Errorf("Expected 0 components after clear, got %d", fm.GetComponentCount())
	}
	if component1.IsFocused() {
		t.Error("Expected component1 to be blurred after clear")
	}
	if fm.GetFocusedComponentID() != "" {
		t.Error("Expected no focused component after clear")
	}
}