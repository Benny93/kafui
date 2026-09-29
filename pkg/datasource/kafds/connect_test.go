package kafds

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withConnect overrides the Connect cluster resolver for the duration of the
// test, restoring the previous resolver afterwards.
func withConnect(t *testing.T, clusters ...appconfig.ConnectCluster) {
	t.Helper()
	prev := loadConnectClusters
	loadConnectClusters = func(context string) []appconfig.ConnectCluster { return clusters }
	t.Cleanup(func() { loadConnectClusters = prev })
}

func TestConnectClient_Verbs_AuthAndContentType(t *testing.T) {
	var gotMethod, gotAuth, gotAccept, gotContentType, gotBody, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		gotContentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c, err := newConnectClient(appconfig.ConnectCluster{
		Name: "c1", Address: srv.URL, Username: "u", Password: "p",
	})
	require.NoError(t, err)

	t.Run("get sends auth + accept", func(t *testing.T) {
		var out map[string]bool
		require.NoError(t, c.doGet("/connectors", &out))
		assert.Equal(t, http.MethodGet, gotMethod)
		assert.Equal(t, "/connectors", gotPath)
		assert.True(t, strings.HasPrefix(gotAuth, "Basic "))
		assert.Equal(t, "application/json", gotAccept)
		assert.True(t, out["ok"])
	})

	t.Run("post sends body + content-type", func(t *testing.T) {
		require.NoError(t, c.doPost("/connectors", map[string]string{"name": "x"}, nil))
		assert.Equal(t, http.MethodPost, gotMethod)
		assert.Equal(t, "application/json", gotContentType)
		assert.JSONEq(t, `{"name":"x"}`, gotBody)
	})

	t.Run("put and delete verbs", func(t *testing.T) {
		require.NoError(t, c.doPut("/connectors/x/config", map[string]string{"a": "b"}, nil))
		assert.Equal(t, http.MethodPut, gotMethod)
		require.NoError(t, c.doDelete("/connectors/x", nil))
		assert.Equal(t, http.MethodDelete, gotMethod)
	})
}

func TestConnectClient_NoAuthWhenUnset(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer srv.Close()
	c, err := newConnectClient(appconfig.ConnectCluster{Name: "c1", Address: srv.URL})
	require.NoError(t, err)
	require.NoError(t, c.doGet("/", nil))
	assert.Empty(t, gotAuth)
}

func TestMapConnectError(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantAs  interface{}
		passthr bool
	}{
		{name: "404 -> not found", status: 404, body: `{"message":"unknown"}`, wantAs: &api.ConnectorNotFoundError{}},
		{name: "409 already exists -> exists", status: 409, body: `{"message":"Connector foo already exists"}`, wantAs: &api.ConnectorAlreadyExistsError{}},
		{name: "409 rebalance -> passthrough", status: 409, body: `{"message":"rebalance in progress"}`, passthr: true},
		{name: "500 -> passthrough", status: 500, body: `{"message":"boom"}`, passthr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			c, _ := newConnectClient(appconfig.ConnectCluster{Name: "c1", Address: srv.URL})
			err := c.doGet("/connectors/foo", nil)
			mapped := mapConnectError(err, "c1", "foo")
			if tc.passthr {
				var ce *connectError
				assert.True(t, errors.As(mapped, &ce))
				return
			}
			switch tc.wantAs.(type) {
			case *api.ConnectorNotFoundError:
				var e api.ConnectorNotFoundError
				assert.True(t, errors.As(mapped, &e))
			case *api.ConnectorAlreadyExistsError:
				var e api.ConnectorAlreadyExistsError
				assert.True(t, errors.As(mapped, &e))
			}
		})
	}
}

func TestConnectClient_UnknownCluster(t *testing.T) {
	withConnect(t, appconfig.ConnectCluster{Name: "known", Address: "http://x"})
	kp := KafkaDataSourceKaf{}
	_, err := kp.connectClient("nope")
	var e api.ConnectClusterNotFoundError
	require.True(t, errors.As(err, &e))
	assert.Equal(t, "nope", e.Connect)
}

