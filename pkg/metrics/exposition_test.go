package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeLister is a static SnapshotLister for handler tests.
type fakeLister struct{ items []api.ClusterMetrics }

func (f fakeLister) List() []api.ClusterMetrics { return f.items }

func sampleSnapshots() fakeLister {
	return fakeLister{items: []api.ClusterMetrics{
		{
			Cluster:          "prod",
			BrokerCount:      3,
			TopicCount:       10,
			MessagesInPerSec: 5,
			BytesInPerSec:    api.RateUnknown,
			BytesOutPerSec:   api.RateUnknown,
			Brokers:          []api.BrokerMetrics{{ID: 1, LeaderCount: 4, ReplicaCount: 8, SegmentSize: 1024}},
		},
		{
			Cluster:          "staging",
			BrokerCount:      1,
			TopicCount:       2,
			MessagesInPerSec: api.RateUnknown,
			BytesInPerSec:    api.RateUnknown,
			BytesOutPerSec:   api.RateUnknown,
			Brokers:          []api.BrokerMetrics{{ID: 5, LeaderCount: 2}},
		},
	}}
}

func doGet(t *testing.T, h http.Handler, path string) *http.Response {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	resp, err := http.Get(srv.URL + path)
	require.NoError(t, err)
	return resp
}

func body(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b := make([]byte, 1<<16)
	n, _ := resp.Body.Read(b)
	return string(b[:n])
}

func TestExpositionLabelInjectionAndMerge(t *testing.T) {
	h := NewExpositionHandler(sampleSnapshots(), func(string) bool { return true })
	resp := doGet(t, h, "/metrics")
	assert.Equal(t, ExpositionContentType, resp.Header.Get("Content-Type"))
	out := body(t, resp)

	// cluster_name label injected on every family.
	assert.Contains(t, out, `kafui_cluster_brokers{cluster_name="prod"} 3`)
	assert.Contains(t, out, `kafui_cluster_brokers{cluster_name="staging"} 1`)
	// broker samples additionally carry broker_id.
	assert.Contains(t, out, `broker_id="1"`)
	assert.Contains(t, out, `cluster_name="prod"`)
	// Same-name family merged: only one HELP/TYPE header for brokers.
	assert.Equal(t, 1, strings.Count(out, "# TYPE kafui_cluster_brokers gauge"))
	assert.Equal(t, 2, strings.Count(out, "kafui_cluster_brokers{"))
}

func TestExpositionOptOut404AndGlobalExclusion(t *testing.T) {
	enabled := func(cluster string) bool { return cluster != "staging" } // staging opted out
	h := NewExpositionHandler(sampleSnapshots(), enabled)

	// Own path is 404 when opted out.
	resp := doGet(t, h, "/metrics/staging")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Excluded from the global endpoint too.
	resp = doGet(t, h, "/metrics")
	out := body(t, resp)
	assert.Contains(t, out, `cluster_name="prod"`)
	assert.NotContains(t, out, `cluster_name="staging"`)
}

func TestExpositionUnknownCluster404(t *testing.T) {
	h := NewExpositionHandler(sampleSnapshots(), nil)
	resp := doGet(t, h, "/metrics/nope")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestExpositionSingleCluster(t *testing.T) {
	h := NewExpositionHandler(sampleSnapshots(), nil)
	resp := doGet(t, h, "/metrics/prod")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	out := body(t, resp)
	assert.Contains(t, out, `cluster_name="prod"`)
	assert.NotContains(t, out, `cluster_name="staging"`)
}
