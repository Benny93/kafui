package kafds

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/appconfig"
	kafcfg "github.com/birdayz/kaf/pkg/config"
)

// deviceAuthURLFor resolves a cluster's device-authorization endpoint from the
// kafui overlay config. It is a variable so tests can inject a value without
// touching disk. The kaf config (~/.kaf/config) is never read or written here.
var deviceAuthURLFor = func(cluster string) string {
	cfg, err := appconfig.Load(appconfig.DefaultPath())
	if err != nil {
		return ""
	}
	ext, ok := cfg.Clusters[cluster]
	if !ok || ext.SASL == nil {
		return ""
	}
	return ext.SASL.DeviceAuthURL
}

// PrepareOAuthDeviceFlow runs the interactive OAuth2 device-code grant for the
// active cluster when configured, BEFORE the TUI redirects stdout (AA-13). It
// loads the kaf config at cfgPath (pass "" for the default), validates the
// OAUTHBEARER credential combination, and — when device flow applies and no
// usable/refreshable cached token exists — displays the verification URL and
// user code on w and caches the resulting token. It is a no-op for non-device
// clusters and returns a descriptive error for invalid configuration.
func PrepareOAuthDeviceFlow(cfgPath string, w io.Writer) error {
	kc, err := kafcfg.ReadConfig(cfgPath)
	if err != nil {
		return nil // no kaf config: nothing to prepare (kafds handles defaults)
	}
	kc.ClusterOverride = clusterOverride
	cluster := kc.ActiveCluster()
	if cluster == nil || cluster.SASL == nil {
		return nil
	}
	deviceURL := deviceAuthURLFor(cluster.Name)
	if err := validateOAuthConfig(cluster.SASL, deviceURL); err != nil {
		return fmt.Errorf("cluster %q: %w", cluster.Name, err)
	}
	if !isDeviceFlow(cluster.SASL, deviceURL) {
		return nil
	}
	// A cached token that is valid now, or that carries a refresh token (usable
	// silently at connection time), means no interactive prompt is needed.
	if ct, _ := loadCachedToken(cluster.Name); ct != nil && (ct.valid(time.Now(), refreshBuffer) || ct.RefreshToken != "") {
		return nil
	}
	d := &deviceAuthConfig{
		DeviceAuthURL: deviceURL,
		TokenURL:      cluster.SASL.TokenURL,
		ClientID:      cluster.SASL.ClientID,
		Scopes:        cluster.SASL.Scopes,
	}
	ct, err := d.run(context.Background(), w)
	if err != nil {
		return fmt.Errorf("device authentication for cluster %q: %w", cluster.Name, err)
	}
	return saveCachedToken(cluster.Name, ct)
}

// deviceCodeGrantType is the RFC 8628 device-code token grant type.
const deviceCodeGrantType = "urn:ietf:params:oauth:grant-type:device_code"

// deviceAuthConfig holds the endpoints and client identity for the OAuth2
// device-code grant (AA-13). It uses only stdlib HTTP form posts.
type deviceAuthConfig struct {
	DeviceAuthURL string
	TokenURL      string
	ClientID      string
	Scopes        []string

	httpClient *http.Client
	// sleep and now are seams for deterministic tests.
	sleep func(time.Duration)
	now   func() time.Time
}

func (d *deviceAuthConfig) client() *http.Client {
	if d.httpClient != nil {
		return d.httpClient
	}
	return &http.Client{Timeout: tokenFetchTimeout}
}

func (d *deviceAuthConfig) sleeper() func(time.Duration) {
	if d.sleep != nil {
		return d.sleep
	}
	return time.Sleep
}

func (d *deviceAuthConfig) clock() func() time.Time {
	if d.now != nil {
		return d.now
	}
	return time.Now
}

// deviceCodeResponse is the device-authorization endpoint response.
type deviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// tokenResponse is the token endpoint response for both the device-code poll and
// the refresh grant.
type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// requestDeviceCode calls the device-authorization endpoint.
func (d *deviceAuthConfig) requestDeviceCode(ctx context.Context) (*deviceCodeResponse, error) {
	form := url.Values{}
	form.Set("client_id", d.ClientID)
	if len(d.Scopes) > 0 {
		form.Set("scope", strings.Join(d.Scopes, " "))
	}
	resp, err := d.postForm(ctx, d.DeviceAuthURL, form)
	if err != nil {
		return nil, fmt.Errorf("device authorization request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("device authorization request failed: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var dc deviceCodeResponse
	if err := json.Unmarshal(body, &dc); err != nil {
		return nil, fmt.Errorf("decoding device authorization response: %w", err)
	}
	if dc.DeviceCode == "" {
		return nil, fmt.Errorf("device authorization response missing device_code")
	}
	return &dc, nil
}

// poll repeatedly hits the token endpoint until the user authorizes, an error
// occurs, or the context is cancelled. It honors authorization_pending and
// slow_down per RFC 8628.
func (d *deviceAuthConfig) poll(ctx context.Context, deviceCode string, interval time.Duration) (*cachedToken, error) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	sleep := d.sleeper()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		form := url.Values{}
		form.Set("grant_type", deviceCodeGrantType)
		form.Set("device_code", deviceCode)
		form.Set("client_id", d.ClientID)
		tr, status, err := d.postToken(ctx, form)
		if err != nil {
			return nil, err
		}
		switch tr.Error {
		case "":
			if status < 200 || status >= 300 {
				return nil, fmt.Errorf("token endpoint returned HTTP %d", status)
			}
			return d.toCachedToken(tr), nil
		case "authorization_pending":
			sleep(interval)
		case "slow_down":
			interval += 5 * time.Second
			sleep(interval)
		default:
			return nil, fmt.Errorf("device authorization failed: %s: %s", tr.Error, tr.ErrorDescription)
		}
	}
}