func TestGetConnectClusters_ReachableAndUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = w.Write([]byte(`{"version":"3.7.0","commit":"abc","kafka_cluster_id":"kc1"}`))
		case "/connectors":
			// expand=status
			_, _ = w.Write([]byte(`{
				"a":{"status":{"connector":{"state":"RUNNING"},"tasks":[{"id":0,"state":"RUNNING"},{"id":1,"state":"FAILED"}]}},
				"b":{"status":{"connector":{"state":"FAILED"},"tasks":[{"id":0,"state":"FAILED"}]}}
			}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	withConnect(t,
		appconfig.ConnectCluster{Name: "reachable", Address: srv.URL},
		appconfig.ConnectCluster{Name: "down", Address: "http://127.0.0.1:1"},
	)
	kp := KafkaDataSourceKaf{}

	clusters, err := kp.GetConnectClusters(true)
	require.NoError(t, err)
	require.Len(t, clusters, 2)

	assert.True(t, clusters[0].Reachable)
	assert.Equal(t, "3.7.0", clusters[0].Version)
	assert.Equal(t, "kc1", clusters[0].KafkaClusterID)
	assert.Equal(t, 2, clusters[0].ConnectorCount)
	assert.Equal(t, 1, clusters[0].FailedConnectorCount)
	assert.Equal(t, 3, clusters[0].TaskCount)
	assert.Equal(t, 2, clusters[0].FailedTaskCount)

	assert.False(t, clusters[1].Reachable)
	assert.Empty(t, clusters[1].Version)
}

func TestGetConnectorNames(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`["zeta","alpha"]`))
	}))
	defer srv.Close()
	withConnect(t, appconfig.ConnectCluster{Name: "c1", Address: srv.URL})
	kp := KafkaDataSourceKaf{}
	names, err := kp.GetConnectorNames("c1")
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha", "zeta"}, names)
}

func TestGetConnectors_AggregationAndDegraded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/connectors" && r.URL.RawQuery != "":
			_, _ = w.Write([]byte(`{
				"src":{"info":{"config":{"connector.class":"FooSource"},"type":"source"},"status":{"connector":{"state":"RUNNING","worker_id":"w1"},"tasks":[{"id":0,"state":"RUNNING"}]}},
				"snk":{"info":{"config":{"connector.class":"BarSink","topics":"t1"},"type":"sink"},"status":{"connector":{"state":"RUNNING"},"tasks":[{"id":0,"state":"RUNNING"}]}}
			}`))
		case strings.HasSuffix(r.URL.Path, "/topics"):
			name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/connectors/"), "/topics")
			_, _ = w.Write([]byte(`{"` + name + `":{"topics":["t1"]}}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	withConnect(t,
		appconfig.ConnectCluster{Name: "c1", Address: srv.URL, ConsumerNamePattern: "cg-<connector>"},
		appconfig.ConnectCluster{Name: "down", Address: "http://127.0.0.1:1"},
	)
	kp := KafkaDataSourceKaf{}
	conns, err := kp.GetConnectors()
	require.NoError(t, err)
	// down cluster omitted; only c1's 2 connectors
	require.Len(t, conns, 2)
	byName := map[string]api.Connector{}
	for _, c := range conns {
		byName[c.Name] = c
	}
	assert.Equal(t, api.ConnectorTypeSource, byName["src"].Type)
	assert.Equal(t, "FooSource", byName["src"].Class)
	assert.Equal(t, api.ConnectorTypeSink, byName["snk"].Type)
	assert.Equal(t, "cg-snk", byName["snk"].ConsumerGroup)
	assert.Equal(t, []string{"t1"}, byName["snk"].Topics)
}

func TestGetConnectorDetails_MaskingAndMissingStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/connectors/foo":
			_, _ = w.Write([]byte(`{"name":"foo","type":"sink","config":{"connector.class":"X","topics":"t","database.password":"s3cret"}}`))
		case r.URL.Path == "/connectors/foo/status":
			w.WriteHeader(404) // missing status
		case r.URL.Path == "/connectors/foo/topics":
			_, _ = w.Write([]byte(`{"foo":{"topics":["t"]}}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	withConnect(t, appconfig.ConnectCluster{Name: "c1", Address: srv.URL})
	kp := KafkaDataSourceKaf{}
	d, err := kp.GetConnectorDetails("c1", "foo")
	require.NoError(t, err)
	assert.Equal(t, api.ConnectorStateUnassigned, d.State)
	assert.Empty(t, d.Tasks)
	assert.Equal(t, api.ConnectorSecretPlaceholder, d.Config["database.password"])
	assert.Equal(t, "X", d.Config["connector.class"])
	assert.Equal(t, "connect-foo", d.ConsumerGroup)
}

func TestCreateConnector_DuplicateName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		_, _ = w.Write([]byte(`{"message":"Connector foo already exists"}`))
	}))
	defer srv.Close()
	withConnect(t, appconfig.ConnectCluster{Name: "c1", Address: srv.URL})
	kp := KafkaDataSourceKaf{}
	_, err := kp.CreateConnector("c1", "foo", map[string]string{"connector.class": "X"})
	var e api.ConnectorAlreadyExistsError
	assert.True(t, errors.As(err, &e))
}

func TestConnectWriteVerbs(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		if r.URL.Path == "/connectors/foo/config" {
			_, _ = w.Write([]byte(`{"config":{"connector.class":"X","topics":"t"},"type":"sink"}`))
		}
	}))
	defer srv.Close()
	withConnect(t, appconfig.ConnectCluster{Name: "c1", Address: srv.URL})
	kp := KafkaDataSourceKaf{}

	require.NoError(t, kp.DeleteConnector("c1", "foo"))
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/connectors/foo", gotPath)

	require.NoError(t, kp.PauseConnector("c1", "foo"))
	assert.Equal(t, "/connectors/foo/pause", gotPath)
	require.NoError(t, kp.ResumeConnector("c1", "foo"))
	assert.Equal(t, "/connectors/foo/resume", gotPath)
	require.NoError(t, kp.StopConnector("c1", "foo"))
	assert.Equal(t, "/connectors/foo/stop", gotPath)
	assert.Equal(t, http.MethodPut, gotMethod)

	require.NoError(t, kp.RestartConnector("c1", "foo"))
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/connectors/foo/restart", gotPath)

	require.NoError(t, kp.RestartConnectorTask("c1", "foo", 2))
	assert.Equal(t, "/connectors/foo/tasks/2/restart", gotPath)

	_, err := kp.UpdateConnectorConfig("c1", "foo", map[string]string{"connector.class": "X", "topics": "t"})
	require.NoError(t, err)
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Contains(t, gotBody, "connector.class")
}

func TestResetConnectorOffsets_StoppedGuard(t *testing.T) {
	var deleteCalled bool
	state := "RUNNING"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/connectors/foo/status" {
			_, _ = w.Write([]byte(`{"name":"foo","connector":{"state":"` + state + `"},"tasks":[]}`))
			return
		}
		if r.Method == http.MethodDelete && r.URL.Path == "/connectors/foo/offsets" {
			deleteCalled = true
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()
	withConnect(t, appconfig.ConnectCluster{Name: "c1", Address: srv.URL})
	kp := KafkaDataSourceKaf{}

	t.Run("not stopped rejects locally", func(t *testing.T) {
		state = "RUNNING"
		deleteCalled = false
		err := kp.ResetConnectorOffsets("c1", "foo")
		var e api.ConnectorNotStoppedError
		require.True(t, errors.As(err, &e))
		assert.Equal(t, "RUNNING", e.State)
		assert.False(t, deleteCalled)
	})
	t.Run("stopped calls delete", func(t *testing.T) {
		state = "STOPPED"
		deleteCalled = false
		require.NoError(t, kp.ResetConnectorOffsets("c1", "foo"))
		assert.True(t, deleteCalled)
	})
}

func TestGetConnectorPlugins(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"class":"FooSource","type":"source","version":"1.0"}]`))
	}))
	defer srv.Close()
	withConnect(t, appconfig.ConnectCluster{Name: "c1", Address: srv.URL})
	kp := KafkaDataSourceKaf{}
	plugins, err := kp.GetConnectorPlugins("c1")
	require.NoError(t, err)
	require.Len(t, plugins, 1)
	assert.Equal(t, "FooSource", plugins[0].Class)
	assert.Equal(t, "source", plugins[0].Type)
}

