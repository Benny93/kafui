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
// whose end offset cannot be read contributes 0.
func computeTotalLag(committed map[string]map[int32]int64, reader groupOffsetReader) *int64 {
	if len(committed) == 0 {
		return nil
	}
	var total int64
	for topic, parts := range committed {
		for p, off := range parts {
			end, err := reader.GetOffset(topic, p, sarama.OffsetNewest)
			if err != nil {
				continue // contributes 0
			}
			if l := end - off; l > 0 {
				total += l
			}
		}
	}
	return &total
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
	defer admin.Close()

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
	for topic, parts := range committed {
		for p := range parts {
			tpSet[api.TopicPartition{Topic: topic, Partition: p}] = struct{}{}
		}
	}
	for tp := range ownerOf {
		tpSet[tp] = struct{}{}
	}

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

		end, err := reader.GetOffset(tp.Topic, tp.Partition, sarama.OffsetNewest)
		if err == nil {
			po.EndOffset = end
		}
		if committedVal != nil {
			var lag int64
			if err == nil {
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
	groupDetailCache   = map[string]groupDetailCacheEntry{}
	groupDetailCacheMu sync.Mutex
	groupDetailTTL     = 30 * time.Second
)

// GetConsumerGroupDetails implements api.KafkaDataSource (CG-4).
func (kp KafkaDataSourceKaf) GetConsumerGroupDetails(groupIDs []string) ([]api.ConsumerGroup, error) {
	if len(groupIDs) == 0 {
		return []api.ConsumerGroup{}, nil
	}

	// Resolve cache hits first; collect misses to describe.
	cached := map[string]api.ConsumerGroup{}
	var misses []string
	groupDetailCacheMu.Lock()
	now := time.Now()
	for _, id := range groupIDs {
		if e, ok := groupDetailCache[id]; ok && now.Sub(e.at) < groupDetailTTL {
			cached[id] = e.group
		} else {
			misses = append(misses, id)
		}
	}
	groupDetailCacheMu.Unlock()

	if len(misses) > 0 {
		admin, err := getClusterAdmin()
		if err != nil {
			return nil, err
		}
		defer admin.Close()

		descs, err := admin.DescribeConsumerGroups(misses)
		if err != nil {
			return nil, fmt.Errorf("describing consumer groups: %w", err)
		}

		reader, err := newGroupOffsetReader()
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		groupDetailCacheMu.Lock()
		for _, id := range misses {
			row := enrichGroup(id, findGroupDesc(descs, id), admin, reader)
			cached[id] = row
			groupDetailCache[id] = groupDetailCacheEntry{group: row, at: time.Now()}
		}
		groupDetailCacheMu.Unlock()
	}

	out := make([]api.ConsumerGroup, 0, len(groupIDs))
	for _, id := range groupIDs {
		out = append(out, cached[id])
	}
	return out, nil
}

// enrichGroup builds an enriched list row for a single group. Best-effort: a
// group that failed to describe keeps state Unknown and nil lag.
func enrichGroup(name string, desc *sarama.GroupDescription, admin ClusterAdminInterface, reader groupOffsetReader) api.ConsumerGroup {
	if desc == nil || desc.ErrorCode != 0 {
		return api.ConsumerGroup{Name: name, State: api.GroupStateUnknown, CoordinatorID: -1}
	}

	offsetsResp, err := admin.ListConsumerGroupOffsets(name, nil)
	if err != nil {
		shared.Log.Warn("enrichGroup: failed to list offsets", "group", name, "err", err)
	}
	committed := committedOffsets(offsetsResp)
	members := groupMembers(desc)

	return api.ConsumerGroup{
		Name:              name,
		State:             normalizeGroupState(desc.State),
		Consumers:         len(members),
		MemberCount:       len(members),
		TopicCount:        distinctTopics(committed, members),
		Lag:               computeTotalLag(committed, reader),
		CoordinatorID:     coordinatorID(reader, name),
		PartitionAssignor: desc.Protocol,
		IsSimple:          desc.ProtocolType != "consumer",
	}
}

// --- CG-5: topic-scoped listing ---

// GetConsumerGroupsForTopic implements api.KafkaDataSource (CG-5).
func (kp KafkaDataSourceKaf) GetConsumerGroupsForTopic(topic string) ([]api.ConsumerGroup, error) {
	admin, err := getClusterAdmin()
	if err != nil {
		return nil, err
	}
	defer admin.Close()

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

	const chunkSize = 50
	var result []api.ConsumerGroup
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
			if row, ok := scopeGroupToTopic(name, desc, admin, reader, topic); ok {
				result = append(result, row)
			}
		}
	}
	return result, nil
}

// scopeGroupToTopic returns a topic-scoped list row and whether the group is
// related to the topic (has committed offsets for it OR an assigned partition).
func scopeGroupToTopic(name string, desc *sarama.GroupDescription, admin ClusterAdminInterface, reader groupOffsetReader, topic string) (api.ConsumerGroup, bool) {
	offsetsResp, _ := admin.ListConsumerGroupOffsets(name, nil)
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
		return api.ConsumerGroup{}, false
	}

	// Topic-scoped lag: nil when the group has no committed offsets for the topic.
	var lag *int64
	if hasCommitted {
		scoped := map[string]map[int32]int64{topic: committed[topic]}
		lag = computeTotalLag(scoped, reader)
	}

	return api.ConsumerGroup{
		Name:              name,
		State:             normalizeGroupState(desc.State),
		Consumers:         memberCount,
		MemberCount:       memberCount,
		TopicCount:        1,
		Lag:               lag,
		CoordinatorID:     coordinatorID(reader, name),
		PartitionAssignor: desc.Protocol,
		IsSimple:          desc.ProtocolType != "consumer",
	}, true
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
	defer admin.Close()

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
	defer admin.Close()

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
	delete(groupDetailCache, groupID)
	groupDetailCacheMu.Unlock()
}
