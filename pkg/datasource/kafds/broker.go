package kafds

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
)

// logDirTimeout bounds DescribeLogDirs so a slow/unresponsive broker yields an
// empty result instead of hanging the UI (BR-5). It is a var so tests can shrink it.
var logDirTimeout = 10 * time.Second

// clusterReadOnly reports whether the active cluster is configured read-only.
// It is a seam: real cluster-capability detection (feature 1) wires in later,
// and tests override it to exercise the read-only-config path (BR-6).
var clusterReadOnly = func() bool { return false }

// GetBrokers implements api.KafkaDataSource. It describes the cluster and maps
// each online broker to api.BrokerInfo, marking the active controller.
func (kp KafkaDataSourceKaf) GetBrokers() ([]api.BrokerInfo, error) {
	admin, err := getClusterAdmin()
	if err != nil {
		return nil, err
	}
	brokers, controllerID, err := admin.DescribeCluster()
	if err != nil {
		return nil, fmt.Errorf("describing cluster: %w", err)
	}
	return brokersToInfo(brokers, controllerID), nil
}

// brokersToInfo is the pure mapping from sarama brokers to api.BrokerInfo.
func brokersToInfo(brokers []*sarama.Broker, controllerID int32) []api.BrokerInfo {
	infos := make([]api.BrokerInfo, 0, len(brokers))
	for _, b := range brokers {
		host, port := splitHostPort(b.Addr())
		infos = append(infos, api.BrokerInfo{
			ID:           b.ID(),
			Host:         host,
			Port:         port,
			Rack:         b.Rack(),
			IsController: b.ID() == controllerID,
		})
	}
	return infos
}

// splitHostPort parses "host:port"; on failure it returns the raw address as
// host with port 0.
func splitHostPort(addr string) (string, int32) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, 0
	}
	port, _ := strconv.Atoi(portStr)
	return host, int32(port)
}

// GetBrokerConfig implements api.KafkaDataSource.
func (kp KafkaDataSourceKaf) GetBrokerConfig(brokerID int32) ([]api.BrokerConfigEntry, error) {
	admin, err := getClusterAdmin()
	if err != nil {
		return nil, err
	}
	brokers, _, err := admin.DescribeCluster()
	if err != nil {
		return nil, fmt.Errorf("describing cluster: %w", err)
	}
	if !brokerExists(brokers, brokerID) {
		return nil, api.BrokerNotFoundError{BrokerID: brokerID}
	}

	entries, err := admin.DescribeConfig(sarama.ConfigResource{
		Type: sarama.BrokerResource,
		Name: strconv.Itoa(int(brokerID)),
	})
	if err != nil {
		return nil, fmt.Errorf("describing broker %d config: %w", brokerID, err)
	}
	return configEntriesToAPI(entries, clusterReadOnly()), nil
}

// configEntriesToAPI maps sarama config entries to api.BrokerConfigEntry. When
// forceReadOnly is true (read-only cluster) every entry is reported read-only.
func configEntriesToAPI(entries []sarama.ConfigEntry, forceReadOnly bool) []api.BrokerConfigEntry {
	out := make([]api.BrokerConfigEntry, 0, len(entries))
	for _, e := range entries {
		syns := make([]api.BrokerConfigSynonym, 0, len(e.Synonyms))
		for _, s := range e.Synonyms {
			syns = append(syns, api.BrokerConfigSynonym{
				Name:   s.ConfigName,
				Value:  s.ConfigValue,
				Source: configSourceString(s.Source),
			})
		}
		out = append(out, api.BrokerConfigEntry{
			Name:      e.Name,
			Value:     e.Value,
			Source:    configSourceString(e.Source),
			Sensitive: e.Sensitive,
			ReadOnly:  e.ReadOnly || forceReadOnly,
			Synonyms:  syns,
		})
	}
	return out
}

// configSourceString maps a sarama ConfigSource to a friendly label.
func configSourceString(s sarama.ConfigSource) string {
	switch s {
	case sarama.SourceTopic:
		return "Topic"
	case sarama.SourceDynamicBroker:
		return "Dynamic broker config"
	case sarama.SourceDynamicDefaultBroker:
		return "Dynamic default broker config"
	case sarama.SourceStaticBroker:
		return "Static broker config"
	case sarama.SourceDefault:
		return "Default config"
	default:
		return "Unknown"
	}
}

// AlterBrokerConfig implements api.KafkaDataSource using an incremental SET so
// other dynamic configs are preserved.
func (kp KafkaDataSourceKaf) AlterBrokerConfig(brokerID int32, key, value string) error {
	admin, err := getClusterAdmin()
	if err != nil {
		return err
	}
	brokers, _, err := admin.DescribeCluster()
	if err != nil {
		return fmt.Errorf("describing cluster: %w", err)
	}
	if !brokerExists(brokers, brokerID) {
		return api.BrokerNotFoundError{BrokerID: brokerID}
	}

	v := value
	entries := map[string]sarama.IncrementalAlterConfigsEntry{
		key: {Operation: sarama.IncrementalAlterConfigsOperationSet, Value: &v},
	}
	if err := admin.IncrementalAlterConfig(sarama.BrokerResource, strconv.Itoa(int(brokerID)), entries, false); err != nil {
		return api.InvalidConfigError{Key: key, Reason: err.Error(), Cause: err}
	}
	return nil
}

