package mainpage

import (
	"time"

	"github.com/Benny93/kafui/pkg/api"
)

// Custom message types for the main page
type (
	SearchTopicsMsg        string
	ClearSearchMsg         struct{}
	SwitchResourceMsg      ResourceType
	CurrentResourceListMsg struct {
		ResourceType ResourceType
		Items        []interface{} // Changed from []list.Item to []interface{}
	}
	TopicListMsg []TopicItem
	ErrorMsg     error
	TimerTickMsg time.Time

	// SelectContextMsg is sent when the user confirms a context selection.
	// The handler calls SetContext and reloads the topic list.
	SelectContextMsg struct {
		ContextName string
	}

	// TopicCountsLoadedMsg carries the result of the async message-count fetch.
	// Keys are topic names; values are total message counts across all partitions.
	TopicCountsLoadedMsg map[string]int64

	// SchemaDetailsLoadedMsg carries the result of an async schema-details fetch.
	// Each Schema has Subject, Version, ID, and SchemaType populated.
	SchemaDetailsLoadedMsg []api.Schema

	// TopicDetailsLoadedMsg carries the full topic map (NumPartitions,
	// ReplicationFactor) after the async detail-fetch phase completes.
	// A nil map signals that the detail fetch failed; stubs remain visible.
	TopicDetailsLoadedMsg map[string]api.Topic

	// ClearClipboardFeedbackMsg is sent after a short delay to clear the
	// "Copied!" feedback text from the table footer.
	ClearClipboardFeedbackMsg struct{}

	// ConsumerGroupDetailsLoadedMsg carries the result of the async, per-visible-page
	// consumer-group enrichment (GetConsumerGroupDetails). Each ConsumerGroup has
	// real State, MemberCount, TopicCount, Lag, CoordinatorID populated.
	ConsumerGroupDetailsLoadedMsg []api.ConsumerGroup

	// groupDeletedMsg is dispatched after a confirmed consumer-group deletion.
	groupDeletedMsg struct {
		groupID string
		err     error
	}

	// topicExtInfo carries the extended, per-visible-page topic enrichment:
	// out-of-sync (under-replicated) partition count and on-disk size.
	topicExtInfo struct {
		outOfSync  int
		size       int64
		isInternal bool
	}

	// TopicDetailsExtLoadedMsg carries the extended topic enrichment keyed by
	// topic name (OSR + size), fetched only for the currently visible page.
	TopicDetailsExtLoadedMsg map[string]topicExtInfo

	// topicCreatedMsg reports the outcome of a CreateTopic call (create/clone).
	topicCreatedMsg struct {
		name string
		err  error
	}

	// topicDeletedMsg / topicRecreatedMsg / topicPurgedMsg report single-row
	// mutation outcomes dispatched from a confirmation dialog's OnConfirm.
	topicDeletedMsg struct {
		name string
		err  error
	}
	topicRecreatedMsg struct {
		name string
		err  error
	}
	topicPurgedMsg struct {
		name string
		err  error
	}

	// topicBatchResultMsg reports the outcome of a batch delete/purge over the
	// current multi-selection. failures lists per-topic error strings.
	topicBatchResultMsg struct {
		action   string
		total    int
		failures []string
	}

	// BrokerStatsLoadedMsg carries the result of the async broker-stats fetch
	// (second phase of the two-phase broker load): per-broker partition/disk
	// statistics plus the cluster-wide summary shown in the sidebar panel.
	BrokerStatsLoadedMsg struct {
		Stats   map[int32]api.BrokerStats
		Summary api.BrokerSummary
	}

	// aclDeletedMsg reports the outcome of a confirmed ACL deletion (AQ-15).
	aclDeletedMsg struct {
		summary string
		err     error
	}

	// aclCreatedMsg reports the outcome of an ACL create / convenience-flow
	// submission (AQ-16/AQ-17). failures lists per-binding error strings.
	aclCreatedMsg struct {
		created  int
		failures []string
		err      error
	}

	// aclSyncedMsg reports the outcome of a confirmed declarative CSV sync (AQ-18).
	aclSyncedMsg struct {
		created int
		deleted int
		err     error
	}

	// quotaAlteredMsg reports the outcome of a quota upsert/delete (AQ-20).
	// action is one of "created", "updated", "deleted".
	quotaAlteredMsg struct {
		action string
		err    error
	}

	// connectClusterSelectedMsg drills from the Connect-clusters overview into
	// the aggregated connectors view, pre-filtered to the chosen cluster (KC-11).
	connectClusterSelectedMsg struct {
		cluster string
	}

	// connectorCreatedMsg reports the outcome of a create-connector submission
	// (KC-17). On success, connect+name identify the new connector to open.
	connectorCreatedMsg struct {
		connect string
		name    string
		err     error
	}
)

// TopicItem represents a topic item for display
type TopicItem struct {
	name  string
	topic api.Topic
}

func (t TopicItem) FilterValue() string {
	return t.name
}

func (t TopicItem) GetID() string {
	return t.name
}

func (t TopicItem) GetTopic() api.Topic {
	return t.topic
}

// ResourceType represents different types of Kafka resources
type ResourceType int

const (
	TopicResourceType ResourceType = iota
	ConsumerGroupResourceType
	SchemaResourceType
	ContextResourceType
	ACLResourceType
	BrokerResourceType
	QuotaResourceType
	ConnectClusterResourceType
	ConnectorResourceType
)

func (rt ResourceType) String() string {
	switch rt {
	case TopicResourceType:
		return "topics"
	case ConsumerGroupResourceType:
		return "consumer-groups"
	case SchemaResourceType:
		return "schemas"
	case ContextResourceType:
		return "contexts"
	case ACLResourceType:
		return "acls"
	case BrokerResourceType:
		return "brokers"
	case QuotaResourceType:
		return "quotas"
	case ConnectClusterResourceType:
		return "connect-clusters"
	case ConnectorResourceType:
		return "connectors"
	default:
		return "unknown"
	}
}

// Resource represents a Kafka resource that can be displayed and managed
type Resource interface {
	GetType() ResourceType
	GetName() string
	GetData() ([]ResourceItem, error)
}

// ResourceItem represents a displayable resource item
type ResourceItem interface {
	GetID() string
	GetValues() []string
	GetDetails() map[string]string
}
