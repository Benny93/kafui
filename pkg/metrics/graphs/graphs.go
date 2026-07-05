// Package graphs is a built-in catalog of Prometheus graph descriptions and the
// logic that renders their query templates and executes them against the
// optional time-series backend (pkg/metrics/promquery).
//
// A graph template uses {{name}} placeholders. The placeholder {{cluster}} is
// reserved and always bound to the active cluster name; any other placeholder
// must be declared in the graph's Params list and supplied at execution time.
// Templates are validated at init with a lightweight grammar check (all
// placeholders known, non-empty after dummy substitution) rather than a full
// PromQL parser, keeping the package dependency-free.
package graphs

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/metrics/promquery"
)

// GraphKind distinguishes instant queries from range queries.
type GraphKind string

const (
	KindInstant GraphKind = "instant"
	KindRange   GraphKind = "range"
)

// clusterParam is the reserved placeholder bound to the active cluster name.
const clusterParam = "cluster"

// placeholderRE matches {{name}} placeholders (name = word chars).
var placeholderRE = regexp.MustCompile(`\{\{\s*(\w+)\s*\}\}`)

// Graph is a single catalog entry.
type Graph struct {
	ID            string
	Title         string
	Template      string
	Kind          GraphKind
	DefaultPeriod time.Duration // range graphs only
	Params        []string      // required params excluding the reserved cluster
}

// UnknownGraphError is returned when an id is not in the catalog.
type UnknownGraphError struct{ ID string }

func (e UnknownGraphError) Error() string { return fmt.Sprintf("unknown graph id %q", e.ID) }

// MissingParamsError names the params a Render/Execute call did not supply.
type MissingParamsError struct {
	ID     string
	Params []string
}

func (e MissingParamsError) Error() string {
	return fmt.Sprintf("graph %q is missing required parameters: %s", e.ID, strings.Join(e.Params, ", "))
}

// Catalog is an ordered, id-indexed set of graphs.
type Catalog struct {
	byID  map[string]Graph
	order []string
}

// DefaultCatalog returns the built-in graph catalog. It panics only if the
// built-in templates are internally inconsistent, which Validate guards against
// in tests and at startup.
func DefaultCatalog() Catalog {
	return NewCatalog(builtinGraphs())
}

// builtinGraphs is the shipped set: broker disk-usage (instant + range) and a
// topic-parameterized partition-offsets range graph.
func builtinGraphs() []Graph {
	return []Graph{
		{
			ID:       "broker-disk-usage",
			Title:    "Broker disk usage (bytes)",
			Template: `sum by (broker_id) (kafka_log_size{cluster_name="{{cluster}}"})`,
			Kind:     KindInstant,
		},
		{
			ID:            "broker-disk-usage-range",
			Title:         "Broker disk usage over time (bytes)",
			Template:      `sum by (broker_id) (kafka_log_size{cluster_name="{{cluster}}"})`,
			Kind:          KindRange,
			DefaultPeriod: time.Hour,
		},
		{
			ID:            "topic-partition-offsets",
			Title:         "Topic partition offsets over time",
			Template:      `kafka_topic_partition_current_offset{cluster_name="{{cluster}}",topic="{{topic}}"}`,
			Kind:          KindRange,
			DefaultPeriod: time.Hour,
			Params:        []string{"topic"},
		},
	}
}

// NewCatalog indexes the given graphs. Duplicate ids keep the first entry.
func NewCatalog(gs []Graph) Catalog {
	c := Catalog{byID: make(map[string]Graph, len(gs))}
	for _, g := range gs {
		if _, ok := c.byID[g.ID]; ok {
			continue
		}
		c.byID[g.ID] = g
		c.order = append(c.order, g.ID)
	}
	return c
}

// List returns the catalog graphs in declaration order.
func (c Catalog) List() []Graph {
	out := make([]Graph, 0, len(c.order))
	for _, id := range c.order {
		out = append(out, c.byID[id])
	}
	return out
}

// Get returns a graph by id.
func (c Catalog) Get(id string) (Graph, bool) {
	g, ok := c.byID[id]
	return g, ok
}