// refresh exchanges a refresh token for a fresh access token.
func (d *deviceAuthConfig) refresh(ctx context.Context, refreshToken string) (*cachedToken, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", d.ClientID)
	tr, status, err := d.postToken(ctx, form)
	if err != nil {
		return nil, err
	}
	if tr.Error != "" {
		return nil, fmt.Errorf("token refresh failed: %s: %s", tr.Error, tr.ErrorDescription)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("token refresh returned HTTP %d", status)
	}
	ct := d.toCachedToken(tr)
	// Preserve the prior refresh token if the server did not rotate it.
	if ct.RefreshToken == "" {
		ct.RefreshToken = refreshToken
	}
	return ct, nil
}

func (d *deviceAuthConfig) toCachedToken(tr *tokenResponse) *cachedToken {
	ct := &cachedToken{AccessToken: tr.AccessToken, RefreshToken: tr.RefreshToken}
	if tr.ExpiresIn > 0 {
		ct.Expiry = d.clock()().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}
	return ct
}

func (d *deviceAuthConfig) postForm(ctx context.Context, endpoint string, form url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	return d.client().Do(req)
}

// postToken posts to the token endpoint and decodes the JSON body regardless of
// status (RFC 6749 error responses are JSON with a non-2xx status).
func (d *deviceAuthConfig) postToken(ctx context.Context, form url.Values) (*tokenResponse, int, error) {
	resp, err := d.postForm(ctx, d.TokenURL, form)
	if err != nil {
		return nil, 0, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var tr tokenResponse
	if len(body) > 0 {
		if err := json.Unmarshal(body, &tr); err != nil {
			return nil, resp.StatusCode, fmt.Errorf("decoding token response: %w", err)
		}
	}
	return &tr, resp.StatusCode, nil
}

// run performs the full interactive device-code grant: request a code, display
// the verification URL + user code on w, poll for the token, and return it. It
// is called before the TUI redirects stdout (AA-13).
func (d *deviceAuthConfig) run(ctx context.Context, w io.Writer) (*cachedToken, error) {
	dc, err := d.requestDeviceCode(ctx)
	if err != nil {
		return nil, err
	}
	uri := dc.VerificationURIComplete
	if uri == "" {
		uri = dc.VerificationURI
	}
	fmt.Fprintf(w, "\nTo authenticate, open the following URL in your browser:\n  %s\n", uri)
	if dc.VerificationURIComplete == "" {
		fmt.Fprintf(w, "and enter the code: %s\n", dc.UserCode)
	}
	fmt.Fprintln(w, "Waiting for authorization...")
	return d.poll(ctx, dc.DeviceCode, time.Duration(dc.Interval)*time.Second)
}

// validateOAuthConfig checks the OAUTHBEARER credential combination and returns
// a descriptive error for invalid combinations. deviceAuthURL is the resolved
// device-authorization endpoint (may be empty). It is a no-op for non-OAUTHBEARER
// mechanisms and clusters without SASL.
func validateOAuthConfig(sasl *kafcfg.SASL, deviceAuthURL string) error {
	if sasl == nil || sasl.Mechanism != "OAUTHBEARER" {
		return nil
	}
	// Device-flow validation only applies when a device endpoint is configured.
	if deviceAuthURL != "" {
		if sasl.Token != "" {
			return fmt.Errorf("OAUTHBEARER: cannot combine a static token with the device-authorization flow (deviceAuthURL)")
		}
		if sasl.ClientSecret != "" {
			return fmt.Errorf("OAUTHBEARER: cannot combine a client secret (client-credentials grant) with the device-authorization flow (deviceAuthURL)")
		}
		if sasl.ClientID == "" {
			return fmt.Errorf("OAUTHBEARER device flow requires a clientID")
		}
		if sasl.TokenURL == "" {
			return fmt.Errorf("OAUTHBEARER device flow requires a tokenURL")
		}
		return nil
	}
	// Non-device paths: keep the existing static-token and client-credentials
	// contracts. A token or a client-credentials triple (or nothing, deferred to
	// the broker) is acceptable; this only catches an obviously partial setup.
	if sasl.Token == "" && sasl.ClientSecret != "" && (sasl.ClientID == "" || sasl.TokenURL == "") {
		return fmt.Errorf("OAUTHBEARER client-credentials grant requires clientID and tokenURL")
	}
	return nil
}

// isDeviceFlow reports whether a cluster should use the device-code grant.
func isDeviceFlow(sasl *kafcfg.SASL, deviceAuthURL string) bool {
	return sasl != nil &&
		sasl.Mechanism == "OAUTHBEARER" &&
		deviceAuthURL != "" &&
		sasl.Token == "" &&
		sasl.ClientSecret == ""
}
