package topic

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/messagefilter"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// smartFilterPrefix marks a search entry as a smart-filter expression (MSG-25).
const smartFilterPrefix = "~"

// unicodeEscape returns the \uXXXX (lower-case hex) representation of every
// non-ASCII rune in s, leaving ASCII runes as-is. Used so a search for "ü" also
// matches content that stores it escaped as "ü" (MSG-23).
func unicodeEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 128 {
			b.WriteRune(r)
		} else {
			b.WriteString(fmt.Sprintf("\\u%04x", r))
		}
	}
	return b.String()
}

// stringMatch reports whether the message matches the substring query across
// key, value, and header names/values, also matching unicode-escaped forms of a
// non-ASCII query (MSG-23). Matching is case-insensitive.
func stringMatch(msg api.Message, query string) bool {
	if query == "" {
		return true
	}
	lq := strings.ToLower(query)
	needles := []string{lq}
	if esc := strings.ToLower(unicodeEscape(query)); esc != lq {
		// A non-ASCII query also matches its \uXXXX escaped form (lower + upper hex).
		needles = append(needles, esc, strings.ToUpper(unicodeEscape(query)))
	}
	hay := []string{strings.ToLower(msg.Key), strings.ToLower(msg.Value)}
	for _, h := range msg.Headers {
		hay = append(hay, strings.ToLower(h.Key), strings.ToLower(h.Value))
	}
	for _, straw := range hay {
		for _, n := range needles {
			if strings.Contains(straw, n) {
				return true
			}
		}
	}
	return false
}

// applyFilter recomputes filteredMessages from messages using either the active
// smart filter or substring matching (MSG-23/24). Callers hold no lock.
func (m *Model) applyFilter() {
	m.smartFilterErrs = 0
	if m.smartFilter == nil && (!m.searchMode || m.searchInput.Value() == "") {
		m.filteredMessages = m.messages
		return
	}
	filtered := make([]api.Message, 0, len(m.messages))
	if m.smartFilter != nil {
		for _, msg := range m.messages {
			ok, err := m.smartFilter.Eval(msg)
			if err != nil {
				m.smartFilterErrs++ // filter error tolerance: skip and count (MSG-24)
				continue
			}
			if ok {
				filtered = append(filtered, msg)
			}
		}
	} else {
		q := m.searchInput.Value()
		for _, msg := range m.messages {
			if stringMatch(msg, q) {
				filtered = append(filtered, msg)
			}
		}
	}
	m.filteredMessages = filtered
}

// setSmartFilter compiles expr and installs it as the active filter, or clears
// the filter when expr is empty. Returns a UIError-bearing command on a compile
// error (MSG-25).
func (m *Model) setSmartFilter(expr string) tea.Cmd {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		m.smartFilter = nil
		return nil
	}
	f, err := messagefilter.Compile(expr)
	if err != nil {
		m.smartFilter = nil
		return core.NotifyError("Invalid filter", err)
	}
	m.smartFilter = f
	m.statusMessage = "Filter " + f.ID() + ": " + f.Expr()
	return nil
}

// --- MSG-25: saved filters overlay ---

func (k *Keys) handleShowSavedFilters(model *Model) tea.Cmd {
	model.showSavedFilters = true
	model.savedFilterCursor = 0
	model.markRenderDirty()
	return nil
}

func (k *Keys) handleSavedFiltersKey(model *Model, msg tea.KeyMsg) tea.Cmd {
	filters := shared.LoadPrefs().SavedFilters
	switch msg.String() {
	case "esc", "q":
		model.showSavedFilters = false
		model.markRenderDirty()
		return nil
	case "up", "k":
		if model.savedFilterCursor > 0 {
			model.savedFilterCursor--
			model.markRenderDirty()
		}
		return nil
	case "down", "j":
		if model.savedFilterCursor < len(filters)-1 {
			model.savedFilterCursor++
			model.markRenderDirty()
		}
		return nil
	case "d":
		// Delete the highlighted saved filter.
		if model.savedFilterCursor >= 0 && model.savedFilterCursor < len(filters) {
			p := shared.LoadPrefs()
			p.SavedFilters = append(p.SavedFilters[:model.savedFilterCursor], p.SavedFilters[model.savedFilterCursor+1:]...)
			_ = shared.SavePrefs(p)
			if model.savedFilterCursor > 0 {
				model.savedFilterCursor--
			}
			model.markRenderDirty()
		}
		return nil
	case "enter":
		// Apply the highlighted saved filter.
		if model.savedFilterCursor >= 0 && model.savedFilterCursor < len(filters) {
			model.showSavedFilters = false
			cmd := model.setSmartFilter(filters[model.savedFilterCursor].Expr)
			model.FilterMessages()
			model.markRenderDirty()
			return cmd
		}
		return nil
	}
	return nil
}

// saveCurrentFilter persists the active smart filter under a generated name (MSG-25).
func (m *Model) saveCurrentFilter() tea.Cmd {
	if m.smartFilter == nil {
		return core.NewNotification(core.StatusWarning, "No filter", "Enter a ~expression filter first")
	}
	p := shared.LoadPrefs()
	name := "filter-" + m.smartFilter.ID()
	// Replace any existing entry with the same expression.
	updated := p.SavedFilters[:0]
	for _, f := range p.SavedFilters {
		if f.Expr != m.smartFilter.Expr() {
			updated = append(updated, f)
		}
	}
	p.SavedFilters = append(updated, shared.SavedFilter{Name: name, Expr: m.smartFilter.Expr()})
	if err := shared.SavePrefs(p); err != nil {
		return core.NotifyError("Save filter failed", err)
	}
	return core.NewNotification(core.StatusSuccess, "Filter saved", name)
}

func (m *Model) renderSavedFiltersOverlay(width int) string {
	muted := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)
	header := lipgloss.NewStyle().Foreground(stylesPkg.Primary).Bold(true)
	var b strings.Builder
	b.WriteString(header.Render("Saved filters"))
	b.WriteString("\n\n")
	filters := shared.LoadPrefs().SavedFilters
	if len(filters) == 0 {
		b.WriteString(muted.Render("No saved filters. In search, type ~<expr> then press ctrl+s to save."))
		b.WriteString("\n\n")
		b.WriteString(muted.Render("esc: close"))
		return b.String()
	}
	for i, f := range filters {
		line := fmt.Sprintf("  %-20s %s", truncate(f.Name, 20), truncate(f.Expr, width-26))
		if i == m.savedFilterCursor {
			line = lipgloss.NewStyle().Foreground(stylesPkg.BgBase).Background(stylesPkg.Primary).Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(muted.Render("↑/↓: select • enter: apply • d: delete • esc: close"))
	return b.String()
}
