package kafds

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	kafcfg "github.com/birdayz/kaf/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	orig := tokenCacheDir
	tokenCacheDir = func() (string, error) { return dir, nil }
	defer func() { tokenCacheDir = orig }()

	tok := &cachedToken{AccessToken: "abc", RefreshToken: "ref", Expiry: time.Now().Add(time.Hour).Round(time.Second)}
	require.NoError(t, saveCachedToken("prod", tok))

	got, err := loadCachedToken("prod")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "abc", got.AccessToken)
	assert.Equal(t, "ref", got.RefreshToken)

	// Missing cluster returns (nil, nil).
	got, err = loadCachedToken("absent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestCachedTokenValidity(t *testing.T) {
	now := time.Now()
	assert.False(t, (&cachedToken{}).valid(now, time.Second))
	assert.True(t, (&cachedToken{AccessToken: "x"}).valid(now, time.Second)) // zero expiry = non-expiring
	assert.True(t, (&cachedToken{AccessToken: "x", Expiry: now.Add(time.Hour)}).valid(now, time.Minute))
	assert.False(t, (&cachedToken{AccessToken: "x", Expiry: now.Add(time.Second)}).valid(now, time.Minute))
}

func TestDeviceFlowPoller(t *testing.T) {
	var tokenHits int
	mux := http.NewServeMux()
	mux.HandleFunc("/device", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "client-x", r.Form.Get("client_id"))
		w.Write([]byte(`{"device_code":"dev123","user_code":"WXYZ","verification_uri":"https://idp/activate","expires_in":600,"interval":1}`))
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		tokenHits++
		if tokenHits < 3 {
			// First two polls: authorization pending.
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"authorization_pending"}`))
			return
		}
		assert.Equal(t, deviceCodeGrantType, r.Form.Get("grant_type"))
		assert.Equal(t, "dev123", r.Form.Get("device_code"))
		w.Write([]byte(`{"access_token":"tok-abc","refresh_token":"ref-abc","expires_in":3600}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	var slept int
	d := &deviceAuthConfig{
		DeviceAuthURL: srv.URL + "/device",
		TokenURL:      srv.URL + "/token",
		ClientID:      "client-x",
		sleep:         func(time.Duration) { slept++ }, // no real waiting
	}
	tok, err := d.run(context.Background(), io.Discard)
	require.NoError(t, err)
	assert.Equal(t, "tok-abc", tok.AccessToken)
	assert.Equal(t, "ref-abc", tok.RefreshToken)
	assert.True(t, tok.Expiry.After(time.Now()))
	assert.Equal(t, 3, tokenHits)
	assert.Equal(t, 2, slept) // slept once per pending poll
}

func TestDeviceFlowRefresh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		assert.Equal(t, "old-ref", r.Form.Get("refresh_token"))
		w.Write([]byte(`{"access_token":"new-tok","expires_in":3600}`))
	}))
	defer srv.Close()

	d := &deviceAuthConfig{TokenURL: srv.URL, ClientID: "c"}
	tok, err := d.refresh(context.Background(), "old-ref")
	require.NoError(t, err)
	assert.Equal(t, "new-tok", tok.AccessToken)
	// Refresh token preserved when the server does not rotate it.
	assert.Equal(t, "old-ref", tok.RefreshToken)
}

func TestValidateOAuthConfig(t *testing.T) {
	cases := []struct {
		name      string
		sasl      *kafcfg.SASL
		deviceURL string
		wantErr   bool
	}{
		{"non-oauth is fine", &kafcfg.SASL{Mechanism: "PLAIN"}, "", false},
		{"nil sasl fine", nil, "", false},
		{"device flow valid", &kafcfg.SASL{Mechanism: "OAUTHBEARER", ClientID: "c", TokenURL: "t"}, "https://d", false},
		{"device + static token rejected", &kafcfg.SASL{Mechanism: "OAUTHBEARER", Token: "x", ClientID: "c", TokenURL: "t"}, "https://d", true},
		{"device + client secret rejected", &kafcfg.SASL{Mechanism: "OAUTHBEARER", ClientSecret: "s", ClientID: "c", TokenURL: "t"}, "https://d", true},
		{"device missing clientID", &kafcfg.SASL{Mechanism: "OAUTHBEARER", TokenURL: "t"}, "https://d", true},
		{"device missing tokenURL", &kafcfg.SASL{Mechanism: "OAUTHBEARER", ClientID: "c"}, "https://d", true},
		{"partial client-credentials rejected", &kafcfg.SASL{Mechanism: "OAUTHBEARER", ClientSecret: "s", ClientID: "c"}, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateOAuthConfig(tc.sasl, tc.deviceURL)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsDeviceFlow(t *testing.T) {
	assert.True(t, isDeviceFlow(&kafcfg.SASL{Mechanism: "OAUTHBEARER", ClientID: "c"}, "https://d"))
	assert.False(t, isDeviceFlow(&kafcfg.SASL{Mechanism: "OAUTHBEARER", ClientID: "c"}, "")) // no device url
	assert.False(t, isDeviceFlow(&kafcfg.SASL{Mechanism: "OAUTHBEARER", Token: "x"}, "https://d"))
	assert.False(t, isDeviceFlow(&kafcfg.SASL{Mechanism: "OAUTHBEARER", ClientSecret: "s"}, "https://d"))
	assert.False(t, isDeviceFlow(&kafcfg.SASL{Mechanism: "PLAIN"}, "https://d"))
}
