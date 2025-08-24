package topic

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/charmbracelet/lipgloss"
)

// View handles rendering for the topic page
type View struct {
	dimensions core.Dimensions
	theme      core.Theme
	styles     *ViewStyles
}

// ViewStyles contains all the styles used in the topic page
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
	Selected     lipgloss.Style
}

// NewView creates a new View instance
func NewView() *View {
	theme := core.DefaultTheme()
	return &View{
		theme:  theme,
		styles: createViewStyles(theme),
	}
}

// createViewStyles creates the styling configuration for the topic page
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

		Selected: lipgloss.NewStyle().
			Background(lipgloss.Color("205")).
			Foreground(lipgloss.Color("0")),
	}
}

// Render renders the topic page view
func (v *View) Render(model *Model) string {
	if model.dimensions.Width == 0 {
		return "Loading topic page..."
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

	// Main content area with controls, messages, and search
	controlsSection := v.styles.MainPanel.
		Width(contentWidth).
		Render(v.renderControls(model))

	messagesSection := v.styles.MainPanel.
		Width(contentWidth).
		Height(contentHeight - 6). // Account for controls and search
		Render(v.renderMessages(model))

	var searchSection string
	if model.searchMode {
		searchSection = v.styles.MainPanel.
			Width(contentWidth).
			Render(model.searchInput.View())
	}

	// Combine main content sections
	var mainContent string
	if model.searchMode {
		mainContent = lipgloss.JoinVertical(
			lipgloss.Left,
			controlsSection,
			messagesSection,
			searchSection,
		)
	} else {
		mainContent = lipgloss.JoinVertical(
			lipgloss.Left,
			controlsSection,
			messagesSection,
		)
	}

	// Sidebar with topic information
	sidebar := v.renderSidebar(model, sidebarWidth, contentHeight)

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

	// Controls section
	controlsSection := v.styles.MainPanel.
		Width(contentWidth).
		Render(v.renderControls(model))

	// Messages section (take most of the space)
	messagesHeight := contentHeight - 8 // Account for controls, search, and padding
	if model.searchMode {
		messagesHeight -= 3 // Additional space for search input
	}

	messagesSection := v.styles.MainPanel.
		Width(contentWidth).
		Height(messagesHeight).
		Render(v.renderMessages(model))

	// Search section (if in search mode)
	var searchSection string
	if model.searchMode {
		searchSection = v.styles.MainPanel.
			Width(contentWidth).
			Render(model.searchInput.View())
	}

	// Schema info section (compact version)
	schemaInfo := v.renderCompactSchemaInfo(model)

	// Footer
	footer := v.renderFooter(model)

	// Combine all sections
	sections := []string{header, controlsSection, messagesSection}
	if model.searchMode {
		sections = append(sections, searchSection)
	}
	sections = append(sections, schemaInfo, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (v *View) renderHeader(model *Model) string {
	resourceIndicator := v.styles.ResourceType.Render("TOPIC")
	headerText := fmt.Sprintf("%s Kafui - Topic: %s", resourceIndicator, model.topicName)

	return v.styles.Header.
		Width(model.dimensions.Width).
		Render(headerText)
}

func (v *View) renderSidebar(model *Model, width, height int) string {
	sidebarContent := lipgloss.JoinVertical(
		lipgloss.Left,
		v.styles.Title.Render("TOPIC INFO"),
		v.renderTopicInfo(model),
		lipgloss.NewStyle().MarginTop(2).Render(""),
		v.styles.Subtitle.Render("SELECTED MESSAGE"),
		v.renderSelectedMessageInfo(model),
		lipgloss.NewStyle().MarginTop(2).Render(""),
		v.styles.Subtitle.Render("SHORTCUTS"),
		v.renderShortcuts(model),
	)

	return v.styles.Sidebar.
		Width(width).
		Height(height).
		Render(sidebarContent)
}

func (v *View) renderCompactSchemaInfo(model *Model) string {
	// In compact mode, show a simplified schema info
	selectedMsg := model.GetSelectedMessage()
	if selectedMsg == nil {
		return ""
	}

	info := []string{v.styles.Title.Render("SELECTED MESSAGE INFO")}

	// Basic message information
	info = append(info, fmt.Sprintf("Partition: %d | Offset: %d", selectedMsg.Partition, selectedMsg.Offset))

	// Add schema information if available
	if model.selectedMessageSchema != nil {
		if model.selectedMessageSchema.KeySchema != nil {
			info = append(info, fmt.Sprintf("Key Schema: %s", model.selectedMessageSchema.KeySchema.RecordName))
		}
		if model.selectedMessageSchema.ValueSchema != nil {
			info = append(info, fmt.Sprintf("Value Schema: %s", model.selectedMessageSchema.ValueSchema.RecordName))
		}
	} else if selectedMsg.KeySchemaID != "" || selectedMsg.ValueSchemaID != "" {
		if selectedMsg.KeySchemaID != "" {
			info = append(info, fmt.Sprintf("Key Schema ID: %s", selectedMsg.KeySchemaID))
		}
		if selectedMsg.ValueSchemaID != "" {
			info = append(info, fmt.Sprintf("Value Schema ID: %s", selectedMsg.ValueSchemaID))
		}
	}

	return v.styles.MainPanel.Render(lipgloss.JoinVertical(lipgloss.Left, info...))
}

func (v *View) renderTopicInfo(model *Model) string {
	info := fmt.Sprintf(
		"Name: %s\nPartitions: %d\nReplication Factor: %d\nMessages: %d",
		model.topicName,
		model.topicDetails.NumPartitions,
		model.topicDetails.ReplicationFactor,
		len(model.messages),
	)

	// Format config entries if any
	if len(model.topicDetails.ConfigEntries) > 0 {
		configLines := []string{"\nConfiguration:"}
		count := 0
		for key, value := range model.topicDetails.ConfigEntries {
			if count >= 5 { // Limit to 5 config entries to avoid overcrowding
				configLines = append(configLines, "  ...")
				break
			}
			if value != nil {
				configLines = append(configLines, fmt.Sprintf("  %s: %s", key, *value))
			} else {
				configLines = append(configLines, fmt.Sprintf("  %s: <nil>", key))
			}
			count++
		}
		info += strings.Join(configLines, "\n")
	}

	return v.styles.InfoPanel.Render(info)
}

// renderSelectedMessageInfo renders information about the currently selected message including schema info
func (v *View) renderSelectedMessageInfo(model *Model) string {
	selectedMsg := model.GetSelectedMessage()
	if selectedMsg == nil {
		return v.styles.InfoPanel.Render("No message selected")
	}

	// Basic message information
	info := fmt.Sprintf(
		"Partition: %d\nOffset: %d",
		selectedMsg.Partition,
		selectedMsg.Offset,
	)

	// Add schema information if available
	if model.selectedMessageSchema != nil {
		schemaInfo := "\n\nSCHEMA INFO:"

		// Key schema information
		if model.selectedMessageSchema.KeySchema != nil {
			schemaInfo += fmt.Sprintf(
				"\nKey Schema: %s (ID: %d)",
				model.selectedMessageSchema.KeySchema.RecordName,
				model.selectedMessageSchema.KeySchema.ID,
			)
		} else if selectedMsg.KeySchemaID != "" {
			schemaInfo += fmt.Sprintf("\nKey Schema ID: %s (Not Avro)", selectedMsg.KeySchemaID)
		} else {
			schemaInfo += "\nKey Schema: Not available"
		}

		// Value schema information
		if model.selectedMessageSchema.ValueSchema != nil {
			schemaInfo += fmt.Sprintf(
				"\nValue Schema: %s (ID: %d)",
				model.selectedMessageSchema.ValueSchema.RecordName,
				model.selectedMessageSchema.ValueSchema.ID,
			)
		} else if selectedMsg.ValueSchemaID != "" {
			schemaInfo += fmt.Sprintf("\nValue Schema ID: %s (Not Avro)", selectedMsg.ValueSchemaID)
		} else {
			schemaInfo += "\nValue Schema: Not available"
		}

		info += schemaInfo
	} else if selectedMsg.KeySchemaID != "" || selectedMsg.ValueSchemaID != "" {
		// Show schema IDs even if schema info couldn't be loaded
		schemaInfo := "\n\nSCHEMA INFO:"
		if selectedMsg.KeySchemaID != "" {
			schemaInfo += fmt.Sprintf("\nKey Schema ID: %s", selectedMsg.KeySchemaID)
		}
		if selectedMsg.ValueSchemaID != "" {
			schemaInfo += fmt.Sprintf("\nValue Schema ID: %s", selectedMsg.ValueSchemaID)
		}
		info += schemaInfo
	}

	return v.styles.InfoPanel.Render(info)
}

func (v *View) renderControls(model *Model) string {
	controls := fmt.Sprintf(
		"Format: %s | Partition: All | Follow: %t | Paused: %t",
		"JSON", // Default format
		model.consumeFlags.Follow,
		model.paused,
	)

	return v.styles.InfoPanel.Render(controls)
}

func (v *View) renderMessages(model *Model) string {
	if model.loading {
		return fmt.Sprintf("%s Loading messages...", model.spinner.View())
	}

	if len(model.filteredMessages) == 0 {
		// Check if we're actively consuming but haven't received messages yet
		if model.consuming {
			return fmt.Sprintf("%s Waiting for messages...", model.spinner.View())
		}
		// Not consuming at all
		return "No messages available. Press 'r' to start consumption or check connection."
	}

	return model.messageTable.View()
}

func (v *View) renderShortcuts(model *Model) string {
	shortcuts := model.keys.GetShortcuts()
	return lipgloss.JoinVertical(lipgloss.Left, shortcuts...)
}

func (v *View) renderFooter(model *Model) string {
	// Left side: Selection and connection information
	selected := "None"
	if len(model.filteredMessages) > 0 {
		cursor := model.messageTable.Cursor()
		if cursor >= 0 && cursor < len(model.filteredMessages) {
			selected = fmt.Sprintf("Message %d/%d", cursor+1, len(model.filteredMessages))
		}
	}

	leftInfo := fmt.Sprintf("Selected: %s | Connection: %s", selected, model.connectionStatus)

	// Center: Status message
	status := fmt.Sprintf("%s %s", model.spinner.View(), model.statusMessage)
	if model.error != nil {
		status = fmt.Sprintf("Error: %v", model.error)
	}

	// Right side: Last update time
	rightInfo := fmt.Sprintf("Last update: %s", model.lastUpdate.Format("15:04:05"))

	// Calculate spacing
	totalContentWidth := len(leftInfo) + len(status) + len(rightInfo)
	if totalContentWidth < model.dimensions.Width {
		leftPadding := (model.dimensions.Width - totalContentWidth) / 3
		rightPadding := model.dimensions.Width - totalContentWidth - leftPadding

		footerContent := leftInfo +
			strings.Repeat(" ", leftPadding) +
			status +
			strings.Repeat(" ", rightPadding) +
			rightInfo

		return v.styles.Footer.Width(model.dimensions.Width).Render(footerContent)
	}

	// If content is too wide, just show status
	return v.styles.Footer.Width(model.dimensions.Width).Render(status)
}

// SetDimensions updates the view dimensions
func (v *View) SetDimensions(width, height int) {
	v.dimensions = core.Dimensions{Width: width, Height: height}
}

// SetTheme updates the view theme
func (v *View) SetTheme(theme core.Theme) {
	v.theme = theme
	v.styles = createViewStyles(theme)
}

// RenderError renders an error state
func (v *View) RenderError(model *Model, err error) string {
	if model.dimensions.Width == 0 {
		return "Error: " + err.Error()
	}

	errorContent := fmt.Sprintf("Error in topic page: %s\n\nPress 'r' to retry or 'Esc' to go back", err.Error())

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(v.theme.Error)).
		Width(model.dimensions.Width - 4).
		Align(lipgloss.Center).
		Padding(2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(v.theme.Error))

	content := errorStyle.Render(errorContent)

	return lipgloss.JoinVertical(
		lipgloss.Center,
		v.renderHeader(model),
		content,
		v.renderFooter(model),
	)
}

// RenderLoading renders a loading state
func (v *View) RenderLoading(model *Model) string {
	if model.dimensions.Width == 0 {
		return "Loading..."
	}

	loadingContent := fmt.Sprintf("%s Connecting to topic: %s\n\nPlease wait...", model.spinner.View(), model.topicName)

	loadingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(v.theme.Info)).
		Width(model.dimensions.Width - 4).
		Align(lipgloss.Center).
		Padding(2)

	content := loadingStyle.Render(loadingContent)

	return lipgloss.JoinVertical(
		lipgloss.Center,
		v.renderHeader(model),
		content,
		v.renderFooter(model),
	)
}

// RenderHelp renders help information
func (v *View) RenderHelp(model *Model) string {
	helpContent := []string{
		"Topic Page Help",
		"",
		"Message Navigation:",
		"  j/↓        Move down",
		"  k/↑        Move up",
		"  g/Home     Go to top",
		"  G/End      Go to bottom",
		"  Enter      View message details",
		"",
		"Message Operations:",
		"  c          Copy message key",
		"  v          Copy message value",
		"  d          Show message details",
		"",
		"Consumption Control:",
		"  Space      Pause/resume consumption",
		"  r          Retry connection",
		"",
		"Search:",
		"  /          Search messages",
		"  Esc        Exit search",
		"",
		"General:",
		"  q/Esc      Back to topics",
		"  Ctrl+C     Quit application",
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(v.theme.Info)).
		Width(model.dimensions.Width - 4).
		Padding(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(v.theme.Primary))

	content := helpStyle.Render(strings.Join(helpContent, "\n"))

	return content
}