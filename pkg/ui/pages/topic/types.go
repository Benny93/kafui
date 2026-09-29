package topic

import (
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components"
)

// ConsumeMode controls how the topic page fetches and displays messages.
type ConsumeMode int

const (
	// ModeNewest shows the latest N messages, paginated (Follow=false).
	ModeNewest ConsumeMode = iota
	// ModeOldest shows messages starting from the earliest offset, paginated (Follow=false).
	ModeOldest
	// ModeLive streams messages arriving in real time (Follow=true).
	ModeLive
)

// String returns a short label for display in the UI.
func (m ConsumeMode) String() string {
	switch m {
	case ModeNewest:
		return "Newest"
	case ModeOldest:
		return "Oldest"
	case ModeLive:
		return "Live"
	}
	return "Unknown"
}

// Next cycles to the next mode in order: Newest → Oldest → Live → Newest.
func (m ConsumeMode) Next() ConsumeMode {
	return (m + 1) % 3
}

// Custom message types for the topic page.
//
// Every message produced by a fetch or a live stream carries the Topic and the
// fetch generation (Gen) it was started for. The router delivers messages to
// whichever page is current, and a refresh, seek or mode switch starts a new
// generation, so handlers drop any message whose Topic or Gen is not the
// model's current one (see Model.isCurrent).
type (
	// Consumption messages
	MessageConsumedMsg struct {
		Message api.Message
		Topic   string
		Gen     uint64
	}

	StartConsumingMsg struct {
		MsgChan <-chan api.Message
		ErrChan <-chan error
		Cancel  func()
		Topic   string
		Gen     uint64
	}

	StopConsumingMsg struct{}

	// Continuous listening messages
	ContinuousListenMsg struct {
		Topic string
		Gen   uint64
	}

	ContinuousErrorListenMsg struct {
		Topic string
		Gen   uint64
	}

	// streamClosedMsg reports that a live stream's message or error channel
	// was closed. The handler decides whether that is an error.
	streamClosedMsg struct {
		Topic string
		Gen   uint64
	}

	// liveErrorMsg carries an error reported by a live stream.
	liveErrorMsg struct {
		Topic string
		Gen   uint64
		Err   error
	}

	// Connection status messages
	ConnectionStatusMsg string

	// Retry messages
	RetryConsumptionMsg struct {
		Attempt   int
		LastError error
		Topic     string
		Gen       uint64
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

	// Fetch messages (non-streaming). Err is the error the fetch hit, if any;
	// Messages may still hold what arrived before it. Append is true for
	// batch-pagination fetches, whose results merge into the loaded set.
	MessagesFetchedMsg struct {
		Messages []api.Message
		Err      error
		Append   bool
		Topic    string
		Gen      uint64
	}

	// StartFetchMsg signals that a background progress-tracked fetch has started.
	// ProgressCh delivers per-item ProgressMsg updates (owned by FetchProgressBar).
	// ResultCh delivers the final MessagesFetchedMsg when the fetch completes.
	// When Append is true the handler merges results into the existing message store
	// rather than replacing it.
	StartFetchMsg struct {
		ProgressCh <-chan components.ProgressMsg
		ResultCh   <-chan MessagesFetchedMsg
		Total      int
		Append     bool // true for batch-pagination fetches
		Topic      string
		Gen        uint64
	}

	// VisibleMessagesDecodedMsg carries the result of lazily decoding the visible page.
	// The handler merges decoded Key/Value back into the model's message store.
	VisibleMessagesDecodedMsg struct {
		Messages []api.Message
		Topic    string
		Gen      uint64
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
	Query           string
	PartitionFilter *int32
	OffsetRange     *OffsetRange
	TimeRange       *TimeRange
	KeyPattern      string
	ValuePattern    string
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
	Topic       string
	Partition   int32
	Offset      int64
	Follow      bool
	MaxMessages int
	Timeout     time.Duration
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
	MaxRetries        int
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	BackoffFactor     float64
	EnableExponential bool
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:        3,
		InitialDelay:      time.Second * 2,
		MaxDelay:          time.Second * 30,
		BackoffFactor:     2.0,
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
