package kafds

import (
	"errors"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
	"github.com/birdayz/kaf/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// installAdminIface wires an arbitrary ClusterAdminInterface as the active admin.
func installAdminIface(admin ClusterAdminInterface) func() {
	origFactory := kafkaClientFactory
	origCluster := currentCluster
	kafkaClientFactory = &MockKafkaClientFactory{MockClusterAdmin: admin}
	currentCluster = &config.Cluster{Brokers: []string{"localhost:9092"}}
	return func() {
		kafkaClientFactory = origFactory
		currentCluster = origCluster
	}
}

// withFastRetries shrinks the bounded-retry knobs for tests and restores them.
func withFastRetries(t *testing.T) {
	t.Helper()
	origMD, origMDDelay := topicMetadataRetries, topicMetadataDelay
	origRC, origRCDelay := recreateRetries, recreateDelay
	topicMetadataRetries, topicMetadataDelay = 3, time.Millisecond
	recreateRetries, recreateDelay = 3, time.Millisecond
	t.Cleanup(func() {
		topicMetadataRetries, topicMetadataDelay = origMD, origMDDelay
		recreateRetries, recreateDelay = origRC, origRCDelay
	})
}

// withOffsets installs a fetchTopicOffsets stub and restores it.
func withOffsets(t *testing.T, offs map[int32]offsets) {
	t.Helper()
	orig := fetchTopicOffsets
	fetchTopicOffsets = func(topic string, partitions []int32) (map[int32]offsets, error) {
		return offs, nil
	}
	t.Cleanup(func() { fetchTopicOffsets = orig })
}

// --- TP-2: GetTopicConfig ---

// authzErrAdmin returns an authorization error from DescribeConfig.
type authzErrAdmin struct{ *MockClusterAdmin }

func (a authzErrAdmin) DescribeConfig(resource sarama.ConfigResource) ([]sarama.ConfigEntry, error) {
	return nil, sarama.ErrTopicAuthorizationFailed
}

func TestGetTopicConfig(t *testing.T) {
	t.Run("maps entries and derives default from synonyms", func(t *testing.T) {
		admin := &MockClusterAdmin{MockConfigEntries: []sarama.ConfigEntry{
			{Name: "retention.ms", Value: "999", Source: sarama.SourceTopic, Synonyms: []*sarama.ConfigSynonym{
				{ConfigName: "retention.ms", ConfigValue: "999", Source: sarama.SourceTopic},
				{ConfigName: "log.retention.ms", ConfigValue: "604800000", Source: sarama.SourceDefault},
			}},
			{Name: "secret.key", Value: "", Source: sarama.SourceDefault, Sensitive: true},
		}}
		restore := installMockAdmin(admin)
		defer restore()

		entries, err := brokerDS().GetTopicConfig("t")
		require.NoError(t, err)
		byName := map[string]api.TopicConfigEntry{}
		for _, e := range entries {
			byName[e.Name] = e
		}
		assert.Equal(t, "999", byName["retention.ms"].Value)
		assert.Equal(t, "604800000", byName["retention.ms"].Default, "default derived from SourceDefault synonym")
		assert.Equal(t, "Topic", byName["retention.ms"].Source)
		assert.True(t, byName["secret.key"].Sensitive)
	})

	t.Run("authorization failure returns empty slice", func(t *testing.T) {
		restore := installMockAdmin(&MockClusterAdmin{})
		defer restore()
		// Swap in an admin whose DescribeConfig returns an authz error.
		kafkaClientFactory = &MockKafkaClientFactory{MockClusterAdmin: authzErrAdmin{&MockClusterAdmin{}}}

		entries, err := brokerDS().GetTopicConfig("t")
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

// --- TP-3: GetTopicDetails / buildTopicDetails ---

func TestBuildTopicDetails(t *testing.T) {
	md := &sarama.TopicMetadata{
		Name:       "orders",
		IsInternal: false,
		Partitions: []*sarama.PartitionMetadata{
			{ID: 0, Leader: 1, Replicas: []int32{1, 2, 3}, Isr: []int32{1, 2, 3}},
			{ID: 1, Leader: 2, Replicas: []int32{2, 3, 1}, Isr: []int32{2, 3}}, // under-replicated
		},
	}
	offs := map[int32]offsets{
		0: {oldest: 5, newest: 105},
		1: {oldest: 0, newest: 50},
	}
	d := buildTopicDetails(md, offs)

	assert.Equal(t, "orders", d.Name)
	assert.Equal(t, int16(3), d.ReplicationFactor)
	assert.Equal(t, 1, d.UnderReplicatedPartitions)
	assert.Equal(t, 6, d.TotalReplicas)
	assert.Equal(t, 5, d.InSyncReplicas)
	assert.Equal(t, int64(100+50), d.MessageCount())
	assert.Equal(t, int64(100), d.Partitions[0].MessageCount())
	assert.True(t, d.Partitions[1].IsUnderReplicated())
}

func TestGetTopicDetails_NotFound(t *testing.T) {
	restore := installMockAdmin(&MockClusterAdmin{MockTopicMetadata: nil})
	defer restore()

	_, err := brokerDS().GetTopicDetails("missing")
	var nf api.TopicNotFoundError
	assert.True(t, errors.As(err, &nf))
}

// --- TP-4: aggregateTopicSizes ---

func TestAggregateTopicSizes(t *testing.T) {
	// topic "a": p0 led by broker 1, p1 led by broker 2. Sizes must count leader
	// replicas only, so replica copies on the non-leader broker are ignored.
	leaders := map[string]map[int32]int32{
		"a": {0: 1, 1: 2},
		"b": {0: 1},
	}
	logDirs := map[int32][]sarama.DescribeLogDirsResponseDirMetadata{
		1: {{Path: "/d", Topics: []sarama.DescribeLogDirsResponseTopic{
			{Topic: "a", Partitions: []sarama.DescribeLogDirsResponsePartition{
				{PartitionID: 0, Size: 100}, // leader replica of a-0 -> counted
				{PartitionID: 1, Size: 999}, // NOT leader of a-1 -> ignored
			}},
			{Topic: "b", Partitions: []sarama.DescribeLogDirsResponsePartition{{PartitionID: 0, Size: 40}}},
		}}},
		2: {{Path: "/d", Topics: []sarama.DescribeLogDirsResponseTopic{
			{Topic: "a", Partitions: []sarama.DescribeLogDirsResponsePartition{
				{PartitionID: 1, Size: 200}, // leader replica of a-1 -> counted
			}},
		}}},
		// A broker whose dir errored contributes nothing.
		3: {{Path: "/x", ErrorCode: sarama.ErrKafkaStorageError, Topics: []sarama.DescribeLogDirsResponseTopic{
			{Topic: "a", Partitions: []sarama.DescribeLogDirsResponsePartition{{PartitionID: 0, Size: 7777}}},
		}}},
	}
	sizes := aggregateTopicSizes(logDirs, leaders)
	assert.Equal(t, int64(300), sizes["a"], "a-0 (100) + a-1 (200)")
	assert.Equal(t, int64(40), sizes["b"])
}

// --- TP-5: CreateTopic ---

func TestCreateTopic(t *testing.T) {
	withFastRetries(t)

	t.Run("success waits for visibility", func(t *testing.T) {
		admin := &MockClusterAdmin{MockTopicMetadata: []*sarama.TopicMetadata{{Name: "new", Err: sarama.ErrNoError}}}
		restore := installMockAdmin(admin)
		defer restore()

		err := brokerDS().CreateTopic("new", 3, 2, map[string]*string{})
		require.NoError(t, err)
		require.Len(t, admin.CreateTopicCalls, 1)
		assert.Equal(t, int32(3), admin.CreateTopicCalls[0].Detail.NumPartitions)
		assert.Equal(t, int16(2), admin.CreateTopicCalls[0].Detail.ReplicationFactor)
	})

	t.Run("already exists mapped", func(t *testing.T) {
		admin := &MockClusterAdmin{CreateTopicErr: sarama.ErrTopicAlreadyExists}
		restore := installMockAdmin(admin)
		defer restore()

		err := brokerDS().CreateTopic("dup", 1, 1, nil)
		var e api.TopicAlreadyExistsError
		assert.True(t, errors.As(err, &e))
	})

	t.Run("validation error mapped", func(t *testing.T) {
		admin := &MockClusterAdmin{CreateTopicErr: sarama.ErrInvalidReplicationFactor}
		restore := installMockAdmin(admin)
		defer restore()

		err := brokerDS().CreateTopic("bad", 1, 99, nil)
		var e api.TopicValidationError
		assert.True(t, errors.As(err, &e))
	})

	t.Run("visibility exhaustion returns timeout", func(t *testing.T) {
		admin := &MockClusterAdmin{MockTopicMetadata: nil} // never visible
		restore := installMockAdmin(admin)
		defer restore()

		err := brokerDS().CreateTopic("ghost", 1, 1, nil)
		var e api.MetadataTimeoutError
		assert.True(t, errors.As(err, &e))
	})
}

// --- TP-6: DeleteTopic + capability ---

func TestDeleteTopic(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		admin := &MockClusterAdmin{}
		restore := installMockAdmin(admin)
		defer restore()
		require.NoError(t, brokerDS().DeleteTopic("t"))
		assert.Equal(t, []string{"t"}, admin.DeleteTopicCalls)
	})
	t.Run("unknown topic mapped", func(t *testing.T) {
		admin := &MockClusterAdmin{DeleteTopicErr: sarama.ErrUnknownTopicOrPartition}
		restore := installMockAdmin(admin)
		defer restore()
		err := brokerDS().DeleteTopic("t")
		var e api.TopicNotFoundError
		assert.True(t, errors.As(err, &e))
	})
}

func TestParseDeletionEnabled(t *testing.T) {
	tests := []struct {
		name    string
		entries []sarama.ConfigEntry
		want    bool
	}{
		{"true", []sarama.ConfigEntry{{Name: "delete.topic.enable", Value: "true"}}, true},
		{"false", []sarama.ConfigEntry{{Name: "delete.topic.enable", Value: "false"}}, false},
		{"missing defaults true", []sarama.ConfigEntry{{Name: "other", Value: "x"}}, true},
		{"garbage defaults true", []sarama.ConfigEntry{{Name: "delete.topic.enable", Value: "maybe"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseDeletionEnabled(tt.entries))
		})
	}
}

func TestIsTopicDeletionEnabled(t *testing.T) {
	resetTopicDeletionCache()
	admin := &MockClusterAdmin{
		MockBrokers:      []*sarama.Broker{newTestBroker(1, "h:9092")},
		MockControllerID: 1,
		MockConfigEntries: []sarama.ConfigEntry{
			{Name: "delete.topic.enable", Value: "false"},
		},
	}
	restore := installMockAdmin(admin)
	defer restore()

	// IsTopicDeletionEnabled keys its cache by GetContext(), which reads the
	// config manager — use a DS with real deps so it does not deref a nil one.
	ds := NewKafkaDataSourceKafWithDeps(kafkaClientFactory, &DefaultConfigManager{})
	enabled, err := ds.IsTopicDeletionEnabled()
	require.NoError(t, err)
	assert.False(t, enabled)
	resetTopicDeletionCache()
}

// --- TP-7: UpdateTopicConfig ---

func TestUpdateTopicConfig(t *testing.T) {
	t.Run("sends exactly the passed entries", func(t *testing.T) {
		admin := &MockClusterAdmin{}
		restore := installMockAdmin(admin)
		defer restore()

		v := "1000"
		err := brokerDS().UpdateTopicConfig("t", map[string]*string{"retention.ms": &v})
		require.NoError(t, err)
		require.Len(t, admin.IncrementalAlterConfigCalls, 1)
		assert.Equal(t, AlterConfigCall{Name: "t", Key: "retention.ms", Value: "1000"}, admin.IncrementalAlterConfigCalls[0])
	})
	t.Run("cluster rejection mapped", func(t *testing.T) {
		admin := &MockClusterAdmin{AlterConfigErr: errors.New("policy violation")}
		restore := installMockAdmin(admin)
		defer restore()

		v := "1"
		err := brokerDS().UpdateTopicConfig("t", map[string]*string{"k": &v})
		var e api.InvalidConfigError
		assert.True(t, errors.As(err, &e))
	})
}

// --- TP-8: IncreasePartitions ---

func metaWithPartitions(name string, n int) []*sarama.TopicMetadata {
	parts := make([]*sarama.PartitionMetadata, n)
	for i := 0; i < n; i++ {
		parts[i] = &sarama.PartitionMetadata{ID: int32(i), Leader: 1, Replicas: []int32{1}, Isr: []int32{1}}
	}
	return []*sarama.TopicMetadata{{Name: name, Err: sarama.ErrNoError, Partitions: parts}}
}

func TestIncreasePartitions(t *testing.T) {
	withOffsets(t, map[int32]offsets{})

	t.Run("increase ok", func(t *testing.T) {
		admin := &MockClusterAdmin{MockTopicMetadata: metaWithPartitions("t", 3)}
		restore := installMockAdmin(admin)
		defer restore()

		require.NoError(t, brokerDS().IncreasePartitions("t", 6))
		require.Len(t, admin.CreatePartitionsCalls, 1)
		assert.Equal(t, int32(6), admin.CreatePartitionsCalls[0].Count)
	})
	t.Run("decrease rejected", func(t *testing.T) {
		admin := &MockClusterAdmin{MockTopicMetadata: metaWithPartitions("t", 3)}
		restore := installMockAdmin(admin)
		defer restore()

		err := brokerDS().IncreasePartitions("t", 2)
		var e api.PartitionDecreaseError
		require.True(t, errors.As(err, &e))
		assert.Equal(t, int32(3), e.Current)
		assert.Empty(t, admin.CreatePartitionsCalls)
	})
	t.Run("equal rejected", func(t *testing.T) {
		admin := &MockClusterAdmin{MockTopicMetadata: metaWithPartitions("t", 3)}
		restore := installMockAdmin(admin)
		defer restore()

		err := brokerDS().IncreasePartitions("t", 3)
		var e api.PartitionNoopError
		assert.True(t, errors.As(err, &e))
		assert.Empty(t, admin.CreatePartitionsCalls)
	})
}

// --- TP-9: PurgeTopicMessages ---

func TestPurgeTopicMessages(t *testing.T) {
	withOffsets(t, map[int32]offsets{0: {oldest: 0, newest: 100}, 1: {oldest: 0, newest: 200}})

	deleteCfg := []sarama.ConfigEntry{{Name: "cleanup.policy", Value: "delete"}}

	t.Run("whole topic to HWM", func(t *testing.T) {
		admin := &MockClusterAdmin{
			MockTopicMetadata: metaWithPartitions("t", 2),
			MockConfigEntries: deleteCfg,
		}
		restore := installMockAdmin(admin)
		defer restore()

		require.NoError(t, brokerDS().PurgeTopicMessages("t", -1))
		require.Len(t, admin.DeleteRecordsCalls, 1)
		assert.Equal(t, map[int32]int64{0: 100, 1: 200}, admin.DeleteRecordsCalls[0].PartitionOffsets)
	})
	t.Run("single partition", func(t *testing.T) {
		admin := &MockClusterAdmin{
			MockTopicMetadata: metaWithPartitions("t", 2),
			MockConfigEntries: deleteCfg,
		}
		restore := installMockAdmin(admin)
		defer restore()

		require.NoError(t, brokerDS().PurgeTopicMessages("t", 1))
		require.Len(t, admin.DeleteRecordsCalls, 1)
		assert.Equal(t, map[int32]int64{1: 200}, admin.DeleteRecordsCalls[0].PartitionOffsets)
	})
	t.Run("compact policy rejected", func(t *testing.T) {
		admin := &MockClusterAdmin{
			MockTopicMetadata: metaWithPartitions("t", 2),
			MockConfigEntries: []sarama.ConfigEntry{{Name: "cleanup.policy", Value: "compact"}},
		}
		restore := installMockAdmin(admin)
		defer restore()

		err := brokerDS().PurgeTopicMessages("t", -1)
		var e api.CleanupPolicyError
		assert.True(t, errors.As(err, &e))
		assert.Empty(t, admin.DeleteRecordsCalls)
	})
}

// --- TP-10: RecreateTopic ---

// flakyCreateAdmin reports "already exists" for the first N CreateTopic calls.
type flakyCreateAdmin struct {
	*MockClusterAdmin
	failFirst int
	calls     int
}

func (a *flakyCreateAdmin) CreateTopic(topic string, detail *sarama.TopicDetail, validateOnly bool) error {
	a.calls++
	if a.calls <= a.failFirst {
		return sarama.ErrTopicAlreadyExists
	}
	return nil
}

func TestRecreateTopic(t *testing.T) {
	withFastRetries(t)
	withOffsets(t, map[int32]offsets{})

	base := &MockClusterAdmin{
		MockTopicMetadata: metaWithPartitions("t", 2),
		MockConfigEntries: []sarama.ConfigEntry{{Name: "retention.ms", Value: "5", Source: sarama.SourceTopic}},
	}

	t.Run("retries while deletion propagates then succeeds", func(t *testing.T) {
		admin := &flakyCreateAdmin{MockClusterAdmin: base, failFirst: 2}
		restore := installAdminIface(admin)
		defer restore()

		require.NoError(t, brokerDS().RecreateTopic("t"))
		assert.Equal(t, 3, admin.calls, "2 failures + 1 success")
	})

	t.Run("exhaustion returns timeout", func(t *testing.T) {
		admin := &flakyCreateAdmin{MockClusterAdmin: base, failFirst: 99}
		restore := installAdminIface(admin)
		defer restore()

		err := brokerDS().RecreateTopic("t")
		var e api.RecreateTimeoutError
		assert.True(t, errors.As(err, &e))
	})
}

// --- TP-11: ChangeReplicationFactor (wiring) ---

func TestChangeReplicationFactor(t *testing.T) {
	withOffsets(t, map[int32]offsets{})
	admin := &MockClusterAdmin{
		MockTopicMetadata: []*sarama.TopicMetadata{{Name: "t", Err: sarama.ErrNoError, Partitions: []*sarama.PartitionMetadata{
			{ID: 0, Leader: 1, Replicas: []int32{1, 2}, Isr: []int32{1, 2}},
			{ID: 1, Leader: 2, Replicas: []int32{2, 3}, Isr: []int32{2, 3}},
		}}},
		MockBrokers:      []*sarama.Broker{newTestBroker(1, "h:9092"), newTestBroker(2, "h:9093"), newTestBroker(3, "h:9094")},
		MockControllerID: 1,
	}
	restore := installMockAdmin(admin)
	defer restore()

	t.Run("increase applies reassignment", func(t *testing.T) {
		admin.AlterReassignmentCalls = nil
		require.NoError(t, brokerDS().ChangeReplicationFactor("t", 3))
		require.Len(t, admin.AlterReassignmentCalls, 1)
		for _, rs := range admin.AlterReassignmentCalls[0].Assignment {
			assert.Len(t, rs, 3)
		}
	})
	t.Run("equal factor rejected before broker call", func(t *testing.T) {
		admin.AlterReassignmentCalls = nil
		err := brokerDS().ChangeReplicationFactor("t", 2)
		var e api.InvalidReplicationFactorError
		assert.True(t, errors.As(err, &e))
		assert.Empty(t, admin.AlterReassignmentCalls)
	})
}
