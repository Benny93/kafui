package kafds

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sync"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
)

// --- test fakes ---

// fakeGroupReader implements groupOffsetReader with static per-partition offsets.
type fakeGroupReader struct {
	newest   map[string]map[int32]int64
	oldest   map[string]map[int32]int64
	errAt    map[string]map[int32]bool // partitions whose GetOffset errors
	coord    int32
	coordErr bool
	closed   bool

	mu          sync.Mutex
	offsetCalls int // GetOffset invocations
}

func (f *fakeGroupReader) GetOffset(topic string, p int32, t int64) (int64, error) {
	f.mu.Lock()
	f.offsetCalls++
	f.mu.Unlock()
	if f.errAt[topic][p] {
		return 0, errors.New("offset unavailable")
	}
	switch t {
	case sarama.OffsetOldest:
		return f.oldest[topic][p], nil
	default: // OffsetNewest
		return f.newest[topic][p], nil
	}
}

func (f *fakeGroupReader) Partitions(topic string) ([]int32, error) {
	var ps []int32
	for p := range f.newest[topic] {
		ps = append(ps, p)
	}
	return ps, nil
}

func (f *fakeGroupReader) Coordinator(group string) (*sarama.Broker, error) {
	if f.coordErr {
		return nil, errors.New("no coordinator")
	}
	return newTestBroker(f.coord, "coord:9092"), nil
}

func (f *fakeGroupReader) Close() error { f.closed = true; return nil }

func installFakeReader(r groupOffsetReader) func() {
	orig := newGroupOffsetReader
	newGroupOffsetReader = func() (groupOffsetReader, error) { return r, nil }
	return func() { newGroupOffsetReader = orig }
}

// encodeAssignment hand-encodes a ConsumerGroupMemberAssignment (v0) matching
// sarama's wire format so GetMemberAssignment decodes it back.
func encodeAssignment(topics map[string][]int32) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, int16(0))           // Version
	_ = binary.Write(buf, binary.BigEndian, int32(len(topics))) // topic array length
	for topic, parts := range topics {
		_ = binary.Write(buf, binary.BigEndian, int16(len(topic)))
		buf.WriteString(topic)
		_ = binary.Write(buf, binary.BigEndian, int32(len(parts)))
		for _, p := range parts {
			_ = binary.Write(buf, binary.BigEndian, p)
		}
	}
	_ = binary.Write(buf, binary.BigEndian, int32(-1)) // UserData: nil
	return buf.Bytes()
}

// memberWithAssignment builds a Sarama group member whose MemberAssignment
// decodes to the given topic->partitions.
func memberWithAssignment(t *testing.T, id, client, host string, topics map[string][]int32) *sarama.GroupMemberDescription {
	t.Helper()
	return &sarama.GroupMemberDescription{
		MemberId:         id,
		ClientId:         client,
		ClientHost:       host,
		MemberAssignment: encodeAssignment(topics),
	}
}

// offsetsResp builds an OffsetFetchResponse from topic->partition->offset.
func offsetsResp(committed map[string]map[int32]int64) *sarama.OffsetFetchResponse {
	resp := &sarama.OffsetFetchResponse{Blocks: map[string]map[int32]*sarama.OffsetFetchResponseBlock{}}
	for topic, parts := range committed {
		resp.Blocks[topic] = map[int32]*sarama.OffsetFetchResponseBlock{}
		for p, off := range parts {
			resp.Blocks[topic][p] = &sarama.OffsetFetchResponseBlock{Offset: off}
		}
	}
	return resp
}

func clearGroupCache() {
	groupDetailCacheMu.Lock()
	groupDetailCache = map[string]groupDetailCacheEntry{}
	groupDetailCacheMu.Unlock()
}

// --- CG-3: GetConsumerGroupDetail ---

