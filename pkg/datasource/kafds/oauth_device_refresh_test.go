package kafds

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/birdayz/kaf/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolateTokenCache redirects the on-disk token cache to a temp dir for one test.
func isolateTokenCache(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	prev := tokenCacheDir
	tokenCacheDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { tokenCacheDir = prev })
	return dir
}

// deviceProvider builds a device-mode provider bound to the given token URL,
// with the cache already seeded for the cluster.
func deviceProvider(tokenURL, cluster, refreshTok string, expiry time.Time) *tokenProvider {
	return &tokenProvider{
		deviceMode: true,
		cluster:    cluster,
		deviceCfg: &deviceAuthConfig{
			TokenURL: tokenURL,
			ClientID: "cid",
		},
		refreshTok: refreshTok,
		expiresAt:  expiry,
		replaceAt:  expiry.Add(-refreshBuffer),
	}
}

func writeTokenCache(t *testing.T, dir, cluster string, ct *cachedToken) {
	t.Helper()
	data, err := json.Marshal(ct)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, cluster+".json"), data, 0o600))
}

func TestTokenProvider_RefreshDeviceToken(t *testing.T) {
	t.Run("valid cached token performs no HTTP call", func(t *testing.T) {
		dir := isolateTokenCache(t)
		writeTokenCache(t, dir, "c1", &cachedToken{
			AccessToken: "cached", RefreshToken: "r1", Expiry: time.Now().Add(time.Hour),
		})
		prov := deviceProvider("http://unused", "c1", "r1", time.Now().Add(time.Hour))
		prov.currentToken = "seeded"

		require.NoError(t, prov.refreshDeviceToken())
		assert.Equal(t, "seeded", prov.currentToken, "still-valid token is left alone")
	})

	t.Run("another refresh while waiting short-circuits", func(t *testing.T) {
		isolateTokenCache(t)
		prov := deviceProvider("http://unused", "c1", "r1", time.Now().Add(time.Hour))
		prov.currentToken = "fresh"
		// currentToken != "" && now < replaceAt → early return, no cache read.

		require.NoError(t, prov.refreshDeviceToken())
		assert.Equal(t, "fresh", prov.currentToken)
	})

	t.Run("refresh grant renews the token and persists it", func(t *testing.T) {
		dir := isolateTokenCache(t)
		var grants atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			grants.Add(1)
			require.NoError(t, r.ParseForm())
			assert.Equal(t, "refresh_token", r.Form.Get("grant_type"))
			assert.Equal(t, "r1", r.Form.Get("refresh_token"))
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "new-access", "expires_in": 3600,
			})
		}))
		defer srv.Close()
		writeTokenCache(t, dir, "c1", &cachedToken{AccessToken: "old", RefreshToken: "r1"})

		prov := deviceProvider(srv.URL, "c1", "r1", time.Now().Add(-time.Minute))
		require.NoError(t, prov.refreshDeviceToken())

		assert.Equal(t, "new-access", prov.currentToken)
		assert.Equal(t, int32(1), grants.Load())
		assert.Equal(t, "r1", prov.refreshTok, "unrotated refresh token is preserved")
		// The renewed token was written back to the cache file.
		raw, err := os.ReadFile(filepath.Join(dir, "c1.json"))
		require.NoError(t, err)
		var saved cachedToken
		require.NoError(t, json.Unmarshal(raw, &saved))
		assert.Equal(t, "new-access", saved.AccessToken)
	})

	t.Run("cache refresh token overrides the seeded one", func(t *testing.T) {
		dir := isolateTokenCache(t)
		var got string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = r.ParseForm()
			got = r.Form.Get("refresh_token")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "a2", "expires_in": 60})
		}))
		defer srv.Close()
		writeTokenCache(t, dir, "c1", &cachedToken{AccessToken: "old", RefreshToken: "cache-r"})

		prov := deviceProvider(srv.URL, "c1", "stale-r", time.Now().Add(-time.Minute))
		require.NoError(t, prov.refreshDeviceToken())
		assert.Equal(t, "cache-r", got, "the on-disk (freshest) refresh token wins")
	})

	t.Run("no refresh token but usable token is not an error", func(t *testing.T) {
		dir := isolateTokenCache(t)
		writeTokenCache(t, dir, "c1", &cachedToken{AccessToken: "old"})

		prov := deviceProvider("http://unused", "c1", "", time.Now().Add(-time.Minute))
		prov.currentToken = "still-usable"
		require.NoError(t, prov.refreshDeviceToken())
		assert.Equal(t, "still-usable", prov.currentToken)
	})

	t.Run("nothing to refresh with points at re-running kafui", func(t *testing.T) {
		isolateTokenCache(t)
		prov := deviceProvider("http://unused", "c1", "", time.Time{})

		err := prov.refreshDeviceToken()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "re-run kafui")
	})

	t.Run("corrupt cache surfaces as error", func(t *testing.T) {
		dir := isolateTokenCache(t)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "c1.json"), []byte("{not json"), 0o600))

		prov := deviceProvider("http://unused", "c1", "", time.Now().Add(-time.Minute))
		err := prov.refreshDeviceToken()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing token cache")
	})

	t.Run("Token drives the device path when expired", func(t *testing.T) {
		dir := isolateTokenCache(t)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "via-token", "expires_in": 3600})
		}))
		defer srv.Close()
		writeTokenCache(t, dir, "c1", &cachedToken{AccessToken: "old", RefreshToken: "r1"})

		prov := deviceProvider(srv.URL, "c1", "r1", time.Now().Add(-time.Minute))
		tok, err := prov.Token()
		require.NoError(t, err)
		assert.Equal(t, "via-token", tok.Token)
	})
}

func TestNewTokenProvider_DeviceSeeding(t *testing.T) {
	dir := isolateTokenCache(t)
	origCluster := currentCluster
	t.Cleanup(func() {
		currentCluster = origCluster
		resetTokenProvider()
	})
	resetTokenProvider()

	expiry := time.Now().Add(2 * time.Hour)
	writeTokenCache(t, dir, "dev", &cachedToken{
		AccessToken: "dev-access", RefreshToken: "dev-refresh", Expiry: expiry,
	})

	currentCluster = &config.Cluster{Name: "dev", SASL: &config.SASL{
		Mechanism: "OAUTHBEARER", ClientID: "cid", TokenURL: "https://issuer/token",
	}}
	// The device endpoint from config marks this as a device-flow cluster.
	prevDeviceURL := deviceAuthURLFor
	deviceAuthURLFor = func(string) string { return "https://issuer/device" }
	t.Cleanup(func() { deviceAuthURLFor = prevDeviceURL })

	prov := newTokenProvider()
	assert.True(t, prov.deviceMode)
	assert.Equal(t, "dev-access", prov.currentToken, "seeded from the on-disk cache")
	assert.Equal(t, "dev-refresh", prov.refreshTok)
	assert.True(t, prov.replaceAt.After(time.Now()), "replace time derived from cached expiry")
}

func TestNewTokenProvider_PanicsWhenFirstFetchFails(t *testing.T) {
	origCluster := currentCluster
	t.Cleanup(func() {
		currentCluster = origCluster
		resetTokenProvider()
	})
	resetTokenProvider()

	currentCluster = &config.Cluster{Name: "cc", SASL: &config.SASL{
		Mechanism: "OAUTHBEARER", ClientID: "cid", ClientSecret: "sec",
		TokenURL: "http://127.0.0.1:1/token", // connection refused
	}}

	assert.Panics(t, func() { newTokenProvider() },
		"first client-credentials fetch failing must panic as documented")
}
