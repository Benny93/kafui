package analysis

import (
	"fmt"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregator_CountsAndOffsets(t *testing.T) {
	a := NewAggregator("t")
	a.Add(api.Message{Partition: 0, Offset: 10, Key: "k1", Value: "v1"})
	a.Add(api.Message{Partition: 0, Offset: 5, Key: "k2", Value: "v2"})
	a.Add(api.Message{Partition: 1, Offset: 100, Key: "k3", Value: "v3"})

	res := a.Result()
	assert.Equal(t, int64(3), res.MessageCount)
	assert.Equal(t, int64(5), res.MinOffset)
	assert.Equal(t, int64(100), res.MaxOffset)
	require.Len(t, res.Partitions, 2)
	assert.Equal(t, int32(0), res.Partitions[0].Partition)
	assert.Equal(t, int64(2), res.Partitions[0].MessageCount)
	assert.Equal(t, int64(5), res.Partitions[0].MinOffset)
	assert.Equal(t, int64(1), res.Partitions[1].MessageCount)
}

func TestAggregator_NullCounting(t *testing.T) {
	a := NewAggregator("t")
	a.Add(api.Message{Key: "k", Value: "v"})
	a.Add(api.Message{Key: "", Value: "v2"})             // null key
	a.Add(api.Message{Key: "k3", Value: ""})             // null value
	a.Add(api.Message{Key: "", Value: ""})               // both null
	a.Add(api.Message{RawKey: []byte{1}, RawValue: nil}) // raw key present -> not null key, null value

	res := a.Result()
	assert.Equal(t, int64(2), res.NullKeys)
	assert.Equal(t, int64(3), res.NullValues) // msgs 3, 4, and 5 (raw value nil)
}

func TestAggregator_DistinctBound(t *testing.T) {
	a := NewAggregator("t")
	// 10 messages but only 3 distinct values.
	for i := 0; i < 10; i++ {
		a.Add(api.Message{Key: fmt.Sprintf("k%d", i%3), Value: fmt.Sprintf("v%d", i%3)})
	}
	res := a.Result()
	assert.Equal(t, int64(3), res.ApproxDistinctKeys)
	assert.Equal(t, int64(3), res.ApproxDistinctValues)
	// Estimate must never exceed the message count.
	assert.LessOrEqual(t, res.ApproxDistinctKeys, res.MessageCount)
}

func TestAggregator_Percentiles(t *testing.T) {
	a := NewAggregator("t")
	// Values of sizes 1..100 bytes.
	for i := 1; i <= 100; i++ {
		a.Add(api.Message{Key: "k", Value: string(make([]byte, i))})
	}
	res := a.Result()
	assert.Equal(t, int64(1), res.ValueSize.Min)
	assert.Equal(t, int64(100), res.ValueSize.Max)
	assert.Equal(t, int64(100), res.ValueSize.Count)
	// Nearest-rank: p50 ~ 51st smallest = 51.
	assert.Equal(t, int64(51), res.ValueSize.P50)
	assert.Equal(t, int64(96), res.ValueSize.P95)
	assert.Equal(t, int64(100), res.ValueSize.P999)
	assert.InDelta(t, 50.5, res.ValueSize.Avg, 0.01)
}

func TestAggregator_HourlyBuckets(t *testing.T) {
	a := NewAggregator("t")
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	// Two messages in the same hour, one in the next hour.
	a.Add(api.Message{Value: "v", Timestamp: now})
	a.Add(api.Message{Value: "v", Timestamp: now.Add(30 * time.Minute)})
	a.Add(api.Message{Value: "v", Timestamp: now.Add(90 * time.Minute)})
	// One message 20 days in the past -> pruned from the last-14-days window.
	a.Add(api.Message{Value: "v", Timestamp: now.Add(-20 * 24 * time.Hour)})

	res := a.Result()
	hour0 := now.Truncate(time.Hour).Unix()
	hour1 := now.Add(90 * time.Minute).Truncate(time.Hour).Unix()
	assert.Equal(t, int64(2), res.HourlyCounts[hour0])
	assert.Equal(t, int64(1), res.HourlyCounts[hour1])
	// The 20-day-old bucket must be pruned.
	oldHour := now.Add(-20 * 24 * time.Hour).Truncate(time.Hour).Unix()
	_, present := res.HourlyCounts[oldHour]
	assert.False(t, present, "buckets older than 14 days must be pruned")
	assert.False(t, res.MinTimestamp.IsZero())
}

func TestProgress_Percentage(t *testing.T) {
	assert.Equal(t, 0.0, api.AnalysisProgress{TotalOffsets: 0}.Percentage())
	assert.Equal(t, 50.0, api.AnalysisProgress{ProcessedOffsets: 50, TotalOffsets: 100}.Percentage())
	// Over-shoot is capped at 100.
	assert.Equal(t, 100.0, api.AnalysisProgress{ProcessedOffsets: 150, TotalOffsets: 100}.Percentage())
}
