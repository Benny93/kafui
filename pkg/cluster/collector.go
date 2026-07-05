// Package cluster provides a UI-independent background collector that polls
// every configured cluster for health, statistics, and capabilities, caching
// the results so the dashboard and sidebar can read them without blocking.
package cluster

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/Benny93/kafui/pkg/api"
)

// perClusterTimeout bounds a single cluster's collection so one slow/unreachable
// cluster can't stall the whole cycle.
const perClusterTimeout = 10 * time.Second

// maxParallel bounds concurrent per-cluster collections.
// ponytail: fixed cap; make configurable if cluster counts grow large.
const maxParallel = 8

// Collector polls clusters and caches overviews + statistics. Safe for
// concurrent use: the background loop writes while the UI reads.
type Collector struct {
	ds       api.KafkaDataSource
	interval time.Duration
	readOnly func(cluster string) bool

	mu    sync.RWMutex
	cache map[string]cached
	order []string // stable cluster ordering
}

type cached struct {
	overview api.ClusterOverview
	stats    api.ClusterStatistics
	hasStats bool
}

// New builds a Collector. isReadOnly reports the per-cluster read-only flag
// (may be nil). interval <= 0 falls back to 30s.
func New(ds api.KafkaDataSource, interval time.Duration, isReadOnly func(string) bool) *Collector {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if isReadOnly == nil {
		isReadOnly = func(string) bool { return false }
	}
	c := &Collector{
		ds:       ds,
		interval: interval,
		readOnly: isReadOnly,
		cache:    map[string]cached{},
	}
	c.seed()
	return c
}

// seed marks every configured cluster as initializing before the first cycle.
func (c *Collector) seed() {
	names, err := c.ds.GetContexts()
	if err != nil {
		return
	}
	sort.Strings(names)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.order = names
	for _, n := range names {
		c.cache[n] = cached{overview: api.ClusterOverview{
			Name:             n,
			Status:           api.ClusterInitializing,
			ReadOnly:         c.readOnly(n),
			BytesInPerSec:    -1,
			BytesOutPerSec:   -1,
			MessagesInPerSec: -1,
		}}
	}
}

// Interval returns the configured collection interval.
func (c *Collector) Interval() time.Duration { return c.interval }

// CollectAll runs one collection cycle across every cluster in parallel and
// updates the cache. One cluster failing never affects the others.
func (c *Collector) CollectAll(ctx context.Context) {
	c.mu.RLock()
	names := append([]string(nil), c.order...)
	c.mu.RUnlock()

	sem := make(chan struct{}, maxParallel)
	var wg sync.WaitGroup
	for _, name := range names {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			c.collectOne(ctx, name)
		}(name)
	}
	wg.Wait()
}

// RefreshCluster forces an immediate collection of a single cluster and returns
// the refreshed overview. Unknown names yield a ClusterNotFoundError.
func (c *Collector) RefreshCluster(ctx context.Context, name string) (api.ClusterOverview, error) {
	c.mu.RLock()
	_, known := c.cache[name]
	c.mu.RUnlock()
	if !known {
		return api.ClusterOverview{}, api.ClusterNotFoundError{Name: name}
	}
	c.collectOne(ctx, name)
	return c.overview(name)
}

// collectOne collects a single cluster with a bounded timeout and updates cache.
func (c *Collector) collectOne(ctx context.Context, name string) {
	cctx, cancel := context.WithTimeout(ctx, perClusterTimeout)
	defer cancel()

	stats, statsErr := c.ds.GetClusterStatistics(cctx, name)
	caps, _ := c.ds.GetClusterCapabilities(cctx, name)

	c.mu.Lock()
	defer c.mu.Unlock()
	prev := c.cache[name]
	ov := api.ClusterOverview{
		Name:             name,
		ReadOnly:         c.readOnly(name),
		Capabilities:     caps,
		BytesInPerSec:    -1,
		BytesOutPerSec:   -1,
		MessagesInPerSec: -1,
	}
	if statsErr != nil {
		// Offline: retain previous stats, record error.
		ov.Status = api.ClusterOffline
		ov.LastError = statsErr.Error()
		if prev.hasStats {
			ov.BrokerCount = prev.stats.BrokerCount
			ov.OnlinePartitionCount = prev.stats.OnlinePartitions
			ov.Version = prev.stats.Version
			c.cache[name] = cached{overview: ov, stats: prev.stats, hasStats: true}
			return
		}
		c.cache[name] = cached{overview: ov}
		return
	}
	ov.Status = api.ClusterOnline
	ov.BrokerCount = stats.BrokerCount
	ov.OnlinePartitionCount = stats.OnlinePartitions
	ov.Version = stats.Version
	if names, err := c.ds.GetTopicNames(); err == nil {
		ov.TopicCount = len(names)
	} else if prev.hasStats {
		ov.TopicCount = prev.overview.TopicCount
	}
	c.cache[name] = cached{overview: ov, stats: stats, hasStats: true}
}

// ListClusters returns cached overviews for all clusters (stable order).
func (c *Collector) ListClusters() []api.ClusterOverview {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]api.ClusterOverview, 0, len(c.order))
	for _, n := range c.order {
		if e, ok := c.cache[n]; ok {
			out = append(out, e.overview)
		}
	}
	return out
}

// GetStatistics returns cached detailed stats for a cluster (cache-only).
func (c *Collector) GetStatistics(name string) (api.ClusterStatistics, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.cache[name]
	if !ok {
		return api.ClusterStatistics{}, api.ClusterNotFoundError{Name: name}
	}
	if !e.hasStats {
		return api.ClusterStatistics{}, api.NotSupportedError{Operation: "statistics not yet collected"}
	}
	return e.stats, nil
}

func (c *Collector) overview(name string) (api.ClusterOverview, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.cache[name]
	if !ok {
		return api.ClusterOverview{}, api.ClusterNotFoundError{Name: name}
	}
	return e.overview, nil
}
