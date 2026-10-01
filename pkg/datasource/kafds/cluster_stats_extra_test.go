package kafds

import (
	"context"
	"errors"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// statsAdmin is a broker-summary fixture: 3 brokers, controller 1, one online
// fully-isr partition led by broker 1, log dirs on brokers 1 and 2.
func statsAdmin() *MockClusterAdmin {
	a := logDirsAdmin()
	a.MockTopicMetadata = []*sarama.TopicMetadata{{Name: "t", Err: sarama.ErrNoError, Partitions: []*sarama.PartitionMetadata{
		{ID: 0, Leader: 1, Replicas: []int32{1, 2}, Isr: []int32{1, 2}},
	}}}
	return a
}

// statsDS is like brokerDS() but carries the config manager that GetContext
// (and therefore the statistics/capabilities paths) dereferences.
func statsDS() KafkaDataSourceKaf {
	return KafkaDataSourceKaf{configManager: &DefaultConfigManager{}}
}

func TestGetClusterStatistics_ActiveCluster(t *testing.T) {
	withTwoClusters(t)
	restore := installMockAdmin(statsAdmin())
	defer restore()
	ds := statsDS()

	// Empty name and the active name take the same path.
	for _, name := range []string{"", "a"} {
		stats, err := ds.GetClusterStatistics(context.Background(), name)
		require.NoError(t, err)
		assert.Equal(t, 3, stats.BrokerCount)
		assert.Equal(t, int32(1), stats.ControllerID)
		assert.Equal(t, 1, stats.OnlinePartitions)
		assert.Equal(t, 0, stats.OfflinePartitions)
		assert.Equal(t, 2, stats.InSyncReplicas)
		assert.Equal(t, 0, stats.OutOfSyncReplicas)
		assert.Equal(t, 0, stats.UnderReplicatedPartitions)
		assert.Equal(t, "unknown", stats.CoordinationType, "controller type is best-effort until KRaft quorum probing exists")
		// One disk-usage entry per broker; sizes are known only where log
		// dirs answered (broker 1), error/absent dirs leave the entry zeroed.
		require.Len(t, stats.DiskUsage, 3)
		byID := map[int32]api.BrokerDiskUsage{}
		for _, d := range stats.DiskUsage {
			byID[d.BrokerID] = d
		}
		assert.Equal(t, int64(10), byID[1].TotalSegmentSize)
		assert.Equal(t, 1, byID[1].SegmentCount)
		assert.Equal(t, int64(0), byID[2].TotalSegmentSize, "storage-error dir reports no size")
		assert.Equal(t, int64(0), byID[3].TotalSegmentSize, "broker without log-dir data reports no size")
	}
}

func TestGetClusterStatistics_BrokerStatsErrorPropagates(t *testing.T) {
	withTwoClusters(t)
	admin := &MockClusterAdmin{ShouldFailDescribeCluster: true}
	restore := installMockAdmin(admin)
	defer restore()

	_, err := statsDS().GetClusterStatistics(context.Background(), "a")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "describing cluster")
}

func TestGetClusterCapabilities_AllThree(t *testing.T) {
	withTwoClusters(t)
	restore := installMockAdmin(&MockClusterAdmin{})
	defer restore()
	withKsqlStub(t, &appconfig.KsqlEndpoint{URL: "http://ksql:8088"})
	cfg.Clusters[0].SchemaRegistryURL = "http://sr:8081"

	caps, err := statsDS().GetClusterCapabilities(context.Background(), "a")
	require.NoError(t, err)
	assert.ElementsMatch(t, []api.Capability{api.CapSchemaRegistry, api.CapKsqlDB, api.CapACLView}, caps)
}

// A cluster without the pieces reports no capabilities, and the ACL probe is
// only paid for on the active cluster.
func TestGetClusterCapabilities_NoneAndProbeFailure(t *testing.T) {
	withTwoClusters(t)
	admin := &MockClusterAdmin{ListAclsErr: errors.New("SECURITY_DISABLED")}
	restore := installMockAdmin(admin)
	defer restore()
	withKsqlStub(t, nil)

	caps, err := statsDS().GetClusterCapabilities(context.Background(), "a")
	require.NoError(t, err)
	assert.Empty(t, caps, "failed ACL probe drops only that capability")

	// Inactive cluster: no ACL probe at all, even with a configured registry.
	cfg.Clusters[1].SchemaRegistryURL = "http://sr:8081"
	caps, err = statsDS().GetClusterCapabilities(context.Background(), "b")
	require.NoError(t, err)
	assert.Equal(t, []api.Capability{api.CapSchemaRegistry}, caps,
		"the inactive cluster's capabilities come from configuration only")
}

// withKsqlStub swaps the ksql endpoint loader for a fixed answer.
func withKsqlStub(t *testing.T, ep *appconfig.KsqlEndpoint) {
	t.Helper()
	prev := loadKsqlEndpoint
	loadKsqlEndpoint = func(string) *appconfig.KsqlEndpoint { return ep }
	t.Cleanup(func() { loadKsqlEndpoint = prev })
}
