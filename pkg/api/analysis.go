package api

import "time"

// AnalysisState is the lifecycle state of a topic analysis. (TP-29/TP-30)
type AnalysisState string

const (
	AnalysisRunning   AnalysisState = "running"
	AnalysisCompleted AnalysisState = "completed"
	AnalysisFailed    AnalysisState = "failed"
)

// AnalysisProgress is a point-in-time snapshot of a running analysis.
type AnalysisProgress struct {
	StartTime        time.Time
	ProcessedOffsets int64
	TotalOffsets     int64
	MessagesScanned  int64
	BytesScanned     int64
}

// Percentage returns the completion percentage, capped to [0,100]. When the
// total is unknown (0) it returns 0.
func (p AnalysisProgress) Percentage() float64 {
	if p.TotalOffsets <= 0 {
		return 0
	}
	pct := float64(p.ProcessedOffsets) / float64(p.TotalOffsets) * 100
	if pct > 100 {
		return 100
	}
	if pct < 0 {
		return 0
	}
	return pct
}

// SizeDistribution summarises a set of byte sizes.
type SizeDistribution struct {
	Count int64
	Sum   int64
	Min   int64
	Max   int64
	Avg   float64
	P50   int64
	P75   int64
	P95   int64
	P99   int64
	P999  int64
}

// PartitionAnalysis holds the aggregated stats scoped to a single partition.
type PartitionAnalysis struct {
	Partition    int32
	MessageCount int64
	MinOffset    int64
	MaxOffset    int64
}

// TopicAnalysisResult is the completed aggregation of a topic scan.
type TopicAnalysisResult struct {
	Topic                string
	MessageCount         int64
	MinOffset            int64
	MaxOffset            int64
	MinTimestamp         time.Time
	MaxTimestamp         time.Time
	NullKeys             int64
	NullValues           int64
	ApproxDistinctKeys   int64
	ApproxDistinctValues int64
	KeySize              SizeDistribution
	ValueSize            SizeDistribution
	// HourlyCounts maps a Unix-hour bucket (seconds truncated to the hour) to a
	// message count, retained only for the last 14 days.
	HourlyCounts map[int64]int64
	Partitions   []PartitionAnalysis
	CompletedAt  time.Time
}

// TopicAnalysis is the state + payload returned by GetTopicAnalysis. Exactly one
// of Result / Err is meaningful depending on State.
type TopicAnalysis struct {
	Topic    string
	State    AnalysisState
	Progress AnalysisProgress
	Result   *TopicAnalysisResult
	Err      string
	ErrAt    time.Time
}
