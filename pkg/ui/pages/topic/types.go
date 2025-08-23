package topic

import (
	"time"

	"github.com/Benny93/kafui/pkg/api"
)

// Custom message types for the topic page
type (
	// Consumption messages
	MessageConsumedMsg struct {
		Message api.Message
	}

	StartConsumingMsg struct {
		MsgChan <-chan api.Message
		ErrChan <-chan error
		Cancel  func()
	}

	StopConsumingMsg struct{}

	// Continuous listening messages
	ContinuousListenMsg struct{}

	ContinuousErrorListenMsg struct{}

	// Connection status messages
	ConnectionStatusMsg string

	// Retry messages
	RetryConsumptionMsg struct {
		Attempt   int
		LastError error
	}

	ConnectionFailedMsg struct {
		Attempts  int
		LastError error
	}

	// Search messages
	SearchMessagesMsg string

	ClearSearchMsg struct{}

	// Message selection
	MessageSelectedMsg struct {
		Message api.Message
	}

	// Timer messages
	TimerTickMsg time.Time

	ErrorMsg error
)

// Constants for connection status
const (
	StatusConnecting   = "connecting"
	StatusConnected    = "connected"
	StatusDisconnected = "disconnected"
	StatusFailed       = "failed"
	StatusRetrying     = "retrying"
)

// Constants for consumption states
const (
	StateIdle      = "idle"
	StateConsuming = "consuming"
	StatePaused    = "paused"
	StateStopped   = "stopped"
	StateError     = "error"
)

// ConsumptionState represents the current state of message consumption
type ConsumptionState struct {
	State            string
	MessagesConsumed int
	LastMessage      *api.Message
	LastError        error
	StartTime        time.Time
	LastUpdateTime   time.Time
}

// MessageFilter represents filtering options for messages
type MessageFilter struct {
	Query         string
	PartitionFilter *int32
	OffsetRange   *OffsetRange
	TimeRange     *TimeRange
	KeyPattern    string
	ValuePattern  string
}

// OffsetRange represents a range of offsets
type OffsetRange struct {
	Start int64
	End   int64
}

// TimeRange represents a time range
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// ConsumptionConfig represents configuration for message consumption
type ConsumptionConfig struct {
	Topic      string
	Partition  int32
	Offset     int64
	Follow     bool
	MaxMessages int
	Timeout    time.Duration
}

// MessageDisplayFormat represents how messages should be displayed
type MessageDisplayFormat struct {
	ShowHeaders    bool
	ShowTimestamp  bool
	ShowPartition  bool
	ShowOffset     bool
	ValueFormat    string // "raw", "json", "avro", etc.
	KeyFormat      string
	MaxKeyLength   int
	MaxValueLength int
}

// TopicInfo represents detailed information about a topic
type TopicInfo struct {
	Name              string
	Partitions        int32
	ReplicationFactor int16
	ConfigEntries     map[string]*string
	MessageCount      int64
	ConsumerGroups    []string
}

// ErrorContext provides context for errors
type ErrorContext struct {
	Operation string
	Topic     string
	Partition int32
	Offset    int64
	Timestamp time.Time
	Details   map[string]interface{}
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	EnableExponential bool
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:      3,
		InitialDelay:    time.Second * 2,
		MaxDelay:        time.Second * 30,
		BackoffFactor:   2.0,
		EnableExponential: true,
	}
}

// DefaultMessageDisplayFormat returns default display format
func DefaultMessageDisplayFormat() MessageDisplayFormat {
	return MessageDisplayFormat{
		ShowHeaders:    true,
		ShowTimestamp:  true,
		ShowPartition:  true,
		ShowOffset:     true,
		ValueFormat:    "raw",
		KeyFormat:      "raw",
		MaxKeyLength:   20,
		MaxValueLength: 40,
	}
}