package resource_detail

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
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

	// Template system
	reusableApp *templateui.ReusableApp
}

// NewModel creates a new resource detail page model
// Deprecated: Use NewModelWithCommon for new code
func NewModel(resourceItem shared.ResourceItem, resourceType string) *Model {
	common := core.NewCommon(nil) // Note: dataSource not used in resource detail
	return NewModelWithCommon(resourceItem, resourceType, common)
}

// NewModelWithCommon creates a new resource detail page model using the Common context pattern
func NewModelWithCommon(resourceItem shared.ResourceItem, resourceType string, common *core.Common) *Model {
	m := &Model{
		resourceItem: resourceItem,
		resourceType: resourceType,
	}

	// Initialize components
	m.handlers = NewHandlers(m)
	m.keys = NewKeys()
	m.view = NewView()

	// Create content provider
	contentProvider := &ResourceDetailContentProvider{model: m}

	// Create app configuration
	config := &providers.AppConfig{
		ContentProvider:      contentProvider,
		ShowSidebarByDefault: false,
	}

	// Create the reusable app
	m.reusableApp = templateui.NewReusableApp(config)

	// Set key map for footer
	m.reusableApp.SetKeyMap(m.keys.bindings)

	return m
}

// ResourceDetailContentProvider provides content for the resource detail page
type ResourceDetailContentProvider struct {
	model *Model
}

func (p *ResourceDetailContentProvider) RenderContent(width, height int) string {
	return p.model.view.Render(p.model)
}

func (p *ResourceDetailContentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd {
	return nil
}

func (p *ResourceDetailContentProvider) InitContent() tea.Cmd {
	return nil
}

func (p *ResourceDetailContentProvider) IsInputMode() bool {
	return false
}

// GetContentSize returns the estimated content size for scrollbar calculation
func (p *ResourceDetailContentProvider) GetContentSize(width int) int {
	// Estimate based on content lines
	// This is a simplified estimation - adjust based on actual content
	return 20
}

// Init implements the Page interface
func (m *Model) Init() tea.Cmd {
	return m.reusableApp.Init()
}

// Update implements the Page interface
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Delegate to the reusable app
	updatedApp, cmd := m.reusableApp.Update(msg)
	if updatedReusableApp, ok := updatedApp.(*templateui.ReusableApp); ok {
		m.reusableApp = updatedReusableApp
	}

	// Also handle messages specifically if needed via handlers
	_, handlerCmd := m.handlers.Handle(m, msg)

	return m, tea.Batch(cmd, handlerCmd)
}

// View implements the Page interface
func (m *Model) View() string {
	return m.reusableApp.View()
}

// SetDimensions implements the Page interface
func (m *Model) SetDimensions(width, height int) {
	m.dimensions = core.Dimensions{Width: width, Height: height}
	m.view.SetDimensions(width, height)
	m.reusableApp.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

// GetID implements the Page interface
func (m *Model) GetID() string {
	if m.resourceItem != nil {
		return "resource_detail:" + m.resourceItem.GetID()
	}
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

// GetCommon returns the shared context (returns nil for resource detail as it doesn't use Common)
func (m *Model) GetCommon() *core.Common {
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
