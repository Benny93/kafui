package api

import (
	"context"
	"sort"
	"sync"
	"time"
)

// ---- MSG-6: stable timestamp ordering --------------------------------------

// SortMessages orders a fetched page for display. When descending is true
// (backward modes: newest, to-offset, to-timestamp) messages are ordered
// newest-first; otherwise oldest-first. The sort is stable and, within a single
// partition, always preserves offset order regardless of equal timestamps.
func SortMessages(msgs []Message, descending bool) {
	sort.SliceStable(msgs, func(i, j int) bool {
		a, b := msgs[i], msgs[j]
		if a.Partition == b.Partition {
			// Same partition: offset order is authoritative.
			if descending {
				return a.Offset > b.Offset
			}
			return a.Offset < b.Offset
		}
		if !a.Timestamp.Equal(b.Timestamp) {
			if descending {
				return a.Timestamp.After(b.Timestamp)
			}
			return a.Timestamp.Before(b.Timestamp)
		}
		// Equal timestamps across partitions: keep deterministic by partition.
		return a.Partition < b.Partition
	})
}

// ---- MSG-8: page size clamping ---------------------------------------------

const (
	DefaultPageSize = 100 // used when a requested page size is non-positive
	MaxPageSize     = 500 // hard upper bound
)

// ClampPageSize returns a valid page size: non-positive requests fall back to
// DefaultPageSize and requests above MaxPageSize are capped at MaxPageSize.
func ClampPageSize(n int) int {
	if n <= 0 {
		return DefaultPageSize
	}
	if n > MaxPageSize {
		return MaxPageSize
	}
	return n
}

// ---- MSG-7: browse phases and consuming statistics -------------------------

type BrowsePhase string

const (
	PhaseCreatingConsumer BrowsePhase = "creating-consumer"
	PhasePolling          BrowsePhase = "polling"
	PhaseDone             BrowsePhase = "done"
)

// BrowseStats accumulates counters over the lifetime of a browse fetch.
type BrowseStats struct {
	MessagesConsumed int64
	BytesConsumed    int64
	FilterErrors     int64
	ElapsedMs        int64
}

// AddMessage folds one consumed message into the running totals.
func (s *BrowseStats) AddMessage(m Message) {
	s.MessagesConsumed++
	if m.KeySize != nil {
		s.BytesConsumed += int64(*m.KeySize)
	}
	if m.ValueSize != nil {
		s.BytesConsumed += int64(*m.ValueSize)
	}
	s.BytesConsumed += int64(m.HeadersSize)
}

// BrowseEvent is emitted during a browse to report progress. The datasource
// pushes these through the consume pipeline; the UI renders them.
type BrowseEvent struct {
	Phase       BrowsePhase
	Description string
	Stats       BrowseStats
	Done        bool
}

// ---- MSG-9: browse session (cursor equivalent) -----------------------------

// BrowseSession preserves the query context of a browse so follow-up pages
// reuse the exact original query. It is the in-process equivalent of a cursor.
type BrowseSession struct {
	Topic      string
	Seek       SeekMode
	Partitions []int32
	Filter     string
	KeySerde   string
	ValueSerde string
	PageSize   int

	// NextPositions holds the next offset to poll per partition. It is updated
	// by RecordPage as pages are consumed.
	NextPositions map[int32]int64

	more bool
}

// NewBrowseSession builds a session from the flags that started a browse.
func NewBrowseSession(topic string, flags ConsumeFlags, filter string, pageSize int) *BrowseSession {
	return &BrowseSession{
		Topic:         topic,
		Seek:          flags.Seek,
		Partitions:    append([]int32(nil), flags.Partitions...),
		Filter:        filter,
		PageSize:      ClampPageSize(pageSize),
		NextPositions: map[int32]int64{},
	}
}

// IsStale reports whether the session no longer matches the active query and
// therefore must not be continued.
func (s *BrowseSession) IsStale(topic string, seek SeekMode) bool {
	return s.Topic != topic || s.Seek != seek
}

// RecordPage stores the next per-partition positions to poll and whether more
// data may remain beyond this page.
func (s *BrowseSession) RecordPage(nextPositions map[int32]int64, more bool) {
	if s.NextPositions == nil {
		s.NextPositions = map[int32]int64{}
	}
	for p, off := range nextPositions {
		s.NextPositions[p] = off
	}
	s.more = more
}

// HasMore reports whether a next page may be available.
func (s *BrowseSession) HasMore() bool { return s.more }

// ---- MSG-10: tailing throttle and polling resource controls ---------------

// DefaultTailRate is the live-tail delivery cap in messages per second.
const DefaultTailRate = 20

// RateLimiter enforces a fixed maximum delivery rate. Wait blocks until the
// next slot is available or the context is done. A rate of zero disables it.
type RateLimiter struct {
	interval time.Duration
	mu       sync.Mutex
	next     time.Time
}

// NewRateLimiter builds a limiter capped at perSecond deliveries. perSecond <= 0
// yields an unlimited (no-op) limiter.
func NewRateLimiter(perSecond int) *RateLimiter {
	if perSecond <= 0 {
		return &RateLimiter{}
	}
	return &RateLimiter{interval: time.Second / time.Duration(perSecond)}
}

// Wait blocks until the caller may deliver the next message.
func (r *RateLimiter) Wait(ctx context.Context) error {
	if r == nil || r.interval == 0 {
		return nil
	}
	r.mu.Lock()
	now := time.Now()
	if r.next.Before(now) {
		r.next = now
	}
	wait := r.next.Sub(now)
	r.next = r.next.Add(r.interval)
	r.mu.Unlock()

	if wait <= 0 {
		return nil
	}
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// ByteRateDelay returns how long to pause after consuming byteCount bytes to
// respect maxBytesPerSec. A non-positive limit or byte count yields no delay.
func ByteRateDelay(byteCount, maxBytesPerSec int) time.Duration {
	if maxBytesPerSec <= 0 || byteCount <= 0 {
		return 0
	}
	return time.Duration(float64(byteCount) / float64(maxBytesPerSec) * float64(time.Second))
}
