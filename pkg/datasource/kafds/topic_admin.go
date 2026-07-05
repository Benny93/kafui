package kafds

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
)

// Bounded-retry knobs (vars so tests can shrink them).
var (
	// topicMetadataRetries/Delay bound the post-create visibility poll (TP-5).
	topicMetadataRetries = 10
	topicMetadataDelay   = 500 * time.Millisecond
	// recreateRetries/Delay bound the delete-then-recreate loop (TP-10).
	recreateRetries = 20
	recreateDelay   = 500 * time.Millisecond
)

// fetchTopicOffsets returns earliest/latest offsets per partition. It is a seam:
// getClient() talks to a real broker, so tests override this to inject offsets.
var fetchTopicOffsets = func(topic string, partitions []int32) (map[int32]offsets, error) {
	client, err := getClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()
	out := make(map[int32]offsets, len(partitions))
	for _, p := range partitions {
		o, err := getOffsets(client, topic, p)
		if err != nil {
			continue // best effort; a partition without offsets is reported as 0..0
		}
		out[p] = *o
	}
	return out, nil
}

// --- TP-2: GetTopicConfig ---

// GetTopicConfig implements api.KafkaDataSource. On an authorization failure it
// returns an empty slice (not an error), per spec.
//
// ponytail: sarama's ClusterAdmin.DescribeConfig does not set IncludeSynonyms,
// so on real clusters config synonyms (and thus derived defaults) may be empty.
// Default derivation below is best-effort and fully exercised by the mock admin,
// which can populate synonyms. Wiring a synonym-aware describe would need a new
// admin method beyond the pass-through interface.
func (kp KafkaDataSourceKaf) GetTopicConfig(topicName string) ([]api.TopicConfigEntry, error) {
	admin, err := getClusterAdmin()
	if err != nil {
		return nil, err
	}
	entries, err := admin.DescribeConfig(sarama.ConfigResource{
		Type: sarama.TopicResource,
		Name: topicName,
	})
	if err != nil {
		if isAuthorizationError(err) {
			return []api.TopicConfigEntry{}, nil
		}
		return nil, fmt.Errorf("describing config for topic %q: %w", topicName, err)
	}
	return topicConfigEntriesToAPI(entries), nil
}

// topicConfigEntriesToAPI is the pure mapping from sarama config entries to
// api.TopicConfigEntry, deriving the default value from synonyms.
func topicConfigEntriesToAPI(entries []sarama.ConfigEntry) []api.TopicConfigEntry {
	out := make([]api.TopicConfigEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, api.TopicConfigEntry{
			Name:      e.Name,
			Value:     e.Value,
			Default:   deriveDefault(e),
			Source:    configSourceString(e.Source),
			Sensitive: e.Sensitive,
			ReadOnly:  e.ReadOnly,
		})
	}
	return out
}

// deriveDefault returns the default value of a config entry: from a
// DEFAULT_CONFIG or STATIC_BROKER_CONFIG synonym when present, otherwise the
// entry's own value when it is itself the default.
func deriveDefault(e sarama.ConfigEntry) string {
	for _, s := range e.Synonyms {
		if s.Source == sarama.SourceDefault || s.Source == sarama.SourceStaticBroker {
			return s.ConfigValue
		}
	}
	if e.Default || e.Source == sarama.SourceDefault {
		return e.Value
	}
	return ""
}

// --- TP-3: GetTopicDetails ---

// GetTopicDetails implements api.KafkaDataSource.
func (kp KafkaDataSourceKaf) GetTopicDetails(topicName string) (api.TopicDetails, error) {
	admin, err := getClusterAdmin()
	if err != nil {
		return api.TopicDetails{}, err
	}
	md, err := admin.DescribeTopics([]string{topicName})
	if err != nil {
		return api.TopicDetails{}, fmt.Errorf("describing topic %q: %w", topicName, err)
	}
	if len(md) == 0 || md[0] == nil {
		return api.TopicDetails{}, api.TopicNotFoundError{TopicName: topicName}
	}
	t := md[0]
	if t.Err != sarama.ErrNoError && errors.Is(t.Err, sarama.ErrUnknownTopicOrPartition) {
		return api.TopicDetails{}, api.TopicNotFoundError{TopicName: topicName, Cause: t.Err}
	}

	ids := make([]int32, 0, len(t.Partitions))
	for _, p := range t.Partitions {
		if p != nil {
			ids = append(ids, p.ID)
		}
	}
	offs, _ := fetchTopicOffsets(topicName, ids) // best effort
	return buildTopicDetails(t, offs), nil
}

