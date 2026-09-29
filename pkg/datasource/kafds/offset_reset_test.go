package kafds

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
)

// fakeResetter implements offsetResetter with static offsets and records commits.
type fakeResetter struct {
	newest     map[string]map[int32]int64
	oldest     map[string]map[int32]int64
	tsOffset   map[string]map[int32]int64 // returned for timestamp lookups (t > 0)
	partitions map[string][]int32
	committed  map[int32]int64 // captured on Commit
	commitErr  error
}

func (f *fakeResetter) GetOffset(topic string, p int32, t int64) (int64, error) {
	switch t {
	case sarama.OffsetOldest:
		return f.oldest[topic][p], nil
	case sarama.OffsetNewest:
		return f.newest[topic][p], nil
	default: // timestamp lookup
		if m, ok := f.tsOffset[topic]; ok {
			if v, ok := m[p]; ok {
				return v, nil
			}
		}
		return -1, nil // no record at/after timestamp
	}
}

func (f *fakeResetter) Partitions(topic string) ([]int32, error) {
	return f.partitions[topic], nil
}

func (f *fakeResetter) Commit(groupID, topic string, offsets map[int32]int64) error {
	if f.commitErr != nil {
		return f.commitErr
	}
	f.committed = offsets
	return nil
}

func (f *fakeResetter) Close() error { return nil }

func installFakeResetter(r offsetResetter) func() {
	orig := newOffsetResetter
	newOffsetResetter = func(groupID string) (offsetResetter, error) { return r, nil }
	return func() { newOffsetResetter = orig }
}

// emptyGroupAdmin returns an admin whose group is Empty (reset precondition ok).
func emptyGroupAdmin(groupID string) *MockClusterAdmin {
	return &MockClusterAdmin{
		MockConsumerGroups:    map[string]string{groupID: "consumer"},
		MockGroupDescriptions: []*sarama.GroupDescription{{GroupId: groupID, State: "Empty", ProtocolType: "consumer"}},
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

// --- CG-7: validation ---

func TestResetOffsets_Validation(t *testing.T) {
	restore := installMockAdmin(emptyGroupAdmin("g1"))
	defer restore()
	restoreR := installFakeResetter(&fakeResetter{})
	defer restoreR()

	tests := []struct {
		name string
		req  api.OffsetResetRequest
	}{
		{"unknown mode", api.OffsetResetRequest{GroupID: "g1", Topic: "t1", Mode: "bogus"}},
		{"timestamp without ts", api.OffsetResetRequest{GroupID: "g1", Topic: "t1", Mode: api.OffsetResetTimestamp}},
		{"explicit without offsets", api.OffsetResetRequest{GroupID: "g1", Topic: "t1", Mode: api.OffsetResetExplicit}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := brokerDS().ResetConsumerGroupOffsets(context.Background(), tt.req)
			var ivr api.InvalidOffsetResetError
			assert.True(t, errors.As(err, &ivr), "want InvalidOffsetResetError, got %v", err)
		})
	}
}

// --- CG-7: precondition ---

func TestResetOffsets_RejectsActiveGroup(t *testing.T) {
	admin := &MockClusterAdmin{
		MockConsumerGroups:    map[string]string{"g1": "consumer"},
		MockGroupDescriptions: []*sarama.GroupDescription{{GroupId: "g1", State: "Stable", ProtocolType: "consumer"}},
	}
	restore := installMockAdmin(admin)
	defer restore()
	restoreR := installFakeResetter(&fakeResetter{})
	defer restoreR()

	err := brokerDS().ResetConsumerGroupOffsets(context.Background(), api.OffsetResetRequest{
		GroupID: "g1", Topic: "t1", Mode: api.OffsetResetEarliest,
	})
	var ne api.GroupNotEmptyError
	assert.True(t, errors.As(err, &ne))
	assert.Equal(t, api.GroupStateStable, ne.State)
}

func TestResetOffsets_GroupNotFound(t *testing.T) {
	restore := installMockAdmin(&MockClusterAdmin{MockConsumerGroups: map[string]string{}})
	defer restore()
	restoreR := installFakeResetter(&fakeResetter{})
	defer restoreR()

	err := brokerDS().ResetConsumerGroupOffsets(context.Background(), api.OffsetResetRequest{
		GroupID: "missing", Topic: "t1", Mode: api.OffsetResetEarliest,
	})
	var nf api.GroupNotFoundError
	assert.True(t, errors.As(err, &nf))
}

// --- CG-7: earliest/latest + partition scope ---

func TestResetOffsets_EarliestLatestAndScope(t *testing.T) {
	tests := []struct {
		name       string
		mode       api.OffsetResetMode
		partitions []int32
		want       map[int32]int64
	}{
		{
			name: "earliest all partitions",
			mode: api.OffsetResetEarliest,
			want: map[int32]int64{0: 10, 1: 20},
		},
		{
			name: "latest all partitions",
			mode: api.OffsetResetLatest,
			want: map[int32]int64{0: 100, 1: 200},
		},
		{
			name:       "latest single partition scope",
			mode:       api.OffsetResetLatest,
			partitions: []int32{1},
			want:       map[int32]int64{1: 200},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := installMockAdmin(emptyGroupAdmin("g1"))
			defer restore()
			r := &fakeResetter{
				oldest:     map[string]map[int32]int64{"t1": {0: 10, 1: 20}},
				newest:     map[string]map[int32]int64{"t1": {0: 100, 1: 200}},
				partitions: map[string][]int32{"t1": {0, 1}},
			}
			restoreR := installFakeResetter(r)
			defer restoreR()

			err := brokerDS().ResetConsumerGroupOffsets(context.Background(), api.OffsetResetRequest{
				GroupID: "g1", Topic: "t1", Mode: tt.mode, Partitions: tt.partitions,
			})
			assert.NoError(t, err)
			assert.Equal(t, tt.want, r.committed)
		})
	}
}

