package broker

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/authz"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/components/menu"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/keys"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readOnlyCommon is a Common whose gate forces global read-only mode, so every
// altering action (config edit, replica move) is denied while read actions pass.
func readOnlyCommon(t *testing.T) *core.Common {
	t.Helper()
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	g, err := authz.NewGate(appconfig.AuthzSettings{}, nil, true)
	require.NoError(t, err)
	g.SetCluster("prod")
	return &core.Common{DataSource: ds, Styles: stylesPkg.DefaultStyles(), Gate: g}
}

// labelsOf collects entry labels for membership assertions.
func labelsOf(entries []menu.Entry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Label)
	}
	return out
}

func TestBrokerControls_KeyScope(t *testing.T) {
	m := newModel(testCommon(), 1, api.BrokerInfo{ID: 1}, true)
	assert.Equal(t, keys.ScopeListContent, m.KeyScope())
}

func TestBrokerControls_IsInputMode(t *testing.T) {
	m := newModel(testCommon(), 1, api.BrokerInfo{ID: 1}, true)
	assert.False(t, m.IsInputMode(), "idle broker page is not in input mode")

	m.searching = true
	assert.True(t, m.IsInputMode(), "an open config search captures keystrokes")
	m.searching = false

	m.editing = true
	assert.True(t, m.IsInputMode(), "an inline config edit captures keystrokes")
	m.editing = false

	m.moveForm = form.New(nil)
	assert.True(t, m.IsInputMode(), "the reassignment form captures keystrokes")
	m.moveForm = nil
	assert.False(t, m.IsInputMode())
}

func TestBrokerControls_Unwind(t *testing.T) {
	// Esc collapses the search first.
	m := newModel(testCommon(), 1, api.BrokerInfo{ID: 1}, true)
	m.searching = true
	cmd, ok := m.Unwind()
	assert.True(t, ok)
	assert.Nil(t, cmd)
	assert.False(t, m.searching)

	// Then an expanded log directory.
	m = newModel(testCommon(), 1, api.BrokerInfo{ID: 1}, true)
	m.expanded = 2
	cmd, ok = m.Unwind()
	assert.True(t, ok)
	assert.Nil(t, cmd)
	assert.Equal(t, -1, m.expanded)

	// Nothing to unwind: the shell navigates back instead.
	cmd, ok = m.Unwind()
	assert.False(t, ok, "at rest there is nothing to unwind")
	assert.Nil(t, cmd)
}

func TestBrokerControls_ActionsEnabledWhenPermitted(t *testing.T) {
	m := newModel(testCommon(), 1, api.BrokerInfo{ID: 1}, true)
	entries := m.ContextActions()

	// Base entries: three tabs + edit + search + refresh. expanded < 0 so the
	// move entry is absent.
	assert.Len(t, entries, 6)
	for _, e := range entries {
		assert.False(t, e.Disabled, "a permitted profile enables %q", e.Label)
		assert.Equal(t, "", e.Reason)
	}
	assert.NotContains(t, labelsOf(entries), "Move the selected replica to another log directory…")
}

func TestBrokerControls_ActionsMoveEntryAppearsWhenExpanded(t *testing.T) {
	m := newModel(testCommon(), 1, api.BrokerInfo{ID: 1}, true)
	m.expanded = 0
	entries := m.ContextActions()
	assert.Len(t, entries, 7, "expanding a directory adds the move action")
	assert.Contains(t, labelsOf(entries), "Move the selected replica to another log directory…")
}

func TestBrokerControls_ActionsGatedWhenReadOnly(t *testing.T) {
	m := newModel(readOnlyCommon(t), 1, api.BrokerInfo{ID: 1}, true)
	m.expanded = 0
	entries := m.ContextActions()

	byLabel := map[string]bool{}
	for _, e := range entries {
		if e.Disabled {
			assert.NotEqual(t, "", e.Reason, "a disabled entry must state why: %q", e.Label)
		}
		byLabel[e.Label] = e.Disabled
	}

	// Altering actions are denied under read-only mode.
	assert.True(t, byLabel["Edit the selected configuration entry"], "edit is gated")
	assert.True(t, byLabel["Move the selected replica to another log directory…"], "move is gated")
	// Read / navigation actions stay available.
	assert.False(t, byLabel["Log directories"], "tab switch is not gated")
	assert.False(t, byLabel["Search configuration"], "search is not gated")
	assert.False(t, byLabel["Refresh"], "refresh is not gated")
}

// Every advertised action must be runnable without panicking or reaching the
// network: the closures only build commands (tab switches, forms, confirmations).
func TestBrokerControls_ActionRunsAreWired(t *testing.T) {
	m := newModel(testCommon(), 1, api.BrokerInfo{ID: 1}, true)
	m.expanded = 0
	entries := m.ContextActions()
	for _, e := range entries {
		require.NotNil(t, e.Run, "action %q has no handler", e.Label)
		_ = e.Run()
	}

	// Spot-check the two closures that mutate view state.
	fresh := newModel(testCommon(), 1, api.BrokerInfo{ID: 1}, true)
	for _, e := range fresh.ContextActions() {
		switch e.Label {
		case "Metrics":
			e.Run()
			assert.Equal(t, tabMetrics, fresh.active)
		case "Search configuration":
			e.Run()
			assert.True(t, fresh.searching)
		}
	}
}
