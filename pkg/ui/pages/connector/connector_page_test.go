package connector

import (
	"errors"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/core"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func keyPress(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func testCommon() *core.Common {
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	return &core.Common{DataSource: ds, Styles: stylesPkg.DefaultStyles()}
}

// loadedModel returns a model with details loaded for the given connector.
func loadedModel(t *testing.T, connect, name string) *Model {
	t.Helper()
	m := newModel(testCommon(), connect, name)
	m.handle(m.loadDetails()())
	require.True(t, m.detailsLoaded, "details must load for %s/%s", connect, name)
	return m
}

func TestConnectorPage_FoundAndNotFound(t *testing.T) {
	t.Run("existing connector", func(t *testing.T) {
		m := newModel(testCommon(), "connect-primary", "orders-source")
		m.handle(m.loadDetails()())
		assert.False(t, m.notFound)
		assert.True(t, m.detailsLoaded)
		assert.Equal(t, "orders-source", m.details.Name)
	})

	t.Run("unknown connector", func(t *testing.T) {
		m := newModel(testCommon(), "connect-primary", "does-not-exist")
		m.handle(m.loadDetails()())
		assert.True(t, m.notFound)
		assert.False(t, m.detailsLoaded)
	})
}

func TestConnectorPage_TabSwitching(t *testing.T) {
	m := loadedModel(t, "connect-primary", "orders-source")
	assert.Equal(t, tabOverview, m.active)

	m.switchTab(tabTasks)
	assert.Equal(t, tabTasks, m.active)
	m.switchTab(tabConfig)
	assert.Equal(t, tabConfig, m.active)
	m.switchTab(tabTopics)
	assert.Equal(t, tabTopics, m.active)

	// Number keys select tabs too.
	m.handleKey(keyPress("2"))
	assert.Equal(t, tabTasks, m.active)
}

func TestConnectorPage_ConfigMaskedAndTasks(t *testing.T) {
	m := loadedModel(t, "connect-primary", "orders-source")
	// orders-source has a database.password → masked in returned config.
	assert.True(t, m.configHasMasked(), "secret config value should be masked")
	assert.Equal(t, api.ConnectorSecretPlaceholder, m.details.Config["database.password"])
	// Task table built with a row per task.
	assert.Len(t, m.tasksTable.Rows(), len(m.details.Tasks))
}

func TestConnectorPage_ConfigEditUpdatesOnConfirm(t *testing.T) {
	m := loadedModel(t, "connect-primary", "orders-source")
	m.active = tabConfig
	m.beginConfigEdit()
	require.True(t, m.editing)

	// Change the config to a new valid JSON object.
	m.configEditor.SetValue(`{"connector.class":"io.debezium.connector.postgresql.PostgresConnector","tasks.max":"7","name":"orders-source"}`)

	cmd := m.commitConfigEdit()
	require.NotNil(t, cmd)
	confirm, ok := cmd().(core.ShowConfirmMsg)
	require.True(t, ok, "changed config must request confirmation")
	require.NotNil(t, confirm.OnConfirm)

	updated, ok := confirm.OnConfirm().(configUpdatedMsg)
	require.True(t, ok)
	assert.NoError(t, updated.err)

	m.handle(updated)
	assert.False(t, m.editing, "successful save exits edit mode")

	// Datasource received the edited config.
	details, err := m.common.DataSource.GetConnectorDetails("connect-primary", "orders-source")
	require.NoError(t, err)
	assert.Equal(t, "7", details.Config["tasks.max"])
}

func TestConnectorPage_ConfigEditUnchangedIsNoOp(t *testing.T) {
	m := loadedModel(t, "connect-primary", "orders-source")
	m.active = tabConfig
	m.beginConfigEdit()
	// Value unchanged → no confirmation, warning notification instead.
	cmd := m.commitConfigEdit()
	require.NotNil(t, cmd)
	_, isConfirm := cmd().(core.ShowConfirmMsg)
	assert.False(t, isConfirm, "unchanged config must not request confirmation")
}

func TestConnectorPage_LifecycleConfirmsAndCalls(t *testing.T) {
	m := loadedModel(t, "connect-primary", "orders-source") // RUNNING

	cmd := m.lifecycle("pause", m.common.DataSource.PauseConnector)
	confirm, ok := cmd().(core.ShowConfirmMsg)
	require.True(t, ok, "lifecycle action must request confirmation")
	require.NotNil(t, confirm.OnConfirm)

	res, ok := confirm.OnConfirm().(lifecycleResultMsg)
	require.True(t, ok)
	assert.NoError(t, res.err)
	assert.Equal(t, "pause", res.action)

	// Datasource applied the state change exactly (connector now PAUSED).
	details, err := m.common.DataSource.GetConnectorDetails("connect-primary", "orders-source")
	require.NoError(t, err)
	assert.Equal(t, api.ConnectorStatePaused, details.State)
}

func TestConnectorPage_DeleteReturnsBack(t *testing.T) {
	m := loadedModel(t, "connect-primary", "metrics-sink-s3")
	cmd := m.deleteConnector()
	confirm := cmd().(core.ShowConfirmMsg)
	res := confirm.OnConfirm().(lifecycleResultMsg)
	require.NoError(t, res.err)
	assert.True(t, res.deleted)

	// handleLifecycleResult emits a BackMsg to return to the listing.
	_ = m.handleLifecycleResult(res)
	_, err := m.common.DataSource.GetConnectorDetails("connect-primary", "metrics-sink-s3")
	assert.Error(t, err, "connector should be gone after delete")
}

func TestConnectorPage_ResetOffsetsNonStoppedSurfacesError(t *testing.T) {
	m := loadedModel(t, "connect-primary", "orders-source") // RUNNING, not STOPPED

	cmd := m.resetOffsets()
	confirm := cmd().(core.ShowConfirmMsg)
	res := confirm.OnConfirm().(lifecycleResultMsg)

	require.Error(t, res.err)
	var nse api.ConnectorNotStoppedError
	assert.True(t, errors.As(res.err, &nse), "expected ConnectorNotStoppedError, got %T", res.err)
}

func TestConnectorPage_ResetOffsetsStoppedSucceeds(t *testing.T) {
	m := loadedModel(t, "connect-primary", "audit-sink-jdbc") // STOPPED

	cmd := m.resetOffsets()
	confirm := cmd().(core.ShowConfirmMsg)
	res := confirm.OnConfirm().(lifecycleResultMsg)
	assert.NoError(t, res.err)
}

func TestConnectorPage_RestartFailedTasks(t *testing.T) {
	m := loadedModel(t, "connect-primary", "orders-sink-es") // has a FAILED task

	cmd := m.restartTasks("failed", func(tk api.ConnectorTask) bool {
		return tk.State == api.ConnectorStateFailed
	})
	confirm := cmd().(core.ShowConfirmMsg)
	res := confirm.OnConfirm().(taskRestartResultMsg)
	assert.Equal(t, 1, res.total, "exactly one failed task should be restarted")
	assert.Empty(t, res.failures)
}
