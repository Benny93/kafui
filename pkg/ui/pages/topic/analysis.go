package topic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AnalysisLoadedMsg carries a GetTopicAnalysis result (TP-31).
type AnalysisLoadedMsg struct {
	Topic    string
	Analysis *api.TopicAnalysis
	Err      error
}

// analysisTickMsg drives the ~1s poll while an analysis is running.
type analysisTickMsg struct{ Topic string }

// fetchAnalysis loads the current analysis state for a topic.
func fetchAnalysis(ds api.KafkaDataSource, topic string) tea.Cmd {
	return func() tea.Msg {
		a, err := ds.GetTopicAnalysis(topic)
		return AnalysisLoadedMsg{Topic: topic, Analysis: a, Err: err}
	}
}

// handleShowAnalysis opens the analysis overlay and fetches current state.
func (k *Keys) handleShowAnalysis(model *Model) tea.Cmd {
	model.showAnalysis = true
	model.analysisLoading = true
	model.markRenderDirty()
	return fetchAnalysis(model.dataSource, model.topicName)
}

// handleAnalysisKey handles keys while the analysis overlay is open.
func (k *Keys) handleAnalysisKey(model *Model, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "q":
		model.showAnalysis = false
		model.markRenderDirty()
		return nil
	case "s", "enter":
		// Start or restart analysis, guarded by a confirmation (full scan can be expensive).
		return k.confirmStartAnalysis(model)
	case "x":
		// Cancel a running analysis.
		if model.analysis != nil && model.analysis.State == api.AnalysisRunning {
			ds := model.dataSource
			topic := model.topicName
			return func() tea.Msg {
				_ = ds.CancelTopicAnalysis(topic)
				return fetchAnalysis(ds, topic)()
			}
		}
		return nil
	}
	return nil
}

// confirmStartAnalysis asks for confirmation before starting a full-topic scan.
func (k *Keys) confirmStartAnalysis(model *Model) tea.Cmd {
	ds := model.dataSource
	topic := model.topicName
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Start topic analysis",
			Message:      fmt.Sprintf("Analyse %q? This performs a full scan from earliest to the current end offsets and may be expensive on large topics.", topic),
			ConfirmLabel: "Start",
			OnConfirm: func() tea.Msg {
				if err := ds.StartTopicAnalysis(context.Background(), topic); err != nil {
					return AnalysisLoadedMsg{Topic: topic, Err: err}
				}
				return fetchAnalysis(ds, topic)()
			},
		}
	}
}

// handleAnalysisLoaded stores the analysis state and schedules a poll while running.
func (h *Handlers) handleAnalysisLoaded(model *Model, msg AnalysisLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Topic != model.topicName {
		return model, nil
	}
	model.analysisLoading = false
	if msg.Err != nil {
		return model, core.NotifyError("Analysis failed", msg.Err)
	}
	model.analysis = msg.Analysis
	model.markRenderDirty()
	// Only schedule the poll tick while running.
	if model.analysis != nil && model.analysis.State == api.AnalysisRunning {
		topic := model.topicName
		return model, tea.Tick(time.Second, func(time.Time) tea.Msg {
			return analysisTickMsg{Topic: topic}
		})
	}
	return model, nil
}

// handleAnalysisTick re-fetches analysis state while running and the overlay is open.
func (h *Handlers) handleAnalysisTick(model *Model, msg analysisTickMsg) (tea.Model, tea.Cmd) {
	if msg.Topic != model.topicName || !model.showAnalysis {
		return model, nil
	}
	if model.analysis == nil || model.analysis.State != api.AnalysisRunning {
		return model, nil
	}
	return model, fetchAnalysis(model.dataSource, model.topicName)
}

