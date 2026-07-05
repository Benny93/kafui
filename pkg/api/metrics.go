package api

import "time"

// This file defines the metrics-and-monitoring vocabulary consumed by the
// background metrics collector (pkg/metrics) and the metrics page.
//
// kafui's primary metric source is offset deltas: message-in rates are derived
// by sampling per-topic message counts over time. Byte rates are only known
// when a metrics endpoint (Prometheus/JMX-exporter) is configured; until that
// path is implemented they are reported as RateUnknown (-1).

// RateUnknown marks a rate field whose value has not been (or cannot be)
// collected, distinguishing "unknown" from a real zero rate. It matches the
// convention already used by ClusterOverview.
const RateUnknown = -1.0

// MetricPoint is a single timestamped sample in a time series.
type MetricPoint struct {
	Time  time.Time
	Value float64
}

// TimeSeries is a time-ordered sequence of samples (oldest first) used to feed
// sparklines and summary aggregates on the metrics page.
type TimeSeries struct {
	Points []MetricPoint
}

// Values returns just the sample values in order, convenient for sparkline
// rendering.
func (ts TimeSeries) Values() []float64 {
	out := make([]float64, len(ts.Points))
	for i, p := range ts.Points {
		out[i] = p.Value
	}
	return out
}

// Summary aggregates a time-series window. OK is false for an empty series.
type Summary struct {
	Min   float64
	Max   float64
	Avg   float64
	Last  float64
	Count int
	OK    bool
}

// Summary computes min/max/avg/last over the retained window. Negative
// ("unknown", RateUnknown) samples are skipped so a gap does not poison the
// aggregate; a series that is entirely unknown yields OK=false.
func (ts TimeSeries) Summary() Summary {
	var s Summary
	var sum float64
	for _, p := range ts.Points {
		if p.Value < 0 {
			continue
		}
		if !s.OK {
			s.Min, s.Max = p.Value, p.Value
			s.OK = true
		}
		if p.Value < s.Min {
			s.Min = p.Value
		}
		if p.Value > s.Max {
			s.Max = p.Value
		}
		sum += p.Value
		s.Last = p.Value
		s.Count++
	}
	if s.Count > 0 {
		s.Avg = sum / float64(s.Count)
	}
	return s
}

// TopicMetrics is per-topic collected metrics for one cluster.
type TopicMetrics struct {
	Name             string
	PartitionCount   int32
	MessageCount     int64
	MessagesInPerSec float64 // derived from message-count deltas; RateUnknown until a prior sample exists
}

// BrokerMetrics is per-broker collected metrics. It is adapted from the
// broker-stats primitive; byte rates require a configured metrics endpoint and
// are otherwise RateUnknown.
type BrokerMetrics struct {
	ID             int32
	LeaderCount    int
	ReplicaCount   int
	SegmentSize    int64
	BytesInPerSec  float64 // RateUnknown unless a metrics endpoint is configured
	BytesOutPerSec float64 // RateUnknown unless a metrics endpoint is configured
}

// ClusterMetrics is the display-oriented metrics snapshot for one cluster,
// cached by the background collector and read by the metrics page. Rate fields
// use RateUnknown (-1) to mean "not yet / not collected".
type ClusterMetrics struct {
	Cluster          string
	CollectedAt      time.Time
	BrokerCount      int
	TopicCount       int
	PartitionCount   int
	MessageCount     int64
	MessagesInPerSec float64 // RateUnknown until a second collection cycle establishes a delta
	BytesInPerSec    float64 // RateUnknown unless a metrics endpoint is configured
	BytesOutPerSec   float64 // RateUnknown unless a metrics endpoint is configured
	Topics           []TopicMetrics
	Brokers          []BrokerMetrics
}
