package messagedetail

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/atotto/clipboard"
)

// Model represents the detail page state for viewing individual messages
type Model struct {
	// Data
	topicName    string
	message      api.Message
	dataSource   api.KafkaDataSource
	schemaInfo   *api.MessageSchemaInfo

	// State
	dimensions core.Dimensions
	error      error
	statusMsg  string
	statusTime time.Time

	// Components
	handlers *Handlers
	keys     *Keys
	view     *View

	// Viewport components for scrollable content
	keyViewport   viewport.Model
	valueViewport viewport.Model
	
	// Focus management
	focusedViewport string // "key" or "value"

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
func NewModel(dataSource api.KafkaDataSource, topicName string, message api.Message) *Model {
	// Initialize viewport components
	keyViewport := viewport.New(30, 10)
	valueViewport := viewport.New(50, 20)
	
	// Set default content
	keyContent := "<null>"
	if message.Key != "" {
		keyContent = message.Key
	}
	
	valueContent := "<null>"
	if message.Value != "" {
		valueContent = message.Value
	}
	
	keyViewport.SetContent(addLineNumbers(keyContent))
	valueViewport.SetContent(addLineNumbers(valueContent))

	m := &Model{
		dataSource: dataSource,
		topicName:  topicName,
		message:    message,
		displayFormat: MessageDisplayFormat{
			ValueFormat: "pretty",
			KeyFormat:   "raw",
			WrapLines:   true,
			ShowBytes:   false,
		},
		showHeaders:  true,
		showMetadata: true,
		keyViewport:   keyViewport,
		valueViewport: valueViewport,
		focusedViewport: "value", // Value viewport focused by default
	}

	// Load schema information if available (lazy loading)
	// m.loadSchemaInfo() - moved to lazy loading

	// Initialize components
	m.handlers = NewHandlers(m)
	m.keys = NewKeys()
	m.view = NewView()

	return m
}

// loadSchemaInfo loads schema information for the message (lazy loading)
func (m *Model) loadSchemaInfo() {
	if m.dataSource == nil {
		return
	}
	
	// Only load if schema IDs are present
	if m.message.KeySchemaID == "" && m.message.ValueSchemaID == "" {
		return
	}
	
	// Load schema information from data source
	schemaInfo, err := m.dataSource.GetMessageSchemaInfo(m.message.KeySchemaID, m.message.ValueSchemaID)
	if err != nil {
		// Log error but don't fail - schema info is optional
		return
	}
	
	m.schemaInfo = schemaInfo
}

// GetSchemaInfo returns schema information, loading it lazily if needed
func (m *Model) GetSchemaInfo() *api.MessageSchemaInfo {
	if m.schemaInfo == nil && (m.message.KeySchemaID != "" || m.message.ValueSchemaID != "") {
		m.loadSchemaInfo()
	}
	return m.schemaInfo
}

// LoadSchemaInfoAsync loads schema information asynchronously for better UX
func (m *Model) LoadSchemaInfoAsync() tea.Cmd {
	// Only load if we have schema IDs and haven't loaded yet
	if m.schemaInfo != nil || (m.message.KeySchemaID == "" && m.message.ValueSchemaID == "") {
		return nil
	}
	
	return func() tea.Msg {
		m.loadSchemaInfo()
		return nil // Could return a custom message for UI updates if needed
	}
}

// Init implements the Page interface
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update implements the Page interface
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Clear status message after 3 seconds
	if m.statusMsg != "" && time.Since(m.statusTime) > 3*time.Second {
		m.statusMsg = ""
	}
	
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
	
	// Update viewport dimensions
	if width > 0 && height > 0 {
		m.keyViewport.Width = width/2 - 10
		m.keyViewport.Height = height/3 - 5
		m.valueViewport.Width = width/2 - 10
		m.valueViewport.Height = height/3 - 5
	}
}

// GetID implements the Page interface
func (m *Model) GetID() string {
	return "message_detail"
}

// GetTitle implements the Page interface
func (m *Model) GetTitle() string {
	if m.topicName != "" {
		return fmt.Sprintf("Message Detail: %s", m.topicName)
	}
	return "Message Detail"
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
	// Handle focus gain - reload schema info when page becomes active
	return m.LoadSchemaInfoAsync()
}

