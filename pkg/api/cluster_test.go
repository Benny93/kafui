package api

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClusterErrorsUnwrap(t *testing.T) {
	base := errors.New("root")

	nf := ClusterNotFoundError{Name: "prod", Cause: base}
	assert.Contains(t, nf.Error(), "prod")
	assert.ErrorIs(t, nf, base)

	ro := ClusterReadOnlyError{Cluster: "prod", Operation: "DeleteTopic", Cause: base}
	assert.Contains(t, ro.Error(), "read-only")
	assert.Contains(t, ro.Error(), "DeleteTopic")
	assert.ErrorIs(t, ro, base)

	ns := NotSupportedError{Operation: "GetClusterStatistics"}
	assert.Contains(t, ns.Error(), "GetClusterStatistics")
}

func TestClusterOverviewHasCapability(t *testing.T) {
	o := ClusterOverview{Capabilities: []Capability{CapSchemaRegistry, CapACLView}}
	assert.True(t, o.HasCapability(CapSchemaRegistry))
	assert.True(t, o.HasCapability(CapACLView))
	assert.False(t, o.HasCapability(CapACLEdit))
	assert.False(t, o.HasCapability(CapKafkaConnect))
}

// contextsStubDS implements KafkaDataSource by embedding a nil interface and
// overriding only GetContexts, the single method ValidateClusterOverride uses.
type contextsStubDS struct {
	KafkaDataSource
	contexts []string
	err      error
}

func (s contextsStubDS) GetContexts() ([]string, error) { return s.contexts, s.err }

func TestValidateClusterOverride(t *testing.T) {
	ds := contextsStubDS{contexts: []string{"vehub-dev-aks", "vehub-preprod-aks"}}

	assert.NoError(t, ValidateClusterOverride(ds, ""), "no override requested")
	assert.NoError(t, ValidateClusterOverride(ds, "vehub-dev-aks"), "known cluster")

	err := ValidateClusterOverride(ds, "no-such-cluster")
	var nf ClusterNotFoundError
	assert.ErrorAs(t, err, &nf)
	assert.Equal(t, "no-such-cluster", nf.Name)
	assert.Contains(t, err.Error(), "vehub-dev-aks")
	assert.Contains(t, err.Error(), "vehub-preprod-aks")

	// GetContexts failing must not block startup — validation is best-effort.
	failing := contextsStubDS{err: errors.New("boom")}
	assert.NoError(t, ValidateClusterOverride(failing, "anything"))
}
