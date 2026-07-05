package authz

import (
	"fmt"
	"regexp"
	"sort"
	"sync"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
)

// compiledPerm is a validated permission with its regex compiled and actions
// expanded (implied view etc. already applied).
type compiledPerm struct {
	resource ResourceType
	pattern  string         // original regex text ("" = matches any name)
	re       *regexp.Regexp // nil when pattern is empty
	actions  map[Action]bool
}

// matches reports whether this permission grants action on the named resource.
// An empty name (create/unnamed check) matches only pattern-less permissions.
func (p compiledPerm) matches(rt ResourceType, name string, action Action) bool {
	if p.resource != rt || !p.actions[action] {
		return false
	}
	if name == "" {
		return p.re == nil
	}
	if p.re == nil {
		return true
	}
	return p.re.MatchString(name)
}

type compiledProfile struct {
	name  string
	perms []compiledPerm
}

func (cp *compiledProfile) allows(rt ResourceType, name string, action Action) bool {
	for _, p := range cp.perms {
		if p.matches(rt, name, action) {
			return true
		}
	}
	return false
}

// Gate evaluates permission and read-only checks against the active cluster's
// profile. It is safe for concurrent use and re-resolves the active profile on
// SetCluster (called by the guard on context switch).
type Gate struct {
	mu sync.RWMutex

	enabled       bool
	forceReadOnly bool                    // global --read-only flag
	readOnly      func(cluster string) bool // per-cluster read-only from config

	profiles      map[string]*compiledProfile
	byCluster     map[string]*compiledProfile
	def           *compiledProfile
	activeName    string // configured override
	cluster       string
	active        *compiledProfile
}

// NewGate compiles and validates the authz configuration, returning a fail-fast
// error for empty clusters, permissions missing resource/actions, unknown
// resources/actions, or invalid name regexes. readOnly reports whether a cluster
// is configured read-only; forceReadOnly is the global --read-only CLI flag.
func NewGate(cfg appconfig.AuthzSettings, readOnly func(cluster string) bool, forceReadOnly bool) (*Gate, error) {
	g := &Gate{
		enabled:       cfg.Enabled(),
		forceReadOnly: forceReadOnly,
		readOnly:      readOnly,
		profiles:      map[string]*compiledProfile{},
		byCluster:     map[string]*compiledProfile{},
		activeName:    cfg.ActiveProfile,
	}

	for _, prof := range cfg.Profiles {
		if len(prof.Clusters) == 0 {
			return nil, fmt.Errorf("authz profile %q: at least one cluster is required", prof.Name)
		}
		cp, err := compileProfile(prof)
		if err != nil {
			return nil, err
		}
		g.profiles[prof.Name] = cp
		for _, cl := range prof.Clusters {
			g.byCluster[cl] = cp
		}
	}
	if cfg.Default != nil {
		cp, err := compileProfile(*cfg.Default)
		if err != nil {
			return nil, err
		}
		if cp.name == "" {
			cp.name = "default"
		}
		g.def = cp
	}
	if g.activeName != "" {
		if _, ok := g.profiles[g.activeName]; !ok {
			return nil, fmt.Errorf("authz activeProfile %q: no such profile", g.activeName)
		}
	}
	return g, nil
}

