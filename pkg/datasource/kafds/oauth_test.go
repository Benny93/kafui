package kafds

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/birdayz/kaf/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// resetTokenProvider resets the singleton for testing
func resetTokenProvider() {
	once = sync.Once{}
	tokenProv = nil
}

// mockOAuthServer creates a mock OAuth server for testing
func mockOAuthServer(t *testing.T, responseToken string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		
		if statusCode == http.StatusOK {
			response := `{
				"access_token": "` + responseToken + `",
				"token_type": "Bearer",
				"expires_in": 3600
			}`
			w.Write([]byte(response))
		} else {
			w.Write([]byte(`{"error": "invalid_client"}`))
		}
	}))
}

func TestTokenProvider_Token(t *testing.T) {
	originalCluster := currentCluster
	defer func() {
		currentCluster = originalCluster
		resetTokenProvider()
	}()

	t.Run("valid_static_token", func(t *testing.T) {
		resetTokenProvider()
		
		// Set up static token configuration
		currentCluster = &config.Cluster{
			SASL: &config.SASL{
				Token: "static-test-token-123",
			},
		}
		
		tp := newTokenProvider()
		require.NotNil(t, tp)
		assert.True(t, tp.staticToken)
		assert.Equal(t, "static-test-token-123", tp.currentToken)
		
		token, err := tp.Token()
		require.NoError(t, err)
		require.NotNil(t, token)
		assert.Equal(t, "static-test-token-123", token.Token)
		assert.Nil(t, token.Extensions)
	})
	
	t.Run("valid_dynamic_token", func(t *testing.T) {
		resetTokenProvider()
		
		// Create mock OAuth server
		server := mockOAuthServer(t, "dynamic-test-token-456", http.StatusOK)
		defer server.Close()
		
		// Set up dynamic token configuration
		currentCluster = &config.Cluster{
			SASL: &config.SASL{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				TokenURL:     server.URL,
			},
		}
		
		tp := newTokenProvider()
		require.NotNil(t, tp)
		assert.False(t, tp.staticToken)
		assert.Equal(t, "dynamic-test-token-456", tp.currentToken)
		
		token, err := tp.Token()
		require.NoError(t, err)
		require.NotNil(t, token)
		assert.Equal(t, "dynamic-test-token-456", token.Token)
		assert.Nil(t, token.Extensions)
	})
	
	t.Run("empty_static_token", func(t *testing.T) {
		resetTokenProvider()
		
		// Set up empty static token configuration - this will actually trigger dynamic token flow
		// because len(cluster.SASL.Token) == 0, so let's provide a mock server
		server := mockOAuthServer(t, "empty-fallback-token", http.StatusOK)
		defer server.Close()
		
		currentCluster = &config.Cluster{
			SASL: &config.SASL{
				Token:        "",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret", 
				TokenURL:     server.URL,
			},
		}
		
		tp := newTokenProvider()
		require.NotNil(t, tp)
		assert.False(t, tp.staticToken) // Will be false because empty token triggers dynamic flow
		assert.Equal(t, "empty-fallback-token", tp.currentToken)
		
		token, err := tp.Token()
		require.NoError(t, err)
		require.NotNil(t, token)
		assert.Equal(t, "empty-fallback-token", token.Token)
	})
	
	t.Run("oauth_server_error", func(t *testing.T) {
		resetTokenProvider()
		
		// Create mock OAuth server that returns error
		server := mockOAuthServer(t, "", http.StatusUnauthorized)
		defer server.Close()
		
		// Set up dynamic token configuration
		currentCluster = &config.Cluster{
			SASL: &config.SASL{
				ClientID:     "invalid-client-id",
				ClientSecret: "invalid-client-secret",
				TokenURL:     server.URL,
			},
		}
		
		// This should panic during newTokenProvider() due to initial token fetch failure
		assert.Panics(t, func() {
			newTokenProvider()
		})
	})
}

