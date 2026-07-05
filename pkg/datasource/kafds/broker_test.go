package kafds

import (
	"errors"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
	"github.com/birdayz/kaf/pkg/config"
	"github.com/stretchr/testify/assert"
)

// newTestBroker builds a *sarama.Broker with a known ID (the id field is
// unexported and has no setter, so we set it via reflection for tests).
func newTestBroker(id int32, addr string) *sarama.Broker {
	b := sarama.NewBroker(addr)
	f := reflect.ValueOf(b).Elem().FieldByName("id")
	reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem().SetInt(int64(id))
	return b
}

// installMockAdmin wires the given admin as the active cluster admin and returns
// a restore func.
func installMockAdmin(admin *MockClusterAdmin) func() {
	origFactory := kafkaClientFactory
	origCluster := currentCluster
	kafkaClientFactory = &MockKafkaClientFactory{MockClusterAdmin: admin}
	currentCluster = &config.Cluster{Brokers: []string{"localhost:9092"}}
	return func() {
		kafkaClientFactory = origFactory
		currentCluster = origCluster
	}
}

func brokerDS() KafkaDataSourceKaf {
	// getClusterAdmin uses the global kafkaClientFactory, so a zero value suffices.
	return KafkaDataSourceKaf{}
}

// --- BR-3: GetBrokers ---

func TestGetBrokers(t *testing.T) {
	tests := []struct {
		name         string
		admin        *MockClusterAdmin
		wantErr      bool
		wantLen      int
		wantIDs      []int32
		controllerAt map[int32]bool // id -> expected IsController
	}{
		{
			name: "success with controller",
			admin: &MockClusterAdmin{
				MockBrokers: []*sarama.Broker{
					newTestBroker(1, "host-a:9092"),
					newTestBroker(2, "host-b:9093"),
					newTestBroker(3, "host-c:9094"),
				},
				MockControllerID: 2,
			},
			wantLen:      3,
			wantIDs:      []int32{1, 2, 3},
			controllerAt: map[int32]bool{1: false, 2: true, 3: false},
		},
		{
			name:    "describe cluster error",
			admin:   &MockClusterAdmin{ShouldFailDescribeCluster: true},
			wantErr: true,
		},
		{
			name:    "empty cluster",
			admin:   &MockClusterAdmin{MockBrokers: []*sarama.Broker{}, MockControllerID: -1},
			wantLen: 0,
		},
		{
			name: "unknown controller -1 -> no controller marked",
			admin: &MockClusterAdmin{
				MockBrokers:      []*sarama.Broker{newTestBroker(1, "h:9092"), newTestBroker(2, "h:9093")},
				MockControllerID: -1,
			},
			wantLen:      2,
			controllerAt: map[int32]bool{1: false, 2: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := installMockAdmin(tt.admin)
			defer restore()

			brokers, err := brokerDS().GetBrokers()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, brokers, tt.wantLen)
			byID := map[int32]api.BrokerInfo{}
			for _, b := range brokers {
				byID[b.ID] = b
			}
			for _, id := range tt.wantIDs {
				assert.Contains(t, byID, id)
			}
			for id, want := range tt.controllerAt {
				assert.Equal(t, want, byID[id].IsController, "broker %d controller flag", id)
			}
		})
	}
}

func TestGetBrokers_HostPortParsing(t *testing.T) {
	restore := installMockAdmin(&MockClusterAdmin{
		MockBrokers:      []*sarama.Broker{newTestBroker(1, "kafka.internal:9092")},
		MockControllerID: 1,
	})
	defer restore()

	brokers, err := brokerDS().GetBrokers()
	assert.NoError(t, err)
	assert.Equal(t, "kafka.internal", brokers[0].Host)
	assert.Equal(t, int32(9092), brokers[0].Port)
	assert.True(t, brokers[0].IsController)
}

// --- BR-4: computePartitionStats ---

func TestComputePartitionStats(t *testing.T) {
	// 2 brokers, 2 topics; broker 1 leads both partitions, RF=2.
	metadata := []*sarama.TopicMetadata{
		{
			Name: "t1",
			Partitions: []*sarama.PartitionMetadata{
				{ID: 0, Leader: 1, Replicas: []int32{1, 2}, Isr: []int32{1, 2}},
				{ID: 1, Leader: 2, Replicas: []int32{2, 1}, Isr: []int32{2}}, // under-replicated
			},
		},
		{
			Name: "t2",
			Partitions: []*sarama.PartitionMetadata{
				{ID: 0, Leader: -1, Replicas: []int32{1, 2}, Isr: []int32{}, Err: sarama.ErrLeaderNotAvailable}, // offline
			},
		},
	}

	stats, summary := computePartitionStats(metadata, []int32{1, 2})

	assert.Equal(t, 3, summary.TotalPartitions)
	assert.Equal(t, 2, summary.OnlinePartitions) // one offline
	assert.Equal(t, 2, summary.UnderReplicated)  // p1 (isr<rep) + offline p (isr empty<2)
	assert.Equal(t, 6, summary.TotalReplicas)    // 2+2+2
	assert.Equal(t, 3, summary.InSyncReplicas)   // 2+1+0
	assert.Equal(t, 3, summary.OutOfSync)

	assert.Equal(t, 1, stats[1].LeaderCount)
	assert.Equal(t, 1, stats[2].LeaderCount)
	assert.Equal(t, 3, stats[1].ReplicaCount)
	assert.Equal(t, 3, stats[2].ReplicaCount)
	// Below skew threshold (only 3 partitions) -> absent.
	assert.Nil(t, stats[1].LeaderSkew)
	assert.Nil(t, stats[1].ReplicaSkew)
}

