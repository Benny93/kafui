// Package core provides core types and interfaces for the Kafui UI framework.
package core

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/ui/layout"
)

// Component defines the interface that all UI components must implement.
// This pattern is inspired by Elm architecture and Bubble Tea conventions.
type Component interface {
	// Init initializes the component and returns an initial command.
	Init() tea.Cmd

	// Update handles messages and returns updated component and commands.
	Update(msg tea.Msg) (tea.Model, tea.Cmd)

	// View renders the component to a string.
	View() string

	// SetDimensions sets the component's dimensions.
	SetDimensions(width, height int)
}

// BaseComponent provides common functionality for all UI components.
// Embed this struct in your component to get default implementations.
//
// Example:
//
//	type SearchBar struct {
//		BaseComponent
//		textInput textinput.Model
//		// ... other fields
//	}
type BaseComponent struct {
	width  int
	height int
	id     string
}

// NewBaseComponent creates a new BaseComponent with the given dimensions.
func NewBaseComponent(width, height int) BaseComponent {
	return BaseComponent{
		width:  width,
		height: height,
	}
}

// Init provides a default initialization that does nothing.
// Override this method in your component if initialization is needed.
func (b *BaseComponent) Init() tea.Cmd {
	return nil
}

// Update provides a default update that does nothing.
// Override this method in your component to handle messages.
func (b *BaseComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return b, nil
}

// View provides a default view that returns empty string.
// Override this method in your component to render content.
func (b *BaseComponent) View() string {
	return ""
}

// SetDimensions sets the component's dimensions.
// This is used for layout calculations.
func (b *BaseComponent) SetDimensions(width, height int) {
	b.width = width
	b.height = height
}

// GetWidth returns the component's width.
func (b *BaseComponent) GetWidth() int {
	return b.width
}

// GetHeight returns the component's height.
func (b *BaseComponent) GetHeight() int {
	return b.height
}

// SetID sets the component's identifier.
func (b *BaseComponent) SetID(id string) {
	b.id = id
}

// GetID returns the component's identifier.
func (b *BaseComponent) GetID() string {
	return b.id
}

// ComponentConfig holds common configuration for components.
type ComponentConfig struct {
	// ID is a unique identifier for the component
	ID string

	// Width and Height set initial dimensions
	Width  int
	Height int

	// Common context for accessing shared dependencies
	Common *Common
}

// ApplyConfig applies the configuration to a base component.
func (b *BaseComponent) ApplyConfig(config ComponentConfig) {
	b.id = config.ID
	b.width = config.Width
	b.height = config.Height
}

// ComponentWithLayout extends Component with layout-specific methods.
type ComponentWithLayout interface {
	Component

	// GetLayout returns the component's layout rectangle.
	GetLayout() layout.Rectangle

	// SetLayout sets the component's layout rectangle.
	SetLayout(rect layout.Rectangle)
}
