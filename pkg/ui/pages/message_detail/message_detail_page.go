package messagedetail

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/atotto/clipboard"
)

// Model represents the detail page state for viewing individual messages (kept for compatibility)
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

	// Display configuration
	displayFormat MessageDisplayFormat
	showHeaders   bool
	showMetadata  bool
	
	// Focus management
	focusedViewport string // "key" or "value"
}

// MessageDisplayFormat represents how the message should be displayed
type MessageDisplayFormat struct {
	ValueFormat string // "raw", "json", "pretty", "hex"
	KeyFormat   string
	WrapLines   bool
	ShowBytes   bool
}

// NewModel creates a new detail page model (kept for compatibility)
func NewModel(dataSource api.KafkaDataSource, topicName string, message api.Message) *Model {
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
		focusedViewport: "value", // Value viewport focused by default
	}

	return m
}

// Business logic methods for the Model

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
	// Return the JSON as-is since the template system handles styling
	// Syntax highlighting can be added later through the template system's styling
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

// GetSchemaInfo returns schema information, loading it lazily if needed
func (m *Model) GetSchemaInfo() *api.MessageSchemaInfo {
	if m.schemaInfo == nil && (m.message.KeySchemaID != "" || m.message.ValueSchemaID != "") {
		m.loadSchemaInfo()
	}
	return m.schemaInfo
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

// LoadSchemaInfoAsync loads schema information asynchronously for better UX
func (m *Model) LoadSchemaInfoAsync() tea.Cmd {
	// Only load if we have schema IDs and haven't loaded yet
	if m.schemaInfo != nil || (m.message.KeySchemaID == "" && m.message.ValueSchemaID == "") {
		return nil
	}
	
	return func() tea.Msg {
		m.loadSchemaInfo()
		// Return a custom message to trigger UI refresh
		return SchemaLoadedMsg{Success: m.schemaInfo != nil}
	}
}

// SchemaLoadedMsg indicates that schema loading has completed
type SchemaLoadedMsg struct {
	Success bool
}

// SetDimensions sets the model dimensions
func (m *Model) SetDimensions(width, height int) {
	m.dimensions = core.Dimensions{Width: width, Height: height}
}

// GetID returns the page ID
func (m *Model) GetID() string {
	return "message_detail"
}

// GetTitle returns the page title
func (m *Model) GetTitle() string {
	if m.topicName != "" {
		return fmt.Sprintf("Message Detail: %s", m.topicName)
	}
	return "Message Detail"
}

// OnFocus handles focus gain
func (m *Model) OnFocus() tea.Cmd {
	return m.LoadSchemaInfoAsync()
}

// OnBlur handles focus loss
func (m *Model) OnBlur() tea.Cmd {
	return nil
}

// KeyMap defines the key bindings for the message detail page
type KeyMap struct {
	NextTab        key.Binding
	PrevTab        key.Binding
	ToggleFormat   key.Binding
	SwitchFocus    key.Binding
	Copy           key.Binding
	ScrollUp       key.Binding
	ScrollDown     key.Binding
	Back           key.Binding
	Quit           key.Binding
	Help           key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.NextTab, k.ToggleFormat, k.Copy, k.Back, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NextTab, k.ToggleFormat, k.SwitchFocus}, // first column
		{k.Copy, k.ScrollUp, k.ScrollDown, k.Back, k.Help, k.Quit}, // second column
	}
}

// DefaultKeyMap contains the default key bindings for the message detail page
var DefaultKeyMap = KeyMap{
	NextTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "next tab"),
	),
	ToggleFormat: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "toggle format (Content tab)"),
	),
	SwitchFocus: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch focus (Content tab)"),
	),
	Copy: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "copy content (Content tab)"),
	),
	ScrollUp: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "scroll up"),
	),
	ScrollDown: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "scroll down"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
}

// Additional key bindings not in the main KeyMap but handled by the content provider
var AdditionalKeys = struct {
	Refresh key.Binding
}{
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh schema info"),
	),
}

// MessageDetailPageModel wraps the ReusableApp with message detail-specific providers
type MessageDetailPageModel struct {
	dataSource      api.KafkaDataSource
	topicName       string
	message         api.Message
	reusableApp     *templateui.ReusableApp
	contentProvider *MessageDetailContentProvider
	detailModel     *Model
}

