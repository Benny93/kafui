package core

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// HelpSystem provides an enhanced help system with context-sensitive information
type HelpSystem struct {
	visible     bool
	width       int
	height      int
	currentPage Page
	styles      HelpStyles
}

// HelpStyles contains styling for the help system
type HelpStyles struct {
	Container    lipgloss.Style
	Title        lipgloss.Style
	SectionTitle lipgloss.Style
	KeyBinding   lipgloss.Style
	Description  lipgloss.Style
	Footer       lipgloss.Style
	Separator    lipgloss.Style
}

// HelpSection represents a section in the help display
type HelpSection struct {
	Title    string
	Bindings []HelpBinding
}

// HelpBinding represents a key binding with description
type HelpBinding struct {
	Key         string
	Description string
	Important   bool // Highlight important bindings
}

// NewHelpSystem creates a new help system
func NewHelpSystem() *HelpSystem {
	return &HelpSystem{
		visible: false,
		styles:  createHelpStyles(),
	}
}

// createHelpStyles creates the default help styles
func createHelpStyles() HelpStyles {
	return HelpStyles{
		Container: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Margin(1).
			Background(lipgloss.Color("235")),
		
		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			Align(lipgloss.Center).
			MarginBottom(1),
		
		SectionTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true).
			MarginTop(1).
			MarginBottom(1),
		
		KeyBinding: lipgloss.NewStyle().
			Foreground(lipgloss.Color("228")).
			Bold(true),
		
		Description: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		
		Footer: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true).
			Align(lipgloss.Center).
			MarginTop(1),
		
		Separator: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			MarginTop(1).
			MarginBottom(1),
	}
}

// Toggle toggles the help system visibility
func (h *HelpSystem) Toggle() {
	h.visible = !h.visible
}

// Show shows the help system
func (h *HelpSystem) Show() {
	h.visible = true
}

// Hide hides the help system
func (h *HelpSystem) Hide() {
	h.visible = false
}

// IsVisible returns whether the help system is visible
func (h *HelpSystem) IsVisible() bool {
	return h.visible
}

// SetCurrentPage sets the current page for context-sensitive help
func (h *HelpSystem) SetCurrentPage(page Page) {
	h.currentPage = page
}

// SetDimensions sets the dimensions for the help system
func (h *HelpSystem) SetDimensions(width, height int) {
	h.width = width
	h.height = height
}

// Render renders the help system
func (h *HelpSystem) Render() string {
	if !h.visible {
		return ""
	}
	
	// Calculate container dimensions
	containerWidth := h.width - 4  // Account for margins
	containerHeight := h.height - 4
	
	if containerWidth < 40 || containerHeight < 10 {
		return h.renderCompactHelp()
	}
	
	return h.renderFullHelp(containerWidth, containerHeight)
}

// renderFullHelp renders the full help display
func (h *HelpSystem) renderFullHelp(width, height int) string {
	var content strings.Builder
	
	// Title
	title := "Kafui Help"
	if h.currentPage != nil {
		title = fmt.Sprintf("Kafui Help - %s", h.currentPage.GetTitle())
	}
	content.WriteString(h.styles.Title.Width(width-4).Render(title))
	content.WriteString("\n")
	
	// Collect help sections
	sections := h.collectHelpSections()
	
	// Render sections
	for i, section := range sections {
		if i > 0 {
			content.WriteString(h.styles.Separator.Render("─"))
			content.WriteString("\n")
		}
		
		content.WriteString(h.styles.SectionTitle.Render(section.Title))
		content.WriteString("\n")
		
		for _, binding := range section.Bindings {
			keyStyle := h.styles.KeyBinding
			if binding.Important {
				keyStyle = keyStyle.Background(lipgloss.Color("52"))
			}
			
			line := fmt.Sprintf("  %s  %s",
				keyStyle.Render(binding.Key),
				h.styles.Description.Render(binding.Description))
			content.WriteString(line)
			content.WriteString("\n")
		}
	}
	
	// Footer
	content.WriteString(h.styles.Footer.Width(width-4).Render("Press '?' again to close help"))
	
	// Apply container styling
	return h.styles.Container.
		Width(width).
		Height(height).
		Render(content.String())
}

