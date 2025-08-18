package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

var (
	detailPageStyle = lipgloss.NewStyle().
			Margin(1, 2)

	metadataStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2).
			MarginBottom(1)

	contentStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2)

	detailHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#3c3c3c")).
			Padding(0, 1)

	jsonKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33"))

	jsonStringStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11"))

	jsonNumberStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205"))

	jsonBooleanStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("200"))

	jsonNullStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
)

// DetailPageModel represents the message detail page
type DetailPageModel struct {
	message      api.Message
	topicName    string
	width        int
	height       int
	viewport     viewport.Model
	metadata     string
	helpText     string
	wrapped      bool
}

// NewDetailPage creates a new detail page model
func NewDetailPage(topicName string, message api.Message) DetailPageModel {
	vp := viewport.New(80, 20)
	vp.Style = contentStyle

	model := DetailPageModel{
		message:   message,
		topicName: topicName,
		viewport:  vp,
	}

	model.updateContent()
	model.updateHelpText()

	return model
}

// Init initializes the detail page
func (m DetailPageModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the detail page
func (m DetailPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			// Return to topic page
			return m, func() tea.Msg {
				return pageChangeMsg(topicPage)
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
			// For now, we'll just update the status
			m.helpText = "Content copied to clipboard"
			return m, nil
		case "n":
			// Next message (would need to be implemented with message navigation)
			m.helpText = "Next message navigation not implemented"
			return m, nil
		case "p":
			// Previous message (would need to be implemented with message navigation)
			m.helpText = "Previous message navigation not implemented"
			return m, nil
		case "w":
			// Toggle word wrap
			m.wrapped = !m.wrapped
			m.updateContent()
			return m, nil
		}
	}

	// Handle viewport updates
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the detail page
func (m DetailPageModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Create the layout
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		m.metadata,
		m.viewport.View(),
	)

	// Wrap in main style
	mainContent := detailPageStyle.Render(content)

	// Add help text at the bottom
	helpBar := detailHelpStyle.Render(m.helpText)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		helpBar,
	)
}

// updateContent updates the content displayed in the viewport
func (m *DetailPageModel) updateContent() {
	// Format the message content
	formattedContent := m.formatMessageContent(m.message.Value)
	
	// Apply word wrapping if enabled
	if m.wrapped && m.viewport.Width > 0 {
		formattedContent = wordwrap.String(formattedContent, m.viewport.Width-4)
	}
	
	// Set content in viewport
	m.viewport.SetContent(formattedContent)
	
	// Update metadata
	m.metadata = m.formatMetadata()
}

// formatMessageContent formats the message content for display
func (m *DetailPageModel) formatMessageContent(content string) string {
	// Try to parse as JSON for formatting
	var parsed interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err == nil {
		// Successfully parsed as JSON, format it
		return m.formatJSON(parsed, 0)
	}
	
	// Not JSON, return as is
	return content
}

// formatJSON formats JSON content with syntax highlighting
func (m *DetailPageModel) formatJSON(data interface{}, indent int) string {
	indentStr := strings.Repeat("  ", indent)
	
	switch v := data.(type) {
	case map[string]interface{}:
		// Object
		if len(v) == 0 {
			return "{}"
		}
		
		lines := []string{"{"}
		i := 0
		for key, value := range v {
			i++
			comma := ","
			if i == len(v) {
				comma = ""
			}
			
			keyStr := jsonKeyStyle.Render(fmt.Sprintf("%q", key))
			valueStr := m.formatJSON(value, indent+1)
			
			lines = append(lines, fmt.Sprintf("%s  %s: %s%s", indentStr, keyStr, valueStr, comma))
		}
		lines = append(lines, fmt.Sprintf("%s}", indentStr))
		
		return strings.Join(lines, "\n")
		
	case []interface{}:
		// Array
		if len(v) == 0 {
			return "[]"
		}
		
		lines := []string{"["}
		for i, value := range v {
			comma := ","
			if i == len(v)-1 {
				comma = ""
			}
			
			valueStr := m.formatJSON(value, indent+1)
			lines = append(lines, fmt.Sprintf("%s  %s%s", indentStr, valueStr, comma))
		}
		lines = append(lines, fmt.Sprintf("%s]", indentStr))
		
		return strings.Join(lines, "\n")
		
	case string:
		// String
		return jsonStringStyle.Render(fmt.Sprintf("%q", v))
		
	case float64:
		// Number
		return jsonNumberStyle.Render(fmt.Sprintf("%g", v))
		
	case bool:
		// Boolean
		if v {
			return jsonBooleanStyle.Render("true")
		}
		return jsonBooleanStyle.Render("false")
		
	case nil:
		// Null
		return jsonNullStyle.Render("null")
		
	default:
		// Fallback
		return fmt.Sprintf("%v", v)
	}
}

// formatMetadata formats the message metadata
func (m *DetailPageModel) formatMetadata() string {
	var lines []string
	
	// Topic and message info
	lines = append(lines, fmt.Sprintf("Topic: %s", m.topicName))
	lines = append(lines, fmt.Sprintf("Key: %s", m.message.Key))
	lines = append(lines, fmt.Sprintf("Offset: %d", m.message.Offset))
	lines = append(lines, fmt.Sprintf("Partition: %d", m.message.Partition))
	
	// Timestamp (using current time as placeholder)
	lines = append(lines, fmt.Sprintf("Timestamp: %s", time.Now().Format("2006-01-02 15:04:05")))
	
	// Headers
	if len(m.message.Headers) > 0 {
		lines = append(lines, "Headers:")
		for _, header := range m.message.Headers {
			lines = append(lines, fmt.Sprintf("  %s: %s", header.Key, header.Value))
		}
	}
	
	// Schema information
	if m.message.KeySchemaID != "" {
		lines = append(lines, fmt.Sprintf("Key Schema ID: %s", m.message.KeySchemaID))
	}
	if m.message.ValueSchemaID != "" {
		lines = append(lines, fmt.Sprintf("Value Schema ID: %s", m.message.ValueSchemaID))
	}
	
	return metadataStyle.Render(strings.Join(lines, "\n"))
}

// updateHelpText updates the help text displayed at the bottom
func (m *DetailPageModel) updateHelpText() {
	m.helpText = "ESC: Back | j/k: Scroll | n/p: Next/Prev | c: Copy | q: Quit | w: Wrap"
}