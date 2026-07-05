package api

import (
	"context"
	"strings"
	"time"
)

type MessageHeader struct {
	Key   string
	Value string
}
type MessageHeaders []MessageHeader

// TimestampType describes how a message's Timestamp was assigned by the broker.
type TimestampType string

const (
	TimestampTypeNone      TimestampType = "none"       // no timestamp available
	TimestampTypeCreate    TimestampType = "create"     // CreateTime (producer-assigned)
	TimestampTypeLogAppend TimestampType = "log-append" // LogAppendTime (broker-assigned)
)

type Message struct {
	Key   string
	Value string
	// RawKey and RawValue hold the original Kafka bytes for Avro-encoded messages.
	// They are populated at consumption time and decoded lazily via DecodeMessage.
	RawKey        []byte
	RawValue      []byte
	Offset        int64
	Partition     int32
	KeySchemaID   string
	ValueSchemaID string
	Headers       []MessageHeader
	// Timestamp is the message's broker timestamp. Zero when unknown. Used by the
	// analysis engine for time-range and hourly-bucket aggregation.
	Timestamp time.Time
	// TimestampType records how Timestamp was assigned (create vs log-append).
	TimestampType TimestampType
	// KeySize/ValueSize hold the raw byte sizes of the key/value. They are nil
	// when the key/value is null (as opposed to an empty byte slice, size 0).
	KeySize   *int
	ValueSize *int
	// HeadersSize is the summed byte size of all header keys and values.
	HeadersSize int
	// KeyNull/ValueNull distinguish a null key/value from an empty one.
	KeyNull   bool
	ValueNull bool
	// KeySerde/ValueSerde name the serde used to render the key/value. Until the
	// serde framework lands these carry the active decoder name (e.g. "avro").
	KeySerde   string
	ValueSerde string
}

type Topic struct {
	// NumPartitions contains the number of partitions to create in the topic
	NumPartitions int32
	// ReplicationFactor contains the number of replicas to create for each partition
	ReplicationFactor int16
	// ReplicaAssignment contains the manual partition assignment, or the empty
	// array if we are using automatic assignment.
	ReplicaAssignment map[int32][]int32
	// ConfigEntries contains the custom topic configurations to set.
	ConfigEntries map[string]*string
	// Num of messages in the topic across all partitions
	MessageCount int64
}

// SeekMode selects where a browse starts (and, for backward modes, ends).
type SeekMode string

const (
	SeekNewest        SeekMode = "newest"         // most recent messages (default)
	SeekOldest        SeekMode = "oldest"         // from the earliest available offset
	SeekLive          SeekMode = "live"           // tail newly produced messages
	SeekFromOffset    SeekMode = "from-offset"    // forward from a given offset
	SeekToOffset      SeekMode = "to-offset"      // backward window ending at a given offset
	SeekFromTimestamp SeekMode = "from-timestamp" // forward from a given timestamp
	SeekToTimestamp   SeekMode = "to-timestamp"   // backward window ending at a given timestamp
)

// Backward reports whether the seek mode reads newest-first (results ordered descending).
func (m SeekMode) Backward() bool {
	switch m {
	case SeekNewest, SeekToOffset, SeekToTimestamp, "":
		return true
	default:
		return false
	}
}

type ConsumeFlags struct {
	Follow        bool
	Tail          int32
	OffsetFlag    string
	GroupFlag     string
	LimitMessages int64 // stop after N messages (0 = unlimited, runs until HWM or idle timeout)

	// Typed seek model (MSG-1). When Seek is set it takes precedence over
	// OffsetFlag for per-partition offset resolution; OffsetFlag remains for
	// backward compatibility with existing callers.
	Seek          SeekMode
	SeekOffset    *int64     // required for SeekFromOffset / SeekToOffset
	SeekTimestamp *time.Time // required for SeekFromTimestamp / SeekToTimestamp
	Partitions    []int32    // empty = all partitions
}