// buildTopicDetails is the pure aggregation from topic metadata + offsets to
// api.TopicDetails.
func buildTopicDetails(t *sarama.TopicMetadata, offs map[int32]offsets) api.TopicDetails {
	d := api.TopicDetails{Name: t.Name, IsInternal: t.IsInternal}
	maxRF := 0
	for _, p := range t.Partitions {
		if p == nil {
			continue
		}
		o := offs[p.ID]
		d.Partitions = append(d.Partitions, api.PartitionInfo{
			ID:              p.ID,
			Leader:          p.Leader,
			Replicas:        p.Replicas,
			ISR:             p.Isr,
			OfflineReplicas: p.OfflineReplicas,
			EarliestOffset:  o.oldest,
			LatestOffset:    o.newest,
		})
		d.TotalReplicas += len(p.Replicas)
		d.InSyncReplicas += len(p.Isr)
		if len(p.Isr) < len(p.Replicas) {
			d.UnderReplicatedPartitions++
		}
		if len(p.Replicas) > maxRF {
			maxRF = len(p.Replicas)
		}
	}
	d.ReplicationFactor = int16(maxRF)
	sort.Slice(d.Partitions, func(i, j int) bool { return d.Partitions[i].ID < d.Partitions[j].ID })
	return d
}

// --- TP-4: GetTopicSizes ---

// GetTopicSizes implements api.KafkaDataSource. Sizes count leader replicas only
// so replicated bytes are not double-counted. Best-effort: topics absent from
// metadata are omitted. DescribeLogDirs is bounded by describeLogDirsWithTimeout
// (TP-4/BUG-4) so a broker that doesn't support/answer it (e.g. some managed
// Kafka offerings) degrades to empty sizes instead of hanging the caller — which
// otherwise blocks the same tea.Cmd that resolves the OSR column, making both
// spin forever in the topics table.
func (kp KafkaDataSourceKaf) GetTopicSizes(topicNames []string) (map[string]int64, error) {
	admin, err := getClusterAdmin()
	if err != nil {
		return nil, err
	}
	brokers, _, err := admin.DescribeCluster()
	if err != nil {
		return nil, fmt.Errorf("describing cluster: %w", err)
	}
	brokerIDs := make([]int32, 0, len(brokers))
	for _, b := range brokers {
		brokerIDs = append(brokerIDs, b.ID())
	}

	md, err := admin.DescribeTopics(topicNames)
	if err != nil {
		return nil, fmt.Errorf("describing topics: %w", err)
	}
	leaders := leadersByTopic(md)

	logDirs, timedOut := describeLogDirsWithTimeout(admin, brokerIDs)
	if timedOut {
		// Broker never answered: sizes are genuinely unknown, not zero. Return an
		// empty map so callers render "N/A" rather than a misleading "0 B".
		return map[string]int64{}, nil
	}
	return aggregateTopicSizes(logDirs, leaders), nil
}

// leadersByTopic maps topic -> partition -> leader broker id from metadata.
func leadersByTopic(md []*sarama.TopicMetadata) map[string]map[int32]int32 {
	out := make(map[string]map[int32]int32, len(md))
	for _, t := range md {
		if t == nil || t.Err != sarama.ErrNoError {
			continue
		}
		parts := make(map[int32]int32, len(t.Partitions))
		for _, p := range t.Partitions {
			if p != nil {
				parts[p.ID] = p.Leader
			}
		}
		out[t.Name] = parts
	}
	return out
}

