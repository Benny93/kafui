package messagedetail

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/ui/components"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		// Update viewport content when format changes
		// This is now handled in the view component
		return nil
	case key.Matches(msg, k.bindings.ToggleHeaders):
		model.ToggleHeaders()
		return nil
	case key.Matches(msg, k.bindings.ToggleMetadata):
		model.ToggleMetadata()
		return nil
	case key.Matches(msg, k.bindings.Copy):
		// Implement copy functionality with feedback
		model.CopyContentWithFeedback()
		return nil
	}
	return nil
}

// GetKeyBindings returns all key bindings as a slice
func (k *Keys) GetKeyBindings() []key.Binding {
	return []key.Binding{
		k.bindings.Back,
		k.bindings.Quit,
		k.bindings.ToggleFormat,
		k.bindings.ToggleHeaders,
		k.bindings.ToggleMetadata,
		k.bindings.Copy,
	}
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
		// Handle viewport navigation keys first
		switch msg.String() {
		case "up":
			// Scroll the focused viewport up
			model.view.LineUp(1, model)
			return model, nil
		case "down":
			// Scroll the focused viewport down
			model.view.LineDown(1, model)
			return model, nil
		case "tab":
			model.SwitchFocus()
			return model, nil
		}

		cmd := model.keys.HandleKey(model, msg)
		return model, cmd
	}

	return model, nil
}

// View handles rendering for the detail page
type View struct {
	dimensions  core.Dimensions
	theme       core.Theme
	styles      *ViewStyles
	keyViewer   *components.JSONContentView
	valueViewer *components.JSONContentView
}

// ViewStyles contains all the styles used in the detail page
type ViewStyles struct {
	Header       lipgloss.Style
	Footer       lipgloss.Style
	Sidebar      lipgloss.Style
	MainPanel    lipgloss.Style
	InfoPanel    lipgloss.Style
	Title        lipgloss.Style
	Subtitle     lipgloss.Style
	ResourceType lipgloss.Style
	Layout       lipgloss.Style
	Content      lipgloss.Style
	Value        lipgloss.Style
	Key          lipgloss.Style
	Metadata     lipgloss.Style
}

// NewView creates a new View instance
func NewView() *View {
	theme := core.DefaultTheme()
	return &View{
		theme:  theme,
		styles: createViewStyles(theme),
	}
}

// createViewStyles creates the styling configuration for the detail page
func createViewStyles(theme core.Theme) *ViewStyles {
	return &ViewStyles{
		Header: lipgloss.NewStyle().
			Background(lipgloss.Color(theme.Primary)).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Width(100).
			Align(lipgloss.Left).
			Padding(0, 1),

		Footer: lipgloss.NewStyle().
			Background(lipgloss.Color(theme.Secondary)).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1).
			Width(100),

		Sidebar: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Primary)).
			Padding(1),

		MainPanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Secondary)).
			Padding(1),

		InfoPanel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Info)),

		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Primary)).
			Bold(true),

		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Secondary)).
			Bold(true),

		ResourceType: lipgloss.NewStyle().
			Background(lipgloss.Color(theme.Accent)).
			Foreground(lipgloss.Color("#000000")).
			Padding(0, 1).
			Bold(true),

		Layout: lipgloss.NewStyle().
			Padding(1),

		Content: lipgloss.NewStyle().
			Padding(1).
			Margin(0, 1),

		Value: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Success)).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Secondary)).
			Padding(1),

		Key: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Warning)).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Secondary)).
			Padding(1),

		Metadata: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Info)).
			Italic(true),
	}
}

// Render renders the detail page view
func (v *View) Render(model *Model) string {
	if model.dimensions.Width == 0 {
		return "Loading message details..."
	}

	// Determine if we're in compact mode
	compactMode := model.dimensions.Width < 100 || model.dimensions.Height < 25

	if compactMode {
		return v.renderCompactView(model)
	}
	return v.renderFullView(model)
}

