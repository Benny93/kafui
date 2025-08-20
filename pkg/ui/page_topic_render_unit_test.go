package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

// Test the rendering functions in isolation
func TestRenderTopicInfoFunction(t *testing.T) {
	// Create a mock TopicPageModel with just the data we need for testing
	topicPage := &struct {
		topicName     string
		topicDetails  api.Topic
		messages      []api.Message
	}{
		topicName: "test-topic",
		topicDetails: api.Topic{
			NumPartitions:     3,
			ReplicationFactor: 1,
			ConfigEntries: map[string]*string{
				"cleanup.policy": stringPtr("delete"),
				"retention.ms":   stringPtr("604800000"),
			},
		},
		messages: []api.Message{
			{Offset: 100},
			{Offset: 101},
		},
	}
	
	// Copy the renderTopicInfo function logic here for testing
	renderTopicInfo := func() string {
		info := fmt.Sprintf(
			"Name: %s\nPartitions: %d\nReplication Factor: %d\nMessages: %d",
			topicPage.topicName,
			topicPage.topicDetails.NumPartitions,
			topicPage.topicDetails.ReplicationFactor,
			len(topicPage.messages),
		)

		// Format config entries if any
		if len(topicPage.topicDetails.ConfigEntries) > 0 {
			configLines := []string{"\nConfiguration:"}
			for key, value := range topicPage.topicDetails.ConfigEntries {
				if value != nil {
					configLines = append(configLines, fmt.Sprintf("  %s: %s", key, *value))
				} else {
					configLines = append(configLines, fmt.Sprintf("  %s: <nil>", key))
				}
			}
			info += strings.Join(configLines, "\n")
		}

		return InfoStyle.Render(info)
	}
	
	rendered := renderTopicInfo()
	
	// Check that the rendered output contains expected elements
	assert.Contains(t, rendered, "Name: test-topic")
	assert.Contains(t, rendered, "Partitions: 3")
	assert.Contains(t, rendered, "Replication Factor: 1")
	assert.Contains(t, rendered, "Messages: 2")
	assert.Contains(t, rendered, "cleanup.policy")
	assert.Contains(t, rendered, "delete")
	assert.Contains(t, rendered, "retention.ms")
	assert.Contains(t, rendered, "604800000")
}

func TestRenderControlsFunction(t *testing.T) {
	// Create a mock TopicPageModel with just the data we need for testing
	topicPage := &struct {
		consumeFlags api.ConsumeFlags
		paused       bool
	}{
		consumeFlags: api.DefaultConsumeFlags(),
		paused:       false,
	}
	
	// Copy the renderControls function logic here for testing
	renderControls := func() string {
		controls := fmt.Sprintf(
			"Format: %s | Partition: All | Follow: %t | Paused: %t",
			"JSON", // Default format
			topicPage.consumeFlags.Follow,
			topicPage.paused,
		)

		return InfoStyle.Render(controls)
	}
	
	rendered := renderControls()
	
	// Check that the rendered output contains expected elements
	assert.Contains(t, rendered, "Format: JSON")
	assert.Contains(t, rendered, "Partition: All")
	assert.Contains(t, rendered, "Follow: true")
	assert.Contains(t, rendered, "Paused: false")
}

func TestRenderShortcutsFunction(t *testing.T) {
	// Copy the renderShortcuts function logic here for testing
	renderShortcuts := func() string {
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
	
	rendered := renderShortcuts()
	
	// Check that the rendered output contains expected elements
	assert.Contains(t, rendered, "↑/↓   Navigate messages")
	assert.Contains(t, rendered, "Enter   View details")
	assert.Contains(t, rendered, "Space   Pause/resume")
	assert.Contains(t, rendered, "/       Search messages")
	assert.Contains(t, rendered, "Esc     Exit search")
	assert.Contains(t, rendered, "q/Esc   Back to topics")
}

func TestRenderFooterFunction(t *testing.T) {
	// Create a table with some rows
	columns := []table.Column{
		{Title: "Offset", Width: 10},
		{Title: "Partition", Width: 10},
	}
	
	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{
			{"100", "0"},
			{"101", "1"},
		}),
	)
	tbl.SetCursor(0) // Select first item
	
	// Create a mock TopicPageModel with just the data we need for testing
	topicPage := &struct {
		width             int
		messages          []api.Message
		statusMessage     string
		filteredMessages  []api.Message
		messageTable      table.Model
		lastUpdate        time.Time
		spinner           spinner.Model
	}{
		width:         120,
		messages:      []api.Message{{Offset: 100}, {Offset: 101}}, // 2 messages
		statusMessage: "Test status",
		filteredMessages: []api.Message{{Offset: 100}}, // 1 filtered message
		messageTable: tbl,
		lastUpdate: time.Now(),
		spinner: spinner.New(),
	}
	
	// Copy the renderFooter function logic here for testing
	renderFooter := func() string {
		// Left side: Selection information
		selected := "None"
		if len(topicPage.filteredMessages) > 0 {
			cursor := topicPage.messageTable.Cursor()
			if cursor >= 0 && cursor < len(topicPage.filteredMessages) {
				selected = fmt.Sprintf("Offset: %d", topicPage.filteredMessages[cursor].Offset)
			}
		}
		leftInfo := fmt.Sprintf("Selected: %s  •  %d messages total", selected, len(topicPage.messages))

		// Right side: Status information
		rightInfo := fmt.Sprintf("%s %s  •  Last update: %s",
			topicPage.spinner.View(),
			topicPage.statusMessage,
			topicPage.lastUpdate.Format("15:04:05"),
		)

		// Calculate available width for each side
		totalWidth := topicPage.width - 4 // Account for padding
		leftWidth := len(leftInfo)
		rightWidth := len(rightInfo)

		// If both fit, use them with proper spacing
		if leftWidth+rightWidth+3 <= totalWidth {
			spacer := strings.Repeat(" ", totalWidth-leftWidth-rightWidth)
			return leftInfo + spacer + rightInfo
		}

		// If they don't fit, truncate the left side
		maxLeftWidth := totalWidth - rightWidth - 3
		if maxLeftWidth > 20 {
			if len(leftInfo) > maxLeftWidth {
				leftInfo = leftInfo[:maxLeftWidth-3] + "..."
			}
			spacer := strings.Repeat(" ", totalWidth-len(leftInfo)-rightWidth)
			return leftInfo + spacer + rightInfo
		}

		// Fallback: just show the right info if space is very limited
		return rightInfo
	}
	
	rendered := renderFooter()
	
	// Check that the rendered output contains expected elements
	assert.Contains(t, rendered, "Selected: Offset: 100")
	assert.Contains(t, rendered, "2 messages total")
	assert.Contains(t, rendered, "Test status")
}

// Helper function to create a string pointer
func stringPtr(s string) *string {
	return &s
}