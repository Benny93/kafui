package kafds

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/ui/shared"
)

// ksqlAcceptType is the ksqlDB REST content type used for both /ksql and /query.
const ksqlAcceptType = "application/vnd.ksql.v1+json"

// defaultKsqlMaxResponseBytes bounds a ksqlDB response read when the config does
// not set an explicit limit (20 MB).
const defaultKsqlMaxResponseBytes int64 = 20 << 20

// loadKsqlEndpoint resolves the ksqlDB endpoint configured for the given Kafka
// context from the kafui overlay config. It is a package variable so tests can
// substitute an in-memory config without touching disk. The kaf config file
// (~/.kaf/config) is never read or written here.
var loadKsqlEndpoint = func(context string) *appconfig.KsqlEndpoint {
	cfg, err := appconfig.Load(appconfig.DefaultPath())
	if err != nil {
		shared.Log.Warn("loading kafui config for ksql", "err", err)
		return nil
	}
	return cfg.Clusters[context].Ksql
}

// ksqlClient is a small HTTP client for one ksqlDB endpoint (or a comma-separated
// failover list) supporting POST /ksql, a streaming POST /query, basic auth,
// optional TLS, connection-level failover and a response-size limit. Construct
// via newKsqlClient.
type ksqlClient struct {
	baseURLs         []string
	lastGood         int
	username         string
	password         string
	maxResponseBytes int64
	http             *http.Client
}

// newKsqlClient builds a client from a ksqlDB endpoint config. It returns
// (nil, nil) when no endpoint is configured (nil config or empty URL list) so
// callers can decide between an empty result and a KsqlNotConfiguredError.
func newKsqlClient(ep *appconfig.KsqlEndpoint) (*ksqlClient, error) {
	if ep == nil {
		return nil, nil
	}
	var urls []string
	for _, u := range strings.Split(ep.URL, ",") {
		if u = strings.TrimRight(strings.TrimSpace(u), "/"); u != "" {
			urls = append(urls, u)
		}
	}
	if len(urls) == 0 {
		return nil, nil
	}

	transport, err := ksqlTLSTransport(ep)
	if err != nil {
		return nil, err
	}
	maxBytes := ep.MaxResponseBytes
	if maxBytes <= 0 {
		maxBytes = defaultKsqlMaxResponseBytes
	}
	return &ksqlClient{
		baseURLs:         urls,
		username:         ep.Username,
		password:         ep.Password,
		maxResponseBytes: maxBytes,
		http: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}, nil
}

// ksqlTLSTransport builds an http.RoundTripper honoring the endpoint's TLS
// settings. It returns nil (use http defaults) when no TLS fields are set.
func ksqlTLSTransport(ep *appconfig.KsqlEndpoint) (http.RoundTripper, error) {
	if ep.TLSCAPath == "" && ep.TLSCertPath == "" && ep.TLSKeyPath == "" {
		return nil, nil
	}
	tlsCfg := &tls.Config{}
	if ep.TLSCAPath != "" {
		pem, err := os.ReadFile(ep.TLSCAPath)
		if err != nil {
			return nil, fmt.Errorf("reading ksql CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("ksql CA cert %q contains no valid certificates", ep.TLSCAPath)
		}
		tlsCfg.RootCAs = pool
	}
	if ep.TLSCertPath != "" && ep.TLSKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(ep.TLSCertPath, ep.TLSKeyPath)
		if err != nil {
			return nil, fmt.Errorf("loading ksql client cert/key: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	return &http.Transport{TLSClientConfig: tlsCfg}, nil
}

// ksqlClientForContext resolves the ksqlDB endpoint for the active context and
// returns a client. When no endpoint is configured it returns a
// KsqlNotConfiguredError.
func (kp KafkaDataSourceKaf) ksqlClient() (*ksqlClient, error) {
	c, err := newKsqlClient(loadKsqlEndpoint(kp.GetContext()))
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, api.KsqlNotConfiguredError{}
	}
	return c, nil
}

// setAuthHeaders applies Accept, Content-Type and basic auth to a ksqlDB request.
func (c *ksqlClient) setAuthHeaders(req *http.Request, hasBody bool) {
	req.Header.Set("Accept", ksqlAcceptType)
	if hasBody {
		req.Header.Set("Content-Type", ksqlAcceptType)
	}
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}
}

// doRaw executes a request, trying each configured base URL in turn on
// connection-level failures. It returns the HTTP status and the (size-limited)
// response body. An error is returned only when every endpoint fails to connect
// (KsqlNoInstancesError) — an HTTP response, even 4xx/5xx, is returned without
// error so callers can interpret the body.
func (c *ksqlClient) doRaw(ctx context.Context, method, path string, body interface{}) (int, []byte, error) {
	var reqBody []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, fmt.Errorf("encoding ksql request: %w", err)
		}
		reqBody = b
	}

	var connErr error
	n := len(c.baseURLs)
	for i := 0; i < n; i++ {
		idx := (c.lastGood + i) % n
		base := c.baseURLs[idx]

		var rdr io.Reader
		if reqBody != nil {
			rdr = bytes.NewReader(reqBody)
		}
		req, err := http.NewRequest(method, base+path, rdr)
		if err != nil {
			return 0, nil, err
		}
		if ctx != nil {
			req = req.WithContext(ctx)
		}
		c.setAuthHeaders(req, reqBody != nil)

		resp, err := c.http.Do(req)
		if err != nil {
			// Connection-level failure: remember and try the next URL.
			connErr = err
			continue
		}
		c.lastGood = idx
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, c.maxResponseBytes))
		resp.Body.Close()
		return resp.StatusCode, respBody, nil
	}
	return 0, nil, api.KsqlNoInstancesError{Configured: n, Cause: connErr}
}