func TestTokenProvider_RefreshToken(t *testing.T) {
	originalCluster := currentCluster
	defer func() {
		currentCluster = originalCluster
		resetTokenProvider()
	}()
	
	t.Run("valid_refresh", func(t *testing.T) {
		resetTokenProvider()
		
		// Create mock OAuth server
		server := mockOAuthServer(t, "refreshed-token-789", http.StatusOK)
		defer server.Close()
		
		// Set up dynamic token configuration
		currentCluster = &config.Cluster{
			SASL: &config.SASL{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				TokenURL:     server.URL,
			},
		}
		
		tp := newTokenProvider()
		require.NotNil(t, tp)
		
		// Force token to be expired by setting replaceAt to past
		tp.replaceAt = time.Now().Add(-time.Hour)
		
		// Call refreshToken directly
		err := tp.refreshToken()
		require.NoError(t, err)
		assert.Equal(t, "refreshed-token-789", tp.currentToken)
		assert.True(t, tp.replaceAt.After(time.Now()))
	})
	
	t.Run("refresh_with_server_error", func(t *testing.T) {
		resetTokenProvider()
		
		// Create mock OAuth server that returns error
		server := mockOAuthServer(t, "", http.StatusInternalServerError)
		defer server.Close()
		
		// Set up dynamic token configuration with initial success
		currentCluster = &config.Cluster{
			SASL: &config.SASL{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				TokenURL:     server.URL,
			},
		}
		
		// This will panic during initial token fetch
		assert.Panics(t, func() {
			newTokenProvider()
		})
	})
	
	t.Run("concurrent_refresh", func(t *testing.T) {
		resetTokenProvider()
		
		// Create mock OAuth server
		server := mockOAuthServer(t, "concurrent-token", http.StatusOK)
		defer server.Close()
		
		// Set up dynamic token configuration
		currentCluster = &config.Cluster{
			SASL: &config.SASL{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				TokenURL:     server.URL,
			},
		}
		
		tp := newTokenProvider()
		require.NotNil(t, tp)
		
		// Force token to be expired
		tp.replaceAt = time.Now().Add(-time.Hour)
		
		// Test concurrent refresh calls
		var wg sync.WaitGroup
		errors := make([]error, 5)
		
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				errors[index] = tp.refreshToken()
			}(i)
		}
		
		wg.Wait()
		
		// All should succeed (or at least not error due to concurrency)
		for i, err := range errors {
			assert.NoError(t, err, "Goroutine %d should not error", i)
		}
		
		assert.Equal(t, "concurrent-token", tp.currentToken)
	})
	
	t.Run("no_refresh_needed", func(t *testing.T) {
		resetTokenProvider()
		
		// Create mock OAuth server
		server := mockOAuthServer(t, "no-refresh-needed", http.StatusOK)
		defer server.Close()
		
		// Set up dynamic token configuration
		currentCluster = &config.Cluster{
			SASL: &config.SASL{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				TokenURL:     server.URL,
			},
		}
		
		tp := newTokenProvider()
		require.NotNil(t, tp)
		
		originalToken := tp.currentToken
		
		// Don't force expiration - token should still be valid
		// replaceAt should be in the future
		
		err := tp.refreshToken()
		require.NoError(t, err)
		
		// Token should remain the same since no refresh was needed
		assert.Equal(t, originalToken, tp.currentToken)
	})
}

func TestTokenProvider_Singleton(t *testing.T) {
	originalCluster := currentCluster
	defer func() {
		currentCluster = originalCluster
		resetTokenProvider()
	}()
	
	resetTokenProvider()
	
	// Set up static token configuration
	currentCluster = &config.Cluster{
		SASL: &config.SASL{
			Token: "singleton-test-token",
		},
	}
	
	// Create multiple instances - should return the same singleton
	tp1 := newTokenProvider()
	tp2 := newTokenProvider()
	
	assert.Same(t, tp1, tp2, "newTokenProvider should return the same singleton instance")
	assert.Equal(t, "singleton-test-token", tp1.currentToken)
	assert.Equal(t, "singleton-test-token", tp2.currentToken)
}

