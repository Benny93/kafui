package metrics

import (
	"strings"
	"testing"

	"github.com/Benny93/kafui/pkg/metrics/promquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphPickerStorageGating(t *testing.T) {
	p := newGraphPicker()
	p.setStorage(false)
	assert.False(t, p.hasGraphs(), "no graphs without storage")
	p.setStorage(true)
	assert.True(t, p.hasGraphs(), "graphs listed with storage")
}

func TestGraphPickerParamFlow(t *testing.T) {
	p := newGraphPicker()
	p.setStorage(true)

	// Find the topic-parameterized graph (needs one param).
	var idx = -1
	for i, g := range p.available {
		if len(g.Params) == 1 {
			idx = i
			break
		}
	}
	require.GreaterOrEqual(t, idx, 0)
	p.cursor = idx
	gr, ok := p.selected()
	require.True(t, ok)

	// beginParams returns false (needs input) and enters prompting mode.
	ready := p.beginParams(gr)
	assert.False(t, ready)
	assert.True(t, p.prompting)

	// Supply the param; submit completes and marks ready.
	p.input.SetValue("orders")
	ready = p.submitParam()
	assert.True(t, ready)
	assert.False(t, p.prompting)
	assert.Equal(t, "orders", p.params[gr.Params[0]])
}

func TestGraphPickerParamlessReadyImmediately(t *testing.T) {
	p := newGraphPicker()
	p.setStorage(true)
	// Pick a param-less graph.
	var gr = p.available[0]
	for _, g := range p.available {
		if len(g.Params) == 0 {
			gr = g
			break
		}
	}
	require.Empty(t, gr.Params)
	assert.True(t, p.beginParams(gr))
	assert.False(t, p.prompting)
}

// fakeSpark satisfies the grapher interface for renderResult.
type fakeSpark struct{ data []float64 }

func (f *fakeSpark) SetData(d []float64) { f.data = d }
func (f *fakeSpark) View() string        { return "[spark]" }

func TestRenderResultMatrix(t *testing.T) {
	res := &promquery.Result{
		Type: promquery.ResultMatrix,
		Matrix: []promquery.Series{
			{Metric: map[string]string{"topic": "orders"}, Points: []promquery.Point{{V: 1}, {V: 2}, {V: 3}}},
		},
	}
	sp := &fakeSpark{}
	out := renderResult("g", res, sp, func(s string) string { return s })
	assert.Contains(t, out, "[spark]")
	assert.Equal(t, []float64{1, 2, 3}, sp.data)
}

func TestRenderResultVector(t *testing.T) {
	res := &promquery.Result{
		Type: promquery.ResultVector,
		Vector: []promquery.Sample{
			{Metric: map[string]string{"broker_id": "1"}, Point: promquery.Point{V: 42}},
		},
	}
	out := renderResult("g", res, &fakeSpark{}, func(s string) string { return s })
	assert.Contains(t, out, "42")
	assert.True(t, strings.Contains(out, "broker_id=1"))
}
