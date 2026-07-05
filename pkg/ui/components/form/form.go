// Package form provides a reusable, typed form model for create/edit flows.
//
// A Form is composed of typed fields (text, select/enum, bool toggle, numeric)
// with vertical focus traversal, per-field validation and Submit/Cancel actions.
// It renders as a plain string for a hosting page's content area (not a modal),
// and emits FormSubmitMsg / FormCancelMsg on the respective actions.
package form

import (
	"fmt"
	"strconv"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FieldType enumerates the supported field kinds.
type FieldType int

const (
	// Text is a free-form text input.
	Text FieldType = iota
	// Select cycles through a fixed set of Options.
	Select
	// Bool is a true/false toggle.
	Bool
	// Numeric is a text input constrained to numeric values.
	Numeric
)

// Validator validates a raw field value, returning a non-nil error to reject it.
type Validator func(value string) error

// Field describes a single form field.
type Field struct {
	Name      string    // key used in the submitted Values map
	Label     string    // human-readable label
	Type      FieldType // field kind
	Required  bool      // whether an empty value is rejected
	Options   []string  // choices for Select fields
	Validator Validator // optional custom validation
	Default   string    // initial value (text/numeric) or default option
}

// FormSubmitMsg is emitted when the form is submitted with all fields valid.
type FormSubmitMsg struct {
	Values map[string]string
}

// FormCancelMsg is emitted when the form is cancelled.
type FormCancelMsg struct{}

type fieldState struct {
	def      Field
	input    textinput.Model // used for Text and Numeric
	selected int             // used for Select (index into Options)
	boolVal  bool            // used for Bool
	err      string          // current inline validation error
}

func (fs *fieldState) value() string {
	switch fs.def.Type {
	case Select:
		if len(fs.def.Options) == 0 {
			return ""
		}
		return fs.def.Options[fs.selected]
	case Bool:
		return strconv.FormatBool(fs.boolVal)
	default:
		return fs.input.Value()
	}
}

// Form is a reusable form component.
type Form struct {
	core.BaseComponent

	fields []*fieldState
	focus  int // 0..len(fields)-1 = fields; len = Submit; len+1 = Cancel
}

// New creates a Form from the given field definitions.
func New(fields []Field) *Form {
	f := &Form{}
	for _, def := range fields {
		fs := &fieldState{def: def}
		switch def.Type {
		case Text, Numeric:
			ti := textinput.New()
			ti.SetValue(def.Default)
			ti.Width = 40
			fs.input = ti
		case Select:
			for i, o := range def.Options {
				if o == def.Default {
					fs.selected = i
				}
			}
		case Bool:
			fs.boolVal = def.Default == "true"
		}
		f.fields = append(f.fields, fs)
	}
	return f
}

// Init implements the Bubble Tea model contract.
func (f *Form) Init() tea.Cmd {
	return textinput.Blink
}

// Focus focuses the form's first field.
func (f *Form) Focus() tea.Cmd {
	f.focus = 0
	return f.syncFocus()
}

// SetDimensions sets the component dimensions and adjusts input widths.
func (f *Form) SetDimensions(width, height int) {
	f.BaseComponent.SetDimensions(width, height)
	w := width - 4
	if w < 10 {
		w = 10
	}
	for _, fs := range f.fields {
		if fs.def.Type == Text || fs.def.Type == Numeric {
			fs.input.Width = w
		}
	}
}

// submitIndex / cancelIndex are the pseudo focus positions for the buttons.
func (f *Form) submitIndex() int { return len(f.fields) }
func (f *Form) cancelIndex() int { return len(f.fields) + 1 }

// syncFocus focuses the textinput of the currently focused field (if any) and
// blurs the rest, returning the textinput focus command.
func (f *Form) syncFocus() tea.Cmd {
	var cmd tea.Cmd
	for i, fs := range f.fields {
		if fs.def.Type != Text && fs.def.Type != Numeric {
			continue
		}
		if i == f.focus {
			cmd = fs.input.Focus()
		} else {
			fs.input.Blur()
		}
	}
	return cmd
}

func (f *Form) moveFocus(delta int) tea.Cmd {
	n := len(f.fields) + 2 // fields + submit + cancel
	f.focus = (f.focus + delta + n) % n
	return f.syncFocus()
}

// Update handles a message and reports whether it was consumed.
func (f *Form) Update(msg tea.Msg) (tea.Cmd, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		// Forward non-key messages (e.g. blink) to the focused text input.
		return f.updateFocusedInput(msg), false
	}

	switch key.String() {
	case "esc":
		return func() tea.Msg { return FormCancelMsg{} }, true
	case "tab", "down":
		return f.moveFocus(1), true
	case "shift+tab", "up":
		return f.moveFocus(-1), true
	case "enter":
		return f.handleEnter(), true
	}

	// Field-specific input.
	if f.focus < len(f.fields) {
		fs := f.fields[f.focus]
		switch fs.def.Type {
		case Select:
			return f.handleSelectKey(fs, key), true
		case Bool:
			if key.String() == " " || key.String() == "left" || key.String() == "right" {
				fs.boolVal = !fs.boolVal
				return nil, true
			}
		case Text, Numeric:
			var cmd tea.Cmd
			fs.input, cmd = fs.input.Update(msg)
			return cmd, true
		}
	}
	return nil, false
}

