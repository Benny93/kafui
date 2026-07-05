package datasource

import (
	"context"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/audit"
	"github.com/Benny93/kafui/pkg/authz"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// spyDS embeds the full mock to satisfy api.KafkaDataSource, overriding just the
// methods we assert on so we can observe whether the guard delegated.
type spyDS struct {
	*mock.KafkaDataSourceMock
	ctx           string
	deleteCalled  bool
	produceCalled bool
	topicNames    []string
}

func newSpy() *spyDS {
	return &spyDS{KafkaDataSourceMock: &mock.KafkaDataSourceMock{}, ctx: "prod", topicNames: []string{"orders-eu", "payments"}}
}

func (s *spyDS) GetContext() string        { return s.ctx }
func (s *spyDS) SetContext(n string) error { s.ctx = n; return nil }
func (s *spyDS) DeleteTopic(name string) error {
	s.deleteCalled = true
	return nil
}
func (s *spyDS) ProduceMessage(ctx context.Context, topic string, rec api.ProduceRecord) error {
	s.produceCalled = true
	return nil
}
func (s *spyDS) GetTopicNames() ([]string, error) { return s.topicNames, nil }

// recWriter captures audit records.
type recWriter struct{ records []audit.Record }

func (w *recWriter) Write(r audit.Record) error { w.records = append(w.records, r); return nil }

func adminGate(t *testing.T, forceReadOnly bool) *authz.Gate {
	t.Helper()
	cfg := appconfig.AuthzSettings{Profiles: []appconfig.Profile{{
		Name: "admin", Clusters: []string{"prod"},
		Permissions: []appconfig.Permission{{Resource: "topic", Actions: []string{"all"}}},
	}}}
	g, err := authz.NewGate(cfg, nil, forceReadOnly)
	require.NoError(t, err)
	g.SetCluster("prod")
	return g
}

func viewerGate(t *testing.T) *authz.Gate {
	t.Helper()
	cfg := appconfig.AuthzSettings{Profiles: []appconfig.Profile{{
		Name: "viewer", Clusters: []string{"prod"},
		Permissions: []appconfig.Permission{{Resource: "topic", Name: "orders-.*", Actions: []string{"view"}}},
	}}}
	g, err := authz.NewGate(cfg, nil, false)
	require.NoError(t, err)
	g.SetCluster("prod")
	return g
}

func TestGuardDelegatesAllowedWriteAndAudits(t *testing.T) {
	spy := newSpy()
	w := &recWriter{}
	svc := audit.NewService(true, audit.LevelAll, w, nil)
	g := NewGuard(spy, adminGate(t, false), svc)

	err := g.DeleteTopic("orders-eu")
	require.NoError(t, err)
	assert.True(t, spy.deleteCalled, "allowed write delegates to inner")
	require.Len(t, w.records, 1, "exactly one audit record")
	assert.Equal(t, "DeleteTopic", w.records[0].Operation)
	assert.Equal(t, audit.ResultSuccess, w.records[0].Result)
	assert.Equal(t, "prod", w.records[0].Cluster)
}

func TestGuardDeniedWriteShortCircuits(t *testing.T) {
	spy := newSpy()
	w := &recWriter{}
	svc := audit.NewService(true, audit.LevelAll, w, nil)
	g := NewGuard(spy, viewerGate(t), svc)

	err := g.DeleteTopic("orders-eu")
	require.Error(t, err)
	var denied api.AccessDeniedError
	assert.ErrorAs(t, err, &denied)
	assert.False(t, spy.deleteCalled, "denied write never reaches inner (no effect)")
	require.Len(t, w.records, 1)
	assert.Equal(t, audit.ResultAccessDenied, w.records[0].Result)
}

func TestGuardReadOnlyBlocksWriteWithClusterReadOnlyError(t *testing.T) {
	spy := newSpy()
	w := &recWriter{}
	svc := audit.NewService(true, audit.LevelAll, w, nil)
	g := NewGuard(spy, adminGate(t, true), svc) // forceReadOnly

	err := g.ProduceMessage(context.Background(), "orders-eu", api.ProduceRecord{})
	require.Error(t, err)
	var ro api.ClusterReadOnlyError
	assert.ErrorAs(t, err, &ro)
	assert.False(t, spy.produceCalled)
	require.Len(t, w.records, 1)
	assert.Equal(t, audit.ResultAccessDenied, w.records[0].Result)
}

func TestGuardListingFilteredByViewPermission(t *testing.T) {
	spy := newSpy() // returns orders-eu, payments
	g := NewGuard(spy, viewerGate(t), nil)

	names, err := g.GetTopicNames()
	require.NoError(t, err)
	assert.Equal(t, []string{"orders-eu"}, names, "only view-permitted topics survive")
}

func TestGuardDisabledGateNoFiltering(t *testing.T) {
	spy := newSpy()
	g, err := authz.NewGate(appconfig.AuthzSettings{}, nil, false)
	require.NoError(t, err)
	guard := NewGuard(spy, g, nil)

	names, ferr := guard.GetTopicNames()
	require.NoError(t, ferr)
	assert.ElementsMatch(t, []string{"orders-eu", "payments"}, names, "authz disabled => all listed")
}

func TestGuardAlterOnlyLevelSkipsReads(t *testing.T) {
	spy := newSpy()
	w := &recWriter{}
	svc := audit.NewService(true, audit.LevelAlterOnly, w, nil)
	g := NewGuard(spy, adminGate(t, false), svc)

	// A successful delete is altering => recorded.
	require.NoError(t, g.DeleteTopic("orders-eu"))
	assert.Len(t, w.records, 1)
}

func TestGuardSetContextReResolvesGate(t *testing.T) {
	spy := newSpy()
	cfg := appconfig.AuthzSettings{Profiles: []appconfig.Profile{
		{Name: "prod-viewer", Clusters: []string{"prod"}, Permissions: []appconfig.Permission{{Resource: "topic", Actions: []string{"view"}}}},
		{Name: "stg-admin", Clusters: []string{"staging"}, Permissions: []appconfig.Permission{{Resource: "topic", Actions: []string{"all"}}}},
	}}
	gate, err := authz.NewGate(cfg, nil, false)
	require.NoError(t, err)
	g := NewGuard(spy, gate, nil)

	assert.Equal(t, "prod-viewer", gate.ActiveProfileName())
	require.NoError(t, g.SetContext("staging"))
	assert.Equal(t, "stg-admin", gate.ActiveProfileName(), "context switch re-resolves the active profile")
	assert.True(t, gate.Allowed(authz.ActionDelete, authz.ResourceTopic, "x"), "staging admin can delete")
}
