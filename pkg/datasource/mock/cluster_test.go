package mock

import (
	"context"
	"errors"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMock() *KafkaDataSourceMock {
	ds := &KafkaDataSourceMock{}
	ds.Init("")
	return ds
}

func TestMockClusterVariation(t *testing.T) {
	ds := newMock()
	ctx := context.Background()

	// dev: online, full capabilities, writable
	caps, err := ds.GetClusterCapabilities(ctx, "kafka-dev")
	require.NoError(t, err)
	assert.Contains(t, caps, api.CapACLEdit)
	assert.Contains(t, caps, api.CapSchemaRegistry)

	// test: read-only
	info, err := ds.GetClusterDetails("kafka-test")
	require.NoError(t, err)
	assert.True(t, info.ReadOnly)
	testCaps, _ := ds.GetClusterCapabilities(ctx, "kafka-test")
	assert.NotContains(t, testCaps, api.CapACLEdit)

	// prod: offline -> stats error, validation reports broker failure
	_, err = ds.GetClusterStatistics(ctx, "kafka-prod")
	assert.Error(t, err)
	results, err := ds.ValidateClusterConnection(ctx, "kafka-prod")
	require.NoError(t, err)
	var brokerFailed bool
	for _, r := range results {
		if r.Component == "broker" {
			brokerFailed = !r.OK
		}
	}
	assert.True(t, brokerFailed)
}

func TestMockClusterNotFound(t *testing.T) {
	ds := newMock()
	_, err := ds.GetClusterStatistics(context.Background(), "does-not-exist")
	var nf api.ClusterNotFoundError
	assert.True(t, errors.As(err, &nf))
}
