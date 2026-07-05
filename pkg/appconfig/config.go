// Package appconfig owns the kafui-managed configuration document.
//
// kafui deliberately never rewrites ~/.kaf/config (the kaf library's YAML
// round-trip strips TLS cert paths). This package is kafui's own home for
// settings the kaf schema cannot express — read-only flags, optional-integration
// endpoints, UI preferences, redaction rules — layered over the read-only kaf file.
//
// Precedence (highest wins): CLI flags > kafui file > kaf file > defaults.
package appconfig

import (
	"fmt"
	"time"

	"github.com/Benny93/kafui/pkg/serde"
)

// Config is the kafui-owned configuration document (loaded from config.yaml).
type Config struct {
	// DynamicConfigEnabled gates in-UI cluster editing (the setup wizard).
	DynamicConfigEnabled bool `yaml:"dynamicConfigEnabled"`

	// UI preferences persisted across runs.
	UI UISettings `yaml:"ui"`

	// ReleaseCheck controls the optional latest-release check.
	ReleaseCheck ReleaseCheckSettings `yaml:"releaseCheck"`

	// AutoReload controls hot-reloading of this config file while kafui runs.
	AutoReload AutoReloadSettings `yaml:"autoReload"`

	// Redaction controls secret masking in displayed configuration.
	Redaction RedactionSettings `yaml:"redaction"`

	// RefreshInterval is the background statistics collection cadence.
	RefreshInterval time.Duration `yaml:"refreshInterval"`

	// Clusters holds per-cluster extension entries keyed by cluster name.
	Clusters map[string]ClusterExtension `yaml:"clusters"`

	// Authz holds the local permission-profile configuration (AA-2). A missing
	// section (no profiles and no default) leaves authorization disabled: every
	// operation is allowed. Read-only mode is independent of this section.
	Authz AuthzSettings `yaml:"authz"`

	// Audit configures the local JSONL audit log (AA-6). Disabled by default.
	Audit AuditSettings `yaml:"audit"`
}

// AuthzSettings is the permission-profile configuration. Profiles collapse the
// spec's identity-bound roles to named profiles selected per cluster, since a
// local kafui has exactly one operator.
type AuthzSettings struct {
	// ActiveProfile, when set, forces this profile as the active one regardless
	// of cluster membership (an explicit override).
	ActiveProfile string `yaml:"activeProfile"`

	// Profiles are the named permission profiles.
	Profiles []Profile `yaml:"profiles"`

	// Default is the fallback profile (permissions only, all clusters) applied
	// when no named profile covers the active cluster.
	Default *Profile `yaml:"default"`
}

// Enabled reports whether any profile or a default profile is configured. When
// disabled the Gate allows everything.
func (a AuthzSettings) Enabled() bool {
	return len(a.Profiles) > 0 || a.Default != nil
}

// Profile is a named set of permissions scoped to one or more clusters.
type Profile struct {
	Name        string       `yaml:"name"`
	Clusters    []string     `yaml:"clusters"`
	Permissions []Permission `yaml:"permissions"`
}

// Permission grants a set of actions on a resource type, optionally narrowed to
// resource names matching Name (a regex, full-match). An empty Name matches any
// resource of the type.
type Permission struct {
	Resource string   `yaml:"resource"`
	Name     string   `yaml:"name"`
	Actions  []string `yaml:"actions"`
}

// AuditSettings configures the local audit log.
type AuditSettings struct {
	// Enabled turns auditing on. Default false ⇒ no records written.
	Enabled bool `yaml:"enabled"`
	// Level is "alter_only" (default) or "all". alter_only skips read-only ops.
	Level string `yaml:"level"`
	// Path overrides the audit log location (default ~/.kafui/audit.log).
	Path string `yaml:"path"`
}

// UISettings are persisted UI preferences.
type UISettings struct {
	Theme       string `yaml:"theme"` // "auto", "dark", "light"
	ShowSidebar bool   `yaml:"showSidebar"`
	CompactMode bool   `yaml:"compactMode"`
	Timezone    string `yaml:"timezone"` // "local", "UTC", or an IANA name
}

// AutoReloadSettings control config-file hot-reloading (AC-16). Disabled by
// default. When enabled, kafui polls the config file's mtime every Interval and
// hot-applies reloadable settings (UI prefs, cluster extensions) — it never
// auto-reconnects the active cluster, only surfaces a notice.
type AutoReloadSettings struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
}

// ReleaseCheckSettings control the GitHub latest-release check.
type ReleaseCheckSettings struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
}

// RedactionSettings control secret masking.
type RedactionSettings struct {
	Enabled  bool     `yaml:"enabled"`
	Patterns []string `yaml:"patterns"` // when non-empty, replaces the default pattern list
}

