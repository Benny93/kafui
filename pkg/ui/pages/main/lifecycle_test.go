package mainpage

import (
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/core"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The router runs Init once and then OnFocus on every activation, including
// the first. Init already loads the content, so the first OnFocus must not
// load again; a later OnFocus (returning to the page) still refreshes.
func TestMainPage_FirstActivationLoadsOnce(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	m := NewModelWithCommon(core.NewCommon(ds))

	m.Init()
	assert.Nil(t, m.OnFocus(), "first OnFocus after Init must not reload")
	assert.NotNil(t, m.OnFocus(), "returning to the page reloads")
}

// PERF-5: only the latest tick chain re-arms; a superseded one dies.
func TestHeaderTick_StaleChainDropped(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	h := NewKafuiHeaderDataProvider(ds)

	h.InitHeader()
	old := h.tickGen
	h.InitHeader()

	assert.Nil(t, h.HandleHeaderUpdate(headerTickMsg{gen: old, at: time.Now()}), "stale chain must not re-arm")

	cmd := h.HandleHeaderUpdate(headerTickMsg{gen: h.tickGen, at: time.Now()})
	require.NotNil(t, cmd, "current chain re-arms")
	batch, ok := cmd().(tea.BatchMsg)
	require.True(t, ok)
	require.NotEmpty(t, batch)
	_, isTick := batch[0]().(TimerTickMsg)
	assert.True(t, isTick, "current chain broadcasts TimerTickMsg")
}

// The root breadcrumb and the page title name the same resource list, so the
// crumb keeps its name when the user opens a topic or group from it.
func TestBreadcrumbMatchesTitle(t *testing.T) {
	for _, rt := range []ResourceType{TopicResourceType, ConsumerGroupResourceType, SchemaResourceType,
		ContextResourceType, ACLResourceType, BrokerResourceType, QuotaResourceType,
		ConnectClusterResourceType, ConnectorResourceType} {
		k := NewKafuiContentProvider(newMockDS())
		k.switchResource(SwitchResourceMsg(rt))
		msg := k.breadcrumbCmd()()
		crumb, ok := msg.(core.BreadcrumbUpdateMsg)
		if assert.True(t, ok) && assert.Len(t, crumb.Items, 1) {
			assert.Equal(t, k.resourceLabel(), crumb.Items[0])
			if rt != TopicResourceType {
				assert.NotEqual(t, "Topics", crumb.Items[0], rt.String()) // every list has its own name
			}
		}
	}
}
