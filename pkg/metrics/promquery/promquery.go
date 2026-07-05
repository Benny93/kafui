// Package promquery is a minimal client for the Prometheus HTTP query API
// (/api/v1/query and /api/v1/query_range) built on the standard library only.
//
// It targets the optional TimeSeriesURLs backend from a cluster's metrics
// configuration (MM-2): a list of interchangeable Prometheus-compatible query
// endpoints. Requests try the last-known-good URL first and fail over to the
// remaining URLs on connection-level errors; a NoLiveInstancesError is returned
// only when every URL is unreachable. HTTP error responses (4xx/5xx) from a
// reachable instance are surfaced as an error without further failover.
package promquery

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// targetPoints is the approximate number of samples a range query aims to
// return; the step is chosen to spread this many points across the interval.
const targetPoints = 200

// defaultTimeout bounds a single query request.
const defaultTimeout = 30 * time.Second

// NoLiveInstancesError is returned when every configured query URL failed at the
// connection level.
type NoLiveInstancesError struct {
	Configured int
	Cause      error
}

func (e NoLiveInstancesError) Error() string {
	return fmt.Sprintf("no live time-series instances (%d configured): %v", e.Configured, e.Cause)
}

func (e NoLiveInstancesError) Unwrap() error { return e.Cause }

// QueryError wraps a non-2xx / status!="success" API response.
type QueryError struct {
	StatusCode int
	ErrorType  string
	Message    string
}

func (e QueryError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("prometheus query error (%d %s): %s", e.StatusCode, e.ErrorType, e.Message)
	}
	return fmt.Sprintf("prometheus query error: HTTP %d", e.StatusCode)
}

// Client queries a set of interchangeable Prometheus query endpoints.
type Client struct {
	baseURLs []string
	lastGood int
	http     *http.Client
}

// New builds a client over the given base URLs (trailing slashes trimmed). An
// optional custom CA PEM path enables TLS verification against that CA. It
// returns (nil, nil) when no URLs are configured so callers can distinguish
// "no backend" from an error.
func New(urls []string, caPath string) (*Client, error) {
	var clean []string
	for _, u := range urls {
		if u = strings.TrimRight(strings.TrimSpace(u), "/"); u != "" {
			clean = append(clean, u)
		}
	}
	if len(clean) == 0 {
		return nil, nil
	}
	hc := &http.Client{Timeout: defaultTimeout}
	if caPath != "" {
		pem, err := os.ReadFile(caPath)
		if err != nil {
			return nil, fmt.Errorf("reading time-series CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("time-series CA cert %q contains no valid certificates", caPath)
		}
		hc.Transport = &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}}
	}
	return &Client{baseURLs: clean, http: hc}, nil
}

// computeStep chooses a range-query step so the result targets ~targetPoints
// samples across [start,end]. It truncates to whole seconds with a 1s floor, so
// a 1h window yields ~18s and a 5m window floors to 1s.
func computeStep(start, end time.Time) time.Duration {
	span := end.Sub(start)
	if span <= 0 {
		return time.Second
	}
	step := (span / targetPoints).Truncate(time.Second)
	if step < time.Second {
		step = time.Second
	}
	return step
}

// Query runs an instant query at time ts (now when zero).
func (c *Client) Query(ctx context.Context, promql string, ts time.Time) (*Result, error) {
	v := url.Values{}
	v.Set("query", promql)
	if !ts.IsZero() {
		v.Set("time", formatTime(ts))
	}
	return c.do(ctx, "/api/v1/query", v)
}

// QueryRange runs a range query over [start,end] with a computed step.
func (c *Client) QueryRange(ctx context.Context, promql string, start, end time.Time) (*Result, error) {
	step := computeStep(start, end)
	v := url.Values{}
	v.Set("query", promql)
	v.Set("start", formatTime(start))
	v.Set("end", formatTime(end))
	v.Set("step", strconv.FormatFloat(step.Seconds(), 'f', -1, 64))
	return c.do(ctx, "/api/v1/query_range", v)
}

func formatTime(t time.Time) string {
	return strconv.FormatFloat(float64(t.UnixNano())/1e9, 'f', -1, 64)
}

// do issues the request against each base URL in turn (starting at lastGood),
// failing over on connection errors. On the first reachable instance it decodes
// (or errors on) the response.
func (c *Client) do(ctx context.Context, path string, form url.Values) (*Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var connErr error
	n := len(c.baseURLs)
	for i := 0; i < n; i++ {
		idx := (c.lastGood + i) % n
		base := c.baseURLs[idx]
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path+"?"+form.Encode(), nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.http.Do(req)
		if err != nil {
			connErr = err
			continue
		}
		c.lastGood = idx
		defer resp.Body.Close()
		return decode(resp)
	}
	return nil, NoLiveInstancesError{Configured: n, Cause: connErr}
}

// apiEnvelope is the standard Prometheus HTTP API response envelope.
type apiEnvelope struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string          `json:"resultType"`
		Result     json.RawMessage `json:"result"`
	} `json:"data"`
	ErrorType string `json:"errorType"`
	Error     string `json:"error"`
}

func decode(resp *http.Response) (*Result, error) {
	var env apiEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, QueryError{StatusCode: resp.StatusCode}
		}
		return nil, fmt.Errorf("decoding prometheus response: %w", err)
	}
	if env.Status != "success" {
		return nil, QueryError{StatusCode: resp.StatusCode, ErrorType: env.ErrorType, Message: env.Error}
	}
	return decodeData(env.Data.ResultType, env.Data.Result)
}
