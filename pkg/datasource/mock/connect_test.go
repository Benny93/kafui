package mock

import (
	"errors"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newConnectMock() *KafkaDataSourceMock {
	m := &KafkaDataSourceMock{}
	m.Init("")
	return m
}

func TestMockConnect_Clusters(t *testing.T) {
	m := newConnectMock()
	clusters, err := m.GetConnectClusters(true)
	require.NoError(t, err)
	require.Len(t, clusters, 2)

	var primary, secondary api.ConnectCluster
	for _, c := range clusters {
		if c.Name == "connect-primary" {
			primary = c
		} else {
			secondary = c
		}
	}
	assert.True(t, primary.Reachable)
	assert.Positive(t, primary.ConnectorCount)
	assert.Positive(t, primary.FailedConnectorCount)
	assert.Positive(t, primary.FailedTaskCount)

	assert.False(t, secondary.Reachable)
	assert.Zero(t, secondary.ConnectorCount)
}

func TestMockConnect_AggregationOmitsUnreachable(t *testing.T) {
	m := newConnectMock()
	conns, err := m.GetConnectors()
	require.NoError(t, err)
	require.NotEmpty(t, conns)
	for _, c := range conns {
		assert.Equal(t, "connect-primary", c.ConnectCluster, "only reachable cluster's connectors listed")
	}
	// sink connectors get a derived consumer group
	for _, c := range conns {
		if c.Type == api.ConnectorTypeSink {
			assert.NotEmpty(t, c.ConsumerGroup)
		}
	}
}

func TestMockConnect_DetailsMasked(t *testing.T) {
	m := newConnectMock()
	d, err := m.GetConnectorDetails("connect-primary", "orders-source")
	require.NoError(t, err)
	assert.Equal(t, api.ConnectorSecretPlaceholder, d.Config["database.password"])
	assert.Equal(t, "postgres", d.Config["database.hostname"])

	_, err = m.GetConnectorDetails("connect-primary", "nope")
	var nf api.ConnectorNotFoundError
	assert.True(t, errors.As(err, &nf))
}

func TestMockConnect_LifecycleTransitions(t *testing.T) {
	m := newConnectMock()
	require.NoError(t, m.PauseConnector("connect-primary", "orders-source"))
	d, _ := m.GetConnectorDetails("connect-primary", "orders-source")
	assert.Equal(t, api.ConnectorStatePaused, d.State)

	require.NoError(t, m.ResumeConnector("connect-primary", "orders-source"))
	d, _ = m.GetConnectorDetails("connect-primary", "orders-source")
	assert.Equal(t, api.ConnectorStateRunning, d.State)

	require.NoError(t, m.StopConnector("connect-primary", "orders-source"))
	d, _ = m.GetConnectorDetails("connect-primary", "orders-source")
	assert.Equal(t, api.ConnectorStateStopped, d.State)
}

func TestMockConnect_CreateDeleteRoundTrip(t *testing.T) {
	m := newConnectMock()
	_, err := m.CreateConnector("connect-primary", "new-sink", map[string]string{
		"connector.class": "FooSink", "topics": "t1",
	})
	require.NoError(t, err)
	d, err := m.GetConnectorDetails("connect-primary", "new-sink")
	require.NoError(t, err)
	assert.Equal(t, api.ConnectorTypeSink, d.Type)

	// duplicate rejected
	_, err = m.CreateConnector("connect-primary", "new-sink", map[string]string{"connector.class": "FooSink"})
	var exists api.ConnectorAlreadyExistsError
	assert.True(t, errors.As(err, &exists))

	require.NoError(t, m.DeleteConnector("connect-primary", "new-sink"))
	_, err = m.GetConnectorDetails("connect-primary", "new-sink")
	var nf api.ConnectorNotFoundError
	assert.True(t, errors.As(err, &nf))
}

func TestMockConnect_UpdateConfig(t *testing.T) {
	m := newConnectMock()
	_, err := m.UpdateConnectorConfig("connect-primary", "orders-source", map[string]string{
		"connector.class": "NewClass", "tasks.max": "5",
	})
	require.NoError(t, err)
	d, _ := m.GetConnectorDetails("connect-primary", "orders-source")
	assert.Equal(t, "NewClass", d.Class)
	assert.Equal(t, "5", d.Config["tasks.max"])
}

func TestMockConnect_ResetOffsetsRequiresStopped(t *testing.T) {
	m := newConnectMock()
	// orders-source is RUNNING initially
	err := m.ResetConnectorOffsets("connect-primary", "orders-source")
	var ns api.ConnectorNotStoppedError
	require.True(t, errors.As(err, &ns))

	require.NoError(t, m.StopConnector("connect-primary", "orders-source"))
	require.NoError(t, m.ResetConnectorOffsets("connect-primary", "orders-source"))
}

func TestMockConnect_RestartTask(t *testing.T) {
	m := newConnectMock()
	// orders-sink-es task 1 is FAILED
	require.NoError(t, m.RestartConnectorTask("connect-primary", "orders-sink-es", 1))
	d, _ := m.GetConnectorDetails("connect-primary", "orders-sink-es")
	for _, tk := range d.Tasks {
		if tk.ID == 1 {
			assert.Equal(t, api.ConnectorStateRunning, tk.State)
		}
	}
}

func TestMockConnect_PluginsAndValidation(t *testing.T) {
	m := newConnectMock()
	plugins, err := m.GetConnectorPlugins("connect-primary")
	require.NoError(t, err)
	assert.NotEmpty(t, plugins)

	// missing required field flagged
	res, err := m.ValidateConnectorConfig("connect-primary", "FooSource", map[string]string{})
	require.NoError(t, err)
	assert.Positive(t, res.ErrorCount)

	res, err = m.ValidateConnectorConfig("connect-primary", "FooSource", map[string]string{
		"name": "x", "connector.class": "FooSource",
	})
	require.NoError(t, err)
	assert.Zero(t, res.ErrorCount)
}

func TestMockConnect_UnknownCluster(t *testing.T) {
	m := newConnectMock()
	_, err := m.GetConnectorNames("nope")
	var nf api.ConnectClusterNotFoundError
	assert.True(t, errors.As(err, &nf))
}
