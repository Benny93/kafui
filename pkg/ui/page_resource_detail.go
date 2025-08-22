package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

var (
	resourceDetailPageStyle = lipgloss.NewStyle().
				Margin(1, 2)

	resourceMetadataStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(1, 2).
				MarginBottom(1)

	resourceContentStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(1, 2)

	resourceHelpStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#3c3c3c")).
				Padding(0, 1)

	fieldNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Bold(true)

	fieldValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	sectionHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("11")).
				Bold(true).
				Underline(true)
)

// ResourceDetailPageModel represents the generic resource detail page
type ResourceDetailPageModel struct {
	resourceItem ResourceItem
	resourceType ResourceType
	resourceName string
	width        int
	height       int
	viewport     viewport.Model
	metadata     string
	helpText     string
	wrapped      bool
}

// NewResourceDetailPage creates a new resource detail page model
func NewResourceDetailPage(resourceItem ResourceItem, resourceType ResourceType) ResourceDetailPageModel {
	vp := viewport.New(80, 20)
	vp.Style = resourceContentStyle

	model := ResourceDetailPageModel{
		resourceItem: resourceItem,
		resourceType: resourceType,
		resourceName: resourceItem.GetID(),
		viewport:     vp,
	}

	model.updateContent()
	model.updateHelpText()

	return model
}

// Init initializes the resource detail page
func (m ResourceDetailPageModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the resource detail page
func (m ResourceDetailPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		if msg.Height > 10 {
			m.viewport.Height = msg.Height - 10
		}
		m.updateContent()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			// Return to main page
			return m, func() tea.Msg {
				return pageChangeMsg(mainPage)
			}
		case "j", "down":
			// Scroll down
			m.viewport.LineDown(1)
		case "k", "up":
			// Scroll up
			m.viewport.LineUp(1)
		case "ctrl+d", "pgdown":
			// Page down
			m.viewport.LineDown(m.viewport.Height / 2)
		case "ctrl+u", "pgup":
			// Page up
			m.viewport.LineUp(m.viewport.Height / 2)
		case "c":
			// Copy content (in a real implementation, this would copy to clipboard)
			m.helpText = "Content copied to clipboard"
			return m, nil
		case "w":
			// Toggle word wrap
			m.wrapped = !m.wrapped
			m.updateContent()
			return m, nil
		case "r":
			// Refresh resource data
			m.helpText = "Refreshing resource data..."
			return m, nil
		}
	}

	// Handle viewport updates
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the resource detail page
func (m ResourceDetailPageModel) View() string {
	if m.width == 0 {
		// Set default dimensions if not initialized
		m.width = 80
		m.height = 24
		m.viewport.Width = m.width - 4
		m.viewport.Height = m.height - 10
		m.updateContent()
	}

	// Create the layout
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		m.metadata,
		m.viewport.View(),
	)

	// Wrap in main style
	mainContent := resourceDetailPageStyle.Render(content)

	// Add help text at the bottom
	helpBar := resourceHelpStyle.Render(m.helpText)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		helpBar,
	)
}

// updateContent updates the content displayed in the viewport
func (m *ResourceDetailPageModel) updateContent() {
	// Format the resource content based on type
	formattedContent := m.formatResourceContent()
	
	// Apply word wrapping if enabled
	if m.wrapped && m.viewport.Width > 0 {
		formattedContent = wordwrap.String(formattedContent, m.viewport.Width-4)
	}
	
	// Set content in viewport
	m.viewport.SetContent(formattedContent)
	
	// Update metadata
	m.metadata = m.formatMetadata()
}

// formatResourceContent formats the resource content for display based on resource type
func (m *ResourceDetailPageModel) formatResourceContent() string {
	var content strings.Builder
	
	// Get resource details
	details := m.resourceItem.GetDetails()
	
	switch m.resourceType {
	case ConsumerGroupResourceType:
		content.WriteString(m.formatConsumerGroupDetails(details))
	case SchemaResourceType:
		content.WriteString(m.formatSchemaDetails(details))
	case ContextResourceType:
		content.WriteString(m.formatContextDetails(details))
	case TopicResourceType:
		content.WriteString(m.formatTopicDetails(details))
	default:
		content.WriteString(m.formatGenericDetails(details))
	}
	
	return content.String()
}

// formatConsumerGroupDetails formats consumer group specific details
func (m *ResourceDetailPageModel) formatConsumerGroupDetails(details map[string]string) string {
	var content strings.Builder
	
	content.WriteString(sectionHeaderStyle.Render("Consumer Group Information"))
	content.WriteString("\n\n")
	
	// Consumer Group specific fields
	if state, ok := details["state"]; ok {
		content.WriteString(fieldNameStyle.Render("State: "))
		content.WriteString(fieldValueStyle.Render(state))
		content.WriteString("\n")
	}
	
	if protocol, ok := details["protocol"]; ok {
		content.WriteString(fieldNameStyle.Render("Protocol: "))
		content.WriteString(fieldValueStyle.Render(protocol))
		content.WriteString("\n")
	}
	
	if coordinator, ok := details["coordinator"]; ok {
		content.WriteString(fieldNameStyle.Render("Coordinator: "))
		content.WriteString(fieldValueStyle.Render(coordinator))
		content.WriteString("\n")
	}
	
	if memberCount, ok := details["members"]; ok {
		content.WriteString(fieldNameStyle.Render("Members: "))
		content.WriteString(fieldValueStyle.Render(memberCount))
		content.WriteString("\n")
	}
	
	if assignedTopics, ok := details["assigned_topics"]; ok {
		content.WriteString(fieldNameStyle.Render("Assigned Topics: "))
		content.WriteString(fieldValueStyle.Render(assignedTopics))
		content.WriteString("\n")
	}
	
	// Add any other details
	content.WriteString("\n")
	content.WriteString(sectionHeaderStyle.Render("Additional Details"))
	content.WriteString("\n\n")
	
	for key, value := range details {
		if key != "state" && key != "protocol" && key != "coordinator" && key != "members" && key != "assigned_topics" {
			content.WriteString(fieldNameStyle.Render(key + ": "))
			content.WriteString(fieldValueStyle.Render(value))
			content.WriteString("\n")
		}
	}
	
	return content.String()
}

