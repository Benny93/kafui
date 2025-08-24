package core

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// FocusManager handles focus management for components within pages
type FocusManager struct {
	focusableComponents []Focusable
	currentFocus        int
	enabled             bool
}

// Focusable represents a component that can receive focus
type Focusable interface {
	// Focus gives focus to the component
	Focus() tea.Cmd
	
	// Blur removes focus from the component
	Blur() tea.Cmd
	
	// IsFocused returns whether the component currently has focus
	IsFocused() bool
	
	// GetID returns a unique identifier for the component
	GetID() string
	
	// CanFocus returns whether the component can receive focus
	CanFocus() bool
}

// FocusableComponent provides a base implementation of Focusable
type FocusableComponent struct {
	id       string
	focused  bool
	canFocus bool
}

// NewFocusableComponent creates a new focusable component
func NewFocusableComponent(id string) *FocusableComponent {
	return &FocusableComponent{
		id:       id,
		focused:  false,
		canFocus: true,
	}
}

// Focus implements Focusable
func (f *FocusableComponent) Focus() tea.Cmd {
	f.focused = true
	return nil
}

// Blur implements Focusable
func (f *FocusableComponent) Blur() tea.Cmd {
	f.focused = false
	return nil
}

// IsFocused implements Focusable
func (f *FocusableComponent) IsFocused() bool {
	return f.focused
}

// GetID implements Focusable
func (f *FocusableComponent) GetID() string {
	return f.id
}

// CanFocus implements Focusable
func (f *FocusableComponent) CanFocus() bool {
	return f.canFocus
}

// SetCanFocus sets whether the component can receive focus
func (f *FocusableComponent) SetCanFocus(canFocus bool) {
	f.canFocus = canFocus
}

// NewFocusManager creates a new focus manager
func NewFocusManager() *FocusManager {
	return &FocusManager{
		focusableComponents: make([]Focusable, 0),
		currentFocus:        -1,
		enabled:             true,
	}
}

// AddComponent adds a focusable component to the manager
func (fm *FocusManager) AddComponent(component Focusable) {
	fm.focusableComponents = append(fm.focusableComponents, component)
}

// RemoveComponent removes a focusable component from the manager
func (fm *FocusManager) RemoveComponent(id string) {
	for i, component := range fm.focusableComponents {
		if component.GetID() == id {
			// Remove from slice
			fm.focusableComponents = append(fm.focusableComponents[:i], fm.focusableComponents[i+1:]...)
			
			// Adjust current focus if necessary
			if i < fm.currentFocus {
				fm.currentFocus--
			} else if i == fm.currentFocus {
				// Current focused component was removed, focus next available
				if fm.currentFocus >= len(fm.focusableComponents) {
					fm.currentFocus = len(fm.focusableComponents) - 1
				}
				fm.focusCurrentComponent()
			}
			break
		}
	}
}

// FocusNext moves focus to the next focusable component
func (fm *FocusManager) FocusNext() tea.Cmd {
	if !fm.enabled || len(fm.focusableComponents) == 0 {
		return nil
	}
	
	// Blur current component
	if fm.currentFocus >= 0 && fm.currentFocus < len(fm.focusableComponents) {
		fm.focusableComponents[fm.currentFocus].Blur()
	}
	
	// Find next focusable component
	startIndex := fm.currentFocus
	for i := 0; i < len(fm.focusableComponents); i++ {
		fm.currentFocus = (fm.currentFocus + 1) % len(fm.focusableComponents)
		if fm.focusableComponents[fm.currentFocus].CanFocus() {
			return fm.focusCurrentComponent()
		}
		
		// Prevent infinite loop
		if fm.currentFocus == startIndex {
			break
		}
	}
	
	return nil
}

// FocusPrevious moves focus to the previous focusable component
func (fm *FocusManager) FocusPrevious() tea.Cmd {
	if !fm.enabled || len(fm.focusableComponents) == 0 {
		return nil
	}
	
	// Blur current component
	if fm.currentFocus >= 0 && fm.currentFocus < len(fm.focusableComponents) {
		fm.focusableComponents[fm.currentFocus].Blur()
	}
	
	// Find previous focusable component
	startIndex := fm.currentFocus
	for i := 0; i < len(fm.focusableComponents); i++ {
		fm.currentFocus--
		if fm.currentFocus < 0 {
			fm.currentFocus = len(fm.focusableComponents) - 1
		}
		
		if fm.focusableComponents[fm.currentFocus].CanFocus() {
			return fm.focusCurrentComponent()
		}
		
		// Prevent infinite loop
		if fm.currentFocus == startIndex {
			break
		}
	}
	
	return nil
}

// FocusComponent focuses a specific component by ID
func (fm *FocusManager) FocusComponent(id string) tea.Cmd {
	if !fm.enabled {
		return nil
	}
	
	// Blur current component
	if fm.currentFocus >= 0 && fm.currentFocus < len(fm.focusableComponents) {
		fm.focusableComponents[fm.currentFocus].Blur()
	}
	
	// Find and focus the specified component
	for i, component := range fm.focusableComponents {
		if component.GetID() == id && component.CanFocus() {
			fm.currentFocus = i
			return fm.focusCurrentComponent()
		}
	}
	
	return nil
}

// GetFocusedComponent returns the currently focused component
func (fm *FocusManager) GetFocusedComponent() Focusable {
	if fm.currentFocus >= 0 && fm.currentFocus < len(fm.focusableComponents) {
		return fm.focusableComponents[fm.currentFocus]
	}
	return nil
}

// GetFocusedComponentID returns the ID of the currently focused component
func (fm *FocusManager) GetFocusedComponentID() string {
	if component := fm.GetFocusedComponent(); component != nil {
		return component.GetID()
	}
	return ""
}

// SetEnabled enables or disables focus management
func (fm *FocusManager) SetEnabled(enabled bool) {
	fm.enabled = enabled
	if !enabled {
		// Blur all components when disabled
		for _, component := range fm.focusableComponents {
			component.Blur()
		}
	}
}

// IsEnabled returns whether focus management is enabled
func (fm *FocusManager) IsEnabled() bool {
	return fm.enabled
}

// HandleKeyMsg handles key messages for focus navigation
func (fm *FocusManager) HandleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	if !fm.enabled {
		return nil
	}
	
	switch {
	case key.Matches(msg, DefaultGlobalKeys.NextPage): // Tab
		return fm.FocusNext()
	case key.Matches(msg, DefaultGlobalKeys.PrevPage): // Shift+Tab
		return fm.FocusPrevious()
	}
	
	return nil
}

// focusCurrentComponent focuses the current component
func (fm *FocusManager) focusCurrentComponent() tea.Cmd {
	if fm.currentFocus >= 0 && fm.currentFocus < len(fm.focusableComponents) {
		return fm.focusableComponents[fm.currentFocus].Focus()
	}
	return nil
}

// Clear removes all components and resets focus
func (fm *FocusManager) Clear() {
	// Blur all components
	for _, component := range fm.focusableComponents {
		component.Blur()
	}
	
	fm.focusableComponents = fm.focusableComponents[:0]
	fm.currentFocus = -1
}

// GetComponentCount returns the number of focusable components
func (fm *FocusManager) GetComponentCount() int {
	return len(fm.focusableComponents)
}

// GetFocusableComponents returns all focusable components
func (fm *FocusManager) GetFocusableComponents() []Focusable {
	// Return a copy to prevent external modification
	components := make([]Focusable, len(fm.focusableComponents))
	copy(components, fm.focusableComponents)
	return components
}