func TestGetConsumerGroupDetail(t *testing.T) {
	member := memberWithAssignment(t, "m1", "client-1", "10.0.0.1", map[string][]int32{"t1": {0, 1}})
	admin := &MockClusterAdmin{
		MockConsumerGroups: map[string]string{"g1": "consumer"},
		MockGroupDescriptions: []*sarama.GroupDescription{{
			GroupId:      "g1",
			State:        "Stable",
			ProtocolType: "consumer",
			Protocol:     "range",
			Members:      map[string]*sarama.GroupMemberDescription{"m1": member},
		}},
		// committed on p0 only; p1 assigned but uncommitted.
		MockGroupOffsets: offsetsResp(map[string]map[int32]int64{"t1": {0: 100}}),
	}
	restore := installMockAdmin(admin)
	defer restore()
	restoreReader := installFakeReader(&fakeGroupReader{
		newest: map[string]map[int32]int64{"t1": {0: 150, 1: 500}},
		coord:  7,
	})
	defer restoreReader()

	detail, err := brokerDS().GetConsumerGroupDetail("g1")
	assert.NoError(t, err)
	assert.Equal(t, api.GroupStateStable, detail.State)
	assert.Equal(t, "range", detail.PartitionAssignor)
	assert.False(t, detail.IsSimple)
	assert.Equal(t, int32(7), detail.CoordinatorID)
	assert.Len(t, detail.Members, 1)
	assert.Len(t, detail.TopicOffsets, 2)

	byPart := map[int32]api.PartitionOffset{}
	for _, po := range detail.TopicOffsets {
		byPart[po.Partition] = po
	}
	// p0: committed 100, end 150 -> lag 50, attributed to m1.
	assert.NotNil(t, byPart[0].CommittedOffset)
	assert.Equal(t, int64(100), *byPart[0].CommittedOffset)
	assert.NotNil(t, byPart[0].Lag)
	assert.Equal(t, int64(50), *byPart[0].Lag)
	assert.Equal(t, "m1", byPart[0].MemberID)
	assert.Equal(t, "10.0.0.1", byPart[0].MemberHost)
	// p1: assigned but uncommitted -> nil committed, nil lag, member attributed.
	assert.Nil(t, byPart[1].CommittedOffset)
	assert.Nil(t, byPart[1].Lag)
	assert.Equal(t, "m1", byPart[1].MemberID)
}

func TestGetConsumerGroupDetail_NotFound(t *testing.T) {
	admin := &MockClusterAdmin{MockConsumerGroups: map[string]string{"other": "consumer"}}
	restore := installMockAdmin(admin)
	defer restore()
	restoreReader := installFakeReader(&fakeGroupReader{})
	defer restoreReader()

	_, err := brokerDS().GetConsumerGroupDetail("missing")
	var nf api.GroupNotFoundError
	assert.True(t, errors.As(err, &nf))
	assert.Equal(t, "missing", nf.GroupID)
}

func TestGetConsumerGroupDetail_EndOffsetUnavailable(t *testing.T) {
	admin := &MockClusterAdmin{
		MockConsumerGroups:    map[string]string{"g1": "consumer"},
		MockGroupDescriptions: []*sarama.GroupDescription{{GroupId: "g1", State: "Empty", ProtocolType: "consumer"}},
		MockGroupOffsets:      offsetsResp(map[string]map[int32]int64{"t1": {0: 100}}),
	}
	restore := installMockAdmin(admin)
	defer restore()
	restoreReader := installFakeReader(&fakeGroupReader{
		errAt: map[string]map[int32]bool{"t1": {0: true}},
	})
	defer restoreReader()

	detail, err := brokerDS().GetConsumerGroupDetail("g1")
	assert.NoError(t, err)
	assert.Len(t, detail.TopicOffsets, 1)
	// committed present but end unavailable -> lag defined as 0.
	assert.NotNil(t, detail.TopicOffsets[0].Lag)
	assert.Equal(t, int64(0), *detail.TopicOffsets[0].Lag)
}

// --- CG-4: GetConsumerGroupDetails (batch) ---

