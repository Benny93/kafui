package schemadetail

import (
	"fmt"
	"github.com/Benny93/kafui/pkg/ui/keys"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// enterVersions switches to the version list, loading versions if needed (SR-11).
func (m *Model) enterVersions() tea.Cmd {
	m.mode = modeVersions
	m.versionCursor = 0
	if m.versionsLoaded {
		return nil
	}
	return m.loadVersionsCmd()
}

// displayVersions returns the versions newest-first for display.
func (m *Model) displayVersions() []api.SchemaVersion {
	out := make([]api.SchemaVersion, len(m.versions))
	for i, v := range m.versions {
		out[len(m.versions)-1-i] = v
	}
	return out
}

// latestVersion returns the highest version number, or 0 when none are loaded.
func (m *Model) latestVersion() int {
	if len(m.versions) == 0 {
		return 0
	}
	return m.versions[len(m.versions)-1].Version
}

// SelectedVersion returns the version number highlighted in the list, or 0.
func (m *Model) SelectedVersion() int {
	dv := m.displayVersions()
	if m.versionCursor < 0 || m.versionCursor >= len(dv) {
		return 0
	}
	return dv[m.versionCursor].Version
}

func handleVersionsKey(m *Model, msg tea.KeyMsg) tea.Cmd {
	dv := m.displayVersions()
	action, bound := keys.Default.Resolve(keys.ScopeOverlay, msg.String())
	if !bound {
		return nil
	}
	switch action {
	case keys.ActionUp:
		if m.versionCursor > 0 {
			m.versionCursor--
		}
	case keys.ActionDown:
		if m.versionCursor < len(dv)-1 {
			m.versionCursor++
		}
	case keys.ActionActivate:
		if v := m.SelectedVersion(); v > 0 {
			return m.selectVersion(dv[m.versionCursor])
		}
	case keys.ActionEdit:
		// Diffing is the "look at it another way" action for a version.
		if len(m.versions) >= 2 {
			return m.enterDiff(m.SelectedVersion(), m.latestVersion())
		}
	case keys.ActionDelete:
		if v := m.SelectedVersion(); v > 0 {
			return m.confirmDeleteVersionCmd(v)
		}
	case keys.ActionCancel:
		m.mode = modeContent
	}
	return nil
}

// selectVersion loads a chosen version's content into the read-only view.
func (m *Model) selectVersion(v api.SchemaVersion) tea.Cmd {
	m.version = v.Version
	m.schemaID = v.ID
	if v.SchemaType != "" {
		m.schemaType = v.SchemaType
	}
	m.mode = modeContent
	m.loading = true
	return m.LoadContentAsync()
}

func renderVersionList(m *Model, width, height int) string {
	titleStyle := lipgloss.NewStyle().Foreground(stylesPkg.Primary).Bold(true)
	cursorStyle := lipgloss.NewStyle().Foreground(stylesPkg.BgBase).Background(stylesPkg.Primary)
	rowStyle := lipgloss.NewStyle().Foreground(stylesPkg.FgBase)
	mutedStyle := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)

	if !m.versionsLoaded {
		return mutedStyle.Render("Loading versions…")
	}
	dv := m.displayVersions()
	if len(dv) == 0 {
		return mutedStyle.Render("No versions found for this subject.")
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Versions of %s (%d)", m.subject, len(dv))))
	b.WriteString("\n\n")
	latest := m.latestVersion()
	for i, v := range dv {
		typ := v.SchemaType
		if typ == "" {
			typ = "AVRO"
		}
		marker := ""
		if v.Version == latest {
			marker = "  (latest)"
		}
		line := fmt.Sprintf("v%-4d  id:%-6d  %-8s%s", v.Version, v.ID, typ, marker)
		if i == m.versionCursor {
			b.WriteString(cursorStyle.Render("▸ " + line))
		} else {
			b.WriteString(rowStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(keys.Hint(keys.ScopeOverlay, keys.ActionUp, "select", keys.ActionActivate, "view", keys.ActionEdit, "diff vs latest", keys.ActionDelete, "delete version", keys.ActionCancel, "back")))
	return b.String()
}
