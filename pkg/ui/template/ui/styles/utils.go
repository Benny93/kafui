package styles

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/rivo/uniseg"
)

// Section creates a section header with a line
func Section(text string, width int) string {
	t := CurrentTheme()
	char := "─"
	length := lipgloss.Width(text) + 1
	remainingWidth := width - length

	if remainingWidth > 0 {
		return lipgloss.JoinHorizontal(
			lipgloss.Left,
			t.S().Subtitle.Render(text),
			" ",
			t.S().Muted.Render(strings.Repeat(char, remainingWidth)),
		)
	}
	return t.S().Subtitle.Render(text)
}

// Title creates a title with decorative lines
func Title(title string, width int) string {
	t := CurrentTheme()
	char := "─"
	length := lipgloss.Width(title) + 2 // +2 for spaces
	remainingWidth := width - length

	if remainingWidth > 0 {
		sideWidth := remainingWidth / 2
		leftSide := strings.Repeat(char, sideWidth)
		rightSide := strings.Repeat(char, remainingWidth-sideWidth)

		return lipgloss.JoinHorizontal(
			lipgloss.Left,
			t.S().Muted.Render(leftSide),
			" ",
			t.S().Title.Render(title),
			" ",
			t.S().Muted.Render(rightSide),
		)
	}
	return t.S().Title.Render(title)
}

// ApplyForegroundGrad renders a string with a horizontal gradient foreground
func ApplyForegroundGrad(input string, color1, color2 lipgloss.Color) string {
	if input == "" {
		return ""
	}
	
	var clusters []string
	gr := uniseg.NewGraphemes(input)
	for gr.Next() {
		clusters = append(clusters, string(gr.Runes()))
	}

	if len(clusters) == 1 {
		style := CurrentTheme().S().Base.Foreground(color1)
		return style.Render(input)
	}

	ramp := blendColors(len(clusters), color1, color2)
	var o strings.Builder
	
	for i, cluster := range clusters {
		style := CurrentTheme().S().Base.Foreground(ramp[i])
		fmt.Fprint(&o, style.Render(cluster))
	}
	
	return o.String()
}

// ApplyBoldForegroundGrad renders a string with a bold horizontal gradient foreground
func ApplyBoldForegroundGrad(input string, color1, color2 lipgloss.Color) string {
	if input == "" {
		return ""
	}
	
	var clusters []string
	gr := uniseg.NewGraphemes(input)
	for gr.Next() {
		clusters = append(clusters, string(gr.Runes()))
	}

	if len(clusters) == 1 {
		style := CurrentTheme().S().Base.Foreground(color1).Bold(true)
		return style.Render(input)
	}

	ramp := blendColors(len(clusters), color1, color2)
	var o strings.Builder
	
	for i, cluster := range clusters {
		style := CurrentTheme().S().Base.Foreground(ramp[i]).Bold(true)
		fmt.Fprint(&o, style.Render(cluster))
	}
	
	return o.String()
}

// blendColors returns a slice of colors blended between the given colors
func blendColors(size int, color1, color2 lipgloss.Color) []lipgloss.Color {
	if size <= 1 {
		return []lipgloss.Color{color1}
	}

	c1, _ := colorful.Hex(string(color1))
	c2, _ := colorful.Hex(string(color2))
	
	colors := make([]lipgloss.Color, size)
	for i := 0; i < size; i++ {
		t := float64(i) / float64(size-1)
		blended := c1.BlendHcl(c2, t)
		colors[i] = lipgloss.Color(blended.Hex())
	}
	
	return colors
}

// StatusIcon returns an icon for the given status
func StatusIcon(status string) string {
	switch status {
	case "online":
		return "●"
	case "offline":
		return "○"
	case "error":
		return "×"
	case "warning":
		return "⚠"
	case "success":
		return "✓"
	default:
		return "●"
	}
}

// Size mode constants based on CRUSH CLI patterns
const (
	// Window size breakpoints
	MinimumWindowWidth  = 25  // Below this shows "Window too small!"
	MinimumWindowHeight = 15  // Below this shows "Window too small!"
	
	// Small screen breakpoints (for small logo and compact layout)
	SmallScreenWidth  = 55  // Below this uses small logo and compact info
	SmallScreenHeight = 20  // Below this uses small logo and compact info
	
	// Compact mode breakpoints (for sidebar hiding and compact components)
	CompactModeWidth  = 120 // Below this enters compact mode
	CompactModeHeight = 30  // Below this enters compact mode
	
	// Big header mode breakpoint (for large splash screen)
	BigHeaderModeWidth = 140 // Above this shows big header with full logo
)

// SizeMode represents the current UI size mode
type SizeMode int

const (
	// SizeModeMinimum shows only "Window too small!" message
	SizeModeMinimum SizeMode = iota
	
	// SizeModeSmall shows small logo and minimal content
	SizeModeSmall
	
	// SizeModeCompact shows normal logo but compact layout
	SizeModeCompact
	
	// SizeModeNormal shows full layout with sidebar
	SizeModeNormal
	
	// SizeModeBig shows large header and full layout
	SizeModeBig
)

