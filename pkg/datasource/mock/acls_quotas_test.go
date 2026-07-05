package mock

import (
	"errors"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockACLs_SeedCarriesPatternType(t *testing.T) {
	m := &KafkaDataSourceMock{}
	got, err := m.GetACLs()
	require.NoError(t, err)
	require.NotEmpty(t, got)
	for _, e := range got {
		assert.NotEmpty(t, e.PatternType, "every seeded ACL must carry a pattern type")
	}
}

func TestMockACLs_CreateThenList(t *testing.T) {
	m := &KafkaDataSourceMock{}
	before, _ := m.GetACLs()

	entry := api.ACLEntry{Principal: "User:new", Host: "*", ResourceType: "Topic", ResourceName: "t1", PatternType: "Literal", Operation: "Write", Permission: "Allow"}
	require.NoError(t, m.CreateACL(entry))

	after, _ := m.GetACLs()
	assert.Len(t, after, len(before)+1)
	assert.Contains(t, after, entry)
}

func TestMockACLs_CreateValidation(t *testing.T) {
	m := &KafkaDataSourceMock{}
	err := m.CreateACL(api.ACLEntry{Principal: "bad", ResourceType: "Topic", ResourceName: "t", Operation: "Read", Permission: "Allow"})
	var ve api.ACLValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestMockACLs_DeleteThenList(t *testing.T) {
	m := &KafkaDataSourceMock{}
	entry := api.ACLEntry{Principal: "User:new", Host: "*", ResourceType: "Topic", ResourceName: "t1", PatternType: "Literal", Operation: "Write", Permission: "Allow"}
	require.NoError(t, m.CreateACL(entry))
	require.NoError(t, m.DeleteACL(entry))

	after, _ := m.GetACLs()
	assert.NotContains(t, after, entry)
}

func TestMockACLs_DeleteUnknown(t *testing.T) {
	m := &KafkaDataSourceMock{}
	err := m.DeleteACL(api.ACLEntry{Principal: "User:ghost", ResourceType: "Topic", ResourceName: "none", PatternType: "Literal", Operation: "Read", Permission: "Allow"})
	var nf api.ACLNotFoundError
	assert.True(t, errors.As(err, &nf))
}

func TestMockACLs_Filtered(t *testing.T) {
	m := &KafkaDataSourceMock{}
	all, _ := m.GetACLs()
	topics, err := m.GetACLsFiltered(api.ACLFilter{ResourceType: "Topic"})
	require.NoError(t, err)
	assert.Less(t, len(topics), len(all))
	for _, e := range topics {
		assert.Equal(t, "Topic", e.ResourceType)
	}
}

func TestMockQuotas_Lifecycle(t *testing.T) {
	m := &KafkaDataSourceMock{}
	seeded, err := m.GetClientQuotas()
	require.NoError(t, err)
	require.NotEmpty(t, seeded)

	// Create a new entity.
	u := "carol"
	require.NoError(t, m.AlterClientQuotas(api.ClientQuotaEntity{User: &u}, map[string]float64{"producer_byte_rate": 100}))
	got := findQuota(t, m, func(e api.ClientQuotaEntry) bool { return e.Entity.User != nil && *e.Entity.User == "carol" })
	assert.Equal(t, map[string]float64{"producer_byte_rate": 100}, got.Quotas)

	// Update with replace semantics: old key dropped.
	require.NoError(t, m.AlterClientQuotas(api.ClientQuotaEntity{User: &u}, map[string]float64{"consumer_byte_rate": 200}))
	got = findQuota(t, m, func(e api.ClientQuotaEntry) bool { return e.Entity.User != nil && *e.Entity.User == "carol" })
	assert.Equal(t, map[string]float64{"consumer_byte_rate": 200}, got.Quotas)

	// Delete via empty map.
	require.NoError(t, m.AlterClientQuotas(api.ClientQuotaEntity{User: &u}, nil))
	list, _ := m.GetClientQuotas()
	for _, e := range list {
		if e.Entity.User != nil {
			assert.NotEqual(t, "carol", *e.Entity.User)
		}
	}
}

func TestMockQuotas_NoEntity(t *testing.T) {
	m := &KafkaDataSourceMock{}
	err := m.AlterClientQuotas(api.ClientQuotaEntity{}, map[string]float64{"a": 1})
	var qe api.QuotaValidationError
	assert.True(t, errors.As(err, &qe))
}

func findQuota(t *testing.T, m *KafkaDataSourceMock, pred func(api.ClientQuotaEntry) bool) api.ClientQuotaEntry {
	t.Helper()
	list, err := m.GetClientQuotas()
	require.NoError(t, err)
	for _, e := range list {
		if pred(e) {
			return e
		}
	}
	t.Fatal("quota entry not found")
	return api.ClientQuotaEntry{}
}
