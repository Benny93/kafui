// Package broker implements the broker detail page (dynamic page ID
// "broker:<id>"). It renders a summary strip plus three tabs — Log Dirs,
// Configs and Metrics — over the shared template shell. The page is created by
// the router; see NewModelWithCommon / NewModelWithInfo for the constructors the
// router wires to the "broker:<id>" dynamic ID.
package broker

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Benny93/kafui/pkg/ui/components/tabstrip"
	"github.com/Benny93/kafui/pkg/ui/shared"
)

func (m *Model) render(width, height int) string {
	// Size the tables to the actual content area (minus the frame border) so
	// they fill the pane and never overflow it. height budget: summary(1) +
	// tabbar(1) + blank(1) + hint(1) + frame(2) ≈ 6, plus slack.
	innerW := width - 2
	if innerW < 20 {
		innerW = 20
	}
	th := height - 8
	if th < 3 {
		th = 3
	}
	m.logTable.SetWidth(innerW)
	m.cfgTable.SetWidth(innerW)
	if m.expanded >= 0 {
		half := th/2 - 1
		if half < 2 {
			half = 2
		}
		m.logTable.SetHeight(half)
		m.partTable.SetWidth(innerW)
		m.partTable.SetHeight(half)
	} else {
		m.logTable.SetHeight(th)
	}
	m.cfgTable.SetHeight(th)

	var b strings.Builder
	b.WriteString(m.summaryStrip())
	b.WriteString("\n")
	b.WriteString(m.tabBar())
	b.WriteString("\n\n")

	if m.notFound {
		b.WriteString(m.common.Styles.Error.Render(fmt.Sprintf("Broker %d not found.", m.brokerID)))
		b.WriteString("\n")
		b.WriteString(m.common.Styles.Muted.Render("Press r to retry."))
		return b.String()
	}

	if m.moveForm != nil {
		b.WriteString(m.common.Styles.Header.Render("Move replica log directory"))
		b.WriteString("\n\n")
		b.WriteString(m.moveForm.View())
		return b.String()
	}

	switch m.active {
	case tabConfigs:
		b.WriteString(m.renderConfigs())
	case tabMetrics:
		b.WriteString(m.renderMetrics())
	default:
		b.WriteString(m.renderLogDirs())
	}
	return b.String()
}

func (m *Model) summaryStrip() string {
	host := m.info.Host
	port := strconv.FormatInt(int64(m.info.Port), 10)
	seg := "N/A"
	if size, count, ok := m.diskUsage(); ok {
		seg = shared.FormatDiskUsage(size, count)
	}
	parts := []string{
		m.common.Styles.Header.Render(fmt.Sprintf("Broker %d", m.brokerID)),
		"Host: " + host,
		"Port: " + port,
		"Disk: " + seg,
	}
	if m.info.IsController {
		parts = append(parts, m.common.Styles.StatusStyle.Success.Render("★ Active Controller"))
	}
	return strings.Join(parts, "   ")
}

// tabBar renders the shared, click-and-hover-aware tab strip.
func (m *Model) tabBar() string {
	m.tabs().SetActive(int(m.active))
	return m.tabs().View()
}

// tabs lazily builds this screen's tab strip. Its zone ids must stay stable
// across renders, so the strip is created once and reused.
func (m *Model) tabs() *tabstrip.Model {
	if m.tabStrip == nil {
		titles := make([]string, 0, len(tabTitles))
		for _, t := range tabTitles {
			titles = append(titles, t.String())
		}
		m.tabStrip = tabstrip.New("broker", titles)
	}
	return m.tabStrip
}

func (m *Model) renderMetrics() string {
	if !m.metricsLoaded {
		return m.common.Styles.Muted.Render("Loading metrics…")
	}
	if m.metricsErr != nil {
		return m.common.Styles.Muted.Render("Metrics data not available")
	}
	return m.metricsViewer.View()
}

// --- ordering / filtering helpers (pure) ---
