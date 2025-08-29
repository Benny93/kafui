package messagedetail

import (
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	tea "github.com/charmbracelet/bubbletea"
)

// MessageDetailContentProvider implements the ContentProvider interface for message detail view
type MessageDetailContentProvider struct {
	model *Model
}

// NewMessageDetailContentProvider creates a new content provider for message detail
func NewMessageDetailContentProvider(model *Model) *MessageDetailContentProvider {
	return &MessageDetailContentProvider{
		model: model,
	}
}

// RenderContent renders the message detail content
func (m *MessageDetailContentProvider) RenderContent(width, height int) string {
	if m.model == nil {
		return "No message data available"
	}

	var content strings.Builder
	
	// Clear expired status messages
	if m.model.statusMsg != "" && time.Since(m.model.statusTime) > 3*time.Second {
		m.model.statusMsg = ""
	}
	
	// Message metadata section
	if m.model.showMetadata {
		content.WriteString(m.renderMetadata())
		content.WriteString("\n\n")
	}

	// Headers section
	if m.model.showHeaders && len(m.model.message.Headers) > 0 {
		content.WriteString(m.renderHeaders())
		content.WriteString("\n\n")
	}

	// Key section
	content.WriteString(m.renderKeySection(width))
	content.WriteString("\n\n")

	// Value section
	content.WriteString(m.renderValueSection(width))

	// Status message
	if m.model.statusMsg != "" {
		content.WriteString("\n\n")
		content.WriteString(fmt.Sprintf("📋 %s", m.model.statusMsg))
	}

	return content.String()
}

// renderMetadata renders the message metadata
func (m *MessageDetailContentProvider) renderMetadata() string {
	info := m.model.GetMessageInfo()
	var metadata strings.Builder
	
	metadata.WriteString("Message Metadata:\n")
	metadata.WriteString("─────────────────\n")
	
	for key, value := range info {
		metadata.WriteString(fmt.Sprintf("%-15s: %s\n", key, value))
	}
	
	return metadata.String()
}

// renderHeaders renders the message headers
func (m *MessageDetailContentProvider) renderHeaders() string {
	var headers strings.Builder
	
	headers.WriteString("Headers:\n")
	headers.WriteString("────────\n")
	
	for key, value := range m.model.message.Headers {
		headers.WriteString(fmt.Sprintf("%-20s: %s\n", key, value))
	}
	
	return headers.String()
}

// renderKeySection renders the message key section
func (m *MessageDetailContentProvider) renderKeySection(width int) string {
	var keySection strings.Builder
	
	title := "Key"
	if m.model.focusedViewport == "key" {
		title += " [FOCUSED]"
	}
	title += fmt.Sprintf(" (%s)", m.model.displayFormat.KeyFormat)
	
	keySection.WriteString(title + ":\n")
	keySection.WriteString(strings.Repeat("─", min(len(title)+1, width-4)) + "\n")
	
	formattedKey := m.model.GetFormattedKey()
	keySection.WriteString(formattedKey)
	
	return keySection.String()
}

// renderValueSection renders the message value section
func (m *MessageDetailContentProvider) renderValueSection(width int) string {
	var valueSection strings.Builder
	
	title := "Value"
	if m.model.focusedViewport == "value" {
		title += " [FOCUSED]"
	}
	title += fmt.Sprintf(" (%s)", m.model.displayFormat.ValueFormat)
	
	valueSection.WriteString(title + ":\n")
	valueSection.WriteString(strings.Repeat("─", min(len(title)+1, width-4)) + "\n")
	
	formattedValue := m.model.GetFormattedValue()
	valueSection.WriteString(formattedValue)
	
	return valueSection.String()
}

// HandleContentUpdate handles content updates
func (m *MessageDetailContentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd {
	if m.model == nil {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "f":
			// Toggle display format
			m.model.ToggleDisplayFormat()
			return nil
		case "h":
			// Toggle headers
			m.model.ToggleHeaders()
			return nil
		case "m":
			// Toggle metadata
			m.model.ToggleMetadata()
			return nil
		case "tab":
			// Switch focus between key and value
			m.model.SwitchFocus()
			return nil
		case "c":
			// Copy content to clipboard
			m.model.CopyContentWithFeedback()
			return nil
		case "r":
			// Refresh schema info
			return m.model.LoadSchemaInfoAsync()
		case "up", "k":
			// Scroll up (handled by viewport in template system)
			return nil
		case "down", "j":
			// Scroll down (handled by viewport in template system)
			return nil
		}
	}

	return nil
}

// InitContent initializes the content provider
func (m *MessageDetailContentProvider) InitContent() tea.Cmd {
	if m.model != nil {
		return m.model.LoadSchemaInfoAsync()
	}
	return nil
}

// MessageDetailHeaderDataProvider implements the HeaderDataProvider interface for message detail
type MessageDetailHeaderDataProvider struct {
	model *Model
}

// NewMessageDetailHeaderDataProvider creates a new header data provider for message detail
func NewMessageDetailHeaderDataProvider(model *Model) *MessageDetailHeaderDataProvider {
	return &MessageDetailHeaderDataProvider{
		model: model,
	}
}

// GetBrandName returns the brand name
func (m *MessageDetailHeaderDataProvider) GetBrandName() string {
	return "Kafui™"
}

// GetAppName returns the application name
func (m *MessageDetailHeaderDataProvider) GetAppName() string {
	return "Message Detail"
}

