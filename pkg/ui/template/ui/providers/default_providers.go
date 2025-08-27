package providers

import (
	"strings"
	"time"
	"ui_example/ui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DefaultContentProvider provides the original example content
type DefaultContentProvider struct{}

func NewDefaultContentProvider() *DefaultContentProvider {
	return &DefaultContentProvider{}
}

func (d *DefaultContentProvider) RenderContent(width, height int) string {
	t := styles.CurrentTheme()
	
	// Calculate available space for content
	availableWidth := width - 6  // Account for border and padding
	availableHeight := height - 6 // Account for border and padding
	
	if availableWidth <= 0 || availableHeight <= 0 {
		return ""
	}
	
	// Determine size mode for adaptive content
	sizeMode := styles.GetSizeMode(width, height)
	
	// Create the main content sections based on size mode
	var sections []string
	
	// Logo section (varies by size mode)
	logo := styles.RenderLogo(availableWidth, sizeMode, "1.0.0")
	if logo != "" {
		sections = append(sections, logo)
		sections = append(sections, "")
	}
	
	// Description with CRUSH styling (adaptive based on size)
	var description []string
	if sizeMode >= styles.SizeModeCompact {
		description = []string{
			"This example demonstrates the CRUSH CLI design patterns:",
			"",
			"✓ Multiple size modes (minimum/small/compact/normal/big)",
			"✓ Fixed 31-character sidebar width",
			"✓ Rounded borders and clean sections", 
			"✓ Files, Servers, and Status sections",
			"✓ Responsive layout with breakpoints",
			"✓ Real-time data updates",
			"✓ Beautiful gradient text effects",
			"✓ Debug information (Ctrl+D to toggle)",
			"",
			"The sidebar shows live data that updates every 5 seconds.",
			"Resize the window to see different size modes in action!",
		}
	} else {
		// Compact description for small screens
		description = []string{
			"CRUSH UI Framework",
			"",
			"✓ Responsive design",
			"✓ Multiple size modes", 
			"✓ Real-time updates",
			"",
			"Resize window to see modes!",
		}
	}
	
	for _, line := range description {
		if strings.HasPrefix(line, "✓") {
			sections = append(sections, t.S().Success.Render(line))
		} else if line == "" {
			sections = append(sections, "")
		} else {
			sections = append(sections, t.S().Text.Render(line))
		}
	}
	
	sections = append(sections, "")
	
	// Controls section with CRUSH styling (adaptive)
	if sizeMode >= styles.SizeModeCompact {
		controlsTitle := styles.Section("Controls", availableWidth)
		sections = append(sections, controlsTitle)
		sections = append(sections, "")
		
		controls := []string{
			"T / Ctrl+S  Toggle sidebar (normal+ mode)",
			"Ctrl+R      Refresh data",
			"Ctrl+D      Toggle debug info",
			"Q / Ctrl+C  Quit application",
		}
		
		for _, control := range controls {
			parts := strings.SplitN(control, "  ", 2)
			if len(parts) == 2 {
				keyStyle := t.S().Base.Foreground(t.Accent).Bold(true)
				descStyle := t.S().Muted
				line := keyStyle.Render(parts[0]) + "  " + descStyle.Render(parts[1])
				sections = append(sections, line)
			} else {
				sections = append(sections, t.S().Text.Render(control))
			}
		}
	} else {
		// Minimal controls for small screens
		sections = append(sections, t.S().Muted.Render("Q to quit • Ctrl+D for debug"))
	}
	
	// Join all sections
	content := strings.Join(sections, "\n")
	
	// Center the content vertically
	contentLines := strings.Split(content, "\n")
	if len(contentLines) < availableHeight {
		paddingTop := (availableHeight - len(contentLines)) / 2
		for i := 0; i < paddingTop; i++ {
			contentLines = append([]string{""}, contentLines...)
		}
	}
	
	// Center each line horizontally
	var centeredLines []string
	for _, line := range contentLines {
		lineWidth := lipgloss.Width(line)
		if lineWidth < availableWidth {
			padding := (availableWidth - lineWidth) / 2
			line = strings.Repeat(" ", padding) + line
		}
		centeredLines = append(centeredLines, line)
	}
	
	return strings.Join(centeredLines, "\n")
}

func (d *DefaultContentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd {
	return nil
}

func (d *DefaultContentProvider) InitContent() tea.Cmd {
	return nil
}

// DefaultHeaderDataProvider provides the original header data
type DefaultHeaderDataProvider struct {
	lastUpdate time.Time
}

func NewDefaultHeaderDataProvider() *DefaultHeaderDataProvider {
	return &DefaultHeaderDataProvider{
		lastUpdate: time.Now(),
	}
}

func (d *DefaultHeaderDataProvider) GetBrandName() string {
	return "Example™"
}

func (d *DefaultHeaderDataProvider) GetAppName() string {
	return "CRUSH UI"
}

func (d *DefaultHeaderDataProvider) GetStatusData() map[string]interface{} {
	return map[string]interface{}{
		"time":        d.lastUpdate.Format("15:04:05"),
		"status":      "online",
		"connections": 3,
		"memory":      "45%",
	}
}

func (d *DefaultHeaderDataProvider) HandleHeaderUpdate(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tickMsg:
		d.lastUpdate = time.Time(msg)
		return tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	}
	return nil
}

func (d *DefaultHeaderDataProvider) InitHeader() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type tickMsg time.Time