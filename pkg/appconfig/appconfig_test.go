package appconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "auto", cfg.UI.Theme)
	assert.True(t, cfg.ReleaseCheck.Enabled)
	assert.NotNil(t, cfg.Clusters)
}

func TestLoadMalformedFileErrors(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.yaml")
	require.NoError(t, os.WriteFile(p, []byte("::: not yaml :::\n\t- x"), 0o600))
	_, err := Load(p)
	assert.Error(t, err)
}

func TestLoadOverridesDefaults(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(p, []byte("dynamicConfigEnabled: true\nui:\n  theme: light\nrefreshInterval: 45s\n"), 0o600))
	cfg, err := Load(p)
	require.NoError(t, err)
	assert.True(t, cfg.DynamicConfigEnabled)
	assert.Equal(t, "light", cfg.UI.Theme)
	assert.Equal(t, "45s", cfg.RefreshInterval.String())
}

func TestFlatten(t *testing.T) {
	in := map[string]any{
		"a": map[string]any{"b": map[string]any{"c": "v"}},
		"x": 5,
	}
	out := Flatten(in)
	assert.Equal(t, "v", out["a.b.c"])
	assert.Equal(t, "5", out["x"])
}

func TestValidate(t *testing.T) {
	t.Run("single unnamed gets default", func(t *testing.T) {
		clusters := []ClusterConfig{{Name: "", Brokers: []string{"b:9092"}}}
		require.NoError(t, Validate(clusters))
		assert.Equal(t, "default", clusters[0].Name)
	})
	t.Run("duplicate names rejected", func(t *testing.T) {
		err := Validate([]ClusterConfig{
			{Name: "a", Brokers: []string{"b:9092"}},
			{Name: "a", Brokers: []string{"b:9092"}},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})
	t.Run("missing brokers rejected", func(t *testing.T) {
		err := Validate([]ClusterConfig{{Name: "a"}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "broker")
	})
}

func TestRedact(t *testing.T) {
	r := NewRedactor(RedactionSettings{Enabled: true})
	assert.Equal(t, redactPlaceholder, r.Redact("password", "hunter2"))
	assert.Equal(t, redactPlaceholder, r.Redact("SASL.Password", "x"))
	assert.Equal(t, redactPlaceholder, r.Redact("ssl.keystore.password", "x"))
	assert.Equal(t, "prod-kafka:9092", r.Redact("brokers", "prod-kafka:9092"))
	// externalized reference passes through
	assert.Equal(t, "${env:MY_SECRET}", r.Redact("password", "${env:MY_SECRET}"))

	t.Run("disabled", func(t *testing.T) {
		r := NewRedactor(RedactionSettings{Enabled: false})
		assert.Equal(t, "hunter2", r.Redact("password", "hunter2"))
	})
	t.Run("custom patterns replace defaults", func(t *testing.T) {
		r := NewRedactor(RedactionSettings{Enabled: true, Patterns: []string{"custom"}})
		assert.Equal(t, redactPlaceholder, r.Redact("my.custom.thing", "v"))
		assert.Equal(t, "v", r.Redact("password", "v")) // default no longer applies
	})
}
