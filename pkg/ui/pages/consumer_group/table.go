package consumergroup

import (
	"sort"
	"strconv"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// sortKeys is the cycle order for the topic table sort.
var sortKeys = []string{"topic", "lag"}

func (m *Model) cycleSort() {
	m.sortCol = (m.sortCol + 1) % len(sortKeys)
	if m.sortCol == 0 {
		// wrapping back to "topic" toggles direction so both asc/desc are reachable.
		m.sortDesc = !m.sortDesc
	}
	m.rebuildTopicRows(m.detail)
}

// rebuildTopicRows aggregates the detail's per-partition offsets into one row per
// distinct topic (honoring the active topic filter and sort), computes each
// topic's aggregate lag (nil when the topic has no lag values), captures any
// auto-refresh trend baseline, and rebuilds the topic + partition tables.
func (m *Model) rebuildTopicRows(detail api.ConsumerGroupDetail) {
	byTopic := map[string][]api.PartitionOffset{}
	for _, po := range detail.TopicOffsets {
		byTopic[po.Topic] = append(byTopic[po.Topic], po)
	}

	filter := strings.ToLower(m.topicFilter)
	rows := make([]topicRow, 0, len(byTopic))
	for topic, parts := range byTopic {
		if filter != "" && !strings.Contains(strings.ToLower(topic), filter) {
			continue
		}
		sort.Slice(parts, func(i, j int) bool { return parts[i].Partition < parts[j].Partition })
		tr := topicRow{topic: topic, partitions: parts, aggLag: aggregateLag(parts)}
		if m.trendBaseline != nil {
			if prev, ok := m.trendBaseline[topic]; ok {
				p := prev
				tr.prevAggLag = &p
			}
		}
		rows = append(rows, tr)
	}

	sort.SliceStable(rows, func(i, j int) bool {
		less := rows[i].topic < rows[j].topic
		if sortKeys[m.sortCol] == "lag" {
			less = lagOrZero(rows[i].aggLag) < lagOrZero(rows[j].aggLag)
		}
		if m.sortDesc {
			return !less
		}
		return less
	})

	m.topicRows = rows
	if m.expanded >= len(rows) {
		m.expanded = -1
	}
	m.rebuildTopicTable()
	if m.expanded >= 0 {
		m.rebuildPartTable()
	}
}

func (m *Model) rebuildTopicTable() {
	trs := make([]table.Row, 0, len(m.topicRows))
	for _, tr := range m.topicRows {
		trs = append(trs, table.Row{
			tr.topic,
			strconv.Itoa(len(tr.partitions)),
			formatLagPtr(tr.aggLag),
			m.trendArrow(tr),
		})
	}
	cur := m.topicTable.Cursor()
	m.topicTable.SetRows(trs)
	if cur >= len(trs) {
		cur = 0
	}
	m.topicTable.SetCursor(cur)
}

func (m *Model) rebuildPartTable() {
	if m.expanded < 0 || m.expanded >= len(m.topicRows) {
		m.partTable.SetRows(nil)
		return
	}
	var rows []table.Row
	for _, p := range m.topicRows[m.expanded].partitions {
		rows = append(rows, table.Row{
			strconv.FormatInt(int64(p.Partition), 10),
			orDash(p.MemberID),
			orDash(p.MemberHost),
			formatOffset(p.CommittedOffset),
			strconv.FormatInt(p.EndOffset, 10),
			formatLagPtr(p.Lag),
		})
	}
	m.partTable.SetRows(rows)
	m.partTable.SetCursor(0)
}

// aggregateLag sums the (non-nil) partition lags; nil when there are none.
func aggregateLag(parts []api.PartitionOffset) *int64 {
	var total int64
	any := false
	for _, p := range parts {
		if p.Lag != nil {
			total += *p.Lag
			any = true
		}
	}
	if !any {
		return nil
	}
	return &total
}

// groupTotalLag sums the aggregate lag across all topics; nil when none defined.
func groupTotalLag(detail api.ConsumerGroupDetail) *int64 {
	var total int64
	any := false
	for _, po := range detail.TopicOffsets {
		if po.Lag != nil {
			total += *po.Lag
			any = true
		}
	}
	if !any {
		return nil
	}
	return &total
}

func lagOrZero(lag *int64) int64 {
	if lag == nil {
		return 0
	}
	return *lag
}

func formatLagPtr(lag *int64) string {
	if lag == nil {
		return "—"
	}
	return strconv.FormatInt(*lag, 10)
}

func formatOffset(o *int64) string {
	if o == nil {
		return "" // assigned-but-uncommitted partition: empty committed cell
	}
	return strconv.FormatInt(*o, 10)
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// groupStateStyled renders a state with a semantic colour role.
func groupStateStyled(common *core.Common, state string) string {
	var style lipgloss.Style
	switch state {
	case api.GroupStateStable:
		style = lipgloss.NewStyle().Foreground(stylesPkg.Success)
	case api.GroupStatePreparingRebalance, api.GroupStateCompletingRebalance:
		style = lipgloss.NewStyle().Foreground(stylesPkg.Warning)
	case api.GroupStateDead:
		style = lipgloss.NewStyle().Foreground(stylesPkg.Error)
	default:
		style = lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)
	}
	return style.Render(state)
}

// asGroupNotFound reports whether err is (or wraps) an api.GroupNotFoundError.
func asGroupNotFound(err error, dst *api.GroupNotFoundError) bool {
	for err != nil {
		if e, ok := err.(api.GroupNotFoundError); ok {
			*dst = e
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
