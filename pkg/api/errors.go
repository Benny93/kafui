package api

import (
	"fmt"
	"slices"
	"strings"
)

// ConnectionError represents a connection-related error
type ConnectionError struct {
	Message string
	Cause   error
}

func (e ConnectionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("connection error: %s (caused by: %v)", e.Message, e.Cause)
	}
	return fmt.Sprintf("connection error: %s", e.Message)
}

func (e ConnectionError) Unwrap() error {
	return e.Cause
}

// NewConnectionError creates a new connection error
func NewConnectionError(message string) ConnectionError {
	return ConnectionError{Message: message}
}

// NewConnectionErrorWithCause creates a new connection error with a cause
func NewConnectionErrorWithCause(message string, cause error) ConnectionError {
	return ConnectionError{Message: message, Cause: cause}
}

// TimeoutError represents a timeout-related error
type TimeoutError struct {
	Message string
	Timeout string
}

func (e TimeoutError) Error() string {
	return fmt.Sprintf("timeout error: %s (timeout: %s)", e.Message, e.Timeout)
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(message, timeout string) TimeoutError {
	return TimeoutError{Message: message, Timeout: timeout}
}

// AuthenticationError represents an authentication-related error
type AuthenticationError struct {
	Message string
	Method  string
}

func (e AuthenticationError) Error() string {
	return fmt.Sprintf("authentication error: %s (method: %s)", e.Message, e.Method)
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(message, method string) AuthenticationError {
	return AuthenticationError{Message: message, Method: method}
}

// AuthorizationError represents an authorization-related error
type AuthorizationError struct {
	Message  string
	Resource string
	Action   string
}

func (e AuthorizationError) Error() string {
	return fmt.Sprintf("authorization error: %s (resource: %s, action: %s)", e.Message, e.Resource, e.Action)
}

// NewAuthorizationError creates a new authorization error
func NewAuthorizationError(message, resource, action string) AuthorizationError {
	return AuthorizationError{Message: message, Resource: resource, Action: action}
}

// TopicError represents a topic-related error
type TopicError struct {
	Message   string
	TopicName string
}

func (e TopicError) Error() string {
	return fmt.Sprintf("topic error: %s (topic: %s)", e.Message, e.TopicName)
}

// NewTopicError creates a new topic error
func NewTopicError(message, topicName string) TopicError {
	return TopicError{Message: message, TopicName: topicName}
}

// PartitionError represents a partition-related error
type PartitionError struct {
	Message     string
	TopicName   string
	PartitionID int32
}

func (e PartitionError) Error() string {
	return fmt.Sprintf("partition error: %s (topic: %s, partition: %d)", e.Message, e.TopicName, e.PartitionID)
}

// NewPartitionError creates a new partition error
func NewPartitionError(message, topicName string, partitionID int32) PartitionError {
	return PartitionError{Message: message, TopicName: topicName, PartitionID: partitionID}
}

// ClusterNotFoundError is returned when a cluster/context name is unknown.
type ClusterNotFoundError struct {
	Name      string
	Available []string // known cluster names, when the caller has them handy
	Cause     error
}

func (e ClusterNotFoundError) Error() string {
	if len(e.Available) > 0 {
		return fmt.Sprintf("cluster %q not found in config (available: %s)", e.Name, strings.Join(e.Available, ", "))
	}
	return fmt.Sprintf("cluster not found: %s", e.Name)
}

func (e ClusterNotFoundError) Unwrap() error { return e.Cause }

// ValidateClusterOverride checks that override (if non-empty) names a real
// configured context on ds, so callers can fail fast on a typo'd
// --cluster/-c instead of silently falling back to a default cluster.
func ValidateClusterOverride(ds KafkaDataSource, override string) error {
	if override == "" {
		return nil
	}
	contexts, err := ds.GetContexts()
	if err != nil {
		return nil // can't validate; let the normal connection error surface instead
	}
	if slices.Contains(contexts, override) {
		return nil
	}
	return ClusterNotFoundError{Name: override, Available: contexts}
}

// ClusterReadOnlyError is returned when a mutating operation is attempted on a
// cluster configured as read-only.
type ClusterReadOnlyError struct {
	Cluster   string
	Operation string
	Cause     error
}

func (e ClusterReadOnlyError) Error() string {
	if e.Operation != "" {
		return fmt.Sprintf("cluster %q is read-only: %s is not permitted", e.Cluster, e.Operation)
	}
	return fmt.Sprintf("cluster %q is read-only", e.Cluster)
}

func (e ClusterReadOnlyError) Unwrap() error { return e.Cause }

// AccessDeniedError is returned by the local authorization Gate when the active
// permission profile does not grant the attempted action on a resource. It is a
// self-imposed guardrail, not broker-side enforcement.
type AccessDeniedError struct {
	Resource string
	Name     string // resource name, empty for unnamed/create checks
	Action   string
	Cause    error
}

func (e AccessDeniedError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("access denied: %s on %s %q", e.Action, e.Resource, e.Name)
	}
	return fmt.Sprintf("access denied: %s on %s", e.Action, e.Resource)
}

func (e AccessDeniedError) Unwrap() error { return e.Cause }

// NotSupportedError is returned by datasource stubs for capabilities not yet
// implemented by that backend.
type NotSupportedError struct {
	Operation string
}

func (e NotSupportedError) Error() string {
	return fmt.Sprintf("operation not supported: %s", e.Operation)
}

// BrokerNotFoundError is returned when a broker ID is unknown to the cluster.
type BrokerNotFoundError struct {
	BrokerID int32
	Cause    error
}

func (e BrokerNotFoundError) Error() string {
	return fmt.Sprintf("broker not found: %d", e.BrokerID)
}

func (e BrokerNotFoundError) Unwrap() error { return e.Cause }

// LogDirNotFoundError is returned when a log directory path is unknown.
type LogDirNotFoundError struct {
	Path  string
	Cause error
}

func (e LogDirNotFoundError) Error() string {
	return fmt.Sprintf("log directory not found: %s", e.Path)
}

func (e LogDirNotFoundError) Unwrap() error { return e.Cause }

// InvalidConfigError is returned when the cluster rejects a config change. It
// carries the cluster's own rejection message in Reason.
type InvalidConfigError struct {
	Key    string
	Reason string
	Cause  error
}

func (e InvalidConfigError) Error() string {
	return fmt.Sprintf("invalid config %q: %s", e.Key, e.Reason)
}

func (e InvalidConfigError) Unwrap() error { return e.Cause }

// GroupNotFoundError is returned when a consumer group id is unknown to the cluster.
type GroupNotFoundError struct {
	GroupID string
	Cause   error
}

func (e GroupNotFoundError) Error() string {
	return fmt.Sprintf("consumer group not found: %s", e.GroupID)
}

func (e GroupNotFoundError) Unwrap() error { return e.Cause }

// GroupNotEmptyError is returned when a mutating operation (e.g. offset reset)
// requires an inactive group but the group still has active members. It carries
// the group's current state.
type GroupNotEmptyError struct {
	GroupID string
	State   string
	Cause   error
}

func (e GroupNotEmptyError) Error() string {
	return fmt.Sprintf("consumer group %q is not empty (state: %s)", e.GroupID, e.State)
}

func (e GroupNotEmptyError) Unwrap() error { return e.Cause }

// InvalidOffsetResetError is returned when an offset reset request fails
// validation (unknown mode, missing timestamp/offsets, etc.).
type InvalidOffsetResetError struct {
	Reason string
	Cause  error
}

func (e InvalidOffsetResetError) Error() string {
	return fmt.Sprintf("invalid offset reset: %s", e.Reason)
}

func (e InvalidOffsetResetError) Unwrap() error { return e.Cause }

// MetricsNotAvailableError is returned when per-broker metrics cannot be retrieved.
type MetricsNotAvailableError struct {
	BrokerID int32
	Cause    error
}

func (e MetricsNotAvailableError) Error() string {
	return fmt.Sprintf("metrics not available for broker %d", e.BrokerID)
}

func (e MetricsNotAvailableError) Unwrap() error { return e.Cause }

// MetricsNotConfiguredError is returned by accessors that require a configured
// metrics endpoint (e.g. byte-rate scraping / range graphs) when the active
// cluster has no metrics configuration. Offset-delta metrics remain available
// without configuration, so this is only used by the endpoint-dependent paths.
type MetricsNotConfiguredError struct {
	Cluster string
	Cause   error
}

func (e MetricsNotConfiguredError) Error() string {
	if e.Cluster == "" {
		return "metrics collection is not configured"
	}
	return fmt.Sprintf("metrics collection is not configured for cluster %q", e.Cluster)
}

func (e MetricsNotConfiguredError) Unwrap() error { return e.Cause }

// TopicNotFoundError is returned when a topic name is unknown to the cluster. (TP-3)
type TopicNotFoundError struct {
	TopicName string
	Cause     error
}

func (e TopicNotFoundError) Error() string {
	return fmt.Sprintf("topic not found: %s", e.TopicName)
}

func (e TopicNotFoundError) Unwrap() error { return e.Cause }

// TopicAlreadyExistsError is returned when creating a topic that already exists. (TP-5)
type TopicAlreadyExistsError struct {
	TopicName string
	Cause     error
}

func (e TopicAlreadyExistsError) Error() string {
	return fmt.Sprintf("topic already exists: %s", e.TopicName)
}

func (e TopicAlreadyExistsError) Unwrap() error { return e.Cause }

// TopicValidationError is returned when the broker rejects a topic create/alter
// request as invalid (bad name, invalid config, too-high replication factor). (TP-5)
type TopicValidationError struct {
	TopicName string
	Reason    string
	Cause     error
}

func (e TopicValidationError) Error() string {
	return fmt.Sprintf("topic %q rejected: %s", e.TopicName, e.Reason)
}

func (e TopicValidationError) Unwrap() error { return e.Cause }

// MetadataTimeoutError is returned when a created topic does not become visible
// in cluster metadata within the bounded retry window. (TP-5)
type MetadataTimeoutError struct {
	TopicName string
	Cause     error
}

func (e MetadataTimeoutError) Error() string {
	return fmt.Sprintf("topic %q not visible in metadata before timeout", e.TopicName)
}

func (e MetadataTimeoutError) Unwrap() error { return e.Cause }

// TopicDeletionDisabledError is returned when topic deletion is attempted on a
// cluster with delete.topic.enable=false. (TP-6)
type TopicDeletionDisabledError struct {
	TopicName string
}

func (e TopicDeletionDisabledError) Error() string {
	return "topic deletion is disabled on this cluster (delete.topic.enable=false)"
}

// PartitionDecreaseError is returned when a partition-count change would reduce
// the number of partitions, which Kafka does not permit. (TP-8)
type PartitionDecreaseError struct {
	TopicName string
	Current   int32
	Requested int32
}

func (e PartitionDecreaseError) Error() string {
	return fmt.Sprintf("cannot decrease partitions for %q from %d to %d: partition count can only be increased", e.TopicName, e.Current, e.Requested)
}

// PartitionNoopError is returned when the requested partition count equals the
// current count. (TP-8)
type PartitionNoopError struct {
	TopicName string
	Current   int32
}

func (e PartitionNoopError) Error() string {
	return fmt.Sprintf("topic %q already has %d partitions", e.TopicName, e.Current)
}

// CleanupPolicyError is returned when a message-purge is attempted on a topic
// whose cleanup.policy does not include "delete". (TP-9)
type CleanupPolicyError struct {
	TopicName string
	Policy    string
}

func (e CleanupPolicyError) Error() string {
	return fmt.Sprintf("cannot purge messages for %q: cleanup.policy is %q (must include \"delete\")", e.TopicName, e.Policy)
}

// RecreateTimeoutError is returned when a recreated topic's prior instance does
// not finish deleting before the bounded retry window expires. (TP-10)
type RecreateTimeoutError struct {
	TopicName string
	Cause     error
}

func (e RecreateTimeoutError) Error() string {
	return fmt.Sprintf("topic %q could not be recreated: prior deletion still propagating", e.TopicName)
}

func (e RecreateTimeoutError) Unwrap() error { return e.Cause }

// InvalidReplicationFactorError is returned when a requested replication factor
// fails validation (equal to current, < 1, or > available brokers). (TP-11)
type InvalidReplicationFactorError struct {
	TopicName string
	Reason    string
}

func (e InvalidReplicationFactorError) Error() string {
	return fmt.Sprintf("invalid replication factor for %q: %s", e.TopicName, e.Reason)
}

// --- ACL & quota errors (AQ-3, AQ-10) ---

// ACLValidationError is returned when an ACL binding fails validation before
// any broker call (bad principal, empty field, unrecognized enum value).
type ACLValidationError struct {
	Field  string
	Reason string
	Cause  error
}

func (e ACLValidationError) Error() string {
	return fmt.Sprintf("invalid ACL %s: %s", e.Field, e.Reason)
}

func (e ACLValidationError) Unwrap() error { return e.Cause }

// ACLNotFoundError is returned when a delete targets a binding that does not
// exist (the broker matched zero ACLs).
type ACLNotFoundError struct {
	Entry ACLEntry
	Cause error
}

func (e ACLNotFoundError) Error() string {
	return fmt.Sprintf("ACL binding not found: %s %s on %s:%s (%s) for %s",
		e.Entry.Permission, e.Entry.Operation, e.Entry.ResourceType, e.Entry.ResourceName, e.Entry.PatternType, e.Entry.Principal)
}

func (e ACLNotFoundError) Unwrap() error { return e.Cause }

// QuotaValidationError is returned when a client-quota request fails validation
// before any broker call (e.g. no entity identifier set).
type QuotaValidationError struct {
	Reason string
	Cause  error
}

func (e QuotaValidationError) Error() string {
	return fmt.Sprintf("invalid client quota: %s", e.Reason)
}

func (e QuotaValidationError) Unwrap() error { return e.Cause }

// InvalidSeekError is returned when ConsumeFlags carry an invalid seek model
// (unknown mode, or an offset/timestamp mode missing its value). (MSG-1)
type InvalidSeekError struct {
	Mode   string
	Reason string
	Cause  error
}

func (e InvalidSeekError) Error() string {
	return fmt.Sprintf("invalid seek %q: %s", e.Mode, e.Reason)
}

func (e InvalidSeekError) Unwrap() error { return e.Cause }

// ProduceError is returned when producing a record to a topic fails. (MSG-30)
type ProduceError struct {
	Topic  string
	Reason string
	Cause  error
}

func (e ProduceError) Error() string {
	msg := fmt.Sprintf("failed to produce to %q", e.Topic)
	if e.Reason != "" {
		msg += ": " + e.Reason
	}
	if e.Cause != nil {
		msg += fmt.Sprintf(" (%v)", e.Cause)
	}
	return msg
}

func (e ProduceError) Unwrap() error { return e.Cause }

// AnalysisAlreadyRunningError is returned when a topic analysis is started while
// one is already in progress for the same topic. (TP-29)
type AnalysisAlreadyRunningError struct {
	TopicName string
}

func (e AnalysisAlreadyRunningError) Error() string {
	return fmt.Sprintf("analysis already running for topic %q", e.TopicName)
}

// --- Schema registry errors (SR-2) ---

// SchemaRegistryNotConfiguredError is returned when a schema-registry operation
// is attempted on a cluster that has no schema registry configured. Listing
// calls may return empty instead, but content and mutation calls must error.
type SchemaRegistryNotConfiguredError struct {
	Cause error
}

func (e SchemaRegistryNotConfiguredError) Error() string {
	return "no schema registry configured for this cluster"
}

func (e SchemaRegistryNotConfiguredError) Unwrap() error { return e.Cause }

// SubjectNotFoundError is returned when a subject is unknown to the registry.
type SubjectNotFoundError struct {
	Subject string
	Cause   error
}

func (e SubjectNotFoundError) Error() string {
	return fmt.Sprintf("schema subject not found: %s", e.Subject)
}

func (e SubjectNotFoundError) Unwrap() error { return e.Cause }

// SchemaVersionNotFoundError is returned when a subject exists but the requested
// version does not.
type SchemaVersionNotFoundError struct {
	Subject string
	Version int
	Cause   error
}

func (e SchemaVersionNotFoundError) Error() string {
	return fmt.Sprintf("schema version %d not found for subject %s", e.Version, e.Subject)
}

func (e SchemaVersionNotFoundError) Unwrap() error { return e.Cause }

// SchemaIncompatibleError is returned when a candidate schema is incompatible
// with the subject's existing versions (registry HTTP 409). It carries the
// registry's own explanation in Message.
type SchemaIncompatibleError struct {
	Subject string
	Message string
	Cause   error
}

func (e SchemaIncompatibleError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("schema incompatible with subject %s: %s", e.Subject, e.Message)
	}
	return fmt.Sprintf("schema incompatible with subject %s", e.Subject)
}

