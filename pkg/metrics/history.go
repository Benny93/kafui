package metrics

import (
	"sync"
	"time"

	"github.com/Benny93/kafui/pkg/api"
)

// defaultHistoryCap is the number of samples retained per series. At the
// default 5s poll interval this is ~20 minutes of history — enough for the
// sparklines on the metrics page. History is process-lifetime only.
const defaultHistoryCap = 240

// ring is a fixed-capacity, O(1)-append ring buffer of time-series points.
// Old points are overwritten on wrap-around (that is the retention/pruning
// mechanism). It is safe for concurrent append/read.
type ring struct {
	mu   sync.Mutex
	buf  []api.MetricPoint
	next int
	size int
}

func newRing(capacity int) *ring {
	if capacity <= 0 {
		capacity = defaultHistoryCap
	}
	return &ring{buf: make([]api.MetricPoint, capacity)}
}

// append records a sample, overwriting the oldest once at capacity.
func (r *ring) append(t time.Time, v float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf[r.next] = api.MetricPoint{Time: t, Value: v}
	r.next = (r.next + 1) % len(r.buf)
	if r.size < len(r.buf) {
		r.size++
	}
}

// series returns an oldest-first copy of the retained points.
func (r *ring) series() api.TimeSeries {
	r.mu.Lock()
	defer r.mu.Unlock()
	pts := make([]api.MetricPoint, 0, r.size)
	start := 0
	if r.size == len(r.buf) {
		start = r.next // buffer full: oldest is at next
	}
	for i := 0; i < r.size; i++ {
		pts = append(pts, r.buf[(start+i)%len(r.buf)])
	}
	return api.TimeSeries{Points: pts}
}

// clusterHistory holds the retained series for a single cluster.
type clusterHistory struct {
	messagesIn *ring
	bytesIn    *ring
	bytesOut   *ring
}

func newClusterHistory(capacity int) *clusterHistory {
	return &clusterHistory{
		messagesIn: newRing(capacity),
		bytesIn:    newRing(capacity),
		bytesOut:   newRing(capacity),
	}
}
