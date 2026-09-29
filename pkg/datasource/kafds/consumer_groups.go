package kafds

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/IBM/sarama"
)

// groupOffsetReader abstracts the client-side reads the consumer-group code
// needs. sarama.Client satisfies it; tests substitute a fake via
// newGroupOffsetReader.
type groupOffsetReader interface {
	GetOffset(topic string, partition int32, time int64) (int64, error)
	Partitions(topic string) ([]int32, error)
	Coordinator(consumerGroup string) (*sarama.Broker, error)
	Close() error
}

// newGroupOffsetReader creates a client-side reader. Replaceable in tests.
var newGroupOffsetReader = func() (groupOffsetReader, error) {
	return getClient()
}

// normalizeGroupState maps Sarama's backend state strings to the canonical
// api.GroupState* values.
func normalizeGroupState(s string) string {
	switch s {
	case "Stable":
		return api.GroupStateStable
	case "PreparingRebalance":
		return api.GroupStatePreparingRebalance
	case "CompletingRebalance":
		return api.GroupStateCompletingRebalance
	case "Empty":
		return api.GroupStateEmpty
	case "Dead":
		return api.GroupStateDead
	default:
		return api.GroupStateUnknown
	}
}

// findGroupDesc returns the description with the given group id, or nil.
func findGroupDesc(descs []*sarama.GroupDescription, groupID string) *sarama.GroupDescription {
	for _, d := range descs {
		if d != nil && d.GroupId == groupID {
			return d
		}
	}
	return nil
}

// committedOffsets extracts topic->partition->offset for partitions that have a
// committed offset (Offset >= 0).
func committedOffsets(resp *sarama.OffsetFetchResponse) map[string]map[int32]int64 {
	out := map[string]map[int32]int64{}
	if resp == nil {
		return out
	}
	for topic, parts := range resp.Blocks {
		for p, block := range parts {
			if block == nil || block.Offset < 0 {
				continue
			}
			if out[topic] == nil {
				out[topic] = map[int32]int64{}
			}
			out[topic][p] = block.Offset
		}
	}
	return out
}

// groupMembers converts a Sarama group description's members (with decoded
// assignments) to api.GroupMember values.
func groupMembers(desc *sarama.GroupDescription) []api.GroupMember {
	members := make([]api.GroupMember, 0, len(desc.Members))
	for _, m := range desc.Members {
		gm := api.GroupMember{ConsumerID: m.MemberId, ClientID: m.ClientId, Host: m.ClientHost}
		if assign, err := m.GetMemberAssignment(); err == nil && assign != nil {
			for topic, parts := range assign.Topics {
				for _, p := range parts {
					gm.Assignments = append(gm.Assignments, api.TopicPartition{Topic: topic, Partition: p})
				}
			}
		}
		members = append(members, gm)
	}
	return members
}

// coordinatorID resolves the coordinator broker id for a group, best-effort
// (returns -1 on failure).
func coordinatorID(reader groupOffsetReader, groupID string) int32 {
	if b, err := reader.Coordinator(groupID); err == nil && b != nil {
		return b.ID()
	}
	return -1
}

// computeTotalLag sums (end - committed) across all committed partitions.
// Returns nil (undefined) when there are no committed offsets at all; a partition
// missing from ends (end offset unreadable) contributes 0.
func computeTotalLag(committed map[string]map[int32]int64, ends map[api.TopicPartition]int64) *int64 {
	if len(committed) == 0 {
		return nil
	}
	var total int64
	for topic, parts := range committed {
		for p, off := range parts {
			end, ok := ends[api.TopicPartition{Topic: topic, Partition: p}]
			if !ok {
				continue // contributes 0
			}
			if l := end - off; l > 0 {
				total += l
			}
		}
	}
	return &total
}

// addCommitted adds every committed partition to the set.
func addCommitted(set map[api.TopicPartition]struct{}, committed map[string]map[int32]int64) {
	for topic, parts := range committed {
		for p := range parts {
			set[api.TopicPartition{Topic: topic, Partition: p}] = struct{}{}
		}
	}
}