func (e SchemaIncompatibleError) Unwrap() error { return e.Cause }

// SchemaValidationError is returned when the registry rejects a schema as
// invalid (registry HTTP 422). It carries the registry's message.
type SchemaValidationError struct {
	Message string
	Cause   error
}

func (e SchemaValidationError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("invalid schema: %s", e.Message)
	}
	return "invalid schema"
}

func (e SchemaValidationError) Unwrap() error { return e.Cause }

// --- Kafka Connect errors (KC-2) ---

// ConnectClusterNotFoundError is returned when an operation references a Connect
// cluster name that is not configured for the active Kafka cluster.
type ConnectClusterNotFoundError struct {
	Connect string // the Connect cluster name that was not found
	Cluster string // the active Kafka cluster/context it was looked up in
	Cause   error
}

func (e ConnectClusterNotFoundError) Error() string {
	if e.Cluster != "" {
		return fmt.Sprintf("connect cluster %q not configured for kafka cluster %q", e.Connect, e.Cluster)
	}
	return fmt.Sprintf("connect cluster %q not configured", e.Connect)
}

func (e ConnectClusterNotFoundError) Unwrap() error { return e.Cause }

// ConnectorNotFoundError is returned when a connector name is unknown to the
// Connect cluster.
type ConnectorNotFoundError struct {
	Connector string
	Connect   string
	Cause     error
}