// aggregateTopicSizes sums, per topic, only the sizes of partition replicas held
// by that partition's leader broker.
func aggregateTopicSizes(logDirs map[int32][]sarama.DescribeLogDirsResponseDirMetadata, leaders map[string]map[int32]int32) map[string]int64 {
	sizes := make(map[string]int64)
	// Seed known topics so a topic with a leader but no on-disk data reports 0.
	for topic := range leaders {
		sizes[topic] = 0
	}
	for brokerID, dirs := range logDirs {
		for _, dir := range dirs {
			if dir.ErrorCode != sarama.ErrNoError {
				continue
			}
			for _, t := range dir.Topics {
				parts, ok := leaders[t.Topic]
				if !ok {
					continue
				}
				for _, p := range t.Partitions {
					if parts[p.PartitionID] == brokerID {
						sizes[t.Topic] += p.Size
					}
				}
			}
		}
	}
	return sizes
}

// --- TP-5: CreateTopic ---

// CreateTopic implements api.KafkaDataSource.
func (kp KafkaDataSourceKaf) CreateTopic(name string, numPartitions int32, replicationFactor int16, configs map[string]*string) error {
	admin, err := getClusterAdmin()
	if err != nil {
		return err
	}
	detail := &sarama.TopicDetail{
		NumPartitions:     numPartitions,
		ReplicationFactor: replicationFactor,
		ConfigEntries:     configs,
	}
	if err := admin.CreateTopic(name, detail, false); err != nil {
		return mapTopicCreateError(name, err)
	}
	return waitForTopicVisible(admin, name)
}

// waitForTopicVisible polls DescribeTopics until the topic appears or the bounded
// retry window is exhausted (TP-5).
func waitForTopicVisible(admin ClusterAdminInterface, name string) error {
	for i := 0; i < topicMetadataRetries; i++ {
		md, err := admin.DescribeTopics([]string{name})
		if err == nil && len(md) > 0 && md[0] != nil && md[0].Err == sarama.ErrNoError && md[0].Name == name {
			return nil
		}
		time.Sleep(topicMetadataDelay)
	}
	return api.MetadataTimeoutError{TopicName: name}
}

// mapTopicCreateError translates broker errors into typed API errors.
func mapTopicCreateError(name string, err error) error {
	switch {
	case errors.Is(err, sarama.ErrTopicAlreadyExists):
		return api.TopicAlreadyExistsError{TopicName: name, Cause: err}
	case errors.Is(err, sarama.ErrInvalidTopic),
		errors.Is(err, sarama.ErrInvalidPartitions),
		errors.Is(err, sarama.ErrInvalidReplicationFactor),
		errors.Is(err, sarama.ErrInvalidConfig),
		errors.Is(err, sarama.ErrPolicyViolation):
		return api.TopicValidationError{TopicName: name, Reason: err.Error(), Cause: err}
	default:
		return fmt.Errorf("creating topic %q: %w", name, err)
	}
}

// --- TP-6: DeleteTopic + deletion-capability detection ---

// DeleteTopic implements api.KafkaDataSource.
func (kp KafkaDataSourceKaf) DeleteTopic(name string) error {
	admin, err := getClusterAdmin()
	if err != nil {
		return err
	}
	if err := admin.DeleteTopic(name); err != nil {
		if errors.Is(err, sarama.ErrUnknownTopicOrPartition) {
			return api.TopicNotFoundError{TopicName: name, Cause: err}
		}
		return fmt.Errorf("deleting topic %q: %w", name, err)
	}
	return nil
}

var (
	deletionEnabledCache   = map[string]bool{}
	deletionEnabledCacheMu sync.Mutex
)

// resetTopicDeletionCache clears the per-context capability cache (test helper).
func resetTopicDeletionCache() {
	deletionEnabledCacheMu.Lock()
	deletionEnabledCache = map[string]bool{}
	deletionEnabledCacheMu.Unlock()
}

