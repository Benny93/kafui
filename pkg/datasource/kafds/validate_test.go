package kafds

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/IBM/sarama"
	"github.com/birdayz/kaf/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingFactory records how many admin clients it was asked to create so a
// test can assert that a probe short-circuits before attempting a connection.
type countingFactory struct {
	admin     ClusterAdminInterface
	failAdmin bool
	calls     int
}

func (f *countingFactory) CreateClusterAdmin(_ []string, _ *sarama.Config) (ClusterAdminInterface, error) {
	f.calls++
	if f.failAdmin {
		return nil, errors.New("dial tcp: connection refused")
	}
	return f.admin, nil
}

func (f *countingFactory) CreateClient(_ []string, _ *sarama.Config) (sarama.Client, error) {
	return nil, nil
}

func candidateWith(name string, ext appconfig.ClusterExtension) appconfig.Config {
	return appconfig.Config{Clusters: map[string]appconfig.ClusterExtension{name: ext}}
}

func TestValidateCandidate_BrokerOK(t *testing.T) {
	f := &countingFactory{admin: &MockClusterAdmin{}}
	kds := NewKafkaDataSourceKafWithDeps(f, &MockConfigManager{})

	report := kds.ValidateCandidate(context.Background(), candidateWith("c1", appconfig.ClusterExtension{
		Brokers: []string{"localhost:9092"},
	}))

	require.Len(t, report.Clusters, 1)
	assert.Equal(t, "c1", report.Clusters[0].Cluster)
	require.Len(t, report.Clusters[0].Results, 1)
	assert.Equal(t, "broker", report.Clusters[0].Results[0].Component)
	assert.True(t, report.Clusters[0].Results[0].OK)
	assert.Equal(t, 1, f.calls)
}

func TestValidateCandidate_BrokerFail(t *testing.T) {
	f := &countingFactory{admin: &MockClusterAdmin{ShouldFailListTopics: true}}
	kds := NewKafkaDataSourceKafWithDeps(f, &MockConfigManager{})

	report := kds.ValidateCandidate(context.Background(), candidateWith("c1", appconfig.ClusterExtension{
		Brokers: []string{"localhost:9092"},
	}))

	require.Len(t, report.Clusters, 1)
	res := report.Clusters[0].Results
	require.Len(t, res, 1)
	assert.Equal(t, "broker", res[0].Component)
	assert.False(t, res[0].OK)
	assert.NotEmpty(t, res[0].Err)
}

func TestValidateCandidate_BrokerAdminCreationFails(t *testing.T) {
	f := &countingFactory{failAdmin: true}
	kds := NewKafkaDataSourceKafWithDeps(f, &MockConfigManager{})

	report := kds.ValidateCandidate(context.Background(), candidateWith("c1", appconfig.ClusterExtension{
		Brokers: []string{"localhost:9092"},
	}))

	require.Len(t, report.Clusters, 1)
	assert.False(t, report.Clusters[0].Results[0].OK)
}

func TestValidateCandidate_SchemaRegistryOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/subjects", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	f := &countingFactory{admin: &MockClusterAdmin{}}
	kds := NewKafkaDataSourceKafWithDeps(f, &MockConfigManager{})

	report := kds.ValidateCandidate(context.Background(), candidateWith("c1", appconfig.ClusterExtension{
		Brokers:           []string{"localhost:9092"},
		SchemaRegistryURL: srv.URL,
	}))

	require.Len(t, report.Clusters, 1)
	res := report.Clusters[0].Results
	require.Len(t, res, 2)
	assert.Equal(t, "schema-registry", res[1].Component)
	assert.True(t, res[1].OK, res[1].Err)
}

func TestValidateCandidate_SchemaRegistryFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	f := &countingFactory{admin: &MockClusterAdmin{}}
	kds := NewKafkaDataSourceKafWithDeps(f, &MockConfigManager{})

	report := kds.ValidateCandidate(context.Background(), candidateWith("c1", appconfig.ClusterExtension{
		Brokers:           []string{"localhost:9092"},
		SchemaRegistryURL: srv.URL,
	}))

	res := report.Clusters[0].Results
	require.Len(t, res, 2)
	assert.Equal(t, "schema-registry", res[1].Component)
	assert.False(t, res[1].OK)
	assert.NotEmpty(t, res[1].Err)
}

func TestValidateCandidate_UnreadableTruststoreShortCircuits(t *testing.T) {
	f := &countingFactory{admin: &MockClusterAdmin{}}
	kds := NewKafkaDataSourceKafWithDeps(f, &MockConfigManager{})

	report := kds.ValidateCandidate(context.Background(), candidateWith("c1", appconfig.ClusterExtension{
		Brokers: []string{"localhost:9092"},
		TLS:     &appconfig.TLSConfig{CAPath: "/nonexistent/ca.pem"},
	}))

	require.Len(t, report.Clusters, 1)
	res := report.Clusters[0].Results
	require.Len(t, res, 1)
	assert.Equal(t, "tls", res[0].Component)
	assert.False(t, res[0].OK)
	// No connection must be attempted when the TLS material fails to load.
	assert.Equal(t, 0, f.calls, "broker probe must not run after TLS load failure")
}

func TestValidateCandidate_EmptyCandidate(t *testing.T) {
	f := &countingFactory{admin: &MockClusterAdmin{}}
	kds := NewKafkaDataSourceKafWithDeps(f, &MockConfigManager{})

	report := kds.ValidateCandidate(context.Background(), appconfig.Config{})

	assert.Empty(t, report.Clusters)
	assert.Equal(t, 0, f.calls)
}

// TestReload_KafConfigUntouched mirrors TestKafkaDataSourceKaf_SetContext_NoWrite:
// applying an in-UI config change (Reload) must never read or write ~/.kaf/config.
func TestReload_KafConfigUntouched(t *testing.T) {
	dir := t.TempDir()
	kafPath := filepath.Join(dir, "config")
	original := []byte("current-cluster: existing\nclusters:\n- name: existing\n  brokers: [localhost:9092]\n")
	require.NoError(t, os.WriteFile(kafPath, original, 0o644))

	mockCM := &MockConfigManager{}
	kds := NewKafkaDataSourceKafWithDeps(&MockKafkaClientFactory{}, mockCM)

	// Seed the in-memory kaf config with the existing cluster.
	cfg = config.Config{
		CurrentCluster: "existing",
		Clusters:       []*config.Cluster{{Name: "existing", Brokers: []string{"localhost:9092"}}},
	}
	currentCluster = &config.Cluster{Name: "existing", Brokers: []string{"localhost:9092"}}

	effective := appconfig.Config{Clusters: map[string]appconfig.ClusterExtension{
		"newcluster": {Brokers: []string{"broker:9092"}},
	}}
	require.NoError(t, kds.Reload(effective))

	// The new, fully-kafui-defined cluster is merged in memory.
	names := map[string]bool{}
	for _, c := range cfg.Clusters {
		names[c.Name] = true
	}
	assert.True(t, names["existing"])
	assert.True(t, names["newcluster"])

	// The kaf file on disk is byte-for-byte unchanged and was never read.
	after, err := os.ReadFile(kafPath)
	require.NoError(t, err)
	assert.Equal(t, original, after, "~/.kaf/config must be untouched across an apply")
	assert.Equal(t, 0, mockCM.ReadConfigCallCount, "Reload must not read config from disk")
}