// GetSizeMode determines the appropriate size mode based on window dimensions
func GetSizeMode(width, height int) SizeMode {
	// Check for minimum size first
	if width < MinimumWindowWidth || height < MinimumWindowHeight {
		return SizeModeMinimum
	}
	
	// Check for small screen
	if width < SmallScreenWidth || height < SmallScreenHeight {
		return SizeModeSmall
	}
	
	// Check for compact mode
	if width < CompactModeWidth || height < CompactModeHeight {
		return SizeModeCompact
	}
	
	// Check for big header mode
	if width >= BigHeaderModeWidth {
		return SizeModeBig
	}
	
	// Default to normal mode
	return SizeModeNormal
}

// String returns a string representation of the size mode
func (s SizeMode) String() string {
	switch s {
	case SizeModeMinimum:
		return "minimum"
	case SizeModeSmall:
		return "small"
	case SizeModeCompact:
		return "compact"
	case SizeModeNormal:
		return "normal"
	case SizeModeBig:
		return "big"
	default:
		return "unknown"
	}
}

// DebugInfo renders debug information showing component dimensions
func DebugInfo(componentName string, width, height int) string {
	t := CurrentTheme()
	sizeMode := GetSizeMode(width, height)
	
	debugText := fmt.Sprintf("[%s: %dx%d, %s]", componentName, width, height, sizeMode.String())
	
	return t.S().Base.
		Foreground(t.FgSubtle).
		Background(t.BgSubtle).
		Padding(0, 1).
		Render(debugText)
}

// RenderLogo renders the CRUSH-style logo based on size mode
func RenderLogo(width int, sizeMode SizeMode, version string) string {
	switch sizeMode {
	case SizeModeMinimum: // SizeModeMinimum - no logo
		return ""
		
	case SizeModeSmall: // SizeModeSmall - small logo like CRUSH SmallRender
		return RenderSmallLogo(width)
		
	case SizeModeCompact, SizeModeNormal: // SizeModeCompact, SizeModeNormal - normal logo
		return RenderNormalLogo(width, version, false)
		
	case SizeModeBig: // SizeModeBig - big logo with extra styling
		return RenderNormalLogo(width, version, true)
		
	default:
		return RenderNormalLogo(width, version, false)
	}
}

// RenderSmallLogo renders a small version similar to CRUSH SmallRender
func RenderSmallLogo(width int) string {
	t := CurrentTheme()
	title := t.S().Base.Foreground(t.Secondary).Render("Example™")
	title = fmt.Sprintf("%s %s", title, ApplyBoldForegroundGrad("CRUSH UI", t.Secondary, t.Primary))
	
	remainingWidth := width - lipgloss.Width(title) - 1
	if remainingWidth > 0 {
		lines := strings.Repeat("╱", remainingWidth)
		title = fmt.Sprintf("%s %s", title, t.S().Base.Foreground(t.Primary).Render(lines))
	}
	return title
}

// RenderNormalLogo renders the normal logo with optional big mode styling
func RenderNormalLogo(width int, version string, bigMode bool) string {
	t := CurrentTheme()
	
	var logoLines []string
	
	if bigMode {
		// Big mode - larger ASCII art style logo
		logoLines = []string{
			"",
			ApplyBoldForegroundGrad("  ██████╗ ██████╗ ██╗   ██╗███████╗██╗  ██╗", t.Primary, t.Secondary),
			ApplyBoldForegroundGrad(" ██╔════╝ ██╔══██╗██║   ██║██╔════╝██║  ██║", t.Primary, t.Secondary),
			ApplyBoldForegroundGrad(" ██║  ███╗██████╔╝██║   ██║███████╗███████║", t.Primary, t.Secondary),
			ApplyBoldForegroundGrad(" ██║   ██║██╔══██╗██║   ██║╚════██║██╔══██║", t.Primary, t.Secondary),
			ApplyBoldForegroundGrad(" ╚██████╔╝██║  ██║╚██████╔╝███████║██║  ██║", t.Primary, t.Secondary),
			ApplyBoldForegroundGrad("  ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝", t.Primary, t.Secondary),
			"",
			t.S().Base.Foreground(t.Secondary).Render("                    Example™"),
			"",
		}
	} else {
		// Normal mode - text-based logo
		brandLine := t.S().Base.Foreground(t.Secondary).Render("Example™")
		titleLine := ApplyBoldForegroundGrad("CRUSH UI FRAMEWORK", t.Primary, t.Secondary)
		
		logoLines = []string{
			"",
			brandLine,
			titleLine,
			"",
		}
	}
	
	if version != "" {
		versionLine := t.S().Muted.Render(fmt.Sprintf("v%s", version))
		logoLines = append(logoLines, versionLine)
		logoLines = append(logoLines, "")
	}
	
	// Center the logo
	var centeredLines []string
	for _, line := range logoLines {
		lineWidth := lipgloss.Width(line)
		if lineWidth < width {
			padding := (width - lineWidth) / 2
			line = strings.Repeat(" ", padding) + line
		}
		centeredLines = append(centeredLines, line)
	}
	
	return strings.Join(centeredLines, "\n")
}