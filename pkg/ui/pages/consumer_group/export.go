package consumergroup

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Benny93/kafui/pkg/ui/core"
	tea "github.com/charmbracelet/bubbletea"
)

// exportCSV (CG-16) writes the currently displayed topic rows (post-filter,
// current sort) as CSV — one row per topic plus the per-partition breakdown —
// to a timestamped file, surfacing the absolute path in the status bar.
func (m *Model) exportCSV() tea.Cmd {
	if len(m.topicRows) == 0 {
		return nil
	}
	rows := m.topicRows
	group := m.groupID
	return func() tea.Msg {
		filename := fmt.Sprintf("consumer-group-%s-%s.csv", sanitize(group), time.Now().Format("20060102-150405"))
		f, err := os.Create(filename)
		if err != nil {
			return core.NotifyError("CSV export failed", err)()
		}
		defer f.Close()
		cw := csv.NewWriter(f)
		if err := cw.Write([]string{"Topic", "Partition", "Committed", "End", "Lag", "Consumer", "Host"}); err != nil {
			return core.NotifyError("CSV export failed", err)()
		}
		for _, tr := range rows {
			// Aggregate row (partition column empty).
			_ = cw.Write([]string{tr.topic, "", "", "", formatLagPtr(tr.aggLag), "", ""})
			for _, p := range tr.partitions {
				_ = cw.Write([]string{
					tr.topic,
					fmt.Sprintf("%d", p.Partition),
					formatOffset(p.CommittedOffset),
					fmt.Sprintf("%d", p.EndOffset),
					formatLagPtr(p.Lag),
					p.MemberID,
					p.MemberHost,
				})
			}
		}
		cw.Flush()
		if err := cw.Error(); err != nil {
			return core.NotifyError("CSV export failed", err)()
		}
		abs, _ := filepath.Abs(filename)
		return core.NotificationMsg{Severity: core.StatusInfo, Title: "Group exported", Message: abs}
	}
}

// sanitize replaces path-hostile characters in a group id for the filename.
func sanitize(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch r {
		case '/', '\\', ':', ' ':
			out = append(out, '_')
		default:
			out = append(out, r)
		}
	}
	return string(out)
}
