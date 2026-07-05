package api

import "slices"

// ClusterStatus is the health status of a cluster as tracked by the background collector.
type ClusterStatus string

const (
	ClusterInitializing ClusterStatus = "initializing"
	ClusterOnline       ClusterStatus = "online"
	ClusterOffline      ClusterStatus = "offline"
)

// Capability names an optional feature a cluster supports. UI sections are gated on these.
type Capability string

const (
	CapSchemaRegistry Capability = "schema-registry"
	CapKafkaConnect   Capability = "kafka-connect"
	CapKsqlDB         Capability = "ksqldb"
	CapMetrics        Capability = "metrics"
	CapTopicDeletion  Capability = "topic-deletion"
	CapACLView        Capability = "acl-view"
	CapACLEdit        Capability = "acl-edit"
)

// ClusterOverview is the cached, display-oriented summary of a cluster shown on the dashboard.
// Rate fields use -1 to mean "unknown/not yet collected".
type ClusterOverview struct {
	Name                 string
	Status               ClusterStatus
	LastError            string
	BrokerCount          int
	OnlinePartitionCount int
	TopicCount           int
	BytesInPerSec        float64
	BytesOutPerSec       float64
	MessagesInPerSec     float64
	ReadOnly             bool
	Version              string
	Capabilities         []Capability
}

// BrokerDiskUsage is per-broker log-directory usage.
type BrokerDiskUsage struct {
	BrokerID         int32
	TotalSegmentSize int64
	SegmentCount     int
}

// ClusterStatistics is a freshly-collected detailed snapshot of a cluster.
type ClusterStatistics struct {
	BrokerCount               int
	ControllerID              int32
	OnlinePartitions          int
	OfflinePartitions         int
	InSyncReplicas            int
	OutOfSyncReplicas         int
	UnderReplicatedPartitions int
	DiskUsage                 []BrokerDiskUsage
	Version                   string
	CoordinationType          string // "kraft", "zookeeper", or "unknown"
}

// ValidationResult is the outcome of probing one component of a cluster connection.
type ValidationResult struct {
	Component string // "broker", "schema-registry", "tls", "connect:<name>", "ksql", "metrics"
	OK        bool
	Err       string
}

// ClusterValidation groups the per-component probe results for one cluster.
type ClusterValidation struct {
	Cluster string
	Results []ValidationResult
}

// ValidationReport is the outcome of probing a candidate configuration, with one
// entry per cluster. An empty candidate yields an empty report (nil Clusters).
type ValidationReport struct {
	Clusters []ClusterValidation
}

// HasCapability reports whether the overview lists the given capability.
func (o ClusterOverview) HasCapability(c Capability) bool {
	return slices.Contains(o.Capabilities, c)
}