func TestValidateConnectorConfig(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{
			"name":"FooSource","error_count":1,"groups":["Common"],
			"configs":[{"value":{"name":"topics","value":"","errors":["Missing required configuration"],"visible":true}}]
		}`))
	}))
	defer srv.Close()
	withConnect(t, appconfig.ConnectCluster{Name: "c1", Address: srv.URL})
	kp := KafkaDataSourceKaf{}
	res, err := kp.ValidateConnectorConfig("c1", "FooSource", map[string]string{"connector.class": "FooSource"})
	require.NoError(t, err)
	assert.Equal(t, "/connector-plugins/FooSource/config/validate", gotPath)
	assert.Equal(t, 1, res.ErrorCount)
	require.Len(t, res.Configs, 1)
	assert.Equal(t, "topics", res.Configs[0].Name)
	assert.Equal(t, []string{"Missing required configuration"}, res.Configs[0].Errors)
}

func TestNewConnectClient_ReusesHTTPClientPerConfig(t *testing.T) {
	cc := appconfig.ConnectCluster{Name: "reuse", Address: "http://reuse.invalid:8083"}
	a, err := newConnectClient(cc)
	require.NoError(t, err)
	b, err := newConnectClient(cc)
	require.NoError(t, err)
	assert.Same(t, a.http, b.http, "same config must share one http.Client/Transport")

	tr, ok := a.http.Transport.(*http.Transport)
	require.True(t, ok)
	assert.NotZero(t, tr.IdleConnTimeout, "idle keep-alive connections must expire")

	edited := cc
	edited.Address = "http://other.invalid:8083"
	c, err := newConnectClient(edited)
	require.NoError(t, err)
	assert.NotSame(t, a.http, c.http, "an edited config gets its own client")
}

func TestGetConnectors_FetchesTopicsConcurrently(t *testing.T) {
	const n = 16
	var (
		mu                sync.Mutex
		inFlight, maxSeen int
		releaseOnce       sync.Once
	)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/connectors" {
			parts := make([]string, 0, n)
			for i := 0; i < n; i++ {
				parts = append(parts, fmt.Sprintf(`"c%02d":{"info":{"config":{},"type":"source"}}`, i))
			}
			_, _ = w.Write([]byte("{" + strings.Join(parts, ",") + "}"))
			return
		}
		name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/connectors/"), "/topics")
		mu.Lock()
		inFlight++
		if inFlight > maxSeen {
			maxSeen = inFlight
		}
		if maxSeen >= 2 {
			releaseOnce.Do(func() { close(release) })
		}
		mu.Unlock()
		// Serial fetching never has a second request in flight: wait (bounded)
		// for one so the test proves the requests overlap.
		select {
		case <-release:
		case <-time.After(2 * time.Second):
		}
		mu.Lock()
		inFlight--
		mu.Unlock()
		_, _ = w.Write([]byte(`{"` + name + `":{"topics":["` + name + `-t"]}}`))
	}))
	defer srv.Close()
	withConnect(t, appconfig.ConnectCluster{Name: "many", Address: srv.URL})

	conns, err := KafkaDataSourceKaf{}.GetConnectors()
	require.NoError(t, err)
	require.Len(t, conns, n)
	for _, c := range conns {
		assert.Equal(t, []string{c.Name + "-t"}, c.Topics)
	}
	mu.Lock()
	defer mu.Unlock()
	assert.Greater(t, maxSeen, 1)
	assert.LessOrEqual(t, maxSeen, connectTopicsFanout)
}

// A CA rotated in place (same path, so the same config) must be picked up:
// the cached client is rebuilt from the new file and trusts the new server.
func TestConnectHTTPClient_ReloadsRotatedCA(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	oldSrv := httptest.NewTLSServer(handler)
	defer oldSrv.Close()
	// httptest's servers all share one built-in cert, so the rotated-to
	// server gets its own self-signed one.
	newSrv := httptest.NewUnstartedServer(handler)
	newSrv.TLS = &tls.Config{Certificates: []tls.Certificate{selfSignedCert(t)}}
	newSrv.StartTLS()
	defer newSrv.Close()

	caPath := filepath.Join(t.TempDir(), "ca.pem")
	writeCA := func(srv *httptest.Server, mod time.Time) {
		pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
		require.NoError(t, os.WriteFile(caPath, pemBytes, 0o600))
		require.NoError(t, os.Chtimes(caPath, mod, mod))
	}
	cc := appconfig.ConnectCluster{Name: "rotate", Address: newSrv.URL, TLSCAPath: caPath}
	t.Cleanup(func() {
		connectHTTPClientsMu.Lock()
		delete(connectHTTPClients, cc)
		connectHTTPClientsMu.Unlock()
	})

	writeCA(oldSrv, time.Now().Add(-time.Hour))
	before, err := connectHTTPClient(cc)
	require.NoError(t, err)
	again, err := connectHTTPClient(cc)
	require.NoError(t, err)
	assert.Same(t, before, again, "unchanged files keep the cached client")
	_, err = before.Get(newSrv.URL)
	require.Error(t, err, "the old CA does not trust the new server")

	writeCA(newSrv, time.Now())
	after, err := connectHTTPClient(cc)
	require.NoError(t, err)
	assert.NotSame(t, before, after, "a rotated CA rebuilds the client")
	resp, err := after.Get(newSrv.URL)
	require.NoError(t, err)
	_ = resp.Body.Close()
}

// selfSignedCert returns a self-signed CA certificate for 127.0.0.1.
func selfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "rotated"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	leaf, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}
}
