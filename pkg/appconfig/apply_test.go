package appconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func fullCluster(brokers ...string) ClusterExtension {
	return ClusterExtension{Brokers: brokers}
}

func TestApplyCluster_WritesKafuiFileWithContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	running := Default()

	merged, err := ApplyCluster(path, running, "", "prod", ClusterExtension{
		Brokers:           []string{"broker:9092"},
		SchemaRegistryURL: "http://sr:8081",
		SASL:              &SASLConfig{Mechanism: "PLAIN", Username: "u", Password: "p"},
	})
	require.NoError(t, err)

	// Effective config carries the new cluster.
	ext, ok := merged.Clusters["prod"]
	require.True(t, ok)
	assert.Equal(t, []string{"broker:9092"}, ext.Brokers)

	// The file was written and reloads to the same cluster.
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var reloaded Config
	require.NoError(t, yaml.Unmarshal(data, &reloaded))
	assert.Equal(t, []string{"broker:9092"}, reloaded.Clusters["prod"].Brokers)
	assert.Equal(t, "PLAIN", reloaded.Clusters["prod"].SASL.Mechanism)
}

func TestApplyCluster_CreatesParentDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "deeper", "config.yaml")

	_, err := ApplyCluster(path, Default(), "", "c1", fullCluster("b:9092"))
	require.NoError(t, err)
	_, statErr := os.Stat(path)
	assert.NoError(t, statErr)
}

func TestApplyCluster_RejectsDirectoryTarget(t *testing.T) {
	dir := t.TempDir()
	running := Default()

	merged, err := ApplyCluster(dir, running, "", "c1", fullCluster("b:9092"))
	require.Error(t, err)
	// Running config returned unchanged (no cluster added).
	assert.NotContains(t, merged.Clusters, "c1")
}

func TestApplyCluster_RejectsUnwritableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses file permission checks")
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("dynamicConfigEnabled: false\n"), 0o400))

	merged, err := ApplyCluster(path, Default(), "", "c1", fullCluster("b:9092"))
	require.Error(t, err)
	assert.NotContains(t, merged.Clusters, "c1")
}

func TestApplyCluster_InvalidCandidateLeavesRunningUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	running := Default()
	running.Clusters["existing"] = fullCluster("keep:9092")

	// Empty brokers ⇒ mandatory-field failure.
	merged, err := ApplyCluster(path, running, "", "bad", ClusterExtension{})
	require.Error(t, err)

	// Running config unchanged and nothing written.
	assert.NotContains(t, merged.Clusters, "bad")
	assert.Contains(t, merged.Clusters, "existing")
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr), "no file should be written on validation failure")
}

func TestApplyCluster_RenameReplacesOriginal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	running := Default()
	running.Clusters["old"] = fullCluster("b:9092")

	merged, err := ApplyCluster(path, running, "old", "new", fullCluster("b:9092"))
	require.NoError(t, err)
	assert.NotContains(t, merged.Clusters, "old")
	assert.Contains(t, merged.Clusters, "new")
}

func TestDeleteCluster_RemovesAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	running := Default()
	running.Clusters["gone"] = fullCluster("b:9092")

	merged, err := DeleteCluster(path, running, "gone")
	require.NoError(t, err)
	assert.NotContains(t, merged.Clusters, "gone")

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var reloaded Config
	require.NoError(t, yaml.Unmarshal(data, &reloaded))
	assert.NotContains(t, reloaded.Clusters, "gone")
}
