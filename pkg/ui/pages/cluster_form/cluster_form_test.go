package cluster_form

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/datasource/mock"
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
		}), appconfig.ClusterExtension{})
		require.NoError(t, err)
		require.NotNil(t, ext.SASL)
		assert.Equal(t, "PLAIN", ext.SASL.Mechanism)
		assert.Equal(t, "u", ext.SASL.Username)
		assert.Equal(t, "p", ext.SASL.Password)
		assert.Empty(t, ext.SASL.ClientID)
		assert.Equal(t, []string{"b1:9092", "b2:9092"}, ext.Brokers)
		assert.Empty(t, ext.SASL.DeviceAuthURL, "nothing stored, so nothing carried over")
	})

	for _, mech := range []string{"SCRAM-SHA-256", "SCRAM-SHA-512"} {
		t.Run(mech+" uses username/password", func(t *testing.T) {
			_, ext, err := candidateFromValues(base(map[string]string{
				fSaslMechanism: mech, fSaslUsername: "u", fSaslPassword: "p",
			}), appconfig.ClusterExtension{})
			require.NoError(t, err)
			require.NotNil(t, ext.SASL)
			assert.Equal(t, mech, ext.SASL.Mechanism)
			assert.Equal(t, "u", ext.SASL.Username)
		})
	}

	t.Run("AWS_MSK_IAM carries only the mechanism", func(t *testing.T) {
		_, ext, err := candidateFromValues(base(map[string]string{
			fSaslMechanism: "AWS_MSK_IAM", fSaslUsername: "ignored", fSaslPassword: "ignored",
			fSaslClientID: "ignored",
		}), appconfig.ClusterExtension{})
		require.NoError(t, err)
		require.NotNil(t, ext.SASL)
		assert.Equal(t, "AWS_MSK_IAM", ext.SASL.Mechanism)
		assert.Empty(t, ext.SASL.Username)
		assert.Empty(t, ext.SASL.Password)
		assert.Empty(t, ext.SASL.ClientID)
	})

	t.Run("OAUTHBEARER uses client credentials", func(t *testing.T) {
		_, ext, err := candidateFromValues(base(map[string]string{
			fSaslMechanism: "OAUTHBEARER", fSaslClientID: "cid",
			fSaslClientSecret: "sec", fSaslTokenURL: "http://token",
			fSaslUsername: "ignored",
		}), appconfig.ClusterExtension{})
		require.NoError(t, err)
		require.NotNil(t, ext.SASL)
		assert.Equal(t, "OAUTHBEARER", ext.SASL.Mechanism)
		assert.Equal(t, "cid", ext.SASL.ClientID)
		assert.Equal(t, "sec", ext.SASL.ClientSecret)
		assert.Equal(t, "http://token", ext.SASL.TokenURL)
		assert.Empty(t, ext.SASL.Username)
	})

	t.Run("none omits SASL", func(t *testing.T) {
		_, ext, err := candidateFromValues(base(map[string]string{fSaslMechanism: noneOption}), appconfig.ClusterExtension{})
		require.NoError(t, err)
		assert.Nil(t, ext.SASL)
	})

	t.Run("PLAINTEXT maps to empty security protocol", func(t *testing.T) {
		_, ext, err := candidateFromValues(map[string]string{
			fName: "c", fBrokers: "b:9092", fSecurityProtocol: "PLAINTEXT",
		}, appconfig.ClusterExtension{})
		require.NoError(t, err)
		assert.Empty(t, ext.SecurityProtocol)
	})

	t.Run("extension stubs and TLS", func(t *testing.T) {
		_, ext, err := candidateFromValues(map[string]string{
			fName: "c", fBrokers: "b:9092",
			fTLSCa: "/ca.pem", fTLSInsecure: "true",
			fSchemaURL: "http://sr", fConnectName: "kc", fConnectAddress: "http://connect",
			fKsqlURL: "http://ksql", fMetricsURL: "http://metrics", fReadOnly: "true",
		}, appconfig.ClusterExtension{})
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
		_, _, err := candidateFromValues(map[string]string{fBrokers: "b:9092"}, appconfig.ClusterExtension{})
		assert.Error(t, err)
	})
}

