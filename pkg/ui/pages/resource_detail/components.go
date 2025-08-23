package resource_detail

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Keys handles key bindings for the resource detail page
type Keys struct {
	bindings keyMap
}

type keyMap struct {
	Back key.Binding
	Quit key.Binding
}

// NewKeys creates a new Keys instance
func NewKeys() *Keys {
	return &Keys{
		bindings: keyMap{
			Back: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "back"),
			),
			Quit: key.NewBinding(
				key.WithKeys("ctrl+c", "q"),
				key.WithHelp("ctrl+c/q", "quit"),
			),
		},
	}
}

// HandleKey processes key events
func (k *Keys) HandleKey(model *Model, msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, k.bindings.Back):
		return func() tea.Msg {
			return PageChangeMsg{PageID: "main"}
		}
	case key.Matches(msg, k.bindings.Quit):
		return tea.Quit
	}
	return nil
}

// PageChangeMsg represents a page change message
type PageChangeMsg struct {
	PageID string
	Data   interface{}
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