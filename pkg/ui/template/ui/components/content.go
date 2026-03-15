package components

import (
	"strings"

	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/Benny93/kafui/pkg/ui/template/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	// ContentLeftPadding is the total horizontal padding (border + inner padding)
	ContentLeftPadding = 4
	// MaxContentWidth is the maximum width for content readability (matches CRUSH standard)
	// Increased for Kafui to better support wide message tables
	MaxContentWidth = 240
)

type Content interface {
	Component
	Sizeable
	Focusable
}

type content struct {
	width, height int
	focused       bool
	provider      providers.ContentProvider
	scrollOffset  int // Scroll offset for overflow content
}

func NewContent() Content {
	return &content{
		provider: providers.NewDefaultContentProvider(),
	}
}

func NewContentWithProvider(provider providers.ContentProvider) Content {
	return &content{
		provider: provider,
	}
}

// cappedContentWidth calculates the maximum content width for readability and layout stability
func cappedContentWidth(availableWidth int) int {
	return min(availableWidth-ContentLeftPadding, MaxContentWidth)
}

func (c *content) Init() tea.Cmd {
	if c.provider != nil {
		return c.provider.InitContent()
	}
	return nil
}

func (c *content) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmd tea.Cmd

	// Let the provider handle the message
	if c.provider != nil {
		cmd = c.provider.HandleContentUpdate(msg)
	}

	return c, cmd
}

func (c *content) View() string {
	if c.width == 0 || c.height == 0 {
		return ""
	}

	t := styles.CurrentTheme()

	// Calculate capped width for content readability
	contentWidth := cappedContentWidth(c.width)

	// Get content from provider with capped width
	var content string
	if c.provider != nil {
		content = c.provider.RenderContent(contentWidth, c.height)
	}

	// Check if content needs scrollbar
	// Account for border (2) and padding (1 top + 1 bottom = 2) when calculating available viewport height
	viewportHeight := c.height - 4 // Border (2) + Padding (2)
	contentLines := strings.Split(content, "\n")
	contentHeight := len(contentLines)
	needsScrollbar := contentHeight > viewportHeight

	// Add scrollbar if needed
	if needsScrollbar {
		scrollbar := Scrollbar(viewportHeight, contentHeight, viewportHeight, c.scrollOffset)
		if scrollbar != "" {
			// Join content and scrollbar horizontally
			content = lipgloss.JoinHorizontal(lipgloss.Top, content, scrollbar)
		}
	}

	// Apply styling based on focus state
	var style lipgloss.Style
	if c.focused {
		style = t.S().Base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderFocus)
	} else {
		style = t.S().Base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border)
	}

	// Size and render the content area
	return style.
		Width(c.width - 2).   // Account for border
		Height(c.height - 2). // Account for border
		MaxHeight(c.height - 2).
		Align(lipgloss.Top, lipgloss.Left).
		Padding(1).
		Render(content)
}

func (c *content) SetSize(width, height int) tea.Cmd {
	c.width = width
	c.height = height
	return nil
}

func (c *content) GetSize() (int, int) {
	return c.width, c.height
}

func (c *content) Focus() tea.Cmd {
	c.focused = true
	return nil
}

func (c *content) Blur() tea.Cmd {
	c.focused = false
	return nil
}

func (c *content) IsFocused() bool {
	return c.focused
}
