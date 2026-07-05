package appconfig

import (
	"strconv"
	"strings"
	"time"
)

// DefaultMetricsPollInterval is the collector cadence used when a cluster's
// metrics config does not specify one. It is short so the metrics page shows
// live offset-delta rates quickly.
const DefaultMetricsPollInterval = 5 * time.Second

// MetricsSettings is the typed view of a cluster's optional metrics
// configuration, parsed from the free-form ClusterExtension.Metrics map (kept
// as a map for forward-compatibility with the application-config schema).
//
// Offset-delta metrics (message-in rates) are always available regardless of
// this config. Endpoint is only needed for byte-rate scraping / range graphs,
// which are a documented stub in this build.
type MetricsSettings struct {
	// Enabled reports whether metrics collection is switched on. It defaults to
	// true: offset-delta metrics cost little and are always useful.
	Enabled bool
	// PollInterval is the background collection cadence (0 ⇒ collector default).
	PollInterval time.Duration
	// Endpoint is an optional Prometheus/JMX-exporter metrics URL. Empty means
	// offset-delta-only collection (byte rates reported as unknown).
	Endpoint string
	// Type selects the collection mechanism: "PROMETHEUS" (default) scrapes the
	// exposition endpoint; "JMX" is honored only through a Jolokia HTTP bridge
	// (JolokiaURL) and otherwise degrades to a warning and empty broker metrics
	// (MM-17). Stored upper-cased.
	Type string
	// JolokiaURL is the optional Jolokia HTTP-bridge base URL used when Type is
	// "JMX" (MM-17). Empty ⇒ JMX degrades gracefully.
	JolokiaURL string
	// Username/Password are optional basic-auth credentials for scraping and the
	// Jolokia bridge.
	Username string
	Password string
	// TimeSeriesURLs is the optional list of Prometheus-compatible query API base
	// URLs used for range/instant graphs (MM-14/MM-15). Empty ⇒ no graph backend.
	TimeSeriesURLs []string
	// TLSCAPath is an optional custom CA PEM path for the query/scrape HTTPS
	// clients.
	TLSCAPath string
	// ExpositionEnabled opts a cluster in to the flag-gated Prometheus exposition
	// endpoint (MM-16). Defaults to true so an unset value still exports.
	ExpositionEnabled bool
}

// MetricsTypeJMX is the config value selecting JMX-over-Jolokia collection.
const MetricsTypeJMX = "JMX"

// MetricsTypePrometheus is the default collection mechanism.
const MetricsTypePrometheus = "PROMETHEUS"

// ParseMetricsSettings turns the free-form metrics map into typed settings,
// applying defaults for absent keys. Recognized keys (case-insensitive):
//
//	enable/enabled       bool   (default true)
//	pollInterval/interval duration string, e.g. "10s" (default DefaultMetricsPollInterval)
//	endpoint/url         string (default "")
func ParseMetricsSettings(m map[string]string) MetricsSettings {
	s := MetricsSettings{Enabled: true, PollInterval: DefaultMetricsPollInterval, ExpositionEnabled: true}
	get := func(keys ...string) (string, bool) {
		for k, v := range m {
			for _, want := range keys {
				if strings.EqualFold(strings.TrimSpace(k), want) {
					return strings.TrimSpace(v), true
				}
			}
		}
		return "", false
	}
	if v, ok := get("enable", "enabled"); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			s.Enabled = b
		}
	}
	if v, ok := get("pollInterval", "interval"); ok {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			s.PollInterval = d
		}
	}
	if v, ok := get("endpoint", "url"); ok {
		s.Endpoint = v
	}
	if v, ok := get("type"); ok && v != "" {
		s.Type = strings.ToUpper(v)
	}
	if v, ok := get("jolokiaUrl", "jolokiaURL", "bridgeUrl"); ok {
		s.JolokiaURL = v
	}
	if v, ok := get("username", "user"); ok {
		s.Username = v
	}
	if v, ok := get("password", "pass"); ok {
		s.Password = v
	}
	if v, ok := get("tlsCaPath", "caPath"); ok {
		s.TLSCAPath = v
	}
	if v, ok := get("timeSeriesUrls", "timeSeriesURLs", "queryUrls", "queryURLs"); ok {
		for _, u := range strings.Split(v, ",") {
			if u = strings.TrimRight(strings.TrimSpace(u), "/"); u != "" {
				s.TimeSeriesURLs = append(s.TimeSeriesURLs, u)
			}
		}
	}
	if v, ok := get("exposition", "expositionEnabled"); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			s.ExpositionEnabled = b
		}
	}
	return s
}

// MetricsSettings returns the typed metrics settings for this cluster extension.
func (e ClusterExtension) MetricsSettings() MetricsSettings {
	return ParseMetricsSettings(e.Metrics)
}
