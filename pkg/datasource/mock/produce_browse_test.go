package mock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func firstTopic(t *testing.T, m *KafkaDataSourceMock) string {
	topics, err := m.GetTopics()
	require.NoError(t, err)
	for name := range topics {
		return name
	}
	t.Fatal("no topics")
	return ""
}

func ptrI32(v int32) *int32 { return &v }

// MSG-30: produced messages are browsable via ConsumeTopic.
func TestMockProduceMakesMessagesBrowsable(t *testing.T) {
	m := &KafkaDataSourceMock{}
	m.Init("")
	topic := firstTopic(t, m)

	p := int32(0)
	err := m.ProduceMessage(context.Background(), topic, api.ProduceRecord{
		Key:       []byte("order-42"),
		Value:     []byte(`{"id":42}`),
		Partition: &p,
	})
	require.NoError(t, err)

	// Browse just partition 0 with a limit; the produced message must appear.
	var got []api.Message
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	m.ConsumeTopic(ctx, topic, api.ConsumeFlags{
		Follow:        false,
		Seek:          api.SeekOldest,
		Partitions:    []int32{0},
		LimitMessages: 1,
	}, func(msg api.Message) { got = append(got, msg) }, func(any) {})

	require.NotEmpty(t, got)
	assert.Equal(t, "order-42", got[0].Key)
	assert.Equal(t, `{"id":42}`, got[0].Value)
	assert.Equal(t, int32(0), got[0].Partition)
	require.NotNil(t, got[0].ValueSize)
	assert.Equal(t, len(`{"id":42}`), *got[0].ValueSize)
}

func TestMockProduceValidation(t *testing.T) {
	m := &KafkaDataSourceMock{}
	m.Init("")
	topic := firstTopic(t, m)

	t.Run("unknown topic", func(t *testing.T) {
		err := m.ProduceMessage(context.Background(), "does-not-exist", api.ProduceRecord{Value: []byte("x")})
		var te api.TopicNotFoundError
		assert.True(t, errors.As(err, &te))
	})

	t.Run("partition out of range", func(t *testing.T) {
		err := m.ProduceMessage(context.Background(), topic, api.ProduceRecord{Value: []byte("x"), Partition: ptrI32(9999)})
		var pe api.PartitionError
		assert.True(t, errors.As(err, &pe))
	})

	t.Run("nil value is null record", func(t *testing.T) {
		err := m.ProduceMessage(context.Background(), topic, api.ProduceRecord{Partition: ptrI32(0)})
		require.NoError(t, err)
	})
}

// MSG-2/3/4: browseMessages seek/partition/limit filtering.
func TestBrowseMessages(t *testing.T) {
	t0 := time.Unix(100, 0)
	t1 := time.Unix(200, 0)
	t2 := time.Unix(300, 0)
	msgs := []api.Message{
		{Partition: 0, Offset: 10, Timestamp: t0},
		{Partition: 0, Offset: 11, Timestamp: t1},
		{Partition: 1, Offset: 20, Timestamp: t2},
	}

	t.Run("partition filter", func(t *testing.T) {
		out := browseMessages(msgs, api.ConsumeFlags{Partitions: []int32{1}})
		require.Len(t, out, 1)
		assert.Equal(t, int32(1), out[0].Partition)
	})

	t.Run("no partitions means all", func(t *testing.T) {
		out := browseMessages(msgs, api.ConsumeFlags{Seek: api.SeekOldest})
		assert.Len(t, out, 3)
	})

	t.Run("from-offset clamped above end returns last", func(t *testing.T) {
		out := browseMessages(msgs, api.ConsumeFlags{Seek: api.SeekFromOffset, SeekOffset: func() *int64 { v := int64(9999); return &v }()})
		require.Len(t, out, 1)
		assert.Equal(t, int64(20), out[0].Offset) // max offset
	})

	t.Run("from-offset in range", func(t *testing.T) {
		out := browseMessages(msgs, api.ConsumeFlags{Seek: api.SeekFromOffset, SeekOffset: func() *int64 { v := int64(11); return &v }()})
		assert.Len(t, out, 2) // offsets 11 and 20
	})

	t.Run("to-timestamp keeps earlier", func(t *testing.T) {
		out := browseMessages(msgs, api.ConsumeFlags{Seek: api.SeekToTimestamp, SeekTimestamp: &t2})
		assert.Len(t, out, 2) // t0 and t1 are before t2
	})

	t.Run("limit", func(t *testing.T) {
		out := browseMessages(msgs, api.ConsumeFlags{Seek: api.SeekOldest, LimitMessages: 1})
		assert.Len(t, out, 1)
	})
}
