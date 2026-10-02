package form

import (
	"fmt"
	"testing"

	"github.com/Benny93/kafui/pkg/appconfig"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// A Password field's Default is a stored credential, so it must never be painted
// as plaintext; typed content renders masked but still submits in full.
func TestPasswordRendersMasked(t *testing.T) {
	f := New([]Field{{Name: "pw", Label: "Password", Type: Password, Default: "hunter2"}})
	f.Focus()

	out := f.View()
	assert.NotContains(t, out, "hunter2", "the stored secret must not be rendered")
	assert.Contains(t, out, appconfig.RedactPlaceholder(), "a stored secret is signalled by the mask marker")
	assert.Empty(t, f.Values()["pw"], "an untouched Password field submits empty (= keep current)")

	// Typing renders the mask rather than the text, while the submitted value
	// stays the real credential.
	typeInto(f, "abc")
	assert.NotContains(t, f.View(), "abc", "typed secret must render masked")
	assert.Contains(t, f.View(), "*")
	assert.Equal(t, "abc", f.Values()["pw"])
}

// An externalized ${env:VAR} reference is a pointer, not secret material: like
// appconfig.Redactor, it must display and round-trip verbatim.
func TestPasswordProviderRefRoundTrips(t *testing.T) {
	f := New([]Field{{Name: "pw", Label: "Password", Type: Password, Default: "${env:KAFUI_PW}"}})
	f.Focus()

	assert.Contains(t, f.View(), "${env:KAFUI_PW}", "a reference is safe to display")
	assert.Equal(t, "${env:KAFUI_PW}", f.Values()["pw"], "editing must not drop the reference")
	// It is not a stored secret, so there is nothing to preserve.
	assert.False(t, f.fields[0].secretSet)
}

// Masking follows content: only a complete ${provider:...} reference is rendered
// legibly, so a half-typed reference or raw secret material never leaks.
func TestPasswordMaskFollowsContent(t *testing.T) {
	f := New([]Field{{Name: "pw", Label: "Password", Type: Password}})
	f.Focus()

	typeInto(f, "${env:KAF")
	assert.NotContains(t, f.View(), "${env:KAF", "an incomplete reference stays masked")

	typeInto(f, "UI_PW}")
	assert.Contains(t, f.View(), "${env:KAFUI_PW}", "a complete reference is legible")

	// Clearing it and typing real secret material re-masks immediately.
	for i := 0; i < len("${env:KAFUI_PW}"); i++ {
		f.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	require.Empty(t, f.Values()["pw"])

	typeInto(f, "hunter2")
	assert.NotContains(t, f.View(), "hunter2", "raw secret material is masked")
	assert.Equal(t, "hunter2", f.Values()["pw"])
}

// An untouched Password field preserving a stored secret submits empty, which
// must not be rejected as missing input.
func TestPasswordRequiredWithStoredSecretSubmits(t *testing.T) {
	f := New([]Field{{Name: "pw", Label: "Password", Type: Password, Required: true, Default: "hunter2"}})
	f.Focus()

	f.focus = f.submitIndex()
	cmd, consumed := f.Update(key("enter"))
	require.True(t, consumed)
	_, ok := msgOf(cmd).(FormSubmitMsg)
	assert.True(t, ok, "a preserved stored secret satisfies 'required'")
	assert.Empty(t, f.fields[0].err)
}

// A Password field with no stored secret is an ordinary empty required input.
func TestPasswordRequiredWithoutStoredSecretBlocks(t *testing.T) {
	f := New([]Field{{Name: "pw", Label: "Password", Type: Password, Required: true}})
	f.Focus()

	f.focus = f.submitIndex()
	cmd, _ := f.Update(key("enter"))
	assert.Nil(t, msgOf(cmd), "submit must be blocked with nothing stored")
	assert.Equal(t, "required", f.fields[0].err)
}

// Password must be treated as textual everywhere Text and Numeric are: focus
// traversal and typing both have to reach the input.
func TestPasswordIsTextualForTraversal(t *testing.T) {
	f := New([]Field{
		{Name: "a", Label: "A", Type: Text},
		{Name: "pw", Label: "Password", Type: Password},
	})
	f.Focus()
	f.SetDimensions(60, 20)

	f.Update(key("tab"))
	assert.Equal(t, 1, f.focus)
	assert.True(t, f.fields[1].input.Focused(), "the Password input must take focus")
	assert.Equal(t, f.fields[0].input.Width, f.fields[1].input.Width, "sized like a text field")

	typeInto(f, "x")
	assert.Equal(t, "x", f.Values()["pw"])
}
