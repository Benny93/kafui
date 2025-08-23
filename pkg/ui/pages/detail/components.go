package detail

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Keys handles key bindings for the detail page
type Keys struct {
	bindings keyMap
}

type keyMap struct {
	Back           key.Binding
	Quit           key.Binding
	ToggleFormat   key.Binding
	ToggleHeaders  key.Binding
	ToggleMetadata key.Binding
	Copy           key.Binding
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
			ToggleFormat: key.NewBinding(
				key.WithKeys("f"),
				key.WithHelp("f", "toggle format"),
			),
			ToggleHeaders: key.NewBinding(
				key.WithKeys("h"),
				key.WithHelp("h", "toggle headers"),
			),
			ToggleMetadata: key.NewBinding(
				key.WithKeys("m"),
				key.WithHelp("m", "toggle metadata"),
			),
			Copy: key.NewBinding(
				key.WithKeys("c"),
				key.WithHelp("c", "copy content"),
			),
		},
	}
}

// HandleKey processes key events
func (k *Keys) HandleKey(model *Model, msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, k.bindings.Back):
		return func() tea.Msg {
			return PageChangeMsg{PageID: "topic"}
		}
	case key.Matches(msg, k.bindings.Quit):
		return tea.Quit
	case key.Matches(msg, k.bindings.ToggleFormat):
		model.ToggleDisplayFormat()
		return nil
	case key.Matches(msg, k.bindings.ToggleHeaders):
		model.ToggleHeaders()
		return nil
	case key.Matches(msg, k.bindings.ToggleMetadata):
		model.ToggleMetadata()
		return nil
	case key.Matches(msg, k.bindings.Copy):
		// TODO: Implement copy functionality
		return nil
	}
	return nil
}

// PageChangeMsg represents a page change message
type PageChangeMsg struct {
	PageID string
	Data   interface{}
}

// Handlers manages event handling for the detail page
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

// View handles rendering for the detail page
type View struct {
	dimensions core.Dimensions
}

// NewView creates a new View instance
func NewView() *View {
	return &View{}
}

// Render renders the detail page view
func (v *View) Render(model *Model) string {
	if model.dimensions.Width == 0 {
		return "Loading message details..."
	}

	// Simple implementation for now
	content := fmt.Sprintf(
		"Topic: %s\nMessage Details\n\nKey: %s\nValue: %s\n\nPress 'esc' to go back",
		model.topicName,
		model.GetFormattedKey(),
		model.GetFormattedValue(),
	)

	return content
}

// SetDimensions updates the view dimensions
func (v *View) SetDimensions(width, height int) {
	v.dimensions = core.Dimensions{Width: width, Height: height}
}