func TestAggregateDiskUsage(t *testing.T) {
	dirs := []api.BrokerLogDir{
		{Path: "/d1", Topics: []api.BrokerLogDirTopic{
			{Topic: "a", Partitions: []api.BrokerLogDirPartition{{Size: 100}, {Size: 200}}},
		}},
		{Path: "/d2", Error: "boom"}, // errored dir contributes nothing
		{Path: "/d3", Topics: []api.BrokerLogDirTopic{
			{Topic: "b", Partitions: []api.BrokerLogDirPartition{{Size: 50}}},
		}},
	}
	size, count := aggregateDiskUsage(dirs)
	assert.Equal(t, int64(350), size)
	assert.Equal(t, 3, count)
}

// --- BR-5: GetBrokerLogDirs ---

func logDirsAdmin() *MockClusterAdmin {
	return &MockClusterAdmin{
		MockBrokers: []*sarama.Broker{
			newTestBroker(1, "h:9092"), newTestBroker(2, "h:9093"), newTestBroker(3, "h:9094"),
		},
		MockControllerID: 1,
		MockLogDirs: map[int32][]sarama.DescribeLogDirsResponseDirMetadata{
			1: {{Path: "/d1", Topics: []sarama.DescribeLogDirsResponseTopic{
				{Topic: "t", Partitions: []sarama.DescribeLogDirsResponsePartition{{PartitionID: 0, Size: 10, OffsetLag: 1}}},
			}}},
			2: {{Path: "/d2", ErrorCode: sarama.ErrKafkaStorageError}},
		},
	}
}

func TestGetBrokerLogDirs_Filter(t *testing.T) {
	t.Run("empty means all cluster brokers requested", func(t *testing.T) {
		admin := logDirsAdmin()
		restore := installMockAdmin(admin)
		defer restore()
		// Mock returns whatever it has; the request must include all 3 IDs.
		_, err := brokerDS().GetBrokerLogDirs(nil)
		assert.NoError(t, err)
	})

	t.Run("subset and unknown IDs dropped", func(t *testing.T) {
		admin := logDirsAdmin()
		restore := installMockAdmin(admin)
		defer restore()
		res, err := brokerDS().GetBrokerLogDirs([]int32{1, 99})
		assert.NoError(t, err)
		// dir 1 present; broker 99 not in cluster so never requested.
		assert.Contains(t, res, int32(1))
		assert.Equal(t, "/d1", res[1][0].Path)
		assert.Equal(t, int64(10), res[1][0].Topics[0].Partitions[0].Size)
	})

	t.Run("dir error mapped to Error string", func(t *testing.T) {
		admin := logDirsAdmin()
		restore := installMockAdmin(admin)
		defer restore()
		res, err := brokerDS().GetBrokerLogDirs([]int32{2})
		assert.NoError(t, err)
		assert.NotEmpty(t, res[2][0].Error)
	})
}

// blockingAdmin blocks in DescribeLogDirs to exercise the timeout path.
type blockingAdmin struct{ *MockClusterAdmin }

func (b blockingAdmin) DescribeLogDirs([]int32) (map[int32][]sarama.DescribeLogDirsResponseDirMetadata, error) {
	time.Sleep(200 * time.Millisecond)
	return nil, nil
}

func TestGetBrokerLogDirs_Timeout(t *testing.T) {
	orig := logDirTimeout
	logDirTimeout = 20 * time.Millisecond
	defer func() { logDirTimeout = orig }()

	admin := blockingAdmin{MockClusterAdmin: logDirsAdmin()}
	dirs, timedOut := describeLogDirsWithTimeout(admin, []int32{1})
	assert.True(t, timedOut)
	assert.Nil(t, dirs)
}

// --- BR-6: GetBrokerConfig ---