func (f *Form) handleSelectKey(fs *fieldState, key tea.KeyMsg) tea.Cmd {
	if len(fs.def.Options) == 0 {
		return nil
	}
	switch key.String() {
	case "right", " ":
		fs.selected = (fs.selected + 1) % len(fs.def.Options)
	case "left":
		fs.selected = (fs.selected - 1 + len(fs.def.Options)) % len(fs.def.Options)
	}
	return nil
}

func (f *Form) handleEnter() tea.Cmd {
	switch f.focus {
	case f.submitIndex():
		return f.submit()
	case f.cancelIndex():
		return func() tea.Msg { return FormCancelMsg{} }
	default:
		return f.moveFocus(1)
	}
}

func (f *Form) updateFocusedInput(msg tea.Msg) tea.Cmd {
	if f.focus < len(f.fields) {
		fs := f.fields[f.focus]
		if fs.def.Type == Text || fs.def.Type == Numeric {
			var cmd tea.Cmd
			fs.input, cmd = fs.input.Update(msg)
			return cmd
		}
	}
	return nil
}

// submit validates all fields; if valid it emits FormSubmitMsg, otherwise it
// records inline errors and blocks submission (returns nil).
func (f *Form) submit() tea.Cmd {
	if !f.Validate() {
		return nil
	}
	values := f.Values()
	return func() tea.Msg { return FormSubmitMsg{Values: values} }
}

// Validate runs validation on every field, recording inline errors, and reports
// whether the whole form is valid.
func (f *Form) Validate() bool {
	valid := true
	for _, fs := range f.fields {
		if err := validateField(fs); err != nil {
			fs.err = err.Error()
			valid = false
		} else {
			fs.err = ""
		}
	}
	return valid
}

func validateField(fs *fieldState) error {
	v := fs.value()
	if fs.def.Required && v == "" {
		return fmt.Errorf("required")
	}
	if fs.def.Type == Numeric && v != "" {
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			return fmt.Errorf("must be a number")
		}
	}
	if fs.def.Validator != nil {
		if err := fs.def.Validator(v); err != nil {
			return err
		}
	}
	return nil
}

// Values returns the current field values keyed by field name.
func (f *Form) Values() map[string]string {
	out := make(map[string]string, len(f.fields))
	for _, fs := range f.fields {
		out[fs.def.Name] = fs.value()
	}
	return out
}

// View renders the form as a plain string for the content area.
func (f *Form) View() string {
	labelStyle := lipgloss.NewStyle().Foreground(styles.FgBase).Bold(true)
	focusedLabel := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	markerStyle := lipgloss.NewStyle().Foreground(styles.Error)
	valueStyle := lipgloss.NewStyle().Foreground(styles.FgBase)
	mutedStyle := lipgloss.NewStyle().Foreground(styles.FgMuted)
	errStyle := lipgloss.NewStyle().Foreground(styles.Error)

	var rows []string
	for i, fs := range f.fields {
		ls := labelStyle
		if i == f.focus {
			ls = focusedLabel
		}
		label := ls.Render(fs.def.Label)
		if fs.def.Required {
			label += markerStyle.Render(" *")
		}
		rows = append(rows, label)
		rows = append(rows, "  "+f.renderFieldValue(fs, i == f.focus, valueStyle, mutedStyle))
		if fs.err != "" {
			rows = append(rows, "  "+errStyle.Render(fs.err))
		}
		rows = append(rows, "")
	}

	rows = append(rows, f.renderButtons())
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (f *Form) renderFieldValue(fs *fieldState, focused bool, valueStyle, mutedStyle lipgloss.Style) string {
	switch fs.def.Type {
	case Select:
		v := fs.value()
		if v == "" {
			v = "(no options)"
		}
		s := valueStyle.Render("< " + v + " >")
		if focused {
			s += mutedStyle.Render("  (←/→ to change)")
		}
		return s
	case Bool:
		box := "[ ]"
		if fs.boolVal {
			box = "[x]"
		}
		s := valueStyle.Render(box)
		if focused {
			s += mutedStyle.Render("  (space to toggle)")
		}
		return s
	default:
		return fs.input.View()
	}
}

func (f *Form) renderButtons() string {
	base := lipgloss.NewStyle().Foreground(styles.FgBase).Padding(0, 2).Border(lipgloss.NormalBorder())
	active := lipgloss.NewStyle().Foreground(styles.FgBase).Background(styles.Primary).Bold(true).Padding(0, 2).Border(lipgloss.NormalBorder()).BorderForeground(styles.Primary)

	submit := base
	cancel := base
	if f.focus == f.submitIndex() {
		submit = active
	}
	if f.focus == f.cancelIndex() {
		cancel = active
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, submit.Render("Submit"), " ", cancel.Render("Cancel"))
}
