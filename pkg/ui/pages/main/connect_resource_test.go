package mainpage

import (
	"context"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/cluster"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectorRowColumns(t *testing.T) {
	item := &ConnectorResourceItem{conn: api.Connector{
		ConnectCluster:  "connect-primary",
		Name:            "orders-sink-es",
		Class:           "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
		Type:            api.ConnectorTypeSink,
		Topics:          []string{"orders"},
		State:           api.ConnectorStateFailed,
		TaskCount:       3,
		FailedTaskCount: 1,
		ConsumerGroup:   "connect-orders-sink-es",
	}}

	vals := item.GetValues()
	require.Len(t, vals, 8)
	assert.Equal(t, "orders-sink-es", vals[0])
	assert.Equal(t, "connect-primary", vals[1])
	assert.Equal(t, "sink", vals[2])
	assert.Equal(t, "ElasticsearchSinkConnector", vals[3]) // short class
	assert.Equal(t, "orders", vals[4])
	assert.Equal(t, api.ConnectorStateFailed, vals[5])
	assert.Equal(t, "connect-orders-sink-es", vals[6])
	assert.Equal(t, "2/3", vals[7]) // running/total
	assert.Equal(t, "connect-primary/orders-sink-es", item.GetID())
}

func TestConnectorStateColoring(t *testing.T) {
	// Distinct semantic foreground colours per state (rendered ANSI is stripped
	// in non-TTY test runs, so compare the style's foreground colour directly).
	running := connectorStateStyle(api.ConnectorStateRunning).GetForeground()
	failed := connectorStateStyle(api.ConnectorStateFailed).GetForeground()
	paused := connectorStateStyle(api.ConnectorStatePaused).GetForeground()
	stopped := connectorStateStyle(api.ConnectorStateStopped).GetForeground()

	assert.NotEqual(t, running, failed)
	assert.NotEqual(t, running, paused)
	assert.NotEqual(t, failed, paused)
	assert.NotEqual(t, running, stopped)

	// Failed task count is highlighted in the row data.
	row := connectorRowData(&ConnectorResourceItem{conn: api.Connector{
		Name: "c", State: api.ConnectorStateFailed, TaskCount: 2, FailedTaskCount: 1,
	}}, "")
	assert.Contains(t, row[colConnState].(string), api.ConnectorStateFailed)
}

func TestConnectClusterRowColumns(t *testing.T) {
	t.Run("reachable", func(t *testing.T) {
		item := &ConnectClusterResourceItem{cluster: api.ConnectCluster{
			Name: "connect-primary", Version: "3.7.0", Reachable: true,
			ConnectorCount: 4, TaskCount: 7, FailedTaskCount: 2,
		}}
		vals := item.GetValues()
		require.Len(t, vals, 4)
		assert.Equal(t, "connect-primary", vals[0])
		assert.Equal(t, "3.7.0", vals[1])
		assert.Equal(t, "4", vals[2])
		assert.Equal(t, "5", vals[3]) // running tasks = 7-2
	})
	t.Run("unreachable", func(t *testing.T) {
		item := &ConnectClusterResourceItem{cluster: api.ConnectCluster{Name: "connect-secondary", Reachable: false}}
		vals := item.GetValues()
		assert.Equal(t, "unreachable", vals[1])
	})
}

func TestConnectorMatchesQuery(t *testing.T) {
	c := api.Connector{
		Name: "orders-sink-es", ConnectCluster: "connect-primary",
		Class: "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
		Type:  api.ConnectorTypeSink, State: api.ConnectorStateFailed,
	}
	tests := []struct {
		query string
		want  bool
	}{
		{"orders", true},
		{"nope", false},
		{"status:FAILED", true},
		{"status:running", false},
		{"type:sink", true},
		{"type:source", false},
		{"connect:connect-primary", true},
		{"plugin:elasticsearch", true},
		{"type:sink status:FAILED", true},
		{"type:sink status:running", false},
	}
	for _, tc := range tests {
		t.Run(tc.query, func(t *testing.T) {
			assert.Equal(t, tc.want, connectorMatchesQuery(c, tc.query))
		})
	}
}

