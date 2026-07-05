// Package analysis implements the UI-independent topic scan + aggregation engine
// (TP-29/TP-30). The Aggregator is a pure, allocation-bounded accumulator that a
// Runner feeds sampled messages; the Registry owns the background lifecycle.
package analysis

import (
	"sort"
	"time"

	"github.com/Benny93/kafui/pkg/api"
)

// distinctCap bounds the distinct-value sets so a huge topic cannot exhaust
// memory. Once a set reaches the cap the estimate is frozen at the cap.
//
// ponytail: a HyperLogLog / t-digest would give unbounded, memoryless estimates
// and streaming percentiles. Since the scan is already offset-bounded (see the
// Runner), an exact set + sorted-slice percentiles is correct and simpler.
const distinctCap = 100_000

// fourteenDays bounds the retained hourly buckets.
const fourteenDays = 14 * 24 * time.Hour

// Aggregator accumulates message statistics for a topic scan. It is not safe for
// concurrent use; the Runner owns a single instance per scan.
type Aggregator struct {
	topic string

	count      int64
	minOffset  int64
	maxOffset  int64
	haveOffset bool

	minTs, maxTs time.Time

	nullKeys, nullValues int64

	distinctKeys   map[string]struct{}
	distinctValues map[string]struct{}

	keySizes   []int64
	valueSizes []int64

	hourly map[int64]int64

	// per-partition rollup
	partitions map[int32]*partitionAgg
}

type partitionAgg struct {
	count     int64
	minOffset int64
	maxOffset int64
}

// NewAggregator returns a ready Aggregator for the named topic.
func NewAggregator(topic string) *Aggregator {
	return &Aggregator{
		topic:          topic,
		distinctKeys:   make(map[string]struct{}),
		distinctValues: make(map[string]struct{}),
		hourly:         make(map[int64]int64),
		partitions:     make(map[int32]*partitionAgg),
	}
}

// Add folds one message into the aggregation.
func (a *Aggregator) Add(msg api.Message) {
	a.count++

	// Offsets (topic-wide).
	if !a.haveOffset {
		a.minOffset, a.maxOffset = msg.Offset, msg.Offset
		a.haveOffset = true
	} else {
		if msg.Offset < a.minOffset {
			a.minOffset = msg.Offset
		}
		if msg.Offset > a.maxOffset {
			a.maxOffset = msg.Offset
		}
	}

	// Per-partition offsets/counts.
	pa := a.partitions[msg.Partition]
	if pa == nil {
		pa = &partitionAgg{minOffset: msg.Offset, maxOffset: msg.Offset}
		a.partitions[msg.Partition] = pa
	}
	pa.count++
	if msg.Offset < pa.minOffset {
		pa.minOffset = msg.Offset
	}
	if msg.Offset > pa.maxOffset {
		pa.maxOffset = msg.Offset
	}

	// Timestamps + hourly buckets.
	if !msg.Timestamp.IsZero() {
		if a.minTs.IsZero() || msg.Timestamp.Before(a.minTs) {
			a.minTs = msg.Timestamp
		}
		if msg.Timestamp.After(a.maxTs) {
			a.maxTs = msg.Timestamp
		}
		bucket := msg.Timestamp.Truncate(time.Hour).Unix()
		a.hourly[bucket]++
	}

	// Key stats.
	kSize, kNull := fieldSize(msg.Key, msg.RawKey)
	if kNull {
		a.nullKeys++
	} else {
		a.addDistinct(a.distinctKeys, msg.Key, msg.RawKey)
	}
	a.keySizes = append(a.keySizes, kSize)

	// Value stats.
	vSize, vNull := fieldSize(msg.Value, msg.RawValue)
	if vNull {
		a.nullValues++
	} else {
		a.addDistinct(a.distinctValues, msg.Value, msg.RawValue)
	}
	a.valueSizes = append(a.valueSizes, vSize)
}

func (a *Aggregator) addDistinct(set map[string]struct{}, s string, raw []byte) {
	if len(set) >= distinctCap {
		return
	}
	key := s
	if key == "" && len(raw) > 0 {
		key = string(raw)
	}
	set[key] = struct{}{}
}

// Count returns the number of messages folded so far.
func (a *Aggregator) Count() int64 { return a.count }

// Result materialises the aggregation into an api.TopicAnalysisResult.
func (a *Aggregator) Result() api.TopicAnalysisResult {
	res := api.TopicAnalysisResult{
		Topic:                a.topic,
		MessageCount:         a.count,
		MinOffset:            a.minOffset,
		MaxOffset:            a.maxOffset,
		MinTimestamp:         a.minTs,
		MaxTimestamp:         a.maxTs,
		NullKeys:             a.nullKeys,
		NullValues:           a.nullValues,
		ApproxDistinctKeys:   boundEstimate(int64(len(a.distinctKeys)), a.count),
		ApproxDistinctValues: boundEstimate(int64(len(a.distinctValues)), a.count),
		KeySize:              sizeDistribution(a.keySizes),
		ValueSize:            sizeDistribution(a.valueSizes),
		HourlyCounts:         a.prunedHourly(),
		CompletedAt:          time.Now(),
	}

	pids := make([]int32, 0, len(a.partitions))
	for id := range a.partitions {
		pids = append(pids, id)
	}
	sort.Slice(pids, func(i, j int) bool { return pids[i] < pids[j] })
	for _, id := range pids {
		pa := a.partitions[id]
		res.Partitions = append(res.Partitions, api.PartitionAnalysis{
			Partition:    id,
			MessageCount: pa.count,
			MinOffset:    pa.minOffset,
			MaxOffset:    pa.maxOffset,
		})
	}
	return res
}

// prunedHourly returns the hourly buckets within 14 days of the max timestamp.
func (a *Aggregator) prunedHourly() map[int64]int64 {
	out := make(map[int64]int64, len(a.hourly))
	if a.maxTs.IsZero() {
		return out
	}
	cutoff := a.maxTs.Add(-fourteenDays).Unix()
	for bucket, n := range a.hourly {
		if bucket >= cutoff {
			out[bucket] = n
		}
	}
	return out
}

// fieldSize returns the byte size of a message field and whether it is null
// (empty string and no raw bytes).
func fieldSize(s string, raw []byte) (int64, bool) {
	if len(raw) > 0 {
		return int64(len(raw)), false
	}
	if s == "" {
		return 0, true
	}
	return int64(len(s)), false
}

// boundEstimate ensures a distinct estimate never exceeds the message count.
func boundEstimate(estimate, count int64) int64 {
	if estimate > count {
		return count
	}
	return estimate
}

// sizeDistribution computes sum/min/max/avg and percentiles from a set of sizes.
func sizeDistribution(sizes []int64) api.SizeDistribution {
	d := api.SizeDistribution{Count: int64(len(sizes))}
	if len(sizes) == 0 {
		return d
	}
	sorted := append([]int64{}, sizes...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	d.Min = sorted[0]
	d.Max = sorted[len(sorted)-1]
	for _, s := range sorted {
		d.Sum += s
	}
	d.Avg = float64(d.Sum) / float64(len(sorted))
	d.P50 = percentile(sorted, 0.50)
	d.P75 = percentile(sorted, 0.75)
	d.P95 = percentile(sorted, 0.95)
	d.P99 = percentile(sorted, 0.99)
	d.P999 = percentile(sorted, 0.999)
	return d
}

// percentile returns the nearest-rank percentile of a pre-sorted slice.
func percentile(sorted []int64, p float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	rank := int(p * float64(len(sorted)))
	if rank >= len(sorted) {
		rank = len(sorted) - 1
	}
	return sorted[rank]
}
