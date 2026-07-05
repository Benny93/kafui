package cmd

import (
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/stretchr/testify/assert"
)

func TestRunHealthMockHealthy(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	// Mock broker + schema registry are always reachable → exit 0.
	assert.Equal(t, 0, runHealth(ds))
}

// TestNewHealthDataSourceUnknownCluster guards against bug #1 regressing: an
// unknown --cluster/-c must be rejected instead of silently falling back to
// localhost:9092.
func TestNewHealthDataSourceUnknownCluster(t *testing.T) {
	originalCluster := clusterFlag
	defer func() { clusterFlag = originalCluster }()

	clusterFlag = "no-such-cluster"
	ds, err := newHealthDataSource(false)
	assert.Nil(t, ds)
	assert.ErrorContains(t, err, "no-such-cluster")
}
