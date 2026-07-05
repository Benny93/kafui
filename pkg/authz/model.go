// Package authz is kafui's local authorization layer: a permission model, an
// action vocabulary, and a Gate that classifies every datasource operation as
// allowed or denied for the active cluster profile.
//
// kafui is a single-user local tool, so this is a self-imposed guardrail (like a
// read-only kubeconfig), not a security boundary — the real enforcement remains
// broker-side ACLs plus the SASL/TLS credentials in ~/.kaf/config. When no
// profiles are configured the Gate is disabled and allows everything; read-only
// mode is an independent switch that always denies altering actions.
package authz

// ResourceType enumerates the Kafka resource categories kafui can act on.
type ResourceType string

const (
	ResourceTopic          ResourceType = "topic"
	ResourceConsumerGroup  ResourceType = "consumer-group"
	ResourceSchema         ResourceType = "schema"
	ResourceConnectCluster ResourceType = "connect-cluster"
	ResourceConnector      ResourceType = "connector"
	ResourceSQLEngine      ResourceType = "sql-engine"
	ResourceACL            ResourceType = "acl"
	ResourceAudit          ResourceType = "audit"
	ResourceClientQuota    ResourceType = "client-quotas"
	ResourceClusterConfig  ResourceType = "cluster-configuration"
	ResourceAppConfig      ResourceType = "application-configuration"
)

// Action names an operation class. Read actions leave the cluster unchanged;
// altering actions mutate state and are the ones read-only mode blocks.
type Action string

const (
	// ActionAll is the wildcard action expanded to every action of a resource.
	ActionAll Action = "all"

	ActionView            Action = "view"
	ActionReadMessages    Action = "read messages"
	ActionProduceMessages Action = "produce messages"
	ActionDeleteMessages  Action = "delete messages"
	ActionRunAnalysis     Action = "run analysis"
	ActionCreate          Action = "create"
	ActionEdit            Action = "edit"
	ActionDelete          Action = "delete"
	ActionResetOffsets    Action = "reset offsets"
	ActionExecute         Action = "execute"
	ActionModifyCompat    Action = "modify compatibility"
	ActionPause           Action = "pause"
	ActionResume          Action = "resume"
	ActionRestart         Action = "restart"
)

// registry maps each resource type to its valid actions, each flagged altering
// (true) or read-only (false). This is the single source of truth for the action
// vocabulary and the altering classification.
var registry = map[ResourceType]map[Action]bool{
	ResourceTopic: {
		ActionView:            false,
		ActionReadMessages:    false,
		ActionRunAnalysis:     false,
		ActionProduceMessages: true,
		ActionDeleteMessages:  true,
		ActionCreate:          true,
		ActionEdit:            true,
		ActionDelete:          true,
	},
	ResourceConsumerGroup: {
		ActionView:         false,
		ActionDelete:       true,
		ActionResetOffsets: true,
	},
	ResourceSchema: {
		ActionView:         false,
		ActionCreate:       true,
		ActionEdit:         true,
		ActionDelete:       true,
		ActionModifyCompat: true,
	},
	ResourceConnectCluster: {
		ActionView: false,
	},
	ResourceConnector: {
		ActionView:         false,
		ActionCreate:       true,
		ActionEdit:         true,
		ActionDelete:       true,
		ActionPause:        true,
		ActionResume:       true,
		ActionRestart:      true,
		ActionResetOffsets: true,
	},
	ResourceSQLEngine: {
		ActionView:    false,
		ActionExecute: true,
	},
	ResourceACL: {
		ActionView:   false,
		ActionCreate: true,
		ActionDelete: true,
	},
	ResourceAudit: {
		ActionView: false,
	},
	ResourceClientQuota: {
		ActionView: false,
		ActionEdit: true,
	},
	ResourceClusterConfig: {
		ActionView: false,
		ActionEdit: true,
	},
	ResourceAppConfig: {
		ActionView: false,
		ActionEdit: true,
	},
}

// KnownResource reports whether rt is a recognized resource type.
func KnownResource(rt ResourceType) bool {
	_, ok := registry[rt]
	return ok
}

// KnownAction reports whether action is valid for the resource type. The "all"
// wildcard is considered known for any known resource.
func KnownAction(rt ResourceType, action Action) bool {
	acts, ok := registry[rt]
	if !ok {
		return false
	}
	if action == ActionAll {
		return true
	}
	_, ok = acts[action]
	return ok
}

// IsAltering reports whether the (resource, action) pair mutates cluster state.
// Unknown pairs are treated as altering (fail safe: deny under read-only).
func IsAltering(rt ResourceType, action Action) bool {
	acts, ok := registry[rt]
	if !ok {
		return true
	}
	altering, ok := acts[action]
	if !ok {
		return true
	}
	return altering
}

// ActionsFor returns every action valid for the resource type (excluding the
// "all" wildcard), in a stable order (view first).
func ActionsFor(rt ResourceType) []Action {
	acts := registry[rt]
	out := make([]Action, 0, len(acts))
	if _, ok := acts[ActionView]; ok {
		out = append(out, ActionView)
	}
	for a := range acts {
		if a != ActionView {
			out = append(out, a)
		}
	}
	return out
}

// Perm is a single (resource, action) grant.
type Perm struct {
	Resource ResourceType
	Action   Action
}

// Expand normalizes a permission into the full set of concrete grants it implies:
//   - the "all" wildcard expands to every action of the resource;
//   - any non-view action implies view on the same resource;
//   - any connector action implies view on the reaching connect-cluster.
func Expand(p Perm) []Perm {
	var out []Perm
	seen := map[Perm]bool{}
	add := func(q Perm) {
		if !seen[q] {
			seen[q] = true
			out = append(out, q)
		}
	}

	actions := []Action{p.Action}
	if p.Action == ActionAll {
		actions = ActionsFor(p.Resource)
	}
	for _, a := range actions {
		add(Perm{Resource: p.Resource, Action: a})
		if a != ActionView {
			add(Perm{Resource: p.Resource, Action: ActionView}) // implied view
		}
	}
	// A connector grant implies visibility of its connect-cluster.
	if p.Resource == ResourceConnector {
		add(Perm{Resource: ResourceConnectCluster, Action: ActionView})
	}
	return out
}
