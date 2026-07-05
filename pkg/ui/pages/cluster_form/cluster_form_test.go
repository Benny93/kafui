package cluster_form

import (
	"context"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/ui/core"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCandidateFromValues_SASLMappings(t *testing.T) {
	base := func(extra map[string]string) map[string]string {
		v := map[string]string{
			fName:             "prod",
			fBrokers:          "b1:9092, b2:9092",
			fSecurityProtocol: "SASL_SSL",
		}
		for k, val := range extra {
			v[k] = val
		}
		return v
	}

	t.Run("PLAIN uses username/password", func(t *testing.T) {
		_, ext, err := candidateFromValues(base(map[string]string{
			fSaslMechanism: "PLAIN", fSaslUsername: "u", fSaslPassword: "p",
			fSaslClientID: "ignored",
		}))
		require.NoError(t, err)
		require.NotNil(t, ext.SASL)
		assert.Equal(t, "PLAIN", ext.SASL.Mechanism)
		assert.Equal(t, "u", ext.SASL.Username)
		assert.Equal(t, "p", ext.SASL.Password)
		assert.Empty(t, ext.SASL.ClientID)
		assert.Equal(t, []string{"b1:9092", "b2:9092"}, ext.Brokers)
	})

	for _, mech := range []string{"SCRAM-SHA-256", "SCRAM-SHA-512"} {
		t.Run(mech+" uses username/password", func(t *testing.T) {
			_, ext, err := candidateFromValues(base(map[string]string{
				fSaslMechanism: mech, fSaslUsername: "u", fSaslPassword: "p",
			}))
			require.NoError(t, err)
			require.NotNil(t, ext.SASL)
			assert.Equal(t, mech, ext.SASL.Mechanism)
			assert.Equal(t, "u", ext.SASL.Username)
		})
	}

	t.Run("OAUTHBEARER uses client credentials", func(t *testing.T) {
		_, ext, err := candidateFromValues(base(map[string]string{
			fSaslMechanism: "OAUTHBEARER", fSaslClientID: "cid",
			fSaslClientSecret: "sec", fSaslTokenURL: "http://token",
			fSaslUsername: "ignored",
		}))
		require.NoError(t, err)
		require.NotNil(t, ext.SASL)
		assert.Equal(t, "OAUTHBEARER", ext.SASL.Mechanism)
		assert.Equal(t, "cid", ext.SASL.ClientID)
		assert.Equal(t, "sec", ext.SASL.ClientSecret)
		assert.Equal(t, "http://token", ext.SASL.TokenURL)
		assert.Empty(t, ext.SASL.Username)
	})

	t.Run("none omits SASL", func(t *testing.T) {
		_, ext, err := candidateFromValues(base(map[string]string{fSaslMechanism: noneOption}))
		require.NoError(t, err)
		assert.Nil(t, ext.SASL)
	})

	t.Run("PLAINTEXT maps to empty security protocol", func(t *testing.T) {
		_, ext, err := candidateFromValues(map[string]string{
			fName: "c", fBrokers: "b:9092", fSecurityProtocol: "PLAINTEXT",
		})
		require.NoError(t, err)
		assert.Empty(t, ext.SecurityProtocol)
	})

	t.Run("extension stubs and TLS", func(t *testing.T) {
		_, ext, err := candidateFromValues(map[string]string{
			fName: "c", fBrokers: "b:9092",
			fTLSCa: "/ca.pem", fTLSInsecure: "true",
			fSchemaURL: "http://sr", fConnectName: "kc", fConnectAddress: "http://connect",
			fKsqlURL: "http://ksql", fMetricsURL: "http://metrics", fReadOnly: "true",
		})
		require.NoError(t, err)
		assert.True(t, ext.ReadOnly)
		require.NotNil(t, ext.TLS)
		assert.Equal(t, "/ca.pem", ext.TLS.CAPath)
		assert.True(t, ext.TLS.Insecure)
		assert.Equal(t, "http://sr", ext.SchemaRegistryURL)
		require.Len(t, ext.Connect, 1)
		assert.Equal(t, "kc", ext.Connect[0].Name)
		require.NotNil(t, ext.Ksql)
		assert.Equal(t, "http://ksql", ext.Ksql.URL)
		assert.Equal(t, "http://metrics", ext.Metrics["url"])
	})

	t.Run("empty name is an error", func(t *testing.T) {
		_, _, err := candidateFromValues(map[string]string{fBrokers: "b:9092"})
		assert.Error(t, err)
	})
}

func enabledCommon() *core.Common {
	c := core.NewCommon(nil)
	c.AppConfig.DynamicConfigEnabled = true
	return c
}

func TestNewModel_DisabledToggleRejects(t *testing.T) {
	c := core.NewCommon(nil) // Default() ⇒ DynamicConfigEnabled false
	m := NewModelWithCommon(c, "")

	assert.True(t, m.disabled)

	// Init surfaces an explanatory error notification.
	cmd := m.Init()
	require.NotNil(t, cmd)
	msg := cmd()
	notif, ok := msg.(core.NotificationMsg)
	require.True(t, ok)
	assert.Equal(t, core.StatusError, notif.Severity)
	assert.Contains(t, notif.Message, "dynamicConfigEnabled")

	// Any key navigates back rather than editing.
	_, keyCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, keyCmd)
	_, isBack := keyCmd().(core.BackMsg)
	assert.True(t, isBack)
}

func TestValidateAction_InvokesService(t *testing.T) {
	c := enabledCommon()
	c.AppConfig.Clusters["c1"] = appconfig.ClusterExtension{Brokers: []string{"b:9092"}}

	m := NewModelWithCommon(c, "c1")
	require.False(t, m.disabled)

	var gotCandidate appconfig.Config
	called := false
	m.validate = func(_ context.Context, candidate appconfig.Config) api.ValidationReport {
		called = true
		gotCandidate = candidate
		return api.ValidationReport{Clusters: []api.ClusterValidation{
			{Cluster: "c1", Results: []api.ValidationResult{{Component: "broker", OK: true}}},
		}}
	}

	m.Update(tea.KeyMsg{Type: tea.KeyCtrlV})

	require.True(t, called, "ctrl+v must invoke the AC-11 validation service")
	require.Contains(t, gotCandidate.Clusters, "c1")
	assert.Equal(t, []string{"b:9092"}, gotCandidate.Clusters["c1"].Brokers)
	require.NotNil(t, m.results)
	require.Len(t, m.results.Clusters, 1)
	assert.True(t, m.results.Clusters[0].Results[0].OK)
}
