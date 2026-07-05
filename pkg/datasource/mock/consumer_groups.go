package mock

import (
	"context"
	"sort"

	"github.com/Benny93/kafui/pkg/api"
)

// mockGroup is the rich in-memory model backing the consumer-group detail and
// mutation methods. Offsets are topic -> partition -> offset.
type mockGroup struct {
	state        string
	protocolType string
	assignor     string
	coordinator  int32
	members      []api.GroupMember
	committed    map[string]map[int32]int64 // committed offsets (absent = uncommitted)
	end          map[string]map[int32]int64 // partition high-water marks
}

// ensureGroupState lazily builds a deterministic set of mock groups covering the
// scenarios the tests need: active groups with lag, an Empty group (for reset /
// offset deletion), a group with an assigned-but-uncommitted partition, and a
// group with no committed offsets at all (undefined lag).
func (kp *KafkaDataSourceMock) ensureGroupState() {
	kp.groupMu.Lock()
	defer kp.groupMu.Unlock()
	if kp.groups != nil {
		return
	}

	kp.groups = map[string]*mockGroup{
		"order-processor": {
			state: api.GroupStateStable, protocolType: "consumer", assignor: "range", coordinator: 1,
			members: []api.GroupMember{
				{ConsumerID: "consumer-1", ClientID: "order-svc", Host: "10.0.0.1", Assignments: []api.TopicPartition{{Topic: "order-events", Partition: 0}}},
				{ConsumerID: "consumer-2", ClientID: "order-svc", Host: "10.0.0.2", Assignments: []api.TopicPartition{{Topic: "order-events", Partition: 1}}},
				{ConsumerID: "consumer-3", ClientID: "order-svc", Host: "10.0.0.3", Assignments: []api.TopicPartition{{Topic: "order-events", Partition: 2}}},
			},
			committed: map[string]map[int32]int64{"order-events": {0: 100, 1: 200, 2: 150}},
			end:       map[string]map[int32]int64{"order-events": {0: 150, 1: 250, 2: 150}},
		},
		"payment-service": {
			state: api.GroupStateStable, protocolType: "consumer", assignor: "roundrobin", coordinator: 2,
			members: []api.GroupMember{
				{ConsumerID: "pay-1", ClientID: "pay-svc", Host: "10.0.1.1", Assignments: []api.TopicPartition{{Topic: "payment-events", Partition: 0}}},
				{ConsumerID: "pay-2", ClientID: "pay-svc", Host: "10.0.1.2", Assignments: []api.TopicPartition{{Topic: "payment-events", Partition: 1}}},
			},
			committed: map[string]map[int32]int64{"payment-events": {0: 500, 1: 400}},
			end:       map[string]map[int32]int64{"payment-events": {0: 600, 1: 400}},
		},
		"analytics-consumer": {
			// One partition assigned but not yet committed (clickstream p1).
			state: api.GroupStateStable, protocolType: "consumer", assignor: "range", coordinator: 3,
			members: []api.GroupMember{
				{ConsumerID: "ana-1", ClientID: "analytics", Host: "10.0.2.1", Assignments: []api.TopicPartition{{Topic: "clickstream-events", Partition: 0}, {Topic: "clickstream-events", Partition: 1}}},
			},
			committed: map[string]map[int32]int64{"clickstream-events": {0: 1000}},
			end:       map[string]map[int32]int64{"clickstream-events": {0: 1500, 1: 2000}},
		},
		"inventory-sync": {
			// Empty group with committed offsets — usable for reset & offset delete.
			state: api.GroupStateEmpty, protocolType: "consumer", assignor: "range", coordinator: 1,
			committed: map[string]map[int32]int64{"inventory-events": {0: 10, 1: 20}},
			end:       map[string]map[int32]int64{"inventory-events": {0: 60, 1: 20}},
		},
		"load-test-processor": {
			// Empty group with no committed offsets at all — undefined (nil) lag.
			state: api.GroupStateEmpty, protocolType: "consumer", assignor: "", coordinator: 2,
			committed: map[string]map[int32]int64{},
			end:       map[string]map[int32]int64{},
		},
	}
}

func i64(v int64) *int64 { return &v }

