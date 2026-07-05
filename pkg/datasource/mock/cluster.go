package mock

import (
	"context"

	"github.com/Benny93/kafui/pkg/api"
)

// mockClusterProfile describes the synthetic health/capability data for a context so
// the dashboard, capability gating, and read-only mode are all exercisable via --mock.
type mockClusterProfile struct {
	readOnly     bool
	offline      bool
	version      string
	capabilities []api.Capability
	stats        api.ClusterStatistics
}

var mockClusterProfiles = map[string]mockClusterProfile{
	"kafka-dev": {
		version:      "3.7.0",
		capabilities: []api.Capability{api.CapSchemaRegistry, api.CapKsqlDB, api.CapKafkaConnect, api.CapMetrics, api.CapTopicDeletion, api.CapACLView, api.CapACLEdit},
		stats: api.ClusterStatistics{
			BrokerCount: 3, ControllerID: 1, OnlinePartitions: 42, OfflinePartitions: 0,
			InSyncReplicas: 126, OutOfSyncReplicas: 0, UnderReplicatedPartitions: 0,
			DiskUsage: []api.BrokerDiskUsage{
				{BrokerID: 1, TotalSegmentSize: 1_500_000_000, SegmentCount: 120},
				{BrokerID: 2, TotalSegmentSize: 1_480_000_000, SegmentCount: 118},
				{BrokerID: 3, TotalSegmentSize: 1_510_000_000, SegmentCount: 121},
			},
			Version: "3.7.0", CoordinationType: "kraft",
		},
	},
	"kafka-test": {
		readOnly:     true,
		version:      "3.5.1",
		capabilities: []api.Capability{api.CapSchemaRegistry, api.CapTopicDeletion, api.CapACLView},
		stats: api.ClusterStatistics{
			BrokerCount: 2, ControllerID: 1, OnlinePartitions: 12, OfflinePartitions: 0,
			InSyncReplicas: 24, OutOfSyncReplicas: 0, UnderReplicatedPartitions: 0,
			DiskUsage: []api.BrokerDiskUsage{
				{BrokerID: 1, TotalSegmentSize: 300_000_000, SegmentCount: 30},
				{BrokerID: 2, TotalSegmentSize: 310_000_000, SegmentCount: 31},
			},
			Version: "3.5.1", CoordinationType: "zookeeper",
		},
	},
	"kafka-prod": {
		offline:      true,
		version:      "3.6.0",
		capabilities: []api.Capability{api.CapSchemaRegistry, api.CapACLView},
	},
}

func (kp *KafkaDataSourceMock) profile(clusterName string) (mockClusterProfile, error) {
	if _, ok := mockContexts[clusterName]; !ok {
		return mockClusterProfile{}, api.ClusterNotFoundError{Name: clusterName}
	}
	if p, ok := mockClusterProfiles[clusterName]; ok {
		return p, nil
	}
	return mockClusterProfile{version: "unknown"}, nil
}

// GetClusterStatistics implements api.KafkaDataSource.
func (kp *KafkaDataSourceMock) GetClusterStatistics(_ context.Context, clusterName string) (api.ClusterStatistics, error) {
	p, err := kp.profile(clusterName)
	if err != nil {
		return api.ClusterStatistics{}, err
	}
	if p.offline {
		return api.ClusterStatistics{}, api.NewConnectionError("mock cluster is offline")
	}
	return p.stats, nil
}

// GetClusterCapabilities implements api.KafkaDataSource.
func (kp *KafkaDataSourceMock) GetClusterCapabilities(_ context.Context, clusterName string) ([]api.Capability, error) {
	p, err := kp.profile(clusterName)
	if err != nil {
		return nil, err
	}
	return p.capabilities, nil
}

// ValidateClusterConnection implements api.KafkaDataSource.
func (kp *KafkaDataSourceMock) ValidateClusterConnection(_ context.Context, clusterName string) ([]api.ValidationResult, error) {
	p, err := kp.profile(clusterName)
	if err != nil {
		return nil, err
	}
	results := []api.ValidationResult{}
	if p.offline {
		results = append(results, api.ValidationResult{Component: "broker", OK: false, Err: "dial tcp: connection refused"})
	} else {
		results = append(results, api.ValidationResult{Component: "broker", OK: true})
	}
	// schema-registry component only when configured (dev/prod have a URL, per GetClusterDetails)
	if clusterName == "kafka-prod" {
		results = append(results, api.ValidationResult{Component: "schema-registry", OK: false, Err: "500 Internal Server Error"})
	} else if clusterName == "kafka-dev" {
		results = append(results, api.ValidationResult{Component: "schema-registry", OK: true})
	}
	return results, nil
}

// isReadOnly reports the mock read-only flag for a cluster (used by GetClusterDetails).
func isReadOnlyMock(clusterName string) bool {
	if p, ok := mockClusterProfiles[clusterName]; ok {
		return p.readOnly
	}
	return false
}
