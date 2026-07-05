package editor

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

const sampleContent = `line one
matching alpha
line three
matching beta
line five`

func TestViewerSearchFindsAndNavigates(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		wantMatches int
		wantFirst   int // expected line index of the first match
	}{
		{"two matches", "matching", 2, 1},
		{"single match", "five", 1, 4},
		{"no match", "zzz", 0, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewViewer(sampleContent)
			v.SetDimensions(40, 10)
			v.Search(tt.query)

			assert.Equal(t, tt.wantMatches, v.MatchCount())
			assert.Equal(t, tt.wantFirst, v.MatchLine())
		})
	}

	t.Run("next and prev wrap around", func(t *testing.T) {
		v := NewViewer(sampleContent)
		v.SetDimensions(40, 10)
		v.Search("matching")

		assert.Equal(t, 0, v.MatchIndex())
		v.NextMatch()
		assert.Equal(t, 1, v.MatchIndex())
		v.NextMatch() // wraps to first
		assert.Equal(t, 0, v.MatchIndex())
		v.PrevMatch() // wraps to last
		assert.Equal(t, 1, v.MatchIndex())
	})

	t.Run("slash key opens search prompt", func(t *testing.T) {
		v := NewViewer(sampleContent)
		v.SetDimensions(40, 10)
		v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
		assert.Contains(t, v.View(), "/")
	})
}

func TestViewerWrapToggleChangesRendering(t *testing.T) {
	long := strings.Repeat("abcdefghij", 10) // 100 runes, one line
	v := NewViewer(long)
	v.SetDimensions(20, 10)

	before := v.View()
	assert.False(t, v.Wrapped())

	v.ToggleWrap()
	assert.True(t, v.Wrapped())
	after := v.View()

	assert.NotEqual(t, before, after, "wrap toggle should change rendering")
}

func TestSoftWrap(t *testing.T) {
	segs := softWrap("abcdefghij", 4)
	assert.Equal(t, []string{"abcd", "efgh", "ij"}, segs)
	assert.Equal(t, []string{"short"}, softWrap("short", 20))
}

func TestDiffRendersExpectedMarkers(t *testing.T) {
	oldText := "keep\nremove\nshared"
	newText := "keep\nadded\nshared"

	out := Diff(oldText, newText)

	assert.Contains(t, out, "  keep")   // context line
	assert.Contains(t, out, "- remove") // deletion
	assert.Contains(t, out, "+ added")  // addition
	assert.Contains(t, out, "  shared") // context line

	// The deletion must appear before the insertion in the output.
	assert.Less(t, strings.Index(out, "- remove"), strings.Index(out, "+ added"))
}

func TestDiffLinesEditScript(t *testing.T) {
	ops := diffLines(
		[]string{"a", "b", "c"},
		[]string{"a", "x", "c"},
	)
	kinds := make([]opKind, len(ops))
	for i, op := range ops {
		kinds[i] = op.kind
	}
	assert.Equal(t, []opKind{opEqual, opDelete, opInsert, opEqual}, kinds)
}

func TestHighlightJSONNonEmpty(t *testing.T) {
	line := `  "name": "value",`
	out := HighlightJSON(line)
	assert.NotEmpty(t, out)
	// Highlighting must preserve the underlying characters.
	assert.Contains(t, out, "name")
	assert.Contains(t, out, "value")
}

func TestEditorValue(t *testing.T) {
	e := NewEditor("hello")
	assert.Equal(t, "hello", e.Value())
	e.SetValue("world")
	assert.Equal(t, "world", e.Value())
	e.SetDimensions(40, 5)
	assert.NotPanics(t, func() { _ = e.View() })
}

func TestDiffViewRenders(t *testing.T) {
	d := NewDiffView("a\nb", "a\nc")
	d.SetDimensions(40, 10)
	assert.NotPanics(t, func() { _ = d.View() })
	d.SetContent("x", "y")
	assert.NotPanics(t, func() { _ = d.View() })
}
