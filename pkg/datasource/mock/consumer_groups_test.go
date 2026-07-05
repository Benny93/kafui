package mock

import (
	"context"
	"errors"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
)

func newGroupMock() *KafkaDataSourceMock {
	m := &KafkaDataSourceMock{}
	m.Init("")
	return m
}

func TestMockGetConsumerGroupDetail(t *testing.T) {
	m := newGroupMock()

	t.Run("lag math and member attribution", func(t *testing.T) {
		d, err := m.GetConsumerGroupDetail("order-processor")
		assert.NoError(t, err)
		assert.Equal(t, api.GroupStateStable, d.State)
		assert.Len(t, d.Members, 3)
		byPart := map[int32]api.PartitionOffset{}
		for _, po := range d.TopicOffsets {
			byPart[po.Partition] = po
		}
		assert.Equal(t, int64(50), *byPart[0].Lag) // 150-100
		assert.Equal(t, int64(0), *byPart[2].Lag)  // 150-150
		assert.NotEmpty(t, byPart[0].MemberID)
	})

	t.Run("assigned but uncommitted partition", func(t *testing.T) {
		d, err := m.GetConsumerGroupDetail("analytics-consumer")
		assert.NoError(t, err)
		byPart := map[int32]api.PartitionOffset{}
		for _, po := range d.TopicOffsets {
			byPart[po.Partition] = po
		}
		assert.Nil(t, byPart[1].CommittedOffset) // uncommitted
		assert.Nil(t, byPart[1].Lag)
		assert.NotEmpty(t, byPart[1].MemberID) // still attributed
	})

	t.Run("not found", func(t *testing.T) {
		_, err := m.GetConsumerGroupDetail("nope")
		var nf api.GroupNotFoundError
		assert.True(t, errors.As(err, &nf))
	})
}

func TestMockGetConsumerGroupDetails(t *testing.T) {
	m := newGroupMock()
	rows, err := m.GetConsumerGroupDetails([]string{"order-processor", "load-test-processor", "unknown"})
	assert.NoError(t, err)
	byName := map[string]api.ConsumerGroup{}
	for _, r := range rows {
		byName[r.Name] = r
	}
	assert.Equal(t, int64(100), *byName["order-processor"].Lag) // 50+50+0
	assert.Nil(t, byName["load-test-processor"].Lag)            // no committed offsets
	assert.Equal(t, api.GroupStateUnknown, byName["unknown"].State)
}

func TestMockGetConsumerGroupsForTopic(t *testing.T) {
	m := newGroupMock()
	groups, err := m.GetConsumerGroupsForTopic("order-events")
	assert.NoError(t, err)
	assert.Len(t, groups, 1)
	assert.Equal(t, "order-processor", groups[0].Name)
	assert.Equal(t, 3, groups[0].MemberCount)
	assert.NotNil(t, groups[0].Lag)
}

func TestMockDeleteConsumerGroup(t *testing.T) {
	m := newGroupMock()

	// Active group rejected.
	err := m.DeleteConsumerGroup("order-processor")
	var ne api.GroupNotEmptyError
	assert.True(t, errors.As(err, &ne))

	// Empty group deleted; subsequent detail is not found.
	assert.NoError(t, m.DeleteConsumerGroup("inventory-sync"))
	_, err = m.GetConsumerGroupDetail("inventory-sync")
	var nf api.GroupNotFoundError
	assert.True(t, errors.As(err, &nf))
}

func TestMockDeleteConsumerGroupOffsets(t *testing.T) {
	m := newGroupMock()
	assert.NoError(t, m.DeleteConsumerGroupOffsets("inventory-sync", "inventory-events"))
	d, err := m.GetConsumerGroupDetail("inventory-sync")
	assert.NoError(t, err)
	assert.Empty(t, d.TopicOffsets) // topic offsets gone
}

func TestMockResetConsumerGroupOffsets(t *testing.T) {
	t.Run("rejects active group", func(t *testing.T) {
		m := newGroupMock()
		err := m.ResetConsumerGroupOffsets(context.Background(), api.OffsetResetRequest{
			GroupID: "order-processor", Topic: "order-events", Mode: api.OffsetResetEarliest,
		})
		var ne api.GroupNotEmptyError
		assert.True(t, errors.As(err, &ne))
	})

	t.Run("earliest on empty group", func(t *testing.T) {
		m := newGroupMock()
		err := m.ResetConsumerGroupOffsets(context.Background(), api.OffsetResetRequest{
			GroupID: "inventory-sync", Topic: "inventory-events", Mode: api.OffsetResetEarliest,
		})
		assert.NoError(t, err)
		d, _ := m.GetConsumerGroupDetail("inventory-sync")
		for _, po := range d.TopicOffsets {
			assert.Equal(t, int64(0), *po.CommittedOffset)
		}
	})

	t.Run("explicit clamps to bounds", func(t *testing.T) {
		m := newGroupMock()
		err := m.ResetConsumerGroupOffsets(context.Background(), api.OffsetResetRequest{
			GroupID:          "inventory-sync",
			Topic:            "inventory-events",
			Mode:             api.OffsetResetExplicit,
			Partitions:       []int32{0},
			PartitionOffsets: map[int32]int64{0: 9999}, // above end (60) -> clamp to 60
		})
		assert.NoError(t, err)
		d, _ := m.GetConsumerGroupDetail("inventory-sync")
		for _, po := range d.TopicOffsets {
			if po.Partition == 0 {
				assert.Equal(t, int64(60), *po.CommittedOffset)
			}
		}
	})

	t.Run("invalid mode", func(t *testing.T) {
		m := newGroupMock()
		err := m.ResetConsumerGroupOffsets(context.Background(), api.OffsetResetRequest{
			GroupID: "inventory-sync", Topic: "inventory-events", Mode: "bogus",
		})
		var ivr api.InvalidOffsetResetError
		assert.True(t, errors.As(err, &ivr))
	})
}
