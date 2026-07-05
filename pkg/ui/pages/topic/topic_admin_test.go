package topic

import (
	"context"
	"strings"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	formpkg "github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// spyDataSource wraps the full mock datasource and records the mutation calls
// exercised by the topic-admin tests.
type spyDataSource struct {
	*mock.KafkaDataSourceMock
	increaseCalls    []int32
	deleteCalls      []string
	purgeCalls       [][2]interface{} // {name, partition}
	updateCalls      []map[string]*string
	replicationCalls []int16
}

func newSpy() *spyDataSource {
	m := &mock.KafkaDataSourceMock{}
	m.Init("")
	return &spyDataSource{KafkaDataSourceMock: m}
}

func (s *spyDataSource) IncreasePartitions(name string, totalCount int32) error {
	s.increaseCalls = append(s.increaseCalls, totalCount)
	return s.KafkaDataSourceMock.IncreasePartitions(name, totalCount)
}
func (s *spyDataSource) DeleteTopic(name string) error {
	s.deleteCalls = append(s.deleteCalls, name)
	return s.KafkaDataSourceMock.DeleteTopic(name)
}
func (s *spyDataSource) PurgeTopicMessages(name string, partition int32) error {
	s.purgeCalls = append(s.purgeCalls, [2]interface{}{name, partition})
	return s.KafkaDataSourceMock.PurgeTopicMessages(name, partition)
}
func (s *spyDataSource) UpdateTopicConfig(name string, entries map[string]*string) error {
	s.updateCalls = append(s.updateCalls, entries)
	return s.KafkaDataSourceMock.UpdateTopicConfig(name, entries)
}
func (s *spyDataSource) ChangeReplicationFactor(name string, f int16) error {
	s.replicationCalls = append(s.replicationCalls, f)
	return s.KafkaDataSourceMock.ChangeReplicationFactor(name, f)
}

// run executes a tea.Cmd to completion and returns its message (nil-safe).
func run(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

// --- TP-23: health styling + message count summation ---

func TestHealthStyle(t *testing.T) {
	assert.Equal(t, stylesPkg.Success, healthStyle(0).GetForeground(), "fully replicated → green")
	assert.Equal(t, stylesPkg.Error, healthStyle(2).GetForeground(), "under-replicated → red")
}

func TestOverviewMessageCountSummation(t *testing.T) {
	m := NewModel(newSpy(), "user-events", api.Topic{NumPartitions: 2})
	m.overview = &api.TopicDetails{
		Partitions: []api.PartitionInfo{
			{ID: 0, EarliestOffset: 0, LatestOffset: 100},
			{ID: 1, EarliestOffset: 10, LatestOffset: 60},
		},
	}
	out := m.renderOverviewOverlay(120)
	assert.Contains(t, out, "150", "message count is Σ(latest-earliest) = 100 + 50")
}

func TestRenderReplicasMarksLeaderAndOutOfSync(t *testing.T) {
	p := api.PartitionInfo{Leader: 1, Replicas: []int32{1, 2, 3}, ISR: []int32{1, 2}}
	out := renderReplicas(p)
	assert.Contains(t, out, "*1", "leader marked with *")
	// replica 3 is out of sync (in Replicas, not in ISR) → styled, but still present.
	assert.Contains(t, out, "3")
}

// --- TP-24: config row rendering ---

func TestSettingsRendering(t *testing.T) {
	m := NewModel(newSpy(), "user-events", api.Topic{})
	m.settingsConfig = []api.TopicConfigEntry{
		{Name: "retention.ms", Value: "-1", Default: "604800000"},                    // unbounded + override
		{Name: "max.message.bytes", Value: "1048576", Default: "1048576"},            // no override, byte format
		{Name: "sasl.jaas.config", Value: "secret", Sensitive: true, Default: "sec"}, // masked
	}
	out := m.renderSettingsOverlay(120)
	assert.Contains(t, out, "unbounded", "non-positive .ms rendered as unbounded")
	assert.Contains(t, out, "1.00 MB", "byte size rendered human-readable")
	assert.Contains(t, out, shared.ConfigValueMask, "sensitive value masked")
	assert.NotContains(t, out, "secret", "raw sensitive value never shown")
}

func TestSettingsEmptyStateNotError(t *testing.T) {
	m := NewModel(newSpy(), "user-events", api.Topic{})
	m.settingsLoading = false
	m.settingsConfig = nil
	out := m.renderSettingsOverlay(120)
	assert.Contains(t, strings.ToLower(out), "permission", "empty list → empty state, not an error")
	assert.NotContains(t, strings.ToLower(out), "failed")
}

func TestIsOverride(t *testing.T) {
	assert.True(t, isOverride(api.TopicConfigEntry{Value: "x", Default: "y"}))
	assert.False(t, isOverride(api.TopicConfigEntry{Value: "x", Default: "x"}))
	assert.False(t, isOverride(api.TopicConfigEntry{Value: "x", Default: ""}), "no default → not an override")
}

// --- TP-25: diff builder ---

func TestDiffConfigChanges(t *testing.T) {
	loaded := []api.TopicConfigEntry{
		{Name: "cleanup.policy", Value: "delete", Default: "delete"},
		{Name: "retention.ms", Value: "604800000", Default: "604800000"},
		{Name: "max.message.bytes", Value: "1048576", Default: "1048576"},
	}
	values := map[string]string{
		"cleanup.policy":    "delete",     // unchanged
		"retention.ms":      "1000",       // changed
		"max.message.bytes": "1048576",    // unchanged
		"min.insync.replicas": "",         // new empty custom → excluded
	}
	changes := diffConfigChanges(loaded, values)
	require.Len(t, changes, 1, "only the changed entry is submitted")
	require.Contains(t, changes, "retention.ms")
	require.NotNil(t, changes["retention.ms"])
	assert.Equal(t, "1000", *changes["retention.ms"])
}

func TestSettingsFormSubmitCallsUpdateWithDiff(t *testing.T) {
	spy := newSpy()
	m := NewModel(spy, "user-events", api.Topic{})
	m.showSettingsEdit = true
	m.loadedConfig = []api.TopicConfigEntry{
		{Name: "retention.ms", Value: "604800000", Default: "604800000"},
	}
	_, cmd := m.handlers.handleSettingsFormSubmit(m, map[string]string{"retention.ms": "1000"})
	run(cmd) // executes UpdateTopicConfig
	require.Len(t, spy.updateCalls, 1)
	require.Contains(t, spy.updateCalls[0], "retention.ms")
	assert.Equal(t, "1000", *spy.updateCalls[0]["retention.ms"])
	assert.False(t, m.showSettingsEdit, "overlay closes after submit")
}

func TestSettingsFormSubmitNoChangesSkipsUpdate(t *testing.T) {
	spy := newSpy()
	m := NewModel(spy, "user-events", api.Topic{})
	m.loadedConfig = []api.TopicConfigEntry{{Name: "retention.ms", Value: "604800000"}}
	_, cmd := m.handlers.handleSettingsFormSubmit(m, map[string]string{"retention.ms": "604800000"})
	run(cmd)
	assert.Empty(t, spy.updateCalls, "no datasource call when nothing changed")
}

// --- TP-26: partition increase / replication ---

func TestIncreasePartitionsConfirmChain(t *testing.T) {
	spy := newSpy()
	m := NewModel(spy, "user-events", api.Topic{NumPartitions: 6})
	m.keys.handleIncreasePartitionsDialog(m)
	_, cmd := m.handlers.handleMutationFormSubmit(m, map[string]string{"value": "12"})
	// The submit produces a ShowConfirmMsg; run its OnConfirm to reach the datasource.
	cm := run(cmd).(core.ShowConfirmMsg)
	assert.Empty(t, spy.increaseCalls, "no call before confirmation")
	mut := run(cm.OnConfirm).(topicMutationMsg)
	assert.NoError(t, mut.Err)
	require.Equal(t, []int32{12}, spy.increaseCalls)
}

func TestIncreasePartitionsRejectsDecrease(t *testing.T) {
	spy := newSpy()
	m := NewModel(spy, "user-events", api.Topic{NumPartitions: 6})
	m.keys.handleIncreasePartitionsDialog(m)
	_, cmd := m.handlers.handleMutationFormSubmit(m, map[string]string{"value": "2"})
	cm := run(cmd).(core.ShowConfirmMsg)
	mut := run(cm.OnConfirm).(topicMutationMsg)
	require.Error(t, mut.Err)
	var decErr api.PartitionDecreaseError
	assert.ErrorAs(t, mut.Err, &decErr, "decrease surfaced as PartitionDecreaseError")
	assert.False(t, m.showMutationForm, "form closed, no state corruption")
}

func TestMutationCancelMakesNoCall(t *testing.T) {
	spy := newSpy()
	m := NewModel(spy, "user-events", api.Topic{NumPartitions: 6})
	m.keys.handleIncreasePartitionsDialog(m)
	// Cancel via the form cancel message routed through Handle.
	m.Update(formCancel())
	assert.Empty(t, spy.increaseCalls)
	assert.False(t, m.showMutationForm)
}

// --- TP-27: delete + per-partition purge ---

func TestDeleteEmitsConfirmAndCallsDelete(t *testing.T) {
	spy := newSpy()
	// Use a throwaway topic: DeleteTopic mutates the shared mock state.
	const topic = "dead-letter-queue"
	m := NewModel(spy, topic, api.Topic{})
	cmd := m.keys.handleDeleteTopic(m)
	cm := run(cmd).(core.ShowConfirmMsg)
	assert.True(t, cm.Danger)
	assert.Empty(t, spy.deleteCalls, "no delete before confirm")
	out := run(cm.OnConfirm).(topicMutationMsg)
	require.NoError(t, out.Err)
	require.Equal(t, []string{topic}, spy.deleteCalls)
	assert.True(t, out.Back, "delete navigates back to the list")
}

func TestDeleteInternalTopicBlocked(t *testing.T) {
	spy := newSpy()
	m := NewModel(spy, "__consumer_offsets", api.Topic{})
	cmd := m.keys.handleDeleteTopic(m)
	msg := run(cmd)
	_, isConfirm := msg.(core.ShowConfirmMsg)
	assert.False(t, isConfirm, "internal topic shows a hint, not a modal")
	assert.Empty(t, spy.deleteCalls)
}

func TestDeleteDisabledOnCluster(t *testing.T) {
	spy := newSpy()
	spy.SetDeletionDisabled(true)
	defer spy.SetDeletionDisabled(false)
	m := NewModel(spy, "audit-log", api.Topic{})
	msg := run(m.keys.handleDeleteTopic(m))
	_, isConfirm := msg.(core.ShowConfirmMsg)
	assert.False(t, isConfirm, "deletion disabled → hint, not a modal")
	assert.Empty(t, spy.deleteCalls)
}

func TestPerPartitionPurgePassesHighlightedID(t *testing.T) {
	spy := newSpy()
	m := NewModel(spy, "user-events", api.Topic{})
	m.showOverview = true
	m.overview = &api.TopicDetails{Partitions: []api.PartitionInfo{{ID: 0}, {ID: 1}, {ID: 2}}}
	m.partitionCursor = 2
	cmd := m.keys.handleOverviewKey(m, keyMsg("x"))
	cm := run(cmd).(core.ShowConfirmMsg)
	run(cm.OnConfirm)
	require.Len(t, spy.purgeCalls, 1)
	assert.Equal(t, "user-events", spy.purgeCalls[0][0])
	assert.Equal(t, int32(2), spy.purgeCalls[0][1], "purge targets the highlighted partition")
}

// --- TP-31: analysis states ---

func TestAnalysisNeverAnalyzedRendersStart(t *testing.T) {
	m := NewModel(newSpy(), "user-events", api.Topic{})
	m.analysis = nil
	m.analysisLoading = false
	out := m.renderAnalysisOverlay(120)
	assert.Contains(t, strings.ToLower(out), "not been analysed")
	assert.Contains(t, strings.ToLower(out), "start analysis")
}

func TestAnalysisCompletedRendersTotals(t *testing.T) {
	spy := newSpy()
	require.NoError(t, spy.StartTopicAnalysis(context.Background(), "user-events"))
	a, err := spy.GetTopicAnalysis("user-events")
	require.NoError(t, err)
	require.NotNil(t, a)
	m := NewModel(spy, "user-events", api.Topic{})
	m.analysis = a
	out := m.renderAnalysisOverlay(120)
	assert.Contains(t, out, "Messages:")
	assert.Contains(t, out, "Offset range:")
}

func TestAnalysisEmptyTopicMessage(t *testing.T) {
	m := NewModel(newSpy(), "user-events", api.Topic{})
	m.analysis = &api.TopicAnalysis{
		State:  api.AnalysisCompleted,
		Result: &api.TopicAnalysisResult{MessageCount: 0},
	}
	out := m.renderAnalysisOverlay(120)
	assert.Contains(t, strings.ToLower(out), "appears to be empty")
}

func TestAnalysisPollScheduledOnlyWhileRunning(t *testing.T) {
	m := NewModel(newSpy(), "user-events", api.Topic{})
	m.showAnalysis = true

	_, runningCmd := m.handlers.handleAnalysisLoaded(m, AnalysisLoadedMsg{
		Topic: "user-events", Analysis: &api.TopicAnalysis{State: api.AnalysisRunning},
	})
	assert.NotNil(t, runningCmd, "running state schedules a poll tick")

	_, doneCmd := m.handlers.handleAnalysisLoaded(m, AnalysisLoadedMsg{
		Topic: "user-events", Analysis: &api.TopicAnalysis{State: api.AnalysisCompleted},
	})
	assert.Nil(t, doneCmd, "completed state schedules no further tick")
}

// --- test helpers ---

func formCancel() tea.Msg { return formpkg.FormCancelMsg{} }

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}