func TestGetConsumerGroupDetails_BatchBestEffort(t *testing.T) {
	clearGroupCache()
	memberG1 := memberWithAssignment(t, "m1", "c1", "h1", map[string][]int32{"t1": {0}})
	admin := &MockClusterAdmin{
		MockGroupDescriptions: []*sarama.GroupDescription{
			{GroupId: "g1", State: "Stable", ProtocolType: "consumer", Protocol: "range", Members: map[string]*sarama.GroupMemberDescription{"m1": memberG1}},
			{GroupId: "g2", State: "Empty", ProtocolType: "consumer"},
			{GroupId: "g3", ErrorCode: int16(sarama.ErrGroupAuthorizationFailed)}, // describe failure
		},
		MockGroupOffsets: offsetsResp(map[string]map[int32]int64{"t1": {0: 10}}),
	}
	restore := installMockAdmin(admin)
	defer restore()
	restoreReader := installFakeReader(&fakeGroupReader{
		newest: map[string]map[int32]int64{"t1": {0: 60}},
		coord:  1,
	})
	defer restoreReader()

	rows, err := brokerDS().GetConsumerGroupDetails([]string{"g1", "g2", "g3"})
	assert.NoError(t, err)
	assert.Len(t, rows, 3)
	byName := map[string]api.ConsumerGroup{}
	for _, r := range rows {
		byName[r.Name] = r
	}
	// g1 enriched
	assert.Equal(t, api.GroupStateStable, byName["g1"].State)
	assert.Equal(t, 1, byName["g1"].MemberCount)
	assert.NotNil(t, byName["g1"].Lag)
	assert.Equal(t, int64(50), *byName["g1"].Lag)
	// g3 failed describe -> Unknown, nil lag
	assert.Equal(t, api.GroupStateUnknown, byName["g3"].State)
	assert.Nil(t, byName["g3"].Lag)
}

func TestGetConsumerGroupDetails_TopicCountDistinct(t *testing.T) {
	clearGroupCache()
	// committed on t1; assigned to t1 and t2 -> distinct topics = 2.
	member := memberWithAssignment(t, "m1", "c1", "h1", map[string][]int32{"t1": {0}, "t2": {0}})
	admin := &MockClusterAdmin{
		MockGroupDescriptions: []*sarama.GroupDescription{
			{GroupId: "g1", State: "Stable", ProtocolType: "consumer", Members: map[string]*sarama.GroupMemberDescription{"m1": member}},
		},
		MockGroupOffsets: offsetsResp(map[string]map[int32]int64{"t1": {0: 5}}),
	}
	restore := installMockAdmin(admin)
	defer restore()
	restoreReader := installFakeReader(&fakeGroupReader{newest: map[string]map[int32]int64{"t1": {0: 5}}})
	defer restoreReader()

	rows, err := brokerDS().GetConsumerGroupDetails([]string{"g1"})
	assert.NoError(t, err)
	assert.Equal(t, 2, rows[0].TopicCount)
}

func TestGetConsumerGroupDetails_CacheHitAvoidsDescribe(t *testing.T) {
	clearGroupCache()
	admin := &MockClusterAdmin{
		MockGroupDescriptions: []*sarama.GroupDescription{{GroupId: "g1", State: "Stable", ProtocolType: "consumer"}},
		MockGroupOffsets:      offsetsResp(nil),
	}
	restore := installMockAdmin(admin)
	defer restore()
	restoreReader := installFakeReader(&fakeGroupReader{})
	defer restoreReader()

	_, err := brokerDS().GetConsumerGroupDetails([]string{"g1"})
	assert.NoError(t, err)

	// Second call: swap in an admin that would fail describe. Cache hit must
	// avoid touching it.
	failAdmin := &MockClusterAdmin{ShouldFailDescribeGroups: true}
	restore2 := installMockAdmin(failAdmin)
	defer restore2()
	rows, err := brokerDS().GetConsumerGroupDetails([]string{"g1"})
	assert.NoError(t, err)
	assert.Equal(t, api.GroupStateStable, rows[0].State)
}

// --- CG-5: GetConsumerGroupsForTopic ---

