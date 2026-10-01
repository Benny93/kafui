package connector

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/authz"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/components/menu"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readOnlyCommon forces global read-only mode, denying every altering connector
// action (pause / restart / edit / reset / delete) while leaving refresh open.
func readOnlyCommon(t *testing.T) *core.Common {
	t.Helper()
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	g, err := authz.NewGate(appconfig.AuthzSettings{}, nil, true)
	require.NoError(t, err)
	g.SetCluster("prod")
	return &core.Common{DataSource: ds, Styles: stylesPkg.DefaultStyles(), Gate: g}
}

func labelsOf(entries []menu.Entry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Label)
	}
	return out
}

func TestConnectorControls_KeyScope(t *testing.T) {
	m := newModel(testCommon(), "connect-primary", "orders-source")
	assert.Equal(t, keys.ScopeConnector, m.KeyScope())
}

func TestConnectorControls_IsInputMode(t *testing.T) {
	m := newModel(testCommon(), "connect-primary", "orders-source")
	assert.False(t, m.IsInputMode())
	m.editing = true
	assert.True(t, m.IsInputMode(), "the focused config editor captures keystrokes")
}

func TestConnectorControls_Unwind(t *testing.T) {
	m := newModel(testCommon(), "connect-primary", "orders-source")
	m.editing = true
	cmd, ok := m.Unwind()
	assert.True(t, ok)
	assert.Nil(t, cmd)
	assert.False(t, m.editing)

	m.expandedTask = 1
	cmd, ok = m.Unwind()
	assert.True(t, ok)
	assert.Nil(t, cmd)
	assert.Equal(t, -1, m.expandedTask)

	cmd, ok = m.Unwind()
	assert.False(t, ok, "at rest there is nothing to unwind")
	assert.Nil(t, cmd)
}

func TestConnectorControls_ActionsEnabledWhenPermitted(t *testing.T) {
	m := newModel(testCommon(), "connect-primary", "orders-source")
	entries := m.ContextActions()
	assert.Len(t, entries, 10)
	for _, e := range entries {
		assert.False(t, e.Disabled, "a permitted profile enables %q", e.Label)
	}
	// A running connector offers Pause, not Resume.
	assert.Contains(t, labelsOf(entries), "Pause connector")
}

func TestConnectorControls_ActionsResumeLabelWhenPaused(t *testing.T) {
	m := newModel(testCommon(), "connect-primary", "orders-source")
	m.details.State = api.ConnectorStatePaused
	assert.Contains(t, labelsOf(m.ContextActions()), "Resume connector")
}

func TestConnectorControls_ActionsGatedWhenReadOnly(t *testing.T) {
	m := newModel(readOnlyCommon(t), "connect-primary", "orders-source")
	entries := m.ContextActions()

	byLabel := map[string]bool{}
	for _, e := range entries {
		if e.Disabled {
			assert.NotEqual(t, "", e.Reason, "a disabled entry must state why: %q", e.Label)
		}
		byLabel[e.Label] = e.Disabled
	}

	// Every altering action is denied under read-only mode.
	for _, l := range []string{
		"Pause connector", "Restart connector", "Stop connector",
		"Restart the selected task", "Restart all tasks", "Restart failed tasks",
		"Edit configuration", "Reset offsets", "Delete connector",
	} {
		assert.True(t, byLabel[l], "%q is gated", l)
	}
	assert.False(t, byLabel["Refresh"], "refresh is not gated")

	// The destructive entries are flagged so they route through confirmation.
	for _, e := range entries {
		if e.Label == "Reset offsets" || e.Label == "Delete connector" {
			assert.True(t, e.Destructive, "%q must be flagged destructive", e.Label)
		}
	}
}

// Every advertised action must be runnable without panicking or reaching the
// network: each closure only builds a confirmation dialog or a command, and the
// datasource is touched solely inside the not-yet-run OnConfirm handler.
func TestConnectorControls_ActionRunsAreWired(t *testing.T) {
	for _, state := range []string{api.ConnectorStateRunning, api.ConnectorStatePaused} {
		m := newModel(testCommon(), "connect-primary", "orders-source")
		m.details.State = state
		for _, e := range m.ContextActions() {
			require.NotNil(t, e.Run, "action %q has no handler", e.Label)
			_ = e.Run()
		}
	}
}