func (e ConnectorNotFoundError) Error() string {
	return fmt.Sprintf("connector %q not found on connect cluster %q", e.Connector, e.Connect)
}

func (e ConnectorNotFoundError) Unwrap() error { return e.Cause }

// ConnectorAlreadyExistsError is returned when creating a connector whose name
// already exists on the Connect cluster.
type ConnectorAlreadyExistsError struct {
	Connector string
	Connect   string
	Cause     error
}

func (e ConnectorAlreadyExistsError) Error() string {
	return fmt.Sprintf("connector %q already exists on connect cluster %q", e.Connector, e.Connect)
}

func (e ConnectorAlreadyExistsError) Unwrap() error { return e.Cause }

// ConnectorNotStoppedError is returned when an operation requiring a STOPPED
// connector (e.g. resetting offsets) is attempted on a connector in another
// state. It carries the connector's current state.
type ConnectorNotStoppedError struct {
	Connector string
	Connect   string
	State     string
	Cause     error
}

func (e ConnectorNotStoppedError) Error() string {
	return fmt.Sprintf("connector %q on connect cluster %q is not stopped (state: %s)", e.Connector, e.Connect, e.State)
}

func (e ConnectorNotStoppedError) Unwrap() error { return e.Cause }

// --- ksqlDB errors (KS-3) ---

