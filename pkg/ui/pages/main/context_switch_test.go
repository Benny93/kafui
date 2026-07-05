package mainpage

import (
	"context"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockKafkaDataSource is a testify mock of api.KafkaDataSource
type mockKafkaDataSource struct {
	mock.Mock
}

func (m *mockKafkaDataSource) Init(cfgOption string) {
	m.Called(cfgOption)
}

func (m *mockKafkaDataSource) GetTopics() (map[string]api.Topic, error) {
	args := m.Called()
	return args.Get(0).(map[string]api.Topic), args.Error(1)
}

func (m *mockKafkaDataSource) GetContexts() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockKafkaDataSource) GetContext() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockKafkaDataSource) SetContext(contextName string) error {
	args := m.Called(contextName)
	return args.Error(0)
}

func (m *mockKafkaDataSource) GetConsumerGroups() ([]api.ConsumerGroup, error) {
	args := m.Called()
	return args.Get(0).([]api.ConsumerGroup), args.Error(1)
}

func (m *mockKafkaDataSource) GetClusterDetails(clusterName string) (api.ClusterInfo, error) {
	args := m.Called(clusterName)
	return args.Get(0).(api.ClusterInfo), args.Error(1)
}

func (m *mockKafkaDataSource) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) error {
	args := m.Called(ctx, topicName, flags, handleMessage, onError)
	return args.Error(0)
}

func (m *mockKafkaDataSource) ProduceMessage(ctx context.Context, topic string, rec api.ProduceRecord) error {
	return nil
}

func (m *mockKafkaDataSource) GetMessageSchemaInfo(keySchemaID, valueSchemaID string) (*api.MessageSchemaInfo, error) {
	args := m.Called(keySchemaID, valueSchemaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*api.MessageSchemaInfo), args.Error(1)
}

func (m *mockKafkaDataSource) GetTopicMessageCounts(topics map[string]int32) (map[string]int64, error) {
	counts := make(map[string]int64, len(topics))
	for name := range topics {
		counts[name] = 0
	}
	return counts, nil
}

func (m *mockKafkaDataSource) GetSchemas() ([]api.Schema, error) {
	return []api.Schema{}, nil
}

func (m *mockKafkaDataSource) GetSchemaDetails(subjects []string) ([]api.Schema, error) {
	return []api.Schema{}, nil
}

func (m *mockKafkaDataSource) GetSchemaContent(subject string, version int) (string, error) {
	return `{"type":"record","name":"Mock","fields":[]}`, nil
}

func (m *mockKafkaDataSource) GetSchemaVersions(subject string) ([]api.SchemaVersion, error) {
	return nil, nil
}
func (m *mockKafkaDataSource) GetGlobalCompatibility() (api.CompatibilityLevel, error) {
	return "", nil
}
func (m *mockKafkaDataSource) GetSubjectCompatibility(subject string) (api.CompatibilityLevel, bool, error) {
	return "", false, nil
}
func (m *mockKafkaDataSource) RegisterSchema(subject, schemaText, schemaType string) (api.Schema, error) {
	return api.Schema{}, nil
}
func (m *mockKafkaDataSource) CheckSchemaCompatibility(subject, schemaText, schemaType string) (bool, []string, error) {
	return true, nil, nil
}
func (m *mockKafkaDataSource) DeleteSubject(subject string, permanent bool) ([]int, error) {
	return nil, nil
}
func (m *mockKafkaDataSource) DeleteSchemaVersion(subject string, version int, permanent bool) error {
	return nil
}
func (m *mockKafkaDataSource) SetGlobalCompatibility(level api.CompatibilityLevel) error {
	return nil
}
func (m *mockKafkaDataSource) SetSubjectCompatibility(subject string, level api.CompatibilityLevel) error {
	return nil
}

func (m *mockKafkaDataSource) GetACLs() ([]api.ACLEntry, error) {
	return []api.ACLEntry{}, nil
}

func (m *mockKafkaDataSource) GetACLsFiltered(filter api.ACLFilter) ([]api.ACLEntry, error) {
	return []api.ACLEntry{}, nil
}

func (m *mockKafkaDataSource) CreateACL(entry api.ACLEntry) error { return nil }

func (m *mockKafkaDataSource) DeleteACL(entry api.ACLEntry) error { return nil }

func (m *mockKafkaDataSource) GetClientQuotas() ([]api.ClientQuotaEntry, error) {
	return []api.ClientQuotaEntry{}, nil
}

