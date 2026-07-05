package aclcsv

import (
	"errors"
	"strings"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleEntries() []api.ACLEntry {
	return []api.ACLEntry{
		{Principal: "User:CN=alice,OU=eng", Host: "*", ResourceType: "Topic", ResourceName: "orders", PatternType: "Literal", Operation: "Read", Permission: "Allow"},
		{Principal: "User:bob", Host: "10.0.0.1", ResourceType: "Group", ResourceName: "app-", PatternType: "Prefixed", Operation: "Describe", Permission: "Deny"},
	}
}

func TestMarshalParse_RoundTrip(t *testing.T) {
	entries := sampleEntries()
	csv := Marshal(entries)
	// Header present and DN principal (with a comma) quoted.
	assert.True(t, strings.HasPrefix(csv, "Principal,ResourceType,PatternType,ResourceName,Operation,PermissionType,Host"))

	got, err := Parse(csv)
	require.NoError(t, err)
	assert.Equal(t, entries, got)
}

func TestParse_OptionalHeaderVariantsAndBlankLines(t *testing.T) {
	input := "\n" +
		"principal, resourcetype ,patterntype,resourcename,operation,permissiontype,host\n" +
		"\n" +
		"User:alice,Topic,Literal,orders,Read,Allow,*\n" +
		"\n"
	got, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "User:alice", got[0].Principal)
}

func TestParse_NoHeader(t *testing.T) {
	got, err := Parse("User:alice,Topic,Literal,orders,Read,Allow,*\n")
	require.NoError(t, err)
	require.Len(t, got, 1)
}

func TestParse_Malformed(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLine int
		wantCol  int
	}{
		{"too few columns", "User:alice,Topic,Literal,orders,Read,Allow\n", 1, 0},
		{"blank value", "User:alice,Topic,Literal,,Read,Allow,*\n", 1, 4},
		{"bad resource type", "User:alice,Nope,Literal,orders,Read,Allow,*\n", 1, 2},
		{"bad pattern type", "User:alice,Topic,Weird,orders,Read,Allow,*\n", 1, 3},
		{"bad operation", "User:alice,Topic,Literal,orders,Frob,Allow,*\n", 1, 5},
		{"bad permission", "User:alice,Topic,Literal,orders,Read,Maybe,*\n", 1, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			require.Error(t, err)
			var pe CSVParseError
			require.True(t, errors.As(err, &pe))
			assert.Equal(t, tt.wantLine, pe.Line)
			assert.Equal(t, tt.wantCol, pe.Column)
		})
	}
}

func TestParse_MalformedLineNumberWithHeader(t *testing.T) {
	input := "Principal,ResourceType,PatternType,ResourceName,Operation,PermissionType,Host\n" +
		"User:alice,Topic,Literal,orders,Read,Allow,*\n" +
		"User:bob,Nope,Literal,orders,Read,Allow,*\n"
	_, err := Parse(input)
	var pe CSVParseError
	require.True(t, errors.As(err, &pe))
	assert.Equal(t, 3, pe.Line)
	assert.Equal(t, 2, pe.Column)
}

// --- Sync (AQ-9) ---

// syncMockDS embeds the full mock datasource and overrides only the ACL methods
// used by SyncACLs, capturing calls for assertions.
type syncMockDS struct {
	*mock.KafkaDataSourceMock
	current    []api.ACLEntry
	created    []api.ACLEntry
	deleted    []api.ACLEntry
	getACLsErr error
}

func (m *syncMockDS) GetACLs() ([]api.ACLEntry, error) {
	return m.current, m.getACLsErr
}
func (m *syncMockDS) CreateACL(e api.ACLEntry) error { m.created = append(m.created, e); return nil }
func (m *syncMockDS) DeleteACL(e api.ACLEntry) error { m.deleted = append(m.deleted, e); return nil }

func newSyncDS(current []api.ACLEntry) *syncMockDS {
	return &syncMockDS{KafkaDataSourceMock: &mock.KafkaDataSourceMock{}, current: current}
}

func TestSyncACLs(t *testing.T) {
	a := api.ACLEntry{Principal: "User:a", Host: "*", ResourceType: "Topic", ResourceName: "t1", PatternType: "Literal", Operation: "Read", Permission: "Allow"}
	b := api.ACLEntry{Principal: "User:b", Host: "*", ResourceType: "Topic", ResourceName: "t2", PatternType: "Literal", Operation: "Write", Permission: "Allow"}

	t.Run("additions only", func(t *testing.T) {
		ds := newSyncDS(nil)
		plan, err := SyncACLs(ds, []api.ACLEntry{a})
		require.NoError(t, err)
		assert.Equal(t, []api.ACLEntry{a}, plan.ToCreate)
		assert.Empty(t, plan.ToDelete)
		require.NoError(t, plan.Apply(ds))
		assert.Equal(t, []api.ACLEntry{a}, ds.created)
		assert.Empty(t, ds.deleted)
	})

	t.Run("deletions only", func(t *testing.T) {
		ds := newSyncDS([]api.ACLEntry{a, b})
		plan, err := SyncACLs(ds, []api.ACLEntry{a})
		require.NoError(t, err)
		assert.Equal(t, []api.ACLEntry{b}, plan.ToDelete)
		assert.Empty(t, plan.ToCreate)
	})

	t.Run("mixed", func(t *testing.T) {
		ds := newSyncDS([]api.ACLEntry{a})
		plan, err := SyncACLs(ds, []api.ACLEntry{b})
		require.NoError(t, err)
		assert.Equal(t, []api.ACLEntry{b}, plan.ToCreate)
		assert.Equal(t, []api.ACLEntry{a}, plan.ToDelete)
	})

	t.Run("already in sync -> no calls", func(t *testing.T) {
		ds := newSyncDS([]api.ACLEntry{a})
		plan, err := SyncACLs(ds, []api.ACLEntry{a})
		require.NoError(t, err)
		assert.True(t, plan.Empty())
		require.NoError(t, plan.Apply(ds))
		assert.Empty(t, ds.created)
		assert.Empty(t, ds.deleted)
	})
}
