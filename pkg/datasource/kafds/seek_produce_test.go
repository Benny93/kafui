package kafds

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seekClient wraps MockClient (defined in consume_test.go) to also return a
// fixed partition set.
type seekClient struct {
	*MockClient
	partitions []int32
}

func (c *seekClient) Partitions(topic string) ([]int32, error) { return c.partitions, nil }

func ptrInt64(v int64) *int64 { return &v }

// ---- MSG-2/3: seek offset resolution + clamping + timestamp fallback -------

func TestResolvePartitionSeek(t *testing.T) {
	offs := &offsets{oldest: 100, newest: 200}

	t.Run("from-offset in range", func(t *testing.T) {
		cfg := &ConsumeConfig{Seek: api.SeekFromOffset, SeekOffset: ptrInt64(150)}
		start, stop, backward, err := resolvePartitionSeek(&MockClient{}, "t", 0, offs, cfg, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(150), start)
		assert.Equal(t, int64(200), stop)
		assert.False(t, backward)
	})

	t.Run("from-offset clamped below oldest", func(t *testing.T) {
		cfg := &ConsumeConfig{Seek: api.SeekFromOffset, SeekOffset: ptrInt64(1)}
		start, _, _, err := resolvePartitionSeek(&MockClient{}, "t", 0, offs, cfg, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(100), start)
	})

	t.Run("from-offset clamped above newest", func(t *testing.T) {
		cfg := &ConsumeConfig{Seek: api.SeekFromOffset, SeekOffset: ptrInt64(9999)}
		start, _, _, err := resolvePartitionSeek(&MockClient{}, "t", 0, offs, cfg, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(200), start)
	})

	t.Run("to-offset backward window", func(t *testing.T) {
		cfg := &ConsumeConfig{Seek: api.SeekToOffset, SeekOffset: ptrInt64(150), LimitMessagesFlag: 30}
		start, stop, backward, err := resolvePartitionSeek(&MockClient{}, "t", 0, offs, cfg, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(120), start) // 150 - 30
		assert.Equal(t, int64(151), stop)  // inclusive of 150
		assert.True(t, backward)
	})

	t.Run("from-timestamp resolves offset", func(t *testing.T) {
		client := &MockClient{getOffsetFunc: func(topic string, p int32, ts int64) (int64, error) {
			return 175, nil
		}}
		now := time.Now()
		cfg := &ConsumeConfig{Seek: api.SeekFromTimestamp, SeekTimestamp: &now}
		start, _, backward, err := resolvePartitionSeek(client, "t", 0, offs, cfg, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(175), start)
		assert.False(t, backward)
	})

	t.Run("from-timestamp no match falls back to newest", func(t *testing.T) {
		client := &MockClient{getOffsetFunc: func(topic string, p int32, ts int64) (int64, error) {
			return -1, nil
		}}
		now := time.Now()
		cfg := &ConsumeConfig{Seek: api.SeekFromTimestamp, SeekTimestamp: &now}
		start, _, _, err := resolvePartitionSeek(client, "t", 0, offs, cfg, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(200), start)
	})

	t.Run("to-timestamp backward no-match reads from end", func(t *testing.T) {
		client := &MockClient{getOffsetFunc: func(topic string, p int32, ts int64) (int64, error) {
			return -1, nil
		}}
		now := time.Now()
		cfg := &ConsumeConfig{Seek: api.SeekToTimestamp, SeekTimestamp: &now, LimitMessagesFlag: 50}
		start, stop, backward, err := resolvePartitionSeek(client, "t", 0, offs, cfg, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(150), start) // 200 - 50
		assert.Equal(t, int64(200), stop)
		assert.True(t, backward)
	})

	t.Run("legacy tail", func(t *testing.T) {
		cfg := &ConsumeConfig{Tail: 20}
		start, _, backward, err := resolvePartitionSeek(&MockClient{}, "t", 0, offs, cfg, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(180), start) // 200 - 20
		assert.False(t, backward)
	})
}

// ---- MSG-30: produce ------------------------------------------------------

type fakeSyncProducer struct {
	sent    *sarama.ProducerMessage
	sendErr error
}

