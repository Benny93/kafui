package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptrI64(v int64) *int64 { return &v }
func ptrT(v time.Time) *time.Time { return &v }

func TestConsumeFlagsValidate(t *testing.T) {
	ts := time.Now()
	tests := []struct {
		name    string
		flags   ConsumeFlags
		wantErr bool
	}{
		{"default unspecified", ConsumeFlags{}, false},
		{"newest", ConsumeFlags{Seek: SeekNewest}, false},
		{"oldest", ConsumeFlags{Seek: SeekOldest}, false},
		{"live", ConsumeFlags{Seek: SeekLive}, false},
		{"from-offset ok", ConsumeFlags{Seek: SeekFromOffset, SeekOffset: ptrI64(5)}, false},
		{"from-offset missing", ConsumeFlags{Seek: SeekFromOffset}, true},
		{"to-offset missing", ConsumeFlags{Seek: SeekToOffset}, true},
		{"from-timestamp ok", ConsumeFlags{Seek: SeekFromTimestamp, SeekTimestamp: ptrT(ts)}, false},
		{"from-timestamp missing", ConsumeFlags{Seek: SeekFromTimestamp}, true},
		{"to-timestamp missing", ConsumeFlags{Seek: SeekToTimestamp}, true},
		{"unknown mode", ConsumeFlags{Seek: SeekMode("bogus")}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.flags.Validate()
			if tt.wantErr {
				require.Error(t, err)
				var ise InvalidSeekError
				assert.True(t, errors.As(err, &ise), "want InvalidSeekError")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultConsumeFlagsSeek(t *testing.T) {
	assert.Equal(t, SeekNewest, DefaultConsumeFlags().Seek)
}

func TestClampPageSize(t *testing.T) {
	tests := []struct{ in, want int }{
		{0, DefaultPageSize},
		{-1, DefaultPageSize},
		{501, MaxPageSize},
		{250, 250},
		{MaxPageSize, MaxPageSize},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, ClampPageSize(tt.in))
	}
}

func TestSortMessages(t *testing.T) {
	t0 := time.Unix(100, 0)
	t1 := time.Unix(200, 0)
	t2 := time.Unix(300, 0)
	// p0: offsets 1(t0),2(t2); p1: offsets 5(t1)
	mk := func(p int32, off int64, ts time.Time) Message {
		return Message{Partition: p, Offset: off, Timestamp: ts}
	}

	t.Run("forward ascending timestamp", func(t *testing.T) {
		msgs := []Message{mk(0, 2, t2), mk(1, 5, t1), mk(0, 1, t0)}
		SortMessages(msgs, false)
		assert.Equal(t, []int64{1, 5, 2}, offsetsOf(msgs))
	})

	t.Run("backward descending timestamp", func(t *testing.T) {
		msgs := []Message{mk(0, 1, t0), mk(1, 5, t1), mk(0, 2, t2)}
		SortMessages(msgs, true)
		assert.Equal(t, []int64{2, 5, 1}, offsetsOf(msgs))
	})

	t.Run("stable per-partition offset order on equal timestamp", func(t *testing.T) {
		// Same partition, same timestamp: offset order must be preserved.
		msgs := []Message{mk(0, 3, t0), mk(0, 1, t0), mk(0, 2, t0)}
		SortMessages(msgs, false)
		assert.Equal(t, []int64{1, 2, 3}, offsetsOf(msgs))
	})
}

func offsetsOf(msgs []Message) []int64 {
	out := make([]int64, len(msgs))
	for i, m := range msgs {
		out[i] = m.Offset
	}
	return out
}

func TestBrowseStatsAddMessage(t *testing.T) {
	var s BrowseStats
	k, v := 3, 4
	s.AddMessage(Message{KeySize: &k, ValueSize: &v, HeadersSize: 2})
	s.AddMessage(Message{ValueSize: &v}) // null key
	assert.Equal(t, int64(2), s.MessagesConsumed)
	assert.Equal(t, int64(3+4+2+4), s.BytesConsumed)
}

func TestBrowseSession(t *testing.T) {
	flags := ConsumeFlags{Seek: SeekNewest, Partitions: []int32{0, 1}}
	s := NewBrowseSession("orders", flags, "err", 0)
	assert.Equal(t, DefaultPageSize, s.PageSize)
	assert.False(t, s.HasMore())

	s.RecordPage(map[int32]int64{0: 10, 1: 20}, true)
	assert.True(t, s.HasMore())
	assert.Equal(t, int64(10), s.NextPositions[0])

	s.RecordPage(map[int32]int64{0: 15}, false)
	assert.False(t, s.HasMore())
	assert.Equal(t, int64(15), s.NextPositions[0])
	assert.Equal(t, int64(20), s.NextPositions[1])

	assert.False(t, s.IsStale("orders", SeekNewest))
	assert.True(t, s.IsStale("orders", SeekOldest))
	assert.True(t, s.IsStale("payments", SeekNewest))
}

func TestByteRateDelay(t *testing.T) {
	assert.Equal(t, time.Duration(0), ByteRateDelay(100, 0))
	assert.Equal(t, time.Duration(0), ByteRateDelay(0, 100))
	// 1000 bytes at 1000 B/s => 1s
	assert.Equal(t, time.Second, ByteRateDelay(1000, 1000))
	// 500 bytes at 1000 B/s => 500ms
	assert.Equal(t, 500*time.Millisecond, ByteRateDelay(500, 1000))
}

func TestRateLimiter(t *testing.T) {
	// 100/s => ~10ms between slots. First is immediate; 5 more take ~50ms.
	r := NewRateLimiter(100)
	ctx := context.Background()
	start := time.Now()
	for i := 0; i < 6; i++ {
		require.NoError(t, r.Wait(ctx))
	}
	elapsed := time.Since(start)
	assert.GreaterOrEqual(t, elapsed, 40*time.Millisecond)

	// Unlimited limiter never blocks.
	assert.NoError(t, NewRateLimiter(0).Wait(ctx))
}
