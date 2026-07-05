package form

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func sampleFields() []Field {
	return []Field{
		{Name: "name", Label: "Name", Type: Text, Required: true},
		{Name: "cleanup", Label: "Cleanup", Type: Select, Options: []string{"delete", "compact"}, Default: "delete"},
		{Name: "internal", Label: "Internal", Type: Bool},
		{Name: "partitions", Label: "Partitions", Type: Numeric, Required: true},
	}
}

func key(s string) tea.KeyMsg {
	switch s {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// typeInto sends each rune of s as a key message to the form.
func typeInto(f *Form, s string) {
	for _, r := range s {
		f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func msgOf(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func TestFocusTraversal(t *testing.T) {
	tests := []struct {
		name    string
		keys    []string
		wantIdx int
	}{
		{"tab moves down", []string{"tab"}, 1},
		{"down moves down", []string{"down"}, 1},
		{"multiple tabs", []string{"tab", "tab", "tab"}, 3},
		{"shift+tab wraps to cancel", []string{"shift+tab"}, 5}, // 4 fields -> submit(4), cancel(5)
		{"up moves to previous", []string{"tab", "tab", "up"}, 1},
		{"tab past fields to submit", []string{"tab", "tab", "tab", "tab"}, 4},
		{"tab to cancel", []string{"tab", "tab", "tab", "tab", "tab"}, 5},
		{"wrap around to first", []string{"tab", "tab", "tab", "tab", "tab", "tab"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := New(sampleFields())
			f.Focus()
			for _, k := range tt.keys {
				f.Update(key(k))
			}
			assert.Equal(t, tt.wantIdx, f.focus)
		})
	}
}

func TestValidationBlocksSubmit(t *testing.T) {
	f := New(sampleFields())
	f.Focus()

	// Move focus to the submit button and press enter with required fields empty.
	f.focus = f.submitIndex()
	cmd, consumed := f.Update(key("enter"))
	assert.True(t, consumed)
	assert.Nil(t, msgOf(cmd), "submit must be blocked while required fields are invalid")

	// Inline errors should be recorded for the required, empty fields.
	assert.Equal(t, "required", f.fields[0].err)
	assert.Equal(t, "required", f.fields[3].err)
}

func TestNumericValidation(t *testing.T) {
	f := New([]Field{{Name: "n", Label: "N", Type: Numeric, Required: true}})
	f.Focus()
	typeInto(f, "abc")

	f.focus = f.submitIndex()
	cmd, _ := f.Update(key("enter"))
	assert.Nil(t, msgOf(cmd))
	assert.Equal(t, "must be a number", f.fields[0].err)
}

func TestCustomValidator(t *testing.T) {
	f := New([]Field{{
		Name:  "x",
		Label: "X",
		Type:  Text,
		Validator: func(v string) error {
			if v != "ok" {
				return fmt.Errorf("must be ok")
			}
			return nil
		},
	}})
	f.Focus()
	typeInto(f, "no")

	f.focus = f.submitIndex()
	cmd, _ := f.Update(key("enter"))
	assert.Nil(t, msgOf(cmd))
	assert.Equal(t, "must be ok", f.fields[0].err)
}

func TestValueCollectionAndSubmit(t *testing.T) {
	f := New(sampleFields())
	f.Focus()

	// Fill the text field.
	typeInto(f, "orders")

	// Cycle the select field to "compact".
	f.Update(key("tab"))
	f.Update(key("right"))

	// Toggle the bool field.
	f.Update(key("tab"))
	f.Update(key(" "))

	// Fill the numeric field.
	f.Update(key("tab"))
	typeInto(f, "12")

	values := f.Values()
	assert.Equal(t, "orders", values["name"])
	assert.Equal(t, "compact", values["cleanup"])
	assert.Equal(t, "true", values["internal"])
	assert.Equal(t, "12", values["partitions"])

	// Now a valid submit emits FormSubmitMsg with the collected values.
	f.focus = f.submitIndex()
	cmd, consumed := f.Update(key("enter"))
	assert.True(t, consumed)
	msg := msgOf(cmd)
	submit, ok := msg.(FormSubmitMsg)
	assert.True(t, ok, "expected FormSubmitMsg, got %T", msg)
	assert.Equal(t, values, submit.Values)
}

func TestCancelEmitsMsg(t *testing.T) {
	f := New(sampleFields())
	f.Focus()
	typeInto(f, "orders")

	cmd, consumed := f.Update(key("esc"))
	assert.True(t, consumed)
	_, ok := msgOf(cmd).(FormCancelMsg)
	assert.True(t, ok, "esc should emit FormCancelMsg")

	// No validation errors recorded (cancel has no side effects on field state).
	for _, fs := range f.fields {
		assert.Empty(t, fs.err)
	}

	// Cancel via the Cancel button.
	f.focus = f.cancelIndex()
	cmd, consumed = f.Update(key("enter"))
	assert.True(t, consumed)
	_, ok = msgOf(cmd).(FormCancelMsg)
	assert.True(t, ok, "Cancel button should emit FormCancelMsg")
}

func TestBoolToggle(t *testing.T) {
	f := New([]Field{{Name: "b", Label: "B", Type: Bool}})
	f.Focus()
	assert.Equal(t, "false", f.Values()["b"])
	f.Update(key(" "))
	assert.Equal(t, "true", f.Values()["b"])
	f.Update(key(" "))
	assert.Equal(t, "false", f.Values()["b"])
}
