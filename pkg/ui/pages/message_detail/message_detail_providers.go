package messagedetail

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/evertras/bubble-table/table"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

const (
	colMdName  = "md_name"
	colMdValue = "md_value"
	colHdrKey  = "hdr_key"
	colHdrVal  = "hdr_val"
)

// MessageDetailContentProvider implements the ContentProvider interface for message detail view
type MessageDetailContentProvider struct {
	model         *Model
	tabs          []string
	activeTab     int
	keyEditor     textarea.Model
	valueEditor   textarea.Model
	headersTable  table.Model
	metadataTable table.Model
	focusedEditor int // 0 = key, 1 = value (only for Content tab)
	width         int
	height        int
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
	provider.headersTable = createHeadersTable()
	provider.metadataTable = createMetadataTable()

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

	// Update headers table
	m.headersTable = m.headersTable.WithRows(buildHeadersRows(m.model.message.Headers))

	// Update metadata table
	m.metadataTable = m.metadataTable.WithRows(buildMetadataRows(m.model.GetMessageInfo()))
}

// RenderContent renders the message detail content with tabs
func (m *MessageDetailContentProvider) RenderContent(width, height int) string {
	if m.model == nil {
		return "No message data available"
	}

	// Use layout system for dimension calculations if available
	var contentWidth, contentHeight int
	
	if m.model.common != nil && m.model.common.Layout != nil {
		// Use layout system
		layout := m.model.common.Layout
		contentWidth = layout.GetAvailableWidth() - 4
		contentHeight = layout.GetAvailableHeight() - 8
	} else {
		// Fallback to ad-hoc calculation
		contentWidth = width - 10
		contentHeight = height - 15
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
	activeContent := m.renderActiveTabContent(contentWidth, contentHeight)

	// Combine tabs and content
	result := tabsContent + "\n" + activeContent

	// Add status message if present
	if m.model.statusMsg != "" {
		result += "\n\n" + fmt.Sprintf("📋 %s", m.model.statusMsg)
	}

	return result
}

// renderTabs renders the tab navigation. Each tab is zone-marked so that
// mouse left-clicks can switch the active tab without using the keyboard.
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

		// Active tab gets a filled dot prefix so it is unmistakable at a glance.
		label := "  " + tab
		if isActive {
			label = "● " + tab
		}

		rendered := zone.Mark(fmt.Sprintf("md-tab-%d", i), style.Render(label))
		renderedTabs = append(renderedTabs, rendered)
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
		content = lipgloss.JoinVertical(lipgloss.Left,
			m.renderSelectedKeyLabel(m.headersTable, colHdrKey),
			m.headersTable.View(),
		)
	case 2: // Metadata tab
		content = lipgloss.JoinVertical(lipgloss.Left,
			m.renderSelectedKeyLabel(m.metadataTable, colMdName),
			m.metadataTable.View(),
		)
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

// renderSelectedKeyLabel returns a styled one-line label showing the full key
// of the currently highlighted row in the given table. If no row is selected
// or the row has no value for keyCol, an empty string is returned so the
// layout doesn't shift.
func (m *MessageDetailContentProvider) renderSelectedKeyLabel(t table.Model, keyCol string) string {
	rows := t.GetVisibleRows()
	idx := t.GetHighlightedRowIndex()
	if idx < 0 || idx >= len(rows) {
		return ""
	}
	raw := rows[idx].Data[keyCol]
	if raw == nil {
		return ""
	}
	key := fmt.Sprintf("%v", raw)
	if key == "" {
		return ""
	}
	label := lipgloss.NewStyle().
		Foreground(stylesPkg.Primary).
		Bold(true).
		Padding(0, 1).
		Render("▶ " + key)
	return label
}

// renderSplitContentTab renders the key and value editors side by side
func (m *MessageDetailContentProvider) renderSplitContentTab() string {
	// Get schema information
	schemaInfo := m.model.GetSchemaInfo()

	// Max width for the schema label — each side gets roughly half the total width
	// minus the "Schema: " label (8 chars) and some padding.
	const schemaLabelPrefix = "Schema: "
	maxSubjectLen := m.width/2 - len(schemaLabelPrefix) - 6
	if maxSubjectLen < 10 {
		maxSubjectLen = 10
	}

	// Prepare key section — show Avro record name, then subject as fallback.
	keySchemaName := "Schema: N/A"
	if schemaInfo != nil && schemaInfo.KeySchema != nil {
		if display := schemaDisplayName(schemaInfo.KeySchema, maxSubjectLen); display != "" {
			keySchemaName = schemaLabelPrefix + display
		}
	}

	// Prepare value section — same priority: record name > subject.
	valueSchemaName := "Schema: N/A"
	if schemaInfo != nil && schemaInfo.ValueSchema != nil {
		if display := schemaDisplayName(schemaInfo.ValueSchema, maxSubjectLen); display != "" {
			valueSchemaName = schemaLabelPrefix + display
		}
	}

	// Create styled schema headers
	schemaHeaderStyle := lipgloss.NewStyle().
		Foreground(stylesPkg.FgMuted).
		Italic(true).
		Padding(0, 1)

	keySchemaHeader := schemaHeaderStyle.Render(keySchemaName)
	valueSchemaHeader := schemaHeaderStyle.Render(valueSchemaName)

	// Combine schema headers with editors
	keySection := lipgloss.JoinVertical(lipgloss.Left, keySchemaHeader, m.keyEditor.View())
	valueSection := lipgloss.JoinVertical(lipgloss.Left, valueSchemaHeader, m.valueEditor.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, keySection, valueSection)
}

// schemaDisplayName returns the best human-readable label for a SchemaInfo.
// Priority: Avro record name (from the schema "name" field) > subject name.
// Long strings are truncated from the left so the most specific suffix stays visible.
func schemaDisplayName(s *api.SchemaInfo, maxLen int) string {
	if s == nil {
		return ""
	}
	if s.RecordName != "" && s.RecordName != "Unknown" {
		return truncateSubjectLeft(s.RecordName, maxLen)
	}
	if s.Subject != "" {
		return truncateSubjectLeft(s.Subject, maxLen)
	}
	return ""
}

// truncateSubjectLeft truncates s from the beginning when it exceeds maxLen,
// replacing the removed prefix with "...". This keeps the most specific
// (rightmost) part of a dotted schema subject name visible.
func truncateSubjectLeft(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	const ellipsis = "..."
	if maxLen <= len(ellipsis) {
		return ellipsis
	}
	return ellipsis + s[len(s)-(maxLen-len(ellipsis)):]
}

// createHeadersTable returns a freshly initialised bubble-table for the
// Headers tab. Column widths are adjusted dynamically in sizeEditors().
func createHeadersTable() table.Model {
	return table.New([]table.Column{
		table.NewColumn(colHdrKey, "Key", 30),
		table.NewColumn(colHdrVal, "Value", 50),
	}).
		WithPageSize(30).
		Focused(true).
		WithBaseStyle(
			lipgloss.NewStyle().BorderForeground(stylesPkg.FgSubtle),
		).
		HeaderStyle(
			lipgloss.NewStyle().Foreground(stylesPkg.FgMuted).Bold(true),
		).
		HighlightStyle(
			lipgloss.NewStyle().
				Background(stylesPkg.Primary).
				Foreground(stylesPkg.BgBase).
				Bold(true),
		)
}

// buildHeadersRows converts a message's headers slice to a slice of table rows.
func buildHeadersRows(headers []api.MessageHeader) []table.Row {
	rows := make([]table.Row, 0, len(headers))
	for _, h := range headers {
		rows = append(rows, table.NewRow(table.RowData{
			colHdrKey: h.Key,
			colHdrVal: h.Value,
		}))
	}
	return rows
}

// copyHeadersAsCSV serialises the headers table as RFC-4180 CSV and writes to clipboard.
func (m *MessageDetailContentProvider) copyHeadersAsCSV() {
	var buf strings.Builder
	buf.WriteString("Key,Value\n")
	for _, h := range m.model.message.Headers {
		k, v := h.Key, h.Value
		if strings.ContainsAny(k, ",\"\n") {
			k = `"` + strings.ReplaceAll(k, `"`, `""`) + `"`
		}
		if strings.ContainsAny(v, ",\"\n") {
			v = `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
		}
		buf.WriteString(k + "," + v + "\n")
	}
	if err := clipboard.WriteAll(buf.String()); err != nil {
		m.model.statusMsg = "Failed to copy: " + err.Error()
		m.model.statusTime = time.Now()
		return
	}
	m.model.statusMsg = "Headers copied as CSV"
	m.model.statusTime = time.Now()
}

// createMetadataTable returns a freshly initialised bubble-table for the
// Metadata tab. Column widths are adjusted dynamically in sizeEditors().
func createMetadataTable() table.Model {
	return table.New([]table.Column{
		table.NewColumn(colMdName, "Name", 20),
		table.NewColumn(colMdValue, "Value", 40),
	}).
		WithPageSize(30).
		Focused(true).
		WithBaseStyle(
			lipgloss.NewStyle().BorderForeground(stylesPkg.FgSubtle),
		).
		HeaderStyle(
			lipgloss.NewStyle().Foreground(stylesPkg.FgMuted).Bold(true),
		).
		HighlightStyle(
			lipgloss.NewStyle().
				Background(stylesPkg.Primary).
				Foreground(stylesPkg.BgBase).
				Bold(true),
		)
}

// buildMetadataRows converts the GetMessageInfo map to a sorted slice of
// table rows so the display order is stable regardless of map iteration.
func buildMetadataRows(info map[string]string) []table.Row {
	keys := make([]string, 0, len(info))
	for k := range info {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	rows := make([]table.Row, 0, len(info))
	for _, k := range keys {
		rows = append(rows, table.NewRow(table.RowData{
			colMdName:  k,
			colMdValue: info[k],
		}))
	}
	return rows
}

// copyMetadataAsCSV serialises the current metadata rows as RFC-4180 CSV
// (header line + one row per field, sorted by name) and writes to clipboard.
func (m *MessageDetailContentProvider) copyMetadataAsCSV() {
	info := m.model.GetMessageInfo()
	keys := make([]string, 0, len(info))
	for k := range info {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	buf.WriteString("Name,Value\n")
	for _, k := range keys {
		v := info[k]
		// Wrap fields that contain commas, quotes, or newlines in double-quotes.
		if strings.ContainsAny(v, ",\"\n") {
			v = `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
		}
		buf.WriteString(k + "," + v + "\n")
	}

	if err := clipboard.WriteAll(buf.String()); err != nil {
		m.model.statusMsg = "Failed to copy: " + err.Error()
		m.model.statusTime = time.Now()
		return
	}
	m.model.statusMsg = "Metadata copied as CSV"
	m.model.statusTime = time.Now()
}

// exportMessageToFile writes the message to a JSON file next to the working
// directory and reports the path in the status line (MSG-29).
func (m *MessageDetailContentProvider) exportMessageToFile() {
	path := shared.DefaultExportPath(m.model.topicName, m.model.message)
	if err := shared.ExportMessageJSON(path, m.model.topicName, m.model.message); err != nil {
		m.model.statusMsg = "Export failed: " + err.Error()
	} else {
		m.model.statusMsg = "Saved message to " + path
	}
	m.model.statusTime = time.Now()
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

	case 1: // Headers tab — resize the value column to fill available width
		const hdrKeyWidth = 30
		hdrValWidth := m.width - 6 - hdrKeyWidth
		if hdrValWidth < 20 {
			hdrValWidth = 20
		}
		m.headersTable = m.headersTable.WithColumns([]table.Column{
			table.NewColumn(colHdrKey, "Key", hdrKeyWidth),
			table.NewColumn(colHdrVal, "Value", hdrValWidth),
		})

	case 2: // Metadata tab — resize the value column to fill available width
		const nameColWidth = 20
		valueColWidth := m.width - 6 - nameColWidth
		if valueColWidth < 20 {
			valueColWidth = 20
		}
		m.metadataTable = m.metadataTable.WithColumns([]table.Column{
			table.NewColumn(colMdName, "Name", nameColWidth),
			table.NewColumn(colMdValue, "Value", valueColWidth),
		})
	}
}