func (m *mockKafkaDataSource) AlterClientQuotas(entity api.ClientQuotaEntity, quotas map[string]float64) error {
	return nil
}

func (m *mockKafkaDataSource) GetTopicNames() ([]string, error) {
	topics, err := m.GetTopics()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(topics))
	for name := range topics {
		names = append(names, name)
	}
	return names, nil
}

func (m *mockKafkaDataSource) DecodeMessage(_ context.Context, msg api.Message) (api.Message, error) {
	return msg, nil
}

func (m *mockKafkaDataSource) ListSerdes() []string { return []string{"string", "hex", "json"} }

func (m *mockKafkaDataSource) GetClusterStatistics(_ context.Context, _ string) (api.ClusterStatistics, error) {
	return api.ClusterStatistics{}, nil
}

func (m *mockKafkaDataSource) GetClusterCapabilities(_ context.Context, _ string) ([]api.Capability, error) {
	return nil, nil
}

func (m *mockKafkaDataSource) ValidateClusterConnection(_ context.Context, _ string) ([]api.ValidationResult, error) {
	return nil, nil
}
func (m *mockKafkaDataSource) GetBrokers() ([]api.BrokerInfo, error) { return nil, nil }
func (m *mockKafkaDataSource) GetBrokerStats() (map[int32]api.BrokerStats, api.BrokerSummary, error) {
	return nil, api.BrokerSummary{}, nil
}
func (m *mockKafkaDataSource) GetBrokerLogDirs(brokerIDs []int32) (map[int32][]api.BrokerLogDir, error) {
	return nil, nil
}
func (m *mockKafkaDataSource) GetBrokerConfig(brokerID int32) ([]api.BrokerConfigEntry, error) {
	return nil, nil
}
func (m *mockKafkaDataSource) AlterBrokerConfig(brokerID int32, key, value string) error { return nil }
func (m *mockKafkaDataSource) AlterReplicaLogDir(brokerID int32, topic string, partition int32, logDir string) error {
	return nil
}
func (m *mockKafkaDataSource) GetBrokerMetrics(brokerID int32) (string, error) { return "", nil }

// Topic-administration + analysis stubs (TP-1..TP-11, TP-29/TP-30).
func (m *mockKafkaDataSource) GetTopicConfig(topicName string) ([]api.TopicConfigEntry, error) {
	return nil, nil
}
func (m *mockKafkaDataSource) GetTopicDetails(topicName string) (api.TopicDetails, error) {
	return api.TopicDetails{}, nil
}
func (m *mockKafkaDataSource) GetTopicSizes(topicNames []string) (map[string]int64, error) {
	return nil, nil
}
func (m *mockKafkaDataSource) CreateTopic(name string, numPartitions int32, replicationFactor int16, configs map[string]*string) error {
	return nil
}
func (m *mockKafkaDataSource) DeleteTopic(name string) error         { return nil }
func (m *mockKafkaDataSource) IsTopicDeletionEnabled() (bool, error) { return true, nil }
func (m *mockKafkaDataSource) UpdateTopicConfig(name string, entries map[string]*string) error {
	return nil
}
func (m *mockKafkaDataSource) IncreasePartitions(name string, totalCount int32) error { return nil }
func (m *mockKafkaDataSource) PurgeTopicMessages(name string, partition int32) error  { return nil }
func (m *mockKafkaDataSource) RecreateTopic(name string) error                        { return nil }
func (m *mockKafkaDataSource) ChangeReplicationFactor(name string, newFactor int16) error {
	return nil
}
func (m *mockKafkaDataSource) StartTopicAnalysis(ctx context.Context, topicName string) error {
	return nil
}
func (m *mockKafkaDataSource) GetTopicAnalysis(topicName string) (*api.TopicAnalysis, error) {
	return nil, nil
}
func (m *mockKafkaDataSource) CancelTopicAnalysis(topicName string) error { return nil }

