package components

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// JSONContentConfig holds configuration for the JSON content viewer
type JSONContentConfig struct {
	Width          int
	Height         int
	Title          string
	Content        string
	DisplayFormat  string // "raw", "json", "pretty"
	ShowLineNumbers bool
	Focused        bool
}

// JSONContentView represents a reusable JSON content viewer component
type JSONContentView struct {
	config      JSONContentConfig
	viewport    viewport.Model
	titleStyle  lipgloss.Style
	borderStyle lipgloss.Style
}

// NewJSONContentView creates a new JSON content viewer component
func NewJSONContentView(config JSONContentConfig) *JSONContentView {
	// Ensure minimum dimensions
	if config.Width < 10 {
		config.Width = 10
	}
	if config.Height < 5 {
		config.Height = 5
	}
	
	// Account for borders and title
	viewportWidth := config.Width - 2
	if viewportWidth < 1 {
		viewportWidth = 1
	}
	
	viewportHeight := config.Height - 3 // Title + border + content
	if viewportHeight < 1 {
		viewportHeight = 1
	}
	
	vp := viewport.New(viewportWidth, viewportHeight)
	
	// Create content based on format
	content := formatContent(config.Content, config.DisplayFormat)
	if config.ShowLineNumbers {
		content = addLineNumbers(content)
	}
	
	vp.SetContent(content)
	
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("62")).
		Bold(true).
		Padding(0, 1)

	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder())

	if config.Focused {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("205")) // Pink for focused
	} else {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("240")) // Gray for unfocused
	}

	return &JSONContentView{
		config:      config,
		viewport:    vp,
		titleStyle:  titleStyle,
		borderStyle: borderStyle,
	}
}

// formatContent formats content based on the display format
func formatContent(content, format string) string {
	if content == "" {
		return "<null>"
	}

	switch format {
	case "json", "pretty":
		return formatAsJSON(content)
	case "hex":
		return fmt.Sprintf("%x", content)
	default:
		return content
	}
}

// formatAsJSON attempts to parse and pretty print JSON content
func formatAsJSON(content string) string {
	var parsed interface{}

	// Try to unmarshal as JSON
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		// If parsing fails, try to unescape and parse again
		// This handles cases where JSON is double-encoded
		var unescapedContent string
		if err := json.Unmarshal([]byte(content), &unescapedContent); err == nil {
			// Try parsing the unescaped content
			if err := json.Unmarshal([]byte(unescapedContent), &parsed); err == nil {
				// Successfully parsed unescaped content
				content = unescapedContent
			} else {
				// Use the unescaped content as a string
				parsed = unescapedContent
			}
		} else {
			// If parsing fails, return original content
			return content
		}
	}

	// Marshal with indentation for pretty printing
	pretty, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		// If pretty printing fails, return original content
		return content
	}

	// For now, return JSON as-is to avoid ANSI escape sequence issues
	// TODO: Implement proper syntax highlighting that works with viewport
	return string(pretty)
}

// addLineNumbers adds line numbers to content
func addLineNumbers(content string) string {
	if content == "<null>" {
		return content
	}

	lines := strings.Split(content, "\n")
	numberedLines := make([]string, len(lines))

	for i, line := range lines {
		lineNumber := fmt.Sprintf("%4d ", i+1)
		numberedLines[i] = lineNumber + line
	}

	return strings.Join(numberedLines, "\n")
}

// UpdateConfig updates the component configuration
func (jcv *JSONContentView) UpdateConfig(config JSONContentConfig) {
	// Ensure minimum dimensions
	if config.Width < 10 {
		config.Width = 10
	}
	if config.Height < 5 {
		config.Height = 5
	}
	
	jcv.config = config
	
	// Update viewport dimensions
	viewportWidth := config.Width - 2
	if viewportWidth < 1 {
		viewportWidth = 1
	}
	
	viewportHeight := config.Height - 3 // Title + border + content
	if viewportHeight < 1 {
		viewportHeight = 1
	}
	
	jcv.viewport.Width = viewportWidth
	jcv.viewport.Height = viewportHeight
	
	// Update content
	content := formatContent(config.Content, config.DisplayFormat)
	if config.ShowLineNumbers {
		content = addLineNumbers(content)
	}
	jcv.viewport.SetContent(content)
	
	// Update styles
	if config.Focused {
		jcv.borderStyle = jcv.borderStyle.BorderForeground(lipgloss.Color("205")) // Pink for focused
	} else {
		jcv.borderStyle = jcv.borderStyle.BorderForeground(lipgloss.Color("240")) // Gray for unfocused
	}
}

// SetContent updates the content displayed in the viewer
func (jcv *JSONContentView) SetContent(content string) {
	jcv.config.Content = content
	formattedContent := formatContent(content, jcv.config.DisplayFormat)
	if jcv.config.ShowLineNumbers {
		formattedContent = addLineNumbers(formattedContent)
	}
	jcv.viewport.SetContent(formattedContent)
}

// SetDisplayFormat updates the display format
func (jcv *JSONContentView) SetDisplayFormat(format string) {
	jcv.config.DisplayFormat = format
	jcv.SetContent(jcv.config.Content) // Reformat content with new format
}

// SetFocused updates the focus state
func (jcv *JSONContentView) SetFocused(focused bool) {
	jcv.config.Focused = focused
	if focused {
		jcv.borderStyle = jcv.borderStyle.BorderForeground(lipgloss.Color("205")) // Pink for focused
	} else {
		jcv.borderStyle = jcv.borderStyle.BorderForeground(lipgloss.Color("240")) // Gray for unfocused
	}
}

// LineUp moves the viewport up by the given number of lines
func (jcv *JSONContentView) LineUp(lines int) {
	jcv.viewport.LineUp(lines)
}

// LineDown moves the viewport down by the given number of lines
func (jcv *JSONContentView) LineDown(lines int) {
	jcv.viewport.LineDown(lines)
}

// View renders the JSON content viewer
func (jcv *JSONContentView) View() string {
	// Render title
	title := jcv.titleStyle.Render(jcv.config.Title)

	// Render viewport content
	content := jcv.viewport.View()

	// Combine title and content with border
	contentWithBorder := jcv.borderStyle.Render(content)

	// Combine title and content
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		contentWithBorder,
	)
}

// Viewport returns the underlying viewport for direct manipulation
func (jcv *JSONContentView) Viewport() *viewport.Model {
	return &jcv.viewport
}