// groupFanout bounds the concurrent per-group broker calls (OffsetFetch,
// FindCoordinator) made while enriching a batch of groups.
const groupFanout = 8

// forEachBounded calls fn(i) for every i in [0, n), running at most limit calls
// at once, and returns when all of them have finished.
func forEachBounded(n, limit int, fn func(i int)) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, limit)
	for i := 0; i < n; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			fn(i)
		}(i)
	}
	wg.Wait()
}

// leaderLookup is the part of sarama.Client that fetchNewestOffsets needs to
// batch ListOffsets requests per leader broker.
type leaderLookup interface {
	Leader(topic string, partitionID int32) (*sarama.Broker, error)
	Config() *sarama.Config
}

// offsetRequestVersion mirrors the ListOffsets version sarama's client.GetOffset
// picks for the configured Kafka version.
func offsetRequestVersion(cfg *sarama.Config) int16 {
	switch {
	case cfg == nil:
		return 0
	case cfg.Version.IsAtLeast(sarama.V2_1_0_0):
		return 4
	case cfg.Version.IsAtLeast(sarama.V2_0_0_0):
		return 3
	case cfg.Version.IsAtLeast(sarama.V0_11_0_0):
		return 2
	case cfg.Version.IsAtLeast(sarama.V0_10_1_0):
		return 1
	default:
		return 0
	}
}

// fetchNewestOffsets returns the log-end offset of every partition in tps. When
// the reader is a sarama client it sends one ListOffsets request per leader
// broker, in parallel, instead of one request per partition. Partitions the
// batch could not resolve fall back to reader.GetOffset, which refreshes
// metadata and retries. A partition whose end offset cannot be read is absent.
func fetchNewestOffsets(reader groupOffsetReader, tps map[api.TopicPartition]struct{}) map[api.TopicPartition]int64 {
	out := make(map[api.TopicPartition]int64, len(tps))
	if len(tps) == 0 {
		return out
	}
	var (
		mu      sync.Mutex
		pending []api.TopicPartition
	)
	if ll, ok := reader.(leaderLookup); ok {
		type batch struct {
			broker *sarama.Broker
			req    *sarama.OffsetRequest
			tps    []api.TopicPartition
		}
		version := offsetRequestVersion(ll.Config())
		byBroker := map[int32]*batch{}
		for tp := range tps {
			b, err := ll.Leader(tp.Topic, tp.Partition)
			if err != nil || b == nil {
				pending = append(pending, tp)
				continue
			}
			bt := byBroker[b.ID()]
			if bt == nil {
				bt = &batch{broker: b, req: &sarama.OffsetRequest{Version: version}}
				byBroker[b.ID()] = bt
			}
			bt.req.AddBlock(tp.Topic, tp.Partition, sarama.OffsetNewest, 1)
			bt.tps = append(bt.tps, tp)
		}
		var wg sync.WaitGroup
		for _, bt := range byBroker {
			wg.Add(1)
			go func(bt *batch) {
				defer wg.Done()
				resp, err := bt.broker.GetAvailableOffsets(bt.req)
				mu.Lock()
				defer mu.Unlock()
				for _, tp := range bt.tps {
					var block *sarama.OffsetResponseBlock
					if err == nil && resp != nil {
						block = resp.GetBlock(tp.Topic, tp.Partition)
					}
					if block == nil || !errors.Is(block.Err, sarama.ErrNoError) || len(block.Offsets) != 1 {
						pending = append(pending, tp)
						continue
					}
					out[tp] = block.Offsets[0]
				}
			}(bt)
		}
		wg.Wait()
	} else {
		for tp := range tps {
			pending = append(pending, tp)
		}
	}
	for _, tp := range pending {
		if end, err := reader.GetOffset(tp.Topic, tp.Partition, sarama.OffsetNewest); err == nil {
			out[tp] = end
		}
	}
	return out
}