// GetConsumerGroupDetail implements api.KafkaDataSource (CG-3).
func (kp *KafkaDataSourceMock) GetConsumerGroupDetail(groupID string) (api.ConsumerGroupDetail, error) {
	kp.ensureGroupState()
	kp.groupMu.Lock()
	defer kp.groupMu.Unlock()

	g, ok := kp.groups[groupID]
	if !ok {
		return api.ConsumerGroupDetail{}, api.GroupNotFoundError{GroupID: groupID}
	}

	// Attribute assigned partitions to members.
	type owner struct{ id, host string }
	ownerOf := map[api.TopicPartition]owner{}
	for _, m := range g.members {
		for _, a := range m.Assignments {
			ownerOf[a] = owner{id: m.ConsumerID, host: m.Host}
		}
	}

	tpSet := map[api.TopicPartition]struct{}{}
	for topic, parts := range g.committed {
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
		if parts, ok := g.committed[tp.Topic]; ok {
			if c, ok := parts[tp.Partition]; ok {
				committedVal = i64(c)
			}
		}
		po.CommittedOffset = committedVal
		endKnown := false
		if parts, ok := g.end[tp.Topic]; ok {
			if e, ok := parts[tp.Partition]; ok {
				po.EndOffset = e
				endKnown = true
			}
		}
		if committedVal != nil {
			var lag int64
			if endKnown {
				if lag = po.EndOffset - *committedVal; lag < 0 {
					lag = 0
				}
			}
			po.Lag = i64(lag)
		}
		offsets = append(offsets, po)
	}
	sort.Slice(offsets, func(i, j int) bool {
		if offsets[i].Topic != offsets[j].Topic {
			return offsets[i].Topic < offsets[j].Topic
		}
		return offsets[i].Partition < offsets[j].Partition
	})

	members := make([]api.GroupMember, len(g.members))
	copy(members, g.members)

	return api.ConsumerGroupDetail{
		GroupID:           groupID,
		State:             g.state,
		ProtocolType:      g.protocolType,
		PartitionAssignor: g.assignor,
		IsSimple:          g.protocolType != "consumer",
		CoordinatorID:     g.coordinator,
		Members:           members,
		TopicOffsets:      offsets,
	}, nil
}

// mockTotalLag sums (end-committed) over the given committed offsets; nil when
// there are no committed offsets.
func (g *mockGroup) totalLag(committed map[string]map[int32]int64) *int64 {
	if len(committed) == 0 {
		return nil
	}
	var total int64
	for topic, parts := range committed {
		for p, off := range parts {
			if ends, ok := g.end[topic]; ok {
				if e, ok := ends[p]; ok {
					if l := e - off; l > 0 {
						total += l
					}
				}
			}
		}
	}
	return &total
}

func (g *mockGroup) distinctTopics() int {
	set := map[string]struct{}{}
	for topic := range g.committed {
		set[topic] = struct{}{}
	}
	for _, m := range g.members {
		for _, a := range m.Assignments {
			set[a.Topic] = struct{}{}
		}
	}
	return len(set)
}

// GetConsumerGroupDetails implements api.KafkaDataSource (CG-4).
func (kp *KafkaDataSourceMock) GetConsumerGroupDetails(groupIDs []string) ([]api.ConsumerGroup, error) {
	kp.ensureGroupState()
	kp.groupMu.Lock()
	defer kp.groupMu.Unlock()

	out := make([]api.ConsumerGroup, 0, len(groupIDs))
	for _, id := range groupIDs {
		g, ok := kp.groups[id]
		if !ok {
			out = append(out, api.ConsumerGroup{Name: id, State: api.GroupStateUnknown, CoordinatorID: -1})
			continue
		}
		out = append(out, api.ConsumerGroup{
			Name:              id,
			State:             g.state,
			Consumers:         len(g.members),
			MemberCount:       len(g.members),
			TopicCount:        g.distinctTopics(),
			Lag:               g.totalLag(g.committed),
			CoordinatorID:     g.coordinator,
			PartitionAssignor: g.assignor,
			IsSimple:          g.protocolType != "consumer",
		})
	}
	return out, nil
}

