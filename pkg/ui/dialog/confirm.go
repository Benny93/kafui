// Package dialog provides the root-owned modal confirmation overlay used for
// destructive-action confirmation across all pages.
package dialog

import (
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Confirm is a modal yes/no dialog. While active it traps all key input; the
// root model renders it centered over the current page and routes keys to it.
type Confirm struct {
	styles *styles.Styles

	active       bool
	busy         bool
	title        string
	message      string
	confirmLabel string
	danger       bool
	onConfirm    tea.Cmd
	focusConfirm bool // true when the Confirm button has focus (default: Cancel)

	width  int
	height int
}

// New creates a Confirm dialog bound to the given styles.
func New(s *styles.Styles) *Confirm {
	return &Confirm{styles: s, confirmLabel: "Confirm"}
}

// Active reports whether the dialog is currently shown.
func (c *Confirm) Active() bool { return c.active }

// SetDimensions records the terminal size for centering.
func (c *Confirm) SetDimensions(w, h int) { c.width, c.height = w, h }

// Show opens the dialog from a ShowConfirmMsg.
func (c *Confirm) Show(msg core.ShowConfirmMsg) {
	c.active = true
	c.busy = false
	c.title = msg.Title
	c.message = msg.Message
	c.danger = msg.Danger
	c.onConfirm = msg.OnConfirm
	c.confirmLabel = msg.ConfirmLabel
	if c.confirmLabel == "" {
		c.confirmLabel = "Confirm"
	}
	// Default focus to Cancel for safety on destructive actions.
	c.focusConfirm = false
}

func (c *Confirm) close() {
	c.active = false
	c.busy = false
	c.onConfirm = nil
}

// Update handles a message while the dialog is active. It returns a command and
// whether the message was consumed by the dialog (callers should not propagate
// consumed messages to the page underneath).
//
// ponytail: on confirm the dialog dispatches OnConfirm and closes immediately;
// the operation reports its own progress/result via notifications (UI-2) rather
// than holding the modal open with a busy spinner. Upgrade to a persistent busy
// state if an action needs to block interaction until it completes.
func (c *Confirm) Update(msg tea.Msg) (tea.Cmd, bool) {
	if !c.active {
		return nil, false
	}
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(m, keyCancel):
			c.close()
			return resolved(false), true
		case key.Matches(m, keyToggle):
			c.focusConfirm = !c.focusConfirm
			return nil, true
		case key.Matches(m, keyConfirmKey):
			confirmed := c.focusConfirm
			cmd := c.onConfirm
			c.close()
			if confirmed {
				return tea.Batch(cmd, resolved(true)), true
			}
			return resolved(false), true
		}
		// Swallow every other key while modal is open.
		return nil, true
	}
	return nil, false
}

func resolved(v bool) tea.Cmd {
	return func() tea.Msg { return core.ConfirmResolvedMsg{Confirmed: v} }
}

// View renders the dialog centered over the given background.
func (c *Confirm) View(background string) string {
	if !c.active {
		return background
	}
	box := c.renderBox()
	w, h := c.width, c.height
	if w <= 0 || h <= 0 {
		w, h = lipgloss.Width(background), lipgloss.Height(background)
	}
	// ponytail: center on a cleared canvas rather than compositing over the page;
	// the page-behind effect needs ANSI-aware line overlay — add if desired.
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}

func (c *Confirm) renderBox() string {
	accent := styles.Primary
	if c.danger {
		accent = styles.Error
	}
	title := lipgloss.NewStyle().Foreground(accent).Bold(true).Render(c.title)
	body := lipgloss.NewStyle().Foreground(styles.FgBase).Render(c.message)

	cancelBtn := c.button("Cancel", !c.focusConfirm, styles.FgMuted)
	confirmBtn := c.button(c.confirmLabel, c.focusConfirm, accent)
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, cancelBtn, "  ", confirmBtn)

	content := lipgloss.JoinVertical(lipgloss.Left, title, "", body, "", buttons)
	return lipgloss.NewStyle().
		Padding(1, 3).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Render(content)
}

func (c *Confirm) button(label string, focused bool, accent lipgloss.Color) string {
	s := lipgloss.NewStyle().Padding(0, 2)
	if focused {
		return s.Foreground(styles.BgBase).Background(accent).Bold(true).Render(label)
	}
	return s.Foreground(styles.FgMuted).Border(lipgloss.NormalBorder(), false).Render(label)
}

var (
	keyCancel     = key.NewBinding(key.WithKeys("esc"))
	keyToggle     = key.NewBinding(key.WithKeys("left", "right", "tab", "shift+tab", "h", "l"))
	keyConfirmKey = key.NewBinding(key.WithKeys("enter"))
)