// distinctTopics returns the union of committed-offset topics and member
// assignment topics.
func distinctTopics(committed map[string]map[int32]int64, members []api.GroupMember) int {
	set := map[string]struct{}{}
	for topic := range committed {
		set[topic] = struct{}{}
	}
	for _, m := range members {
		for _, a := range m.Assignments {
			set[a.Topic] = struct{}{}
		}
	}
	return len(set)
}

// GetConsumerGroupDetail implements api.KafkaDataSource (CG-3).
func (kp KafkaDataSourceKaf) GetConsumerGroupDetail(groupID string) (api.ConsumerGroupDetail, error) {
	admin, err := getClusterAdmin()
	if err != nil {
		return api.ConsumerGroupDetail{}, err
	}

	names, err := admin.ListConsumerGroups()
	if err != nil {
		return api.ConsumerGroupDetail{}, fmt.Errorf("listing consumer groups: %w", err)
	}
	if _, ok := names[groupID]; !ok {
		return api.ConsumerGroupDetail{}, api.GroupNotFoundError{GroupID: groupID}
	}

	descs, err := admin.DescribeConsumerGroups([]string{groupID})
	if err != nil {
		return api.ConsumerGroupDetail{}, fmt.Errorf("describing consumer group %q: %w", groupID, err)
	}
	desc := findGroupDesc(descs, groupID)
	if desc == nil {
		return api.ConsumerGroupDetail{}, api.GroupNotFoundError{GroupID: groupID}
	}

	offsetsResp, err := admin.ListConsumerGroupOffsets(groupID, nil)
	if err != nil {
		return api.ConsumerGroupDetail{}, fmt.Errorf("listing offsets for group %q: %w", groupID, err)
	}

	reader, err := newGroupOffsetReader()
	if err != nil {
		return api.ConsumerGroupDetail{}, err
	}
	defer reader.Close()

	return buildGroupDetail(desc, offsetsResp, reader), nil
}

// buildGroupDetail assembles a full ConsumerGroupDetail from a describe result,
// committed offsets and a client-side reader for end offsets.
func buildGroupDetail(desc *sarama.GroupDescription, offsetsResp *sarama.OffsetFetchResponse, reader groupOffsetReader) api.ConsumerGroupDetail {
	committed := committedOffsets(offsetsResp)
	members := groupMembers(desc)

	// Attribute each assigned partition to its member.
	type owner struct{ id, host string }
	ownerOf := map[api.TopicPartition]owner{}
	for _, m := range members {
		for _, a := range m.Assignments {
			ownerOf[a] = owner{id: m.ConsumerID, host: m.Host}
		}
	}

	// Union of committed and assigned partitions.
	tpSet := map[api.TopicPartition]struct{}{}
	addCommitted(tpSet, committed)
	for tp := range ownerOf {
		tpSet[tp] = struct{}{}
	}
	ends := fetchNewestOffsets(reader, tpSet)

	offsets := make([]api.PartitionOffset, 0, len(tpSet))
	for tp := range tpSet {
		po := api.PartitionOffset{Topic: tp.Topic, Partition: tp.Partition, EndOffset: -1}
		if o, ok := ownerOf[tp]; ok {
			po.MemberID = o.id
			po.MemberHost = o.host
		}
		var committedVal *int64
		if parts, ok := committed[tp.Topic]; ok {
			if c, ok := parts[tp.Partition]; ok {
				cv := c
				committedVal = &cv
			}
		}
		po.CommittedOffset = committedVal

		end, ok := ends[tp]
		if ok {
			po.EndOffset = end
		}
		if committedVal != nil {
			var lag int64
			if ok {
				if lag = end - *committedVal; lag < 0 {
					lag = 0
				}
			}
			po.Lag = &lag // 0 when end offset unavailable
		}
		offsets = append(offsets, po)
	}
	sort.Slice(offsets, func(i, j int) bool {
		if offsets[i].Topic != offsets[j].Topic {
			return offsets[i].Topic < offsets[j].Topic
		}
		return offsets[i].Partition < offsets[j].Partition
	})

	return api.ConsumerGroupDetail{
		GroupID:           desc.GroupId,
		State:             normalizeGroupState(desc.State),
		ProtocolType:      desc.ProtocolType,
		PartitionAssignor: desc.Protocol,
		IsSimple:          desc.ProtocolType != "consumer",
		CoordinatorID:     coordinatorID(reader, desc.GroupId),
		Members:           members,
		TopicOffsets:      offsets,
	}
}

