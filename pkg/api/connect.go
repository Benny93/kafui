package api

import "strings"

// ConnectorType distinguishes source connectors (into Kafka) from sink
// connectors (out of Kafka).
type ConnectorType string

const (
	ConnectorTypeSource  ConnectorType = "source"
	ConnectorTypeSink    ConnectorType = "sink"
	ConnectorTypeUnknown ConnectorType = "unknown"
)

// Connector/task lifecycle states as reported by the Connect REST API.
const (
	ConnectorStateRunning    = "RUNNING"
	ConnectorStateFailed     = "FAILED"
	ConnectorStatePaused     = "PAUSED"
	ConnectorStateStopped    = "STOPPED"
	ConnectorStateUnassigned = "UNASSIGNED"
	ConnectorStateRestarting = "RESTARTING"
	ConnectorStateDestroyed  = "DESTROYED"
)

// ConnectCluster describes one Kafka Connect cluster together with optional
// aggregated statistics. Stats fields are populated only when requested
// (GetConnectClusters(withStats=true)); Reachable is false when the cluster's
// root endpoint could not be contacted, in which case the runtime fields
// (Version, Commit, KafkaClusterID) are empty.
type ConnectCluster struct {
	Name           string
	Address        string
	Version        string
	Commit         string
	KafkaClusterID string
	Reachable      bool

	// Aggregated statistics (only set when withStats=true).
	ConnectorCount       int
	FailedConnectorCount int
	TaskCount            int
	FailedTaskCount      int
}

// Connector is one connector as shown in the aggregated listing. Trace carries
// the connector-level error trace when the connector is FAILED.
type Connector struct {
	ConnectCluster  string
	Name            string
	Class           string
	Type            ConnectorType
	Topics          []string
	State           string
	WorkerID        string
	Trace           string
	TaskCount       int
	FailedTaskCount int
	ConsumerGroup   string // derived for sink connectors from the configured pattern
}

// ConnectorTask is a single connector task with its runtime status.
type ConnectorTask struct {
	ID       int
	WorkerID string
	State    string // RUNNING | FAILED | PAUSED | RESTARTING | UNASSIGNED
	Trace    string
}

// ConnectorPlugin describes an installed connector plugin.
type ConnectorPlugin struct {
	Class   string
	Type    string // source | sink
	Version string
}

// ConnectorConfigKeyValidation is the validation outcome for a single config
// key, mirroring the Connect REST API's per-value block.
type ConnectorConfigKeyValidation struct {
	Name              string
	Value             string
	Errors            []string
	RecommendedValues []string
	Visible           bool
}

// ConnectorValidationResult is the outcome of validating a candidate config
// against a plugin. ErrorCount is the total number of per-field errors; Configs
// holds the per-field definitions with any error messages.
type ConnectorValidationResult struct {
	Name       string
	ErrorCount int
	Groups     []string
	Configs    []ConnectorConfigKeyValidation
}

// ConnectorDetails combines a connector's configuration, status, tasks and
// topics. Config values are masked (secret-like keys replaced) before return.
type ConnectorDetails struct {
	ConnectCluster string
	Name           string
	Class          string
	Type           ConnectorType
	Config         map[string]string
	State          string
	WorkerID       string
	Trace          string
	Tasks          []ConnectorTask
	Topics         []string
	ConsumerGroup  string
}

// connectorSecretPatterns are matched case-insensitively as substrings of a
// config key to decide whether its value should be masked.
var connectorSecretPatterns = []string{
	"password", "secret", "token", "key", "credential", "sasl.jaas.config",
}

// ConnectorSecretPlaceholder is substituted for masked secret values.
const ConnectorSecretPlaceholder = "********"

// MaskConnectorConfig returns a copy of config with secret-like values replaced
// by ConnectorSecretPlaceholder. Key matching is case-insensitive substring
// matching against a fixed set of patterns (password, secret, token, key,
// credential, sasl.jaas.config). Non-secret keys are left untouched. Empty
// values are left untouched. A nil config yields nil.
func MaskConnectorConfig(config map[string]string) map[string]string {
	if config == nil {
		return nil
	}
	out := make(map[string]string, len(config))
	for k, v := range config {
		if v != "" && isSecretConfigKey(k) {
			out[k] = ConnectorSecretPlaceholder
		} else {
			out[k] = v
		}
	}
	return out
}

func isSecretConfigKey(key string) bool {
	lk := strings.ToLower(key)
	for _, p := range connectorSecretPatterns {
		if strings.Contains(lk, p) {
			return true
		}
	}
	return false
}
