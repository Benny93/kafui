package kafds

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/IBM/sarama"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// gcpScopes are requested from Application Default Credentials. cloud-platform
// authorizes the Managed Kafka API; userinfo.email lets the tokeninfo endpoint
// report the principal for user credentials.
var gcpScopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
}

// gcpTokenInfoURL resolves the principal of an access token when ADC itself
// does not name it (gcloud user credentials). Overridable for tests.
var gcpTokenInfoURL = "https://oauth2.googleapis.com/tokeninfo"

// gcpTokenHeader is the fixed JWT header GCP Managed Kafka expects.
const gcpTokenHeader = `{"typ":"JWT","alg":"GOOG_OAUTH2_TOKEN"}`

// gcpTokenProvider implements sarama.AccessTokenProvider for the
// GCP_OAUTHBEARER SASL mechanism: Google Cloud Managed Service for Apache
// Kafka authenticates OAUTHBEARER clients with a JWT-shaped token whose
// signature segment is a Google OAuth2 access token. It is the Go equivalent
// of Java's com.google.cloud.hosted.kafka.auth.GcpLoginCallbackHandler and
// draws credentials from Application Default Credentials (gcloud ADC login,
// GOOGLE_APPLICATION_CREDENTIALS, GKE Workload Identity, ...), so kafui needs
// no credential configuration.
type gcpTokenProvider struct {
	refreshMutex sync.Mutex
	// source issues Google access tokens; injectable for tests.
	source oauth2.TokenSource
	// principal resolves the account email that becomes the token subject.
	principal func(ctx context.Context, tok *oauth2.Token) (string, error)
	now       func() time.Time
	// cached token state, following the same replace-before-expiry scheme as
	// the client-credentials tokenProvider.
	email        string
	expiresAt    time.Time
	replaceAt    time.Time
	currentToken string
}

var _ sarama.AccessTokenProvider = &gcpTokenProvider{}

// newGCPTokenProvider locates Application Default Credentials up front so a
// missing login fails before any broker connection; the first Token() call
// fetches the initial access token.
func newGCPTokenProvider() (*gcpTokenProvider, error) {
	creds, err := google.FindDefaultCredentials(context.Background(), gcpScopes...)
	if err != nil {
		return nil, fmt.Errorf("load Google Application Default Credentials for GCP_OAUTHBEARER "+
			"(run 'gcloud auth application-default login' or use Workload Identity): %w", err)
	}
	known := emailFromCredentialsJSON(creds.JSON)
	return &gcpTokenProvider{
		source: creds.TokenSource,
		principal: func(ctx context.Context, tok *oauth2.Token) (string, error) {
			return resolveGCPPrincipal(ctx, known, tok)
		},
		now: time.Now,
	}, nil
}

// Token returns a valid GCP Managed Kafka token, refreshing shortly before
// the underlying access token expires.
func (p *gcpTokenProvider) Token() (*sarama.AccessToken, error) {
	p.refreshMutex.Lock()
	defer p.refreshMutex.Unlock()
	if p.currentToken == "" || p.now().After(p.replaceAt) {
		if err := p.refreshToken(context.Background()); err != nil {
			return nil, err
		}
	}
	return &sarama.AccessToken{Token: p.currentToken}, nil
}

func (p *gcpTokenProvider) refreshToken(ctx context.Context) error {
	tok, err := p.source.Token()
	if err != nil {
		return fmt.Errorf("failed to obtain Google access token for GCP_OAUTHBEARER: %w", err)
	}
	if p.email == "" {
		email, err := p.principal(ctx, tok)
		if err != nil {
			return fmt.Errorf("failed to resolve Google principal for GCP_OAUTHBEARER: %w", err)
		}
		p.email = email
	}
	now := p.now()
	expiry := tok.Expiry
	if expiry.IsZero() {
		expiry = now.Add(time.Hour)
	}
	p.currentToken = buildGCPToken(tok.AccessToken, p.email, expiry, now)
	p.expiresAt = expiry
	p.replaceAt = expiry.Add(-refreshBuffer)
	return nil
}

// buildGCPToken assembles header.payload.accessToken, each segment base64url
// encoded without padding, exactly as GcpLoginCallbackHandler does.
func buildGCPToken(accessToken, email string, expiry, issuedAt time.Time) string {
	payload, _ := json.Marshal(struct {
		Exp int64  `json:"exp"`
		Iat int64  `json:"iat"`
		Iss string `json:"iss"`
		Sub string `json:"sub"`
	}{expiry.Unix(), issuedAt.Unix(), "Google", email})
	enc := base64.RawURLEncoding
	return enc.EncodeToString([]byte(gcpTokenHeader)) + "." +
		enc.EncodeToString(payload) + "." +
		enc.EncodeToString([]byte(accessToken))
}

// emailFromCredentialsJSON extracts the principal named by an ADC file: a
// service-account key's client_email, or the target of an impersonated
// service account. User credentials name no principal and return "".
func emailFromCredentialsJSON(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var f struct {
		ClientEmail      string `json:"client_email"`
		ImpersonationURL string `json:"service_account_impersonation_url"`
	}
	if json.Unmarshal(raw, &f) != nil {
		return ""
	}
	if f.ClientEmail != "" {
		return f.ClientEmail
	}
	// .../projects/-/serviceAccounts/<email>:generateAccessToken
	if i := strings.LastIndex(f.ImpersonationURL, "/serviceAccounts/"); i >= 0 {
		rest := f.ImpersonationURL[i+len("/serviceAccounts/"):]
		if j := strings.Index(rest, ":"); j >= 0 {
			return rest[:j]
		}
	}
	return ""
}

// resolveGCPPrincipal returns the email that becomes the token subject: the
// one named by the credentials file, else the GCE/GKE metadata server's
// default account (Workload Identity), else the tokeninfo endpoint.
func resolveGCPPrincipal(ctx context.Context, known string, tok *oauth2.Token) (string, error) {
	if known != "" {
		return known, nil
	}
	if metadata.OnGCE() {
		if email, err := metadata.Email("default"); err == nil && email != "" {
			return email, nil
		}
	}
	// sarama calls the provider synchronously during the SASL handshake while
	// the refresh mutex is held, so bound the request the same way oauth.go does
	// for its token fetch to avoid blocking a (re)connection on a stalled reply.
	cctx, cancel := context.WithTimeout(ctx, tokenFetchTimeout)
	defer cancel()
	client := &http.Client{Timeout: tokenFetchTimeout}
	return tokenInfoEmail(cctx, client, gcpTokenInfoURL, tok.AccessToken)
}

// tokenInfoEmail asks Google's tokeninfo endpoint which account owns the
// access token. The token travels in the POST body, never the URL.
func tokenInfoEmail(ctx context.Context, client *http.Client, endpoint, accessToken string) (string, error) {
	form := url.Values{"access_token": {accessToken}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tokeninfo request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tokeninfo returned HTTP %d", resp.StatusCode)
	}
	var info struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("decode tokeninfo response: %w", err)
	}
	if info.Email == "" {
		return "", fmt.Errorf("tokeninfo response has no email; ADC must include the userinfo.email scope")
	}
	return info.Email, nil
}