// --- CG-4: batched enrichment with a short-lived cache ---

type groupDetailCacheEntry struct {
	group api.ConsumerGroup
	at    time.Time
}

var (
	// groupDetailCache is keyed by groupCacheKey (cluster + group id) so a
	// cluster switch never serves another cluster's group of the same name.
	groupDetailCache   = map[string]groupDetailCacheEntry{}
	groupDetailCacheMu sync.Mutex
	groupDetailTTL     = 30 * time.Second
	// groupDetailCacheGen is bumped by invalidateGroupCache. Enrichment runs
	// without the lock, so a batch that started before an invalidation must
	// not write its (possibly pre-mutation) rows back.
	groupDetailCacheGen uint64
)

// groupCacheKey scopes a group id to the active cluster.
func groupCacheKey(groupID string) string {
	cluster := ""
	if currentCluster != nil {
		cluster = currentCluster.Name
	}
	return cluster + "\x00" + groupID
}

// GetConsumerGroupDetails implements api.KafkaDataSource (CG-4).
func (kp KafkaDataSourceKaf) GetConsumerGroupDetails(groupIDs []string) ([]api.ConsumerGroup, error) {
	if len(groupIDs) == 0 {
		return []api.ConsumerGroup{}, nil
	}

	// Resolve cache hits first; collect misses to describe. Keys are fixed up
	// front so rows are cached under the cluster they were fetched from.
	keys := make(map[string]string, len(groupIDs))
	for _, id := range groupIDs {
		keys[id] = groupCacheKey(id)
	}
	cached := map[string]api.ConsumerGroup{}
	var misses []string
	groupDetailCacheMu.Lock()
	now := time.Now()
	for _, id := range groupIDs {
		if e, ok := groupDetailCache[keys[id]]; ok && now.Sub(e.at) < groupDetailTTL {
			cached[id] = e.group
		} else {
			misses = append(misses, id)
		}
	}
	gen := groupDetailCacheGen
	groupDetailCacheMu.Unlock()

	if len(misses) > 0 {
		admin, err := getClusterAdmin()
		if err != nil {
			return nil, err
		}

		descs, err := admin.DescribeConsumerGroups(misses)
		if err != nil {
			return nil, fmt.Errorf("describing consumer groups: %w", err)
		}

		reader, err := newGroupOffsetReader()
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		// Network enrichment runs without the cache lock; only the writes below
		// take it, so concurrent callers (and cache hits) are not blocked.
		rows := enrichGroups(misses, descs, admin, reader)

		groupDetailCacheMu.Lock()
		at := time.Now()
		for k, e := range groupDetailCache {
			if at.Sub(e.at) >= groupDetailTTL {
				delete(groupDetailCache, k)
			}
		}
		for i, id := range misses {
			cached[id] = rows[i]
			if gen == groupDetailCacheGen {
				groupDetailCache[keys[id]] = groupDetailCacheEntry{group: rows[i], at: at}
			}
		}
		groupDetailCacheMu.Unlock()
	}

	out := make([]api.ConsumerGroup, 0, len(groupIDs))
	for _, id := range groupIDs {
		out = append(out, cached[id])
	}
	return out, nil
}