// Validate checks that offset/timestamp seek modes carry their required value.
func (f ConsumeFlags) Validate() error {
	switch f.Seek {
	case "", SeekNewest, SeekOldest, SeekLive:
		return nil
	case SeekFromOffset, SeekToOffset:
		if f.SeekOffset == nil {
			return InvalidSeekError{Mode: string(f.Seek), Reason: "offset value is required"}
		}
	case SeekFromTimestamp, SeekToTimestamp:
		if f.SeekTimestamp == nil {
			return InvalidSeekError{Mode: string(f.Seek), Reason: "timestamp value is required"}
		}
	default:
		return InvalidSeekError{Mode: string(f.Seek), Reason: "unknown seek mode"}
	}
	return nil
}

func DefaultConsumeFlags() ConsumeFlags {
	return ConsumeFlags{
		Follow:     true,
		Tail:       50,
		OffsetFlag: "latest",
		Seek:       SeekNewest,
	}
}

// ProduceRecord is a message to be produced to a topic. A nil Key or Value
// produces a null record, which is distinct from an empty (non-nil) slice.
type ProduceRecord struct {
	Key       []byte
	Value     []byte
	Headers   []MessageHeader
	Partition *int32 // nil = let the partitioner choose
}

type ConsumerGroup struct {
	Name      string
	State     string
	Consumers int

	// Enrichment fields populated lazily by GetConsumerGroupDetails for the
	// visible page of rows. Lag is nil when undefined (no committed offsets).
	MemberCount       int
	TopicCount        int
	Lag               *int64
	CoordinatorID     int32
	PartitionAssignor string
	IsSimple          bool
}

// ACLEntry represents a single Kafka ACL binding.
type ACLEntry struct {
	Principal    string // e.g. "User:CN=devuser,..."
	Host         string // e.g. "*" or specific IP
	ResourceType string // "Topic", "Group", "Cluster", etc.
	ResourceName string // resource name or "*"
	PatternType  string // "Literal", "Prefixed", etc.
	Operation    string // "Read", "Write", "Describe", etc.
	Permission   string // "Allow" or "Deny"
}

// ACLFilter narrows an ACL listing by resource dimensions. An empty field
// matches any value.
type ACLFilter struct {
	ResourceType string // "Topic", "Group", ... ("" = any)
	ResourceName string // exact resource name ("" = any)
	PatternType  string // "Literal", "Prefixed", ... ("" = any)
}

// ClientQuotaEntity identifies the target of a client quota. Each identifier is
// nil when absent, a pointer to the empty string for the <default> entity, and a
// pointer to a concrete value otherwise.
type ClientQuotaEntity struct {
	User     *string
	ClientID *string
	IP       *string
}

// ClientQuotaEntry is a quota entity together with its configured quota values
// (e.g. "producer_byte_rate" -> 1048576).
type ClientQuotaEntry struct {
	Entity ClientQuotaEntity
	Quotas map[string]float64
}

// ValidatePrincipal checks that a principal is a non-empty "<type>:<name>"
// pair. The name part may itself contain colons (e.g. an SSL DN such as
// "User:CN=alice,OU=eng").
func ValidatePrincipal(principal string) error {
	if principal == "" {
		return ACLValidationError{Field: "principal", Reason: "must not be empty"}
	}
	idx := strings.Index(principal, ":")
	if idx <= 0 || idx == len(principal)-1 {
		return ACLValidationError{Field: "principal", Reason: "must be in the form <type>:<name> (e.g. User:alice)"}
	}
	return nil
}

// ValidateACLEntry validates the shared, backend-agnostic invariants of an ACL
// binding: a well-formed principal and non-empty resource type/name, operation
// and permission. Host and PatternType defaulting is the caller's concern.
func ValidateACLEntry(e ACLEntry) error {
	if err := ValidatePrincipal(e.Principal); err != nil {
		return err
	}
	if e.ResourceType == "" {
		return ACLValidationError{Field: "resourceType", Reason: "must not be empty"}
	}
	if e.ResourceName == "" {
		return ACLValidationError{Field: "resourceName", Reason: "must not be empty"}
	}
	if e.Operation == "" {
		return ACLValidationError{Field: "operation", Reason: "must not be empty"}
	}
	if e.Permission == "" {
		return ACLValidationError{Field: "permission", Reason: "must not be empty"}
	}
	return nil
}

