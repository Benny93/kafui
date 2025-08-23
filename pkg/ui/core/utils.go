package core

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// FormatTime formats a time.Time to a human-readable string
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	return t.Format("15:04:05")
}

// FormatDuration formats a duration to a human-readable string
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	} else {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
}

// TruncateString truncates a string to the specified length with ellipsis
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}

// PadString pads a string to the specified width
func PadString(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// CenterString centers a string within the specified width
func CenterString(s string, width int) string {
	if len(s) >= width {
		return s
	}
	
	padding := width - len(s)
	leftPadding := padding / 2
	rightPadding := padding - leftPadding
	
	return strings.Repeat(" ", leftPadding) + s + strings.Repeat(" ", rightPadding)
}

// WrapText wraps text to fit within the specified width
func WrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	
	lines := []string{}
	currentLine := ""
	
	for _, word := range words {
		// If adding this word would exceed the width, start a new line
		if len(currentLine)+len(word)+1 > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = word
			} else {
				// Word is longer than width, break it
				for len(word) > width {
					lines = append(lines, word[:width])
					word = word[width:]
				}
				currentLine = word
			}
		} else {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		}
	}
	
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	
	return lines
}

// StatusMessage formats a status message with type prefix
func StatusMessage(message string, statusType StatusType) string {
	var prefix string
	switch statusType {
	case StatusError:
		prefix = "ERROR"
	case StatusSuccess:
		prefix = "SUCCESS"
	case StatusWarning:
		prefix = "WARNING"
	case StatusInfo:
		prefix = "INFO"
	default:
		prefix = "STATUS"
	}
	return fmt.Sprintf("[%s] %s", prefix, message)
}

// FormatCount formats a count with appropriate units
func FormatCount(count int) string {
	if count < 1000 {
		return fmt.Sprintf("%d", count)
	} else if count < 1000000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	} else {
		return fmt.Sprintf("%.1fM", float64(count)/1000000)
	}
}

// FormatBytes formats bytes to human-readable string
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// IsKeyMatch checks if a key message matches any of the provided key bindings
func IsKeyMatch(msg tea.KeyMsg, bindings ...key.Binding) bool {
	for _, binding := range bindings {
		if key.Matches(msg, binding) {
			return true
		}
	}
	return false
}

// CalculateTableDimensions calculates optimal dimensions for table columns
func CalculateTableDimensions(totalWidth int, columnWeights []float64) []int {
	if len(columnWeights) == 0 {
		return []int{}
	}
	
	// Normalize weights to sum to 1.0
	totalWeight := 0.0
	for _, weight := range columnWeights {
		totalWeight += weight
	}
	
	// Calculate column widths
	widths := make([]int, len(columnWeights))
	usedWidth := 0
	
	for i, weight := range columnWeights {
		normalizedWeight := weight / totalWeight
		widths[i] = int(float64(totalWidth) * normalizedWeight)
		usedWidth += widths[i]
	}
	
	// Distribute any remaining width to the last column
	if usedWidth < totalWidth {
		widths[len(widths)-1] += totalWidth - usedWidth
	}
	
	return widths
}

// ValidateStringInput validates string input for common use cases
func ValidateStringInput(input string, rules ...func(string) error) error {
	for _, rule := range rules {
		if err := rule(input); err != nil {
			return err
		}
	}
	return nil
}

// Common validation rules
func NotEmpty(input string) error {
	if strings.TrimSpace(input) == "" {
		return fmt.Errorf("input cannot be empty")
	}
	return nil
}

func MaxLength(maxLen int) func(string) error {
	return func(input string) error {
		if len(input) > maxLen {
			return fmt.Errorf("input cannot exceed %d characters", maxLen)
		}
		return nil
	}
}

func MinLength(minLen int) func(string) error {
	return func(input string) error {
		if len(input) < minLen {
			return fmt.Errorf("input must be at least %d characters", minLen)
		}
		return nil
	}
}

// DebugMessage creates a debug message for development
func DebugMessage(format string, args ...interface{}) tea.Cmd {
	return func() tea.Msg {
		message := fmt.Sprintf(format, args...)
		return StatusMsg{
			Message: fmt.Sprintf("[DEBUG] %s", message),
			Type:    StatusInfo,
		}
	}
}

// BatchCommands combines multiple commands into a single batch command
func BatchCommands(cmds ...tea.Cmd) tea.Cmd {
	validCmds := make([]tea.Cmd, 0, len(cmds))
	for _, cmd := range cmds {
		if cmd != nil {
			validCmds = append(validCmds, cmd)
		}
	}
	
	if len(validCmds) == 0 {
		return nil
	}
	
	return tea.Batch(validCmds...)
}

// Timer utilities for creating periodic updates
func CreateTimer(id string, duration time.Duration) tea.Cmd {
	return tea.Tick(duration, func(t time.Time) tea.Msg {
		return TimerTickMsg{
			Time: t,
			ID:   id,
		}
	})
}

// Color utilities for highlighting text
func HighlightMatches(text, query string) string {
	if query == "" {
		return text
	}
	
	// Simple case-insensitive highlighting
	lower := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)
	
	if strings.Contains(lower, lowerQuery) {
		// For now, just return the text as-is
		// In a real implementation, you'd want to wrap matches in styling
		return text
	}
	
	return text
}