// renderFullView renders the full layout for larger screens
func (v *View) renderFullView(model *Model) string {
	// Calculate base dimensions
	sidebarWidth := 35
	contentWidth := model.dimensions.Width - 4 // Account for outer padding

	// Calculate section heights
	headerHeight := 1 // Header is single line
	footerHeight := 1 // Footer is single line

	// Remaining height for body content
	bodyHeight := model.dimensions.Height - headerHeight - footerHeight - 2 // -2 for spacing
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	// Header section
	header := v.renderHeader(model)

	// Calculate content width accounting for sidebar
	mainContentWidth := contentWidth - sidebarWidth - 1 // -1 for spacing between main and sidebar

	// Main content area with message details (subtract header and footer from height)
	mainContent := v.renderMainContent(model, mainContentWidth, bodyHeight)

	// Sidebar with metadata and actions
	sidebar := v.renderSidebar(model, sidebarWidth, bodyHeight)

	// Combine main content and sidebar with proper spacing
	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		mainContent,
		lipgloss.NewStyle().Width(1).Render(""), // Spacer
		sidebar,
	)

	// Footer with key bindings
	footer := v.renderFooter(model)

	// Combine all sections with proper spacing
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		lipgloss.NewStyle().Height(1).Render(""), // Spacing after header
		body,
		lipgloss.NewStyle().Height(1).Render(""), // Spacing before footer
		footer,
	)
}