// enrichGroups builds enriched list rows for a batch of groups, index-aligned
// with names. Per-group offset fetches run through a bounded worker pool and
// all end offsets are read in one batched lookup. Best-effort: a group that
// failed to describe keeps state Unknown and nil lag.
func enrichGroups(names []string, descs []*sarama.GroupDescription, admin ClusterAdminInterface, reader groupOffsetReader) []api.ConsumerGroup {
	type fetched struct {
		desc      *sarama.GroupDescription
		committed map[string]map[int32]int64
		coord     int32
	}
	per := make([]fetched, len(names))
	forEachBounded(len(names), groupFanout, func(i int) {
		name := names[i]
		desc := findGroupDesc(descs, name)
		if desc == nil || desc.ErrorCode != 0 {
			return
		}
		offsetsResp, err := admin.ListConsumerGroupOffsets(name, nil)
		if err != nil {
			shared.Log.Warn("enrichGroups: failed to list offsets", "group", name, "err", err)
		}
		per[i] = fetched{desc: desc, committed: committedOffsets(offsetsResp), coord: coordinatorID(reader, name)}
	})

	tps := map[api.TopicPartition]struct{}{}
	for _, f := range per {
		addCommitted(tps, f.committed)
	}
	ends := fetchNewestOffsets(reader, tps)

	rows := make([]api.ConsumerGroup, len(names))
	for i, name := range names {
		f := per[i]
		if f.desc == nil {
			rows[i] = api.ConsumerGroup{Name: name, State: api.GroupStateUnknown, CoordinatorID: -1}
			continue
		}
		members := groupMembers(f.desc)
		rows[i] = api.ConsumerGroup{
			Name:              name,
			State:             normalizeGroupState(f.desc.State),
			Consumers:         len(members),
			MemberCount:       len(members),
			TopicCount:        distinctTopics(f.committed, members),
			Lag:               computeTotalLag(f.committed, ends),
			CoordinatorID:     f.coord,
			PartitionAssignor: f.desc.Protocol,
			IsSimple:          f.desc.ProtocolType != "consumer",
		}
	}
	return rows
}

// --- CG-5: topic-scoped listing ---

// GetConsumerGroupsForTopic implements api.KafkaDataSource (CG-5).
func (kp KafkaDataSourceKaf) GetConsumerGroupsForTopic(topic string) ([]api.ConsumerGroup, error) {
	admin, err := getClusterAdmin()
	if err != nil {
		return nil, err
	}

	names, err := admin.ListConsumerGroups()
	if err != nil {
		return nil, fmt.Errorf("listing consumer groups: %w", err)
	}
	allNames := make([]string, 0, len(names))
	for name := range names {
		allNames = append(allNames, name)
	}
	sort.Strings(allNames)

	reader, err := newGroupOffsetReader()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// Only this topic's committed offsets matter, so restrict each OffsetFetch
	// to its partitions. Without partition metadata fall back to fetching all.
	var filter map[string][]int32
	if parts, err := reader.Partitions(topic); err == nil && len(parts) > 0 {
		filter = map[string][]int32{topic: parts}
	}

	type candidate struct {
		name string
		desc *sarama.GroupDescription
	}
	const chunkSize = 50
	var cands []candidate
	for start := 0; start < len(allNames); start += chunkSize {
		end := start + chunkSize
		if end > len(allNames) {
			end = len(allNames)
		}
		chunk := allNames[start:end]
		descs, err := admin.DescribeConsumerGroups(chunk)
		if err != nil {
			return nil, fmt.Errorf("describing consumer groups: %w", err)
		}
		for _, name := range chunk {
			desc := findGroupDesc(descs, name)
			if desc == nil || desc.ErrorCode != 0 {
				continue
			}
			cands = append(cands, candidate{name: name, desc: desc})
		}
	}

	type scoped struct {
		row       api.ConsumerGroup
		committed map[int32]int64
		ok        bool
	}
	per := make([]scoped, len(cands))
	forEachBounded(len(cands), groupFanout, func(i int) {
		row, committed, ok := scopeGroupToTopic(cands[i].name, cands[i].desc, admin, reader, topic, filter)
		per[i] = scoped{row: row, committed: committed, ok: ok}
	})

	// The topic's end offsets are the same for every group: read them once.
	tps := map[api.TopicPartition]struct{}{}
	for _, s := range per {
		if s.ok {
			addCommitted(tps, map[string]map[int32]int64{topic: s.committed})
		}
	}
	ends := fetchNewestOffsets(reader, tps)

	var result []api.ConsumerGroup
	for _, s := range per {
		if !s.ok {
			continue
		}
		// Topic-scoped lag: nil when the group has no committed offsets for the topic.
		if len(s.committed) > 0 {
			s.row.Lag = computeTotalLag(map[string]map[int32]int64{topic: s.committed}, ends)
		}
		result = append(result, s.row)
	}
	return result, nil
}