// GetConsumerGroupsForTopic implements api.KafkaDataSource (CG-5).
func (kp *KafkaDataSourceMock) GetConsumerGroupsForTopic(topic string) ([]api.ConsumerGroup, error) {
	kp.ensureGroupState()
	kp.groupMu.Lock()
	defer kp.groupMu.Unlock()

	names := make([]string, 0, len(kp.groups))
	for name := range kp.groups {
		names = append(names, name)
	}
	sort.Strings(names)

	var result []api.ConsumerGroup
	for _, name := range names {
		g := kp.groups[name]
		_, hasCommitted := g.committed[topic]
		memberCount := 0
		for _, m := range g.members {
			for _, a := range m.Assignments {
				if a.Topic == topic {
					memberCount++
					break
				}
			}
		}
		if !hasCommitted && memberCount == 0 {
			continue
		}
		var lag *int64
		if hasCommitted {
			lag = g.totalLag(map[string]map[int32]int64{topic: g.committed[topic]})
		}
		result = append(result, api.ConsumerGroup{
			Name:              name,
			State:             g.state,
			Consumers:         memberCount,
			MemberCount:       memberCount,
			TopicCount:        1,
			Lag:               lag,
			CoordinatorID:     g.coordinator,
			PartitionAssignor: g.assignor,
			IsSimple:          g.protocolType != "consumer",
		})
	}
	return result, nil
}

// DeleteConsumerGroup implements api.KafkaDataSource (CG-6).
func (kp *KafkaDataSourceMock) DeleteConsumerGroup(groupID string) error {
	kp.ensureGroupState()
	kp.groupMu.Lock()
	defer kp.groupMu.Unlock()

	g, ok := kp.groups[groupID]
	if !ok {
		return api.GroupNotFoundError{GroupID: groupID}
	}
	if g.state != api.GroupStateEmpty && g.state != api.GroupStateDead {
		return api.GroupNotEmptyError{GroupID: groupID, State: g.state}
	}
	delete(kp.groups, groupID)
	return nil
}

// DeleteConsumerGroupOffsets implements api.KafkaDataSource (CG-6). Only the
// named topic's committed offsets are removed.
func (kp *KafkaDataSourceMock) DeleteConsumerGroupOffsets(groupID string, topic string) error {
	kp.ensureGroupState()
	kp.groupMu.Lock()
	defer kp.groupMu.Unlock()

	g, ok := kp.groups[groupID]
	if !ok {
		return api.GroupNotFoundError{GroupID: groupID}
	}
	delete(g.committed, topic)
	return nil
}

// ResetConsumerGroupOffsets implements api.KafkaDataSource (CG-7, CG-8).
func (kp *KafkaDataSourceMock) ResetConsumerGroupOffsets(ctx context.Context, req api.OffsetResetRequest) error {
	// Shared validation (mirrors kafds).
	switch req.Mode {
	case api.OffsetResetEarliest, api.OffsetResetLatest:
	case api.OffsetResetTimestamp:
		if req.Timestamp == nil {
			return api.InvalidOffsetResetError{Reason: "timestamp mode requires a timestamp"}
		}
	case api.OffsetResetExplicit:
		if len(req.PartitionOffsets) == 0 {
			return api.InvalidOffsetResetError{Reason: "explicit mode requires per-partition offsets"}
		}
	default:
		return api.InvalidOffsetResetError{Reason: "unrecognized reset mode"}
	}

	kp.ensureGroupState()
	kp.groupMu.Lock()
	defer kp.groupMu.Unlock()

	g, ok := kp.groups[req.GroupID]
	if !ok {
		return api.GroupNotFoundError{GroupID: req.GroupID}
	}
	if g.state != api.GroupStateEmpty && g.state != api.GroupStateDead {
		return api.GroupNotEmptyError{GroupID: req.GroupID, State: g.state}
	}

	// Resolve target partitions (empty => all end-offset partitions of the topic).
	partitions := req.Partitions
	if len(partitions) == 0 {
		for p := range g.end[req.Topic] {
			partitions = append(partitions, p)
		}
	}

	if g.committed[req.Topic] == nil {
		g.committed[req.Topic] = map[int32]int64{}
	}
	for _, p := range partitions {
		oldest := int64(0)
		newest := g.end[req.Topic][p]
		var target int64
		switch req.Mode {
		case api.OffsetResetEarliest:
			target = oldest
		case api.OffsetResetLatest, api.OffsetResetTimestamp:
			target = newest
		case api.OffsetResetExplicit:
			target = req.PartitionOffsets[p]
			if target < oldest {
				target = oldest
			}
			if target > newest {
				target = newest
			}
		}
		g.committed[req.Topic][p] = target
	}
	return nil
}