// renderCompactView renders a compact layout for smaller screens
func (v *View) renderCompactView(model *Model) string {
	// Calculate layout dimensions for compact view
	contentWidth := model.dimensions.Width - 4   // Account for padding
	contentHeight := model.dimensions.Height - 6 // Account for header and footer
	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Header section
	header := v.renderHeader(model)

	// Calculate section dimensions with better space distribution
	// Key gets smaller fixed portion, value gets remaining space
	keySectionHeight := 6                                      // Fixed smaller height for key section
	valueSectionHeight := contentHeight - keySectionHeight - 6 // Remaining space with spacing

	// Ensure minimum heights but don't exceed available space
	if keySectionHeight > contentHeight/4 {
		keySectionHeight = contentHeight / 4
	}
	if keySectionHeight < 1 {
		keySectionHeight = 1
	}

	if valueSectionHeight < 1 {
		valueSectionHeight = 1
	}

	// Both sections get full width
	sectionWidth := contentWidth

	// Update key viewer config
	keyContent := "<null>"
	if model.message.Key != "" {
		keyContent = model.message.Key
	}

	keyConfig := components.JSONContentConfig{
		Width:           sectionWidth,
		Height:          keySectionHeight,
		Title:           "MESSAGE KEY",
		Content:         keyContent,
		DisplayFormat:   model.displayFormat.KeyFormat,
		ShowLineNumbers: true,
		Focused:         model.focusedViewport == "key",
	}

	if v.keyViewer == nil {
		v.keyViewer = components.NewJSONContentView(keyConfig)
	} else {
		v.keyViewer.UpdateConfig(keyConfig)
	}

	// Update value viewer config
	valueContent := "<null>"
	if model.message.Value != "" {
		valueContent = model.message.Value
	}

	valueConfig := components.JSONContentConfig{
		Width:           sectionWidth,
		Height:          valueSectionHeight,
		Title:           "MESSAGE VALUE",
		Content:         valueContent,
		DisplayFormat:   model.displayFormat.ValueFormat,
		ShowLineNumbers: true,
		Focused:         model.focusedViewport == "value",
	}

	if v.valueViewer == nil {
		v.valueViewer = components.NewJSONContentView(valueConfig)
	} else {
		v.valueViewer.UpdateConfig(valueConfig)
	}

	// Render sections
	keySection := v.styles.MainPanel.
		Width(sectionWidth).
		Height(keySectionHeight).
		Render(v.keyViewer.View())

	valueSection := v.styles.MainPanel.
		Width(sectionWidth).
		Height(valueSectionHeight).
		Render(v.valueViewer.View())

	// Headers section (if enabled)
	var headersSection string
	if model.showHeaders && len(model.message.Headers) > 0 {
		headersHeight := 5
		if headersHeight > contentHeight/5 {
			headersHeight = contentHeight / 5
		}

		headersSection = v.styles.MainPanel.
			Width(sectionWidth).
			Height(headersHeight).
			Render(v.renderHeadersSection(model))
	}

	// Schema info section (compact version)
	schemaInfo := v.renderCompactSchemaInfo(model)

	// Footer
	footer := v.renderFooter(model)

	// Combine all sections vertically (key above value)
	var sections []string
	sections = append(sections, header)
	sections = append(sections, keySection)
	sections = append(sections, lipgloss.NewStyle().Height(1).Render("")) // Spacer
	sections = append(sections, valueSection)

	if headersSection != "" {
		sections = append(sections, lipgloss.NewStyle().Height(1).Render("")) // Spacer
		sections = append(sections, headersSection)
	}

	sections = append(sections, schemaInfo)
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (v *View) renderHeader(model *Model) string {
	resourceIndicator := v.styles.ResourceType.Render("MESSAGE")
	headerText := fmt.Sprintf("%s Kafui - Message Detail: %s", resourceIndicator, model.topicName)

	// Create a style for the header that takes up full width and stands out
	headerStyle := v.styles.Header.
		Width(model.dimensions.Width).
		Bold(true).
		Background(lipgloss.Color("62")). // Use a distinctive color
		Foreground(lipgloss.Color("#FFFFFF"))

	return headerStyle.Render(headerText)
}

func (v *View) renderMainContent(model *Model, width, height int) string {
	// Account for layout margins and spacing
	effectiveWidth := width - 4   // Account for left/right margins (2 each side)
	effectiveHeight := height - 2 // Account for vertical spacing

	// Ensure minimum dimensions
	if effectiveWidth < 10 {
		effectiveWidth = 10
	}
	if effectiveHeight < 5 {
		effectiveHeight = 5
	}

	// Calculate content widths accounting for borders
	contentWidth := effectiveWidth - 2 // Account for content borders

	// Start with available height and subtract as needed
	availableHeight := effectiveHeight

	// Calculate space for headers if enabled
	headersHeight := 0
	if model.showHeaders && len(model.message.Headers) > 0 {
		headersHeight = 6 // Fixed height for headers
		if headersHeight > availableHeight/5 {
			headersHeight = availableHeight / 5 // Max 20% of height
		}
		availableHeight -= (headersHeight + 1) // Account for headers and spacing
	}

	// Calculate key and value section heights
	keySectionHeight := 0
	valueSectionHeight := availableHeight

	// If we have enough space, show both sections
	if availableHeight >= 15 { // Minimum space needed for both sections
		keySectionHeight = int(float64(availableHeight) * 0.3)      // 30% for key
		valueSectionHeight = availableHeight - keySectionHeight - 1 // Rest for value, -1 for spacing
	}

	// Ensure minimum heights but don't exceed available space
	if keySectionHeight < 1 {
		keySectionHeight = 1
	}

	if valueSectionHeight < 1 {
		valueSectionHeight = 1
	}

	// Update key viewer config with adjusted dimensions
	keyContent := "<null>"
	if model.message.Key != "" {
		keyContent = model.message.Key
	}

	// Only configure key viewer if we have height for it
	if keySectionHeight > 0 {
		keyConfig := components.JSONContentConfig{
			Width:           contentWidth,
			Height:          keySectionHeight,
			Title:           "MESSAGE KEY",
			Content:         keyContent,
			DisplayFormat:   model.displayFormat.KeyFormat,
			ShowLineNumbers: true,
			Focused:         model.focusedViewport == "key",
		}

		if v.keyViewer == nil {
			v.keyViewer = components.NewJSONContentView(keyConfig)
		} else {
			v.keyViewer.UpdateConfig(keyConfig)
		}
	}

	// Update value viewer config
	valueContent := "<null>"
	if model.message.Value != "" {
		valueContent = model.message.Value
	}

	valueConfig := components.JSONContentConfig{
		Width:           contentWidth,
		Height:          valueSectionHeight,
		Title:           "MESSAGE VALUE",
		Content:         valueContent,
		DisplayFormat:   model.displayFormat.ValueFormat,
		ShowLineNumbers: true,
		Focused:         model.focusedViewport == "value",
	}

	if v.valueViewer == nil {
		v.valueViewer = components.NewJSONContentView(valueConfig)
	} else {
		v.valueViewer.UpdateConfig(valueConfig)
	}

	// Start building the content sections
	var sections []string

	// Configure and render key viewer if we have space for it
	if keySectionHeight > 0 {
		if v.keyViewer == nil {
			v.keyViewer = components.NewJSONContentView(components.JSONContentConfig{
				Width:           contentWidth,
				Height:          keySectionHeight,
				Title:           "MESSAGE KEY",
				Content:         model.message.Key,
				DisplayFormat:   model.displayFormat.KeyFormat,
				ShowLineNumbers: true,
				Focused:         model.focusedViewport == "key",
			})
		} else {
			v.keyViewer.UpdateConfig(components.JSONContentConfig{
				Width:           contentWidth,
				Height:          keySectionHeight,
				Title:           "MESSAGE KEY",
				Content:         model.message.Key,
				DisplayFormat:   model.displayFormat.KeyFormat,
				ShowLineNumbers: true,
				Focused:         model.focusedViewport == "key",
			})
		}

		sections = append(sections, v.keyViewer.View())
		sections = append(sections, lipgloss.NewStyle().Height(1).Render("")) // Spacing
	}

	// Configure and render value viewer
	if v.valueViewer == nil {
		v.valueViewer = components.NewJSONContentView(components.JSONContentConfig{
			Width:           contentWidth,
			Height:          valueSectionHeight,
			Title:           "MESSAGE VALUE",
			Content:         model.message.Value,
			DisplayFormat:   model.displayFormat.ValueFormat,
			ShowLineNumbers: true,
			Focused:         model.focusedViewport == "value",
		})
	} else {
		v.valueViewer.UpdateConfig(components.JSONContentConfig{
			Width:           contentWidth,
			Height:          valueSectionHeight,
			Title:           "MESSAGE VALUE",
			Content:         model.message.Value,
			DisplayFormat:   model.displayFormat.ValueFormat,
			ShowLineNumbers: true,
			Focused:         model.focusedViewport == "value",
		})
	}

	sections = append(sections, v.valueViewer.View())

	// Add headers section if enabled
	if model.showHeaders && len(model.message.Headers) > 0 {
		sections = append(sections, lipgloss.NewStyle().Height(1).Render("")) // Spacing
		headerSection := v.styles.MainPanel.
			Height(headersHeight).
			Render(v.renderHeadersSection(model))
		sections = append(sections, headerSection)
	}

	// Render the final content with proper styling
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return v.styles.MainPanel.
		Width(effectiveWidth).
		Height(effectiveHeight).
		Render(content)
}

func (v *View) renderHeadersSection(model *Model) string {
	title := v.styles.Title.Render("MESSAGE HEADERS")

	if len(model.message.Headers) == 0 {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			v.styles.InfoPanel.Render("No headers present"),
		)
	}

	headersList := []string{}
	for _, header := range model.message.Headers {
		headerStr := fmt.Sprintf("%s: %s", header.Key, header.Value)
		headersList = append(headersList, headerStr)
	}

	headersContent := lipgloss.JoinVertical(lipgloss.Left, headersList...)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.NewStyle().MarginTop(1).Render(""),
		v.styles.Content.Render(headersContent),
	)
}

