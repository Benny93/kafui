package shared

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatTimestampUTC(t *testing.T) {
	require.NoError(t, SetTimezone("UTC"))
	defer SetTimezone("local")

	ts := time.Date(2026, 7, 4, 9, 30, 0, 0, time.UTC)
	assert.Equal(t, "2026-07-04 09:30:00", FormatTimestamp(ts))
}

func TestFormatTimestampNamedZone(t *testing.T) {
	require.NoError(t, SetTimezone("Europe/Berlin"))
	defer SetTimezone("local")

	ts := time.Date(2026, 7, 4, 9, 30, 0, 0, time.UTC) // summer -> UTC+2
	assert.Equal(t, "2026-07-04 11:30:00", FormatTimestamp(ts))
}

func TestFormatTimestampInvalidFallsBackToLocal(t *testing.T) {
	err := SetTimezone("Not/AZone")
	assert.Error(t, err, "invalid zone should report an error")
	defer SetTimezone("local")
	// Falls back to local: formatting must still succeed (non-empty).
	assert.NotEmpty(t, FormatTimestamp(time.Now()))
}

func TestFormatTimestampLocalDefault(t *testing.T) {
	require.NoError(t, SetTimezone("local"))
	ts := time.Now()
	assert.Equal(t, ts.Local().Format(timeLayout), FormatTimestamp(ts))
}
