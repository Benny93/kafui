package editor

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/styles"
)

// Viewer is a read-only content viewer with line numbers, a soft-wrap toggle
// and in-content `/` search with n/N match navigation. It generalizes the
// older JSONContentView: pass Highlight to colorize JSON content.
type Viewer struct {
	core.BaseComponent

	content   string
	lines     []string
	viewport  viewport.Model
	wrap      bool
	highlight bool

	// search state
	search    textinput.Model
	searching bool // true while the `/` prompt is open
	query     string
	matches   []int // indices into lines that contain the current query
	matchIdx  int   // index into matches of the active match
}

var (
	lineNumberStyle = lipgloss.NewStyle().Foreground(styles.FgSubtle)
	matchStyle      = lipgloss.NewStyle().Foreground(styles.BgBase).Background(styles.Warning)
	activeMatch     = lipgloss.NewStyle().Foreground(styles.BgBase).Background(styles.Primary).Bold(true)
)

// NewViewer creates a read-only viewer for the given content.
func NewViewer(content string) *Viewer {
	ti := textinput.New()
	ti.Prompt = "/"
	v := &Viewer{
		viewport: viewport.New(1, 1),
		search:   ti,
		matchIdx: -1,
	}
	v.SetContent(content)
	return v
}

// SetHighlight enables or disables JSON syntax highlighting.
func (v *Viewer) SetHighlight(on bool) {
	v.highlight = on
	v.render()
}

// SetContent replaces the viewed content and resets search state.
func (v *Viewer) SetContent(content string) {
	v.content = content
	v.lines = strings.Split(content, "\n")
	v.query = ""
	v.matches = nil
	v.matchIdx = -1
	v.render()
}

// SetDimensions sets the viewer size.
func (v *Viewer) SetDimensions(width, height int) {
	v.BaseComponent.SetDimensions(width, height)
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	v.viewport.Width = width
	v.viewport.Height = height
	v.render()
}

// ToggleWrap flips soft-wrapping of long lines.
func (v *Viewer) ToggleWrap() {
	v.wrap = !v.wrap
	v.render()
}

// Wrapped reports whether soft-wrap is currently on.
func (v *Viewer) Wrapped() bool { return v.wrap }

// Searching reports whether the in-content `/` search prompt is currently open.
// Callers that add their own single-key hotkeys should defer to the viewer while
// this is true so keystrokes reach the search field unmodified.
func (v *Viewer) Searching() bool { return v.searching }

// Search sets the active query, computes matching lines and jumps to the first
// match. An empty query clears the search.
func (v *Viewer) Search(query string) {
	v.query = query
	v.matches = nil
	v.matchIdx = -1
	if query != "" {
		for i, line := range v.lines {
			if strings.Contains(strings.ToLower(line), strings.ToLower(query)) {
				v.matches = append(v.matches, i)
			}
		}
		if len(v.matches) > 0 {
			v.matchIdx = 0
		}
	}
	v.render()
	v.scrollToMatch()
}

// NextMatch advances to the next search match (wraps around).
func (v *Viewer) NextMatch() {
	if len(v.matches) == 0 {
		return
	}
	v.matchIdx = (v.matchIdx + 1) % len(v.matches)
	v.render()
	v.scrollToMatch()
}

// PrevMatch moves to the previous search match (wraps around).
func (v *Viewer) PrevMatch() {
	if len(v.matches) == 0 {
		return
	}
	v.matchIdx = (v.matchIdx - 1 + len(v.matches)) % len(v.matches)
	v.render()
	v.scrollToMatch()
}

// MatchCount returns the number of lines matching the current query.
func (v *Viewer) MatchCount() int { return len(v.matches) }

// MatchIndex returns the index of the active match, or -1 if there is none.
func (v *Viewer) MatchIndex() int { return v.matchIdx }

// MatchLine returns the line index (0-based) of the active match, or -1.
func (v *Viewer) MatchLine() int {
	if v.matchIdx < 0 || v.matchIdx >= len(v.matches) {
		return -1
	}
	return v.matches[v.matchIdx]
}

func (v *Viewer) scrollToMatch() {
	if v.matchIdx < 0 || v.matchIdx >= len(v.matches) {
		return
	}
	v.viewport.SetYOffset(v.matches[v.matchIdx])
}

// Update handles key input: `/` opens search, enter runs it, esc cancels,
// n/N navigate matches, w toggles wrap. Other keys scroll the viewport.
func (v *Viewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if v.searching {
			switch key.String() {
			case "enter":
				v.searching = false
				v.Search(v.search.Value())
				return v, nil
			case "esc":
				v.searching = false
				v.search.Blur()
				return v, nil
			}
			var cmd tea.Cmd
			v.search, cmd = v.search.Update(msg)
			return v, cmd
		}
		switch key.String() {
		case "/":
			v.searching = true
			v.search.SetValue("")
			return v, v.search.Focus()
		case "n":
			v.NextMatch()
			return v, nil
		case "N":
			v.PrevMatch()
			return v, nil
		case "w":
			v.ToggleWrap()
			return v, nil
		}
	}
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// render rebuilds the viewport content from the current display options.
func (v *Viewer) render() {
	width := v.viewport.Width
	var out []string
	for i, line := range v.lines {
		wrapped := []string{line}
		if v.wrap && width > 6 {
			wrapped = softWrap(line, width-6)
		}
		matched := v.isMatch(i)
		for w, seg := range wrapped {
			text := seg
			switch {
			case matched && v.matches[v.matchIdx] == i:
				text = activeMatch.Render(seg)
			case matched:
				text = matchStyle.Render(seg)
			case v.highlight:
				text = HighlightJSON(seg)
			}
			num := ""
			if w == 0 {
				num = lineNumberStyle.Render(fmt.Sprintf("%4d ", i+1))
			} else {
				num = strings.Repeat(" ", 5)
			}
			out = append(out, num+text)
		}
	}
	v.viewport.SetContent(strings.Join(out, "\n"))
}

func (v *Viewer) isMatch(line int) bool {
	for _, m := range v.matches {
		if m == line {
			return true
		}
	}
	return false
}

// View renders the viewer, appending the search prompt or match count.
func (v *Viewer) View() string {
	body := v.viewport.View()
	if v.searching {
		return lipgloss.JoinVertical(lipgloss.Left, body, v.search.View())
	}
	if v.query != "" {
		status := fmt.Sprintf("/%s  %d matches", v.query, len(v.matches))
		if len(v.matches) > 0 {
			status = fmt.Sprintf("/%s  %d/%d", v.query, v.matchIdx+1, len(v.matches))
		}
		return lipgloss.JoinVertical(lipgloss.Left, body, lineNumberStyle.Render(status))
	}
	return body
}

// softWrap breaks a line into segments no wider than width runes.
func softWrap(line string, width int) []string {
	if width < 1 {
		width = 1
	}
	runes := []rune(line)
	if len(runes) <= width {
		return []string{line}
	}
	var segs []string
	for len(runes) > width {
		segs = append(segs, string(runes[:width]))
		runes = runes[width:]
	}
	segs = append(segs, string(runes))
	return segs
}
