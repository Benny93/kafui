package detail

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the detail page state for viewing individual messages
type Model struct {
	// Data
	topicName string
	message   api.Message

	// State
	dimensions core.Dimensions
	error      error

	// Components
	handlers *Handlers
	keys     *Keys
	view     *View

	// Display configuration
	displayFormat MessageDisplayFormat
	showHeaders   bool
	showMetadata  bool
}

// MessageDisplayFormat represents how the message should be displayed
type MessageDisplayFormat struct {
	ValueFormat string // "raw", "json", "pretty", "hex"
	KeyFormat   string
	WrapLines   bool
	ShowBytes   bool
}

// NewModel creates a new detail page model
func NewModel(topicName string, message api.Message) *Model {
	m := &Model{
		topicName: topicName,
		message:   message,
		displayFormat: MessageDisplayFormat{
			ValueFormat: "pretty",
			KeyFormat:   "raw",
			WrapLines:   true,
			ShowBytes:   false,
		},
		showHeaders:  true,
		showMetadata: true,
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
	return "detail"
}

// Business logic methods

// GetMessageInfo returns formatted message information
func (m *Model) GetMessageInfo() map[string]string {
	return map[string]string{
		"Topic":      m.topicName,
		"Partition":  fmt.Sprintf("%d", m.message.Partition),
		"Offset":     fmt.Sprintf("%d", m.message.Offset),
		"Key Size":   fmt.Sprintf("%d bytes", len(m.message.Key)),
		"Value Size": fmt.Sprintf("%d bytes", len(m.message.Value)),
	}
}

// GetFormattedKey returns the formatted message key
func (m *Model) GetFormattedKey() string {
	if m.message.Key == "" {
		return "<null>"
	}

	switch m.displayFormat.KeyFormat {
	case "hex":
		return fmt.Sprintf("%x", m.message.Key)
	case "json":
		// Try to format as JSON
		return string(m.message.Key) // Simplified for now
	default:
		return string(m.message.Key)
	}
}

// GetFormattedValue returns the formatted message value
func (m *Model) GetFormattedValue() string {
	if m.message.Value == "" {
		return "<null>"
	}

	switch m.displayFormat.ValueFormat {
	case "hex":
		return fmt.Sprintf("%x", m.message.Value)
	case "json", "pretty":
		// Try to format as JSON
		return string(m.message.Value) // Simplified for now
	default:
		return string(m.message.Value)
	}
}

// ToggleDisplayFormat cycles through display formats
func (m *Model) ToggleDisplayFormat() {
	switch m.displayFormat.ValueFormat {
	case "raw":
		m.displayFormat.ValueFormat = "pretty"
	case "pretty":
		m.displayFormat.ValueFormat = "json"
	case "json":
		m.displayFormat.ValueFormat = "hex"
	case "hex":
		m.displayFormat.ValueFormat = "raw"
	default:
		m.displayFormat.ValueFormat = "raw"
	}
}

// ToggleHeaders toggles header display
func (m *Model) ToggleHeaders() {
	m.showHeaders = !m.showHeaders
}

// ToggleMetadata toggles metadata display
func (m *Model) ToggleMetadata() {
	m.showMetadata = !m.showMetadata
}
