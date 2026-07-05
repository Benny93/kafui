package broker

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/core"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCommon() *core.Common {
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	return &core.Common{DataSource: ds, Styles: stylesPkg.DefaultStyles()}
}

func TestBrokerPage_FoundAndNotFound(t *testing.T) {
	common := testCommon()

	t.Run("existing broker", func(t *testing.T) {
		m := newModel(common, 1, api.BrokerInfo{}, false)
		msg := m.loadInfo()()
		m.handle(msg)
		assert.False(t, m.notFound)
		assert.True(t, m.infoLoaded)
		assert.Equal(t, int32(1), m.info.ID)
	})

	t.Run("unknown broker", func(t *testing.T) {
		m := newModel(common, 99, api.BrokerInfo{}, false)
		msg := m.loadInfo()()
		m.handle(msg)
		assert.True(t, m.notFound)
	})
}

func TestBrokerPage_TabSwitching(t *testing.T) {
	m := newModel(testCommon(), 1, api.BrokerInfo{ID: 1}, true)
	assert.Equal(t, tabLogDirs, m.active)

	m.switchTab(tabConfigs)
	assert.Equal(t, tabConfigs, m.active)

	m.switchTab(tabMetrics)
	assert.Equal(t, tabMetrics, m.active)

	// Number keys select tabs too.
	m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	assert.Equal(t, tabLogDirs, m.active)
}

// loadedConfigModel returns a model with configs loaded and the configs tab active.
func loadedConfigModel(t *testing.T) *Model {
	t.Helper()
	m := newModel(testCommon(), 1, api.BrokerInfo{ID: 1}, true)
	m.active = tabConfigs
	m.handle(m.loadConfigs()())
	require.True(t, m.configsLoaded)
	require.NotEmpty(t, m.cfgVisible)
	return m
}

func selectConfig(t *testing.T, m *Model, name string) {
	t.Helper()
	for i, e := range m.cfgVisible {
		if e.Name == name {
			m.cfgTable.SetCursor(i)
			return
		}
	}
	t.Fatalf("config %q not found", name)
}

func TestConfigEdit_ReadOnlyIsNoOp(t *testing.T) {
	m := loadedConfigModel(t)
	selectConfig(t, m, "broker.id") // read-only in mock
	cmd := m.beginEdit()
	assert.False(t, m.editing, "read-only entry must not enter edit mode")
	require.NotNil(t, cmd)
	msg := cmd()
	n, ok := msg.(core.NotificationMsg)
	require.True(t, ok)
	assert.Contains(t, n.Message, "read-only")
}

func TestConfigEdit_EnterAndCancel(t *testing.T) {
	m := loadedConfigModel(t)
	selectConfig(t, m, "compression.type")
	m.beginEdit()
	assert.True(t, m.editing)

	m.handleEditKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, m.editing, "esc cancels edit")
}

func TestConfigEdit_SaveUnchangedIsNoOp(t *testing.T) {
	m := loadedConfigModel(t)
	selectConfig(t, m, "compression.type")
	m.beginEdit()
	// value unchanged
	cmd := m.commitEdit()
	assert.Nil(t, cmd, "unchanged save issues no command")
	assert.False(t, m.editing)
}

func TestConfigEdit_SaveChangedConfirmsAndCalls(t *testing.T) {
	m := loadedConfigModel(t)
	selectConfig(t, m, "compression.type")
	m.beginEdit()
	m.editInput.SetValue("gzip")

	cmd := m.commitEdit()
	require.NotNil(t, cmd)
	confirm, ok := cmd().(core.ShowConfirmMsg)
	require.True(t, ok, "changed save must request confirmation")
	assert.Equal(t, "Are you sure you want to change the value?", confirm.Message)
	require.NotNil(t, confirm.OnConfirm)

	// Confirming calls AlterBrokerConfig exactly once with key+value.
	altered, ok := confirm.OnConfirm().(configAlteredMsg)
	require.True(t, ok)
	assert.Equal(t, "compression.type", altered.key)
	assert.Equal(t, "gzip", altered.value)
	assert.NoError(t, altered.err)

	m.handle(altered)
	assert.False(t, m.editing, "successful save exits edit mode")

	// Verify the datasource actually stored the new value.
	entries, err := m.common.DataSource.GetBrokerConfig(1)
	require.NoError(t, err)
	found := false
	for _, e := range entries {
		if e.Name == "compression.type" {
			assert.Equal(t, "gzip", e.Value)
			found = true
		}
	}
	assert.True(t, found)
}

func TestConfigEdit_InvalidKeepsEditMode(t *testing.T) {
	m := loadedConfigModel(t)
	selectConfig(t, m, "log.retention.ms")
	m.beginEdit()
	m.editInput.SetValue("invalid") // mock rejects this value

	cmd := m.commitEdit()
	confirm := cmd().(core.ShowConfirmMsg)
	altered := confirm.OnConfirm().(configAlteredMsg)
	require.Error(t, altered.err)

	m.handle(altered)
	assert.True(t, m.editing, "invalid config rejection keeps edit mode open")
}

func TestBrokerPage_LogDirRows(t *testing.T) {
	m := newModel(testCommon(), 1, api.BrokerInfo{ID: 1}, true)
	m.handle(m.loadLogDirs()())
	require.True(t, m.logDirsLoaded)
	require.Len(t, m.logDirs, 2)
	assert.Equal(t, 2, len(m.logTable.Rows()))
}

func TestBrokerPage_MetricsLoaded(t *testing.T) {
	m := newModel(testCommon(), 1, api.BrokerInfo{ID: 1}, true)
	m.active = tabMetrics
	m.handle(m.loadMetrics()())
	assert.True(t, m.metricsLoaded)
	assert.NoError(t, m.metricsErr)
}
