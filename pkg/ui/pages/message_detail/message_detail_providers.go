package messagedetail

import (
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MessageDetailContentProvider implements the ContentProvider interface for message detail view
type MessageDetailContentProvider struct {
	model          *Model
	tabs           []string
	activeTab      int
	keyEditor      textarea.Model
	valueEditor    textarea.Model
	headerEditor   textarea.Model
	metadataEditor textarea.Model
	focusedEditor  int // 0 = key, 1 = value (only for Content tab)
	width          int
	height         int
}

// NewMessageDetailContentProvider creates a new content provider for message detail
func NewMessageDetailContentProvider(model *Model) *MessageDetailContentProvider {
	provider := &MessageDetailContentProvider{
		model:         model,
		tabs:          []string{"Content", "Headers", "Metadata"},
		activeTab:     0,
		focusedEditor: 1, // Start with value editor focused
	}

	// Initialize editors
	provider.keyEditor = provider.newTextarea("Key", true)
	provider.valueEditor = provider.newTextarea("Value", false)
	provider.headerEditor = provider.newTextarea("Headers", false)
	provider.metadataEditor = provider.newTextarea("Metadata", false)

	// Set initial content
	provider.updateEditorContent()

	return provider
}

// newTextarea creates a new textarea with consistent styling
func (m *MessageDetailContentProvider) newTextarea(placeholder string, readOnly bool) textarea.Model {
	t := textarea.New()
	t.Prompt = ""
	t.Placeholder = placeholder
	t.ShowLineNumbers = true
	t.Cursor.Style = cursorStyle
	t.FocusedStyle.Placeholder = focusedPlaceholderStyle
	t.BlurredStyle.Placeholder = placeholderStyle
	t.FocusedStyle.CursorLine = cursorLineStyle
	t.FocusedStyle.Base = focusedBorderStyle
	t.BlurredStyle.Base = blurredBorderStyle
	t.FocusedStyle.EndOfBuffer = endOfBufferStyle
	t.BlurredStyle.EndOfBuffer = endOfBufferStyle
	t.KeyMap.DeleteWordBackward.SetEnabled(false)
	t.KeyMap.LineNext = key.NewBinding(key.WithKeys("down"))
	t.KeyMap.LinePrevious = key.NewBinding(key.WithKeys("up"))

	// Make editors read-only for viewing message content
	if readOnly {
		t.KeyMap.CharacterBackward.SetEnabled(false)
		t.KeyMap.CharacterForward.SetEnabled(false)
		t.KeyMap.DeleteAfterCursor.SetEnabled(false)
		t.KeyMap.DeleteBeforeCursor.SetEnabled(false)
		t.KeyMap.DeleteCharacterBackward.SetEnabled(false)
		t.KeyMap.DeleteCharacterForward.SetEnabled(false)
		t.KeyMap.DeleteWordBackward.SetEnabled(false)
		t.KeyMap.DeleteWordForward.SetEnabled(false)
		t.KeyMap.InsertNewline.SetEnabled(false)
		t.KeyMap.Paste.SetEnabled(false)
	}

	t.Blur()
	return t
}

// updateEditorContent updates the content of all editors
func (m *MessageDetailContentProvider) updateEditorContent() {
	if m.model == nil {
		return
	}

	// Update key editor
	m.keyEditor.SetValue(m.model.GetFormattedKey())

	// Update value editor
	m.valueEditor.SetValue(m.model.GetFormattedValue())

	// Update headers editor
	if len(m.model.message.Headers) > 0 {
		var headerContent strings.Builder
		for _, header := range m.model.message.Headers {
			headerContent.WriteString(fmt.Sprintf("%s: %s\n", header.Key, header.Value))
		}
		m.headerEditor.SetValue(headerContent.String())
	} else {
		m.headerEditor.SetValue("No headers available")
	}

	// Update metadata editor
	info := m.model.GetMessageInfo()
	var metadataContent strings.Builder
	for key, value := range info {
		metadataContent.WriteString(fmt.Sprintf("%-15s: %s\n", key, value))
	}
	m.metadataEditor.SetValue(metadataContent.String())
}

// RenderContent renders the message detail content with tabs
func (m *MessageDetailContentProvider) RenderContent(width, height int) string {
	if m.model == nil {
		return "No message data available"
	}

	// Update dimensions and editor content
	m.width = width
	m.height = height
	m.updateEditorContent()
	m.sizeEditors()

	// Clear expired status messages
	if m.model.statusMsg != "" && time.Since(m.model.statusTime) > 3*time.Second {
		m.model.statusMsg = ""
	}

	// Render tabs
	tabsContent := m.renderTabs(width)

	// Render active tab content
	activeContent := m.renderActiveTabContent(width-10, height-15) // Reserve space for tabs

	// Combine tabs and content
	result := tabsContent + "\n" + activeContent

	// Add status message if present
	if m.model.statusMsg != "" {
		result += "\n\n" + fmt.Sprintf("📋 %s", m.model.statusMsg)
	}

	return result
}

// renderTabs renders the tab navigation
func (m *MessageDetailContentProvider) renderTabs(width int) string {
	var renderedTabs []string

	for i, tab := range m.tabs {
		var style lipgloss.Style
		isFirst, isLast, isActive := i == 0, i == len(m.tabs)-1, i == m.activeTab
		if isActive {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}
		border, _, _, _, _ := style.GetBorder()
		if isFirst && isActive {
			border.BottomLeft = ""
		} else if isFirst && !isActive {
			border.BottomLeft = ""
		} else if isLast && isActive {
			border.BottomRight = ""
		} else if isLast && !isActive {
			border.BottomRight = ""
		}
		style = style.Border(border)
		renderedTabs = append(renderedTabs, style.Render(tab))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
}

// renderActiveTabContent renders the content for the currently active tab
func (m *MessageDetailContentProvider) renderActiveTabContent(width, height int) string {
	var content string

	switch m.activeTab {
	case 0: // Content tab (Key and Value split view)
		content = m.renderSplitContentTab()
	case 1: // Headers tab
		content = m.headerEditor.View()
	case 2: // Metadata tab
		content = m.metadataEditor.View()
	default:
		content = "Unknown tab"
	}

	// Apply window styling
	tabWidth := width - windowStyle.GetHorizontalFrameSize()
	if tabWidth < 0 {
		tabWidth = width
	}

	return content
}

// renderSplitContentTab renders the key and value editors side by side
func (m *MessageDetailContentProvider) renderSplitContentTab() string {
	return lipgloss.JoinHorizontal(lipgloss.Top, m.keyEditor.View(), m.valueEditor.View())
}

// sizeEditors updates the size of all editors based on current dimensions
func (m *MessageDetailContentProvider) sizeEditors() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	editorHeight := m.height - 15 // Reserve space for tabs and borders
	if editorHeight < 5 {
		editorHeight = 5
	}

	switch m.activeTab {
	case 0: // Content tab - split view
		editorWidth := (m.width - 6) / 2 // Split width between key and value
		if editorWidth < 10 {
			editorWidth = 10
		}
		m.keyEditor.SetWidth(editorWidth)
		m.keyEditor.SetHeight(editorHeight)
		m.valueEditor.SetWidth(editorWidth)
		m.valueEditor.SetHeight(editorHeight)

		// Set focus based on focusedEditor
		if m.focusedEditor == 0 {
			m.keyEditor.Focus()
			m.valueEditor.Blur()
		} else {
			m.keyEditor.Blur()
			m.valueEditor.Focus()
		}

	case 1: // Headers tab
		m.headerEditor.SetWidth(m.width - 6)
		m.headerEditor.SetHeight(editorHeight)
		m.headerEditor.Focus()

	case 2: // Metadata tab
		m.metadataEditor.SetWidth(m.width - 6)
		m.metadataEditor.SetHeight(editorHeight)
		m.metadataEditor.Focus()
	}
}

// renderContentTab renders the key and value content
func (m *MessageDetailContentProvider) renderContentTab(width int) string {
	var content strings.Builder

	// Key section
	content.WriteString(m.renderKeySection(width))
	content.WriteString("\n\n")

	// Value section
	content.WriteString(m.renderValueSection(width))

	return content.String()
}

// renderHeadersTab renders the headers content
func (m *MessageDetailContentProvider) renderHeadersTab() string {
	if len(m.model.message.Headers) == 0 {
		return "No headers available"
	}
	return m.renderHeaders()
}

// renderMetadataTab renders the metadata content
func (m *MessageDetailContentProvider) renderMetadataTab() string {
	return m.renderMetadata()
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
		case "shift+tab":
			// Navigate to next tab (cycle through)
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
			return nil
		case "f":
			// Toggle display format (only on Content tab)
			if m.activeTab == 0 {
				m.model.ToggleDisplayFormat()
			}
			return nil
		case "tab":
			// Switch focus between key and value (only on Content tab)
			if m.activeTab == 0 {
				m.focusedEditor = 1 - m.focusedEditor // Toggle between 0 and 1
				m.model.SwitchFocus()
				m.sizeEditors() // Update focus state
			}
			return nil
		case "c":
			// Copy content to clipboard (only on Content tab)
			if m.activeTab == 0 {
				m.model.CopyContentWithFeedback()
			}
			return nil
		case "r":
			// Refresh schema info
			return m.model.LoadSchemaInfoAsync()
		case "up", "k", "down", "j":
			// Handle scrolling in editors
			return m.updateActiveEditor(msg)
		}
	}

	// Update the active editor with the message
	return m.updateActiveEditor(msg)
}

