package api

import (
	"errors"
	"testing"
)

// --- BR-1: broker error types ---

func TestBrokerErrors(t *testing.T) {
	cause := errors.New("root cause")

	tests := []struct {
		name    string
		err     error
		wantMsg string
		unwrap  error
	}{
		{
			name:    "BrokerNotFoundError",
			err:     BrokerNotFoundError{BrokerID: 7, Cause: cause},
			wantMsg: "broker not found: 7",
			unwrap:  cause,
		},
		{
			name:    "BrokerNotFoundError no cause",
			err:     BrokerNotFoundError{BrokerID: 3},
			wantMsg: "broker not found: 3",
			unwrap:  nil,
		},
		{
			name:    "LogDirNotFoundError",
			err:     LogDirNotFoundError{Path: "/var/lib/kafka", Cause: cause},
			wantMsg: "log directory not found: /var/lib/kafka",
			unwrap:  cause,
		},
		{
			name:    "InvalidConfigError",
			err:     InvalidConfigError{Key: "retention.ms", Reason: "bad value", Cause: cause},
			wantMsg: `invalid config "retention.ms": bad value`,
			unwrap:  cause,
		},
		{
			name:    "MetricsNotAvailableError",
			err:     MetricsNotAvailableError{BrokerID: 2, Cause: cause},
			wantMsg: "metrics not available for broker 2",
			unwrap:  cause,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
			if got := errors.Unwrap(tt.err); got != tt.unwrap {
				t.Errorf("Unwrap() = %v, want %v", got, tt.unwrap)
			}
		})
	}
}

func TestBrokerNotFoundError_Is(t *testing.T) {
	cause := errors.New("boom")
	err := BrokerNotFoundError{BrokerID: 1, Cause: cause}
	if !errors.Is(err, cause) {
		t.Errorf("errors.Is should find wrapped cause")
	}
}

// --- BR-4: skew rounding / thresholds ---

func TestRoundHalfUp(t *testing.T) {
	tests := []struct {
		v    float64
		want float64
	}{
		{12.35, 12.4}, // exact half rounds up
		{12.34, 12.3},
		{12.36, 12.4},
		{12.25, 12.3},   // exact half rounds up
		{-12.35, -12.4}, // negative half rounds away from zero
		{9.99, 10.0},
		{20.0, 20.0},
		{0.05, 0.1},
	}
	for _, tt := range tests {
		if got := RoundHalfUp(tt.v, 1); got != tt.want {
			t.Errorf("RoundHalfUp(%v, 1) = %v, want %v", tt.v, got, tt.want)
		}
	}
}

func TestComputeSkew(t *testing.T) {
	tests := []struct {
		name            string
		brokerCount     int
		avg             float64
		totalPartitions int
		want            *float64 // nil = absent
	}{
		{
			name:            "below threshold 49 partitions -> absent",
			brokerCount:     20,
			avg:             10,
			totalPartitions: 49,
			want:            nil,
		},
		{
			name:            "at threshold 50 partitions -> computed",
			brokerCount:     12,
			avg:             10,
			totalPartitions: 50,
			want:            f64Ptr(20.0), // ((12-10)/10)*100
		},
		{
			name:            "zero average -> absent",
			brokerCount:     5,
			avg:             0,
			totalPartitions: 100,
			want:            nil,
		},
		{
			name:            "missing broker count treated as 0",
			brokerCount:     0,
			avg:             10,
			totalPartitions: 60,
			want:            f64Ptr(-100.0), // ((0-10)/10)*100
		},
		{
			name:            "half-up rounding edge 12.35 -> 12.4",
			brokerCount:     2247,
			avg:             2000,
			totalPartitions: 60,
			want:            f64Ptr(12.4),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeSkew(tt.brokerCount, tt.avg, tt.totalPartitions)
			switch {
			case tt.want == nil && got != nil:
				t.Errorf("ComputeSkew = %v, want nil", *got)
			case tt.want != nil && got == nil:
				t.Errorf("ComputeSkew = nil, want %v", *tt.want)
			case tt.want != nil && got != nil && *got != *tt.want:
				t.Errorf("ComputeSkew = %v, want %v", *got, *tt.want)
			}
		})
	}
}

func f64Ptr(v float64) *float64 { return &v }
