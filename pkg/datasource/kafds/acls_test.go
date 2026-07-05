package kafds

import (
	"errors"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AQ-2: sarama.ClusterAdmin must satisfy our extended interface.
var _ ClusterAdminInterface = (sarama.ClusterAdmin)(nil)

func strptr(s string) *string { return &s }

func TestGetACLs_MappingAndPatternType(t *testing.T) {
	admin := &MockClusterAdmin{
		MockAcls: []sarama.ResourceAcls{
			{
				Resource: sarama.Resource{ResourceType: sarama.AclResourceTopic, ResourceName: "orders", ResourcePatternType: sarama.AclPatternLiteral},
				Acls:     []*sarama.Acl{{Principal: "User:alice", Host: "*", Operation: sarama.AclOperationRead, PermissionType: sarama.AclPermissionAllow}},
			},
			{
				Resource: sarama.Resource{ResourceType: sarama.AclResourceGroup, ResourceName: "app-", ResourcePatternType: sarama.AclPatternPrefixed},
				Acls:     []*sarama.Acl{{Principal: "User:bob", Host: "10.0.0.1", Operation: sarama.AclOperationDescribe, PermissionType: sarama.AclPermissionDeny}},
			},
		},
	}
	restore := installMockAdmin(admin)
	defer restore()

	got, err := KafkaDataSourceKaf{}.GetACLs()
	require.NoError(t, err)
	require.Len(t, got, 2)
	// Sorted by principal: alice first.
	assert.Equal(t, api.ACLEntry{Principal: "User:alice", Host: "*", ResourceType: "Topic", ResourceName: "orders", PatternType: "Literal", Operation: "Read", Permission: "Allow"}, got[0])
	assert.Equal(t, api.ACLEntry{Principal: "User:bob", Host: "10.0.0.1", ResourceType: "Group", ResourceName: "app-", PatternType: "Prefixed", Operation: "Describe", Permission: "Deny"}, got[1])
}

func TestGetACLsFiltered_TranslatesFilter(t *testing.T) {
	admin := &MockClusterAdmin{}
	restore := installMockAdmin(admin)
	defer restore()

	_, err := KafkaDataSourceKaf{}.GetACLsFiltered(api.ACLFilter{ResourceType: "Topic", ResourceName: "orders", PatternType: "Prefixed"})
	require.NoError(t, err)

	// ListAcls is a passthrough on the mock; assert nothing errored and the
	// translation of a bad enum is rejected before any call.
	_, err = KafkaDataSourceKaf{}.GetACLsFiltered(api.ACLFilter{ResourceType: "Nonsense"})
	var ve api.ACLValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestCreateACL(t *testing.T) {
	t.Run("success maps enums", func(t *testing.T) {
		admin := &MockClusterAdmin{}
		restore := installMockAdmin(admin)
		defer restore()

		err := KafkaDataSourceKaf{}.CreateACL(api.ACLEntry{
			Principal: "User:alice", ResourceType: "Topic", ResourceName: "orders",
			PatternType: "Prefixed", Operation: "Read", Permission: "Allow",
		})
		require.NoError(t, err)
		require.Len(t, admin.CreateACLsCalls, 1)
		ra := admin.CreateACLsCalls[0][0]
		assert.Equal(t, sarama.AclResourceTopic, ra.Resource.ResourceType)
		assert.Equal(t, sarama.AclPatternPrefixed, ra.Resource.ResourcePatternType)
		assert.Equal(t, sarama.AclOperationRead, ra.Acls[0].Operation)
		assert.Equal(t, sarama.AclPermissionAllow, ra.Acls[0].PermissionType)
		assert.Equal(t, "*", ra.Acls[0].Host) // defaulted
	})

	t.Run("invalid principal rejected before call", func(t *testing.T) {
		admin := &MockClusterAdmin{}
		restore := installMockAdmin(admin)
		defer restore()

		err := KafkaDataSourceKaf{}.CreateACL(api.ACLEntry{Principal: "alice", ResourceType: "Topic", ResourceName: "x", Operation: "Read", Permission: "Allow"})
		var ve api.ACLValidationError
		assert.True(t, errors.As(err, &ve))
		assert.Empty(t, admin.CreateACLsCalls)
	})

	t.Run("invalid operation string rejected", func(t *testing.T) {
		admin := &MockClusterAdmin{}
		restore := installMockAdmin(admin)
		defer restore()

		err := KafkaDataSourceKaf{}.CreateACL(api.ACLEntry{Principal: "User:alice", ResourceType: "Topic", ResourceName: "x", Operation: "Frobnicate", Permission: "Allow"})
		var ve api.ACLValidationError
		assert.True(t, errors.As(err, &ve))
		assert.Empty(t, admin.CreateACLsCalls)
	})
}

func TestDeleteACL(t *testing.T) {
	entry := api.ACLEntry{Principal: "User:alice", ResourceType: "Topic", ResourceName: "orders", PatternType: "Literal", Operation: "Read", Permission: "Allow"}

	t.Run("success builds exact filter", func(t *testing.T) {
		admin := &MockClusterAdmin{MockMatchingAcls: []sarama.MatchingAcl{{}}}
		restore := installMockAdmin(admin)
		defer restore()

		err := KafkaDataSourceKaf{}.DeleteACL(entry)
		require.NoError(t, err)
		require.Len(t, admin.DeleteACLCalls, 1)
		f := admin.DeleteACLCalls[0]
		assert.Equal(t, sarama.AclResourceTopic, f.ResourceType)
		assert.Equal(t, "orders", *f.ResourceName)
		assert.Equal(t, "User:alice", *f.Principal)
		assert.Equal(t, sarama.AclOperationRead, f.Operation)
		assert.Equal(t, sarama.AclPermissionAllow, f.PermissionType)
	})

	t.Run("zero matches -> ACLNotFoundError", func(t *testing.T) {
		admin := &MockClusterAdmin{MockMatchingAcls: nil}
		restore := installMockAdmin(admin)
		defer restore()

		err := KafkaDataSourceKaf{}.DeleteACL(entry)
		var nf api.ACLNotFoundError
		assert.True(t, errors.As(err, &nf))
	})
}
