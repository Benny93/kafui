package kafds

import (
	"fmt"
	"strconv"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
)

// GetBrokerStats implements api.KafkaDataSource. It combines partition
// distribution (from topic metadata) with disk usage (from log dirs).
func (kp KafkaDataSourceKaf) GetBrokerStats() (map[int32]api.BrokerStats, api.BrokerSummary, error) {
	admin, err := getClusterAdmin()
	if err != nil {
		return nil, api.BrokerSummary{}, err
	}

	brokers, controllerID, err := admin.DescribeCluster()
	if err != nil {
		return nil, api.BrokerSummary{}, fmt.Errorf("describing cluster: %w", err)
	}

	topicDetails, err := admin.ListTopics()
	if err != nil {
		return nil, api.BrokerSummary{}, fmt.Errorf("listing topics: %w", err)
	}
	names := make([]string, 0, len(topicDetails))
	for name := range topicDetails {
		names = append(names, name)
	}

	var metadata []*sarama.TopicMetadata
	if len(names) > 0 {
		metadata, err = admin.DescribeTopics(names)
		if err != nil {
			return nil, api.BrokerSummary{}, fmt.Errorf("describing topics: %w", err)
		}
	}

	brokerIDs := make([]int32, 0, len(brokers))
	for _, b := range brokers {
		brokerIDs = append(brokerIDs, b.ID())
	}

	stats, summary := computePartitionStats(metadata, brokerIDs)
	summary.ControllerID = controllerIDPtr(controllerID)
	summary.ClusterVersion = kp.bestEffortClusterVersion(admin, controllerID)
	summary.ControllerType = "Unknown"

	// Fold disk usage from log dirs into the per-broker stats (best effort).
	if logDirs, ldErr := kp.GetBrokerLogDirs(brokerIDs); ldErr == nil {
		for id, dirs := range logDirs {
			s := stats[id]
			size, count := aggregateDiskUsage(dirs)
			s.SegmentSize = size
			s.SegmentCount = count
			stats[id] = s
		}
	}

	return stats, summary, nil
}

// computePartitionStats is the pure partition-distribution computation. It
// returns per-broker leader/replica/ISR counts and skews plus a cluster-wide
// summary. Skew is absent (nil) unless total partitions >= 50 and the per-role
// average is non-zero.
func computePartitionStats(metadata []*sarama.TopicMetadata, brokerIDs []int32) (map[int32]api.BrokerStats, api.BrokerSummary) {
	leaderCount := map[int32]int{}
	replicaCount := map[int32]int{}
	isrCount := map[int32]int{}

	var summary api.BrokerSummary
	summary.BrokerCount = len(brokerIDs)

	for _, t := range metadata {
		if t == nil {
			continue
		}
		for _, p := range t.Partitions {
			if p == nil {
				continue
			}
			summary.TotalPartitions++
			if p.Err == sarama.ErrNoError && p.Leader >= 0 {
				summary.OnlinePartitions++
				leaderCount[p.Leader]++
			}
			for _, r := range p.Replicas {
				replicaCount[r]++
				summary.TotalReplicas++
			}
			for _, r := range p.Isr {
				isrCount[r]++
				summary.InSyncReplicas++
			}
			if len(p.Isr) < len(p.Replicas) {
				summary.UnderReplicated++
			}
		}
	}
	summary.OutOfSync = summary.TotalReplicas - summary.InSyncReplicas

	leaderAvg, replicaAvg := 0.0, 0.0
	if summary.BrokerCount > 0 {
		leaderAvg = float64(summary.OnlinePartitions) / float64(summary.BrokerCount)
		replicaAvg = float64(summary.TotalReplicas) / float64(summary.BrokerCount)
	}

	stats := make(map[int32]api.BrokerStats, len(brokerIDs))
	for _, id := range brokerIDs {
		stats[id] = api.BrokerStats{
			LeaderCount:        leaderCount[id],
			ReplicaCount:       replicaCount[id],
			InSyncReplicaCount: isrCount[id],
			LeaderSkew:         api.ComputeSkew(leaderCount[id], leaderAvg, summary.TotalPartitions),
			ReplicaSkew:        api.ComputeSkew(replicaCount[id], replicaAvg, summary.TotalPartitions),
		}
	}
	return stats, summary
}

// aggregateDiskUsage sums segment size and counts partition-logs across a
// broker's log directories.
func aggregateDiskUsage(dirs []api.BrokerLogDir) (int64, int) {
	var size int64
	var count int
	for _, d := range dirs {
		for _, t := range d.Topics {
			for _, p := range t.Partitions {
				size += p.Size
				count++
			}
		}
	}
	return size, count
}

// bestEffortClusterVersion reads inter.broker.protocol.version from the
// controller's config, returning "Unknown" on any failure.
func (kp KafkaDataSourceKaf) bestEffortClusterVersion(admin ClusterAdminInterface, controllerID int32) string {
	entries, err := admin.DescribeConfig(sarama.ConfigResource{
		Type: sarama.BrokerResource,
		Name: strconv.Itoa(int(controllerID)),
	})
	if err != nil {
		return "Unknown"
	}
	for _, e := range entries {
		if e.Name == "inter.broker.protocol.version" && e.Value != "" {
			return e.Value
		}
	}
	return "Unknown"
}

func controllerIDPtr(id int32) *int32 {
	if id < 0 {
		return nil
	}
	v := id
	return &v
}
