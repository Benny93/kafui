package consumergroup

import (
	"context"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/core"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// spyDS records mutation calls so tests can assert they happen exactly once and
// only after confirmation.
type spyDS struct {
	*mock.KafkaDataSourceMock
	deleteCalls int
	resetCalls  int
}

func (s *spyDS) DeleteConsumerGroup(id string) error {
	s.deleteCalls++
	return s.KafkaDataSourceMock.DeleteConsumerGroup(id)
}

func (s *spyDS) ResetConsumerGroupOffsets(ctx context.Context, req api.OffsetResetRequest) error {
	s.resetCalls++
	return s.KafkaDataSourceMock.ResetConsumerGroupOffsets(ctx, req)
}

func newSpy() *spyDS {
	m := &mock.KafkaDataSourceMock{}
	m.Init("")
	return &spyDS{KafkaDataSourceMock: m}
}

func newPage(t *testing.T, ds api.KafkaDataSource, groupID string) *Model {
	t.Helper()
	common := core.NewCommon(ds)
	m, ok := NewModelWithCommon(common, groupID).(*Model)
	require.True(t, ok)
	m.SetDimensions(120, 40)
	return m
}

// loadInto runs the detail load command and feeds the result into the model.
func loadInto(m *Model) {
	msg := m.loadDetail()()
	m.handle(msg)
}

func TestDetail_FoundAndTopicGrouping(t *testing.T) {
	m := newPage(t, newSpy(), "order-processor")
	loadInto(m)

	assert.True(t, m.loaded)
	assert.False(t, m.notFound)
	require.Len(t, m.topicRows, 1)
	assert.Equal(t, "order-events", m.topicRows[0].topic)
	assert.Len(t, m.topicRows[0].partitions, 3)
	// Aggregate lag = 50 + 50 + 0 = 100.
	require.NotNil(t, m.topicRows[0].aggLag)
	assert.Equal(t, int64(100), *m.topicRows[0].aggLag)
}

func TestDetail_NotFound(t *testing.T) {
	m := newPage(t, newSpy(), "does-not-exist")
	loadInto(m)
	assert.True(t, m.loaded)
	assert.True(t, m.notFound)
}

func TestDetail_ExpandCollapse(t *testing.T) {
	m := newPage(t, newSpy(), "order-processor")
	loadInto(m)

	assert.Equal(t, -1, m.expanded)
	m.toggleExpand()
	assert.Equal(t, 0, m.expanded)
	require.Len(t, m.partTable.Rows(), 3)
	m.toggleExpand()
	assert.Equal(t, -1, m.expanded)
}

func TestDetail_UncommittedPartitionRow(t *testing.T) {
	m := newPage(t, newSpy(), "analytics-consumer")
	loadInto(m)
	require.Len(t, m.topicRows, 1)
	m.toggleExpand()
	// Partition 1 is assigned but uncommitted: committed cell empty, member shown.
	var found bool
	for _, p := range m.topicRows[0].partitions {
		if p.Partition == 1 {
			found = true
			assert.Nil(t, p.CommittedOffset)
			assert.Nil(t, p.Lag)
			assert.NotEmpty(t, p.MemberID)
		}
	}
	assert.True(t, found)
}

func TestResetForm_Gating(t *testing.T) {
	m := newPage(t, newSpy(), "inventory-sync")
	loadInto(m)
	require.Nil(t, m.openResetForm())
	require.NotNil(t, m.resetForm)

	// Submit with no partition selected is blocked and records an error.
	cmd := m.resetForm.submit()
	assert.Nil(t, cmd)
	assert.NotEmpty(t, m.resetForm.errMsg)
	assert.Equal(t, api.OffsetResetEarliest, m.resetForm.currentMode()) // default type
}

func TestResetForm_TopicChangeClearsState(t *testing.T) {
	m := newPage(t, newSpy(), "inventory-sync")
	loadInto(m)
	m.openResetForm()
	m.resetForm.selected[0] = true
	m.resetForm.offsetInputs[0] = m.resetForm.tsInput // any non-empty entry
	m.resetForm.clearTopicState()
	assert.Empty(t, m.resetForm.selected)
	assert.Empty(t, m.resetForm.offsetInputs)
}

func TestResetForm_TimestampRequiresValue(t *testing.T) {
	m := newPage(t, newSpy(), "inventory-sync")
	loadInto(m)
	m.openResetForm()
	f := m.resetForm
	// Switch to timestamp mode.
	for f.currentMode() != api.OffsetResetTimestamp {
		f.modeIdx = (f.modeIdx + 1) % len(resetModes)
	}
	assert.True(t, f.hasConditional())
	f.selected[0] = true
	// No timestamp entered → blocked.
	assert.Nil(t, f.submit())
	assert.NotEmpty(t, f.errMsg)
}

func TestResetForm_SubmitEmitsConfirmThenResets(t *testing.T) {
	spy := newSpy()
	m := newPage(t, spy, "inventory-sync")
	loadInto(m)
	m.openResetForm()
	m.resetForm.selected[0] = true // Earliest mode by default

	// Submit produces a resetFormSubmitMsg with the built request.
	submitCmd := m.resetForm.submit()
	require.NotNil(t, submitCmd)
	sub, ok := submitCmd().(resetFormSubmitMsg)
	require.True(t, ok)
	assert.Equal(t, api.OffsetResetEarliest, sub.req.Mode)
	assert.Equal(t, []int32{0}, sub.req.Partitions)

	// The model turns it into a ShowConfirmMsg (no datasource call yet).
	confirmCmd := m.handle(sub)
	require.NotNil(t, confirmCmd)
	confirm, ok := confirmCmd().(core.ShowConfirmMsg)
	require.True(t, ok)
	assert.True(t, confirm.Danger)
	assert.Equal(t, 0, spy.resetCalls)

	// Confirming runs the reset exactly once.
	resMsg := confirm.OnConfirm()
	res, ok := resMsg.(offsetsResetMsg)
	require.True(t, ok)
	assert.NoError(t, res.err)
	assert.Equal(t, 1, spy.resetCalls)
}

func TestDeleteGroup_ConfirmThenDeletesOnce(t *testing.T) {
	spy := newSpy()
	m := newPage(t, spy, "inventory-sync") // Empty group → deletable
	loadInto(m)

	cmd := m.deleteGroup()
	require.NotNil(t, cmd)
	confirm, ok := cmd().(core.ShowConfirmMsg)
	require.True(t, ok)
	assert.True(t, confirm.Danger)
	assert.Equal(t, 0, spy.deleteCalls) // not deleted before confirm

	// Confirm → deletes once.
	msg := confirm.OnConfirm()
	del, ok := msg.(groupDeletedMsg)
	require.True(t, ok)
	assert.NoError(t, del.err)
	assert.Equal(t, 1, spy.deleteCalls)

	// A successful delete navigates back.
	out := m.handle(del)
	require.NotNil(t, out)
	// One of the batched cmds is a BackMsg.
	assertProducesBack(t, out)
}

func TestDeleteOffsets_ConfirmThenDeletes(t *testing.T) {
	m := newPage(t, newSpy(), "inventory-sync")
	loadInto(m)
	m.topicTable.SetCursor(0)

	cmd := m.deleteSelectedTopicOffsets()
	require.NotNil(t, cmd)
	confirm, ok := cmd().(core.ShowConfirmMsg)
	require.True(t, ok)
	msg := confirm.OnConfirm()
	del, ok := msg.(offsetsDeletedMsg)
	require.True(t, ok)
	assert.NoError(t, del.err)
	assert.Equal(t, "inventory-events", del.topic)
}

func TestAutoRefresh_CycleAndTrend(t *testing.T) {
	m := newPage(t, newSpy(), "order-processor")
	loadInto(m)

	assert.Equal(t, int64(0), int64(m.autoInterval))
	m.cycleAutoRefresh() // → 10s
	assert.Greater(t, int64(m.autoInterval), int64(0))
	assert.Equal(t, m.autoInterval, m.common.Config.ConsumerGroupRefreshInterval) // persisted

	// No baseline yet → no trend arrow.
	assert.Empty(t, m.trendArrow(m.topicRows[0]))

	// Capture a baseline lower than current, then a rising reading shows ↑.
	m.trendBaseline = map[string]int64{"order-events": 50}
	m.rebuildTopicRows(m.detail) // aggLag = 100 > 50
	assert.NotEmpty(t, m.trendArrow(m.topicRows[0]))
}

// assertProducesBack drains a (possibly batched) command and asserts a BackMsg
// is produced somewhere in the output.
func assertProducesBack(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			if _, ok := c().(core.BackMsg); ok {
				return
			}
		}
		t.Fatal("no BackMsg in batch")
	}
	if _, ok := msg.(core.BackMsg); ok {
		return
	}
	t.Fatalf("expected BackMsg, got %T", msg)
}
