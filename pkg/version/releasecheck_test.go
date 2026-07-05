package version

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckLatestSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v1.2.3","html_url":"https://example.com/r"}`))
	}))
	defer srv.Close()
	orig := githubReleasesURL
	githubReleasesURL = srv.URL
	defer func() { githubReleasesURL = orig }()

	rel, err := CheckLatest(context.Background(), time.Second)
	require.NoError(t, err)
	assert.Equal(t, "v1.2.3", rel.TagName)
}

func TestCheckLatestErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	orig := githubReleasesURL
	githubReleasesURL = srv.URL
	defer func() { githubReleasesURL = orig }()

	_, err := CheckLatest(context.Background(), time.Second)
	assert.Error(t, err)
}

func TestIsOutdated(t *testing.T) {
	cases := []struct {
		cur, latest string
		want        bool
	}{
		{"v1.0.0", "v1.2.0", true},
		{"1.2.0", "1.2.0", false},
		{"v1.3.0", "v1.2.9", false},
		{"dev", "v1.2.0", false},
		{"", "v1.2.0", false},
		{"v1.2.3-4-gabc-dirty", "v1.2.3", false},
		{"v1.2.2-1-gabc", "v1.2.3", true},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, IsOutdated(c.cur, c.latest), "IsOutdated(%q,%q)", c.cur, c.latest)
	}
}
