package cluster

import (
	"context"
	"errors"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDS is a minimal KafkaDataSource stub exposing only what the collector uses.
type fakeDS struct {
	contexts   []string
	statsErr   map[string]error
	stats      map[string]api.ClusterStatistics
	caps       map[string][]api.Capability
	topicNames []string
	topicsErr  error
}

func (f *fakeDS) GetContexts() ([]string, error) { return f.contexts, nil }
func (f *fakeDS) GetClusterStatistics(_ context.Context, name string) (api.ClusterStatistics, error) {
	if err := f.statsErr[name]; err != nil {
		return api.ClusterStatistics{}, err
	}
	return f.stats[name], nil
}
func (f *fakeDS) GetClusterCapabilities(_ context.Context, name string) ([]api.Capability, error) {
	return f.caps[name], nil
}

// Unused interface methods (collector only calls the four above + GetContexts).
func (f *fakeDS) Init(string)                              {}
func (f *fakeDS) GetTopics() (map[string]api.Topic, error) { return nil, nil }
func (f *fakeDS) GetTopicNames() ([]string, error)         { return f.topicNames, f.topicsErr }
func (f *fakeDS) GetContext() string                                { return "" }
func (f *fakeDS) SetContext(string) error                           { return nil }
func (f *fakeDS) GetClusterDetails(string) (api.ClusterInfo, error) { return api.ClusterInfo{}, nil }
func (f *fakeDS) GetConsumerGroups() ([]api.ConsumerGroup, error)   { return nil, nil }
func (f *fakeDS) ConsumeTopic(context.Context, string, api.ConsumeFlags, api.MessageHandlerFunc, func(any)) error {
	return nil
}
func (f *fakeDS) ProduceMessage(context.Context, string, api.ProduceRecord) error  { return nil }
func (f *fakeDS) GetTopicMessageCounts(map[string]int32) (map[string]int64, error) { return nil, nil }
func (f *fakeDS) GetSchemas() ([]api.Schema, error)                                { return nil, nil }
func (f *fakeDS) GetSchemaDetails([]string) ([]api.Schema, error)                  { return nil, nil }
func (f *fakeDS) GetSchemaContent(string, int) (string, error)                     { return "", nil }
func (f *fakeDS) GetSchemaVersions(string) ([]api.SchemaVersion, error)            { return nil, nil }
func (f *fakeDS) GetGlobalCompatibility() (api.CompatibilityLevel, error)          { return "", nil }
func (f *fakeDS) GetSubjectCompatibility(string) (api.CompatibilityLevel, bool, error) {
	return "", false, nil
}
func (f *fakeDS) RegisterSchema(string, string, string) (api.Schema, error) { return api.Schema{}, nil }
func (f *fakeDS) CheckSchemaCompatibility(string, string, string) (bool, []string, error) {
	return true, nil, nil
}
func (f *fakeDS) DeleteSubject(string, bool) ([]int, error)                    { return nil, nil }
func (f *fakeDS) DeleteSchemaVersion(string, int, bool) error                  { return nil }
func (f *fakeDS) SetGlobalCompatibility(api.CompatibilityLevel) error          { return nil }
func (f *fakeDS) SetSubjectCompatibility(string, api.CompatibilityLevel) error { return nil }
func (f *fakeDS) GetACLs() ([]api.ACLEntry, error)                             { return nil, nil }
func (f *fakeDS) GetACLsFiltered(api.ACLFilter) ([]api.ACLEntry, error)        { return nil, nil }
func (f *fakeDS) CreateACL(api.ACLEntry) error                                 { return nil }
func (f *fakeDS) DeleteACL(api.ACLEntry) error                                 { return nil }
func (f *fakeDS) GetClientQuotas() ([]api.ClientQuotaEntry, error)             { return nil, nil }
func (f *fakeDS) AlterClientQuotas(api.ClientQuotaEntity, map[string]float64) error {
	return nil
}
func (f *fakeDS) GetMessageSchemaInfo(string, string) (*api.MessageSchemaInfo, error) {
	return nil, nil
}
func (f *fakeDS) DecodeMessage(_ context.Context, m api.Message) (api.Message, error) { return m, nil }
func (f *fakeDS) ListSerdes() []string { return []string{"string", "hex", "json"} }
func (f *fakeDS) ValidateClusterConnection(context.Context, string) ([]api.ValidationResult, error) {
	return nil, nil
}
func (f *fakeDS) GetBrokers() ([]api.BrokerInfo, error) { return nil, nil }
func (f *fakeDS) GetBrokerStats() (map[int32]api.BrokerStats, api.BrokerSummary, error) {
	return nil, api.BrokerSummary{}, nil
}
func (f *fakeDS) GetBrokerLogDirs(brokerIDs []int32) (map[int32][]api.BrokerLogDir, error) {
	return nil, nil
}
func (f *fakeDS) GetBrokerConfig(brokerID int32) ([]api.BrokerConfigEntry, error) { return nil, nil }
func (f *fakeDS) AlterBrokerConfig(brokerID int32, key, value string) error       { return nil }
func (f *fakeDS) AlterReplicaLogDir(brokerID int32, topic string, partition int32, logDir string) error {
	return nil
}
func (f *fakeDS) GetBrokerMetrics(brokerID int32) (string, error) { return "", nil }

// Topic-administration + analysis stubs (TP-1..TP-11, TP-29/TP-30).
func (f *fakeDS) GetTopicConfig(topicName string) ([]api.TopicConfigEntry, error) { return nil, nil }
func (f *fakeDS) GetTopicDetails(topicName string) (api.TopicDetails, error) {
	return api.TopicDetails{}, nil
}
func (f *fakeDS) GetTopicSizes(topicNames []string) (map[string]int64, error) { return nil, nil }
func (f *fakeDS) CreateTopic(name string, numPartitions int32, replicationFactor int16, configs map[string]*string) error {
	return nil
}
func (f *fakeDS) DeleteTopic(name string) error                                   { return nil }
func (f *fakeDS) IsTopicDeletionEnabled() (bool, error)                           { return true, nil }
func (f *fakeDS) UpdateTopicConfig(name string, entries map[string]*string) error { return nil }
func (f *fakeDS) IncreasePartitions(name string, totalCount int32) error          { return nil }
func (f *fakeDS) PurgeTopicMessages(name string, partition int32) error           { return nil }
func (f *fakeDS) RecreateTopic(name string) error                                 { return nil }
func (f *fakeDS) ChangeReplicationFactor(name string, newFactor int16) error      { return nil }
func (f *fakeDS) StartTopicAnalysis(ctx context.Context, topicName string) error  { return nil }
func (f *fakeDS) GetTopicAnalysis(topicName string) (*api.TopicAnalysis, error)   { return nil, nil }
func (f *fakeDS) CancelTopicAnalysis(topicName string) error                      { return nil }

func (f *fakeDS) GetConnectClusters(withStats bool) ([]api.ConnectCluster, error) { return nil, nil }
func (f *fakeDS) GetConnectorNames(connect string) ([]string, error)              { return nil, nil }
func (f *fakeDS) GetConnectors() ([]api.Connector, error)                         { return nil, nil }
func (f *fakeDS) GetConnectorDetails(connect, name string) (api.ConnectorDetails, error) {
	return api.ConnectorDetails{}, nil
}
func (f *fakeDS) CreateConnector(connect, name string, config map[string]string) (api.Connector, error) {
	return api.Connector{}, nil
}
func (f *fakeDS) UpdateConnectorConfig(connect, name string, config map[string]string) (api.Connector, error) {
	return api.Connector{}, nil
}
func (f *fakeDS) DeleteConnector(connect, name string) error            { return nil }
func (f *fakeDS) PauseConnector(connect, name string) error             { return nil }
func (f *fakeDS) ResumeConnector(connect, name string) error            { return nil }
func (f *fakeDS) StopConnector(connect, name string) error              { return nil }
func (f *fakeDS) RestartConnector(connect, name string) error           { return nil }
func (f *fakeDS) RestartConnectorTask(connect, name string, taskID int) error { return nil }
func (f *fakeDS) ResetConnectorOffsets(connect, name string) error      { return nil }
func (f *fakeDS) GetConnectorPlugins(connect string) ([]api.ConnectorPlugin, error) {
	return nil, nil
}
func (f *fakeDS) ValidateConnectorConfig(connect, pluginClass string, config map[string]string) (api.ConnectorValidationResult, error) {
	return api.ConnectorValidationResult{}, nil
}
func (f *fakeDS) ListKsqlStreams() ([]api.KsqlStream, error) { return nil, nil }
func (f *fakeDS) ListKsqlTables() ([]api.KsqlTable, error)   { return nil, nil }
func (f *fakeDS) ExecuteKsql(ctx context.Context, sql string, props map[string]string) (<-chan api.KsqlResultTable, error) {
	return nil, nil
}
func (f *fakeDS) GetConsumerGroupDetail(groupID string) (api.ConsumerGroupDetail, error) {
	return api.ConsumerGroupDetail{}, nil
}
func (f *fakeDS) GetConsumerGroupDetails(groupIDs []string) ([]api.ConsumerGroup, error) {
	return nil, nil
}
func (f *fakeDS) GetConsumerGroupsForTopic(topic string) ([]api.ConsumerGroup, error) {
	return nil, nil
}
func (f *fakeDS) DeleteConsumerGroup(groupID string) error               { return nil }
func (f *fakeDS) DeleteConsumerGroupOffsets(groupID, topic string) error { return nil }
func (f *fakeDS) ResetConsumerGroupOffsets(ctx context.Context, req api.OffsetResetRequest) error {
	return nil
}

func newFake() *fakeDS {
	return &fakeDS{
		contexts: []string{"a", "b"},
		statsErr: map[string]error{},
		stats: map[string]api.ClusterStatistics{
			"a": {BrokerCount: 3, OnlinePartitions: 10, Version: "3.7"},
			"b": {BrokerCount: 1, OnlinePartitions: 2, Version: "3.5"},
		},
		caps: map[string][]api.Capability{"a": {api.CapSchemaRegistry}},
	}
}

func TestInitializingBeforeFirstCycle(t *testing.T) {
	c := New(newFake(), 0, nil)
	for _, ov := range c.ListClusters() {
		assert.Equal(t, api.ClusterInitializing, ov.Status)
	}
}

func TestOnlineAfterCollect(t *testing.T) {
	c := New(newFake(), 0, nil)
	c.CollectAll(context.Background())
	for _, ov := range c.ListClusters() {
		assert.Equal(t, api.ClusterOnline, ov.Status)
	}
	st, err := c.GetStatistics("a")
	require.NoError(t, err)
	assert.Equal(t, 3, st.BrokerCount)
}

func TestOfflineIsolation(t *testing.T) {
	f := newFake()
	f.statsErr["b"] = errors.New("connection refused")
	c := New(f, 0, nil)
	c.CollectAll(context.Background())

	byName := map[string]api.ClusterOverview{}
	for _, ov := range c.ListClusters() {
		byName[ov.Name] = ov
	}
	assert.Equal(t, api.ClusterOnline, byName["a"].Status, "one failure must not affect others")
	assert.Equal(t, api.ClusterOffline, byName["b"].Status)
	assert.Contains(t, byName["b"].LastError, "connection refused")
}

func TestRefreshSingleCluster(t *testing.T) {
	f := newFake()
	f.statsErr["a"] = errors.New("down")
	c := New(f, 0, nil)
	c.CollectAll(context.Background())
	require.Equal(t, api.ClusterOffline, mustOverview(t, c, "a").Status)

	// Recover and refresh just "a".
	delete(f.statsErr, "a")
	ov, err := c.RefreshCluster(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, api.ClusterOnline, ov.Status)
}

// TestTopicCountPopulated guards against bug #6 regressing: the dashboard's
// Topics count must be wired from GetTopicNames(), not left at its zero value.
func TestTopicCountPopulated(t *testing.T) {
	f := newFake()
	f.topicNames = []string{"orders", "payments", "shipments"}
	c := New(f, 0, nil)
	c.CollectAll(context.Background())

	ov := mustOverview(t, c, "a")
	assert.Equal(t, 3, ov.TopicCount)
}

func TestRefreshUnknownCluster(t *testing.T) {
	c := New(newFake(), 0, nil)
	_, err := c.RefreshCluster(context.Background(), "nope")
	var nf api.ClusterNotFoundError
	assert.True(t, errors.As(err, &nf))
}

func TestReadOnlyFlag(t *testing.T) {
	c := New(newFake(), 0, func(name string) bool { return name == "b" })
	c.CollectAll(context.Background())
	byName := map[string]api.ClusterOverview{}
	for _, ov := range c.ListClusters() {
		byName[ov.Name] = ov
	}
	assert.True(t, byName["b"].ReadOnly)
	assert.False(t, byName["a"].ReadOnly)
}

func mustOverview(t *testing.T, c *Collector, name string) api.ClusterOverview {
	t.Helper()
	ov, err := c.overview(name)
	require.NoError(t, err)
	return ov
}
