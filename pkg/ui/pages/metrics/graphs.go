package metrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/metrics/graphs"
	"github.com/Benny93/kafui/pkg/metrics/promquery"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// graphResultMsg carries the outcome of an executed graph back to the page.
type graphResultMsg struct {
	id     string
	result *promquery.Result
	err    error
}

// graphPicker is the metrics-page graph browser (MM-15). It lists the built-in
// catalog when a time-series backend (TimeSeriesURLs) is configured for the
// active cluster; without storage the picker stays hidden and the MM-10
// sparklines remain the built-in fallback.
type graphPicker struct {
	catalog   graphs.Catalog
	available []graphs.Graph
	cursor    int
	visible   bool

	// Parameter entry via an inline text input, collected one param at a time.
	prompting bool
	input     textinput.Model
	pending   graphs.Graph
	paramKeys []string
	paramIdx  int
	params    map[string]string

	lastID string
	result *promquery.Result
	runErr error
}

func newGraphPicker() *graphPicker {
	ti := textinput.New()
	ti.Prompt = "> "
	return &graphPicker{catalog: graphs.DefaultCatalog(), input: ti}
}

// setStorage refreshes the available graph list from whether a backend exists.
func (g *graphPicker) setStorage(hasStorage bool) {
	g.available = g.catalog.Available(hasStorage)
	if g.cursor >= len(g.available) {
		g.cursor = 0
	}
}

func (g *graphPicker) hasGraphs() bool { return len(g.available) > 0 }

func (g *graphPicker) moveCursor(delta int) {
	if len(g.available) == 0 {
		return
	}
	g.cursor = (g.cursor + delta + len(g.available)) % len(g.available)
}

// selected returns the currently highlighted graph.
func (g *graphPicker) selected() (graphs.Graph, bool) {
	if g.cursor < 0 || g.cursor >= len(g.available) {
		return graphs.Graph{}, false
	}
	return g.available[g.cursor], true
}

// beginParams starts sequential parameter entry for a graph, or returns true
// when the graph needs no parameters (caller should run immediately).
func (g *graphPicker) beginParams(gr graphs.Graph) (ready bool) {
	if len(gr.Params) == 0 {
		g.pending = gr
		g.params = map[string]string{}
		return true
	}
	g.pending = gr
	g.paramKeys = append([]string(nil), gr.Params...)
	g.paramIdx = 0
	g.params = map[string]string{}
	g.prompting = true
	g.input.SetValue("")
	g.input.Placeholder = gr.Params[0]
	g.input.Focus()
	return false
}

// submitParam records the current input and advances; it returns true when all
// params are collected and the graph is ready to run.
func (g *graphPicker) submitParam() (ready bool) {
	if !g.prompting {
		return false
	}
	g.params[g.paramKeys[g.paramIdx]] = strings.TrimSpace(g.input.Value())
	g.paramIdx++
	if g.paramIdx >= len(g.paramKeys) {
		g.prompting = false
		g.input.Blur()
		return true
	}
	g.input.SetValue("")
	g.input.Placeholder = g.paramKeys[g.paramIdx]
	return false
}

func (g *graphPicker) cancelParams() {
	g.prompting = false
	g.input.Blur()
}

// runCmd executes the pending graph against the backend as a tea.Cmd.
func (g *graphPicker) runCmd(client *promquery.Client, cluster string) tea.Cmd {
	gr := g.pending
	params := g.params
	cat := g.catalog
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		res, err := cat.Execute(ctx, client, gr.ID, cluster, params, time.Time{}, time.Time{})
		return graphResultMsg{id: gr.ID, result: res, err: err}
	}
}

// render draws the graph list and the last result. spark is reused from the
// page to draw range (matrix) results.
func (g *graphPicker) render(b *strings.Builder, muted, header func(string) string, spark grapher) {
	b.WriteString("\n")
	b.WriteString(header("Graphs"))
	b.WriteString("\n")
	if !g.hasGraphs() {
		b.WriteString(muted("No time-series backend configured (set metrics.timeSeriesUrls) — sparklines above are the built-in trend."))
		b.WriteString("\n")
		return
	}
	for i, gr := range g.available {
		marker := "  "
		if i == g.cursor {
			marker = "> "
		}
		b.WriteString(fmt.Sprintf("%s%s (%s)\n", marker, gr.Title, gr.Kind))
	}
	if g.prompting {
		b.WriteString(muted(fmt.Sprintf("set %q: ", g.paramKeys[g.paramIdx])))
		b.WriteString(g.input.View())
		b.WriteString("\n")
		return
	}
	if g.runErr != nil {
		b.WriteString(muted("graph error: " + g.runErr.Error()))
		b.WriteString("\n")
		return
	}
	if g.result != nil {
		b.WriteString(renderResult(g.lastID, g.result, spark, muted))
	}
}

// grapher is the minimal sparkline surface renderResult needs.
type grapher interface {
	SetData([]float64)
	View() string
}

// renderResult turns a query result into a compact display: matrix → sparkline
// of the first series, vector/scalar → value lines.
func renderResult(id string, res *promquery.Result, spark grapher, muted func(string) string) string {
	var b strings.Builder
	b.WriteString(muted("result: " + id))
	b.WriteString("\n")
	switch res.Type {
	case promquery.ResultMatrix:
		if len(res.Matrix) == 0 {
			return b.String() + muted("(no data)") + "\n"
		}
		s := res.Matrix[0]
		vals := make([]float64, len(s.Points))
		for i, p := range s.Points {
			vals[i] = p.V
		}
		spark.SetData(vals)
		b.WriteString(spark.View())
		b.WriteString("\n")
	case promquery.ResultVector:
		if len(res.Vector) == 0 {
			return b.String() + muted("(no data)") + "\n"
		}
		for _, s := range res.Vector {
			b.WriteString(fmt.Sprintf("  %s = %.2f\n", labelString(s.Metric), s.Point.V))
		}
	case promquery.ResultScalar:
		if res.Scalar != nil {
			b.WriteString(fmt.Sprintf("  %.2f\n", res.Scalar.V))
		}
	}
	return b.String()
}

func labelString(m map[string]string) string {
	if len(m) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(m))
	for k, v := range m {
		if k == "__name__" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return "{" + strings.Join(parts, ",") + "}"
}