// KsqlNotConfiguredError is returned when a ksqlDB operation is attempted on a
// cluster that has no ksqlDB endpoint configured.
type KsqlNotConfiguredError struct {
	Cause error
}

func (e KsqlNotConfiguredError) Error() string {
	return "no ksqlDB endpoint configured for this cluster"
}

func (e KsqlNotConfiguredError) Unwrap() error { return e.Cause }

// KsqlNoInstancesError is returned when none of the configured ksqlDB endpoints
// could be reached (connection-level failure on every URL). Configured is the
// number of endpoints that were tried.
type KsqlNoInstancesError struct {
	Configured int
	Cause      error
}

func (e KsqlNoInstancesError) Error() string {
	return "no live ksqlDB instances are available"
}

func (e KsqlNoInstancesError) Unwrap() error { return e.Cause }

// KsqlServerError carries a ksqlDB REST non-2xx response. ksqlDB returns
// {"error_code": ..., "message": ...} bodies; ErrorCode is 0 when unparseable
// and Raw holds the verbatim response body.
type KsqlServerError struct {
	StatusCode int
	ErrorCode  int
	Message    string
	Raw        string
	Cause      error
}

func (e KsqlServerError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("ksqlDB returned %d: %s", e.StatusCode, e.Message)
	}
	if e.Raw != "" {
		return fmt.Sprintf("ksqlDB returned %d: %s", e.StatusCode, e.Raw)
	}
	return fmt.Sprintf("ksqlDB returned %d", e.StatusCode)
}

func (e KsqlServerError) Unwrap() error { return e.Cause }