// doPost posts a body to a ksqlDB path and decodes a 2xx response into out.
// A non-2xx response is mapped to a *KsqlServerError parsed from the ksqlDB
// error body.
func (c *ksqlClient) doPost(path string, body, out interface{}) error {
	status, respBody, err := c.doRaw(context.Background(), http.MethodPost, path, body)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return ksqlServerError(status, respBody)
	}
	if out != nil && len(respBody) > 0 {
		return json.Unmarshal(respBody, out)
	}
	return nil
}

// openStream opens a streaming POST (used for /query). The returned body is left
// open for incremental reads and must be closed by the caller; cancelling ctx
// closes the connection (terminating the server-side query). Reads are bounded
// by the configured response size limit.
func (c *ksqlClient) openStream(ctx context.Context, path string, body interface{}) (io.ReadCloser, error) {
	var reqBody []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encoding ksql request: %w", err)
		}
		reqBody = b
	}

	var connErr error
	n := len(c.baseURLs)
	for i := 0; i < n; i++ {
		idx := (c.lastGood + i) % n
		base := c.baseURLs[idx]

		var rdr io.Reader
		if reqBody != nil {
			rdr = bytes.NewReader(reqBody)
		}
		req, err := http.NewRequest(http.MethodPost, base+path, rdr)
		if err != nil {
			return nil, err
		}
		req = req.WithContext(ctx)
		c.setAuthHeaders(req, reqBody != nil)

		resp, err := c.http.Do(req)
		if err != nil {
			connErr = err
			continue
		}
		c.lastGood = idx
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			errBody, _ := io.ReadAll(io.LimitReader(resp.Body, c.maxResponseBytes))
			resp.Body.Close()
			return nil, ksqlServerError(resp.StatusCode, errBody)
		}
		return limitedReadCloser{r: io.LimitReader(resp.Body, c.maxResponseBytes), c: resp.Body}, nil
	}
	return nil, api.KsqlNoInstancesError{Configured: n, Cause: connErr}
}

// limitedReadCloser applies a read bound while delegating Close to the wrapped
// body so cancelling the context still tears down the connection.
type limitedReadCloser struct {
	r io.Reader
	c io.Closer
}

func (l limitedReadCloser) Read(p []byte) (int, error) { return l.r.Read(p) }
func (l limitedReadCloser) Close() error               { return l.c.Close() }

// ksqlServerError parses a ksqlDB {error_code, message} error body into a typed
// KsqlServerError, falling back to the raw body when unparseable.
func ksqlServerError(status int, body []byte) error {
	e := api.KsqlServerError{StatusCode: status, Raw: strings.TrimSpace(string(body))}
	var parsed struct {
		ErrorCode int    `json:"error_code"`
		Message   string `json:"message"`
	}
	if json.Unmarshal(body, &parsed) == nil {
		e.ErrorCode = parsed.ErrorCode
		e.Message = parsed.Message
	}
	return e
}

// asKsqlServerError extracts a KsqlServerError from an error chain.
func asKsqlServerError(err error) (api.KsqlServerError, bool) {
	var e api.KsqlServerError
	if errors.As(err, &e) {
		return e, true
	}
	return api.KsqlServerError{}, false
}
