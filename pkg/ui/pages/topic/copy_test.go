package topic

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
)

func TestCopySelected(t *testing.T) {
	orig := writeClipboard
	t.Cleanup(func() { writeClipboard = orig })

	newModel := func() *Model {
		m := NewModel(&MockDataSource{}, "test-topic", api.Topic{})
		m.addMessageInternal(api.Message{Offset: 1, Key: "the-key", Value: "the-value"})
		m.sortMessages()
		m.pagination.SetTotalMessages(1)
		return m
	}

	t.Run("copies the value", func(t *testing.T) {
		var got string
		writeClipboard = func(s string) error { got = s; return nil }
		m := newModel()
		cmd := m.keys.handleCopyValue(m)
		require.NotNil(t, cmd, "a copy is confirmed with a toast")
		n, ok := cmd().(core.NotificationMsg)
		require.True(t, ok)
		assert.Equal(t, core.StatusSuccess, n.Severity)
		assert.Equal(t, "the-value", got)
		assert.Contains(t, m.statusMessage, "copied")
	})

	t.Run("copies the key", func(t *testing.T) {
		var got string
		writeClipboard = func(s string) error { got = s; return nil }
		m := newModel()
		m.keys.handleCopyKey(m)
		assert.Equal(t, "the-key", got)
	})

	t.Run("reports a failure", func(t *testing.T) {
		writeClipboard = func(string) error { return errors.New("no clipboard") }
		m := newModel()
		cmd := m.keys.handleCopyValue(m)
		require.NotNil(t, cmd)
		n, ok := cmd().(core.NotificationMsg)
		require.True(t, ok)
		assert.Equal(t, core.StatusError, n.Severity)
		assert.NotContains(t, m.statusMessage, "copied to clipboard")
		assert.Contains(t, m.statusMessage, "Copy failed")
	})
}

// With a row expanded, c copies exactly what the panel shows: the displayed
// value, pretty-printed. Collapsed, it copies the value as stored.
func TestCopyExpandedCopiesPanelContent(t *testing.T) {
	orig := writeClipboard
	t.Cleanup(func() { writeClipboard = orig })
	var got string
	writeClipboard = func(s string) error { got = s; return nil }

	m := expandModel(t, 1, 3)
	sel := m.GetSelectedMessage()
	require.NotNil(t, sel)
	k := NewKeys()
	c := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")}

	k.HandleKey(m, c)
	assert.Equal(t, sel.Value, got, "collapsed: the stored value")

	k.HandleKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	out := render(m, 120, 30)
	cmd := k.HandleKey(m, c)
	want, isJSON := expandedBody(m.displayValue(*sel))
	require.True(t, isJSON)
	assert.Equal(t, want, got, "expanded: the panel's text")
	assert.Contains(t, got, "\n  \"field0\": ")
	for _, line := range strings.Split(got, "\n") {
		assert.Contains(t, out, line, "every copied line is what the panel shows")
	}
	require.NotNil(t, cmd)
	n, ok := cmd().(core.NotificationMsg)
	require.True(t, ok)
	assert.Equal(t, core.StatusSuccess, n.Severity)
	assert.Contains(t, n.Message, "5 lines")
}