func (v *View) renderCompactSchemaInfo(model *Model) string {
	// In compact mode, show a simplified schema info
	schemaInfo := model.GetSchemaInfo() // Use lazy loading getter
	if schemaInfo == nil {
		if model.message.KeySchemaID != "" || model.message.ValueSchemaID != "" {
			// Show schema IDs even if schema info couldn't be loaded
			schemaInfoLines := []string{}
			if model.message.KeySchemaID != "" {
				schemaInfoLines = append(schemaInfoLines, fmt.Sprintf("Key Schema ID: %s", model.message.KeySchemaID))
			}
			if model.message.ValueSchemaID != "" {
				schemaInfoLines = append(schemaInfoLines, fmt.Sprintf("Value Schema ID: %s", model.message.ValueSchemaID))
			}
			return v.styles.MainPanel.Render(lipgloss.JoinVertical(lipgloss.Left, schemaInfoLines...))
		}
		return v.styles.MainPanel.Render("No schema information")
	}

	schemaDetails := []string{}

	// Key schema information
	if schemaInfo.KeySchema != nil {
		schemaDetails = append(schemaDetails,
			fmt.Sprintf("Key Schema: %s (ID: %d)",
				schemaInfo.KeySchema.RecordName,
				schemaInfo.KeySchema.ID))
	} else if model.message.KeySchemaID != "" {
		schemaDetails = append(schemaDetails, fmt.Sprintf("Key Schema ID: %s (Not Avro)", model.message.KeySchemaID))
	} else {
		schemaDetails = append(schemaDetails, "Key Schema: Not available")
	}

	// Value schema information
	if schemaInfo.ValueSchema != nil {
		schemaDetails = append(schemaDetails,
			fmt.Sprintf("Value Schema: %s (ID: %d)",
				schemaInfo.ValueSchema.RecordName,
				schemaInfo.ValueSchema.ID))
	} else if model.message.ValueSchemaID != "" {
		schemaDetails = append(schemaDetails, fmt.Sprintf("Value Schema ID: %s (Not Avro)", model.message.ValueSchemaID))
	} else {
		schemaDetails = append(schemaDetails, "Value Schema: Not available")
	}

	return v.styles.MainPanel.Render(lipgloss.JoinVertical(lipgloss.Left, schemaDetails...))
}

