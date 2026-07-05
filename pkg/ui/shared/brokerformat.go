package shared

import (
	"fmt"
	"strconv"
	"strings"
)

// Skew severity levels returned by SkewSeverity, used to pick a display style.
const (
	SkewNone    = 0 // below the warning threshold (or absent)
	SkewWarning = 1 // |skew| >= 10%
	SkewError   = 2 // |skew| >= 20%
)

// FormatBytes2dp renders a byte count as a human-readable size with two decimal
// places using binary (1024) steps and SI-style unit labels (B, KB, MB, GB, …).
// Bytes below 1 KB are shown as a whole number, e.g. "512 B", larger values as
// "1.00 GB". Negative values are returned verbatim (as a signed byte count).
func FormatBytes2dp(n int64) string {
	if n < 0 {
		return strconv.FormatInt(n, 10) + " B"
	}
	const unit = 1024.0
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KB", "MB", "GB", "TB", "PB", "EB"}
	val := float64(n)
	i := -1
	for val >= unit && i < len(units)-1 {
		val /= unit
		i++
	}
	return fmt.Sprintf("%.2f %s", val, units[i])
}

// FormatDiskUsage renders a broker's disk usage cell: "<size 2dp>, N segment(s)"
// or "N/A" when no log-dir data is available (segment count is zero).
func FormatDiskUsage(segmentSize int64, segmentCount int) string {
	if segmentCount <= 0 {
		return "N/A"
	}
	return fmt.Sprintf("%s, %d segment(s)", FormatBytes2dp(segmentSize), segmentCount)
}

// FormatISR renders the in-sync-replica cell as "isr/replica". The cell is empty
// when the replica count is unavailable (<= 0). alert reports whether ISR is
// below the replica count (i.e. under-replicated) so the caller can style it.
func FormatISR(inSync, replica int) (text string, alert bool) {
	if replica <= 0 {
		return "", false
	}
	return fmt.Sprintf("%d/%d", inSync, replica), inSync < replica
}

// FormatSkew renders a skew percentage as "%.2f%%" or "-" when absent (nil).
func FormatSkew(skew *float64) string {
	if skew == nil {
		return "-"
	}
	return fmt.Sprintf("%.2f%%", *skew)
}

// SkewSeverity classifies a skew value for styling: SkewError when |skew| >= 20%,
// SkewWarning when |skew| >= 10%, otherwise SkewNone. Absent (nil) is SkewNone.
func SkewSeverity(skew *float64) int {
	if skew == nil {
		return SkewNone
	}
	abs := *skew
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs >= 20:
		return SkewError
	case abs >= 10:
		return SkewWarning
	default:
		return SkewNone
	}
}

// ConfigValueMask is the placeholder shown in place of a sensitive config value.
const ConfigValueMask = "**********"

// FormatConfigValue renders a broker config value for display:
//   - sensitive entries are masked as ConfigValueMask;
//   - keys ending ".bytes" with a positive integer value are shown as a
//     human-readable size (non-positive / non-numeric values pass through as-is);
//   - keys ending ".ms" get an " ms" suffix appended;
//   - everything else is returned unchanged.
func FormatConfigValue(name, value string, sensitive bool) string {
	if sensitive {
		return ConfigValueMask
	}
	switch {
	case strings.HasSuffix(name, ".bytes"):
		if n, err := strconv.ParseInt(value, 10, 64); err == nil && n > 0 {
			return FormatBytes2dp(n)
		}
		return value
	case strings.HasSuffix(name, ".ms"):
		if value == "" {
			return value
		}
		return value + " ms"
	default:
		return value
	}
}
