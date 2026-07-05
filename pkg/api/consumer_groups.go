package api

import "time"

// Canonical consumer-group states. Sarama returns backend-specific state
// strings; datasources normalize them to these values (falling back to
// GroupStateUnknown for anything unrecognized).
const (
	GroupStateStable              = "Stable"
	GroupStatePreparingRebalance  = "PreparingRebalance"
	GroupStateCompletingRebalance = "CompletingRebalance"
	GroupStateEmpty               = "Empty"
	GroupStateDead                = "Dead"
	GroupStateUnknown             = "Unknown"
)

// TopicPartition identifies a single partition of a topic.
type TopicPartition struct {
	Topic     string
	Partition int32
}

// GroupMember is a single active member of a consumer group.
type GroupMember struct {
	ConsumerID  string // MemberId assigned by the coordinator
	ClientID    string // ClientId from the member's join request
	Host        string // ClientHost
	Assignments []TopicPartition
}

// PartitionOffset holds the committed/end offsets and derived lag for one
// partition of a consumer group.
//
// CommittedOffset is nil when the group has no committed offset for the
// partition (e.g. a partition assigned to a member that has not committed yet).
// Lag is nil when it is undefined (no committed offset). These pointers keep
// "no value" distinguishable from a genuine 0.
type PartitionOffset struct {
	Topic           string
	Partition       int32
	CommittedOffset *int64
	EndOffset       int64
	Lag             *int64
	MemberID        string // consumer assigned to this partition, if any
	MemberHost      string
}

// ConsumerGroupDetail is the full description of a single consumer group.
type ConsumerGroupDetail struct {
	GroupID           string
	State             string // canonical GroupState* value
	ProtocolType      string
	PartitionAssignor string
	IsSimple          bool // protocol type is not "consumer"
	CoordinatorID     int32
	Members           []GroupMember
	TopicOffsets      []PartitionOffset
}

// GroupLag is an aggregate lag view of a consumer group.
// TotalLag / PerTopic entries are nil when lag is undefined.
type GroupLag struct {
	TotalLag *int64
	PerTopic map[string]*int64
}

// OffsetResetMode selects how target offsets are computed for a reset.
type OffsetResetMode string

const (
	OffsetResetEarliest  OffsetResetMode = "earliest"
	OffsetResetLatest    OffsetResetMode = "latest"
	OffsetResetTimestamp OffsetResetMode = "timestamp"
	OffsetResetExplicit  OffsetResetMode = "explicit"
)

// OffsetResetRequest describes a consumer-group offset reset. An empty
// Partitions slice targets all partitions of the topic.
type OffsetResetRequest struct {
	GroupID          string
	Topic            string
	Mode             OffsetResetMode
	Partitions       []int32
	Timestamp        *time.Time      // required for OffsetResetTimestamp
	PartitionOffsets map[int32]int64 // required for OffsetResetExplicit
}
