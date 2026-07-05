package editor

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/styles"
)

var (
	diffAddStyle = lipgloss.NewStyle().Foreground(styles.Success)
	diffDelStyle = lipgloss.NewStyle().Foreground(styles.Error)
	diffCtxStyle = lipgloss.NewStyle().Foreground(styles.FgMuted)
)

// Diff renders a unified line-by-line diff of oldText vs newText. Context lines
// are prefixed with a space, deletions with `-` (error color) and additions
// with `+` (success color).
func Diff(oldText, newText string) string {
	a := strings.Split(oldText, "\n")
	b := strings.Split(newText, "\n")
	var out []string
	for _, op := range diffLines(a, b) {
		switch op.kind {
		case opEqual:
			out = append(out, diffCtxStyle.Render("  "+op.text))
		case opDelete:
			out = append(out, diffDelStyle.Render("- "+op.text))
		case opInsert:
			out = append(out, diffAddStyle.Render("+ "+op.text))
		}
	}
	return strings.Join(out, "\n")
}

type opKind int

const (
	opEqual opKind = iota
	opDelete
	opInsert
)

type diffOp struct {
	kind opKind
	text string
}

// diffLines computes an ordered edit script between a and b using the classic
// LCS dynamic-programming table.
func diffLines(a, b []string) []diffOp {
	n, m := len(a), len(b)
	// lcs[i][j] = length of LCS of a[i:] and b[j:]
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	var ops []diffOp
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			ops = append(ops, diffOp{opEqual, a[i]})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			ops = append(ops, diffOp{opDelete, a[i]})
			i++
		default:
			ops = append(ops, diffOp{opInsert, b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, diffOp{opDelete, a[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, diffOp{opInsert, b[j]})
	}
	return ops
}

// DiffView is a scrollable component wrapping a rendered unified diff.
type DiffView struct {
	core.BaseComponent

	viewport viewport.Model
}

// NewDiffView creates a diff view for oldText vs newText.
func NewDiffView(oldText, newText string) *DiffView {
	vp := viewport.New(1, 1)
	vp.SetContent(Diff(oldText, newText))
	return &DiffView{viewport: vp}
}

// SetContent recomputes the diff for new inputs.
func (d *DiffView) SetContent(oldText, newText string) {
	d.viewport.SetContent(Diff(oldText, newText))
}

// SetDimensions sizes the diff view.
func (d *DiffView) SetDimensions(width, height int) {
	d.BaseComponent.SetDimensions(width, height)
	if width > 0 {
		d.viewport.Width = width
	}
	if height > 0 {
		d.viewport.Height = height
	}
}

// Update forwards messages to the viewport for scrolling.
func (d *DiffView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return d, cmd
}

// View renders the diff view.
func (d *DiffView) View() string { return d.viewport.View() }