// IsTopicDeletionEnabled implements api.KafkaDataSource. It reads the controller
// broker's delete.topic.enable config; missing/unparseable defaults to true.
func (kp KafkaDataSourceKaf) IsTopicDeletionEnabled() (bool, error) {
	ctx := kp.GetContext()
	deletionEnabledCacheMu.Lock()
	if v, ok := deletionEnabledCache[ctx]; ok {
		deletionEnabledCacheMu.Unlock()
		return v, nil
	}
	deletionEnabledCacheMu.Unlock()

	admin, err := getClusterAdmin()
	if err != nil {
		return false, err
	}
	_, controllerID, err := admin.DescribeCluster()
	if err != nil {
		return false, fmt.Errorf("describing cluster: %w", err)
	}
	if controllerID < 0 {
		controllerID = 0
	}
	entries, err := admin.DescribeConfig(sarama.ConfigResource{
		Type: sarama.BrokerResource,
		Name: strconv.Itoa(int(controllerID)),
	})
	if err != nil {
		return false, fmt.Errorf("describing broker config: %w", err)
	}
	enabled := parseDeletionEnabled(entries)
	deletionEnabledCacheMu.Lock()
	deletionEnabledCache[ctx] = enabled
	deletionEnabledCacheMu.Unlock()
	return enabled, nil
}

// parseDeletionEnabled reads delete.topic.enable; anything other than an explicit
// "false" (including a missing key) is treated as enabled.
func parseDeletionEnabled(entries []sarama.ConfigEntry) bool {
	for _, e := range entries {
		if e.Name == "delete.topic.enable" {
			b, err := strconv.ParseBool(strings.TrimSpace(e.Value))
			if err != nil {
				return true
			}
			return b
		}
	}
	return true
}

// --- TP-7: UpdateTopicConfig ---

// UpdateTopicConfig implements api.KafkaDataSource via an incremental alter so
// unrelated dynamic configs are preserved. A nil value deletes the key.
func (kp KafkaDataSourceKaf) UpdateTopicConfig(name string, entries map[string]*string) error {
	admin, err := getClusterAdmin()
	if err != nil {
		return err
	}
	ops := make(map[string]sarama.IncrementalAlterConfigsEntry, len(entries))
	for key, val := range entries {
		if val == nil {
			ops[key] = sarama.IncrementalAlterConfigsEntry{Operation: sarama.IncrementalAlterConfigsOperationDelete}
			continue
		}
		v := *val
		ops[key] = sarama.IncrementalAlterConfigsEntry{Operation: sarama.IncrementalAlterConfigsOperationSet, Value: &v}
	}
	if err := admin.IncrementalAlterConfig(sarama.TopicResource, name, ops, false); err != nil {
		return api.InvalidConfigError{Key: name, Reason: err.Error(), Cause: err}
	}
	return nil
}

// --- TP-8: IncreasePartitions ---

// IncreasePartitions implements api.KafkaDataSource. It rejects a decrease or a
// no-op before touching the broker.
func (kp KafkaDataSourceKaf) IncreasePartitions(name string, totalCount int32) error {
	details, err := kp.GetTopicDetails(name)
	if err != nil {
		return err
	}
	current := int32(len(details.Partitions))
	switch {
	case totalCount < current:
		return api.PartitionDecreaseError{TopicName: name, Current: current, Requested: totalCount}
	case totalCount == current:
		return api.PartitionNoopError{TopicName: name, Current: current}
	}
	admin, err := getClusterAdmin()
	if err != nil {
		return err
	}
	if err := admin.CreatePartitions(name, totalCount, nil, false); err != nil {
		return fmt.Errorf("increasing partitions for %q: %w", name, err)
	}
	return nil
}

// --- TP-9: PurgeTopicMessages ---

// PurgeTopicMessages implements api.KafkaDataSource. partition == -1 purges all
// partitions to the high-watermark.
func (kp KafkaDataSourceKaf) PurgeTopicMessages(name string, partition int32) error {
	policy, err := kp.topicCleanupPolicy(name)
	if err != nil {
		return err
	}
	if !cleanupPolicyAllowsDelete(policy) {
		return api.CleanupPolicyError{TopicName: name, Policy: policy}
	}
	details, err := kp.GetTopicDetails(name)
	if err != nil {
		return err
	}
	offsets := make(map[int32]int64)
	for _, p := range details.Partitions {
		if partition == -1 || p.ID == partition {
			offsets[p.ID] = p.LatestOffset
		}
	}
	if len(offsets) == 0 {
		return api.PartitionError{Message: "partition not found", TopicName: name, PartitionID: partition}
	}
	admin, err := getClusterAdmin()
	if err != nil {
		return err
	}
	if err := admin.DeleteRecords(name, offsets); err != nil {
		return fmt.Errorf("purging messages for %q: %w", name, err)
	}
	return nil
}

