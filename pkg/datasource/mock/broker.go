package mock

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/api"
)

// Broker mock state is per-instance (see KafkaDataSourceMock fields) so tests
// stay isolated; it is lazily initialised on first access and mutated by the
// Alter* methods.

func f64(v float64) *float64 { return &v }
func i32(v int32) *int32     { return &v }

// ensureBrokerState lazily populates the mock broker data on first use.
func (kp *KafkaDataSourceMock) ensureBrokerState() {
	kp.brokerMu.Lock()
	defer kp.brokerMu.Unlock()
	if kp.brokers != nil {
		return
	}

	kp.brokers = []api.BrokerInfo{
		{ID: 1, Host: "kafka-1.mock", Port: 9092, Rack: "rack-a", IsController: true},
		{ID: 2, Host: "kafka-2.mock", Port: 9092, Rack: "rack-a", IsController: false},
		{ID: 3, Host: "kafka-3.mock", Port: 9092, Rack: "rack-b", IsController: false},
	}

	kp.brokerConfigs = map[int32][]api.BrokerConfigEntry{
		1: mockBrokerConfig(1),
		2: mockBrokerConfig(2),
		3: mockBrokerConfig(3),
	}

	kp.brokerLogDirs = map[int32][]api.BrokerLogDir{
		1: {
			{
				Path: "/var/lib/kafka/logs",
				Topics: []api.BrokerLogDirTopic{
					{Topic: "order-events", Partitions: []api.BrokerLogDirPartition{
						{Partition: 0, Size: 536870912, OffsetLag: 0},
						{Partition: 1, Size: 268435456, OffsetLag: 12},
					}},
					{Topic: "user-events", Partitions: []api.BrokerLogDirPartition{
						{Partition: 0, Size: 134217728, OffsetLag: 0},
					}},
				},
			},
			{Path: "/mnt/disk2/kafka", Error: "kafka: error while reading log directory"},
		},
		2: {
			{
				Path: "/var/lib/kafka/logs",
				Topics: []api.BrokerLogDirTopic{
					{Topic: "order-events", Partitions: []api.BrokerLogDirPartition{
						{Partition: 2, Size: 402653184, OffsetLag: 0},
					}},
				},
			},
			{Path: "/mnt/disk2/kafka", Error: "kafka: error while reading log directory"},
		},
		// Broker 3 has no log-dir data available.
		3: {},
	}
}

// mockBrokerConfig returns ~16 entries covering every source type plus one
// sensitive, one read-only, and keys ending .bytes / .ms.
func mockBrokerConfig(brokerID int32) []api.BrokerConfigEntry {
	brokerIDStr := fmt.Sprintf("%d", brokerID)
	return []api.BrokerConfigEntry{
		{Name: "log.retention.ms", Value: "604800000", Source: "Dynamic broker config"},
		{Name: "log.segment.bytes", Value: "1073741824", Source: "Static broker config"},
		{Name: "log.retention.bytes", Value: "-1", Source: "Dynamic broker config"},
		{Name: "message.max.bytes", Value: "1048588", Source: "Default config"},
		{Name: "socket.request.max.bytes", Value: "104857600", Source: "Default config"},
		{Name: "num.io.threads", Value: "8", Source: "Static broker config"},
		{Name: "num.network.threads", Value: "3", Source: "Dynamic default broker config"},
		{Name: "compression.type", Value: "producer", Source: "Dynamic broker config"},
		{Name: "min.insync.replicas", Value: "2", Source: "Static broker config"},
		{Name: "default.replication.factor", Value: "3", Source: "Default config"},
		{Name: "group.initial.rebalance.delay.ms", Value: "3000", Source: "Default config"},
		{Name: "inter.broker.protocol.version", Value: "3.6-IV2", Source: "Static broker config"},
		{Name: "auto.create.topics.enable", Value: "true", Source: "Default config"},
		{Name: "unclean.leader.election.enable", Value: "false", Source: "Unknown"},
		{Name: "ssl.keystore.password", Value: "", Source: "Static broker config", Sensitive: true},
		{Name: "broker.id", Value: brokerIDStr, Source: "Static broker config", ReadOnly: true},
	}
}

