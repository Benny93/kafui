package shared

import (
	"sync/atomic"
	"time"
)

// timeLayout is the shared display format for timestamps across pages.
const timeLayout = "2006-01-02 15:04:05"

// tzLocation holds the configured display timezone. Stored atomically so the
// formatter is safe to call from any goroutine. nil means system local time.
// ponytail: a package-level formatting setting (not data access) — a global is
// the lazy-correct fit here; pages call FormatTimestamp without threading a tz.
var tzLocation atomic.Pointer[time.Location]

// SetTimezone configures the display timezone from a config value: "local" (or
// ""), "UTC", or any IANA name (e.g. "Europe/Berlin"). An unknown name falls
// back to local time and returns an error so the caller can warn.
func SetTimezone(name string) error {
	switch name {
	case "", "local", "Local":
		tzLocation.Store(nil)
		return nil
	case "UTC", "utc":
		tzLocation.Store(time.UTC)
		return nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		tzLocation.Store(nil)
		return err
	}
	tzLocation.Store(loc)
	return nil
}

// FormatTimestamp renders t in the configured display timezone.
func FormatTimestamp(t time.Time) string {
	if loc := tzLocation.Load(); loc != nil {
		return t.In(loc).Format(timeLayout)
	}
	return t.Local().Format(timeLayout)
}
