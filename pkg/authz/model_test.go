package authz

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAltering(t *testing.T) {
	tests := []struct {
		name     string
		resource ResourceType
		action   Action
		altering bool
	}{
		{"topic view is read", ResourceTopic, ActionView, false},
		{"topic read messages is read", ResourceTopic, ActionReadMessages, false},
		{"topic run analysis is read", ResourceTopic, ActionRunAnalysis, false},
		{"topic produce is altering", ResourceTopic, ActionProduceMessages, true},
		{"topic delete is altering", ResourceTopic, ActionDelete, true},
		{"group reset offsets is altering", ResourceConsumerGroup, ActionResetOffsets, true},
		{"schema modify compat is altering", ResourceSchema, ActionModifyCompat, true},
		{"acl view is read", ResourceACL, ActionView, false},
		{"sql execute is altering", ResourceSQLEngine, ActionExecute, true},
		{"unknown resource is altering (fail safe)", ResourceType("nope"), ActionView, true},
		{"unknown action is altering (fail safe)", ResourceTopic, Action("frobnicate"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.altering, IsAltering(tt.resource, tt.action))
		})
	}
}

func TestKnownAction(t *testing.T) {
	assert.True(t, KnownAction(ResourceTopic, ActionCreate))
	assert.True(t, KnownAction(ResourceTopic, ActionAll), "all is known for any known resource")
	assert.False(t, KnownAction(ResourceTopic, Action("bogus")))
	assert.False(t, KnownAction(ResourceType("bogus"), ActionView))
}

func TestExpandAllWildcard(t *testing.T) {
	got := Expand(Perm{Resource: ResourceConsumerGroup, Action: ActionAll})
	// consumer-group has view, delete, reset offsets.
	assert.ElementsMatch(t, []Perm{
		{ResourceConsumerGroup, ActionView},
		{ResourceConsumerGroup, ActionDelete},
		{ResourceConsumerGroup, ActionResetOffsets},
	}, got)
}

func TestExpandImpliesView(t *testing.T) {
	got := Expand(Perm{Resource: ResourceTopic, Action: ActionDelete})
	assert.Contains(t, got, Perm{ResourceTopic, ActionDelete})
	assert.Contains(t, got, Perm{ResourceTopic, ActionView}, "non-view action implies view")
}

func TestExpandConnectorImpliesConnectClusterView(t *testing.T) {
	got := Expand(Perm{Resource: ResourceConnector, Action: ActionRestart})
	assert.Contains(t, got, Perm{ResourceConnector, ActionRestart})
	assert.Contains(t, got, Perm{ResourceConnector, ActionView})
	assert.Contains(t, got, Perm{ResourceConnectCluster, ActionView}, "connector action reaches connect-cluster")
}

func TestActionsForViewFirst(t *testing.T) {
	acts := ActionsFor(ResourceTopic)
	assert.NotEmpty(t, acts)
	assert.Equal(t, ActionView, acts[0], "view must be listed first")
}