// GetBrokers implements api.KafkaDataSource.
func (kp *KafkaDataSourceMock) GetBrokers() ([]api.BrokerInfo, error) {
	kp.ensureBrokerState()
	kp.brokerMu.Lock()
	defer kp.brokerMu.Unlock()
	out := make([]api.BrokerInfo, len(kp.brokers))
	copy(out, kp.brokers)
	return out, nil
}

// GetBrokerStats implements api.KafkaDataSource with fixed, deterministic values
// (including one replica skew >= 20% for styling tests).
func (kp *KafkaDataSourceMock) GetBrokerStats() (map[int32]api.BrokerStats, api.BrokerSummary, error) {
	kp.ensureBrokerState()

	logDirs, _ := kp.GetBrokerLogDirs(nil)
	disk := func(id int32) (int64, int) {
		var size int64
		var count int
		for _, d := range logDirs[id] {
			for _, t := range d.Topics {
				for _, p := range t.Partitions {
					size += p.Size
					count++
				}
			}
		}
		return size, count
	}

	stats := map[int32]api.BrokerStats{}
	s1z, s1c := disk(1)
	stats[1] = api.BrokerStats{SegmentSize: s1z, SegmentCount: s1c, LeaderCount: 10, ReplicaCount: 30, InSyncReplicaCount: 30, LeaderSkew: f64(5.0), ReplicaSkew: f64(3.2)}
	s2z, s2c := disk(2)
	stats[2] = api.BrokerStats{SegmentSize: s2z, SegmentCount: s2c, LeaderCount: 9, ReplicaCount: 28, InSyncReplicaCount: 25, LeaderSkew: f64(-8.5), ReplicaSkew: f64(22.5)}
	s3z, s3c := disk(3)
	stats[3] = api.BrokerStats{SegmentSize: s3z, SegmentCount: s3c, LeaderCount: 11, ReplicaCount: 32, InSyncReplicaCount: 32, LeaderSkew: f64(12.0), ReplicaSkew: f64(-10.3)}

	summary := api.BrokerSummary{
		BrokerCount:      3,
		ControllerID:     i32(1),
		ClusterVersion:   "3.6-IV2",
		OnlinePartitions: 28,
		TotalPartitions:  30,
		UnderReplicated:  2,
		InSyncReplicas:   87,
		TotalReplicas:    90,
		OutOfSync:        3,
		ControllerType:   "KRaft",
	}
	return stats, summary, nil
}

// GetBrokerLogDirs implements api.KafkaDataSource, honouring the filter/all-brokers
// semantics (empty = all, unknown IDs dropped).
func (kp *KafkaDataSourceMock) GetBrokerLogDirs(brokerIDs []int32) (map[int32][]api.BrokerLogDir, error) {
	kp.ensureBrokerState()
	kp.brokerMu.Lock()
	defer kp.brokerMu.Unlock()

	out := map[int32][]api.BrokerLogDir{}
	if len(brokerIDs) == 0 {
		for id, dirs := range kp.brokerLogDirs {
			out[id] = dirs
		}
		return out, nil
	}
	for _, id := range brokerIDs {
		if dirs, ok := kp.brokerLogDirs[id]; ok {
			out[id] = dirs
		}
	}
	return out, nil
}

// GetBrokerConfig implements api.KafkaDataSource.
func (kp *KafkaDataSourceMock) GetBrokerConfig(brokerID int32) ([]api.BrokerConfigEntry, error) {
	kp.ensureBrokerState()
	kp.brokerMu.Lock()
	defer kp.brokerMu.Unlock()

	entries, ok := kp.brokerConfigs[brokerID]
	if !ok {
		return nil, api.BrokerNotFoundError{BrokerID: brokerID}
	}
	out := make([]api.BrokerConfigEntry, len(entries))
	copy(out, entries)
	return out, nil
}

