package components

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/charmbracelet/lipgloss"
)

// sparkLevels are the eight block glyphs used to draw a compact sparkline,
// lowest to highest.
var sparkLevels = []rune("▁▂▃▄▅▆▇█")

// RenderSparkline renders a series of values as a unicode block sparkline. It
// keeps at most `width` (rightmost) samples. Negative values are treated as
// gaps ("unknown", api.RateUnknown) and rendered as spaces. A series with no
// known values renders as an empty string (callers show a placeholder).
func RenderSparkline(data []float64, width int) string {
	if len(data) == 0 {
		return ""
	}
	if width > 0 && len(data) > width {
		data = data[len(data)-width:]
	}
	min, max, any := 0.0, 0.0, false
	for _, v := range data {
		if v < 0 {
			continue
		}
		if !any {
			min, max, any = v, v, true
			continue
		}
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if !any {
		return ""
	}
	span := max - min
	var b strings.Builder
	for _, v := range data {
		if v < 0 {
			b.WriteRune(' ')
			continue
		}
		idx := 0
		if span > 0 {
			idx = int((v - min) / span * float64(len(sparkLevels)-1))
			if idx < 0 {
				idx = 0
			}
			if idx >= len(sparkLevels) {
				idx = len(sparkLevels) - 1
			}
		}
		b.WriteRune(sparkLevels[idx])
	}
	return b.String()
}

// FormatBytesPerSec formats a bytes-per-second rate with IEC units. A negative
// value is unknown and renders as an en dash.
func FormatBytesPerSec(v float64) string {
	if v < 0 {
		return "–"
	}
	const unit = 1024.0
	if v < unit {
		return fmt.Sprintf("%.0f B/s", v)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	val, i := v, -1
	for val >= unit && i < len(units)-1 {
		val /= unit
		i++
	}
	return fmt.Sprintf("%.1f %s/s", val, units[i])
}

// FormatRate formats a plain per-second rate (e.g. messages/s). Negative is unknown.
func FormatRate(v float64) string {
	if v < 0 {
		return "–"
	}
	return fmt.Sprintf("%.1f/s", v)
}

// Sparkline is a reusable component that renders a value series as a sparkline,
// optionally followed by a min/max/avg summary. Styling is injected from the
// role-based palette (no hex literals here).
type Sparkline struct {
	core.BaseComponent
	data  []float64
	style lipgloss.Style
	label string
}

// NewSparkline builds a Sparkline rendered in the given (role-based) style.
func NewSparkline(style lipgloss.Style) *Sparkline {
	return &Sparkline{style: style}
}

// SetData replaces the series.
func (s *Sparkline) SetData(data []float64) { s.data = data }

// SetLabel sets an optional leading label.
func (s *Sparkline) SetLabel(label string) { s.label = label }

// View renders the sparkline at the component width, degrading to a
// placeholder for an empty/all-unknown series.
func (s *Sparkline) View() string {
	spark := RenderSparkline(s.data, s.GetWidth())
	if spark == "" {
		spark = "no data"
	}
	line := s.style.Render(spark)
	if s.label != "" {
		return s.label + " " + line
	}
	return line
}
