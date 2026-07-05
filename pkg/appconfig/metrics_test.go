package appconfig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseMetricsSettings(t *testing.T) {
	t.Run("defaults when absent", func(t *testing.T) {
		s := ParseMetricsSettings(nil)
		assert.True(t, s.Enabled)
		assert.Equal(t, DefaultMetricsPollInterval, s.PollInterval)
		assert.Empty(t, s.Endpoint)
	})

	t.Run("parses all keys", func(t *testing.T) {
		s := ParseMetricsSettings(map[string]string{
			"enabled":      "false",
			"pollInterval": "15s",
			"endpoint":     "http://exporter:9404/metrics",
		})
		assert.False(t, s.Enabled)
		assert.Equal(t, 15*time.Second, s.PollInterval)
		assert.Equal(t, "http://exporter:9404/metrics", s.Endpoint)
	})

	t.Run("case-insensitive keys and aliases", func(t *testing.T) {
		s := ParseMetricsSettings(map[string]string{
			"Enable":   "true",
			"interval": "30s",
			"url":      "http://x/metrics",
		})
		assert.True(t, s.Enabled)
		assert.Equal(t, 30*time.Second, s.PollInterval)
		assert.Equal(t, "http://x/metrics", s.Endpoint)
	})

	t.Run("invalid values fall back to defaults", func(t *testing.T) {
		s := ParseMetricsSettings(map[string]string{"enabled": "maybe", "pollInterval": "soon"})
		assert.True(t, s.Enabled)
		assert.Equal(t, DefaultMetricsPollInterval, s.PollInterval)
	})

	t.Run("via ClusterExtension", func(t *testing.T) {
		ext := ClusterExtension{Metrics: map[string]string{"pollInterval": "3s"}}
		assert.Equal(t, 3*time.Second, ext.MetricsSettings().PollInterval)
	})
}
