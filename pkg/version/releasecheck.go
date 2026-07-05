package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// githubReleasesURL is the API endpoint for the latest release. Overridable in tests.
var githubReleasesURL = "https://api.github.com/repos/Benny93/kafui/releases/latest"

// Release is the subset of the GitHub release payload we care about.
type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// CheckLatest queries the GitHub releases API for the newest published release.
// All failures are swallowed into a nil result + error; callers treat any error
// as "unknown, don't nag". Honors ctx and the given timeout.
func CheckLatest(ctx context.Context, timeout time.Duration) (*Release, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(cctx, http.MethodGet, githubReleasesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("release check: unexpected status %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("release check: empty tag")
	}
	return &rel, nil
}

// IsOutdated reports whether the running Version is older than latestTag.
// Comparison is a lenient semver-ish compare after stripping a leading "v";
// non-release builds ("dev", empty) are never considered outdated.
func IsOutdated(current, latestTag string) bool {
	cur := normalizeVersion(current)
	latest := normalizeVersion(latestTag)
	if cur == "" || latest == "" || cur == "dev" {
		return false
	}
	return compareVersions(cur, latest) < 0
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	// Drop build/pre-release suffixes (e.g. "1.2.3-4-gabc-dirty" -> "1.2.3").
	for _, sep := range []string{"-", "+"} {
		if i := strings.Index(v, sep); i >= 0 {
			v = v[:i]
		}
	}
	return v
}

// compareVersions compares dotted numeric versions. Returns -1, 0, or 1.
func compareVersions(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	n := max(len(as), len(bs))
	for i := range n {
		av, bv := atoiSafe(as, i), atoiSafe(bs, i)
		if av != bv {
			if av < bv {
				return -1
			}
			return 1
		}
	}
	return 0
}

func atoiSafe(parts []string, i int) int {
	if i >= len(parts) {
		return 0
	}
	n := 0
	for _, r := range parts[i] {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}
