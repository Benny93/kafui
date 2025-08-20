package ui

import (
	"strconv"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

// MinimalTopicPageModel is a minimal version of TopicPageModel for testing
type MinimalTopicPageModel struct {
	topicName         string
	topicDetails      api.Topic
	consumeFlags      api.ConsumeFlags
	messages          []api.Message
	filteredMessages  []api.Message
	messageTable      table.Model
	spinner           spinner.Model
	statusMessage     string
	searchMode        bool
	searchInput       textinput.Model
	width             int
	height            int
}

func NewMinimalTopicPage(topicName string, topicDetails api.Topic) *MinimalTopicPageModel {
	// Initialize message table
	columns := []table.Column{
		{Title: "Offset", Width: 10},
		{Title: "Partition", Width: 10},
		{Title: "Timestamp", Width: 20},
		{Title: "Key", Width: 20},
		{Title: "Value", Width: 40},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(Subtle).
		BorderBottom(true).
		Bold(true)
	s.Selected = lipgloss.NewStyle().
		Background(lipgloss.Color("205")).
		Foreground(lipgloss.Color("0"))
	t.SetStyles(s)

	// Initialize search input
	ti := textinput.New()
	ti.Placeholder = "Search messages..."
	ti.CharLimit = 156
	ti.Width = 30

	// Initialize spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &MinimalTopicPageModel{
		topicName:        topicName,
		topicDetails:     topicDetails,
		consumeFlags:     api.DefaultConsumeFlags(),
		messages:         []api.Message{},
		filteredMessages: []api.Message{},
		messageTable:     t,
		spinner:          sp,
		statusMessage:    "Topic page initialized",
		searchInput:      ti,
	}
}

func (m *MinimalTopicPageModel) View() string {
	// Calculate layout dimensions
	sidebarWidth := 35
	contentWidth := m.width - sidebarWidth - 3 // Account for padding and sidebar gap
	if contentWidth < 0 {
		contentWidth = m.width
	}
	
	contentHeight := m.height - 8 // Account for header and footer
	if contentHeight < 0 {
		contentHeight = 1
	}

	// Header section
	header := HeaderStyle.
		Width(m.width).
		Render("Kafui - Topic: " + m.topicName)

	// Main content area with controls, messages, and search
	controlsSection := MainPanelStyle.
		Width(contentWidth).
		Render(m.renderControls())

	// Calculate available height for messages section
	usedHeight := 2 // controls section height (approx)
	if m.searchMode {
		usedHeight += 2 // search input height (approx)
	}
	
	availableHeight := contentHeight - usedHeight - 1 // -1 for padding
	if availableHeight < 1 {
		availableHeight = 1
	}
	
	// Update table height to fill available space
	m.messageTable.SetHeight(availableHeight)

	messagesSection := MainPanelStyle.
		Width(contentWidth).
		Height(availableHeight).
		Render(m.renderMessages())

	var searchSection string
	if m.searchMode {
		searchSection = MainPanelStyle.
			Width(contentWidth).
			Render(m.searchInput.View())
	}

	// Combine main content sections
	var mainContent string
	if m.searchMode {
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
	sidebarContent := lipgloss.JoinVertical(
		lipgloss.Left,
		TitleStyle.Render("TOPIC INFO"),
		m.renderTopicInfo(),
		lipgloss.NewStyle().MarginTop(2).Render(""),
		SubtitleStyle.Render("SHORTCUTS"),
		m.renderShortcuts(),
	)

	sidebar := SidebarPanelStyle.
		Width(sidebarWidth).
		Height(contentHeight).
		Render(sidebarContent)

	// Combine main content and sidebar
	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		mainContent,
		lipgloss.NewStyle().Width(1).Render(""), // Minimal spacer
		sidebar,
	)

	// Footer with key bindings
	footer := FooterStyle.Width(m.width).Render(m.renderFooter())

	// Combine all sections
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		LayoutStyle.Render(body),
		footer,
	)
}

func (m *MinimalTopicPageModel) renderTopicInfo() string {
	info := "Name: " + m.topicName + "\n"
	info += "Partitions: " + strconv.Itoa(int(m.topicDetails.NumPartitions)) + "\n"
	info += "Replication Factor: " + strconv.Itoa(int(m.topicDetails.ReplicationFactor)) + "\n"
	info += "Messages: " + strconv.Itoa(len(m.messages))

	// Format config entries if any
	if len(m.topicDetails.ConfigEntries) > 0 {
		info += "\nConfiguration:"
		for key, value := range m.topicDetails.ConfigEntries {
			if value != nil {
				info += "\n  " + key + ": " + *value
			} else {
				info += "\n  " + key + ": <nil>"
			}
		}
	}

	return InfoStyle.Render(info)
}

func (m *MinimalTopicPageModel) renderControls() string {
	controls := "Format: JSON | Partition: All | Follow: " + strconv.FormatBool(m.consumeFlags.Follow) + " | Paused: false"
	return InfoStyle.Render(controls)
}

func (m *MinimalTopicPageModel) renderMessages() string {
	if len(m.filteredMessages) == 0 {
		return "No messages available. Press Space to start consumption."
	}

	return m.messageTable.View()
}

func (m *MinimalTopicPageModel) renderShortcuts() string {
	shortcuts := []string{
		"↑/↓   Navigate messages",
		"Enter   View details",
		"Space   Pause/resume",
		"/       Search messages",
		"Esc     Exit search",
		"q/Esc   Back to topics",
	}

	return lipgloss.JoinVertical(lipgloss.Left, shortcuts...)
}

