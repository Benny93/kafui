package authz

import (
	"errors"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func viewerProfile() appconfig.AuthzSettings {
	return appconfig.AuthzSettings{
		Profiles: []appconfig.Profile{
			{
				Name:     "viewer",
				Clusters: []string{"prod"},
				Permissions: []appconfig.Permission{
					{Resource: "topic", Actions: []string{"view", "read messages"}},
				},
			},
			{
				Name:     "orders-admin",
				Clusters: []string{"staging"},
				Permissions: []appconfig.Permission{
					{Resource: "topic", Name: "orders-.*", Actions: []string{"all"}},
				},
			},
		},
	}
}

func TestGateDisabledAllowsEverything(t *testing.T) {
	g, err := NewGate(appconfig.AuthzSettings{}, nil, false)
	require.NoError(t, err)
	g.SetCluster("prod")
	assert.False(t, g.Enabled())
	assert.NoError(t, g.Check(ActionDelete, ResourceTopic, "anything"))
}

func TestGateAllowDenyPerProfile(t *testing.T) {
	g, err := NewGate(viewerProfile(), nil, false)
	require.NoError(t, err)

	g.SetCluster("prod")
	assert.NoError(t, g.Check(ActionView, ResourceTopic, "orders"), "viewer can view")
	assert.NoError(t, g.Check(ActionReadMessages, ResourceTopic, "orders"))
	err = g.Check(ActionDelete, ResourceTopic, "orders")
	assert.Error(t, err, "viewer cannot delete")
	var denied api.AccessDeniedError
	assert.ErrorAs(t, err, &denied)
	assert.Equal(t, "delete", denied.Action)
}

func TestGateNamePatternFullMatch(t *testing.T) {
	g, err := NewGate(viewerProfile(), nil, false)
	require.NoError(t, err)
	g.SetCluster("staging")

	assert.True(t, g.Allowed(ActionDelete, ResourceTopic, "orders-eu"), "matches orders-.*")
	assert.True(t, g.Allowed(ActionView, ResourceTopic, "orders-eu"), "all implies view")
	assert.False(t, g.Allowed(ActionDelete, ResourceTopic, "payments"), "outside pattern")
	assert.False(t, g.Allowed(ActionDelete, ResourceTopic, "x-orders-eu"), "regex must fully match, not partial")
}

func TestGateClusterWithoutProfileDenies(t *testing.T) {
	g, err := NewGate(viewerProfile(), nil, false)
	require.NoError(t, err)
	g.SetCluster("unknown-cluster")
	assert.False(t, g.Allowed(ActionView, ResourceTopic, "anything"), "no profile, no default => deny")
}

func TestGateDefaultProfileFallback(t *testing.T) {
	cfg := viewerProfile()
	cfg.Default = &appconfig.Profile{
		Permissions: []appconfig.Permission{{Resource: "topic", Actions: []string{"view"}}},
	}
	g, err := NewGate(cfg, nil, false)
	require.NoError(t, err)

	g.SetCluster("some-other-cluster")
	assert.True(t, g.Allowed(ActionView, ResourceTopic, "x"), "default profile applies")
	assert.False(t, g.Allowed(ActionDelete, ResourceTopic, "x"))
	assert.Equal(t, "default", g.ActiveProfileName())
}

func TestGateUnnamedCheckMatchesPatternlessOnly(t *testing.T) {
	cfg := appconfig.AuthzSettings{
		Profiles: []appconfig.Profile{{
			Name:     "p",
			Clusters: []string{"c"},
			Permissions: []appconfig.Permission{
				{Resource: "topic", Name: "orders-.*", Actions: []string{"create"}},
			},
		}},
	}
	g, err := NewGate(cfg, nil, false)
	require.NoError(t, err)
	g.SetCluster("c")
	// create is an unnamed check (name unknown) — a pattern-bound permission does
	// not satisfy it.
	assert.False(t, g.Allowed(ActionCreate, ResourceTopic, ""))
}

func TestGateReadOnlyBlocksAltering(t *testing.T) {
	readonly := func(cluster string) bool { return cluster == "prod" }
	// Fully-permissive profile.
	cfg := appconfig.AuthzSettings{Profiles: []appconfig.Profile{{
		Name: "admin", Clusters: []string{"prod"},
		Permissions: []appconfig.Permission{{Resource: "topic", Actions: []string{"all"}}},
	}}}
	g, err := NewGate(cfg, readonly, false)
	require.NoError(t, err)
	g.SetCluster("prod")

	assert.NoError(t, g.Check(ActionView, ResourceTopic, "x"), "read allowed on read-only cluster")
	err = g.Check(ActionDelete, ResourceTopic, "x")
	require.Error(t, err)
	var ro api.ClusterReadOnlyError
	assert.ErrorAs(t, err, &ro, "altering denied with ClusterReadOnlyError even under admin profile")
}

func TestGateGlobalReadOnlyFlagOverridesProfile(t *testing.T) {
	cfg := appconfig.AuthzSettings{Profiles: []appconfig.Profile{{
		Name: "admin", Clusters: []string{"prod"},
		Permissions: []appconfig.Permission{{Resource: "topic", Actions: []string{"all"}}},
	}}}
	g, err := NewGate(cfg, nil, true) // forceReadOnly
	require.NoError(t, err)
	g.SetCluster("prod")

	err = g.Check(ActionCreate, ResourceTopic, "")
	var ro api.ClusterReadOnlyError
	assert.ErrorAs(t, err, &ro)
	assert.True(t, g.ReadOnly())
}

func TestGateReadOnlyWhenAuthzDisabled(t *testing.T) {
	g, err := NewGate(appconfig.AuthzSettings{}, nil, true)
	require.NoError(t, err)
	g.SetCluster("prod")
	assert.False(t, g.Enabled())
	err = g.Check(ActionDelete, ResourceTopic, "x")
	var ro api.ClusterReadOnlyError
	assert.True(t, errors.As(err, &ro), "read-only is independent of authz")
}

func TestNewGateValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  appconfig.AuthzSettings
		want string
	}{
		{
			name: "empty clusters",
			cfg:  appconfig.AuthzSettings{Profiles: []appconfig.Profile{{Name: "p", Permissions: []appconfig.Permission{{Resource: "topic", Actions: []string{"view"}}}}}},
			want: "at least one cluster",
		},
		{
			name: "missing resource",
			cfg:  appconfig.AuthzSettings{Profiles: []appconfig.Profile{{Name: "p", Clusters: []string{"c"}, Permissions: []appconfig.Permission{{Actions: []string{"view"}}}}}},
			want: "missing a resource",
		},
		{
			name: "no actions",
			cfg:  appconfig.AuthzSettings{Profiles: []appconfig.Profile{{Name: "p", Clusters: []string{"c"}, Permissions: []appconfig.Permission{{Resource: "topic"}}}}},
			want: "no actions",
		},
		{
			name: "unknown resource",
			cfg:  appconfig.AuthzSettings{Profiles: []appconfig.Profile{{Name: "p", Clusters: []string{"c"}, Permissions: []appconfig.Permission{{Resource: "widget", Actions: []string{"view"}}}}}},
			want: "unknown resource",
		},
		{
			name: "unknown action",
			cfg:  appconfig.AuthzSettings{Profiles: []appconfig.Profile{{Name: "p", Clusters: []string{"c"}, Permissions: []appconfig.Permission{{Resource: "topic", Actions: []string{"teleport"}}}}}},
			want: "not valid for resource",
		},
		{
			name: "invalid regex",
			cfg:  appconfig.AuthzSettings{Profiles: []appconfig.Profile{{Name: "p", Clusters: []string{"c"}, Permissions: []appconfig.Permission{{Resource: "topic", Name: "[", Actions: []string{"view"}}}}}},
			want: "invalid name pattern",
		},
		{
			name: "unknown active profile",
			cfg:  appconfig.AuthzSettings{ActiveProfile: "ghost", Profiles: []appconfig.Profile{{Name: "p", Clusters: []string{"c"}, Permissions: []appconfig.Permission{{Resource: "topic", Actions: []string{"view"}}}}}},
			want: "no such profile",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGate(tt.cfg, nil, false)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestGateEffectivePermissions(t *testing.T) {
	g, err := NewGate(viewerProfile(), nil, false)
	require.NoError(t, err)
	g.SetCluster("staging")
	perms := g.EffectivePermissions()
	require.NotEmpty(t, perms)
	// "all" on orders-.* expands to include delete and an implied view.
	var haveView, haveDelete bool
	for _, p := range perms {
		assert.Equal(t, "orders-.*", p.Pattern)
		if p.Action == ActionView {
			haveView = true
		}
		if p.Action == ActionDelete {
			haveDelete = true
		}
	}
	assert.True(t, haveView, "implied view present")
	assert.True(t, haveDelete, "all expands to delete")
}
