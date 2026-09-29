package messagedetail

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the detail page state for viewing individual messages (kept for compatibility)
type Model struct {
	// Common context
	common *core.Common

	// Data
	topicName  string
	message    api.Message
	dataSource api.KafkaDataSource
	schemaInfo *api.MessageSchemaInfo
	// schemaLoading is set while a LoadSchemaInfoAsync cmd is in flight, and
	// schemaAttempted once one has finished, so a failed lookup is not
	// repeated on every focus.
	schemaLoading   bool
	schemaAttempted bool

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
		showHeaders:     true,
		showMetadata:    true,
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

	// Per-message metadata (MSG-22): timestamp, timestamp type, serde names,
	// null-ness. Only shown when populated.
	if !m.message.Timestamp.IsZero() {
		info["Timestamp"] = m.message.Timestamp.Format(time.RFC3339)
	}
	if m.message.TimestampType != "" {
		info["Timestamp Type"] = string(m.message.TimestampType)
	}
	if m.message.KeySerde != "" {
		info["Key Serde"] = m.message.KeySerde
	}
	if m.message.ValueSerde != "" {
		info["Value Serde"] = m.message.ValueSerde
	}
	if m.message.KeyNull {
		info["Key"] = "<null>"
	}
	if m.message.ValueNull {
		info["Value"] = "<null>"
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

	// Colouring happens in the viewer (editor.HighlightJSON), which is
	// line-based and so composes with wrapping, line numbers and search.
	return string(pretty)
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

// addLineNumbers adds line numbers to multi-line content
func addLineNumbers(content string) string {
	if content == "<null>" || content == "" {
		return content
	}

	lines := strings.Split(content, "\n")
	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = fmt.Sprintf("%4d %s", i+1, line)
	}
	return strings.Join(result, "\n")
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

// GetSchemaInfo returns the schema information loaded so far, or nil. It never
// fetches: it is called from View, and a schema-registry lookup there would
// block rendering. Loading goes through LoadSchemaInfoAsync.
func (m *Model) GetSchemaInfo() *api.MessageSchemaInfo {
	return m.schemaInfo
}

// LoadSchemaInfoAsync fetches the message's schema information in a tea.Cmd.
// The result comes back as a SchemaLoadedMsg and is applied in Update, so the
// cmd goroutine never touches the model. A load runs at most once per page
// (successful or not) unless ReloadSchemaInfoAsync asks again.
func (m *Model) LoadSchemaInfoAsync() tea.Cmd {
	if m.dataSource == nil || m.schemaLoading || m.schemaAttempted ||
		(m.message.KeySchemaID == "" && m.message.ValueSchemaID == "") {
		return nil
	}
	m.schemaLoading = true

	ds := m.dataSource
	keyID, valueID := m.message.KeySchemaID, m.message.ValueSchemaID
	return func() tea.Msg {
		info, err := ds.GetMessageSchemaInfo(keyID, valueID)
		return SchemaLoadedMsg{
			KeySchemaID:   keyID,
			ValueSchemaID: valueID,
			Info:          info,
			Err:           err,
			Success:       err == nil && info != nil,
		}
	}
}

// ReloadSchemaInfoAsync re-fetches schema information on explicit request,
// even when an earlier load already ran.
func (m *Model) ReloadSchemaInfoAsync() tea.Cmd {
	if m.schemaLoading {
		return nil
	}
	m.schemaAttempted = false
	return m.LoadSchemaInfoAsync()
}

// applySchemaLoaded stores a load result. A result for other schema IDs (a
// load started by another message's page) is ignored. A failed load keeps
// what was already shown; schema info is optional.
func (m *Model) applySchemaLoaded(msg SchemaLoadedMsg) {
	if msg.KeySchemaID != m.message.KeySchemaID || msg.ValueSchemaID != m.message.ValueSchemaID {
		return
	}
	m.schemaLoading = false
	m.schemaAttempted = true
	if msg.Err == nil && msg.Info != nil {
		m.schemaInfo = msg.Info
	}
}

// SchemaLoadedMsg carries the result of LoadSchemaInfoAsync.
type SchemaLoadedMsg struct {
	KeySchemaID   string
	ValueSchemaID string
	Info          *api.MessageSchemaInfo
	Err           error
	Success       bool
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
	// The topic is already the previous breadcrumb; name the message itself.
	return fmt.Sprintf("Message p%d @ %d", m.message.Partition, m.message.Offset)
}

// OnFocus handles focus gain
func (m *Model) OnFocus() tea.Cmd {
	return m.LoadSchemaInfoAsync()
}

// OnBlur handles focus loss. A load still in flight reports to whichever page
// is current when it finishes, so forget it; the next OnFocus starts another.
func (m *Model) OnBlur() tea.Cmd {
	m.schemaLoading = false
	return nil
}

// GetHelpKeyBindings returns this page's bindings, read from the single
// registry so help cannot disagree with behavior.
func GetHelpKeyBindings() []key.Binding {
	return keys.Help(keys.ScopeListContent)
}

// MessageDetailPageModel wraps the ReusableApp with message detail-specific providers
type MessageDetailPageModel struct {
	common          *core.Common
	topicName       string
	message         api.Message
	reusableApp     *templateui.ReusableApp
	contentProvider *MessageDetailContentProvider
	detailModel     *Model
}

// NewMessageDetailPageModel creates a new message detail page model using the template system
// Deprecated: Use NewMessageDetailPageModelWithCommon for new code
func NewMessageDetailPageModel(dataSource api.KafkaDataSource, topicName string, message api.Message) *MessageDetailPageModel {
	common := core.NewCommon(dataSource)
	return NewMessageDetailPageModelWithCommon(common, topicName, message)
}

// NewMessageDetailPageModelWithCommon creates a new message detail page model using the Common context pattern
func NewMessageDetailPageModelWithCommon(common *core.Common, topicName string, message api.Message) *MessageDetailPageModel {
	// Create the core model (reuse existing logic)
	detailModel := NewModel(common.DataSource, topicName, message)
	// Set common context for layout system access
	detailModel.common = common

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

	// Set the key map for the footer using centralized keys
	reusableApp.SetKeyMap(keys.Hints(keys.ScopeListContent))

	return &MessageDetailPageModel{
		common:          common,
		topicName:       topicName,
		message:         message,
		reusableApp:     reusableApp,
		contentProvider: contentProvider,
		detailModel:     detailModel,
	}
}

// GetCommon returns the shared context
func (m *MessageDetailPageModel) GetCommon() *core.Common {
	return m.common
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
	return fmt.Sprintf("detail:%s:%d:%d", m.topicName, m.message.Partition, m.message.Offset)
}

// GetTitle implements the Page interface
func (m *MessageDetailPageModel) GetTitle() string {
	return m.detailModel.GetTitle()
}

// GetHelp implements the Page interface
func (m *MessageDetailPageModel) GetHelp() []key.Binding {
	// Return key bindings for help using centralized keys
	return GetHelpKeyBindings()
}

// HandleNavigation implements the Page interface. Esc is the shell's: it
// unwinds and navigates back, and only reaches this page while the search
// prompt is open, where it must close the prompt rather than leave the page.
func (m *MessageDetailPageModel) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
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