// HandleContentUpdate handles content updates
func (m *MessageDetailContentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd {
	if m.model == nil {
		return nil
	}

	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
			// Pass scroll events directly to the active editor so the user can
			// scroll through long content with the mouse wheel.
			return m.updateActiveEditor(msg)
		case tea.MouseButtonLeft:
			// Check if the user clicked on one of the tab labels.
			for i := range m.tabs {
				z := zone.Get(fmt.Sprintf("md-tab-%d", i))
				if z.InBounds(msg) {
					m.activeTab = i
					return nil
				}
			}
		}

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
			if m.activeTab == 0 {
				m.model.CopyContentWithFeedback()
			} else if m.activeTab == 1 {
				m.copyHeadersAsCSV()
			} else if m.activeTab == 2 {
				m.copyMetadataAsCSV()
			}
			return nil
		case "e":
			// Export the whole message to a JSON file (MSG-29).
			m.exportMessageToFile()
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
		m.headersTable, cmd = m.headersTable.Update(msg)
	case 2: // Metadata tab
		m.metadataTable, cmd = m.metadataTable.Update(msg)
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

func (m *MessageDetailContentProvider) IsInputMode() bool {
	return false
}

// GetContentSize returns the estimated content size for scrollbar calculation
func (m *MessageDetailContentProvider) GetContentSize(width int) int {
	// Estimate based on message content lines
	if m.model == nil {
		return 10
	}
	// Count lines in key and value content
	keyLines := len(strings.Split(m.model.GetFormattedKey(), "\n"))
	valueLines := len(strings.Split(m.model.GetFormattedValue(), "\n"))
	// Add metadata and header lines
	return keyLines + valueLines + 15
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
	highlightColor    = stylesPkg.Primary
	inactiveTabStyle  = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(highlightColor).Padding(0, 1)
	activeTabStyle    = inactiveTabStyle.Border(activeTabBorder, true)
	windowStyle       = lipgloss.NewStyle().BorderForeground(highlightColor).Padding(2, 1).Border(lipgloss.NormalBorder()).UnsetBorderTop()

	// Editor styles
	cursorStyle     = lipgloss.NewStyle().Foreground(stylesPkg.Warning)
	cursorLineStyle = lipgloss.NewStyle().
			Background(stylesPkg.Info).
			Foreground(stylesPkg.FgBase)
	placeholderStyle = lipgloss.NewStyle().
				Foreground(stylesPkg.FgSubtle)
	endOfBufferStyle = lipgloss.NewStyle().
				Foreground(stylesPkg.FgSubtle)
	focusedPlaceholderStyle = lipgloss.NewStyle().
				Foreground(stylesPkg.Primary)
	focusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(highlightColor)
	blurredBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(stylesPkg.FgSubtle)
)
