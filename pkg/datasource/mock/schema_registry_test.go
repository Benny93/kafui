package mock

import (
	"errors"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockRegistry_VersionsAndContent(t *testing.T) {
	kp := &KafkaDataSourceMock{}

	versions, err := kp.GetSchemaVersions("orders-value")
	require.NoError(t, err)
	require.Len(t, versions, 3)
	assert.Equal(t, 1, versions[0].Version)
	assert.Equal(t, 3, versions[2].Version)
	assert.NotEmpty(t, versions[0].Schema, "mock versions carry evolved schema text for diffing")

	latest, err := kp.GetSchemaContent("orders-value", 0)
	require.NoError(t, err)
	assert.Contains(t, latest, "OrderItem", "v3 references a named record")

	v1, err := kp.GetSchemaContent("orders-value", 1)
	require.NoError(t, err)
	assert.NotEqual(t, latest, v1)

	t.Run("unknown subject", func(t *testing.T) {
		_, err := kp.GetSchemaVersions("nope")
		var e api.SubjectNotFoundError
		assert.True(t, errors.As(err, &e))
	})
	t.Run("unknown version", func(t *testing.T) {
		_, err := kp.GetSchemaContent("orders-value", 99)
		var e api.SchemaVersionNotFoundError
		assert.True(t, errors.As(err, &e))
	})
}

func TestMockRegistry_Compatibility(t *testing.T) {
	kp := &KafkaDataSourceMock{}

	global, err := kp.GetGlobalCompatibility()
	require.NoError(t, err)
	assert.Equal(t, api.CompatibilityBackward, global)

	// payments-value has its own level.
	level, specific, err := kp.GetSubjectCompatibility("payments-value")
	require.NoError(t, err)
	assert.True(t, specific)
	assert.Equal(t, api.CompatibilityFull, level)

	// orders-value falls back to global.
	level, specific, err = kp.GetSubjectCompatibility("orders-value")
	require.NoError(t, err)
	assert.False(t, specific)
	assert.Equal(t, api.CompatibilityBackward, level)

	// Details expose the effective level.
	details, err := kp.GetSchemaDetails([]string{"payments-value", "orders-value"})
	require.NoError(t, err)
	assert.Equal(t, string(api.CompatibilityFull), details[0].Compatibility)
	assert.Equal(t, string(api.CompatibilityBackward), details[1].Compatibility)
}

func TestMockRegistry_SetCompatibilityRoundTrip(t *testing.T) {
	kp := &KafkaDataSourceMock{}

	require.NoError(t, kp.SetGlobalCompatibility(api.CompatibilityForward))
	global, _ := kp.GetGlobalCompatibility()
	assert.Equal(t, api.CompatibilityForward, global)

	require.NoError(t, kp.SetSubjectCompatibility("orders-value", api.CompatibilityNone))
	level, specific, _ := kp.GetSubjectCompatibility("orders-value")
	assert.True(t, specific)
	assert.Equal(t, api.CompatibilityNone, level)

	t.Run("invalid level rejected", func(t *testing.T) {
		err := kp.SetGlobalCompatibility(api.CompatibilityLevel("BOGUS"))
		require.Error(t, err)
	})
}

func TestMockRegistry_RegisterRoundTrip(t *testing.T) {
	kp := &KafkaDataSourceMock{}

	before, _ := kp.GetSchemaVersions("orders-value")
	schema, err := kp.RegisterSchema("orders-value", `{"type":"record","name":"OrderCreatedEvent","fields":[]}`, "AVRO")
	require.NoError(t, err)
	assert.Equal(t, len(before)+1, schema.Version)
	assert.NotZero(t, schema.ID)

	after, _ := kp.GetSchemaVersions("orders-value")
	assert.Len(t, after, len(before)+1)

	t.Run("new subject", func(t *testing.T) {
		s, err := kp.RegisterSchema("brand-new-value", `{"type":"string"}`, "")
		require.NoError(t, err)
		assert.Equal(t, 1, s.Version)
		names, _ := kp.GetSchemas()
		found := false
		for _, n := range names {
			if n.Subject == "brand-new-value" {
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("incompatible marker on existing subject", func(t *testing.T) {
		_, err := kp.RegisterSchema("orders-value", `{"INCOMPATIBLE":true}`, "")
		var e api.SchemaIncompatibleError
		assert.True(t, errors.As(err, &e))
	})
}

func TestMockRegistry_CompatibilityCheck(t *testing.T) {
	kp := &KafkaDataSourceMock{}

	ok, msgs, err := kp.CheckSchemaCompatibility("orders-value", `{"type":"string"}`, "")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Empty(t, msgs)

	ok, msgs, err = kp.CheckSchemaCompatibility("orders-value", `{"INCOMPATIBLE":true}`, "")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.NotEmpty(t, msgs)

	// A compatibility check never mutates the version list.
	versions, _ := kp.GetSchemaVersions("orders-value")
	assert.Len(t, versions, 3)

	t.Run("unknown subject", func(t *testing.T) {
		_, _, err := kp.CheckSchemaCompatibility("nope", "{}", "")
		var e api.SubjectNotFoundError
		assert.True(t, errors.As(err, &e))
	})
}

func TestMockRegistry_Delete(t *testing.T) {
	t.Run("delete subject removes it from listing", func(t *testing.T) {
		kp := &KafkaDataSourceMock{}
		deleted, err := kp.DeleteSubject("inventory-value", false)
		require.NoError(t, err)
		assert.Equal(t, []int{1}, deleted)
		_, err = kp.GetSchemaVersions("inventory-value")
		var e api.SubjectNotFoundError
		assert.True(t, errors.As(err, &e))
	})

	t.Run("delete latest version", func(t *testing.T) {
		kp := &KafkaDataSourceMock{}
		require.NoError(t, kp.DeleteSchemaVersion("payments-value", -1, false))
		versions, _ := kp.GetSchemaVersions("payments-value")
		require.Len(t, versions, 1)
		assert.Equal(t, 1, versions[0].Version)
	})

	t.Run("delete missing version", func(t *testing.T) {
		kp := &KafkaDataSourceMock{}
		err := kp.DeleteSchemaVersion("payments-value", 99, false)
		var e api.SchemaVersionNotFoundError
		assert.True(t, errors.As(err, &e))
	})
}