func (v *View) renderSidebar(model *Model, width, height int) string {
	sidebarContent := lipgloss.JoinVertical(
		lipgloss.Left,
		v.styles.Title.Render("MESSAGE INFO"),
		v.renderMessageInfo(model),
		lipgloss.NewStyle().MarginTop(2).Render(""),
		v.styles.Subtitle.Render("SCHEMA INFO"),
		v.renderSchemaInfo(model),
		lipgloss.NewStyle().MarginTop(2).Render(""),
		v.styles.Subtitle.Render("DISPLAY OPTIONS"),
		v.renderDisplayOptions(model),
		lipgloss.NewStyle().MarginTop(2).Render(""),
		v.styles.Subtitle.Render("SHORTCUTS"),
		v.renderShortcuts(model),
	)

	// Ensure consistent padding and height
	return v.styles.Sidebar.
		Width(width).
		Height(height).
		Render(sidebarContent)
}

func (v *View) renderSchemaInfo(model *Model) string {
	schemaInfo := model.GetSchemaInfo() // Use lazy loading getter
	if schemaInfo == nil {
		if model.message.KeySchemaID != "" || model.message.ValueSchemaID != "" {
			// Show schema IDs even if schema info couldn't be loaded
			schemaInfo := []string{}
			if model.message.KeySchemaID != "" {
				schemaInfo = append(schemaInfo, fmt.Sprintf("Key Schema ID: %s", model.message.KeySchemaID))
			}
			if model.message.ValueSchemaID != "" {
				schemaInfo = append(schemaInfo, fmt.Sprintf("Value Schema ID: %s", model.message.ValueSchemaID))
			}
			return v.styles.InfoPanel.Render(lipgloss.JoinVertical(lipgloss.Left, schemaInfo...))
		}
		return v.styles.InfoPanel.Render("No schema information")
	}

	schemaDetails := []string{}

	// Key schema information
	if schemaInfo.KeySchema != nil {
		schemaDetails = append(schemaDetails,
			fmt.Sprintf("Key Schema: %s (ID: %d)",
				schemaInfo.KeySchema.RecordName,
				schemaInfo.KeySchema.ID))
	} else if model.message.KeySchemaID != "" {
		schemaDetails = append(schemaDetails, fmt.Sprintf("Key Schema ID: %s (Not Avro)", model.message.KeySchemaID))
	} else {
		schemaDetails = append(schemaDetails, "Key Schema: Not available")
	}

	// Value schema information
	if schemaInfo.ValueSchema != nil {
		schemaDetails = append(schemaDetails,
			fmt.Sprintf("Value Schema: %s (ID: %d)",
				schemaInfo.ValueSchema.RecordName,
				schemaInfo.ValueSchema.ID))
	} else if model.message.ValueSchemaID != "" {
		schemaDetails = append(schemaDetails, fmt.Sprintf("Value Schema ID: %s (Not Avro)", model.message.ValueSchemaID))
	} else {
		schemaDetails = append(schemaDetails, "Value Schema: Not available")
	}

	return v.styles.InfoPanel.Render(lipgloss.JoinVertical(lipgloss.Left, schemaDetails...))
}

