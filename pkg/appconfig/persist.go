package appconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Save writes the kafui-owned config to path, creating parent directories as
// needed. It ONLY ever writes the kafui file — never ~/.kaf/config. Fails
// (without writing) if the target is a directory or an existing file is not
// writable.
func Save(path string, cfg Config) error {
	if path == "" {
		return fmt.Errorf("save config: empty path")
	}
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			return fmt.Errorf("save config: %s is a directory", path)
		}
		if info.Mode().Perm()&0o200 == 0 {
			return fmt.Errorf("save config: %s is not writable", path)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("save config: creating parent dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("save config: marshal: %w", err)
	}
	// Write atomically via a temp file + rename so a crash can't truncate the file.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("save config: write: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("save config: rename: %w", err)
	}
	return nil
}

// cloneClusters returns a shallow copy of the cluster map so callers can mutate
// the copy without touching the running config. Extension values are copied by
// value (whole entries are replaced on apply, never mutated field-by-field).
func cloneClusters(in map[string]ClusterExtension) map[string]ClusterExtension {
	out := make(map[string]ClusterExtension, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	return out
}

// fullyDefinedClusterConfigs projects the fully-kafui-defined clusters to the
// minimal view Validate consumes. Overlay-only entries (no brokers) reference
// kaf-file clusters and are not broker-validated here.
func fullyDefinedClusterConfigs(clusters map[string]ClusterExtension) []ClusterConfig {
	out := make([]ClusterConfig, 0, len(clusters))
	for name, ext := range clusters {
		if !ext.IsFullyDefined() {
			continue
		}
		out = append(out, ClusterConfig{Name: name, Brokers: ext.Brokers})
	}
	return out
}

// ApplyCluster merges a cluster into the running config (replacing originalName
// on rename, otherwise inserting/replacing by name), structurally validates the
// result, and persists ONLY the kafui-owned file at path. On any error the
// running config is returned unchanged and nothing is written. On success it
// returns the new effective config.
func ApplyCluster(path string, running Config, originalName, name string, ext ClusterExtension) (Config, error) {
	if name == "" {
		return running, ValidationError{Field: "name", Message: "cluster name is required"}
	}
	if !ext.IsFullyDefined() {
		return running, ValidationError{Field: "brokers", Cluster: name, Message: "at least one broker is required"}
	}

	merged := running
	merged.Clusters = cloneClusters(running.Clusters)
	if originalName != "" && originalName != name {
		delete(merged.Clusters, originalName)
	}
	merged.Clusters[name] = ext

	if err := Validate(fullyDefinedClusterConfigs(merged.Clusters)); err != nil {
		return running, err
	}
	if err := Save(path, merged); err != nil {
		return running, err
	}
	return merged, nil
}

// DeleteCluster removes a cluster from the running config and persists ONLY the
// kafui file. On any error the running config is returned unchanged.
func DeleteCluster(path string, running Config, name string) (Config, error) {
	merged := running
	merged.Clusters = cloneClusters(running.Clusters)
	delete(merged.Clusters, name)
	if err := Save(path, merged); err != nil {
		return running, err
	}
	return merged, nil
}
