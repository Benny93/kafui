package kafds

import (
	"context"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
)

// GetClusterStatistics implements api.KafkaDataSource by reusing the broker
// statistics collector (DescribeCluster + metadata + DescribeLogDirs) and
// folding its per-broker/summary view into a cluster-level snapshot.
//
// ponytail: KRaft/ZooKeeper quorum detection is not wrapped by
// ClusterAdminInterface, so CoordinationType is best-effort "unknown". Byte
// throughput isn't available from Sarama admin APIs (metrics feature owns it).
func (kp KafkaDataSourceKaf) GetClusterStatistics(_ context.Context, _ string) (api.ClusterStatistics, error) {
	perBroker, summary, err := kp.GetBrokerStats()
	if err != nil {
		return api.ClusterStatistics{}, err
	}

	stats := api.ClusterStatistics{
		BrokerCount:               summary.BrokerCount,
		OnlinePartitions:          summary.OnlinePartitions,
		OfflinePartitions:         summary.TotalPartitions - summary.OnlinePartitions,
		InSyncReplicas:            summary.InSyncReplicas,
		OutOfSyncReplicas:         summary.OutOfSync,
		UnderReplicatedPartitions: summary.UnderReplicated,
		Version:                   summary.ClusterVersion,
		CoordinationType:          coordinationType(summary.ControllerType),
	}
	if summary.ControllerID != nil {
		stats.ControllerID = *summary.ControllerID
	} else {
		stats.ControllerID = -1
	}
	for id, bs := range perBroker {
		stats.DiskUsage = append(stats.DiskUsage, api.BrokerDiskUsage{
			BrokerID:         id,
			TotalSegmentSize: bs.SegmentSize,
			SegmentCount:     bs.SegmentCount,
		})
	}
	return stats, nil
}

// coordinationType normalizes the broker-summary controller type to the
// cluster-statistics vocabulary.
func coordinationType(controllerType string) string {
	switch strings.ToLower(controllerType) {
	case "kraft":
		return "kraft"
	case "zookeeper":
		return "zookeeper"
	default:
		return "unknown"
	}
}

// GetClusterCapabilities implements api.KafkaDataSource. Capabilities are derived
// from configuration (schema registry) plus a best-effort ACL probe on the active
// cluster. A probe failure removes the capability but never fails the whole call.
func (kp KafkaDataSourceKaf) GetClusterCapabilities(_ context.Context, clusterName string) ([]api.Capability, error) {
	caps := []api.Capability{}

	// schema-registry: configured URL on the named cluster.
	if info, err := kp.GetClusterDetails(clusterName); err == nil && info.SchemaRegistryURL != "" {
		caps = append(caps, api.CapSchemaRegistry)
	}

	// ksqldb: an endpoint configured for the named cluster in the kafui overlay.
	if ep := loadKsqlEndpoint(clusterName); ep != nil && strings.TrimSpace(ep.URL) != "" {
		caps = append(caps, api.CapKsqlDB)
	}

	// acl-view: best effort, only meaningful for the active cluster (uses the
	// active admin client). A DescribeAcls that succeeds implies view access.
	if clusterName == kp.GetContext() {
		if admin, err := getClusterAdmin(); err == nil {
			if _, err := admin.ListAcls(sarama.AclFilter{
				ResourceType:   sarama.AclResourceAny,
				PermissionType: sarama.AclPermissionAny,
				Operation:      sarama.AclOperationAny,
			}); err == nil {
				caps = append(caps, api.CapACLView)
			}
			_ = admin.Close()
		}
	}

	return caps, nil
}

// ValidateClusterConnection implements api.KafkaDataSource. It builds a
// probe-ready extension for the named cluster (empty ⇒ the active cluster) by
// overlaying the kafui entry on the live kaf cluster, then delegates to the
// shared connectivity-validation service (AC-11). Nothing is persisted.
func (kp KafkaDataSourceKaf) ValidateClusterConnection(ctx context.Context, clusterName string) ([]api.ValidationResult, error) {
	if clusterName == "" {
		clusterName = kp.GetContext()
	}
	ext, err := kp.clusterExtensionFor(clusterName)
	if err != nil {
		return nil, err
	}
	return kp.validateCluster(ctx, ext), nil
}