// OnBlur implements the Page interface
func (m *Model) OnBlur() tea.Cmd {
	// Handle focus loss
	return nil
}

// Business logic methods

// GetMessageInfo returns formatted message information
func (m *Model) GetMessageInfo() map[string]string {
	info := map[string]string{
		"Topic":      m.topicName,
		"Partition":  fmt.Sprintf("%d", m.message.Partition),
		"Offset":     fmt.Sprintf("%d", m.message.Offset),
		"Key Size":   fmt.Sprintf("%d bytes", len(m.message.Key)),
		"Value Size": fmt.Sprintf("%d bytes", len(m.message.Value)),
		"Headers":    fmt.Sprintf("%d", len(m.message.Headers)),
	}
	
	// Add schema information if available
	if m.message.KeySchemaID != "" {
		info["Key Schema ID"] = m.message.KeySchemaID
	}
	if m.message.ValueSchemaID != "" {
		info["Value Schema ID"] = m.message.ValueSchemaID
	}
	
	return info
}

// GetFormattedKey returns the formatted message key
func (m *Model) GetFormattedKey() string {
	if m.message.Key == "" {
		return "<null>"
	}

	switch m.displayFormat.KeyFormat {
	case "hex":
		return fmt.Sprintf("%x", m.message.Key)
	case "json", "pretty":
		// Try to format as JSON
		return m.formatAsJSON(m.message.Key)
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
		return m.formatAsJSON(m.message.Value)
	default:
		return string(m.message.Value)
	}
}

// formatAsJSON attempts to parse and pretty print JSON content
func (m *Model) formatAsJSON(content string) string {
	var parsed interface{}
	
	// Try to unmarshal as JSON
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		// If parsing fails, try to unescape and parse again
		// This handles cases where JSON is double-encoded
		var unescapedContent string
		if err := json.Unmarshal([]byte(content), &unescapedContent); err == nil {
			// Try parsing the unescaped content
			if err := json.Unmarshal([]byte(unescapedContent), &parsed); err == nil {
				// Successfully parsed unescaped content
				content = unescapedContent
			} else {
				// Use the unescaped content as a string
				parsed = unescapedContent
			}
		} else {
			// If parsing fails, return original content
			return content
		}
	}
	
	// Marshal with indentation for pretty printing
	pretty, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		// If pretty printing fails, return original content
		return content
	}
	
	// Apply syntax highlighting if enabled
	if m.displayFormat.ValueFormat == "pretty" {
		return m.highlightJSON(string(pretty))
	}
	
	return string(pretty)
}

// highlightJSON applies syntax highlighting to JSON content
func (m *Model) highlightJSON(jsonStr string) string {
	// For now, return the JSON as-is to avoid ANSI escape sequence issues in viewport
	// TODO: Implement proper syntax highlighting that works with viewport
	return jsonStr
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

// addLineNumbers adds line numbers to content
func addLineNumbers(content string) string {
	if content == "<null>" {
		return content
	}
	
	lines := strings.Split(content, "\n")
	numberedLines := make([]string, len(lines))
	
	for i, line := range lines {
		lineNumber := fmt.Sprintf("%4d ", i+1)
		numberedLines[i] = lineNumber + line
	}
	
	return strings.Join(numberedLines, "\n")
}

// SwitchFocus switches focus between key and value viewports
func (m *Model) SwitchFocus() {
	if m.focusedViewport == "key" {
		m.focusedViewport = "value"
	} else {
		m.focusedViewport = "key"
	}
}

// CopyContent copies the content of the focused viewport to clipboard
func (m *Model) CopyContent() error {
	var content string
	switch m.focusedViewport {
	case "key":
		content = m.GetFormattedKey()
	case "value":
		content = m.GetFormattedValue()
	default:
		content = m.GetFormattedValue()
	}
	
	// Try to copy to clipboard
	return clipboard.WriteAll(content)
}

// CopyContentWithFeedback copies content and returns a status message
func (m *Model) CopyContentWithFeedback() string {
	err := m.CopyContent()
	if err != nil {
		status := fmt.Sprintf("Failed to copy content: %v", err)
		m.statusMsg = status
		m.statusTime = time.Now()
		return status
	}
	status := "Content copied to clipboard"
	m.statusMsg = status
	m.statusTime = time.Now()
	return status
}