func (m *mockKafkaDataSource) GetConnectClusters(withStats bool) ([]api.ConnectCluster, error) {
	return nil, nil
}
func (m *mockKafkaDataSource) GetConnectorNames(connect string) ([]string, error) { return nil, nil }
func (m *mockKafkaDataSource) GetConnectors() ([]api.Connector, error)            { return nil, nil }
func (m *mockKafkaDataSource) GetConnectorDetails(connect, name string) (api.ConnectorDetails, error) {
	return api.ConnectorDetails{}, nil
}
func (m *mockKafkaDataSource) CreateConnector(connect, name string, config map[string]string) (api.Connector, error) {
	return api.Connector{}, nil
}
func (m *mockKafkaDataSource) UpdateConnectorConfig(connect, name string, config map[string]string) (api.Connector, error) {
	return api.Connector{}, nil
}
func (m *mockKafkaDataSource) DeleteConnector(connect, name string) error  { return nil }
func (m *mockKafkaDataSource) PauseConnector(connect, name string) error   { return nil }
func (m *mockKafkaDataSource) ResumeConnector(connect, name string) error  { return nil }
func (m *mockKafkaDataSource) StopConnector(connect, name string) error    { return nil }
func (m *mockKafkaDataSource) RestartConnector(connect, name string) error { return nil }
func (m *mockKafkaDataSource) RestartConnectorTask(connect, name string, taskID int) error {
	return nil
}
func (m *mockKafkaDataSource) ResetConnectorOffsets(connect, name string) error { return nil }
func (m *mockKafkaDataSource) GetConnectorPlugins(connect string) ([]api.ConnectorPlugin, error) {
	return nil, nil
}
func (m *mockKafkaDataSource) ValidateConnectorConfig(connect, pluginClass string, config map[string]string) (api.ConnectorValidationResult, error) {
	return api.ConnectorValidationResult{}, nil
}
func (m *mockKafkaDataSource) ListKsqlStreams() ([]api.KsqlStream, error) { return nil, nil }
func (m *mockKafkaDataSource) ListKsqlTables() ([]api.KsqlTable, error)   { return nil, nil }
func (m *mockKafkaDataSource) ExecuteKsql(ctx context.Context, sql string, props map[string]string) (<-chan api.KsqlResultTable, error) {
	return nil, nil
}
func (m *mockKafkaDataSource) GetConsumerGroupDetail(groupID string) (api.ConsumerGroupDetail, error) {
	return api.ConsumerGroupDetail{}, nil
}
func (m *mockKafkaDataSource) GetConsumerGroupDetails(groupIDs []string) ([]api.ConsumerGroup, error) {
	return nil, nil
}
func (m *mockKafkaDataSource) GetConsumerGroupsForTopic(topic string) ([]api.ConsumerGroup, error) {
	return nil, nil
}
func (m *mockKafkaDataSource) DeleteConsumerGroup(groupID string) error               { return nil }
func (m *mockKafkaDataSource) DeleteConsumerGroupOffsets(groupID, topic string) error { return nil }
func (m *mockKafkaDataSource) ResetConsumerGroupOffsets(ctx context.Context, req api.OffsetResetRequest) error {
	return nil
}

// newTestContentProvider creates a KafuiContentProvider wired to the given mock.
func newTestContentProvider(ds *mockKafkaDataSource) *KafuiContentProvider {
	return NewKafuiContentProvider(ds)
}

// runCmd executes a tea.Cmd and returns the resulting tea.Msg (nil if cmd is nil).
func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

// TestContextSelection_WhenEnterPressedOnContextItem_FiresSelectContextMsg verifies
// that pressing Enter while viewing the Contexts resource list produces a
// SelectContextMsg containing the highlighted context name.
func TestContextSelection_WhenEnterPressedOnContextItem_FiresSelectContextMsg(t *testing.T) {
	ds := &mockKafkaDataSource{}
	ds.On("GetContexts").Return([]string{"kafka-dev", "kafka-prod"}, nil)
	ds.On("GetContext").Return("kafka-dev")

	provider := newTestContentProvider(ds)

	// Switch to context resource type and seed items directly so the table has rows.
	provider.switchResource(SwitchResourceMsg(ContextResourceType))
	contextItems := []interface{}{
		newContextResourceListItem("kafka-dev", true),
		newContextResourceListItem("kafka-prod", false),
	}
	provider.allItems = contextItems
	provider.allRows = convertItemsToRows(contextItems, "", 0)
	provider.pagination.SetTotalItems(len(contextItems))
	provider.updateTableForCurrentPage()

	// Press Enter to select the first (highlighted) context.
	cmd := provider.HandleContentUpdate(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("enter")})
	// The table cursor starts at 0, so the selected item should be "kafka-dev".
	msg := runCmd(cmd)

	assert.IsType(t, SelectContextMsg{}, msg, "expected SelectContextMsg when Enter is pressed on a context item")
	selectMsg, _ := msg.(SelectContextMsg)
	assert.Equal(t, "kafka-dev", selectMsg.ContextName)
}

