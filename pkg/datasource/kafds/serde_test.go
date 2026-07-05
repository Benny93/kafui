package kafds

import (
	"context"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/serde"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDecodeMessageRoutesThroughRegistry verifies DecodeMessage decodes raw
// bytes via the serde registry and records the winning serde name.
func TestDecodeMessageRoutesThroughRegistry(t *testing.T) {
	origCluster := currentCluster
	origLoad := loadSerdeConfigs
	t.Cleanup(func() {
		currentCluster = origCluster
		loadSerdeConfigs = origLoad
		invalidateSerdeRegistry()
	})
	currentCluster = nil // no schema registry
	loadSerdeConfigs = func(string) []serde.SerdeConfig { return nil }
	invalidateSerdeRegistry()

	kp := KafkaDataSourceKaf{}
	msg := api.Message{RawValue: []byte(`{"a":1}`)}
	out, err := kp.DecodeMessage(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, serde.NameJSON, out.ValueSerde)
	assert.Contains(t, out.Value, `"a"`)
}

func TestListSerdesFromRegistry(t *testing.T) {
	origCluster := currentCluster
	origLoad := loadSerdeConfigs
	t.Cleanup(func() {
		currentCluster = origCluster
		loadSerdeConfigs = origLoad
		invalidateSerdeRegistry()
	})
	currentCluster = nil
	loadSerdeConfigs = func(string) []serde.SerdeConfig { return nil }
	invalidateSerdeRegistry()

	names := KafkaDataSourceKaf{}.ListSerdes()
	assert.Contains(t, names, serde.NameJSON)
	assert.Contains(t, names, serde.NameString)
	assert.Contains(t, names, serde.NameSchemaRegistry)
}