func TestGetBrokerConfig_Mapping(t *testing.T) {
	admin := &MockClusterAdmin{
		MockBrokers:      []*sarama.Broker{newTestBroker(1, "h:9092")},
		MockControllerID: 1,
		MockConfigEntries: []sarama.ConfigEntry{
			{Name: "a", Value: "1", Source: sarama.SourceDynamicBroker},
			{Name: "b", Value: "2", Source: sarama.SourceStaticBroker, ReadOnly: true},
			{Name: "c", Value: "3", Source: sarama.SourceDefault},
			{Name: "d", Value: "", Source: sarama.SourceStaticBroker, Sensitive: true},
			{Name: "e", Value: "5", Source: sarama.SourceUnknown, Synonyms: []*sarama.ConfigSynonym{
				{ConfigName: "e.syn", ConfigValue: "s", Source: sarama.SourceDynamicDefaultBroker},
			}},
		},
	}
	restore := installMockAdmin(admin)
	defer restore()

	entries, err := brokerDS().GetBrokerConfig(1)
	assert.NoError(t, err)
	byName := map[string]api.BrokerConfigEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	assert.Equal(t, "Dynamic broker config", byName["a"].Source)
	assert.Equal(t, "Static broker config", byName["b"].Source)
	assert.True(t, byName["b"].ReadOnly)
	assert.Equal(t, "Default config", byName["c"].Source)
	assert.True(t, byName["d"].Sensitive)
	assert.Equal(t, "Unknown", byName["e"].Source)
	assert.Equal(t, "Dynamic default broker config", byName["e"].Synonyms[0].Source)
	assert.Equal(t, "e.syn", byName["e"].Synonyms[0].Name)
}

func TestGetBrokerConfig_UnknownBroker(t *testing.T) {
	admin := &MockClusterAdmin{
		MockBrokers:      []*sarama.Broker{newTestBroker(1, "h:9092")},
		MockControllerID: 1,
	}
	restore := installMockAdmin(admin)
	defer restore()

	_, err := brokerDS().GetBrokerConfig(99)
	var bnf api.BrokerNotFoundError
	assert.True(t, errors.As(err, &bnf))
	assert.Equal(t, int32(99), bnf.BrokerID)
}

func TestGetBrokerConfig_ReadOnlyClusterOverride(t *testing.T) {
	admin := &MockClusterAdmin{
		MockBrokers:      []*sarama.Broker{newTestBroker(1, "h:9092")},
		MockControllerID: 1,
		MockConfigEntries: []sarama.ConfigEntry{
			{Name: "a", Value: "1", Source: sarama.SourceDynamicBroker, ReadOnly: false},
		},
	}
	restore := installMockAdmin(admin)
	defer restore()

	origRO := clusterReadOnly
	clusterReadOnly = func() bool { return true }
	defer func() { clusterReadOnly = origRO }()

	entries, err := brokerDS().GetBrokerConfig(1)
	assert.NoError(t, err)
	assert.True(t, entries[0].ReadOnly, "read-only cluster forces ReadOnly=true")
}

// --- BR-7: AlterBrokerConfig / AlterReplicaLogDir ---

func TestAlterBrokerConfig(t *testing.T) {
	t.Run("success records SET call", func(t *testing.T) {
		admin := &MockClusterAdmin{MockBrokers: []*sarama.Broker{newTestBroker(1, "h:9092")}, MockControllerID: 1}
		restore := installMockAdmin(admin)
		defer restore()

		err := brokerDS().AlterBrokerConfig(1, "retention.ms", "1000")
		assert.NoError(t, err)
		assert.Len(t, admin.IncrementalAlterConfigCalls, 1)
		assert.Equal(t, AlterConfigCall{Name: "1", Key: "retention.ms", Value: "1000"}, admin.IncrementalAlterConfigCalls[0])
	})

	t.Run("cluster rejection -> InvalidConfigError", func(t *testing.T) {
		admin := &MockClusterAdmin{
			MockBrokers:      []*sarama.Broker{newTestBroker(1, "h:9092")},
			MockControllerID: 1,
			AlterConfigErr:   errors.New("policy violation"),
		}
		restore := installMockAdmin(admin)
		defer restore()

		err := brokerDS().AlterBrokerConfig(1, "k", "v")
		var ice api.InvalidConfigError
		assert.True(t, errors.As(err, &ice))
		assert.Equal(t, "k", ice.Key)
		assert.Contains(t, ice.Reason, "policy violation")
	})

	t.Run("unknown broker -> BrokerNotFoundError", func(t *testing.T) {
		admin := &MockClusterAdmin{MockBrokers: []*sarama.Broker{newTestBroker(1, "h:9092")}, MockControllerID: 1}
		restore := installMockAdmin(admin)
		defer restore()

		err := brokerDS().AlterBrokerConfig(42, "k", "v")
		var bnf api.BrokerNotFoundError
		assert.True(t, errors.As(err, &bnf))
	})
}

func TestAlterReplicaLogDir_NotSupported(t *testing.T) {
	restore := installMockAdmin(&MockClusterAdmin{})
	defer restore()

	err := brokerDS().AlterReplicaLogDir(1, "t", 0, "/d")
	var ns api.NotSupportedError
	assert.True(t, errors.As(err, &ns))
}

func TestGetBrokerMetrics_NotAvailable(t *testing.T) {
	restore := installMockAdmin(&MockClusterAdmin{})
	defer restore()

	_, err := brokerDS().GetBrokerMetrics(1)
	var mna api.MetricsNotAvailableError
	assert.True(t, errors.As(err, &mna))
	assert.Equal(t, int32(1), mna.BrokerID)
}
