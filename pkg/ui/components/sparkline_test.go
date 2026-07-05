package components

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestRenderSparkline(t *testing.T) {
	tests := []struct {
		name  string
		data  []float64
		width int
		want  string
	}{
		{"empty", nil, 10, ""},
		{"all unknown", []float64{-1, -1}, 10, ""},
		{"flat series lowest bar", []float64{5, 5, 5}, 10, "▁▁▁"},
		{"single point", []float64{7}, 10, "▁"},
		{"ramp spans full range", []float64{0, 1, 2, 3, 4, 5, 6, 7}, 10, "▁▂▃▄▅▆▇█"},
		{"gap rendered as space", []float64{0, -1, 7}, 10, "▁ █"},
		{"width keeps rightmost", []float64{0, 0, 0, 7}, 2, "▁█"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, RenderSparkline(tt.data, tt.width))
		})
	}
}

func TestFormatBytesPerSec(t *testing.T) {
	tests := []struct {
		v    float64
		want string
	}{
		{-1, "–"},
		{0, "0 B/s"},
		{512, "512 B/s"},
		{1024, "1.0 KiB/s"},
		{1536, "1.5 KiB/s"},
		{1048576, "1.0 MiB/s"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, FormatBytesPerSec(tt.v))
	}
}

func TestFormatRate(t *testing.T) {
	assert.Equal(t, "–", FormatRate(-1))
	assert.Equal(t, "0.0/s", FormatRate(0))
	assert.Equal(t, "123.5/s", FormatRate(123.45))
}

func TestSparklineViewPlaceholder(t *testing.T) {
	s := NewSparkline(lipgloss.NewStyle())
	s.SetDimensions(20, 1)
	assert.Contains(t, s.View(), "no data")
	s.SetData([]float64{1, 2, 3})
	assert.NotContains(t, s.View(), "no data")
}