// Editing a cluster must not silently drop a credential the masked form leaves
// blank: ApplyCluster replaces the whole entry, so an empty submission has to
// mean "keep current".
func TestCandidateFromValues_BlankCredentialKeepsStored(t *testing.T) {
	stored := appconfig.ClusterExtension{
		SASL: &appconfig.SASLConfig{
			Mechanism:     "SCRAM-SHA-512",
			Username:      "svc",
			Password:      "stored-secret",
			ClientSecret:  "stored-client-secret",
			DeviceAuthURL: "https://device.example",
		},
		SchemaRegistryPassword: "stored-sr-secret",
	}

	t.Run("blank submissions keep every stored credential", func(t *testing.T) {
		_, ext, err := candidateFromValues(map[string]string{
			fName: "prod", fBrokers: "b:9092",
			fSecurityProtocol: "SASL_SSL", fSaslMechanism: "SCRAM-SHA-512",
			fSaslUsername: "svc", fSaslPassword: "", fSaslClientSecret: "",
			fSchemaPassword: "",
		}, stored)
		require.NoError(t, err)
		require.NotNil(t, ext.SASL)
		assert.Equal(t, "stored-secret", ext.SASL.Password)
		assert.Equal(t, "stored-sr-secret", ext.SchemaRegistryPassword)
		// deviceAuthURL has no form field; it must survive an edit regardless.
		assert.Equal(t, "https://device.example", ext.SASL.DeviceAuthURL)
	})

	t.Run("typed submissions replace", func(t *testing.T) {
		_, ext, err := candidateFromValues(map[string]string{
			fName: "prod", fBrokers: "b:9092",
			fSaslMechanism: "PLAIN", fSaslPassword: "new-secret",
			fSchemaPassword: "new-sr-secret",
		}, stored)
		require.NoError(t, err)
		require.NotNil(t, ext.SASL)
		assert.Equal(t, "new-secret", ext.SASL.Password)
		assert.Equal(t, "new-sr-secret", ext.SchemaRegistryPassword)
	})

	t.Run("OAUTHBEARER keeps the stored client secret", func(t *testing.T) {
		_, ext, err := candidateFromValues(map[string]string{
			fName: "prod", fBrokers: "b:9092",
			fSaslMechanism: "OAUTHBEARER", fSaslClientID: "cid",
			fSaslClientSecret: "", fSaslTokenURL: "https://token",
		}, stored)
		require.NoError(t, err)
		require.NotNil(t, ext.SASL)
		assert.Equal(t, "stored-client-secret", ext.SASL.ClientSecret)
	})

	t.Run("add mode has nothing to keep", func(t *testing.T) {
		_, ext, err := candidateFromValues(map[string]string{
			fName: "fresh", fBrokers: "b:9092", fSaslMechanism: "PLAIN", fSaslPassword: "",
		}, appconfig.ClusterExtension{})
		require.NoError(t, err)
		require.NotNil(t, ext.SASL)
		assert.Empty(t, ext.SASL.Password)
	})
}

func enabledCommon() *core.Common {
	c := core.NewCommon(nil)
	c.AppConfig.DynamicConfigEnabled = true
	return c
}

// secretCluster is an editable cluster carrying one credential per masked field.
func secretCluster() appconfig.ClusterExtension {
	return appconfig.ClusterExtension{
		Brokers:          []string{"b:9092"},
		SecurityProtocol: "SASL_SSL",
		SASL: &appconfig.SASLConfig{
			Mechanism:    "SCRAM-SHA-512",
			Username:     "svc-account",
			Password:     "hunter2-sasl-password",
			ClientSecret: "oauth-client-secret-value",
		},
		SchemaRegistryURL:      "http://sr:8081",
		SchemaRegistryUsername: "sr-user",
		SchemaRegistryPassword: "schema-registry-password-value",
	}
}