// updateActiveEditor updates the currently active editor with the given message
func (m *MessageDetailContentProvider) updateActiveEditor(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch m.activeTab {
	case 0: // Content tab
		if m.focusedEditor == 0 {
			m.keyEditor, cmd = m.keyEditor.Update(msg)
		} else {
			m.valueEditor, cmd = m.valueEditor.Update(msg)
		}
	case 1: // Headers tab
		m.headerEditor, cmd = m.headerEditor.Update(msg)
	case 2: // Metadata tab
		m.metadataEditor, cmd = m.metadataEditor.Update(msg)
	}

	return cmd
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

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Tab styling functions and variables
func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

var (
	inactiveTabBorder = tabBorderWithBottom("", "", "")
	activeTabBorder   = tabBorderWithBottom("", " ", "")
	highlightColor    = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	inactiveTabStyle  = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(highlightColor).Padding(0, 1)
	activeTabStyle    = inactiveTabStyle.Border(activeTabBorder, true)
	windowStyle       = lipgloss.NewStyle().BorderForeground(highlightColor).Padding(2, 1).Border(lipgloss.NormalBorder()).UnsetBorderTop()

	// Editor styles
	cursorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	cursorLineStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("57")).
			Foreground(lipgloss.Color("230"))
	placeholderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("238"))
	endOfBufferStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("235"))
	focusedPlaceholderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("99"))
	focusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(highlightColor)
	blurredBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238"))
)
