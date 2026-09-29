package topic

import (
	"fmt"
	"github.com/Benny93/kafui/pkg/ui/keys"
	"strconv"
	"strings"
	"sync"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TopicDetailsLoadedMsg carries the result of the overview fetch (TP-23):
// GetTopicDetails + GetTopicSizes for a single topic.
type TopicDetailsLoadedMsg struct {
	Topic   string
	Details api.TopicDetails
	Size    int64
	Err     error
}

// fetchTopicOverview fetches partition detail and on-disk size for a topic.
// The two calls are independent round trips, so they run concurrently.
func fetchTopicOverview(ds api.KafkaDataSource, topic string) tea.Cmd {
	return func() tea.Msg {
		var (
			wg    sync.WaitGroup
			sizes map[string]int64
			serr  error
		)
		wg.Add(1)
		go func() {
			defer wg.Done()
			sizes, serr = ds.GetTopicSizes([]string{topic})
		}()
		details, err := ds.GetTopicDetails(topic)
		wg.Wait()

		var size int64
		if err == nil && serr == nil {
			size = sizes[topic]
		}
		return TopicDetailsLoadedMsg{Topic: topic, Details: details, Size: size, Err: err}
	}
}

// topicMetaLoadedMsg carries the topic metadata fetched when the page was
// opened without it (from a consumer group, connector, schema or deep link).
type topicMetaLoadedMsg struct {
	Topic   string
	Details api.TopicDetails
	Err     error
}

// fetchTopicMeta fetches partition metadata to fill in the page's topicDetails.
func fetchTopicMeta(ds api.KafkaDataSource, topic string) tea.Cmd {
	return func() tea.Msg {
		details, err := ds.GetTopicDetails(topic)
		return topicMetaLoadedMsg{Topic: topic, Details: details, Err: err}
	}
}

// handleTopicMetaLoaded fills in the partition count, replication factor and
// message count the navigation did not supply. A failure only leaves them
// unknown; the message fetch reports its own errors.
func (h *Handlers) handleTopicMetaLoaded(model *Model, msg topicMetaLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Topic != model.topicName {
		return model, nil
	}
	if msg.Err != nil {
		shared.Log.Warn("topic metadata unavailable", "topic", model.topicName, "err", msg.Err)
		return model, nil
	}
	model.topicDetails.NumPartitions = int32(len(msg.Details.Partitions))
	model.topicDetails.ReplicationFactor = msg.Details.ReplicationFactor
	model.topicDetails.MessageCount = msg.Details.MessageCount()
	model.markRenderDirty()
	return model, nil
}

// handleShowOverview opens the overview overlay and fetches partition detail.
func (k *Keys) handleShowOverview(model *Model) tea.Cmd {
	model.showOverview = true
	model.overviewLoading = true
	model.overviewErr = nil
	model.partitionCursor = 0
	model.markRenderDirty()
	return fetchTopicOverview(model.dataSource, model.topicName)
}

// handleOverviewKey handles keys while the overview overlay is open.
func (k *Keys) handleOverviewKey(model *Model, msg tea.KeyMsg) tea.Cmd {
	action, bound := keys.Default.Resolve(keys.ScopeOverlay, msg.String())
	if !bound {
		return nil
	}
	switch action {
	case keys.ActionCancel:
		model.showOverview = false
		model.markRenderDirty()
		return nil
	case keys.ActionRefresh:
		// Retry / refresh the overview.
		return k.handleShowOverview(model)
	case keys.ActionUp:
		if model.partitionCursor > 0 {
			model.partitionCursor--
			model.markRenderDirty()
		}
		return nil
	case keys.ActionDown:
		if model.overview != nil && model.partitionCursor < len(model.overview.Partitions)-1 {
			model.partitionCursor++
			model.markRenderDirty()
		}
		return nil
	case keys.ActionDelete:
		// Per-partition purge (TP-27) on the highlighted partition row.
		if model.overview == nil || model.partitionCursor >= len(model.overview.Partitions) {
			return nil
		}
		pid := model.overview.Partitions[model.partitionCursor].ID
		return k.confirmPurgePartition(model, pid)
	}
	return nil
}

// handleTopicDetailsLoaded stores the fetched overview details.
func (h *Handlers) handleTopicDetailsLoaded(model *Model, msg TopicDetailsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Topic != model.topicName {
		return model, nil
	}
	model.overviewLoading = false
	if msg.Err != nil {
		model.overview = nil
		model.overviewErr = msg.Err
	} else {
		d := msg.Details
		model.overview = &d
		model.overviewSize = msg.Size
		model.overviewErr = nil
	}
	if model.overview != nil && model.partitionCursor >= len(model.overview.Partitions) {
		model.partitionCursor = 0
	}
	model.markRenderDirty()
	return model, nil
}

// healthStyle returns the semantic style for a replication-health indicator:
// Success (green) when fully replicated, Error (red) when under-replicated.
func healthStyle(underReplicated int) lipgloss.Style {
	if underReplicated == 0 {
		return lipgloss.NewStyle().Foreground(stylesPkg.Success)
	}
	return lipgloss.NewStyle().Foreground(stylesPkg.Error)
}

// cleanupPolicy returns the topic's cleanup.policy from the known config, or "delete".
func (m *Model) cleanupPolicy() string {
	if p, ok := m.topicDetails.ConfigEntries["cleanup.policy"]; ok && p != nil {
		return *p
	}
	return "delete"
}

// renderOverviewOverlay renders the overview + partition table (TP-23).
func (m *Model) renderOverviewOverlay(width int) string {
	muted := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)
	header := lipgloss.NewStyle().Foreground(stylesPkg.Primary).Bold(true)
	var b strings.Builder

	b.WriteString(header.Render("Overview: " + m.topicName))
	b.WriteString("\n\n")

	if m.overviewLoading {
		b.WriteString(muted.Render("Loading topic details…"))
		return b.String()
	}
	if m.overviewErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(stylesPkg.Error).Bold(true)
		b.WriteString(errStyle.Render("Failed to load topic: " + m.overviewErr.Error()))
		b.WriteString("\n\n")
		b.WriteString(muted.Render(keys.Hint(keys.ScopeOverlay, keys.ActionRefresh, "retry", keys.ActionCancel, "close")))
		return b.String()
	}
	if m.overview == nil {
		b.WriteString(muted.Render("No details available."))
		return b.String()
	}

	d := m.overview
	label := "external"
	if d.IsInternal {
		label = "internal"
	}
	hs := healthStyle(d.UnderReplicatedPartitions)

	b.WriteString(fmt.Sprintf("  Partitions:          %d\n", len(d.Partitions)))
	b.WriteString(fmt.Sprintf("  Replication factor:  %d\n", d.ReplicationFactor))
	b.WriteString("  Under-replicated:    " + hs.Render(strconv.Itoa(d.UnderReplicatedPartitions)) + "\n")
	b.WriteString("  ISR / replicas:      " + hs.Render(fmt.Sprintf("%d/%d", d.InSyncReplicas, d.TotalReplicas)) + "\n")
	b.WriteString(fmt.Sprintf("  Type:                %s\n", label))
	b.WriteString(fmt.Sprintf("  Total size:          %s\n", shared.FormatBytes2dp(m.overviewSize)))
	b.WriteString(fmt.Sprintf("  Cleanup policy:      %s\n", m.cleanupPolicy()))
	b.WriteString(fmt.Sprintf("  Messages:            %d\n", d.MessageCount()))
	b.WriteString("\n")

	// Partition table.
	b.WriteString(muted.Render(fmt.Sprintf("  %-6s %-24s %-12s %-12s %-10s", "ID", "Replicas (*=leader)", "Earliest", "Next", "Messages")))
	b.WriteString("\n")
	for i, p := range d.Partitions {
		line := fmt.Sprintf("  %-6d %-24s %-12d %-12d %-10d",
			p.ID, renderReplicas(p), p.EarliestOffset, p.LatestOffset, p.MessageCount())
		if i == m.partitionCursor {
			line = lipgloss.NewStyle().Foreground(stylesPkg.BgBase).Background(stylesPkg.Primary).Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(muted.Render(keys.Hint(keys.ScopeOverlay, keys.ActionUp, "select partition", keys.ActionDelete, "clear partition", keys.ActionRefresh, "refresh", keys.ActionCancel, "close")))
	return b.String()
}

// renderReplicas renders the replica list, marking the leader with * and
// highlighting out-of-sync replicas (in Replicas but not ISR) in Error style.
func renderReplicas(p api.PartitionInfo) string {
	inSync := make(map[int32]bool, len(p.ISR))
	for _, r := range p.ISR {
		inSync[r] = true
	}
	errStyle := lipgloss.NewStyle().Foreground(stylesPkg.Error)
	parts := make([]string, 0, len(p.Replicas))
	for _, r := range p.Replicas {
		s := strconv.Itoa(int(r))
		if r == p.Leader {
			s = "*" + s
		}
		if !inSync[r] {
			s = errStyle.Render(s)
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ",")
}