// GetStatusData returns status information for the header
func (m *MessageDetailHeaderDataProvider) GetStatusData() map[string]interface{} {
	status := make(map[string]interface{})
	
	if m.model != nil {
		status["topic"] = m.model.topicName
		status["partition"] = fmt.Sprintf("P%d", m.model.message.Partition)
		status["offset"] = fmt.Sprintf("O%d", m.model.message.Offset)
		status["format"] = m.model.displayFormat.ValueFormat
		
		// Add current time as a fallback since message timestamp may not be available
		status["time"] = time.Now().Format("15:04:05")
	}
	
	return status
}

// HandleHeaderUpdate handles header updates
func (m *MessageDetailHeaderDataProvider) HandleHeaderUpdate(msg tea.Msg) tea.Cmd {
	return nil
}

// InitHeader initializes the header provider
func (m *MessageDetailHeaderDataProvider) InitHeader() tea.Cmd {
	return nil
}

// MessageInfoSection implements SidebarSection for message information
type MessageInfoSection struct {
	model *Model
}

// NewMessageInfoSection creates a new message info sidebar section
func NewMessageInfoSection(model *Model) *MessageInfoSection {
	return &MessageInfoSection{
		model: model,
	}
}

// GetTitle returns the section title
func (m *MessageInfoSection) GetTitle() string {
	return "Message Info"
}

// RenderItems returns the items to display in this section
func (m *MessageInfoSection) RenderItems(maxItems, width int) []providers.SidebarItem {
	if m.model == nil {
		return []providers.SidebarItem{}
	}

	items := []providers.SidebarItem{
		{
			Icon:   "●",
			Text:   "Topic",
			Value:  m.model.topicName,
			Status: "info",
		},
		{
			Icon:   "●",
			Text:   "Partition",
			Value:  fmt.Sprintf("%d", m.model.message.Partition),
			Status: "info",
		},
		{
			Icon:   "●",
			Text:   "Offset",
			Value:  fmt.Sprintf("%d", m.model.message.Offset),
			Status: "info",
		},
		{
			Icon:   "●",
			Text:   "Key Size",
			Value:  fmt.Sprintf("%d bytes", len(m.model.message.Key)),
			Status: "muted",
		},
		{
			Icon:   "●",
			Text:   "Value Size",
			Value:  fmt.Sprintf("%d bytes", len(m.model.message.Value)),
			Status: "muted",
		},
	}

	if len(m.model.message.Headers) > 0 {
		items = append(items, providers.SidebarItem{
			Icon:   "●",
			Text:   "Headers",
			Value:  fmt.Sprintf("%d", len(m.model.message.Headers)),
			Status: "info",
		})
	}

	// Add current time as viewing time since message timestamp may not be available
	items = append(items, providers.SidebarItem{
		Icon:   "●",
		Text:   "Viewed At",
		Value:  time.Now().Format("15:04:05"),
		Status: "muted",
	})

	// Limit to maxItems
	if len(items) > maxItems {
		items = items[:maxItems]
	}

	return items
}

// HandleSectionUpdate handles section updates
func (m *MessageInfoSection) HandleSectionUpdate(msg tea.Msg) tea.Cmd {
	return nil
}

// InitSection initializes the section
func (m *MessageInfoSection) InitSection() tea.Cmd {
	return nil
}

// RefreshSection refreshes the section data
func (m *MessageInfoSection) RefreshSection() tea.Cmd {
	return nil
}

// SchemaInfoSection implements SidebarSection for schema information
type SchemaInfoSection struct {
	model *Model
}

// NewSchemaInfoSection creates a new schema info sidebar section
func NewSchemaInfoSection(model *Model) *SchemaInfoSection {
	return &SchemaInfoSection{
		model: model,
	}
}

// GetTitle returns the section title
func (s *SchemaInfoSection) GetTitle() string {
	return "Schema Info"
}

// RenderItems returns the items to display in this section
func (s *SchemaInfoSection) RenderItems(maxItems, width int) []providers.SidebarItem {
	if s.model == nil {
		return []providers.SidebarItem{}
	}

	items := []providers.SidebarItem{}

	if s.model.message.KeySchemaID != "" {
		items = append(items, providers.SidebarItem{
			Icon:   "●",
			Text:   "Key Schema",
			Value:  s.model.message.KeySchemaID,
			Status: "info",
		})
	}

	if s.model.message.ValueSchemaID != "" {
		items = append(items, providers.SidebarItem{
			Icon:   "●",
			Text:   "Value Schema",
			Value:  s.model.message.ValueSchemaID,
			Status: "info",
		})
	}

	// Add schema info if available
	schemaInfo := s.model.GetSchemaInfo()
	if schemaInfo != nil {
		if schemaInfo.KeySchema != nil {
			items = append(items, providers.SidebarItem{
				Icon:   "○",
				Text:   "Key Schema",
				Value:  "Available",
				Status: "muted",
			})
		}
		if schemaInfo.ValueSchema != nil {
			items = append(items, providers.SidebarItem{
				Icon:   "○",
				Text:   "Value Schema",
				Value:  "Available",
				Status: "muted",
			})
		}
	}

	if len(items) == 0 {
		items = append(items, providers.SidebarItem{
			Icon:   "○",
			Text:   "No Schema",
			Value:  "Available",
			Status: "muted",
		})
	}

	// Limit to maxItems
	if len(items) > maxItems {
		items = items[:maxItems]
	}

	return items
}

// HandleSectionUpdate handles section updates
func (s *SchemaInfoSection) HandleSectionUpdate(msg tea.Msg) tea.Cmd {
	return nil
}

// InitSection initializes the section
func (s *SchemaInfoSection) InitSection() tea.Cmd {
	return nil
}

// RefreshSection refreshes the section data
func (s *SchemaInfoSection) RefreshSection() tea.Cmd {
	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}