// ValidateQuotaEntity checks that at least one of the entity's identifiers is
// set (non-nil). A fully-absent entity yields a QuotaValidationError.
func ValidateQuotaEntity(e ClientQuotaEntity) error {
	if e.User == nil && e.ClientID == nil && e.IP == nil {
		return QuotaValidationError{Reason: "entity id not set: at least one of user, client-id or ip is required"}
	}
	return nil
}

type Schema struct {
	Subject    string `json:"subject"`
	Version    int    `json:"version"`
	ID         int    `json:"id"`
	SchemaType string `json:"schemaType"` // AVRO, PROTOBUF, JSON — empty means AVRO
	// Compatibility is the subject's effective compatibility level (falling back
	// to the global level when the subject has no own setting). Empty when not
	// resolved. (SR-6)
	Compatibility string `json:"compatibility,omitempty"`
}

type SchemaInfo struct {
	ID         int    `json:"id"`
	Schema     string `json:"schema"`
	Subject    string `json:"subject"`
	Version    int    `json:"version"`
	RecordName string `json:"recordName"` // The type name (e.g., AddedItemToChartEvent)
}

// MessageSchemaInfo contains schema information for a message's key and value
type MessageSchemaInfo struct {
	KeySchema   *SchemaInfo `json:"keySchema,omitempty"`
	ValueSchema *SchemaInfo `json:"valueSchema,omitempty"`
}

type MessageHandlerFunc func(msg Message)

// ClusterInfo holds the configuration details of a single Kafka cluster/context.
type ClusterInfo struct {
	Name              string
	Brokers           []string
	SchemaRegistryURL string
	IsCurrent         bool
	ReadOnly          bool
}

