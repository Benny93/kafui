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

// fetchTopicOffsets returns earliest/latest offsets per partition. Partitions
// whose lookup failed are missing from the map, and the error then names them
// (the map still holds the partitions that succeeded). It is a seam: the shared
// client talks to a real broker, so tests override this to inject offsets.
var fetchTopicOffsets = func(topic string, partitions []int32) (map[int32]offsets, error) {
	client, err := getSharedClient()
	if err != nil {
		return nil, err
	}
	offs, failed := fetchOffsetsBatched(client, map[string][]int32{topic: partitions})
	out := offs[topic]
	if out == nil {
		out = map[int32]offsets{}
	}
	return out, offsetFailuresError(topic, failed[topic])
}

// offsetFailuresError summarises the partitions of topic whose offset lookup
// failed, wrapping the error of the lowest failed partition. nil when none failed.
func offsetFailuresError(topic string, failed map[int32]error) error {
	if len(failed) == 0 {
		return nil
	}
	ids := make([]int32, 0, len(failed))
	for p := range failed {
		ids = append(ids, p)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return fmt.Errorf("fetching offsets for topic %q: %d partition(s) failed %v: partition %d: %w",
		topic, len(ids), ids, ids[0], failed[ids[0]])
}

// fetchOffsetsBatched looks up the oldest and newest offset of every requested
// partition with one ListOffsets request per leader broker and timestamp, the
// brokers queried in parallel. It replaces two sequential GetOffset round trips
// per partition, which made counting a large cluster take tens of thousands of
// round trips. Partitions whose lookup failed are absent from the result and
// listed with their error in failed.
func fetchOffsetsBatched(client sarama.Client, parts map[string][]int32) (map[string]map[int32]offsets, map[string]map[int32]error) {
	result := make(map[string]map[int32]offsets, len(parts))
	failed := make(map[string]map[int32]error)
	var mu sync.Mutex
	fail := func(topic string, p int32, err error) {
		mu.Lock()
		defer mu.Unlock()
		if failed[topic] == nil {
			failed[topic] = map[int32]error{}
		}
		failed[topic][p] = err
	}

	// One metadata request for every requested topic, so the leader lookups
	// below hit the client's cache instead of refreshing per topic, and see
	// current leaders. A failure here is not fatal: Leader reports it per
	// partition.
	topics := make([]string, 0, len(parts))
	for t, ps := range parts {
		if len(ps) > 0 {
			topics = append(topics, t)
		}
	}
	if len(topics) == 0 {
		return result, failed
	}
	_ = client.RefreshMetadata(topics...)

	type tp struct {
		topic     string
		partition int32
	}
	byBroker := map[*sarama.Broker][]tp{}
	for topic, ps := range parts {
		for _, p := range ps {
			b, err := client.Leader(topic, p)
			if err != nil {
				fail(topic, p, err)
				continue
			}
			byBroker[b] = append(byBroker[b], tp{topic, p})
		}
	}

	version := client.Config().Version
	var wg sync.WaitGroup
	for b, tps := range byBroker {
		wg.Add(1)
		go func(b *sarama.Broker, tps []tp) {
			defer wg.Done()
			var resp [2]*sarama.OffsetResponse // [0] oldest, [1] newest
			for i, ts := range []int64{sarama.OffsetOldest, sarama.OffsetNewest} {
				req := newOffsetRequest(version)
				for _, x := range tps {
					req.AddBlock(x.topic, x.partition, ts, 1)
				}
				r, err := b.GetAvailableOffsets(req)
				if err != nil {
					// Like sarama's own getOffset: drop the connection so the
					// next request reconnects.
					_ = b.Close()
					for _, x := range tps {
						fail(x.topic, x.partition, err)
					}
					return
				}
				resp[i] = r
			}
			for _, x := range tps {
				oldest, err := offsetFromBlock(resp[0], x.topic, x.partition)
				if err == nil {
					var newest int64
					newest, err = offsetFromBlock(resp[1], x.topic, x.partition)
					if err == nil {
						mu.Lock()
						if result[x.topic] == nil {
							result[x.topic] = map[int32]offsets{}
						}
						result[x.topic][x.partition] = offsets{oldest: oldest, newest: newest}
						mu.Unlock()
						continue
					}
				}
				fail(x.topic, x.partition, err)
			}
		}(b, tps)
	}
	wg.Wait()
	return result, failed
}

// newOffsetRequest builds a ListOffsets request at the version sarama's own
// client.GetOffset uses for the configured Kafka version.
func newOffsetRequest(v sarama.KafkaVersion) *sarama.OffsetRequest {
	req := &sarama.OffsetRequest{}
	switch {
	case v.IsAtLeast(sarama.V2_1_0_0):
		req.Version = 4
	case v.IsAtLeast(sarama.V2_0_0_0):
		req.Version = 3
	case v.IsAtLeast(sarama.V0_11_0_0):
		req.Version = 2
	case v.IsAtLeast(sarama.V0_10_1_0):
		req.Version = 1
	}
	return req
}

// offsetFromBlock extracts one partition's offset from a ListOffsets response.
func offsetFromBlock(resp *sarama.OffsetResponse, topic string, partition int32) (int64, error) {
	block := resp.GetBlock(topic, partition)
	if block == nil {
		return -1, sarama.ErrIncompleteResponse
	}
	if !errors.Is(block.Err, sarama.ErrNoError) {
		return -1, block.Err
	}
	if len(block.Offsets) != 1 {
		return -1, sarama.ErrOffsetOutOfRange
	}
	return block.Offsets[0], nil
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
	entries, err := describeTopicConfig(topicName)
	if err != nil && isAuthorizationError(err) {
		return []api.TopicConfigEntry{}, nil
	}
	return entries, err
}

// describeTopicConfig reads a topic's config and, unlike GetTopicConfig,
// returns an authorization failure as an error. Callers that act on the config
// (RecreateTopic, the purge cleanup.policy guard) must not mistake "not allowed
// to read" for "no overrides".
func describeTopicConfig(topicName string) ([]api.TopicConfigEntry, error) {
	admin, err := getClusterAdmin()
	if err != nil {
		return nil, err
	}
	entries, err := admin.DescribeConfig(sarama.ConfigResource{
		Type: sarama.TopicResource,
		Name: topicName,
	})
	if err != nil {
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

// GetTopicDetails implements api.KafkaDataSource. Offsets are best effort: a
// partition whose offsets could not be read reports 0..0. Callers that act on
// offsets (PurgeTopicMessages) fetch them strictly instead.
func (kp KafkaDataSourceKaf) GetTopicDetails(topicName string) (api.TopicDetails, error) {
	t, err := describeTopic(topicName)
	if err != nil {
		return api.TopicDetails{}, err
	}
	offs, _ := fetchTopicOffsets(topicName, partitionIDs(t)) // best effort
	return buildTopicDetails(t, offs), nil
}

// topicMetadataDetails is GetTopicDetails without offsets, for callers that
// only need partitions and replicas (partition count, replication factor).
func topicMetadataDetails(topicName string) (api.TopicDetails, error) {
	t, err := describeTopic(topicName)
	if err != nil {
		return api.TopicDetails{}, err
	}
	return buildTopicDetails(t, nil), nil
}

// describeTopic returns one topic's metadata. Any topic-level error is
// returned: a topic we are not authorized to see, or whose leader is
// unavailable, must not look like a topic with zero partitions, since
// IncreasePartitions and RecreateTopic act on the partition count.
func describeTopic(topicName string) (*sarama.TopicMetadata, error) {
	admin, err := getClusterAdmin()
	if err != nil {
		return nil, err
	}
	md, err := admin.DescribeTopics([]string{topicName})
	if err != nil {
		return nil, fmt.Errorf("describing topic %q: %w", topicName, err)
	}
	if len(md) == 0 || md[0] == nil {
		return nil, api.TopicNotFoundError{TopicName: topicName}
	}
	t := md[0]
	switch {
	case errors.Is(t.Err, sarama.ErrUnknownTopicOrPartition):
		return nil, api.TopicNotFoundError{TopicName: topicName, Cause: t.Err}
	case !errors.Is(t.Err, sarama.ErrNoError):
		return nil, fmt.Errorf("describing topic %q: %w", topicName, t.Err)
	}
	return t, nil
}

// partitionIDs lists the IDs of a topic's partitions.
func partitionIDs(t *sarama.TopicMetadata) []int32 {
	ids := make([]int32, 0, len(t.Partitions))
	for _, p := range t.Partitions {
		if p != nil {
			ids = append(ids, p.ID)
		}
	}
	return ids
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
//
// Only the brokers leading a partition of the requested topics are asked for
// their log dirs, since sizes count leader replicas only. Sizing one topic no
// longer describes every broker in the cluster.
func (kp KafkaDataSourceKaf) GetTopicSizes(topicNames []string) (map[string]int64, error) {
	admin, err := getClusterAdmin()
	if err != nil {
		return nil, err
	}
	md, err := admin.DescribeTopics(topicNames)
	if err != nil {
		return nil, fmt.Errorf("describing topics: %w", err)
	}
	leaders := leadersByTopic(md)
	brokerIDs := leaderBrokerIDs(leaders)
	if len(brokerIDs) == 0 {
		// No partition has a leader, so no broker holds a counted replica.
		return aggregateTopicSizes(nil, leaders), nil
	}

	logDirs, unknown := describeLogDirsWithTimeout(brokerIDs)
	if unknown {
		// The broker never answered, or refused. Sizes are genuinely unknown,
		// not zero: return an empty map so callers render "N/A" rather than a
		// misleading "0 B" for every topic.
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

// leaderBrokerIDs returns the distinct, sorted IDs of the brokers leading at
// least one partition in leaders (a leader of -1 means none).
func leaderBrokerIDs(leaders map[string]map[int32]int32) []int32 {
	seen := map[int32]bool{}
	ids := []int32{}
	for _, parts := range leaders {
		for _, leader := range parts {
			if leader >= 0 && !seen[leader] {
				seen[leader] = true
				ids = append(ids, leader)
			}
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
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
	details, err := topicMetadataDetails(name)
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
	details, err := topicMetadataDetails(name)
	if err != nil {
		return err
	}
	var targets []int32
	for _, p := range details.Partitions {
		if partition == -1 || p.ID == partition {
			targets = append(targets, p.ID)
		}
	}
	if len(targets) == 0 {
		return api.PartitionError{Message: "partition not found", TopicName: name, PartitionID: partition}
	}
	// The high-watermarks are the DeleteRecords targets, so they must be
	// exact: a partition whose offset lookup failed would be "purged" to
	// offset 0, a silent no-op reported as success.
	offs, err := fetchTopicOffsets(name, targets)
	if err != nil {
		return fmt.Errorf("purging messages for %q: %w", name, err)
	}
	offsets := make(map[int32]int64, len(targets))
	for _, id := range targets {
		o, ok := offs[id]
		if !ok {
			return fmt.Errorf("purging messages for %q: no offsets for partition %d", name, id)
		}
		offsets[id] = o.newest
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
// when the key is absent, matching Kafka's default). A config we may not read
// is an error, not "delete": guessing would skip the compact-topic guard.
func (kp KafkaDataSourceKaf) topicCleanupPolicy(name string) (string, error) {
	entries, err := describeTopicConfig(name)
	if err != nil {
		return "", fmt.Errorf("cannot read cleanup.policy of %q: %w", name, err)
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
	details, err := topicMetadataDetails(name)
	if err != nil {
		return err
	}
	// Without a readable config snapshot the recreated topic would silently
	// lose its overrides (retention, cleanup.policy=compact, ...), so abort
	// before anything is deleted.
	cfgEntries, err := describeTopicConfig(name)
	if err != nil {
		return fmt.Errorf("cannot snapshot config for %q before recreate: %w", name, err)
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
	details, err := topicMetadataDetails(name)
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
