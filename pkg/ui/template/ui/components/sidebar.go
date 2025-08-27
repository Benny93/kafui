package components

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/Benny93/kafui/pkg/ui/template/ui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Sidebar interface {
	Component
	Sizeable
	Focusable
	CompactModeToggleable
}

type sidebar struct {
	width, height int
	focused       bool
	compact       bool

	// Provider-based sections
	sections []providers.SidebarSection
}

func NewSidebar() Sidebar {
	return &sidebar{
		sections: []providers.SidebarSection{
			providers.NewFilesSection(),
			providers.NewServersSection(),
			providers.NewStatusSection(),
		},
	}
}

func NewSidebarWithSections(sections []providers.SidebarSection) Sidebar {
	return &sidebar{
		sections: sections,
	}
}

func (s *sidebar) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, section := range s.sections {
		if cmd := section.InitSection(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (s *sidebar) Refresh() tea.Cmd {
	var cmds []tea.Cmd
	for _, section := range s.sections {
		if cmd := section.RefreshSection(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (s *sidebar) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle sidebar-specific key events here
		_ = msg
	}

	// Let all sections handle the message
	for _, section := range s.sections {
		if cmd := section.HandleSectionUpdate(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return s, tea.Batch(cmds...)
}

func (s *sidebar) View() string {
	if s.width == 0 || s.height == 0 {
		return ""
	}

	t := styles.CurrentTheme()

	// CRUSH-style sidebar with rounded border
	borderStyle := t.S().Base.
		Width(s.width).
		Height(s.height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(1)

	// Build sidebar content
	content := s.renderSidebarContent()

	// Add debug info at the bottom
	debugInfo := styles.DebugInfo("Sidebar", s.width, s.height)
	if content != "" && debugInfo != "" {
		content = content + "\n\n" + debugInfo
	} else if debugInfo != "" {
		content = debugInfo
	}

	return borderStyle.Render(content)
}

func (s *sidebar) renderSidebarContent() string {
	availableWidth := s.width - 4   // Account for border and padding
	availableHeight := s.height - 2 // Account for border

	if availableWidth <= 0 || availableHeight <= 0 {
		return ""
	}

	var sections []string

	// Logo section (CRUSH style)
	if !s.compact || s.height >= 30 {
		logo := s.renderLogo(availableWidth)
		sections = append(sections, logo)
		sections = append(sections, "")
	}

	// Calculate remaining height for sections
	remainingHeight := availableHeight - len(sections)

	// Determine how many items to show per section based on available space
	numSections := len(s.sections)
	if numSections == 0 {
		return strings.Join(sections, "\n")
	}

	maxItemsPerSection := s.calculateMaxItems(remainingHeight, numSections)

	// Render all sections
	for i, section := range s.sections {
		sectionContent := s.renderSection(section, maxItemsPerSection, availableWidth)
		sections = append(sections, sectionContent)

		// Add spacing between sections (except after the last one)
		if i < len(s.sections)-1 {
			sections = append(sections, "")
		}
	}

	return strings.Join(sections, "\n")
}

func (s *sidebar) renderLogo(width int) string {
	t := styles.CurrentTheme()
	logo := styles.ApplyBoldForegroundGrad("CRUSH", t.Primary, t.Secondary)
	version := t.S().Muted.Render("v1.0.0")

	// Center the logo
	logoWidth := lipgloss.Width(logo)
	if logoWidth < width {
		padding := (width - logoWidth) / 2
		logo = strings.Repeat(" ", padding) + logo
	}

	return lipgloss.JoinVertical(lipgloss.Left, logo, version)
}

func (s *sidebar) calculateMaxItems(availableHeight, numSections int) int {
	// Reserve space for section headers and spacing
	headerSpace := numSections * 2 // Each section has a header + spacing
	itemSpace := availableHeight - headerSpace

	if itemSpace <= 0 {
		return 1
	}

	maxPerSection := itemSpace / numSections
	if maxPerSection < 2 {
		return 2
	}

	return maxPerSection
}

func (s *sidebar) renderSection(section providers.SidebarSection, maxItems, width int) string {
	t := styles.CurrentTheme()
	var lines []string

	// Section header
	title := section.GetTitle()
	header := styles.Section(title, width)
	lines = append(lines, header)

	// Get items from the section
	items := section.RenderItems(maxItems, width)

	// Render each item
	for _, item := range items {
		statusStyle := s.getItemStatusStyle(item.Status)

		// Format: [icon] text (value)
		line := fmt.Sprintf("%s %s", statusStyle.Render(item.Icon), item.Text)

		// Add value if there's space and it's not empty
		if item.Value != "" {
			valueText := t.S().Muted.Render(fmt.Sprintf("(%s)", item.Value))
			totalWidth := lipgloss.Width(line) + lipgloss.Width(valueText)
			if totalWidth <= width {
				spacing := width - totalWidth
				line = line + strings.Repeat(" ", spacing) + valueText
			}
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (s *sidebar) getItemStatusStyle(status string) lipgloss.Style {
	t := styles.CurrentTheme()

	switch status {
	case "success":
		return t.S().Success
	case "error":
		return t.S().Error
	case "warning":
		return t.S().Warning
	case "info":
		return t.S().Info
	case "muted":
		return t.S().Muted
	default:
		return t.S().Text
	}
}

func (s *sidebar) SetSize(width, height int) tea.Cmd {
	s.width = width
	s.height = height
	return nil
}

func (s *sidebar) GetSize() (int, int) {
	return s.width, s.height
}

func (s *sidebar) Focus() tea.Cmd {
	s.focused = true
	return nil
}

func (s *sidebar) Blur() tea.Cmd {
	s.focused = false
	return nil
}

func (s *sidebar) IsFocused() bool {
	return s.focused
}

func (s *sidebar) SetCompactMode(compact bool) tea.Cmd {
	s.compact = compact
	return nil
}