// The wizard must never paint a live credential on screen. Editing a cluster
// prefills every field, so each stored secret has to stay out of the rendered
// output while the non-secret basics remain visible.
func TestView_PrefilledSecretsAreNeverRendered(t *testing.T) {
	c := enabledCommon()
	c.AppConfig.Clusters["prod"] = secretCluster()

	m := NewModelWithCommon(c, "prod")
	require.False(t, m.disabled)
	m.SetDimensions(120, 40)

	out := m.View()
	for _, secret := range []string{
		"hunter2-sasl-password",
		"oauth-client-secret-value",
		"schema-registry-password-value",
	} {
		assert.NotContains(t, out, secret, "stored credential must not be rendered")
	}

	// Non-secret values still round-trip visibly, and a stored secret is
	// signalled by the mask marker rather than by its content.
	assert.Contains(t, out, "b:9092")
	assert.Contains(t, out, "svc-account")
	assert.Contains(t, out, "http://sr:8081")
	assert.Contains(t, out, appconfig.RedactPlaceholder())

	// Submitting the untouched form must not blank the credentials: ApplyCluster
	// replaces the whole entry, so the blank fields are filled from what is
	// already stored.
	m.savePath = filepath.Join(t.TempDir(), "kafui.yaml")
	require.NotNil(t, m.apply(m.form.Values()))

	saved, err := appconfig.Load(m.savePath)
	require.NoError(t, err)
	require.Contains(t, saved.Clusters, "prod")
	require.NotNil(t, saved.Clusters["prod"].SASL)
	assert.Equal(t, "hunter2-sasl-password", saved.Clusters["prod"].SASL.Password)
	assert.Equal(t, "schema-registry-password-value", saved.Clusters["prod"].SchemaRegistryPassword)
}

// An externalized ${env:VAR} reference is a pointer, not secret material, so the
// field displays it and an edit writes it back unchanged.
func TestView_ProviderRefReferenceRoundTrips(t *testing.T) {
	c := enabledCommon()
	ext := secretCluster()
	ext.SASL.Password = "${env:PROD_KAFKA_PASSWORD}"
	c.AppConfig.Clusters["prod"] = ext

	m := NewModelWithCommon(c, "prod")
	require.False(t, m.disabled)

	out := m.View()
	assert.Contains(t, out, "${env:PROD_KAFKA_PASSWORD}", "a reference is safe to show")
	assert.Equal(t, "${env:PROD_KAFKA_PASSWORD}", m.form.Values()[fSaslPassword])
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

	// Validation is the registry's run/refresh action (r, alias F5); it used to
	// be ctrl+v, a chord that appeared in no help text.
	m.Update(tea.KeyMsg{Type: tea.KeyF5})

	require.True(t, called, "F5 must invoke the AC-11 validation service")
	require.Contains(t, gotCandidate.Clusters, "c1")
	assert.Equal(t, []string{"b:9092"}, gotCandidate.Clusters["c1"].Brokers)
	require.NotNil(t, m.results)
	require.Len(t, m.results.Clusters, 1)
	assert.True(t, m.results.Clusters[0].Results[0].OK)
}

// failingReloadDS is a datasource whose in-place reload fails.
type failingReloadDS struct{ *mock.KafkaDataSourceMock }

func (failingReloadDS) Reload(appconfig.Config) error { return errors.New("broker unreachable") }

// A failed datasource reload used to be discarded (`_ = r.Reload(merged)`):
// the save reported success while the live cluster connection stayed stale.
func TestApply_ReloadFailureIsSurfaced(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	c := core.NewCommon(failingReloadDS{ds})
	c.AppConfig.DynamicConfigEnabled = true
	m := NewModelWithCommon(c, "")
	m.savePath = filepath.Join(t.TempDir(), "kafui.yaml")

	cmd := m.apply(map[string]string{fName: "c1", fBrokers: "b:9092"})
	require.NotNil(t, cmd)
	batch, ok := cmd().(tea.BatchMsg)
	require.True(t, ok)

	var notif core.NotificationMsg
	var back bool
	for _, c := range batch {
		if c == nil {
			continue
		}
		switch msg := c().(type) {
		case core.NotificationMsg:
			notif = msg
		case core.BackMsg:
			back = true
		}
	}
	assert.Equal(t, core.StatusError, notif.Severity)
	assert.Contains(t, notif.Message, "broker unreachable")
	assert.True(t, back, "the config is saved, so the wizard still closes")
}
