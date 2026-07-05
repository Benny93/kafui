package appconfig_view

import (
	"testing"

	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/authz"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/version"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestCommon builds a Common backed by the mock datasource, with a cluster
// extension carrying a secret property and redaction enabled.
func newTestCommon(t *testing.T) *core.Common {
	t.Helper()

	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	contexts, err := mockDS.GetContexts()
	require.NoError(t, err)
	require.NotEmpty(t, contexts)

	cfg := appconfig.Default()
	cfg.Redaction = appconfig.RedactionSettings{Enabled: true}
	cfg.Clusters = map[string]appconfig.ClusterExtension{
		contexts[0]: {
			ReadOnly: true,
			Properties: map[string]any{
				"sasl.password": "hunter2",
				"client.id":     "kafui",
			},
		},
	}

	common := core.NewCommon(mockDS)
	common.ApplyAppConfig(cfg)
	return common
}

func TestModel_View(t *testing.T) {
	common := newTestCommon(t)
	contexts, err := common.DataSource.GetContexts()
	require.NoError(t, err)

	m := NewModelWithCommon(common)
	m.SetDimensions(120, 40)
	// Render once so the viewport is populated.
	out := m.View()

	tests := []struct {
		name   string
		assert func(t *testing.T, doc string)
	}{
		{
			name: "contains version string",
			assert: func(t *testing.T, doc string) {
				assert.Contains(t, doc, version.Get().Version)
			},
		},
		{
			name: "contains a cluster name",
			assert: func(t *testing.T, doc string) {
				assert.Contains(t, doc, contexts[0])
			},
		},
		{
			name: "raw secret is absent",
			assert: func(t *testing.T, doc string) {
				assert.NotContains(t, doc, "hunter2")
			},
		},
		{
			name: "redaction placeholder present",
			assert: func(t *testing.T, doc string) {
				assert.Contains(t, doc, "**********")
			},
		},
		{
			name: "read-only notice present",
			assert: func(t *testing.T, doc string) {
				assert.Contains(t, doc, "read-only")
			},
		},
	}

	// Assert against the raw document (viewport rendering may truncate/wrap the
	// visible slice, so test the source-of-truth document builder directly).
	doc := buildDocument(common)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, doc)
		})
	}

	// Sanity: the rendered view is non-empty.
	assert.NotEmpty(t, out)
}

// TestWhoamiPermissionsRendered checks the effective-permissions ("whoami")
// section renders the resolved permission set including implied-view expansion
// (AA-12).
func TestWhoamiPermissionsRendered(t *testing.T) {
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")
	contexts, err := mockDS.GetContexts()
	require.NoError(t, err)
	require.NotEmpty(t, contexts)

	cfg := appconfig.Default()
	cfg.Authz = appconfig.AuthzSettings{Default: &appconfig.Profile{
		Name: "orders-admin",
		Permissions: []appconfig.Permission{
			{Resource: "topic", Name: "orders-.*", Actions: []string{"delete"}},
		},
	}}

	gate, err := authz.NewGate(cfg.Authz, nil, false)
	require.NoError(t, err)
	gate.SetCluster(contexts[0])

	common := core.NewCommon(mockDS)
	common.ApplyAppConfig(cfg)
	common.Gate = gate
	common.Identity = "tester"

	doc := buildDocument(common)
	assert.Contains(t, doc, "Permissions (whoami)")
	assert.Contains(t, doc, "tester")
	assert.Contains(t, doc, "orders-admin")
	assert.Contains(t, doc, "orders-.*")
	assert.Contains(t, doc, "delete")
	assert.Contains(t, doc, "view", "implied view expanded and shown")
}

func TestModel_PageInterface(t *testing.T) {
	common := newTestCommon(t)
	m := NewModelWithCommon(common)

	assert.Equal(t, "appconfig", m.GetID())
	assert.Equal(t, "Config", m.GetTitle())
	assert.NotEmpty(t, m.GetHelp())
	assert.Nil(t, m.OnFocus())
	assert.Nil(t, m.OnBlur())

	// esc triggers a back navigation command.
	page, cmd := m.HandleNavigation(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, m, page)
	require.NotNil(t, cmd)
	_, isBack := cmd().(core.BackMsg)
	assert.True(t, isBack)
}
