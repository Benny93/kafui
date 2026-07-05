package promquery

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeStep(t *testing.T) {
	base := time.Unix(1000, 0)
	cases := []struct {
		name string
		span time.Duration
		want time.Duration
	}{
		{"1h targets ~18s", time.Hour, 18 * time.Second},
		{"5m floors to 1s", 5 * time.Minute, time.Second},
		{"1m floors to 1s", time.Minute, time.Second},
		{"zero floors to 1s", 0, time.Second},
		{"24h", 24 * time.Hour, 432 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeStep(base, base.Add(tc.span))
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFailoverToSecondURL(t *testing.T) {
	// First server is closed (unreachable); second answers.
	down := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	downURL := down.URL
	down.Close()

	var hit bool
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
	}))
	defer up.Close()

	c, err := New([]string{downURL, up.URL}, "")
	require.NoError(t, err)
	res, err := c.Query(context.Background(), "up", time.Time{})
	require.NoError(t, err)
	assert.True(t, hit, "second (up) server should have been reached")
	assert.Equal(t, ResultVector, res.Type)
	// Sticky last-good: the working index is remembered.
	assert.Equal(t, 1, c.lastGood)
}

func TestAllDownAggregateError(t *testing.T) {
	s1 := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	s2 := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	u1, u2 := s1.URL, s2.URL
	s1.Close()
	s2.Close()

	c, err := New([]string{u1, u2}, "")
	require.NoError(t, err)
	_, err = c.Query(context.Background(), "up", time.Time{})
	require.Error(t, err)
	var nli NoLiveInstancesError
	require.True(t, errors.As(err, &nli), "want NoLiveInstancesError, got %T", err)
	assert.Equal(t, 2, nli.Configured)
}

func TestDecodeVector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[
			{"metric":{"__name__":"up","job":"kafka"},"value":[1609459200,"1"]},
			{"metric":{"__name__":"up","job":"zk"},"value":[1609459200.5,"0"]}
		]}}`))
	}))
	defer srv.Close()

	c, _ := New([]string{srv.URL}, "")
	res, err := c.Query(context.Background(), "up", time.Time{})
	require.NoError(t, err)
	require.Equal(t, ResultVector, res.Type)
	require.Len(t, res.Vector, 2)
	assert.Equal(t, "kafka", res.Vector[0].Metric["job"])
	assert.Equal(t, 1.0, res.Vector[0].Point.V)
	assert.Equal(t, 0.0, res.Vector[1].Point.V)
	assert.Equal(t, int64(1609459200), res.Vector[0].Point.T.Unix())
}

func TestDecodeMatrix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assert the range endpoint received a step parameter.
		assert.NotEmpty(t, r.URL.Query().Get("step"))
		w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[
			{"metric":{"topic":"orders"},"values":[[1609459200,"10"],[1609459230,"20"]]}
		]}}`))
	}))
	defer srv.Close()

	c, _ := New([]string{srv.URL}, "")
	start := time.Unix(1609459200, 0)
	res, err := c.QueryRange(context.Background(), "rate(x[1m])", start, start.Add(time.Hour))
	require.NoError(t, err)
	require.Equal(t, ResultMatrix, res.Type)
	require.Len(t, res.Matrix, 1)
	assert.Equal(t, "orders", res.Matrix[0].Metric["topic"])
	require.Len(t, res.Matrix[0].Points, 2)
	assert.Equal(t, 10.0, res.Matrix[0].Points[0].V)
	assert.Equal(t, 20.0, res.Matrix[0].Points[1].V)
}

func TestQueryErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"status":"error","errorType":"bad_data","error":"parse error"}`))
	}))
	defer srv.Close()

	c, _ := New([]string{srv.URL}, "")
	_, err := c.Query(context.Background(), "!!!", time.Time{})
	require.Error(t, err)
	var qe QueryError
	require.True(t, errors.As(err, &qe))
	assert.Equal(t, "bad_data", qe.ErrorType)
}

func TestNewNoURLs(t *testing.T) {
	c, err := New(nil, "")
	require.NoError(t, err)
	assert.Nil(t, c)
	c, err = New([]string{"  ", "/"}, "")
	require.NoError(t, err)
	assert.Nil(t, c)
}