// Validate checks every template: all placeholders are known (a declared param
// or the reserved cluster) and the template is non-empty after substituting
// dummy values. It returns an error listing every offending id, suitable for a
// fail-fast startup check.
func (c Catalog) Validate() error {
	var bad []string
	for _, id := range c.order {
		if err := validateGraph(c.byID[id]); err != nil {
			bad = append(bad, fmt.Sprintf("%s (%v)", id, err))
		}
	}
	if len(bad) > 0 {
		sort.Strings(bad)
		return fmt.Errorf("invalid graph templates: %s", strings.Join(bad, "; "))
	}
	return nil
}

func validateGraph(g Graph) error {
	if strings.TrimSpace(g.Template) == "" {
		return fmt.Errorf("empty template")
	}
	known := map[string]bool{clusterParam: true}
	for _, p := range g.Params {
		known[p] = true
	}
	for _, m := range placeholderRE.FindAllStringSubmatch(g.Template, -1) {
		if !known[m[1]] {
			return fmt.Errorf("unknown placeholder {{%s}}", m[1])
		}
	}
	// Ensure a fully-substituted template is non-empty (grammar sanity).
	dummy := map[string]string{}
	for p := range known {
		dummy[p] = "x"
	}
	if strings.TrimSpace(substitute(g.Template, dummy)) == "" {
		return fmt.Errorf("template renders empty")
	}
	return nil
}

// substitute replaces every {{name}} placeholder using values; missing names are
// left in place (the caller validates completeness separately).
func substitute(tmpl string, values map[string]string) string {
	return placeholderRE.ReplaceAllStringFunc(tmpl, func(match string) string {
		name := placeholderRE.FindStringSubmatch(match)[1]
		if v, ok := values[name]; ok {
			return v
		}
		return match
	})
}

// Render substitutes the cluster binding and params into a graph template. It
// returns UnknownGraphError for an unknown id and MissingParamsError naming any
// required params that were not supplied.
func (c Catalog) Render(id, cluster string, params map[string]string) (string, error) {
	g, ok := c.byID[id]
	if !ok {
		return "", UnknownGraphError{ID: id}
	}
	var missing []string
	values := map[string]string{clusterParam: cluster}
	for _, p := range g.Params {
		v, ok := params[p]
		if !ok || strings.TrimSpace(v) == "" {
			missing = append(missing, p)
			continue
		}
		values[p] = v
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return "", MissingParamsError{ID: id, Params: missing}
	}
	return substitute(g.Template, values), nil
}

// Execute renders and runs a graph via the query client. A nil client (no
// TimeSeriesURLs configured) yields api.MetricsNotConfiguredError. For range
// graphs, a zero start defaults to now-DefaultPeriod and a zero end to now;
// end<=start is rejected.
func (c Catalog) Execute(ctx context.Context, client *promquery.Client, id, cluster string, params map[string]string, start, end time.Time) (*promquery.Result, error) {
	if client == nil {
		return nil, api.MetricsNotConfiguredError{Cluster: cluster}
	}
	g, ok := c.byID[id]
	if !ok {
		return nil, UnknownGraphError{ID: id}
	}
	query, err := c.Render(id, cluster, params)
	if err != nil {
		return nil, err
	}
	if g.Kind == KindInstant {
		return client.Query(ctx, query, end)
	}
	now := time.Now()
	if end.IsZero() {
		end = now
	}
	if start.IsZero() {
		period := g.DefaultPeriod
		if period <= 0 {
			period = time.Hour
		}
		start = end.Add(-period)
	}
	if !end.After(start) {
		return nil, fmt.Errorf("graph %q: range end %s must be after start %s", id, end.Format(time.RFC3339), start.Format(time.RFC3339))
	}
	return client.QueryRange(ctx, query, start, end)
}

// Available returns the catalog graphs when a time-series backend is configured,
// or an empty slice otherwise (the graph section is hidden without storage).
func (c Catalog) Available(hasStorage bool) []Graph {
	if !hasStorage {
		return nil
	}
	return c.List()
}
