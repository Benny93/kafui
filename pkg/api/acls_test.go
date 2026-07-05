package api

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePrincipal(t *testing.T) {
	tests := []struct {
		name      string
		principal string
		wantErr   bool
	}{
		{"empty", "", true},
		{"missing colon", "Useralice", true},
		{"empty type", ":alice", true},
		{"empty name", "User:", true},
		{"valid simple", "User:alice", false},
		{"valid SSL DN with colons in name", "User:CN=alice,OU=eng:team", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePrincipal(tt.principal)
			if tt.wantErr {
				assert.Error(t, err)
				var ve ACLValidationError
				assert.True(t, errors.As(err, &ve), "expected ACLValidationError")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateACLEntry(t *testing.T) {
	valid := ACLEntry{Principal: "User:alice", ResourceType: "Topic", ResourceName: "orders", Operation: "Read", Permission: "Allow"}
	assert.NoError(t, ValidateACLEntry(valid))

	missingResource := valid
	missingResource.ResourceName = ""
	err := ValidateACLEntry(missingResource)
	var ve ACLValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Equal(t, "resourceName", ve.Field)
}

func TestValidateQuotaEntity(t *testing.T) {
	err := ValidateQuotaEntity(ClientQuotaEntity{})
	var qe QuotaValidationError
	assert.True(t, errors.As(err, &qe), "all-absent entity must be rejected")

	u := "alice"
	assert.NoError(t, ValidateQuotaEntity(ClientQuotaEntity{User: &u}))
}

func TestExpandConsumerACLs(t *testing.T) {
	t.Run("literal topics and groups", func(t *testing.T) {
		got, err := ExpandConsumerACLs("User:alice", "", []string{"orders"}, []string{"g1"}, "", "")
		assert.NoError(t, err)
		want := []ACLEntry{
			{Principal: "User:alice", Host: "*", ResourceType: "Topic", ResourceName: "orders", PatternType: "Literal", Operation: "Read", Permission: "Allow"},
			{Principal: "User:alice", Host: "*", ResourceType: "Topic", ResourceName: "orders", PatternType: "Literal", Operation: "Describe", Permission: "Allow"},
			{Principal: "User:alice", Host: "*", ResourceType: "Group", ResourceName: "g1", PatternType: "Literal", Operation: "Read", Permission: "Allow"},
			{Principal: "User:alice", Host: "*", ResourceType: "Group", ResourceName: "g1", PatternType: "Literal", Operation: "Describe", Permission: "Allow"},
		}
		assert.Equal(t, want, got)
	})

	t.Run("prefixed", func(t *testing.T) {
		got, err := ExpandConsumerACLs("User:alice", "10.0.0.1", nil, nil, "orders-", "app-")
		assert.NoError(t, err)
		assert.Len(t, got, 4)
		for _, e := range got {
			assert.Equal(t, "Prefixed", e.PatternType)
			assert.Equal(t, "10.0.0.1", e.Host)
		}
	})

	t.Run("both list and prefix for same kind rejected", func(t *testing.T) {
		_, err := ExpandConsumerACLs("User:alice", "", []string{"orders"}, nil, "orders-", "")
		var ve ACLValidationError
		assert.True(t, errors.As(err, &ve))
	})

	t.Run("bad principal", func(t *testing.T) {
		_, err := ExpandConsumerACLs("alice", "", []string{"orders"}, nil, "", "")
		assert.Error(t, err)
	})
}

func TestExpandProducerACLs(t *testing.T) {
	t.Run("topic list, exact txID, idempotent", func(t *testing.T) {
		got, err := ExpandProducerACLs("User:alice", "", []string{"orders"}, "", "tx1", "", true)
		assert.NoError(t, err)
		// Topic: Write+Describe+Create (3), TxID: Write+Describe (2), Idempotent (1) = 6
		assert.Len(t, got, 6)
		assert.Contains(t, got, ACLEntry{Principal: "User:alice", Host: "*", ResourceType: "TransactionalID", ResourceName: "tx1", PatternType: "Literal", Operation: "Write", Permission: "Allow"})
		assert.Contains(t, got, ACLEntry{Principal: "User:alice", Host: "*", ResourceType: "Cluster", ResourceName: "kafka-cluster", PatternType: "Literal", Operation: "IdempotentWrite", Permission: "Allow"})
	})

	t.Run("no txID, not idempotent", func(t *testing.T) {
		got, err := ExpandProducerACLs("User:alice", "", []string{"orders"}, "", "", "", false)
		assert.NoError(t, err)
		assert.Len(t, got, 3) // only topic bindings
	})

	t.Run("prefixed txID", func(t *testing.T) {
		got, err := ExpandProducerACLs("User:alice", "", nil, "", "", "tx-", false)
		assert.NoError(t, err)
		assert.Len(t, got, 2)
		for _, e := range got {
			assert.Equal(t, "Prefixed", e.PatternType)
			assert.Equal(t, "TransactionalID", e.ResourceType)
		}
	})
}

func TestExpandStreamAppACLs(t *testing.T) {
	got, err := ExpandStreamAppACLs("User:alice", "", "myapp", []string{"in"}, []string{"out"})
	assert.NoError(t, err)
	// READ on input (1) + WRITE on output (1) + ALL prefixed Topic (1) + ALL prefixed Group (1) = 4
	assert.Len(t, got, 4)
	assert.Contains(t, got, ACLEntry{Principal: "User:alice", Host: "*", ResourceType: "Topic", ResourceName: "in", PatternType: "Literal", Operation: "Read", Permission: "Allow"})
	assert.Contains(t, got, ACLEntry{Principal: "User:alice", Host: "*", ResourceType: "Topic", ResourceName: "out", PatternType: "Literal", Operation: "Write", Permission: "Allow"})
	assert.Contains(t, got, ACLEntry{Principal: "User:alice", Host: "*", ResourceType: "Topic", ResourceName: "myapp", PatternType: "Prefixed", Operation: "All", Permission: "Allow"})
	assert.Contains(t, got, ACLEntry{Principal: "User:alice", Host: "*", ResourceType: "Group", ResourceName: "myapp", PatternType: "Prefixed", Operation: "All", Permission: "Allow"})

	_, err = ExpandStreamAppACLs("User:alice", "", "", nil, nil)
	assert.Error(t, err, "empty appID rejected")
}