func (m *MinimalTopicPageModel) renderFooter() string {
	// Left side: Selection information
	selected := "None"
	if len(m.filteredMessages) > 0 {
		cursor := m.messageTable.Cursor()
		if cursor >= 0 && cursor < len(m.filteredMessages) {
			selected = "Offset: " + strconv.FormatInt(m.filteredMessages[cursor].Offset, 10)
		}
	}
	leftInfo := "Selected: " + selected + "  •  " + strconv.Itoa(len(m.messages)) + " messages total"

	// Right side: Status information
	rightInfo := m.spinner.View() + " " + m.statusMessage + "  •  Last update: 00:00:00"

	// Calculate available width for each side
	totalWidth := m.width - 4 // Account for padding
	leftWidth := len(leftInfo)
	rightWidth := len(rightInfo)

	// If both fit, use them with proper spacing
	if leftWidth+rightWidth+3 <= totalWidth {
		spacer := ""
		for i := 0; i < totalWidth-leftWidth-rightWidth; i++ {
			spacer += " "
		}
		return leftInfo + spacer + rightInfo
	}

	// If they don't fit, truncate the left side
	maxLeftWidth := totalWidth - rightWidth - 3
	if maxLeftWidth > 20 {
		if len(leftInfo) > maxLeftWidth {
			leftInfo = leftInfo[:maxLeftWidth-3] + "..."
		}
		spacer := ""
		for i := 0; i < totalWidth-len(leftInfo)-rightWidth; i++ {
			spacer += " "
		}
		return leftInfo + spacer + rightInfo
	}

	// Fallback: just show the right info if space is very limited
	return rightInfo
}

func TestTopicPageViewRender(t *testing.T) {
	cleanupPolicy := "delete"
	retentionMs := "604800000"
	
	topicDetails := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 1,
		ConfigEntries: map[string]*string{
			"cleanup.policy": &cleanupPolicy,
			"retention.ms":   &retentionMs,
		},
	}

	topicPage := NewMinimalTopicPage("test-topic", topicDetails)
	
	// Set window size for rendering
	topicPage.width = 120
	topicPage.height = 40
	
	// Render the view
	rendered := topicPage.View()
	
	// Check that the rendered output contains expected elements
	assert.Contains(t, rendered, "Kafui - Topic: test-topic")
	assert.Contains(t, rendered, "TOPIC INFO")
	assert.Contains(t, rendered, "Name: test-topic")
	assert.Contains(t, rendered, "Partitions: 3")
	assert.Contains(t, rendered, "Replication Factor: 1")
	assert.Contains(t, rendered, "SHORTCUTS")
	assert.Contains(t, rendered, "No messages available. Press Space to start consumption.")
	
	// Check that configuration entries are displayed
	assert.Contains(t, rendered, "cleanup.policy")
	assert.Contains(t, rendered, "delete")
	assert.Contains(t, rendered, "retention.ms")
	assert.Contains(t, rendered, "604800000")
}

func TestTopicPageViewWithMessages(t *testing.T) {
	topicDetails := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 1,
	}

	topicPage := NewMinimalTopicPage("test-topic", topicDetails)
	
	// Set window size for rendering
	topicPage.width = 120
	topicPage.height = 40
	
	// Add some mock messages
	topicPage.messages = []api.Message{
		{
			Key:       "test-key-1",
			Value:     "test-value-1",
			Offset:    100,
			Partition: 0,
		},
		{
			Key:       "test-key-2",
			Value:     "test-value-2",
			Offset:    101,
			Partition: 1,
		},
	}
	topicPage.filteredMessages = topicPage.messages
	
	// Convert messages to table rows
	rows := make([]table.Row, len(topicPage.filteredMessages))
	for i, msg := range topicPage.filteredMessages {
		// Format timestamp
		timestamp := "2023-01-01 12:00:00"

		// Truncate long values
		value := msg.Value
		if len(value) > 50 {
			value = value[:47] + "..."
		}

		key := msg.Key
		if len(key) > 20 {
			key = key[:17] + "..."
		}

		rows[i] = table.Row{
			strconv.FormatInt(msg.Offset, 10),
			strconv.FormatInt(int64(msg.Partition), 10),
			timestamp,
			key,
			value,
		}
	}

	topicPage.messageTable.SetRows(rows)
	
	// Render the view
	rendered := topicPage.View()
	
	// Check that the rendered output contains expected elements
	assert.Contains(t, rendered, "Kafui - Topic: test-topic")
	assert.Contains(t, rendered, "TOPIC INFO")
	assert.Contains(t, rendered, "Name: test-topic")
	
	// Check that messages are displayed in the table
	assert.Contains(t, rendered, "test-key-1")
	assert.Contains(t, rendered, "test-value-") // Value is truncated
	assert.Contains(t, rendered, "test-key-2")
	assert.Contains(t, rendered, "test-value-") // Value is truncated
}

func TestTopicPageViewSearchMode(t *testing.T) {
	topicDetails := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 1,
	}

	topicPage := NewMinimalTopicPage("test-topic", topicDetails)
	
	// Set window size for rendering
	topicPage.width = 120
	topicPage.height = 40
	
	// Enable search mode
	topicPage.searchMode = true
	
	// Render the view
	rendered := topicPage.View()
	
	// Check that the rendered output contains search elements
	assert.Contains(t, rendered, "Search messages...")
	assert.Contains(t, rendered, "test-topic")
}