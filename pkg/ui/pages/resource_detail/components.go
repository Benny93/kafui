package resource_detail

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Keys resolves this page's keys through the single binding registry. The page
// is a plain content view: back and quit are reserved globals the shell owns,
// so nothing is handled here beyond what the registry declares for content.
type Keys struct {
	scope keys.Scope
}

// NewKeys creates a new Keys instance bound to the content scope.
func NewKeys() *Keys {
	return &Keys{scope: keys.ScopeContent}
}

// GetKeyBindings returns the page's bindings for the help overlay.
func (k *Keys) GetKeyBindings() []key.Binding {
	var out []key.Binding
	for _, b := range keys.Default.InScope(k.scope) {
		out = append(out, b.KeyBinding())
	}
	return out
}

// KeyScope reports the scope the shell resolves this page's keys against.
func (k *Keys) KeyScope() keys.Scope { return k.scope }

// HandleKey processes key events the shell forwarded to this page.
func (k *Keys) HandleKey(model *Model, msg tea.KeyMsg) tea.Cmd {
	action, bound := keys.Default.Resolve(k.scope, msg.String())
	if !bound {
		return nil
	}
	_ = action // the shell owns every action this page reacts to today
	return nil
}

// Handlers manages event handling for the resource detail page
type Handlers struct {
	model *Model
}

// NewHandlers creates a new Handlers instance
func NewHandlers(model *Model) *Handlers {
	return &Handlers{model: model}
}

// Handle routes messages to appropriate handlers
func (h *Handlers) Handle(model *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	h.model = model

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		model.SetDimensions(msg.Width, msg.Height)
		return model, nil
	case tea.KeyMsg:
		cmd := model.keys.HandleKey(model, msg)
		return model, cmd
	}

	return model, nil
}

// View handles rendering for the resource detail page
type View struct {
	dimensions core.Dimensions
}

// NewView creates a new View instance
func NewView() *View {
	return &View{}
}

// Render renders the resource detail page view
func (v *View) Render(model *Model) string {
	if model.dimensions.Width == 0 {
		return "Loading resource details..."
	}

	// Build content
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Resource Details - %s\n", strings.ToUpper(model.resourceType)))
	content.WriteString(fmt.Sprintf("ID: %s\n\n", model.GetResourceID()))

	// Add details
	details := model.GetResourceDetails()
	for key, value := range details {
		content.WriteString(fmt.Sprintf("%s: %s\n", key, value))
	}

	content.WriteString("\n\nPress 'esc' to go back")

	return content.String()
}

// SetDimensions updates the view dimensions
func (v *View) SetDimensions(width, height int) {
	v.dimensions = core.Dimensions{Width: width, Height: height}
}
