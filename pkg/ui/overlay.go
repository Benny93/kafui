package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// composite draws an overlay centred over a base view, keeping the base's cells
// outside the overlay's column span.
//
// The naive version of this replaced the whole terminal row with the overlay
// line, which erased the sidebar, the table's right-hand columns and the page
// frame on every row the box covered — so a modal read as a rendering failure
// rather than as something drawn on top. Splitting each row with ANSI-aware
// truncation keeps the styling of the parts that are not covered.
func composite(base, over string, width, height int) string {
	if over == "" {
		return base
	}
	// A one-cell blank gutter around the box. Without it, the modal covering
	// the sidebar's border and bullet column reads as though the text lost a
	// character; with it, the overlap is obviously something drawn on top.
	over = gutter(over)

	overLines := strings.Split(over, "\n")
	overWidth := lipgloss.Width(over)

	top := (height - len(overLines)) / 2
	if top < 0 {
		top = 0
	}
	left := (width - overWidth) / 2
	if left < 0 {
		left = 0
	}

	baseLines := strings.Split(base, "\n")
	// The overlay may extend past the base's last line — pad rather than drop it.
	for len(baseLines) < top+len(overLines) {
		baseLines = append(baseLines, "")
	}

	for i, overLine := range overLines {
		row := top + i
		baseLine := baseLines[row]

		// Cells [0, left) of the base survive untouched.
		prefix := ansi.Truncate(baseLine, left, "")
		// A short base line has to be padded out to the overlay's left edge, or
		// the overlay would slide back against it.
		if pad := left - ansi.StringWidth(prefix); pad > 0 {
			prefix += strings.Repeat(" ", pad)
		}

		// Pad a short overlay line out to the block width, so every row consumes
		// the same span. Measuring per line and cutting the base by the block
		// width instead would eat cells the base still needed — which showed up
		// as the sidebar losing its border and first character.
		if d := overWidth - ansi.StringWidth(overLine); d > 0 {
			overLine += strings.Repeat(" ", d)
		}

		// Cells [left+overWidth, …) of the base survive too. TruncateLeft drops
		// the leading cells while carrying their styles forward, so colours on
		// the right of the overlay stay correct.
		consumed := ansi.StringWidth(prefix) + ansi.StringWidth(overLine)
		suffix := ansi.TruncateLeft(baseLine, consumed, "")

		baseLines[row] = prefix + overLine + suffix
	}

	return strings.Join(baseLines, "\n")
}

// gutter surrounds a rendered block with one blank cell on every side.
func gutter(block string) string {
	lines := strings.Split(block, "\n")
	w := lipgloss.Width(block)
	blank := strings.Repeat(" ", w+2)

	out := make([]string, 0, len(lines)+2)
	out = append(out, blank)
	for _, l := range lines {
		pad := w - lipgloss.Width(l)
		if pad < 0 {
			pad = 0
		}
		out = append(out, " "+l+strings.Repeat(" ", pad)+" ")
	}
	return strings.Join(append(out, blank), "\n")
}
