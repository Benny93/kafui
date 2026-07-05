package api

// TopicConfigEntry describes a single topic configuration key with the metadata
// needed to render it: its effective value, the derived default, the config
// source, and whether it is sensitive (masked) or read-only. (TP-2)
type TopicConfigEntry struct {
	Name      string
	Value     string
	Default   string
	Source    string
	Sensitive bool
	ReadOnly  bool
}

// PartitionInfo is the per-partition detail of a topic. (TP-3)
type PartitionInfo struct {
	ID              int32
	Leader          int32
	Replicas        []int32
	ISR             []int32
	OfflineReplicas []int32
	EarliestOffset  int64
	LatestOffset    int64
}

// MessageCount returns LatestOffset - EarliestOffset (never negative).
func (p PartitionInfo) MessageCount() int64 {
	if p.LatestOffset < p.EarliestOffset {
		return 0
	}
	return p.LatestOffset - p.EarliestOffset
}

// IsUnderReplicated reports whether the partition has fewer in-sync replicas
// than assigned replicas.
func (p PartitionInfo) IsUnderReplicated() bool {
	return len(p.ISR) < len(p.Replicas)
}

// TopicDetails aggregates per-partition detail plus topic-wide health metrics. (TP-3)
type TopicDetails struct {
	Name                      string
	Partitions                []PartitionInfo
	ReplicationFactor         int16
	IsInternal                bool
	UnderReplicatedPartitions int
	InSyncReplicas            int
	TotalReplicas             int
}

// MessageCount is the sum of per-partition (latest-earliest) message counts.
func (d TopicDetails) MessageCount() int64 {
	var total int64
	for _, p := range d.Partitions {
		total += p.MessageCount()
	}
	return total
}