// topicCleanupPolicy returns the effective cleanup.policy for a topic ("delete"
// when the key is absent, matching Kafka's default).
func (kp KafkaDataSourceKaf) topicCleanupPolicy(name string) (string, error) {
	entries, err := kp.GetTopicConfig(name)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.Name == "cleanup.policy" {
			return e.Value, nil
		}
	}
	return "delete", nil
}

// cleanupPolicyAllowsDelete reports whether a cleanup.policy value permits record
// deletion (i.e. includes "delete").
func cleanupPolicyAllowsDelete(policy string) bool {
	for _, p := range strings.Split(policy, ",") {
		if strings.TrimSpace(p) == "delete" {
			return true
		}
	}
	return false
}

// --- TP-10: RecreateTopic ---

// RecreateTopic implements api.KafkaDataSource: snapshot, delete, then recreate
// with the same partition count / replication factor / non-default configs,
// retrying while the prior instance is still propagating its deletion.
func (kp KafkaDataSourceKaf) RecreateTopic(name string) error {
	details, err := kp.GetTopicDetails(name)
	if err != nil {
		return err
	}
	cfgEntries, err := kp.GetTopicConfig(name)
	if err != nil {
		return err
	}
	numPartitions := int32(len(details.Partitions))
	rf := details.ReplicationFactor
	configs := nonDefaultConfigs(cfgEntries)

	if err := kp.DeleteTopic(name); err != nil {
		if _, ok := err.(api.TopicNotFoundError); !ok {
			return err
		}
	}

	var lastErr error
	for i := 0; i < recreateRetries; i++ {
		lastErr = kp.CreateTopic(name, numPartitions, rf, configs)
		if lastErr == nil {
			return nil
		}
		if _, ok := lastErr.(api.TopicAlreadyExistsError); !ok {
			return lastErr // a non-"still exists" failure is terminal
		}
		time.Sleep(recreateDelay)
	}
	return api.RecreateTimeoutError{TopicName: name, Cause: lastErr}
}

// nonDefaultConfigs extracts topic-level config overrides (Source == "Topic")
// as a create-ready map.
func nonDefaultConfigs(entries []api.TopicConfigEntry) map[string]*string {
	out := make(map[string]*string)
	for _, e := range entries {
		if e.Source == "Topic" {
			v := e.Value
			out[e.Name] = &v
		}
	}
	return out
}

// --- TP-11: ChangeReplicationFactor ---

// ChangeReplicationFactor implements api.KafkaDataSource by computing a balanced
// reassignment across online brokers and applying it.
func (kp KafkaDataSourceKaf) ChangeReplicationFactor(name string, newFactor int16) error {
	details, err := kp.GetTopicDetails(name)
	if err != nil {
		return err
	}
	admin, err := getClusterAdmin()
	if err != nil {
		return err
	}
	brokers, _, err := admin.DescribeCluster()
	if err != nil {
		return fmt.Errorf("describing cluster: %w", err)
	}
	online := make([]int32, 0, len(brokers))
	for _, b := range brokers {
		online = append(online, b.ID())
	}

	current := make([][]int32, len(details.Partitions))
	leaders := make([]int32, len(details.Partitions))
	for i, p := range details.Partitions {
		current[i] = p.Replicas
		leaders[i] = p.Leader
	}

	assignment, err := computeReassignment(current, leaders, online, int(newFactor))
	if err != nil {
		return api.InvalidReplicationFactorError{TopicName: name, Reason: err.Error()}
	}
	if err := admin.AlterPartitionReassignments(name, assignment); err != nil {
		return fmt.Errorf("reassigning replicas for %q: %w", name, err)
	}
	return nil
}

// isAuthorizationError reports whether err is a Kafka authorization failure.
func isAuthorizationError(err error) bool {
	return errors.Is(err, sarama.ErrTopicAuthorizationFailed) ||
		errors.Is(err, sarama.ErrClusterAuthorizationFailed) ||
		errors.Is(err, sarama.ErrGroupAuthorizationFailed)
}
