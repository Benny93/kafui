package graphs

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/metrics/promquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultCatalogValidates(t *testing.T) {
	require.NoError(t, DefaultCatalog().Validate())
}

func TestValidateNamesOffendingID(t *testing.T) {
	c := NewCatalog([]Graph{
		{ID: "good", Template: `up{cluster_name="{{cluster}}"}`, Kind: KindInstant},
		{ID: "broken", Template: `up{topic="{{topic}}"}`, Kind: KindInstant}, // topic not declared
		{ID: "empty", Template: `   `, Kind: KindInstant},
	})
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "broken")
	assert.Contains(t, err.Error(), "empty")
	assert.NotContains(t, err.Error(), "good (")
}

func TestRenderClusterBinding(t *testing.T) {
	c := DefaultCatalog()
	q, err := c.Render("broker-disk-usage", "prod", nil)
	require.NoError(t, err)
	assert.Contains(t, q, `cluster_name="prod"`)
	assert.NotContains(t, q, "{{")
}

func TestRenderParamSubstitution(t *testing.T) {
	c := DefaultCatalog()
	q, err := c.Render("topic-partition-offsets", "prod", map[string]string{"topic": "orders"})
	require.NoError(t, err)
	assert.Contains(t, q, `cluster_name="prod"`)
	assert.Contains(t, q, `topic="orders"`)
}

func TestRenderMissingParams(t *testing.T) {
	c := DefaultCatalog()
	_, err := c.Render("topic-partition-offsets", "prod", nil)
	require.Error(t, err)
	var mpe MissingParamsError
	require.True(t, errors.As(err, &mpe))
	assert.Equal(t, []string{"topic"}, mpe.Params)
}

func TestRenderUnknownID(t *testing.T) {
	_, err := DefaultCatalog().Render("nope", "prod", nil)
	var ue UnknownGraphError
	require.True(t, errors.As(err, &ue))
}

func TestExecuteNoStorage(t *testing.T) {
	_, err := DefaultCatalog().Execute(context.Background(), nil, "broker-disk-usage", "prod", nil, time.Time{}, time.Time{})
	var nce api.MetricsNotConfiguredError
	require.True(t, errors.As(err, &nce))
}

func TestExecuteRangeDefaultsAndRejectsEndBeforeStart(t *testing.T) {
	var gotStart, gotEnd string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotStart = r.URL.Query().Get("start")
		gotEnd = r.URL.Query().Get("end")
		w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[]}}`))
	}))
	defer srv.Close()
	client, err := promquery.New([]string{srv.URL}, "")
	require.NoError(t, err)
	c := DefaultCatalog()

	// Defaulting: zero start/end fills [now-1h, now] and reaches the backend.
	_, err = c.Execute(context.Background(), client, "topic-partition-offsets", "prod", map[string]string{"topic": "t"}, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotEmpty(t, gotStart)
	assert.NotEmpty(t, gotEnd)

	// end <= start rejected.
	now := time.Now()
	_, err = c.Execute(context.Background(), client, "topic-partition-offsets", "prod", map[string]string{"topic": "t"}, now, now.Add(-time.Minute))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be after")
}

func TestExecuteInstant(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasSuffix(r.URL.Path, "/api/v1/query"))
		w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1,"5"]}]}}`))
	}))
	defer srv.Close()
	client, _ := promquery.New([]string{srv.URL}, "")
	res, err := DefaultCatalog().Execute(context.Background(), client, "broker-disk-usage", "prod", nil, time.Time{}, time.Time{})
	require.NoError(t, err)
	require.Equal(t, promquery.ResultVector, res.Type)
	require.Len(t, res.Vector, 1)
	assert.Equal(t, 5.0, res.Vector[0].Point.V)
}

func TestAvailableEmptyWithoutStorage(t *testing.T) {
	c := DefaultCatalog()
	assert.Empty(t, c.Available(false))
	assert.NotEmpty(t, c.Available(true))
}