func (v *View) renderMessageInfo(model *Model) string {
	info := model.GetMessageInfo()
	infoLines := []string{}

	// Display key information first
	if val, exists := info["Topic"]; exists {
		infoLines = append(infoLines, fmt.Sprintf("Topic: %s", val))
	}
	if val, exists := info["Partition"]; exists {
		infoLines = append(infoLines, fmt.Sprintf("Partition: %s", val))
	}
	if val, exists := info["Offset"]; exists {
		infoLines = append(infoLines, fmt.Sprintf("Offset: %s", val))
	}
	if val, exists := info["Key Size"]; exists {
		infoLines = append(infoLines, fmt.Sprintf("Key Size: %s", val))
	}
	if val, exists := info["Value Size"]; exists {
		infoLines = append(infoLines, fmt.Sprintf("Value Size: %s", val))
	}
	if val, exists := info["Headers"]; exists {
		infoLines = append(infoLines, fmt.Sprintf("Headers: %s", val))
	}

	// Add schema information if available
	if val, exists := info["Key Schema ID"]; exists {
		infoLines = append(infoLines, fmt.Sprintf("Key Schema ID: %s", val))
	}
	if val, exists := info["Value Schema ID"]; exists {
		infoLines = append(infoLines, fmt.Sprintf("Value Schema ID: %s", val))
	}

	return v.styles.InfoPanel.Render(lipgloss.JoinVertical(lipgloss.Left, infoLines...))
}

func (v *View) renderDisplayOptions(model *Model) string {
	options := []string{
		fmt.Sprintf("Format: %s", model.displayFormat.ValueFormat),
		fmt.Sprintf("Headers: %v", model.showHeaders),
		fmt.Sprintf("Metadata: %v", model.showMetadata),
		fmt.Sprintf("Wrap Lines: %v", model.displayFormat.WrapLines),
	}

	return v.styles.InfoPanel.Render(lipgloss.JoinVertical(lipgloss.Left, options...))
}

func (v *View) renderShortcuts(model *Model) string {
	shortcuts := []string{
		"f     Toggle format",
		"h     Toggle headers",
		"m     Toggle metadata",
		"c     Copy content",
		"Esc   Back to topic",
		"q     Quit",
	}

	return v.styles.InfoPanel.Render(lipgloss.JoinVertical(lipgloss.Left, shortcuts...))
}

func (v *View) renderFooter(model *Model) string {
	footerText := "Detail View | Use 'f' to toggle format, 'h' for headers, 'm' for metadata | Press 'Esc' to go back"

	// Add status message if present
	if model.statusMsg != "" {
		footerText += " | " + model.statusMsg
	}

	return v.styles.Footer.
		Width(model.dimensions.Width).
		Render(footerText)
}

// SetDimensions updates the view dimensions
func (v *View) SetDimensions(width, height int) {
	v.dimensions = core.Dimensions{Width: width, Height: height}
}

// LineUp scrolls the focused viewport up
func (v *View) LineUp(lines int, model *Model) {
	if model.focusedViewport == "key" && v.keyViewer != nil {
		v.keyViewer.LineUp(lines)
	} else if model.focusedViewport == "value" && v.valueViewer != nil {
		v.valueViewer.LineUp(lines)
	}
}

// LineDown scrolls the focused viewport down
func (v *View) LineDown(lines int, model *Model) {
	if model.focusedViewport == "key" && v.keyViewer != nil {
		v.keyViewer.LineDown(lines)
	} else if model.focusedViewport == "value" && v.valueViewer != nil {
		v.valueViewer.LineDown(lines)
	}
}