// scopeGroupToTopic returns a topic-scoped list row (Lag left nil), the group's
// committed offsets for the topic, and whether the group is related to the
// topic (has committed offsets for it OR an assigned partition). filter is
// passed to ListConsumerGroupOffsets (nil fetches every topic).
func scopeGroupToTopic(name string, desc *sarama.GroupDescription, admin ClusterAdminInterface, reader groupOffsetReader, topic string, filter map[string][]int32) (api.ConsumerGroup, map[int32]int64, bool) {
	offsetsResp, _ := admin.ListConsumerGroupOffsets(name, filter)
	committed := committedOffsets(offsetsResp)
	members := groupMembers(desc)

	_, hasCommitted := committed[topic]

	// Members assigned at least one partition of the topic.
	memberCount := 0
	for _, m := range members {
		for _, a := range m.Assignments {
			if a.Topic == topic {
				memberCount++
				break
			}
		}
	}

	if !hasCommitted && memberCount == 0 {
		return api.ConsumerGroup{}, nil, false
	}

	return api.ConsumerGroup{
		Name:              name,
		State:             normalizeGroupState(desc.State),
		Consumers:         memberCount,
		MemberCount:       memberCount,
		TopicCount:        1,
		CoordinatorID:     coordinatorID(reader, name),
		PartitionAssignor: desc.Protocol,
		IsSimple:          desc.ProtocolType != "consumer",
	}, committed[topic], true
}

// --- CG-6: deletion ---

// mapGroupError maps Sarama's group KErrors to typed api errors.
func mapGroupError(groupID string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sarama.ErrGroupIDNotFound) || errors.Is(err, sarama.ErrInvalidGroupId) {
		return api.GroupNotFoundError{GroupID: groupID, Cause: err}
	}
	if errors.Is(err, sarama.ErrNonEmptyGroup) {
		return api.GroupNotEmptyError{GroupID: groupID, Cause: err}
	}
	return err
}

// DeleteConsumerGroup implements api.KafkaDataSource (CG-6).
func (kp KafkaDataSourceKaf) DeleteConsumerGroup(groupID string) error {
	admin, err := getClusterAdmin()
	if err != nil {
		return err
	}

	invalidateGroupCache(groupID)
	if err := admin.DeleteConsumerGroup(groupID); err != nil {
		return mapGroupError(groupID, err)
	}
	return nil
}

// DeleteConsumerGroupOffsets implements api.KafkaDataSource (CG-6). It deletes
// only the named topic's committed offsets, leaving other topics intact.
func (kp KafkaDataSourceKaf) DeleteConsumerGroupOffsets(groupID string, topic string) error {
	admin, err := getClusterAdmin()
	if err != nil {
		return err
	}

	offsetsResp, err := admin.ListConsumerGroupOffsets(groupID, nil)
	if err != nil {
		return fmt.Errorf("listing offsets for group %q: %w", groupID, err)
	}
	committed := committedOffsets(offsetsResp)
	invalidateGroupCache(groupID)

	partitions := committed[topic]
	// Delete deterministically for stable test behaviour.
	ps := make([]int32, 0, len(partitions))
	for p := range partitions {
		ps = append(ps, p)
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i] < ps[j] })
	for _, p := range ps {
		if err := admin.DeleteConsumerGroupOffset(groupID, topic, p); err != nil {
			return mapGroupError(groupID, err)
		}
	}
	return nil
}

// invalidateGroupCache drops any cached enrichment for a group after a mutation.
func invalidateGroupCache(groupID string) {
	groupDetailCacheMu.Lock()
	delete(groupDetailCache, groupCacheKey(groupID))
	groupDetailCacheGen++
	groupDetailCacheMu.Unlock()
}
