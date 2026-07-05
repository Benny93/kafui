package kafds

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetClientQuotas_Ordering(t *testing.T) {
	admin := &MockClusterAdmin{
		MockQuotas: []sarama.DescribeClientQuotasEntry{
			{Entity: []sarama.QuotaEntityComponent{{EntityType: sarama.QuotaEntityIP, Name: "10.0.0.1"}}, Values: map[string]float64{"connection_creation_rate": 100}},
			{Entity: []sarama.QuotaEntityComponent{{EntityType: sarama.QuotaEntityUser, Name: "bob"}}, Values: map[string]float64{"producer_byte_rate": 1}},
			{Entity: []sarama.QuotaEntityComponent{{EntityType: sarama.QuotaEntityUser, Name: "alice"}}, Values: map[string]float64{"producer_byte_rate": 2}},
		},
	}
	restore := installMockAdmin(admin)
	defer restore()

	got, err := KafkaDataSourceKaf{}.GetClientQuotas()
	require.NoError(t, err)
	require.Len(t, got, 3)
	// user alice, user bob, then ip-only (user absent sorts last).
	require.NotNil(t, got[0].Entity.User)
	assert.Equal(t, "alice", *got[0].Entity.User)
	require.NotNil(t, got[1].Entity.User)
	assert.Equal(t, "bob", *got[1].Entity.User)
	assert.Nil(t, got[2].Entity.User)
	require.NotNil(t, got[2].Entity.IP)
	assert.Equal(t, "10.0.0.1", *got[2].Entity.IP)
}

func TestAlterClientQuotas_SetAndRemove(t *testing.T) {
	// Pre-existing properties for the entity: a=1, b=2.
	admin := &MockClusterAdmin{
		MockQuotas: []sarama.DescribeClientQuotasEntry{
			{Entity: []sarama.QuotaEntityComponent{{EntityType: sarama.QuotaEntityUser, Name: "alice"}}, Values: map[string]float64{"a": 1, "b": 2}},
		},
	}
	restore := installMockAdmin(admin)
	defer restore()

	err := KafkaDataSourceKaf{}.AlterClientQuotas(api.ClientQuotaEntity{User: strptr("alice")}, map[string]float64{"a": 10, "c": 3})
	require.NoError(t, err)

	sets := map[string]float64{}
	removes := map[string]bool{}
	for _, call := range admin.AlterClientQuotasCalls {
		if call.Op.Remove {
			removes[call.Op.Key] = true
		} else {
			sets[call.Op.Key] = call.Op.Value
		}
	}
	assert.Equal(t, map[string]float64{"a": 10, "c": 3}, sets)
	assert.Equal(t, map[string]bool{"b": true}, removes) // b removed, a kept (set)
}

func TestAlterClientQuotas_EmptyMapRemovesAll(t *testing.T) {
	admin := &MockClusterAdmin{
		MockQuotas: []sarama.DescribeClientQuotasEntry{
			{Entity: []sarama.QuotaEntityComponent{{EntityType: sarama.QuotaEntityUser, Name: "alice"}}, Values: map[string]float64{"a": 1, "b": 2}},
		},
	}
	restore := installMockAdmin(admin)
	defer restore()

	err := KafkaDataSourceKaf{}.AlterClientQuotas(api.ClientQuotaEntity{User: strptr("alice")}, nil)
	require.NoError(t, err)
	require.Len(t, admin.AlterClientQuotasCalls, 2)
	for _, call := range admin.AlterClientQuotasCalls {
		assert.True(t, call.Op.Remove)
	}
}

func TestAlterClientQuotas_NoEntity(t *testing.T) {
	admin := &MockClusterAdmin{}
	restore := installMockAdmin(admin)
	defer restore()

	err := KafkaDataSourceKaf{}.AlterClientQuotas(api.ClientQuotaEntity{}, map[string]float64{"a": 1})
	var qe api.QuotaValidationError
	assert.ErrorAs(t, err, &qe)
	assert.Empty(t, admin.AlterClientQuotasCalls)
}