// renderAnalysisOverlay renders the four analysis states (TP-31).
func (m *Model) renderAnalysisOverlay(width int) string {
	muted := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)
	header := lipgloss.NewStyle().Foreground(stylesPkg.Primary).Bold(true)
	var b strings.Builder
	b.WriteString(header.Render("Statistics: " + m.topicName))
	b.WriteString("\n\n")

	if m.analysisLoading && m.analysis == nil {
		b.WriteString(muted.Render("Loading analysis…"))
		return b.String()
	}

	a := m.analysis
	switch {
	case a == nil:
		// Never analyzed.
		b.WriteString(muted.Render("This topic has not been analysed yet."))
		b.WriteString("\n\n")
		b.WriteString(muted.Render("s: start analysis • esc: close"))
	case a.State == api.AnalysisRunning:
		p := a.Progress
		b.WriteString(fmt.Sprintf("  Running… %.1f%%\n", p.Percentage()))
		if !p.StartTime.IsZero() {
			b.WriteString(fmt.Sprintf("  Elapsed:  %s\n", time.Since(p.StartTime).Round(time.Second)))
		}
		b.WriteString(fmt.Sprintf("  Scanned:  %d messages, %s\n", p.MessagesScanned, shared.FormatBytes2dp(p.BytesScanned)))
		b.WriteString("\n")
		b.WriteString(muted.Render("x: cancel • esc: close"))
	case a.State == api.AnalysisFailed:
		errStyle := lipgloss.NewStyle().Foreground(stylesPkg.Error).Bold(true)
		b.WriteString(errStyle.Render("Analysis failed: " + a.Err))
		b.WriteString("\n")
		if !a.ErrAt.IsZero() {
			b.WriteString(muted.Render("  at " + shared.FormatTimestamp(a.ErrAt)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(muted.Render("s: restart • esc: close"))
	default: // completed
		b.WriteString(m.renderAnalysisResult(a))
	}
	return b.String()
}

func (m *Model) renderAnalysisResult(a *api.TopicAnalysis) string {
	muted := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)
	if a.Result == nil {
		return muted.Render("No result payload.")
	}
	r := a.Result
	var b strings.Builder
	if r.MessageCount == 0 {
		b.WriteString(muted.Render("This topic appears to be empty (0 messages)."))
		b.WriteString("\n\n")
		b.WriteString(muted.Render("s: restart • esc: close"))
		return b.String()
	}
	b.WriteString(fmt.Sprintf("  Messages:        %d\n", r.MessageCount))
	b.WriteString(fmt.Sprintf("  Offset range:    %d … %d\n", r.MinOffset, r.MaxOffset))
	b.WriteString(fmt.Sprintf("  Time range:      %s … %s\n", shared.FormatTimestamp(r.MinTimestamp), shared.FormatTimestamp(r.MaxTimestamp)))
	b.WriteString(fmt.Sprintf("  Null keys:       %d\n", r.NullKeys))
	b.WriteString(fmt.Sprintf("  Null values:     %d\n", r.NullValues))
	b.WriteString(fmt.Sprintf("  Distinct keys:   ~%d\n", r.ApproxDistinctKeys))
	b.WriteString(fmt.Sprintf("  Distinct values: ~%d\n", r.ApproxDistinctValues))
	b.WriteString("\n")
	b.WriteString(muted.Render("  Key size:   ") + sizeDistLine(r.KeySize) + "\n")
	b.WriteString(muted.Render("  Value size: ") + sizeDistLine(r.ValueSize) + "\n")
	b.WriteString("\n")
	if len(r.Partitions) > 0 {
		// ponytail: per-partition rows are shown read-only; the spec's optional
		// "Enter to drill into per-partition stats" is deferred until the analysis
		// engine exposes a partition-scoped result (PartitionAnalysis has no
		// distribution/timestamp payload yet).
		b.WriteString(muted.Render(fmt.Sprintf("  %-6s %-12s %-12s %-12s", "Part", "Messages", "MinOffset", "MaxOffset")))
		b.WriteString("\n")
		for _, p := range r.Partitions {
			b.WriteString(fmt.Sprintf("  %-6d %-12d %-12d %-12d\n", p.Partition, p.MessageCount, p.MinOffset, p.MaxOffset))
		}
		b.WriteString("\n")
	}
	if !r.CompletedAt.IsZero() {
		b.WriteString(muted.Render("  Completed " + shared.FormatTimestamp(r.CompletedAt)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(muted.Render("s: restart • esc: close"))
	return b.String()
}

func sizeDistLine(d api.SizeDistribution) string {
	return fmt.Sprintf("min %s / avg %.0fB / max %s / p95 %s / p99 %s",
		shared.FormatBytes2dp(d.Min), d.Avg, shared.FormatBytes2dp(d.Max),
		shared.FormatBytes2dp(d.P95), shared.FormatBytes2dp(d.P99))
}
