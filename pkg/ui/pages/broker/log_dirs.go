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

	"github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

func logDirColumns() []table.Column {
	return []table.Column{
		{Title: "Directory", Width: 34},
		{Title: "Error", Width: 22},
		{Title: "Topics", Width: 8},
		{Title: "Partitions", Width: 12},
	}
}

func partitionColumns() []table.Column {
	return []table.Column{
		{Title: "Topic", Width: 28},
		{Title: "Partition", Width: 10},
		{Title: "Size", Width: 14},
		{Title: "Offset Lag", Width: 12},
	}
}

func (m *Model) handleLogDirsKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		if m.expanded >= 0 {
			m.expanded = -1 // collapse
			return nil
		}
		i := m.logTable.Cursor()
		if i >= 0 && i < len(m.logDirs) {
			m.expanded = i
			m.rebuildPartTable()
		}
		return nil
	case "esc":
		if m.expanded >= 0 {
			m.expanded = -1
			return nil
		}
	}
	return m.forwardToActive(msg)
}

func (m *Model) rebuildLogTable() {
	rows := make([]table.Row, 0, len(m.logDirs))
	for _, d := range m.logDirs {
		parts := 0
		for _, t := range d.Topics {
			parts += len(t.Partitions)
		}
		rows = append(rows, table.Row{d.Path, d.Error, strconv.Itoa(len(d.Topics)), strconv.Itoa(parts)})
	}
	m.logTable.SetRows(rows)
}

func (m *Model) rebuildPartTable() {
	if m.expanded < 0 || m.expanded >= len(m.logDirs) {
		m.partTable.SetRows(nil)
		return
	}
	var rows []table.Row
	for _, t := range m.logDirs[m.expanded].Topics {
		for _, p := range t.Partitions {
			rows = append(rows, table.Row{
				t.Topic,
				strconv.FormatInt(int64(p.Partition), 10),
				shared.FormatBytes2dp(p.Size),
				strconv.FormatInt(p.OffsetLag, 10),
			})
		}
	}
	m.partTable.SetRows(rows)
	m.partTable.SetCursor(0)
}

// selectedPartition returns the topic/partition currently highlighted in the
// expanded partition table.
func (m *Model) selectedPartition() (topic string, partition int32, ok bool) {
	if m.expanded < 0 || m.expanded >= len(m.logDirs) {
		return "", 0, false
	}
	idx := 0
	cursor := m.partTable.Cursor()
	for _, t := range m.logDirs[m.expanded].Topics {
		for _, p := range t.Partitions {
			if idx == cursor {
				return t.Topic, p.Partition, true
			}
			idx++
		}
	}
	return "", 0, false
}

// --- Reassignment form (BR-17) ---

func (m *Model) openMoveForm() tea.Cmd {
	topic, part, ok := m.selectedPartition()
	if !ok {
		return core.NewNotification(core.StatusWarning, "Move replica", "no partition selected")
	}
	// Offer the broker's other log directories as targets.
	var targets []string
	for i, d := range m.logDirs {
		if i == m.expanded {
			continue
		}
		targets = append(targets, d.Path)
	}
	def := ""
	if len(targets) > 0 {
		def = targets[0]
	}
	m.moveForm = form.New([]form.Field{
		{Name: "topic", Label: "Topic", Type: form.Text, Default: topic},
		{Name: "partition", Label: "Partition", Type: form.Text, Default: strconv.FormatInt(int64(part), 10)},
		{Name: "logdir", Label: "Target log dir", Type: form.Text, Required: true, Default: def, Options: targets},
	})
	m.moveForm.SetDimensions(m.dims.Width, m.dims.Height)
	return m.moveForm.Focus()
}

func (m *Model) handleMoveSubmit(msg form.FormSubmitMsg) tea.Cmd {
	m.moveForm = nil
	topic := msg.Values["topic"]
	logDir := msg.Values["logdir"]
	part, _ := strconv.ParseInt(msg.Values["partition"], 10, 32)
	partition := int32(part)
	id := m.brokerID
	ds := m.common.DataSource
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Move replica",
			Message:      fmt.Sprintf("Move %s-%d to %s?", topic, partition, logDir),
			ConfirmLabel: "Move",
			OnConfirm: func() tea.Msg {
				err := ds.AlterReplicaLogDir(id, topic, partition, logDir)
				return replicaMovedMsg{brokerID: id, topic: topic, partition: partition, logDir: logDir, err: err}
			},
		}
	}
}

func (m *Model) handleReplicaMoved(v replicaMovedMsg) tea.Cmd {
	if v.err != nil {
		return func() tea.Msg { return shared.NewUIError("reassign", "Replica move failed", v.err) }
	}
	m.logDirsLoaded = false
	m.expanded = -1
	return tea.Batch(core.NewNotification(core.StatusSuccess, "Replica moved", fmt.Sprintf("%s-%d → %s", v.topic, v.partition, v.logDir)), m.loadLogDirs())
}

// --- Configs tab (BR-15/BR-16) ---

// diskUsage sums partition sizes and counts across the loaded log dirs; ok is
// false until they have loaded successfully with data.
func (m *Model) diskUsage() (size int64, count int, ok bool) {
	if !m.logDirsLoaded || m.logDirsErr != nil || len(m.logDirs) == 0 {
		return 0, 0, false
	}
	for _, d := range m.logDirs {
		for _, t := range d.Topics {
			for _, p := range t.Partitions {
				size += p.Size
				count++
			}
		}
	}
	return size, count, true
}

func (m *Model) renderLogDirs() string {
	if !m.logDirsLoaded {
		return m.common.Styles.Muted.Render("Loading log directories…")
	}
	if m.logDirsErr != nil {
		return m.common.Styles.Error.Render("Error: "+m.logDirsErr.Error()) + "\n" + m.common.Styles.Muted.Render("Press r to retry.")
	}
	if len(m.logDirs) == 0 {
		return m.common.Styles.Muted.Render("Log dir data not available")
	}
	var b strings.Builder
	b.WriteString(stylesPkg.FrameTable(m.logTable.View()))
	if m.expanded >= 0 && m.expanded < len(m.logDirs) {
		b.WriteString("\n\n")
		b.WriteString(m.common.Styles.Header.Render("Partitions in " + m.logDirs[m.expanded].Path))
		b.WriteString("\n")
		b.WriteString(stylesPkg.FrameTable(m.partTable.View()))
		b.WriteString("\n")
		b.WriteString(m.common.Styles.Muted.Render(keys.Hint(keys.ScopeListContent, keys.ActionActivate, "collapse") + "  (move a replica from the actions menu)"))
	} else {
		b.WriteString("\n")
		b.WriteString(m.common.Styles.Muted.Render(keys.Hint(keys.ScopeListContent, keys.ActionActivate, "expand directory")))
	}
	return b.String()
}
