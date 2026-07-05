package appconfig

import (
	"fmt"
	"strings"
)

// ClusterConfig is the minimal per-cluster view validation needs. It is built by
// merging the kaf clusters with kafui extensions (see merge.go).
type ClusterConfig struct {
	Name    string
	Brokers []string
}

// ValidationError describes a startup configuration problem.
type ValidationError struct {
	Field   string
	Cluster string
	Message string
}

func (e ValidationError) Error() string {
	if e.Cluster != "" {
		return fmt.Sprintf("invalid config for cluster %q: %s (%s)", e.Cluster, e.Message, e.Field)
	}
	return fmt.Sprintf("invalid config: %s (%s)", e.Message, e.Field)
}

// Validate checks the merged cluster list at startup:
//   - a single unnamed cluster is assigned the name "default" (mutated in place)
//   - multiple clusters require unique, non-empty names
//   - every cluster must have at least one broker
func Validate(clusters []ClusterConfig) error {
	if len(clusters) == 1 && strings.TrimSpace(clusters[0].Name) == "" {
		clusters[0].Name = "default"
	}

	seen := map[string]bool{}
	for i := range clusters {
		c := &clusters[i]
		name := strings.TrimSpace(c.Name)
		if len(clusters) > 1 && name == "" {
			return ValidationError{Field: "name", Message: "cluster name must not be empty when multiple clusters are configured"}
		}
		if name != "" {
			if seen[name] {
				return ValidationError{Field: "name", Cluster: name, Message: "duplicate cluster name"}
			}
			seen[name] = true
		}
		if len(c.Brokers) == 0 {
			return ValidationError{Field: "brokers", Cluster: name, Message: "at least one broker is required"}
		}
	}
	return nil
}
