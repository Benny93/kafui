// Package jolokia is an HTTP-bridge collector for JMX metrics exposed through a
// Jolokia agent. It is the only supported path for the metrics config Type
// "JMX" (native JMX/RMI from Go is not implemented); see the metrics plan's TUI
// adaptation notes.
//
// It performs a bulk read of the kafka.server domain and maps each numeric MBean
// attribute to a metric named domain_firstProp_attribute (dots and dashes
// replaced by underscores), attaching the remaining MBean properties as labels.
// Non-numeric attributes are skipped. Only the standard library is used.
package jolokia

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// defaultDomain is the JMX domain read for Kafka broker metrics.
const defaultDomain = "kafka.server"

// defaultTimeout bounds a single Jolokia read.
const defaultTimeout = 10 * time.Second

// Sample is one numeric metric derived from an MBean attribute.
type Sample struct {
	Name   string
	Labels map[string]string
	Value  float64
}

// Client reads metrics from a Jolokia agent base URL (e.g.
// http://broker:8778/jolokia). Basic-auth credentials are optional.
type Client struct {
	baseURL  string
	username string
	password string
	domain   string
	http     *http.Client
}

// New builds a client. baseURL should be the Jolokia agent root (no trailing
// slash required).
func New(baseURL, username, password string) *Client {
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		domain:   defaultDomain,
		http:     &http.Client{Timeout: defaultTimeout},
	}
}

// jolokiaResponse is the envelope of a Jolokia read response. For a wildcard
// (pattern) read, Value maps each MBean name to its attribute map.
type jolokiaResponse struct {
	Status int                        `json:"status"`
	Value  map[string]json.RawMessage `json:"value"`
	Error  string                     `json:"error"`
}

// Collect performs the bulk read and returns the derived numeric samples.
func (c *Client) Collect(ctx context.Context) ([]Sample, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	url := c.baseURL + "/read/" + c.domain + ":*"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jolokia read: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jolokia read: HTTP %d", resp.StatusCode)
	}
	return Parse(body)
}

// Parse turns a Jolokia read-response body into samples. It is exported so the
// naming/label/skip rules are directly testable against JSON fixtures.
func Parse(body []byte) ([]Sample, error) {
	var jr jolokiaResponse
	if err := json.Unmarshal(body, &jr); err != nil {
		return nil, fmt.Errorf("decoding jolokia response: %w", err)
	}
	if jr.Error != "" {
		return nil, fmt.Errorf("jolokia error: %s", jr.Error)
	}
	var out []Sample
	// Stable ordering for deterministic output.
	names := make([]string, 0, len(jr.Value))
	for name := range jr.Value {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, mbean := range names {
		domain, first, labels := parseMBean(mbean)
		if domain == "" {
			continue
		}
		var attrs map[string]json.RawMessage
		if err := json.Unmarshal(jr.Value[mbean], &attrs); err != nil {
			continue // not an attribute map; skip
		}
		attrNames := make([]string, 0, len(attrs))
		for a := range attrs {
			attrNames = append(attrNames, a)
		}
		sort.Strings(attrNames)
		for _, attr := range attrNames {
			f, ok := numeric(attrs[attr])
			if !ok {
				continue // non-numeric attribute skipped
			}
			out = append(out, Sample{
				Name:   sanitize(domain) + "_" + sanitize(first) + "_" + attr,
				Labels: cloneLabels(labels),
				Value:  f,
			})
		}
	}
	return out, nil
}

// parseMBean splits "domain:k1=v1,k2=v2" into the domain, the first property's
// value, and the remaining properties as labels.
func parseMBean(mbean string) (domain, firstProp string, labels map[string]string) {
	i := strings.IndexByte(mbean, ':')
	if i < 0 {
		return "", "", nil
	}
	domain = mbean[:i]
	props := strings.Split(mbean[i+1:], ",")
	labels = map[string]string{}
	for idx, p := range props {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		if idx == 0 {
			firstProp = v
			continue
		}
		labels[sanitize(k)] = v
	}
	return domain, firstProp, labels
}

// numeric extracts a float64 from a JSON value, reporting false for
// non-numeric (string, bool, object, array, null) values.
func numeric(raw json.RawMessage) (float64, bool) {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return 0, false
	}
	f, ok := v.(float64)
	return f, ok
}

// sanitize replaces dots and dashes with underscores for Prometheus-safe names.
func sanitize(s string) string {
	return strings.NewReplacer(".", "_", "-", "_").Replace(s)
}

func cloneLabels(m map[string]string) map[string]string {
	if len(m) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
