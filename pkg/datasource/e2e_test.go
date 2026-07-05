package datasource

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/audit"
	"github.com/Benny93/kafui/pkg/authz"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readAudit reads and parses all records from an audit JSONL file.
func readAudit(t *testing.T, path string) []audit.Record {
	t.Helper()
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)
	defer f.Close()
	var out []audit.Record
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var r audit.Record
		require.NoError(t, json.Unmarshal(sc.Bytes(), &r))
		out = append(out, r)
	}
	return out
}

// newStack wires the real mock datasource behind a guard with the given profile
// config, forceReadOnly flag, and audit level, writing audit to a temp file.
func newStack(t *testing.T, cfg appconfig.AuthzSettings, forceReadOnly bool, level audit.Level) (*Guard, string) {
	t.Helper()
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	gate, err := authz.NewGate(cfg, nil, forceReadOnly)
	require.NoError(t, err)
	gate.SetCluster(ds.GetContext())

	path := filepath.Join(t.TempDir(), "audit.log")
	w, err := audit.NewFileWriter(path)
	require.NoError(t, err)
	svc := audit.NewService(true, level, w, nil)

	return NewGuard(ds, gate, svc), path
}

// defaultProfile builds a config whose default profile grants the given topic
// actions on every cluster (so we needn't know the mock's context name).
func defaultProfile(actions ...string) appconfig.AuthzSettings {
	return appconfig.AuthzSettings{Default: &appconfig.Profile{
		Name:        "default",
		Permissions: []appconfig.Permission{{Resource: "topic", Actions: actions}},
	}}
}

func TestE2ERestrictiveProfileDeniesMutationsAndAudits(t *testing.T) {
	g, auditPath := newStack(t, defaultProfile("view", "read messages"), false, audit.LevelAll)

	err := g.DeleteTopic("some-topic")
	var denied api.AccessDeniedError
	assert.ErrorAs(t, err, &denied, "viewer profile denies delete")

	err = g.ProduceMessage(context.Background(), "some-topic", api.ProduceRecord{})
	assert.ErrorAs(t, err, &denied, "viewer profile denies produce")

	records := readAudit(t, auditPath)
	require.Len(t, records, 2, "both denied attempts audited")
	for _, r := range records {
		assert.Equal(t, audit.ResultAccessDenied, r.Result)
	}
}

func TestE2EReadOnlyClusterBlocksAltering(t *testing.T) {
	g, auditPath := newStack(t, defaultProfile("all"), true /* --read-only */, audit.LevelAll)

	err := g.DeleteTopic("some-topic")
	var ro api.ClusterReadOnlyError
	assert.ErrorAs(t, err, &ro, "read-only overrides the all-permissive profile")

	records := readAudit(t, auditPath)
	require.Len(t, records, 1)
	assert.Equal(t, audit.ResultAccessDenied, records[0].Result)
}

func TestE2EAlterOnlyLevelSkipsReads(t *testing.T) {
	g, auditPath := newStack(t, defaultProfile("all"), false, audit.LevelAlterOnly)

	// A read passes through and must NOT be audited at alter_only.
	_, _ = g.GetTopics()
	// A denied-would-be-altering op IS audited (delete is altering).
	g.SetContext(g.GetContext()) // no-op, keeps gate resolved
	_ = g.DeleteTopic("some-topic")

	records := readAudit(t, auditPath)
	for _, r := range records {
		assert.NotEqual(t, "GetTopics", r.Operation, "reads skipped at alter_only")
	}
}
