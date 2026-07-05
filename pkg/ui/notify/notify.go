// Package notify provides the shell-owned notification (status line) system:
// severity-styled, auto-expiring, deduplicated transient messages rendered in
// the footer area. It unifies core.NotificationMsg, core.StatusMsg and
// shared.UIError into a single stream.
package notify

import (
	"strconv"
	"time"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// tickMsg drives auto-expiry.
type tickMsg struct{}

const (
	defaultTTL = 5 * time.Second
	errorTTL   = 10 * time.Second
	tickEvery  = time.Second
)

type entry struct {
	sev     core.StatusType
	title   string
	message string
	expires time.Time // zero = sticky
}

// Manager holds the active notifications. It is owned by the root model.
type Manager struct {
	styles  *styles.Styles
	items   []entry
	ticking bool
	nowFn   func() time.Time // seam for tests
}

// New creates a Manager.
func New(s *styles.Styles) *Manager {
	return &Manager{styles: s, nowFn: time.Now}
}

func (m *Manager) now() time.Time { return m.nowFn() }

// Push adds a notification and returns any command needed to drive expiry.
// Identical consecutive messages are deduplicated (the previous one's timer is
// refreshed rather than adding a duplicate line).
func (m *Manager) Push(n core.NotificationMsg) tea.Cmd {
	ttl := defaultTTL
	if n.Severity == core.StatusError {
		ttl = errorTTL
	}
	var exp time.Time
	if !n.Sticky {
		exp = m.now().Add(ttl)
	}
	e := entry{sev: n.Severity, title: n.Title, message: n.Message, expires: exp}

	if len(m.items) > 0 {
		last := &m.items[len(m.items)-1]
		if last.sev == e.sev && last.title == e.title && last.message == e.message {
			last.expires = exp // dedup: refresh timer
			return m.ensureTick()
		}
	}
	m.items = append(m.items, e)
	// Cap history to keep the render bounded.
	if len(m.items) > 5 {
		m.items = m.items[len(m.items)-5:]
	}
	return m.ensureTick()
}

// HandleMsg intercepts notification-bearing messages. It returns a command and
// whether the message was a notification (already consumed here).
func (m *Manager) HandleMsg(msg tea.Msg) (tea.Cmd, bool) {
	switch v := msg.(type) {
	case core.NotificationMsg:
		return m.Push(v), true
	case core.StatusMsg:
		return m.Push(core.NotificationMsg{Severity: v.Type, Message: v.Message}), true
	case core.StatusMessage:
		return m.Push(core.NotificationMsg{Severity: v.Type, Message: v.Message, Sticky: v.TTL == 0}), true
	case shared.UIError:
		return m.Push(core.NotificationMsg{Severity: core.StatusError, Title: v.Type, Message: v.Error()}), true
	case tickMsg:
		m.prune()
		if len(m.items) == 0 {
			m.ticking = false
			return nil, true
		}
		return tick(), true
	}
	return nil, false
}

func (m *Manager) ensureTick() tea.Cmd {
	if m.ticking {
		return nil
	}
	m.ticking = true
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(tickEvery, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m *Manager) prune() {
	now := m.now()
	kept := m.items[:0]
	for _, e := range m.items {
		if e.expires.IsZero() || e.expires.After(now) {
			kept = append(kept, e)
		}
	}
	m.items = kept
}

// Dismiss clears all notifications (manual dismiss key).
func (m *Manager) Dismiss() { m.items = nil }

// Empty reports whether there are no active notifications.
func (m *Manager) Empty() bool { return len(m.items) == 0 }

// View renders the most recent notification as a single status line (with a
// "(+N more)" suffix when several are queued). Returns "" when empty.
func (m *Manager) View(width int) string {
	if len(m.items) == 0 {
		return ""
	}
	e := m.items[len(m.items)-1]
	text := e.message
	if e.title != "" {
		text = e.title + ": " + e.message
	}
	if extra := len(m.items) - 1; extra > 0 {
		text += "  " + lipgloss.NewStyle().Foreground(styles.FgSubtle).Render("(+"+strconv.Itoa(extra)+" more)")
	}
	line := m.severityStyle(e.sev).Render(" " + iconFor(e.sev) + " " + text + " ")
	if width > 0 {
		line = lipgloss.NewStyle().Width(width).MaxWidth(width).Render(line)
	}
	return line
}

func (m *Manager) severityStyle(sev core.StatusType) lipgloss.Style {
	switch sev {
	case core.StatusError:
		return lipgloss.NewStyle().Foreground(styles.FgBase).Background(styles.Error).Bold(true)
	case core.StatusWarning:
		return lipgloss.NewStyle().Foreground(styles.BgBase).Background(styles.Warning)
	case core.StatusSuccess:
		return lipgloss.NewStyle().Foreground(styles.BgBase).Background(styles.Success)
	default:
		return lipgloss.NewStyle().Foreground(styles.FgBase).Background(styles.Info)
	}
}

func iconFor(sev core.StatusType) string {
	switch sev {
	case core.StatusError:
		return "✗"
	case core.StatusWarning:
		return "⚠"
	case core.StatusSuccess:
		return "✓"
	default:
		return "ℹ"
	}
}

