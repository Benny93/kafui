package consumergroup

import (
	"time"

	"github.com/Benny93/kafui/pkg/api"
)

// Async load / mutation result messages. Each carries the group id it was
// requested for so stale responses (after navigation) can be discarded.
type (
	// detailLoadedMsg carries the result of a GetConsumerGroupDetail fetch.
	detailLoadedMsg struct {
		groupID  string
		detail   api.ConsumerGroupDetail
		notFound bool
		err      error
	}

	// autoRefreshTickMsg drives the auto-refresh loop for a given interval.
	autoRefreshTickMsg struct {
		groupID  string
		interval time.Duration
	}

	// groupDeletedMsg is dispatched after a confirmed group deletion.
	groupDeletedMsg struct {
		groupID string
		err     error
	}

	// offsetsDeletedMsg is dispatched after a confirmed per-topic offset deletion.
	offsetsDeletedMsg struct {
		groupID string
		topic   string
		err     error
	}

	// offsetsResetMsg is dispatched after a confirmed offset reset.
	offsetsResetMsg struct {
		groupID string
		topic   string
		err     error
	}

	// resetFormSubmitMsg is emitted by the reset form once its inputs validate.
	resetFormSubmitMsg struct {
		req api.OffsetResetRequest
	}

	// resetFormCancelMsg is emitted when the reset form is cancelled.
	resetFormCancelMsg struct{}
)

// topicRow is one aggregated topic line in the detail table.
type topicRow struct {
	topic      string
	partitions []api.PartitionOffset
	aggLag     *int64 // sum of partition lags; nil when the topic has no lag values
	prevAggLag *int64 // previous reading for trend indicator; nil when no baseline
}