func compileProfile(prof appconfig.Profile) (*compiledProfile, error) {
	cp := &compiledProfile{name: prof.Name}
	for _, perm := range prof.Permissions {
		rt := ResourceType(perm.Resource)
		if perm.Resource == "" {
			return nil, fmt.Errorf("authz profile %q: permission is missing a resource", prof.Name)
		}
		if !KnownResource(rt) {
			return nil, fmt.Errorf("authz profile %q: unknown resource %q", prof.Name, perm.Resource)
		}
		if len(perm.Actions) == 0 {
			return nil, fmt.Errorf("authz profile %q: permission on %q has no actions", prof.Name, perm.Resource)
		}
		actions := map[Action]bool{}
		for _, a := range perm.Actions {
			act := Action(a)
			if !KnownAction(rt, act) {
				return nil, fmt.Errorf("authz profile %q: action %q is not valid for resource %q", prof.Name, a, perm.Resource)
			}
			for _, e := range Expand(Perm{Resource: rt, Action: act}) {
				if e.Resource == rt {
					actions[e.Action] = true
				}
			}
		}
		var re *regexp.Regexp
		if perm.Name != "" {
			compiled, err := regexp.Compile("^(?:" + perm.Name + ")$")
			if err != nil {
				return nil, fmt.Errorf("authz profile %q: invalid name pattern %q: %w", prof.Name, perm.Name, err)
			}
			re = compiled
		}
		cp.perms = append(cp.perms, compiledPerm{resource: rt, pattern: perm.Name, re: re, actions: actions})
	}
	return cp, nil
}

// SetCluster re-resolves the active profile for the named cluster. Called on
// context switch. The activeProfile override wins; otherwise a profile listing
// the cluster is used, falling back to the default profile.
func (g *Gate) SetCluster(cluster string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cluster = cluster
	switch {
	case g.activeName != "":
		g.active = g.profiles[g.activeName]
	case g.byCluster[cluster] != nil:
		g.active = g.byCluster[cluster]
	default:
		g.active = g.def
	}
}

// Enabled reports whether authorization is active (at least one profile or a
// default profile configured).
func (g *Gate) Enabled() bool { return g.enabled }

// ActiveProfileName returns the name of the resolved active profile, or "" when
// authz is disabled or no profile covers the current cluster.
func (g *Gate) ActiveProfileName() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.active == nil {
		return ""
	}
	return g.active.name
}

// isReadOnly reports whether the cluster is read-only (global flag or per-cluster).
func (g *Gate) isReadOnly(cluster string) bool {
	if g.forceReadOnly {
		return true
	}
	return g.readOnly != nil && g.readOnly(cluster)
}

// ReadOnly reports whether the current cluster is read-only.
func (g *Gate) ReadOnly() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.isReadOnly(g.cluster)
}

// Check evaluates an action on a resource for the current cluster. It returns
// api.ClusterReadOnlyError for an altering action on a read-only cluster,
// api.AccessDeniedError when the active profile denies it, or nil when allowed.
func (g *Gate) Check(action Action, rt ResourceType, name string) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if IsAltering(rt, action) && g.isReadOnly(g.cluster) {
		return api.ClusterReadOnlyError{Cluster: g.cluster, Operation: fmt.Sprintf("%s on %s", action, rt)}
	}
	if !g.enabled {
		return nil
	}
	if g.active != nil && g.active.allows(rt, name, action) {
		return nil
	}
	return api.AccessDeniedError{Resource: string(rt), Name: name, Action: string(action)}
}

// Allowed is the boolean form of Check, for UI decisions.
func (g *Gate) Allowed(action Action, rt ResourceType, name string) bool {
	return g.Check(action, rt, name) == nil
}

// EffectivePerm is one row of the resolved permission set for the whoami view.
type EffectivePerm struct {
	Resource ResourceType
	Pattern  string // "" = any name
	Action   Action
}

// EffectivePermissions returns the flattened, expanded permission set of the
// active profile, sorted for stable rendering. Empty when authz is disabled or
// no profile covers the current cluster.
func (g *Gate) EffectivePermissions() []EffectivePerm {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.active == nil {
		return nil
	}
	var out []EffectivePerm
	for _, p := range g.active.perms {
		for act := range p.actions {
			out = append(out, EffectivePerm{Resource: p.resource, Pattern: p.pattern, Action: act})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Resource != out[j].Resource {
			return out[i].Resource < out[j].Resource
		}
		if out[i].Pattern != out[j].Pattern {
			return out[i].Pattern < out[j].Pattern
		}
		return out[i].Action < out[j].Action
	})
	return out
}