func TestGetConsumerGroupsForTopic(t *testing.T) {
	clearGroupCache()
	// g1 related by assignment only (no committed for t1).
	g1Member := memberWithAssignment(t, "m1", "c1", "h1", map[string][]int32{"t1": {0}})
	// g2 unrelated (works on t9).
	g2Member := memberWithAssignment(t, "m2", "c2", "h2", map[string][]int32{"t9": {0}})
	admin := &MockClusterAdmin{
		MockConsumerGroups: map[string]string{"g1": "consumer", "g2": "consumer"},
		MockGroupDescriptions: []*sarama.GroupDescription{
			{GroupId: "g1", State: "Stable", ProtocolType: "consumer", Members: map[string]*sarama.GroupMemberDescription{"m1": g1Member}},
			{GroupId: "g2", State: "Stable", ProtocolType: "consumer", Members: map[string]*sarama.GroupMemberDescription{"m2": g2Member}},
		},
		MockGroupOffsets: offsetsResp(nil), // no committed offsets for either
	}
	restore := installMockAdmin(admin)
	defer restore()
	restoreReader := installFakeReader(&fakeGroupReader{newest: map[string]map[int32]int64{"t1": {0: 100}}})
	defer restoreReader()

	groups, err := brokerDS().GetConsumerGroupsForTopic("t1")
	assert.NoError(t, err)
	assert.Len(t, groups, 1)
	assert.Equal(t, "g1", groups[0].Name)
	assert.Equal(t, 1, groups[0].MemberCount)
	// No committed offsets for the topic -> undefined (nil) lag.
	assert.Nil(t, groups[0].Lag)
}

// filterRecordingAdmin records the partition filter passed to each
// ListConsumerGroupOffsets call.
type filterRecordingAdmin struct {
	*MockClusterAdmin
	mu      sync.Mutex
	filters []map[string][]int32
}

func (a *filterRecordingAdmin) ListConsumerGroupOffsets(group string, tps map[string][]int32) (*sarama.OffsetFetchResponse, error) {
	a.mu.Lock()
	a.filters = append(a.filters, tps)
	a.mu.Unlock()
	return a.MockClusterAdmin.ListConsumerGroupOffsets(group, tps)
}

func TestGetConsumerGroupsForTopic_FiltersOffsetsAndReadsEndsOnce(t *testing.T) {
	clearGroupCache()
	groups := map[string]string{}
	var descs []*sarama.GroupDescription
	for _, g := range []string{"g1", "g2", "g3"} {
		groups[g] = "consumer"
		descs = append(descs, &sarama.GroupDescription{GroupId: g, State: "Empty", ProtocolType: "consumer"})
	}
	mock := &MockClusterAdmin{
		MockConsumerGroups:    groups,
		MockGroupDescriptions: descs,
		// Every group committed t1/p0=40; p1 is uncommitted (-1).
		MockGroupOffsets: offsetsResp(map[string]map[int32]int64{"t1": {0: 40, 1: -1}}),
	}
	restore := installMockAdmin(mock)
	defer restore()
	admin := &filterRecordingAdmin{MockClusterAdmin: mock}
	kafkaClientFactory.(*MockKafkaClientFactory).MockClusterAdmin = admin
	reader := &fakeGroupReader{newest: map[string]map[int32]int64{"t1": {0: 100, 1: 7}}}
	restoreReader := installFakeReader(reader)
	defer restoreReader()

	rows, err := brokerDS().GetConsumerGroupsForTopic("t1")
	assert.NoError(t, err)
	assert.Len(t, rows, 3)
	for _, r := range rows {
		assert.NotNil(t, r.Lag)
		assert.Equal(t, int64(60), *r.Lag, r.Name)
	}
	// Each OffsetFetch is restricted to the topic's partitions.
	assert.Len(t, admin.filters, 3)
	for _, f := range admin.filters {
		assert.ElementsMatch(t, []int32{0, 1}, f["t1"])
		assert.Len(t, f, 1)
	}
	// The shared end offset of t1/p0 is read once, not once per group.
	assert.Equal(t, 1, reader.offsetCalls)
}

func TestGetConsumerGroupDetails_CacheScopedToCluster(t *testing.T) {
	clearGroupCache()
	restore := installMockAdmin(&MockClusterAdmin{
		MockGroupDescriptions: []*sarama.GroupDescription{{GroupId: "g1", State: "Stable", ProtocolType: "consumer"}},
	})
	defer restore()
	restoreReader := installFakeReader(&fakeGroupReader{})
	defer restoreReader()

	currentCluster.Name = "dev"
	rows, err := brokerDS().GetConsumerGroupDetails([]string{"g1"})
	assert.NoError(t, err)
	assert.Equal(t, api.GroupStateStable, rows[0].State)

	// Same group name on another cluster must miss the cache and hit the
	// (here failing) admin instead of returning dev's row.
	kafkaClientFactory.(*MockKafkaClientFactory).MockClusterAdmin = &MockClusterAdmin{ShouldFailDescribeGroups: true}
	resetSharedClients() // the admin is cached per cluster; drop it so the swap takes effect
	currentCluster.Name = "prod"
	_, err = brokerDS().GetConsumerGroupDetails([]string{"g1"})
	assert.Error(t, err)

	// Back on dev the cached row is still served.
	currentCluster.Name = "dev"
	rows, err = brokerDS().GetConsumerGroupDetails([]string{"g1"})
	assert.NoError(t, err)
	assert.Equal(t, api.GroupStateStable, rows[0].State)
}

