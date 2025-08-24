package detail

import (
	"fmt"

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
	theme      core.Theme
	styles     *ViewStyles
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
			Padding(0, 1).
			Bold(true),

		Footer: lipgloss.NewStyle().
			Background(lipgloss.Color(theme.Secondary)).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1),

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
	// Calculate layout dimensions
	sidebarWidth := 35
	contentWidth := model.dimensions.Width - sidebarWidth - 6 // Account for padding and borders
	contentHeight := model.dimensions.Height - 8              // Account for header and footer

	// Header section
	header := v.renderHeader(model)

	// Main content area with message details
	mainContent := v.renderMainContent(model, contentWidth, contentHeight)

	// Sidebar with metadata and actions
	// Adjust sidebar height to match the main content height
	sidebarContentHeight := contentHeight - 2 // Account for sidebar padding/borders
	sidebar := v.renderSidebar(model, sidebarWidth, sidebarContentHeight)

	// Combine main content and sidebar
	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		mainContent,
		lipgloss.NewStyle().Width(2).Render(""), // Spacer
		sidebar,
	)

	// Footer with key bindings
	footer := v.renderFooter(model)

	// Combine all sections
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		v.styles.Layout.Render(body),
		footer,
	)
}

// renderCompactView renders a compact layout for smaller screens
func (v *View) renderCompactView(model *Model) string {
	// Calculate layout dimensions for compact view
	contentWidth := model.dimensions.Width - 4  // Account for padding
	contentHeight := model.dimensions.Height - 6 // Account for header and footer

	// Header section
	header := v.renderHeader(model)

	// Message content sections (stacked vertically in compact mode)
	keySection := v.styles.MainPanel.
		Width(contentWidth).
		Height(contentHeight/3 - 2).
		Render(v.renderKeySection(model))

	valueSection := v.styles.MainPanel.
		Width(contentWidth).
		Height(contentHeight/3 - 2).
		Render(v.renderValueSection(model))

	// Headers section (if enabled)
	var headersSection string
	if model.showHeaders && len(model.message.Headers) > 0 {
		headersSection = v.styles.MainPanel.
			Width(contentWidth).
			Height(contentHeight/3 - 2).
			Render(v.renderHeadersSection(model))
	}

	// Schema info section (compact version)
	schemaInfo := v.renderCompactSchemaInfo(model)

	// Footer
	footer := v.renderFooter(model)

	// Combine all sections
	sections := []string{header, keySection, valueSection}
	if headersSection != "" {
		sections = append(sections, headersSection)
	}
	sections = append(sections, schemaInfo, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (v *View) renderHeader(model *Model) string {
	resourceIndicator := v.styles.ResourceType.Render("MESSAGE")
	headerText := fmt.Sprintf("%s Kafui - Message Detail: %s", resourceIndicator, model.topicName)

	return v.styles.Header.
		Width(model.dimensions.Width).
		Render(headerText)
}

func (v *View) renderMainContent(model *Model, width, height int) string {
	// Adjust height to account for panel borders and padding
	contentHeight := height - 4 // Account for main panel borders and padding

	// Message key section
	keySection := v.styles.MainPanel.
		Width(width/2 - 2).
		Height(contentHeight/2 - 1).
		Render(v.renderKeySection(model))

	// Message value section
	valueSection := v.styles.MainPanel.
		Width(width/2 - 2).
		Height(contentHeight/2 - 1).
		Render(v.renderValueSection(model))

	// Headers section (if enabled)
	var headersSection string
	if model.showHeaders && len(model.message.Headers) > 0 {
		headersSection = v.styles.MainPanel.
			Width(width).
			Height(contentHeight/4 - 1).
			Render(v.renderHeadersSection(model))
	}

	// Arrange sections
	topRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		keySection,
		lipgloss.NewStyle().Width(2).Render(""), // Spacer
		valueSection,
	)

	if headersSection != "" {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			topRow,
			lipgloss.NewStyle().Height(1).Render(""), // Spacer
			headersSection,
		)
	}

	return topRow
}

func (v *View) renderKeySection(model *Model) string {
	title := v.styles.Title.Render("MESSAGE KEY")
	content := model.GetFormattedKey()

	if content == "<null>" {
		content = v.styles.InfoPanel.Render("No key present")
	} else {
		content = v.styles.Key.Width(30).Render(content)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.NewStyle().MarginTop(1).Render(""),
		content,
	)
}

func (v *View) renderValueSection(model *Model) string {
	title := v.styles.Title.Render("MESSAGE VALUE")
	content := model.GetFormattedValue()

	if content == "<null>" {
		content = v.styles.InfoPanel.Render("No value present")
	} else {
		content = v.styles.Value.Width(30).Render(content)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.NewStyle().MarginTop(1).Render(""),
		content,
	)
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

	return v.styles.Footer.
		Width(model.dimensions.Width).
		Render(footerText)
}

// SetDimensions updates the view dimensions
func (v *View) SetDimensions(width, height int) {
	v.dimensions = core.Dimensions{Width: width, Height: height}
}