func TestTokenProvider_Interface(t *testing.T) {
	originalCluster := currentCluster
	defer func() {
		currentCluster = originalCluster
		resetTokenProvider()
	}()
	
	resetTokenProvider()
	
	// Set up static token configuration
	currentCluster = &config.Cluster{
		SASL: &config.SASL{
			Token: "interface-test-token",
		},
	}
	
	tp := newTokenProvider()
	
	// Verify it implements sarama.AccessTokenProvider interface
	var _ sarama.AccessTokenProvider = tp
	
	// Test the interface method
	token, err := tp.Token()
	require.NoError(t, err)
	require.NotNil(t, token)
	assert.Equal(t, "interface-test-token", token.Token)
}

func TestTokenProvider_TokenExpiration(t *testing.T) {
	originalCluster := currentCluster
	defer func() {
		currentCluster = originalCluster
		resetTokenProvider()
	}()
	
	t.Run("token_refresh_on_expiration", func(t *testing.T) {
		resetTokenProvider()
		
		// Create mock OAuth server that returns different tokens on subsequent calls
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			
			var token string
			if callCount == 1 {
				token = "initial-token"
			} else {
				token = "refreshed-token"
			}
			
			response := `{
				"access_token": "` + token + `",
				"token_type": "Bearer",
				"expires_in": 3600
			}`
			w.Write([]byte(response))
		}))
		defer server.Close()
		
		// Set up dynamic token configuration
		currentCluster = &config.Cluster{
			SASL: &config.SASL{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				TokenURL:     server.URL,
			},
		}
		
		tp := newTokenProvider()
		require.NotNil(t, tp)
		assert.Equal(t, "initial-token", tp.currentToken)
		
		// Force token expiration
		tp.replaceAt = time.Now().Add(-time.Minute)
		
		// Get token - should trigger refresh
		token, err := tp.Token()
		require.NoError(t, err)
		assert.Equal(t, "refreshed-token", token.Token)
		assert.Equal(t, "refreshed-token", tp.currentToken)
	})
}

func TestTokenProvider_Configuration(t *testing.T) {
	originalCluster := currentCluster
	defer func() {
		currentCluster = originalCluster
		resetTokenProvider()
	}()
	
	t.Run("oauth_configuration_setup", func(t *testing.T) {
		resetTokenProvider()
		
		server := mockOAuthServer(t, "config-test-token", http.StatusOK)
		defer server.Close()
		
		currentCluster = &config.Cluster{
			SASL: &config.SASL{
				ClientID:     "test-client-123",
				ClientSecret: "test-secret-456",
				TokenURL:     server.URL,
			},
		}
		
		tp := newTokenProvider()
		require.NotNil(t, tp)
		assert.False(t, tp.staticToken)
		assert.Equal(t, "test-client-123", tp.oauthClientCFG.ClientID)
		assert.Equal(t, "test-secret-456", tp.oauthClientCFG.ClientSecret)
		assert.Equal(t, server.URL, tp.oauthClientCFG.TokenURL)
		assert.NotNil(t, tp.ctx)
		
		// Verify HTTP client timeout is set
		httpClient := tp.ctx.Value(oauth2.HTTPClient).(*http.Client)
		assert.Equal(t, tokenFetchTimeout, httpClient.Timeout)
	})
}

func TestTokenProvider_GlobalVariables(t *testing.T) {
	// Test that global variables have expected values
	assert.Equal(t, time.Second*20, refreshBuffer)
	assert.Equal(t, time.Second*10, tokenFetchTimeout)
}

func TestTokenProvider_EdgeCases(t *testing.T) {
	originalCluster := currentCluster
	defer func() {
		currentCluster = originalCluster
		resetTokenProvider()
	}()
	
	t.Run("nil_cluster_sasl", func(t *testing.T) {
		resetTokenProvider()
		
		currentCluster = &config.Cluster{
			SASL: nil,
		}
		
		// This should panic or handle gracefully
		assert.Panics(t, func() {
			newTokenProvider()
		})
	})
	
	t.Run("empty_oauth_config", func(t *testing.T) {
		resetTokenProvider()
		
		currentCluster = &config.Cluster{
			SASL: &config.SASL{
				ClientID:     "",
				ClientSecret: "",
				TokenURL:     "",
			},
		}
		
		// This should panic during initial token fetch
		assert.Panics(t, func() {
			newTokenProvider()
		})
	})
}