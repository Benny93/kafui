package dialog

import (
	"testing"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mkKey(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestConfirmCancelByDefault(t *testing.T) {
	d := New(styles.DefaultStyles())
	var ran bool
	d.Show(core.ShowConfirmMsg{Title: "Delete", OnConfirm: func() tea.Msg { ran = true; return nil }})
	require.True(t, d.Active())

	// Enter with default focus (Cancel) resolves false and runs nothing.
	cmd, consumed := d.Update(mkKey("enter"))
	assert.True(t, consumed)
	assert.False(t, d.Active())
	msg := cmd()
	assert.Equal(t, core.ConfirmResolvedMsg{Confirmed: false}, msg)
	assert.False(t, ran)
}

func TestConfirmAfterToggle(t *testing.T) {
	d := New(styles.DefaultStyles())
	ran := false
	d.Show(core.ShowConfirmMsg{Title: "Delete", Danger: true, OnConfirm: func() tea.Msg { ran = true; return core.StatusMsg{} }})

	// Move focus to Confirm, then enter.
	_, _ = d.Update(mkKey("tab"))
	cmd, consumed := d.Update(mkKey("enter"))
	assert.True(t, consumed)
	assert.False(t, d.Active())
	// Batched cmd yields a BatchMsg listing the sub-commands; run each.
	require.NotNil(t, cmd)
	if batch, ok := cmd().(tea.BatchMsg); ok {
		for _, sub := range batch {
			if sub != nil {
				sub()
			}
		}
	}
	assert.True(t, ran)
}

func TestConfirmEscCancels(t *testing.T) {
	d := New(styles.DefaultStyles())
	d.Show(core.ShowConfirmMsg{Title: "x"})
	cmd, consumed := d.Update(mkKey("esc"))
	assert.True(t, consumed)
	assert.False(t, d.Active())
	assert.Equal(t, core.ConfirmResolvedMsg{Confirmed: false}, cmd())
}

func TestConfirmSwallowsInputWhileOpen(t *testing.T) {
	d := New(styles.DefaultStyles())
	d.Show(core.ShowConfirmMsg{Title: "x"})
	_, consumed := d.Update(mkKey("q"))
	assert.True(t, consumed, "random keys must be swallowed while modal is open")
	assert.True(t, d.Active())
}

func TestConfirmInactiveIgnores(t *testing.T) {
	d := New(styles.DefaultStyles())
	_, consumed := d.Update(mkKey("enter"))
	assert.False(t, consumed)
}