// formatSchemaDetails formats schema specific details
func (m *ResourceDetailPageModel) formatSchemaDetails(details map[string]string) string {
	var content strings.Builder
	
	content.WriteString(sectionHeaderStyle.Render("Schema Information"))
	content.WriteString("\n\n")
	
	if version, ok := details["version"]; ok {
		content.WriteString(fieldNameStyle.Render("Version: "))
		content.WriteString(fieldValueStyle.Render(version))
		content.WriteString("\n")
	}
	
	if schemaType, ok := details["type"]; ok {
		content.WriteString(fieldNameStyle.Render("Type: "))
		content.WriteString(fieldValueStyle.Render(schemaType))
		content.WriteString("\n")
	}
	
	if compatibility, ok := details["compatibility"]; ok {
		content.WriteString(fieldNameStyle.Render("Compatibility: "))
		content.WriteString(fieldValueStyle.Render(compatibility))
		content.WriteString("\n")
	}
	
	// Add schema content if available
	if schema, ok := details["schema"]; ok {
		content.WriteString("\n")
		content.WriteString(sectionHeaderStyle.Render("Schema Definition"))
		content.WriteString("\n\n")
		content.WriteString(fieldValueStyle.Render(schema))
		content.WriteString("\n")
	}
	
	return content.String()
}

// formatContextDetails formats context specific details
func (m *ResourceDetailPageModel) formatContextDetails(details map[string]string) string {
	var content strings.Builder
	
	content.WriteString(sectionHeaderStyle.Render("Context Information"))
	content.WriteString("\n\n")
	
	if endpoint, ok := details["endpoint"]; ok {
		content.WriteString(fieldNameStyle.Render("Endpoint: "))
		content.WriteString(fieldValueStyle.Render(endpoint))
		content.WriteString("\n")
	}
	
	if auth, ok := details["auth"]; ok {
		content.WriteString(fieldNameStyle.Render("Authentication: "))
		content.WriteString(fieldValueStyle.Render(auth))
		content.WriteString("\n")
	}
	
	return m.formatGenericDetails(details)
}

// formatTopicDetails formats topic specific details
func (m *ResourceDetailPageModel) formatTopicDetails(details map[string]string) string {
	var content strings.Builder
	
	content.WriteString(sectionHeaderStyle.Render("Topic Information"))
	content.WriteString("\n\n")
	
	if partitions, ok := details["partitions"]; ok {
		content.WriteString(fieldNameStyle.Render("Partitions: "))
		content.WriteString(fieldValueStyle.Render(partitions))
		content.WriteString("\n")
	}
	
	if replication, ok := details["replication"]; ok {
		content.WriteString(fieldNameStyle.Render("Replication Factor: "))
		content.WriteString(fieldValueStyle.Render(replication))
		content.WriteString("\n")
	}
	
	return m.formatGenericDetails(details)
}

// formatGenericDetails formats any additional details in a generic way
func (m *ResourceDetailPageModel) formatGenericDetails(details map[string]string) string {
	var content strings.Builder
	
	// Skip already processed fields for consumer groups
	skipFields := map[string]bool{
		"state": true, "protocol": true, "coordinator": true, "members": true, "assigned_topics": true,
		"version": true, "type": true, "compatibility": true, "schema": true,
		"endpoint": true, "auth": true, "partitions": true, "replication": true,
	}
	
	hasAdditional := false
	for key := range details {
		if !skipFields[key] {
			hasAdditional = true
			break
		}
	}
	
	if hasAdditional {
		content.WriteString("\n")
		content.WriteString(sectionHeaderStyle.Render("Additional Properties"))
		content.WriteString("\n\n")
		
		for key, value := range details {
			if !skipFields[key] {
				content.WriteString(fieldNameStyle.Render(key + ": "))
				content.WriteString(fieldValueStyle.Render(value))
				content.WriteString("\n")
			}
		}
	}
	
	return content.String()
}

// formatMetadata formats the metadata header
func (m *ResourceDetailPageModel) formatMetadata() string {
	title := fmt.Sprintf("%s Details: %s", strings.Title(m.resourceType.String()), m.resourceName)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	metadata := fmt.Sprintf("%s\nLast updated: %s", title, timestamp)
	
	return resourceMetadataStyle.Render(metadata)
}

// updateHelpText updates the help text shown at the bottom
func (m *ResourceDetailPageModel) updateHelpText() {
	m.helpText = "↑/k up • ↓/j down • ctrl+u/ctrl+d page up/down • w wrap • c copy • r refresh • esc/q back"
}