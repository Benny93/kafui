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
	// IsInputMode returns true when the content's provider has an active text input.
	// ReusableApp uses this to suppress hotkeys that would steal keystrokes from the input.
	IsInputMode() bool
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

	// Calculate capped width for content readability. Reserve one column for the
	// optional vertical scrollbar (added below when content overflows) so
	// full-width content — e.g. a framed table — doesn't overflow the pane and
	// wrap its border when the scrollbar appears.
	contentWidth := cappedContentWidth(c.width) - 1

	// Get content from provider with capped width
	// Pass inner content dimensions (excluding border+padding overhead) so providers
	// can set table page sizes accurately without guessing the chrome.
	// Inner height = outer height - border(2) - padding(2) = height - 4
	innerHeight := c.height - 4
	if innerHeight < 1 {
		innerHeight = 1
	}
	var content string
	if c.provider != nil {
		content = c.provider.RenderContent(contentWidth, innerHeight)
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

func (c *content) IsInputMode() bool {
	if c.provider != nil {
		return c.provider.IsInputMode()
	}
	return false
}
