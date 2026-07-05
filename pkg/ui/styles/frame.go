package styles

import "github.com/charmbracelet/lipgloss"

// FrameTable wraps a bubbles/table view in a rounded border so detail-page
// tables read as framed tables (matching the main page's evertras look)
// instead of bare rows floating beside the content pane's own border.
// The wrapped view is 2 cells wider and taller than v, so size the table to
// (contentWidth-2) before framing to avoid overflowing the pane.
func FrameTable(v string) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(FgSubtle).
		Render(v)
}
