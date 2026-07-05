package components

import (
	"github.com/charmbracelet/lipgloss"
)

// Breadcrumb represents a breadcrumb navigation component
type Breadcrumb struct {
	width   int
	items   []string
	actions string // page-actions slot rendered right-aligned (UI-11)
}

// SetActions sets the page-actions text rendered on the right of the bar.
func (b *Breadcrumb) SetActions(actions string) {
	b.actions = actions
}

// NewBreadcrumb creates a new breadcrumb component
func NewBreadcrumb() *Breadcrumb {
	return &Breadcrumb{
		items: []string{},
	}
}

// SetItems sets the breadcrumb items
func (b *Breadcrumb) SetItems(items []string) {
	b.items = items
}

// SetWidth sets the component width
func (b *Breadcrumb) SetWidth(width int) {
	b.width = width
}

// HasItems returns true if there are breadcrumb items to display
func (b *Breadcrumb) HasItems() bool {
	return len(b.items) > 0
}

// View renders the breadcrumb component in a status bar style
func (b *Breadcrumb) View() string {
	if len(b.items) == 0 {
		return ""
	}

	// Styles inspired by the user's request
	statusNugget := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFDF5")).
		Padding(0, 1)

	statusBarStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#C1C6B2")).
		Background(lipgloss.Color("#353533")).
		Width(b.width)

	// Item colors (cycling through some of the requested colors)
	styles := []lipgloss.Style{
		statusNugget.Copy().Background(lipgloss.Color("#FF5F87")), // red-ish
		statusNugget.Copy().Background(lipgloss.Color("#A550DF")), // purple
		statusNugget.Copy().Background(lipgloss.Color("#6124DF")), // dark purple
		statusNugget.Copy().Background(lipgloss.Color("#3498DB")), // blue
	}

	var renderedItems []string
	for i, item := range b.items {
		style := styles[i%len(styles)]
		// Last item gets a more prominent bold style if we want, but let's stick to nuggets
		renderedItems = append(renderedItems, style.Render(item))
	}

	// Join items with a bit of space but no separator, or maybe a subtle one
	content := lipgloss.JoinHorizontal(lipgloss.Top, renderedItems...)

	// Page-actions slot (UI-11): right-align advertised page actions on the bar.
	if b.actions != "" {
		actions := statusNugget.Copy().Foreground(lipgloss.Color("#C1C6B2")).Render(b.actions)
		fill := b.width - lipgloss.Width(content) - lipgloss.Width(actions)
		if fill > 0 {
			content = lipgloss.JoinHorizontal(lipgloss.Top, content, lipgloss.NewStyle().Width(fill).Render(""), actions)
		}
	}

	return statusBarStyle.Render(content)
}