// --- CG-8: timestamp + explicit clamping ---

func TestResetOffsets_TimestampMode(t *testing.T) {
	restore := installMockAdmin(emptyGroupAdmin("g1"))
	defer restore()
	r := &fakeResetter{
		newest:     map[string]map[int32]int64{"t1": {0: 500, 1: 900}},
		partitions: map[string][]int32{"t1": {0, 1}},
		// p0 has a record at/after ts (returns 250); p1 has none (-1) -> fallback end 900.
		tsOffset: map[string]map[int32]int64{"t1": {0: 250}},
	}
	restoreR := installFakeResetter(r)
	defer restoreR()

	ts := time.Now()
	err := brokerDS().ResetConsumerGroupOffsets(context.Background(), api.OffsetResetRequest{
		GroupID: "g1", Topic: "t1", Mode: api.OffsetResetTimestamp, Timestamp: ptrTime(ts),
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(250), r.committed[0]) // timestamp hit
	assert.Equal(t, int64(900), r.committed[1]) // -1 -> fallback to end
}

func TestResetOffsets_ExplicitClamping(t *testing.T) {
	restore := installMockAdmin(emptyGroupAdmin("g1"))
	defer restore()
	r := &fakeResetter{
		oldest: map[string]map[int32]int64{"t1": {0: 10, 1: 10, 2: 10}},
		newest: map[string]map[int32]int64{"t1": {0: 100, 1: 100, 2: 100}},
	}
	restoreR := installFakeResetter(r)
	defer restoreR()

	err := brokerDS().ResetConsumerGroupOffsets(context.Background(), api.OffsetResetRequest{
		GroupID:    "g1",
		Topic:      "t1",
		Mode:       api.OffsetResetExplicit,
		Partitions: []int32{0, 1, 2},
		PartitionOffsets: map[int32]int64{
			0: 5,   // below oldest -> clamp up to 10
			1: 500, // above newest -> clamp down to 100
			// 2 missing -> treated as 0 -> clamp up to 10
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(10), r.committed[0])
	assert.Equal(t, int64(100), r.committed[1])
	assert.Equal(t, int64(10), r.committed[2])
}

// --- DS-1: the real resetter lands on the target in either direction ---

// newMockBrokerResetter builds a saramaOffsetResetter for group "g" against a
// mock broker whose committed offset for t1/0 is `committed` (-1 = none).
func newMockBrokerResetter(t *testing.T, committed int64, commitErr sarama.KError) (*saramaOffsetResetter, *sarama.MockBroker) {
	t.Helper()
	broker := sarama.NewMockBroker(t, 1)
	t.Cleanup(broker.Close)
	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(broker.Addr(), broker.BrokerID()).
			SetLeader("t1", 0, broker.BrokerID()),
		"FindCoordinatorRequest": sarama.NewMockFindCoordinatorResponse(t).
			SetCoordinator(sarama.CoordinatorGroup, "g", broker),
		"OffsetFetchRequest": sarama.NewMockOffsetFetchResponse(t).
			SetOffset("g", "t1", 0, committed, "", sarama.ErrNoError),
		"OffsetCommitRequest": sarama.NewMockOffsetCommitResponse(t).
			SetError("g", "t1", 0, commitErr),
	})

	cfg := sarama.NewConfig()
	configureOffsetReset(cfg)
	cfg.Metadata.Retry.Max = 0
	cfg.Metadata.Retry.Backoff = time.Millisecond
	client, err := sarama.NewClient([]string{broker.Addr()}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	r, err := newSaramaOffsetResetter("g", client)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r, broker
}

// lastCommitted returns the offset for t1/0 in the last commit the broker saw.
func lastCommitted(t *testing.T, broker *sarama.MockBroker) (int64, bool) {
	t.Helper()
	var (
		off   int64
		found bool
	)
	for _, rr := range broker.History() {
		if req, ok := rr.Request.(*sarama.OffsetCommitRequest); ok {
			if o, _, err := req.Offset("t1", 0); err == nil {
				off, found = o, true
			}
		}
	}
	return off, found
}

func TestSaramaOffsetResetter_Commit(t *testing.T) {
	tests := []struct {
		name      string
		committed int64
		target    int64
	}{
		{"no committed offset", -1, 42},
		{"forward reset", 10, 50},
		{"backward reset", 100, 5},
		{"same offset", 7, 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, broker := newMockBrokerResetter(t, tt.committed, sarama.ErrNoError)

			err := r.Commit("g", "t1", map[int32]int64{0: tt.target})

			assert.NoError(t, err)
			got, ok := lastCommitted(t, broker)
			assert.True(t, ok, "an OffsetCommitRequest must be sent")
			assert.Equal(t, tt.target, got)
		})
	}

	t.Run("commit failure is returned", func(t *testing.T) {
		r, _ := newMockBrokerResetter(t, 10, sarama.ErrOffsetMetadataTooLarge)

		err := r.Commit("g", "t1", map[int32]int64{0: 50})

		assert.Error(t, err)
		assert.ErrorIs(t, err, sarama.ErrOffsetMetadataTooLarge)
	})
}
