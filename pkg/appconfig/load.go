package appconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultPath returns the default kafui config path ($HOME/.config/kafui/config.yaml).
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.yaml"
	}
	return filepath.Join(home, ".config", "kafui", "config.yaml")
}

// Load reads the kafui config from path. A missing file is not an error: it
// returns defaults with a nil error (per "Missing dynamic config file tolerated").
// A present-but-malformed file is a hard error.
func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("reading kafui config %s: %w", path, err)
	}
	// Decode onto the defaults so unset keys keep their default values.
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Default(), fmt.Errorf("parsing kafui config %s: %w", path, err)
	}
	if cfg.Clusters == nil {
		cfg.Clusters = map[string]ClusterExtension{}
	}
	return cfg, nil
}