// NewMessageDetailPageModel creates a new message detail page model using the template system
func NewMessageDetailPageModel(dataSource api.KafkaDataSource, topicName string, message api.Message) *MessageDetailPageModel {
	// Create the core model (reuse existing logic)
	detailModel := NewModel(dataSource, topicName, message)

	// Create message detail-specific providers
	contentProvider := NewMessageDetailContentProvider(detailModel)
	headerProvider := NewMessageDetailHeaderDataProvider(detailModel)

	// Create sidebar sections
	sidebarSections := []providers.SidebarSection{
		NewMessageInfoSection(detailModel),
		NewSchemaInfoSection(detailModel),
	}

	// Create app configuration using template providers
	config := &providers.AppConfig{
		ContentProvider:             contentProvider,
		HeaderDataProvider:          headerProvider,
		SidebarSections:             sidebarSections,
		ShowSidebarByDefault:        true,
		CompactModeWidthBreakpoint:  120,
		CompactModeHeightBreakpoint: 30,
	}

	// Create the reusable app with our message detail providers
	reusableApp := templateui.NewReusableApp(config)

	// Set the key map for the footer
	reusableApp.SetKeyMap(DefaultKeyMap)

	return &MessageDetailPageModel{
		dataSource:      dataSource,
		topicName:       topicName,
		message:         message,
		reusableApp:     reusableApp,
		contentProvider: contentProvider,
		detailModel:     detailModel,
	}
}

// NewModel creates a new message detail page model (alias for compatibility)
func NewMessageDetailPage(dataSource api.KafkaDataSource, topicName string, message api.Message) *MessageDetailPageModel {
	return NewMessageDetailPageModel(dataSource, topicName, message)
}

// Init implements the Page interface
func (m *MessageDetailPageModel) Init() tea.Cmd {
	return m.reusableApp.Init()
}

// Update implements the Page interface
func (m *MessageDetailPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Delegate to the reusable app
	updatedApp, cmd := m.reusableApp.Update(msg)
	if updatedReusableApp, ok := updatedApp.(*templateui.ReusableApp); ok {
		m.reusableApp = updatedReusableApp
	}
	return m, cmd
}

// View implements the Page interface
func (m *MessageDetailPageModel) View() string {
	return m.reusableApp.View()
}

// SetDimensions implements the Page interface
func (m *MessageDetailPageModel) SetDimensions(width, height int) {
	// Delegate to the reusable app by sending a WindowSizeMsg
	m.reusableApp.Update(tea.WindowSizeMsg{Width: width, Height: height})
	// Also update the detail model dimensions for compatibility
	m.detailModel.SetDimensions(width, height)
}

// GetID implements the Page interface
func (m *MessageDetailPageModel) GetID() string {
	return "message_detail"
}

// GetTitle implements the Page interface
func (m *MessageDetailPageModel) GetTitle() string {
	return m.detailModel.GetTitle()
}

// GetHelp implements the Page interface
func (m *MessageDetailPageModel) GetHelp() []key.Binding {
	// Return key bindings for help using the DefaultKeyMap
	return []key.Binding{
		DefaultKeyMap.NextTab,
		DefaultKeyMap.PrevTab,
		DefaultKeyMap.ToggleFormat,
		DefaultKeyMap.SwitchFocus,
		DefaultKeyMap.Copy,
		AdditionalKeys.Refresh,
		DefaultKeyMap.ScrollUp,
		DefaultKeyMap.ScrollDown,
		DefaultKeyMap.Back,
		DefaultKeyMap.Quit,
		DefaultKeyMap.Help,
	}
}

// HandleNavigation implements the Page interface
func (m *MessageDetailPageModel) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
	// Handle navigation messages
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Navigate back to topic page
			return m, core.NewPageChangeMsg("topic", map[string]interface{}{
				"name": m.topicName,
			})
		}
	}
	return m, nil
}

// OnFocus implements the Page interface
func (m *MessageDetailPageModel) OnFocus() tea.Cmd {
	// Handle focus gain - reload schema info when page becomes active
	return m.detailModel.OnFocus()
}

// OnBlur implements the Page interface
func (m *MessageDetailPageModel) OnBlur() tea.Cmd {
	// Handle focus loss
	return m.detailModel.OnBlur()
}

// GetMessage returns the current message (for compatibility)
func (m *MessageDetailPageModel) GetMessage() api.Message {
	return m.message
}

// GetTopicName returns the topic name (for compatibility)
func (m *MessageDetailPageModel) GetTopicName() string {
	return m.topicName
}

// GetDetailModel returns the underlying detail model (for compatibility)
func (m *MessageDetailPageModel) GetDetailModel() *Model {
	return m.detailModel
}