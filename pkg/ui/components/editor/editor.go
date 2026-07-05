package editor

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/ui/core"
)

// Editor is an editable text component (schema/config/query text) built on
// bubbles/textarea.
type Editor struct {
	core.BaseComponent

	textarea textarea.Model
}

// NewEditor creates an editor pre-filled with content and ready for input.
func NewEditor(content string) *Editor {
	ta := textarea.New()
	ta.SetValue(content)
	ta.Focus()
	return &Editor{textarea: ta}
}

// SetDimensions sizes the underlying textarea.
func (e *Editor) SetDimensions(width, height int) {
	e.BaseComponent.SetDimensions(width, height)
	if width > 0 {
		e.textarea.SetWidth(width)
	}
	if height > 0 {
		e.textarea.SetHeight(height)
	}
}

// Value returns the current editor text.
func (e *Editor) Value() string { return e.textarea.Value() }

// SetValue replaces the editor text.
func (e *Editor) SetValue(s string) { e.textarea.SetValue(s) }

// Focus focuses the editor for input.
func (e *Editor) Focus() tea.Cmd { return e.textarea.Focus() }

// Blur removes focus from the editor.
func (e *Editor) Blur() { e.textarea.Blur() }

// Init implements core.Component.
func (e *Editor) Init() tea.Cmd { return textarea.Blink }

// Update forwards messages to the textarea.
func (e *Editor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	e.textarea, cmd = e.textarea.Update(msg)
	return e, cmd
}

// View renders the editor.
func (e *Editor) View() string { return e.textarea.View() }