// SASLConfig is the broker SASL authentication for a fully-kafui-defined cluster.
// Mechanism is one of PLAIN, SCRAM-SHA-256, SCRAM-SHA-512, OAUTHBEARER.
type SASLConfig struct {
	Mechanism    string `yaml:"mechanism"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	ClientID     string `yaml:"clientID"`
	ClientSecret string `yaml:"clientSecret"`
	TokenURL     string `yaml:"tokenURL"`
	// DeviceAuthURL is the OAuth2 device-authorization endpoint. When set (with a
	// ClientID and TokenURL but no ClientSecret or static Token), kafui runs the
	// interactive device-code grant for OAUTHBEARER (AA-13).
	DeviceAuthURL string `yaml:"deviceAuthURL"`
}

// TLSConfig is the broker TLS material (PEM file paths) for a fully-kafui-defined cluster.
type TLSConfig struct {
	CAPath   string `yaml:"caPath"`
	CertPath string `yaml:"certPath"`
	KeyPath  string `yaml:"keyPath"`
	Insecure bool   `yaml:"insecure"`
}

// ClusterExtension carries per-cluster fields the kaf schema lacks.
//
// When Brokers is non-empty the entry is a *fully kafui-defined* cluster
// (connection details live entirely in the kafui file, since ~/.kaf/config is
// read-only). When Brokers is empty the entry is an overlay attaching only the
// extra fields to a cluster defined in the kaf file. See IsFullyDefined.
type ClusterExtension struct {
	ReadOnly bool `yaml:"readOnly"`

	// --- Fully-kafui-defined cluster connection (AC-13). Empty ⇒ overlay only. ---
	Brokers                []string    `yaml:"brokers,omitempty"`
	KafkaVersion           string      `yaml:"kafkaVersion,omitempty"`
	SecurityProtocol       string      `yaml:"securityProtocol,omitempty"`
	SASL                   *SASLConfig `yaml:"sasl,omitempty"`
	TLS                    *TLSConfig  `yaml:"tls,omitempty"`
	SchemaRegistryURL      string      `yaml:"schemaRegistryUrl,omitempty"`
	SchemaRegistryUsername string      `yaml:"schemaRegistryUsername,omitempty"`
	SchemaRegistryPassword string      `yaml:"schemaRegistryPassword,omitempty"`

	// PollingThrottle bounds background collection rate for this cluster.
	PollingThrottle time.Duration `yaml:"pollingThrottle"`

	// Connect lists Kafka Connect clusters for this Kafka cluster.
	Connect []ConnectCluster `yaml:"connect"`

	// Ksql is the ksqlDB endpoint config (optional).
	Ksql *KsqlEndpoint `yaml:"ksql"`

	// Metrics holds metrics-store settings (optional).
	Metrics map[string]string `yaml:"metrics"`

	// Masking holds data-masking rules for message payloads.
	Masking []string `yaml:"masking"`

	// Serdes holds per-cluster serde bindings (topic-name pattern → key/value
	// serde overrides). Unbound built-in serdes remain selectable regardless.
	Serdes []serde.SerdeConfig `yaml:"serdes"`

	// Properties are free-form custom client properties (dot-flattened on load).
	Properties         map[string]any `yaml:"properties"`
	ConsumerProperties map[string]any `yaml:"consumerProperties"`
	ProducerProperties map[string]any `yaml:"producerProperties"`
}

// IsFullyDefined reports whether this entry defines a cluster's connection
// entirely in the kafui file (as opposed to overlaying a kaf-file cluster).
func (e ClusterExtension) IsFullyDefined() bool {
	return len(e.Brokers) > 0
}

// ConnectCluster describes one Kafka Connect cluster.
type ConnectCluster struct {
	Name                string `yaml:"name"`
	Address             string `yaml:"address"`
	Username            string `yaml:"username"`
	Password            string `yaml:"password"`
	TLSCAPath           string `yaml:"tlsCaPath"`
	TLSCertPath         string `yaml:"tlsCertPath"`
	TLSKeyPath          string `yaml:"tlsKeyPath"`
	ConsumerNamePattern string `yaml:"consumerNamePattern"`
}

// KsqlEndpoint describes a ksqlDB endpoint. URL may be a comma-separated list of
// endpoints for connection-level failover. MaxResponseBytes caps the response
// read size (0 ⇒ the client's 20 MB default).
type KsqlEndpoint struct {
	URL              string `yaml:"url"`
	Username         string `yaml:"username"`
	Password         string `yaml:"password"`
	TLSCAPath        string `yaml:"tlsCaPath"`
	TLSCertPath      string `yaml:"tlsCertPath"`
	TLSKeyPath       string `yaml:"tlsKeyPath"`
	MaxResponseBytes int64  `yaml:"maxResponseBytes"`
}

// String renders the endpoint with the password redacted so it is safe to log.
func (k KsqlEndpoint) String() string {
	pw := ""
	if k.Password != "" {
		pw = "********"
	}
	return fmt.Sprintf("KsqlEndpoint{URL:%q Username:%q Password:%q MaxResponseBytes:%d}", k.URL, k.Username, pw, k.MaxResponseBytes)
}

// Default returns a Config populated with sensible defaults (used when no file exists).
func Default() Config {
	return Config{
		DynamicConfigEnabled: false,
		UI: UISettings{
			Theme:       "auto",
			ShowSidebar: true,
			CompactMode: false,
			Timezone:    "local",
		},
		ReleaseCheck: ReleaseCheckSettings{
			Enabled:  true,
			Interval: time.Hour,
			Timeout:  5 * time.Second,
		},
		AutoReload: AutoReloadSettings{
			Enabled:  false,
			Interval: 3 * time.Second,
		},
		Redaction:       RedactionSettings{Enabled: true},
		RefreshInterval: 30 * time.Second,
		Clusters:        map[string]ClusterExtension{},
		Audit:           AuditSettings{Enabled: false, Level: "alter_only"},
	}
}