// AlterReplicaLogDir implements api.KafkaDataSource.
//
// sarama v1.45.1 does not implement AlterReplicaLogDirsRequest (only the API-key
// constant exists), and ClusterAdmin exposes no helper, so there is no protocol
// path to perform the move against a real cluster. We therefore return a typed
// NotSupportedError; the mock datasource implements the full flow for UI work.
func (kp KafkaDataSourceKaf) AlterReplicaLogDir(brokerID int32, topic string, partition int32, logDir string) error {
	return api.NotSupportedError{Operation: "AlterReplicaLogDir"}
}

// GetBrokerMetrics implements api.KafkaDataSource. Per-broker metrics require the
// metrics-collection pipeline (feature 12), which does not exist yet.
func (kp KafkaDataSourceKaf) GetBrokerMetrics(brokerID int32) (string, error) {
	return "", api.MetricsNotAvailableError{BrokerID: brokerID}
}

// GetBrokerLogDirs implements api.KafkaDataSource.
func (kp KafkaDataSourceKaf) GetBrokerLogDirs(brokerIDs []int32) (map[int32][]api.BrokerLogDir, error) {
	admin, err := getClusterAdmin()
	if err != nil {
		return nil, err
	}

	clusterBrokers, _, err := admin.DescribeCluster()
	if err != nil {
		return nil, fmt.Errorf("describing cluster: %w", err)
	}

	// Determine the effective ID set: empty means all cluster brokers; otherwise
	// keep only requested IDs that actually exist in the cluster.
	ids := effectiveBrokerIDs(clusterBrokers, brokerIDs)
	if len(ids) == 0 {
		return map[int32][]api.BrokerLogDir{}, nil
	}

	raw, timedOut := describeLogDirsWithTimeout(admin, ids)
	if timedOut {
		// Timeout is not an error for the UI — return empty so it renders "N/A".
		return map[int32][]api.BrokerLogDir{}, nil
	}
	return logDirsToAPI(raw), nil
}

// describeLogDirsWithTimeout runs DescribeLogDirs under a timeout. On timeout it
// returns (nil, true); on any admin error it returns (nil, false) with empty data.
func describeLogDirsWithTimeout(admin ClusterAdminInterface, ids []int32) (map[int32][]sarama.DescribeLogDirsResponseDirMetadata, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), logDirTimeout)
	defer cancel()

	type result struct {
		dirs map[int32][]sarama.DescribeLogDirsResponseDirMetadata
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		dirs, err := admin.DescribeLogDirs(ids)
		ch <- result{dirs: dirs, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, true
	case r := <-ch:
		if r.err != nil {
			return nil, false
		}
		return r.dirs, false
	}
}

// logDirsToAPI maps sarama log-dir metadata to api.BrokerLogDir.
func logDirsToAPI(raw map[int32][]sarama.DescribeLogDirsResponseDirMetadata) map[int32][]api.BrokerLogDir {
	out := make(map[int32][]api.BrokerLogDir, len(raw))
	for brokerID, dirs := range raw {
		apiDirs := make([]api.BrokerLogDir, 0, len(dirs))
		for _, d := range dirs {
			errStr := ""
			if d.ErrorCode != sarama.ErrNoError {
				errStr = d.ErrorCode.Error()
			}
			topics := make([]api.BrokerLogDirTopic, 0, len(d.Topics))
			for _, t := range d.Topics {
				parts := make([]api.BrokerLogDirPartition, 0, len(t.Partitions))
				for _, p := range t.Partitions {
					parts = append(parts, api.BrokerLogDirPartition{
						Partition: p.PartitionID,
						Size:      p.Size,
						OffsetLag: p.OffsetLag,
					})
				}
				topics = append(topics, api.BrokerLogDirTopic{Topic: t.Topic, Partitions: parts})
			}
			apiDirs = append(apiDirs, api.BrokerLogDir{Path: d.Path, Error: errStr, Topics: topics})
		}
		out[brokerID] = apiDirs
	}
	return out
}

// effectiveBrokerIDs returns brokerIDs filtered to those present in the cluster,
// or all cluster broker IDs when brokerIDs is empty.
func effectiveBrokerIDs(clusterBrokers []*sarama.Broker, brokerIDs []int32) []int32 {
	present := make(map[int32]bool, len(clusterBrokers))
	all := make([]int32, 0, len(clusterBrokers))
	for _, b := range clusterBrokers {
		present[b.ID()] = true
		all = append(all, b.ID())
	}
	if len(brokerIDs) == 0 {
		return all
	}
	ids := make([]int32, 0, len(brokerIDs))
	for _, id := range brokerIDs {
		if present[id] {
			ids = append(ids, id)
		}
	}
	return ids
}

// brokerExists reports whether id is one of the cluster's brokers.
func brokerExists(brokers []*sarama.Broker, id int32) bool {
	for _, b := range brokers {
		if b.ID() == id {
			return true
		}
	}
	return false
}
