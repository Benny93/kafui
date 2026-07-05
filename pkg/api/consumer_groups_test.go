package api

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestGroupErrorsFormatAndUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	tests := []struct {
		name       string
		err        error
		wantMsg    string
		wantUnwrap error
	}{
		{
			name:       "group not found",
			err:        GroupNotFoundError{GroupID: "g1", Cause: cause},
			wantMsg:    "consumer group not found: g1",
			wantUnwrap: cause,
		},
		{
			name:       "group not empty carries state",
			err:        GroupNotEmptyError{GroupID: "g1", State: GroupStateStable, Cause: cause},
			wantMsg:    `consumer group "g1" is not empty (state: Stable)`,
			wantUnwrap: cause,
		},
		{
			name:       "invalid offset reset",
			err:        InvalidOffsetResetError{Reason: "timestamp mode requires a timestamp", Cause: cause},
			wantMsg:    "invalid offset reset: timestamp mode requires a timestamp",
			wantUnwrap: cause,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
			if got := errors.Unwrap(tt.err); got != tt.wantUnwrap {
				t.Errorf("Unwrap() = %v, want %v", got, tt.wantUnwrap)
			}
		})
	}
}

func TestGroupErrorsIsAs(t *testing.T) {
	wrapped := fmt.Errorf("context: %w", GroupNotEmptyError{GroupID: "g", State: GroupStatePreparingRebalance})
	var gne GroupNotEmptyError
	if !errors.As(wrapped, &gne) {
		t.Fatalf("errors.As failed to extract GroupNotEmptyError")
	}
	if gne.State != GroupStatePreparingRebalance {
		t.Errorf("state = %q, want %q", gne.State, GroupStatePreparingRebalance)
	}

	nf := GroupNotFoundError{GroupID: "x"}
	if !errors.Is(fmt.Errorf("w: %w", nf), nf) {
		t.Errorf("errors.Is failed for GroupNotFoundError")
	}
}

func TestOffsetResetRequestZeroValues(t *testing.T) {
	// Sanity: pointer/sentinel fields distinguish "unset" from zero.
	req := OffsetResetRequest{GroupID: "g", Topic: "t", Mode: OffsetResetTimestamp}
	if req.Timestamp != nil {
		t.Errorf("Timestamp should be nil when unset")
	}
	ts := time.Unix(0, 0)
	req.Timestamp = &ts
	if req.Timestamp == nil {
		t.Errorf("Timestamp should be set")
	}

	po := PartitionOffset{Topic: "t", Partition: 0}
	if po.CommittedOffset != nil || po.Lag != nil {
		t.Errorf("CommittedOffset/Lag should be nil by default")
	}
}
