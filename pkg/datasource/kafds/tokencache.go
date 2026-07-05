package kafds

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// cachedToken is the persisted OAuth2 device-flow token for one cluster. It is
// stored at ~/.kafui/tokens/<cluster>.json with 0600 permissions so a refresh
// token survives across kafui runs (AA-13).
type cachedToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"expiry"`
}

// valid reports whether the access token is present and not within buffer of
// expiring. A zero Expiry is treated as non-expiring.
func (t *cachedToken) valid(now time.Time, buffer time.Duration) bool {
	if t == nil || t.AccessToken == "" {
		return false
	}
	if t.Expiry.IsZero() {
		return true
	}
	return now.Add(buffer).Before(t.Expiry)
}

// tokenCacheDir resolves ~/.kafui/tokens. It is a variable so tests can redirect
// it to a temp directory.
var tokenCacheDir = func() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kafui", "tokens"), nil
}

func tokenCachePath(cluster string) (string, error) {
	dir, err := tokenCacheDir()
	if err != nil {
		return "", err
	}
	// Guard against path traversal from an odd cluster name.
	safe := filepath.Base(cluster)
	if safe == "" || safe == "." || safe == ".." {
		return "", fmt.Errorf("invalid cluster name for token cache: %q", cluster)
	}
	return filepath.Join(dir, safe+".json"), nil
}

// loadCachedToken reads the cached token for a cluster. A missing file returns
// (nil, nil) so callers can distinguish "no token yet" from a read error.
func loadCachedToken(cluster string) (*cachedToken, error) {
	path, err := tokenCachePath(cluster)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading token cache: %w", err)
	}
	var t cachedToken
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parsing token cache: %w", err)
	}
	return &t, nil
}

// saveCachedToken writes the token for a cluster with 0600 perms, creating the
// tokens directory (0700) as needed.
func saveCachedToken(cluster string, t *cachedToken) error {
	path, err := tokenCachePath(cluster)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating token cache dir: %w", err)
	}
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing token cache: %w", err)
	}
	return nil
}