// AlterBrokerConfig implements api.KafkaDataSource, mutating in-memory config.
// A value of "invalid" is rejected to exercise the InvalidConfigError path.
func (kp *KafkaDataSourceMock) AlterBrokerConfig(brokerID int32, key, value string) error {
	kp.ensureBrokerState()
	kp.brokerMu.Lock()
	defer kp.brokerMu.Unlock()

	entries, ok := kp.brokerConfigs[brokerID]
	if !ok {
		return api.BrokerNotFoundError{BrokerID: brokerID}
	}
	if value == "invalid" {
		return api.InvalidConfigError{Key: key, Reason: "Invalid value for configuration " + key}
	}
	for i := range entries {
		if entries[i].Name == key {
			entries[i].Value = value
			entries[i].Source = "Dynamic broker config"
			kp.brokerConfigs[brokerID] = entries
			return nil
		}
	}
	kp.brokerConfigs[brokerID] = append(entries, api.BrokerConfigEntry{Name: key, Value: value, Source: "Dynamic broker config"})
	return nil
}

// AlterReplicaLogDir implements api.KafkaDataSource, moving a partition's log to
// a different directory on the broker.
func (kp *KafkaDataSourceMock) AlterReplicaLogDir(brokerID int32, topic string, partition int32, logDir string) error {
	kp.ensureBrokerState()
	kp.brokerMu.Lock()
	defer kp.brokerMu.Unlock()

	dirs, ok := kp.brokerLogDirs[brokerID]
	if !ok {
		return api.BrokerNotFoundError{BrokerID: brokerID}
	}

	// Target directory must exist on this broker.
	targetExists := false
	for _, d := range dirs {
		if d.Path == logDir {
			targetExists = true
			break
		}
	}
	if !targetExists {
		return api.LogDirNotFoundError{Path: logDir}
	}

	// Locate the topic/partition and move it to the target dir.
	for di := range dirs {
		for ti := range dirs[di].Topics {
			if dirs[di].Topics[ti].Topic != topic {
				continue
			}
			for pi, p := range dirs[di].Topics[ti].Partitions {
				if p.Partition != partition {
					continue
				}
				if dirs[di].Path == logDir {
					return nil // already there
				}
				parts := dirs[di].Topics[ti].Partitions
				dirs[di].Topics[ti].Partitions = append(parts[:pi], parts[pi+1:]...)
				kp.addPartitionToDir(brokerID, logDir, topic, p)
				return nil
			}
		}
	}
	return api.NewPartitionError("topic/partition not found on broker", topic, partition)
}

// addPartitionToDir appends a partition log to the named directory (caller holds lock).
func (kp *KafkaDataSourceMock) addPartitionToDir(brokerID int32, logDir, topic string, p api.BrokerLogDirPartition) {
	dirs := kp.brokerLogDirs[brokerID]
	for di := range dirs {
		if dirs[di].Path != logDir {
			continue
		}
		for ti := range dirs[di].Topics {
			if dirs[di].Topics[ti].Topic == topic {
				dirs[di].Topics[ti].Partitions = append(dirs[di].Topics[ti].Partitions, p)
				return
			}
		}
		dirs[di].Topics = append(dirs[di].Topics, api.BrokerLogDirTopic{Topic: topic, Partitions: []api.BrokerLogDirPartition{p}})
		return
	}
}

// GetBrokerMetrics implements api.KafkaDataSource, returning a sample JSON snapshot.
func (kp *KafkaDataSourceMock) GetBrokerMetrics(brokerID int32) (string, error) {
	kp.ensureBrokerState()
	return fmt.Sprintf(`{
  "brokerId": %d,
  "bytesInPerSec": 1048576.0,
  "bytesOutPerSec": 2097152.0,
  "messagesInPerSec": 3200.5,
  "requestHandlerAvgIdlePercent": 0.87,
  "underReplicatedPartitions": 0,
  "activeControllerCount": 1,
  "partitionCount": 30,
  "leaderCount": 10
}`, brokerID), nil
}
