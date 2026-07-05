package kafds

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/IBM/sarama"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

var (
	once              sync.Once
	tokenProv         *tokenProvider
	refreshBuffer     time.Duration = time.Second * 20
	tokenFetchTimeout time.Duration = time.Second * 10
)

var _ sarama.AccessTokenProvider = &tokenProvider{}

type tokenProvider struct {
	// refreshMutex is used to ensure that tokens are not refreshed concurrently.
	refreshMutex sync.Mutex
	// The time at which the token expires.
	expiresAt time.Time
	// The time at which the token should be replaced.
	replaceAt time.Time
	// The currently cached token value.
	currentToken string
	// ctx for token fetching
	ctx context.Context
	// cfg for token fetching from
	oauthClientCFG *clientcredentials.Config
	// static token
	staticToken bool
	// deviceMode uses the OAuth2 device-code grant (AA-13): tokens are seeded
	// from the on-disk cache (populated by PrepareOAuthDeviceFlow before the TUI)
	// and refreshed via the refresh-token grant.
	deviceMode bool
	cluster    string
	deviceCfg  *deviceAuthConfig
	refreshTok string
}

// This is a singleton
func newTokenProvider() *tokenProvider {
	once.Do(func() {
		cluster := currentCluster
		deviceURL := deviceAuthURLFor(cluster.Name)

		//token from static value, device-code grant, or client-credentials
		switch {
		case len(cluster.SASL.Token) != 0:
			tokenProv = &tokenProvider{
				oauthClientCFG: &clientcredentials.Config{},
				staticToken:    true,
				currentToken:   cluster.SASL.Token,
			}
		case isDeviceFlow(cluster.SASL, deviceURL):
			tokenProv = &tokenProvider{
				deviceMode: true,
				cluster:    cluster.Name,
				deviceCfg: &deviceAuthConfig{
					DeviceAuthURL: deviceURL,
					TokenURL:      cluster.SASL.TokenURL,
					ClientID:      cluster.SASL.ClientID,
					Scopes:        cluster.SASL.Scopes,
				},
			}
			// Seed from the cache written by PrepareOAuthDeviceFlow.
			if ct, _ := loadCachedToken(cluster.Name); ct != nil {
				tokenProv.currentToken = ct.AccessToken
				tokenProv.expiresAt = ct.Expiry
				tokenProv.replaceAt = ct.Expiry.Add(-refreshBuffer)
				tokenProv.refreshTok = ct.RefreshToken
			}
		default:
			tokenProv = &tokenProvider{
				oauthClientCFG: &clientcredentials.Config{
					ClientID:     cluster.SASL.ClientID,
					ClientSecret: cluster.SASL.ClientSecret,
					TokenURL:     cluster.SASL.TokenURL,
				},
				staticToken: false,
			}
		}
		if !tokenProv.staticToken && !tokenProv.deviceMode {
			// create context with timeout
			ctx := context.Background()
			httpClient := &http.Client{Timeout: tokenFetchTimeout}
			ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)
			tokenProv.ctx = ctx

			// get first token
			firstToken, err := tokenProv.oauthClientCFG.Token(ctx)
			if err != nil {
				panic(fmt.Errorf("Could not fetch OAUTH token: " + err.Error()))
			}
			tokenProv.currentToken = firstToken.AccessToken
			tokenProv.expiresAt = firstToken.Expiry
			tokenProv.replaceAt = firstToken.Expiry.Add(-refreshBuffer)
		}
	})
	return tokenProv
}

func (tp *tokenProvider) Token() (*sarama.AccessToken, error) {

	if tp.deviceMode {
		if tp.currentToken == "" || time.Now().After(tp.replaceAt) {
			if err := tp.refreshDeviceToken(); err != nil {
				return nil, err
			}
		}
	} else if !tp.staticToken {
		if time.Now().After(tp.replaceAt) {
			if err := tp.refreshToken(); err != nil {
				return nil, err
			}

		}
	}
	return &sarama.AccessToken{
		Token:      tp.currentToken,
		Extensions: nil,
	}, nil
}

// refreshDeviceToken refreshes the device-flow access token using the cached
// refresh token. The interactive grant runs earlier (PrepareOAuthDeviceFlow),
// so this path never prompts — stdout is already redirected by the TUI.
func (tp *tokenProvider) refreshDeviceToken() error {
	tp.refreshMutex.Lock()
	defer tp.refreshMutex.Unlock()
	if tp.currentToken != "" && time.Now().Before(tp.replaceAt) {
		return nil // another caller refreshed while we waited
	}
	ct, err := loadCachedToken(tp.cluster)
	if err != nil {
		return err
	}
	refreshTok := tp.refreshTok
	if ct != nil && ct.RefreshToken != "" {
		refreshTok = ct.RefreshToken
	}
	if refreshTok == "" {
		if tp.currentToken != "" {
			return nil // no way to refresh yet, but we still have a usable token
		}
		return fmt.Errorf("no cached OAuth token for cluster %q; re-run kafui to authenticate", tp.cluster)
	}
	newTok, err := tp.deviceCfg.refresh(context.Background(), refreshTok)
	if err != nil {
		return err
	}
	if err := saveCachedToken(tp.cluster, newTok); err != nil {
		shared.Log.Warn("could not persist refreshed OAuth token", "cluster", tp.cluster, "err", err)
	}
	tp.currentToken = newTok.AccessToken
	tp.expiresAt = newTok.Expiry
	tp.replaceAt = newTok.Expiry.Add(-refreshBuffer)
	tp.refreshTok = newTok.RefreshToken
	return nil
}

func (tp *tokenProvider) refreshToken() error {
	// Get a lock on the update
	tp.refreshMutex.Lock()
	defer tp.refreshMutex.Unlock()

	// Check whether another call refreshed the token while waiting for the lock to be acquired here
	if time.Now().Before(tp.replaceAt) {
		return nil
	}

	token, err := tp.oauthClientCFG.Token(tp.ctx)
	if err != nil {
		return err
	}
	// Save the token
	tp.currentToken = token.AccessToken
	tp.expiresAt = token.Expiry
	tp.replaceAt = token.Expiry.Add(-refreshBuffer)
	return nil
}
