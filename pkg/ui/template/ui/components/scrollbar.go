package components

import (
	"strings"

	"github.com/Benny93/kafui/pkg/ui/template/ui/styles"
)

// Scrollbar renders a vertical scrollbar based on content and viewport size.
// Returns an empty string if content fits within viewport (no scrolling needed).
//
// Parameters:
//   - height: The available viewport height
//   - contentSize: The total size of the content (number of lines)
//   - viewportSize: The size of the visible viewport
//   - offset: The current scroll offset
func Scrollbar(height, contentSize, viewportSize, offset int) string {
	if height <= 0 || viewportSize <= 0 || contentSize <= viewportSize {
		return ""
	}

	// Calculate thumb size (minimum 1 character)
	thumbSize := max(1, height*viewportSize/contentSize)

	// Calculate thumb position
	maxOffset := contentSize - viewportSize
	if maxOffset <= 0 {
		return ""
	}

	// Calculate where the thumb starts
	trackSpace := height - thumbSize
	thumbPos := 0
	if trackSpace > 0 && maxOffset > 0 {
		thumbPos = min(trackSpace, offset*trackSpace/maxOffset)
	}

	// Build the scrollbar
	var sb strings.Builder
	t := styles.CurrentTheme()

	for i := range height {
		if i > 0 {
			sb.WriteString("\n")
		}
		if i >= thumbPos && i < thumbPos+thumbSize {
			// Thumb (draggable part)
			sb.WriteString(t.S().Muted.Render("│"))
		} else {
			// Track (background)
			sb.WriteString(t.S().Subtle.Render("│"))
		}
	}

	return sb.String()
}
