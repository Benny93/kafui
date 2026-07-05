package schemadetail

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/ui/components/editor"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// enterDiffFromContent opens the diff view comparing the two newest versions,
// loading the version list first when needed (SR-12).
func (m *Model) enterDiffFromContent() tea.Cmd {
	if !m.versionsLoaded {
		// Load versions, then re-enter diff once they arrive is handled by the
		// user pressing 'd' again; for a single keystroke, load and show the list.
		return m.enterVersions()
	}
	if len(m.versions) < 2 {
		m.SetStatus("Need at least two versions to diff")
		return nil
	}
	prev := m.versions[len(m.versions)-2].Version
	latest := m.latestVersion()
	return m.enterDiff(prev, latest)
}

// enterDiff sets up the diff view for two version numbers and loads their text.
func (m *Model) enterDiff(left, right int) tea.Cmd {
	m.mode = modeDiff
	m.diffLeft = left
	m.diffRight = right
	m.diffActive = 0
	if m.diffView == nil {
		m.diffView = editor.NewDiffView("", "")
	}
	cmds := m.ensureVersionContent(left)
	cmds = append(cmds, m.ensureVersionContent(right)...)
	m.refreshDiff()
	return tea.Batch(cmds...)
}

// ensureVersionContent returns commands to fetch a version's text if uncached.
func (m *Model) ensureVersionContent(version int) []tea.Cmd {
	if version <= 0 {
		return nil
	}
	if _, ok := m.contentCache[version]; ok {
		return nil
	}
	return []tea.Cmd{m.loadVersionContentCmd(version)}
}

// refreshDiff recomputes the diff from the cached (pretty-printed) contents.
func (m *Model) refreshDiff() {
	if m.diffView == nil || m.mode != modeDiff {
		return
	}
	left := prettySchema(m.contentCache[m.diffLeft], m.GetSchemaType())
	right := prettySchema(m.contentCache[m.diffRight], m.GetSchemaType())
	m.diffView.SetContent(left, right)
}

// cycleDiffVersion moves the active pane's version by delta through the list.
func (m *Model) cycleDiffVersion(delta int) tea.Cmd {
	if len(m.versions) == 0 {
		return nil
	}
	current := m.diffLeft
	if m.diffActive == 1 {
		current = m.diffRight
	}
	idx := 0
	for i, v := range m.versions {
		if v.Version == current {
			idx = i
			break
		}
	}
	idx += delta
	if idx < 0 {
		idx = 0
	}
	if idx >= len(m.versions) {
		idx = len(m.versions) - 1
	}
	next := m.versions[idx].Version
	if m.diffActive == 1 {
		m.diffRight = next
	} else {
		m.diffLeft = next
	}
	cmds := m.ensureVersionContent(next)
	m.refreshDiff()
	return tea.Batch(cmds...)
}

func handleDiffKey(m *Model, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "tab", "h", "l":
		m.diffActive = 1 - m.diffActive
		return nil
	case "[":
		return m.cycleDiffVersion(-1)
	case "]":
		return m.cycleDiffVersion(1)
	case "esc", "backspace":
		m.mode = modeContent
		return nil
	default:
		if m.diffView != nil {
			_, cmd := m.diffView.Update(msg)
			return cmd
		}
		return nil
	}
}

func renderDiff(m *Model, width, height int) string {
	titleStyle := lipgloss.NewStyle().Foreground(stylesPkg.Primary).Bold(true)
	activeStyle := lipgloss.NewStyle().Foreground(stylesPkg.BgBase).Background(stylesPkg.Primary).Bold(true)
	sideStyle := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)
	mutedStyle := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)

	left := fmt.Sprintf(" v%d ", m.diffLeft)
	right := fmt.Sprintf(" v%d ", m.diffRight)
	if m.diffActive == 0 {
		left = activeStyle.Render(left)
		right = sideStyle.Render(right)
	} else {
		left = sideStyle.Render(left)
		right = activeStyle.Render(right)
	}

	header := titleStyle.Render(fmt.Sprintf("Diff of %s", m.subject)) +
		"   " + left + sideStyle.Render(" → ") + right

	var body string
	if m.diffView != nil {
		m.diffView.SetDimensions(width, height-3)
		body = m.diffView.View()
	}
	hint := mutedStyle.Render("tab switch pane · [ / ] change version · esc back")
	return strings.Join([]string{header, body, hint}, "\n")
}