// TestSelectContextMsg_CallsSetContextAndSwitchesToTopics verifies that handling
// a SelectContextMsg calls SetContext on the data source and switches the active
// resource view back to Topics.
func TestSelectContextMsg_CallsSetContextAndSwitchesToTopics(t *testing.T) {
	ds := &mockKafkaDataSource{}
	ds.On("SetContext", "kafka-prod").Return(nil)
	ds.On("GetTopics").Return(map[string]api.Topic{
		"events": {NumPartitions: 3, ReplicationFactor: 1},
	}, nil)

	provider := newTestContentProvider(ds)
	// Start from Contexts view to simulate the user flow.
	provider.switchResource(SwitchResourceMsg(ContextResourceType))

	cmd := provider.HandleContentUpdate(SelectContextMsg{ContextName: "kafka-prod"})

	// SetContext must have been called with the right name.
	ds.AssertCalled(t, "SetContext", "kafka-prod")

	// After the switch the active resource must be Topics.
	assert.Equal(t, TopicResourceType, provider.currentResource.GetType(),
		"expected resource to switch back to Topics after context selection")

	// A reload command should be returned (non-nil).
	assert.NotNil(t, cmd, "expected a reload command after context switch")
}

// TestSelectContextMsg_WithCorrectContextName verifies that the context name
// carried inside SelectContextMsg is passed verbatim to SetContext.
func TestSelectContextMsg_WithCorrectContextName(t *testing.T) {
	ds := &mockKafkaDataSource{}
	ds.On("SetContext", "kafka-staging").Return(nil)
	ds.On("GetTopics").Return(map[string]api.Topic{}, nil)

	provider := newTestContentProvider(ds)

	provider.HandleContentUpdate(SelectContextMsg{ContextName: "kafka-staging"})

	ds.AssertCalled(t, "SetContext", "kafka-staging")
}

// TestSelectContextMsg_SetContextError_StillSwitchesToTopics verifies that even
// when SetContext returns an error the resource view switches to Topics and a
// command is still returned so the UI does not get stuck.
func TestSelectContextMsg_SetContextError_StillSwitchesToTopics(t *testing.T) {
	ds := &mockKafkaDataSource{}
	ds.On("SetContext", "bad-context").Return(assert.AnError)
	ds.On("GetTopics").Return(map[string]api.Topic{}, nil)

	provider := newTestContentProvider(ds)
	provider.switchResource(SwitchResourceMsg(ContextResourceType))

	cmd := provider.HandleContentUpdate(SelectContextMsg{ContextName: "bad-context"})

	assert.Equal(t, TopicResourceType, provider.currentResource.GetType())
	assert.NotNil(t, cmd)
}

// --- helpers ---

// newContextResourceListItem builds a shared.ResourceListItem wrapping a ContextResourceItem,
// matching exactly how loadCurrentResource populates allItems at runtime.
func newContextResourceListItem(name string, isCurrent bool) interface{} {
	item := &ContextResourceItem{
		id:        name,
		name:      name,
		isCurrent: isCurrent,
	}
	return shared.ResourceListItem{ResourceItem: item}
}

// TestContextColumnsHaveContextHeaders guards against bug #3 regressing: the
// Contexts view must not reuse the Topics column headers (Partitions/Replication).
func TestContextColumnsHaveContextHeaders(t *testing.T) {
	cols := createResourceTableColumns(ContextResourceType)
	require := func(cond bool, msg string) {
		if !cond {
			t.Fatal(msg)
		}
	}
	require(len(cols) == 3, "expected 3 columns for contexts")
	titles := make([]string, len(cols))
	for i, c := range cols {
		titles[i] = c.Title()
	}
	assert.Equal(t, []string{"Name", "Brokers", "Status"}, titles)
}

// TestContextRowShowsBrokersAndStatus verifies a context item's brokers/state
// land under the Brokers/Status columns rather than under generic
// Partitions/Replication cells meant for topics.
func TestContextRowShowsBrokersAndStatus(t *testing.T) {
	item := &ContextResourceItem{
		id:        "vehub-dev-aks",
		name:      "vehub-dev-aks",
		isCurrent: true,
		brokers:   []string{"kafka-bootstrap.vehub-dev:443"},
	}
	row := convertItemsToRows([]interface{}{shared.ResourceListItem{ResourceItem: item}}, "", 0)
	require := func(cond bool, msg string) {
		if !cond {
			t.Fatal(msg)
		}
	}
	require(len(row) == 1, "expected one row")
	data := row[0].Data
	assert.Equal(t, "kafka-bootstrap.vehub-dev:443", data[colPartitions])
	assert.Equal(t, "★ active", data[colReplication])
}