type KafkaDataSource interface {
	Init(cfgOption string)
	GetTopics() (map[string]Topic, error)
	// GetTopicNames returns only topic names using a lightweight metadata request.
	// This is significantly faster than GetTopics() on large clusters because it
	// skips per-partition replica details. Use it to show names immediately, then
	// call GetTopics() asynchronously to fill in partition/replication details.
	GetTopicNames() ([]string, error)
	GetContexts() ([]string, error)
	GetContext() string
	SetContext(contextName string) error
	// GetClusterDetails returns configuration details for the named cluster.
	// Used by the context view to display broker addresses and schema registry URL.
	GetClusterDetails(clusterName string) (ClusterInfo, error)
	GetConsumerGroups() ([]ConsumerGroup, error)
	// GetConsumerGroupDetail returns the full description of ONE consumer group,
	// including per-partition committed/end offsets and lag. It describes only
	// the named group (never the whole listing) and returns a GroupNotFoundError
	// when the group id is unknown.
	GetConsumerGroupDetail(groupID string) (ConsumerGroupDetail, error)
	// GetConsumerGroupDetails enriches a batch of groups (the currently visible
	// page of list rows) with real state, member/topic counts, lag, coordinator
	// and assignor. It is a bounded fan-out — pass only the visible names, never
	// the entire cluster. Groups that fail to describe keep state Unknown and nil
	// lag rather than failing the whole batch.
	GetConsumerGroupDetails(groupIDs []string) ([]ConsumerGroup, error)
	// GetConsumerGroupsForTopic returns groups related to a topic: a group is
	// related if any active member is assigned a partition of the topic or the
	// group has committed offsets for it. This inherently fans out and should be
	// invoked only on explicit user request.
	GetConsumerGroupsForTopic(topic string) ([]ConsumerGroup, error)
	// DeleteConsumerGroup deletes a consumer group. It returns a GroupNotEmptyError
	// when the group still has active members and a GroupNotFoundError when unknown.
	DeleteConsumerGroup(groupID string) error
	// DeleteConsumerGroupOffsets deletes the committed offsets of a single topic
	// for a group, leaving other topics' offsets and the group itself intact.
	DeleteConsumerGroupOffsets(groupID string, topic string) error
	// ResetConsumerGroupOffsets resets a group's committed offsets for a topic.
	// The group must be inactive (Empty or Dead); otherwise a GroupNotEmptyError
	// is returned. Invalid requests yield an InvalidOffsetResetError.
	ResetConsumerGroupOffsets(ctx context.Context, req OffsetResetRequest) error
	ConsumeTopic(ctx context.Context, topicName string, flags ConsumeFlags, handleMessage MessageHandlerFunc, onError func(err any)) error
	// ProduceMessage produces a single record to the topic. It validates that
	// the topic exists and that any explicit partition is in range, returning a
	// TopicNotFoundError, PartitionError, or ProduceError respectively.
	ProduceMessage(ctx context.Context, topic string, rec ProduceRecord) error
	// GetTopicMessageCounts fetches approximate message counts for a set of topics.
	// It accepts a map of topicName → numPartitions (already known from GetTopics).
	// Counts are computed as sum(newestOffset - oldestOffset) across all partitions.
	// Returns a best-effort map: topics that fail are omitted rather than failing the whole call.
	GetTopicMessageCounts(topics map[string]int32) (map[string]int64, error)
	// GetSchemas returns all registered schema subjects from the schema registry.
	// It fetches only subject names — no version/ID/type details — so it completes
	// with a single HTTP request even for large registries.
	// Returns an empty slice (not an error) when no schema registry is configured.
	GetSchemas() ([]Schema, error)
	// GetSchemaDetails fetches the latest version metadata (version, id, schemaType)
	// for a specific set of subjects. Used to lazily populate table rows for the
	// current page rather than loading details for all subjects at once.
	GetSchemaDetails(subjects []string) ([]Schema, error)
	// GetSchemaContent fetches the full schema definition string (Avro JSON, Protobuf IDL, etc.)
	// for the given subject. Pass version=0 to retrieve the latest version.
	GetSchemaContent(subject string, version int) (string, error)
	// GetSchemaVersions lists all registered versions of a subject (metadata only;
	// Schema text is left empty — fetch it lazily via GetSchemaContent). Returns a
	// SubjectNotFoundError for an unknown subject and a
	// SchemaRegistryNotConfiguredError when no registry is configured. (SR-4)
	GetSchemaVersions(subject string) ([]SchemaVersion, error)
	// GetGlobalCompatibility returns the registry's global compatibility level. (SR-5)
	GetGlobalCompatibility() (CompatibilityLevel, error)
	// GetSubjectCompatibility returns a subject's effective compatibility level.
	// When the subject has no own setting it falls back to the global level and
	// returns isSubjectSpecific=false. (SR-5)
	GetSubjectCompatibility(subject string) (level CompatibilityLevel, isSubjectSpecific bool, err error)
	// RegisterSchema registers a new schema under the subject, creating the subject
	// if new or a new version otherwise. schemaType may be empty for AVRO. It maps
	// a 409 to SchemaIncompatibleError and a 422 to SchemaValidationError. (SR-7)
	RegisterSchema(subject, schemaText, schemaType string) (Schema, error)
	// CheckSchemaCompatibility tests a candidate schema against the subject's latest
	// version without registering it, returning the verbose messages on failure. (SR-8)
	CheckSchemaCompatibility(subject, schemaText, schemaType string) (compatible bool, messages []string, err error)
	// DeleteSubject deletes all versions of a subject. permanent=true performs a
	// hard delete (requires a prior soft delete). It returns the deleted version
	// numbers. (SR-9)
	DeleteSubject(subject string, permanent bool) ([]int, error)
	// DeleteSchemaVersion deletes a single version. Pass version=-1 to delete the
	// latest version. permanent=true performs a hard delete. (SR-9)
	DeleteSchemaVersion(subject string, version int, permanent bool) error
	// SetGlobalCompatibility sets the registry's global compatibility level. It
	// validates the level before making any HTTP call. (SR-10)
	SetGlobalCompatibility(level CompatibilityLevel) error
	// SetSubjectCompatibility sets a subject's compatibility level. It validates
	// the level before making any HTTP call. (SR-10)
	SetSubjectCompatibility(subject string, level CompatibilityLevel) error
	// GetACLs returns all ACL bindings for the current cluster.
	// Returns an empty slice (not an error) when the broker returns no ACLs
	// or the connected user lacks the DESCRIBE ACL on cluster resources.
	GetACLs() ([]ACLEntry, error)
	// GetACLsFiltered returns ACL bindings matching the given filter. Empty
	// filter fields match any value; GetACLs is the match-any case.
	GetACLsFiltered(filter ACLFilter) ([]ACLEntry, error)
	// CreateACL creates a single ACL binding. The entry is validated (principal
	// format, non-empty resource/operation/permission); an empty PatternType
	// defaults to Literal. Invalid entries yield an ACLValidationError.
	CreateACL(entry ACLEntry) error
	// DeleteACL deletes the ACL binding exactly matching the full entry
	// definition. It returns an ACLNotFoundError when no binding matches.
	DeleteACL(entry ACLEntry) error
	// GetClientQuotas returns all configured client quotas, deterministically
	// ordered (user -> client-id -> ip, absent identifiers last).
	GetClientQuotas() ([]ClientQuotaEntry, error)
	// AlterClientQuotas upserts the quota values for an entity with replace
	// semantics: the submitted map becomes the entity's complete property set
	// (absent properties are cleared); an empty/nil map deletes the entity.
	// A QuotaValidationError is returned when no entity identifier is set.
	AlterClientQuotas(entity ClientQuotaEntity, quotas map[string]float64) error
	// GetMessageSchemaInfo retrieves schema information for a message's key and value
	// Returns nil for non-Avro messages or when schema information is not available
	GetMessageSchemaInfo(keySchemaID, valueSchemaID string) (*MessageSchemaInfo, error)
	// DecodeMessage decodes the raw bytes of a message (e.g. Avro) into human-readable
	// Key/Value strings. If the message is already decoded or has no raw bytes to decode,
	// it is returned unchanged. This is used for lazy, on-demand decoding of visible messages.
	DecodeMessage(ctx context.Context, msg Message) (Message, error)
	// ListSerdes returns the names of the serdes available for decoding message
	// keys/values (for the topic-page serde selector). The UI offers "auto"
	// (auto-detection) in addition to these explicit names.
	ListSerdes() []string
	// GetClusterStatistics collects a fresh detailed snapshot of the named cluster.
	// No caching happens here; the background collector (pkg/cluster) owns caching.
	GetClusterStatistics(ctx context.Context, clusterName string) (ClusterStatistics, error)
	// GetClusterCapabilities probes which optional features the named cluster supports.
	GetClusterCapabilities(ctx context.Context, clusterName string) ([]Capability, error)
	// ValidateClusterConnection independently probes each component (broker, schema
	// registry, ...) of the named cluster and returns one result per component.
	// Works for non-active clusters without switching the active context.
	ValidateClusterConnection(ctx context.Context, clusterName string) ([]ValidationResult, error)

	// GetBrokers returns the brokers currently online in the active cluster,
	// with IsController set on the active controller.
	GetBrokers() ([]BrokerInfo, error)
	// GetBrokerStats returns per-broker partition-distribution / disk statistics
	// plus a cluster-wide BrokerSummary. Skew fields are absent (nil) when the
	// cluster has too few partitions to be meaningful.
	GetBrokerStats() (map[int32]BrokerStats, BrokerSummary, error)
	// GetBrokerLogDirs returns log directories for the given broker IDs. An empty
	// brokerIDs slice means "all brokers"; unknown IDs are silently dropped. On
	// timeout it returns an empty result rather than an error.
	GetBrokerLogDirs(brokerIDs []int32) (map[int32][]BrokerLogDir, error)
	// GetBrokerConfig returns the configuration entries for a broker. Unknown
	// broker IDs yield a BrokerNotFoundError.
	GetBrokerConfig(brokerID int32) ([]BrokerConfigEntry, error)
	// AlterBrokerConfig sets a single dynamic broker config key, preserving other
	// dynamic configs. A cluster rejection yields an InvalidConfigError.
	AlterBrokerConfig(brokerID int32, key, value string) error
	// AlterReplicaLogDir moves a topic-partition replica to a different log dir
	// on the target broker.
	AlterReplicaLogDir(brokerID int32, topic string, partition int32, logDir string) error
	// GetBrokerMetrics returns a JSON metrics snapshot for a broker, or a
	// MetricsNotAvailableError when metrics collection is not available.
	GetBrokerMetrics(brokerID int32) (string, error)

	// --- Topic administration (TP-2..TP-11) ---

	// GetTopicConfig returns the topic's configuration entries with metadata
	// (default value derived from synonyms, source, sensitive, read-only). It
	// returns an empty slice (not an error) on authorization failure.
	GetTopicConfig(topicName string) ([]TopicConfigEntry, error)
	// GetTopicDetails returns per-partition detail plus topic-wide health
	// metrics. It returns a TopicNotFoundError when the topic does not exist.
	GetTopicDetails(topicName string) (TopicDetails, error)
	// GetTopicSizes returns the on-disk size (leader replicas only) per topic.
	// Best-effort: topics that fail are omitted from the result map.
	GetTopicSizes(topicNames []string) (map[string]int64, error)
	// CreateTopic creates a topic. A replicationFactor of -1 requests the
	// cluster default. Empty-valued config entries must be stripped by the
	// caller. It polls metadata until the topic is visible.
	CreateTopic(name string, numPartitions int32, replicationFactor int16, configs map[string]*string) error
	// DeleteTopic deletes a topic.
	DeleteTopic(name string) error
	// IsTopicDeletionEnabled reports whether the cluster permits topic deletion
	// (delete.topic.enable). Missing/unparseable config defaults to true.
	IsTopicDeletionEnabled() (bool, error)
	// UpdateTopicConfig incrementally alters the given topic config entries,
	// leaving unrelated entries untouched.
	UpdateTopicConfig(name string, entries map[string]*string) error
	// IncreasePartitions raises the topic's total partition count. It rejects a
	// decrease (PartitionDecreaseError) or a no-op (PartitionNoopError).
	IncreasePartitions(name string, totalCount int32) error
	// PurgeTopicMessages deletes records up to the high-watermark. A partition of
	// -1 purges all partitions. It returns a CleanupPolicyError for compact-only
	// topics.
	PurgeTopicMessages(name string, partition int32) error
	// RecreateTopic deletes and re-creates a topic preserving its partition
	// count, replication factor, and non-default configs.
	RecreateTopic(name string) error
	// ChangeReplicationFactor changes the replication factor of every partition
	// via a computed balanced reassignment across online brokers.
	ChangeReplicationFactor(name string, newFactor int16) error

	// --- Topic analysis (TP-29/TP-30) ---

	// StartTopicAnalysis begins a background scan+aggregation of the topic. It
	// returns an AnalysisAlreadyRunningError if one is already in progress and a
	// TopicNotFoundError for an unknown topic.
	StartTopicAnalysis(ctx context.Context, topicName string) error
	// GetTopicAnalysis returns the latest analysis (running/completed/failed) for
	// the topic, or (nil, nil) when none has ever been started.
	GetTopicAnalysis(topicName string) (*TopicAnalysis, error)
	// CancelTopicAnalysis cancels a running analysis, releasing its resources and
	// retaining no result.
	CancelTopicAnalysis(topicName string) error

	// --- Kafka Connect (KC-3) ---

	// GetConnectClusters lists the Connect clusters configured for the active
	// Kafka cluster. Each cluster is probed for its runtime info (version, commit,
	// kafka_cluster_id); an unreachable cluster is still listed with Reachable=false
	// and empty runtime fields (never failing the whole call). When withStats is
	// true it additionally computes per-cluster connector/task counts, which is
	// more expensive as it fetches connector status.
	GetConnectClusters(withStats bool) ([]ConnectCluster, error)
	// GetConnectorNames returns the connector names of a single Connect cluster
	// (fast, names only). Unknown cluster name yields a ConnectClusterNotFoundError.
	GetConnectorNames(connect string) ([]string, error)
	// GetConnectors returns connectors aggregated across all configured Connect
	// clusters, each enriched with type, topics, status and task counts. Unreachable
	// clusters are omitted (their connectors dropped) rather than failing the call.
	GetConnectors() ([]Connector, error)
	// GetConnectorDetails combines a connector's config, status, tasks and topics.
	// Config values are masked. Missing status yields state UNASSIGNED with an
	// empty task list rather than an error.
	GetConnectorDetails(connect, name string) (ConnectorDetails, error)
	// CreateConnector creates a connector from the given config. A duplicate name
	// yields a ConnectorAlreadyExistsError.
	CreateConnector(connect, name string, config map[string]string) (Connector, error)
	// UpdateConnectorConfig replaces a connector's configuration. The supplied
	// config is sent unmasked; masked placeholders must be resolved by the caller.
	UpdateConnectorConfig(connect, name string, config map[string]string) (Connector, error)
	// DeleteConnector deletes a connector. Unknown connector yields a
	// ConnectorNotFoundError.
	DeleteConnector(connect, name string) error
	// PauseConnector pauses a connector and its tasks.
	PauseConnector(connect, name string) error
	// ResumeConnector resumes a paused/stopped connector.
	ResumeConnector(connect, name string) error
	// StopConnector stops a connector (STOPPED state), a prerequisite for offset
	// reset.
	StopConnector(connect, name string) error
	// RestartConnector restarts the connector instance (not its tasks).
	RestartConnector(connect, name string) error
	// RestartConnectorTask restarts a single task of a connector by task id.
	RestartConnectorTask(connect, name string, taskID int) error
	// ResetConnectorOffsets resets a connector's offsets. The connector must be in
	// the STOPPED state; otherwise a ConnectorNotStoppedError is returned without
	// calling the API. Unknown connector yields a ConnectorNotFoundError.
	ResetConnectorOffsets(connect, name string) error
	// GetConnectorPlugins lists the connector plugins installed on a Connect cluster.
	GetConnectorPlugins(connect string) ([]ConnectorPlugin, error)
	// ValidateConnectorConfig validates a candidate configuration against a named
	// plugin class and returns the per-field validation outcome.
	ValidateConnectorConfig(connect, pluginClass string, config map[string]string) (ConnectorValidationResult, error)

	// --- ksqlDB (KS-5, KS-8) ---

	// ListKsqlStreams lists the ksqlDB streams for the active cluster (posts LIST
	// STREAMS; to /ksql). Not-configured yields a KsqlNotConfiguredError.
	ListKsqlStreams() ([]KsqlStream, error)
	// ListKsqlTables lists the ksqlDB tables for the active cluster (posts LIST
	// TABLES; to /ksql). Not-configured yields a KsqlNotConfiguredError.
	ListKsqlTables() ([]KsqlTable, error)
	// ExecuteKsql validates and executes a single ksqlDB statement, delivering all
	// outcomes (schema, data rows, statement responses, and errors) as
	// KsqlResultTables on the returned channel. Validation runs first and routing
	// (SELECT -> streaming query; everything else -> statement) is internal.
	// SELECT queries stream a schema table followed by one table per data row until
	// the query ends or ctx is cancelled; cancelling ctx terminates the server-side
	// query and closes the channel. A KsqlNotConfiguredError is returned (with a
	// nil channel) when no endpoint is configured.
	ExecuteKsql(ctx context.Context, sql string, props map[string]string) (<-chan KsqlResultTable, error)
}