// TestFetchNewestOffsets_BatchesPerBroker checks that a real sarama client gets
// one ListOffsets request per leader broker rather than one per partition.
func TestFetchNewestOffsets_BatchesPerBroker(t *testing.T) {
	broker := sarama.NewMockBroker(t, 1)
	defer broker.Close()
	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(broker.Addr(), broker.BrokerID()).
			SetLeader("t1", 0, broker.BrokerID()).
			SetLeader("t1", 1, broker.BrokerID()).
			SetLeader("t2", 0, broker.BrokerID()),
		"OffsetRequest": sarama.NewMockOffsetResponse(t).
			SetOffset("t1", 0, sarama.OffsetNewest, 150).
			SetOffset("t1", 1, sarama.OffsetNewest, 500).
			SetOffset("t2", 0, sarama.OffsetNewest, 9),
	})
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_1_0_0
	client, err := sarama.NewClient([]string{broker.Addr()}, cfg)
	if !assert.NoError(t, err) {
		return
	}
	defer client.Close()

	ends := fetchNewestOffsets(client, map[api.TopicPartition]struct{}{
		{Topic: "t1", Partition: 0}: {},
		{Topic: "t1", Partition: 1}: {},
		{Topic: "t2", Partition: 0}: {},
	})
	assert.Equal(t, map[api.TopicPartition]int64{
		{Topic: "t1", Partition: 0}: 150,
		{Topic: "t1", Partition: 1}: 500,
		{Topic: "t2", Partition: 0}: 9,
	}, ends)

	offsetRequests := 0
	for _, rr := range broker.History() {
		if _, ok := rr.Request.(*sarama.OffsetRequest); ok {
			offsetRequests++
		}
	}
	assert.Equal(t, 1, offsetRequests)
}

// --- CG-6: delete ---

func TestDeleteConsumerGroup(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		admin := &MockClusterAdmin{}
		restore := installMockAdmin(admin)
		defer restore()
		err := brokerDS().DeleteConsumerGroup("g1")
		assert.NoError(t, err)
		assert.Equal(t, []string{"g1"}, admin.DeleteConsumerGroupCalls)
	})

	t.Run("not found mapped", func(t *testing.T) {
		admin := &MockClusterAdmin{DeleteConsumerGroupErr: sarama.ErrGroupIDNotFound}
		restore := installMockAdmin(admin)
		defer restore()
		err := brokerDS().DeleteConsumerGroup("g1")
		var nf api.GroupNotFoundError
		assert.True(t, errors.As(err, &nf))
	})

	t.Run("not empty mapped", func(t *testing.T) {
		admin := &MockClusterAdmin{DeleteConsumerGroupErr: sarama.ErrNonEmptyGroup}
		restore := installMockAdmin(admin)
		defer restore()
		err := brokerDS().DeleteConsumerGroup("g1")
		var ne api.GroupNotEmptyError
		assert.True(t, errors.As(err, &ne))
	})
}

func TestDeleteConsumerGroupOffsets_OnlyNamedTopic(t *testing.T) {
	admin := &MockClusterAdmin{
		MockGroupOffsets: offsetsResp(map[string]map[int32]int64{
			"t1": {0: 5, 1: 6},
			"t2": {0: 9},
		}),
	}
	restore := installMockAdmin(admin)
	defer restore()

	err := brokerDS().DeleteConsumerGroupOffsets("g1", "t1")
	assert.NoError(t, err)
	// Only t1's partitions deleted (0 and 1), never t2.
	assert.Equal(t, []DeleteOffsetCall{
		{Group: "g1", Topic: "t1", Partition: 0},
		{Group: "g1", Topic: "t1", Partition: 1},
	}, admin.DeleteConsumerGroupOffsetCall)
}