// renderCompactHelp renders a compact help display for small screens
func (h *HelpSystem) renderCompactHelp() string {
	var content strings.Builder
	
	content.WriteString("Help (?):\n")
	
	// Show only essential bindings in compact mode
	essentialBindings := []HelpBinding{
		{"?", "toggle help", true},
		{"q/ctrl+c", "quit", true},
		{"esc", "back", true},
		{"tab", "next component", false},
		{"shift+tab", "prev component", false},
	}
	
	for _, binding := range essentialBindings {
		line := fmt.Sprintf("%s:%s ", binding.Key, binding.Description)
		content.WriteString(line)
	}
	
	return content.String()
}

// collectHelpSections collects help sections from global and page-specific bindings
func (h *HelpSystem) collectHelpSections() []HelpSection {
	sections := make([]HelpSection, 0)
	
	// Global bindings section
	globalSection := HelpSection{
		Title:    "Global Keys",
		Bindings: h.getGlobalBindings(),
	}
	sections = append(sections, globalSection)
	
	// Page-specific bindings section
	if h.currentPage != nil {
		pageBindings := h.getPageBindings()
		if len(pageBindings) > 0 {
			pageSection := HelpSection{
				Title:    fmt.Sprintf("%s Keys", h.getPageTypeName()),
				Bindings: pageBindings,
			}
			sections = append(sections, pageSection)
		}
	}
	
	// Navigation section
	navSection := HelpSection{
		Title:    "Navigation",
		Bindings: h.getNavigationBindings(),
	}
	sections = append(sections, navSection)
	
	// Focus management section
	focusSection := HelpSection{
		Title:    "Focus Management",
		Bindings: h.getFocusBindings(),
	}
	sections = append(sections, focusSection)
	
	return sections
}

// getGlobalBindings returns global key bindings
func (h *HelpSystem) getGlobalBindings() []HelpBinding {
	return []HelpBinding{
		{"?", "toggle help", true},
		{"q", "quit application", true},
		{"ctrl+c", "quit application", true},
	}
}

// getPageBindings returns page-specific key bindings
func (h *HelpSystem) getPageBindings() []HelpBinding {
	if h.currentPage == nil {
		return []HelpBinding{}
	}
	
	bindings := make([]HelpBinding, 0)
	pageBindings := h.currentPage.GetHelp()
	
	for _, binding := range pageBindings {
		help := binding.Help()
		bindings = append(bindings, HelpBinding{
			Key:         help.Key,
			Description: help.Desc,
			Important:   false,
		})
	}
	
	return bindings
}

// getNavigationBindings returns navigation key bindings
func (h *HelpSystem) getNavigationBindings() []HelpBinding {
	return []HelpBinding{
		{"esc", "go back / exit mode", true},
		{"enter", "select / confirm", false},
		{"↑/k", "move up", false},
		{"↓/j", "move down", false},
		{"←/h", "move left", false},
		{"→/l", "move right", false},
	}
}

// getFocusBindings returns focus management key bindings
func (h *HelpSystem) getFocusBindings() []HelpBinding {
	return []HelpBinding{
		{"tab", "next component", false},
		{"shift+tab", "previous component", false},
	}
}

// getPageTypeName returns a friendly name for the current page type
func (h *HelpSystem) getPageTypeName() string {
	if h.currentPage == nil {
		return "Page"
	}
	
	switch h.currentPage.GetID() {
	case "main":
		return "Main Page"
	case "topic":
		return "Topic Page"
	case "message_detail":
		return "Message Detail"
	case "resource_detail":
		return "Resource Detail"
	default:
		return "Page"
	}
}

// GetKeyBindingHelp returns help text for a specific key binding
func (h *HelpSystem) GetKeyBindingHelp(keyBinding key.Binding) string {
	help := keyBinding.Help()
	return fmt.Sprintf("%s: %s", help.Key, help.Desc)
}

// GetQuickHelp returns a quick help string for display in status bars
func (h *HelpSystem) GetQuickHelp() string {
	quickBindings := []string{
		"? help",
		"q quit",
		"esc back",
	}
	
	if h.currentPage != nil {
		// Add one page-specific binding if available
		pageBindings := h.currentPage.GetHelp()
		if len(pageBindings) > 0 {
			help := pageBindings[0].Help()
			quickBindings = append(quickBindings, fmt.Sprintf("%s %s", help.Key, help.Desc))
		}
	}
	
	return strings.Join(quickBindings, " • ")
}