package components

import (
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/Benny93/kafui/pkg/ui/template/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func (c *content) Init() tea.Cmd {
	if c.provider != nil {
		return c.provider.InitContent()
	}
	return nil
}

func (c *content) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle content-specific key events here
		shared.DebugLog("Content.Update: Received KeyMsg: %s", msg.String())
		_ = msg
	}

	// Let the provider handle the message
	if c.provider != nil {
		shared.DebugLog("Content.Update: Delegating message %T to provider", msg)
		cmd = c.provider.HandleContentUpdate(msg)
		if cmd != nil {
			shared.DebugLog("Content.Update: Provider returned a command")
		} else {
			shared.DebugLog("Content.Update: Provider returned nil command")
		}
	} else {
		shared.DebugLog("Content.Update: No provider available")
	}

	return c, cmd
}

func (c *content) View() string {
	if c.width == 0 || c.height == 0 {
		return ""
	}

	t := styles.CurrentTheme()

	// Get content from provider
	var content string
	if c.provider != nil {
		content = c.provider.RenderContent(c.width, c.height)
	}

	// Add debug info at the bottom
	debugInfo := styles.DebugInfo("Content", c.width, c.height)
	if content != "" && debugInfo != "" {
		content = content + "\n\n" + debugInfo
	} else if debugInfo != "" {
		content = debugInfo
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
