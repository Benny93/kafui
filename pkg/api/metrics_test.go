package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeSeriesSummary(t *testing.T) {
	base := time.Unix(0, 0)
	pt := func(i int, v float64) MetricPoint { return MetricPoint{Time: base.Add(time.Duration(i) * time.Second), Value: v} }

	t.Run("empty series not OK", func(t *testing.T) {
		s := TimeSeries{}.Summary()
		assert.False(t, s.OK)
		assert.Equal(t, 0, s.Count)
	})

	t.Run("aggregates min/max/avg/last", func(t *testing.T) {
		ts := TimeSeries{Points: []MetricPoint{pt(0, 10), pt(1, 30), pt(2, 20)}}
		s := ts.Summary()
		assert.True(t, s.OK)
		assert.Equal(t, 3, s.Count)
		assert.Equal(t, 10.0, s.Min)
		assert.Equal(t, 30.0, s.Max)
		assert.Equal(t, 20.0, s.Avg)
		assert.Equal(t, 20.0, s.Last)
	})

	t.Run("skips unknown (negative) samples", func(t *testing.T) {
		ts := TimeSeries{Points: []MetricPoint{pt(0, RateUnknown), pt(1, 40), pt(2, RateUnknown), pt(3, 60)}}
		s := ts.Summary()
		assert.True(t, s.OK)
		assert.Equal(t, 2, s.Count)
		assert.Equal(t, 40.0, s.Min)
		assert.Equal(t, 60.0, s.Max)
		assert.Equal(t, 50.0, s.Avg)
		assert.Equal(t, 60.0, s.Last)
	})

	t.Run("all unknown yields not OK", func(t *testing.T) {
		ts := TimeSeries{Points: []MetricPoint{pt(0, RateUnknown), pt(1, RateUnknown)}}
		assert.False(t, ts.Summary().OK)
	})
}

func TestTimeSeriesValues(t *testing.T) {
	ts := TimeSeries{Points: []MetricPoint{{Value: 1}, {Value: 2}, {Value: 3}}}
	assert.Equal(t, []float64{1, 2, 3}, ts.Values())
}
