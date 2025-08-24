package resource_detail

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the resource detail page state
type Model struct {
	// Data
	resourceItem shared.ResourceItem
	resourceType string
	
	// State
	dimensions core.Dimensions
	error      error
	
	// Components
	handlers *Handlers
	keys     *Keys
	view     *View
}

// NewModel creates a new resource detail page model
func NewModel(resourceItem shared.ResourceItem, resourceType string) *Model {
	m := &Model{
		resourceItem: resourceItem,
		resourceType: resourceType,
	}

	// Initialize components
	m.handlers = NewHandlers(m)
	m.keys = NewKeys()
	m.view = NewView()

	return m
}

// Init implements the Page interface
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update implements the Page interface
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.handlers.Handle(m, msg)
}

// View implements the Page interface
func (m *Model) View() string {
	return m.view.Render(m)
}

// SetDimensions implements the Page interface
func (m *Model) SetDimensions(width, height int) {
	m.dimensions = core.Dimensions{Width: width, Height: height}
	m.view.SetDimensions(width, height)
}

// GetID implements the Page interface
func (m *Model) GetID() string {
	return "resource_detail"
}

// GetTitle implements the Page interface
func (m *Model) GetTitle() string {
	if m.resourceItem != nil {
		return fmt.Sprintf("Resource Detail: %s", m.resourceItem.GetID())
	}
	return "Resource Detail"
}

// GetHelp implements the Page interface
func (m *Model) GetHelp() []key.Binding {
	if m.keys != nil {
		return m.keys.GetKeyBindings()
	}
	return []key.Binding{}
}

// HandleNavigation implements the Page interface
func (m *Model) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
	// Handle page-specific navigation
	return m, nil
}

// OnFocus implements the Page interface
func (m *Model) OnFocus() tea.Cmd {
	// Handle focus gain
	return nil
}

// OnBlur implements the Page interface
func (m *Model) OnBlur() tea.Cmd {
	// Handle focus loss
	return nil
}

// Business logic methods

// GetResourceDetails returns detailed information about the resource
func (m *Model) GetResourceDetails() map[string]string {
	if m.resourceItem == nil {
		return map[string]string{"Error": "No resource item"}
	}
	return m.resourceItem.GetDetails()
}

// GetResourceValues returns the display values for the resource
func (m *Model) GetResourceValues() []string {
	if m.resourceItem == nil {
		return []string{"No resource"}
	}
	return m.resourceItem.GetValues()
}

// GetResourceID returns the resource identifier
func (m *Model) GetResourceID() string {
	if m.resourceItem == nil {
		return "Unknown"
	}
	return m.resourceItem.GetID()
}