func (f *fakeSyncProducer) SendMessage(msg *sarama.ProducerMessage) (int32, int64, error) {
	f.sent = msg
	if f.sendErr != nil {
		return 0, 0, f.sendErr
	}
	return msg.Partition, 1, nil
}
func (f *fakeSyncProducer) SendMessages(msgs []*sarama.ProducerMessage) error { return nil }
func (f *fakeSyncProducer) Close() error                                      { return nil }
func (f *fakeSyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag           { return 0 }
func (f *fakeSyncProducer) IsTransactional() bool                             { return false }
func (f *fakeSyncProducer) BeginTxn() error                                   { return nil }
func (f *fakeSyncProducer) CommitTxn() error                                  { return nil }
func (f *fakeSyncProducer) AbortTxn() error                                   { return nil }
func (f *fakeSyncProducer) AddOffsetsToTxn(map[string][]*sarama.PartitionOffsetMetadata, string) error {
	return nil
}
func (f *fakeSyncProducer) AddMessageToTxn(*sarama.ConsumerMessage, string, *string) error {
	return nil
}

func withFakeProducer(f sarama.SyncProducer, fn func()) {
	orig := newSyncProducer
	newSyncProducer = func(sarama.Client) (sarama.SyncProducer, error) { return f, nil }
	defer func() { newSyncProducer = orig }()
	fn()
}

func TestDoProduce(t *testing.T) {
	client := &seekClient{MockClient: &MockClient{}, partitions: []int32{0, 1, 2}}

	t.Run("success serializes key/value/headers/partition", func(t *testing.T) {
		fake := &fakeSyncProducer{}
		withFakeProducer(fake, func() {
			p := int32(2)
			rec := api.ProduceRecord{
				Key:       []byte("k"),
				Value:     []byte("v"),
				Headers:   []api.MessageHeader{{Key: "h", Value: "1"}},
				Partition: &p,
			}
			err := doProduce(context.Background(), client, "orders", rec)
			require.NoError(t, err)
		})
		require.NotNil(t, fake.sent)
		assert.Equal(t, int32(2), fake.sent.Partition)
		kb, _ := fake.sent.Key.Encode()
		vb, _ := fake.sent.Value.Encode()
		assert.Equal(t, "k", string(kb))
		assert.Equal(t, "v", string(vb))
		require.Len(t, fake.sent.Headers, 1)
		assert.Equal(t, "h", string(fake.sent.Headers[0].Key))
	})

	t.Run("nil key/value produce null record", func(t *testing.T) {
		fake := &fakeSyncProducer{}
		withFakeProducer(fake, func() {
			err := doProduce(context.Background(), client, "orders", api.ProduceRecord{})
			require.NoError(t, err)
		})
		assert.Nil(t, fake.sent.Key)
		assert.Nil(t, fake.sent.Value)
	})

	t.Run("partition out of range rejected", func(t *testing.T) {
		fake := &fakeSyncProducer{}
		withFakeProducer(fake, func() {
			p := int32(9)
			err := doProduce(context.Background(), client, "orders", api.ProduceRecord{Partition: &p})
			require.Error(t, err)
			var pe api.PartitionError
			assert.True(t, errors.As(err, &pe))
		})
	})

	t.Run("missing topic rejected", func(t *testing.T) {
		empty := &seekClient{MockClient: &MockClient{}, partitions: nil}
		fake := &fakeSyncProducer{}
		withFakeProducer(fake, func() {
			err := doProduce(context.Background(), empty, "ghost", api.ProduceRecord{Value: []byte("x")})
			require.Error(t, err)
			var te api.TopicNotFoundError
			assert.True(t, errors.As(err, &te))
		})
	})

	t.Run("send failure wrapped in ProduceError", func(t *testing.T) {
		fake := &fakeSyncProducer{sendErr: errors.New("broker down")}
		withFakeProducer(fake, func() {
			err := doProduce(context.Background(), client, "orders", api.ProduceRecord{Value: []byte("x")})
			require.Error(t, err)
			var pe api.ProduceError
			assert.True(t, errors.As(err, &pe))
			assert.ErrorContains(t, err, "broker down")
		})
	})
}
