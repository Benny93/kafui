package jolokia

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixture = `{
  "status": 200,
  "value": {
    "kafka.server:name=BytesInPerSec,type=BrokerTopicMetrics": {
      "Count": 12345,
      "OneMinuteRate": 4.5,
      "EventType": "bytes"
    },
    "kafka.server:name=MessagesInPerSec,topic=orders,type=BrokerTopicMetrics": {
      "Count": 999
    }
  }
}`

func findSample(samples []Sample, name string) (Sample, bool) {
	for _, s := range samples {
		if s.Name == name {
			return s, true
		}
	}
	return Sample{}, false
}

func TestParseNamingLabelsAndSkip(t *testing.T) {
	samples, err := Parse([]byte(fixture))
	require.NoError(t, err)

	// Naming: domain_firstProp_attribute with dots/dashes -> underscores.
	// First property is "name" so firstProp value = "BytesInPerSec".
	count, ok := findSample(samples, "kafka_server_BytesInPerSec_Count")
	require.True(t, ok, "expected Count metric, got %+v", samples)
	assert.Equal(t, 12345.0, count.Value)

	rate, ok := findSample(samples, "kafka_server_BytesInPerSec_OneMinuteRate")
	require.True(t, ok)
	assert.Equal(t, 4.5, rate.Value)

	// Remaining bean props become labels (type here); the first prop is not a label.
	assert.Equal(t, "BrokerTopicMetrics", count.Labels["type"])
	assert.NotContains(t, count.Labels, "name")

	// Non-numeric attribute (EventType) is skipped.
	_, ok = findSample(samples, "kafka_server_BytesInPerSec_EventType")
	assert.False(t, ok, "non-numeric attribute must be skipped")

	// Multiple properties: topic label preserved on the second bean.
	msgs, ok := findSample(samples, "kafka_server_MessagesInPerSec_Count")
	require.True(t, ok)
	assert.Equal(t, "orders", msgs.Labels["topic"])
	assert.Equal(t, "BrokerTopicMetrics", msgs.Labels["type"])
}

func TestParseError(t *testing.T) {
	_, err := Parse([]byte(`{"status":500,"error":"boom"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestCollectAgainstServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "admin", u)
		assert.Equal(t, "secret", p)
		w.Write([]byte(fixture))
	}))
	defer srv.Close()

	c := New(srv.URL, "admin", "secret")
	samples, err := c.Collect(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, samples)
}

func TestCollectHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	c := New(srv.URL, "", "")
	_, err := c.Collect(context.Background())
	require.Error(t, err)
}
