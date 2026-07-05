package api

import "math"

// BrokerInfo describes a single Kafka broker as returned by a cluster describe.
type BrokerInfo struct {
	ID           int32
	Host         string
	Port         int32
	Rack         string
	IsController bool
}

// BrokerStats holds per-broker partition-distribution and disk-usage statistics.
// ReplicaSkew and LeaderSkew are pointers so an absent value ("not computed",
// e.g. fewer than 50 partitions in the cluster) is distinguishable from 0%.
type BrokerStats struct {
	SegmentSize        int64
	SegmentCount       int
	LeaderCount        int
	ReplicaCount       int
	InSyncReplicaCount int
	ReplicaSkew        *float64
	LeaderSkew         *float64
}

// BrokerLogDirPartition describes a single partition's log within a directory.
type BrokerLogDirPartition struct {
	Partition int32
	Size      int64
	OffsetLag int64
}

// BrokerLogDirTopic groups a topic's partition logs within a directory.
type BrokerLogDirTopic struct {
	Topic      string
	Partitions []BrokerLogDirPartition
}

// BrokerLogDir describes one log directory on a broker. Error is the non-empty
// error string reported by the broker for that directory, if any.
type BrokerLogDir struct {
	Path   string
	Error  string
	Topics []BrokerLogDirTopic
}

// BrokerConfigSynonym is one entry in a config value's synonym chain.
type BrokerConfigSynonym struct {
	Name   string
	Value  string
	Source string
}

// BrokerConfigEntry is a single broker configuration key/value with metadata.
type BrokerConfigEntry struct {
	Name      string
	Value     string
	Source    string
	Sensitive bool
	ReadOnly  bool
	Synonyms  []BrokerConfigSynonym
}

// BrokerSummary aggregates cluster-wide broker/partition health for the summary panel.
type BrokerSummary struct {
	BrokerCount      int
	ControllerID     *int32
	ClusterVersion   string
	OnlinePartitions int
	TotalPartitions  int
	UnderReplicated  int
	InSyncReplicas   int
	TotalReplicas    int
	OutOfSync        int
	ControllerType   string // "KRaft" | "ZooKeeper" | "Unknown"
}

// skewMinPartitions is the total-partition threshold below which skew is not
// meaningful and is reported as absent (nil).
const skewMinPartitions = 50

// RoundHalfUp rounds v to the given number of decimal places, rounding halves
// away from zero (so 12.35 -> 12.4, -12.35 -> -12.4).
func RoundHalfUp(v float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	if v >= 0 {
		return math.Floor(v*pow+0.5) / pow
	}
	return math.Ceil(v*pow-0.5) / pow
}

// ComputeSkew returns the percentage deviation of a single broker's count from
// the per-role average, rounded half-up to one decimal place:
//
//	((brokerCount - avg) / avg) * 100
//
// It returns nil (absent) when totalPartitions is below the significance
// threshold or when the average is zero, so callers can render "N/A"/"-".
func ComputeSkew(brokerCount int, avg float64, totalPartitions int) *float64 {
	if totalPartitions < skewMinPartitions || avg == 0 {
		return nil
	}
	skew := ((float64(brokerCount) - avg) / avg) * 100
	rounded := RoundHalfUp(skew, 1)
	return &rounded
}