func TestConnectResourceGating(t *testing.T) {
	t.Run("hidden when CapKafkaConnect absent", func(t *testing.T) {
		common := gatedCommon(t, []api.Capability{api.CapSchemaRegistry})
		r := NewResourcesSectionWithCommon(common)
		assert.False(t, r.enabled(ConnectClusterResourceType))
		assert.False(t, r.enabled(ConnectorResourceType))

		var names []string
		for _, it := range r.RenderItems(20, 40) {
			names = append(names, it.Text)
		}
		assert.NotContains(t, names, "Connectors")
		assert.NotContains(t, names, "Connect Clusters")
	})

	t.Run("shown when CapKafkaConnect present", func(t *testing.T) {
		common := gatedCommon(t, []api.Capability{api.CapKafkaConnect})
		r := NewResourcesSectionWithCommon(common)
		assert.True(t, r.enabled(ConnectorResourceType))

		var names []string
		for _, it := range r.RenderItems(20, 40) {
			names = append(names, it.Text)
		}
		assert.Contains(t, names, "Connectors")
	})
}

func TestCreateConnectorFlow(t *testing.T) {
	k := newConnectProvider(t)
	k.switchResource(SwitchResourceMsg(ConnectorResourceType))

	cmd := k.openCreateConnectorForm()
	require.NotNil(t, cmd)
	require.True(t, k.showConnectForm, "create form should open")

	t.Run("valid config validates then creates", func(t *testing.T) {
		values := map[string]string{
			"connect": "connect-primary",
			"name":    "new-source",
			"plugin":  "org.apache.kafka.connect.file.FileStreamSourceConnector",
			"config":  "{}",
		}
		res := k.handleConnectorFormSubmit(values)().(connectorCreatedMsg)
		require.NoError(t, res.err, "valid config should pass validation and create")
		assert.Equal(t, "new-source", res.name)

		// Success closes the form and the connector now exists.
		k.handleConnectorCreated(res)
		assert.False(t, k.showConnectForm)
		names, err := k.dataSource.GetConnectorNames("connect-primary")
		require.NoError(t, err)
		assert.Contains(t, names, "new-source")
	})

	t.Run("duplicate name surfaces error and keeps form open", func(t *testing.T) {
		k.showConnectForm = true
		values := map[string]string{
			"connect": "connect-primary",
			"name":    "orders-source", // already exists
			"plugin":  "io.debezium.connector.postgresql.PostgresConnector",
			"config":  "{}",
		}
		res := k.handleConnectorFormSubmit(values)().(connectorCreatedMsg)
		require.Error(t, res.err)
		cmd := k.handleConnectorCreated(res)
		assert.True(t, k.showConnectForm, "form stays open on failure")
		_, ok := cmd().(core.NotificationMsg)
		assert.True(t, ok)
	})
}

// --- helpers ---

type connectCapDS struct {
	*mock.KafkaDataSourceMock
	caps []api.Capability
}

func (s *connectCapDS) GetClusterCapabilities(_ context.Context, _ string) ([]api.Capability, error) {
	return s.caps, nil
}

func gatedCommon(t *testing.T, caps []api.Capability) *core.Common {
	t.Helper()
	ds := &connectCapDS{KafkaDataSourceMock: &mock.KafkaDataSourceMock{}, caps: caps}
	ds.Init("")
	col := cluster.New(ds, time.Minute, nil)
	col.CollectAll(context.Background())
	return &core.Common{DataSource: ds, Collector: col, Config: &core.UIConfig{}}
}

func newConnectProvider(t *testing.T) *KafuiContentProvider {
	t.Helper()
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	return NewKafuiContentProvider(ds)
}
