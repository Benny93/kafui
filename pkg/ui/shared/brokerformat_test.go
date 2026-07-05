package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatBytes2dp(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1048588, "1.00 MB"},
		{1073741824, "1.00 GB"},
		{-1, "-1 B"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, FormatBytes2dp(c.in), "FormatBytes2dp(%d)", c.in)
	}
}

func TestFormatDiskUsage(t *testing.T) {
	assert.Equal(t, "N/A", FormatDiskUsage(0, 0), "absent → N/A")
	assert.Equal(t, "N/A", FormatDiskUsage(1234, 0), "zero segments → N/A")
	assert.Equal(t, "1.00 GB, 3 segment(s)", FormatDiskUsage(1073741824, 3))
}

func TestFormatISR(t *testing.T) {
	cases := []struct {
		name        string
		inSync      int
		replica     int
		wantText    string
		wantAlert   bool
	}{
		{"all in sync", 30, 30, "30/30", false},
		{"under-replicated", 25, 28, "25/28", true},
		{"replica unavailable → empty", 0, 0, "", false},
		{"replica count zero → empty", 5, 0, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			text, alert := FormatISR(c.inSync, c.replica)
			assert.Equal(t, c.wantText, text)
			assert.Equal(t, c.wantAlert, alert)
		})
	}
}

func TestFormatSkew(t *testing.T) {
	assert.Equal(t, "-", FormatSkew(nil))
	v := 3.2
	assert.Equal(t, "3.20%", FormatSkew(&v))
	n := -10.3
	assert.Equal(t, "-10.30%", FormatSkew(&n))
}

func TestSkewSeverity(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	cases := []struct {
		in   *float64
		want int
	}{
		{nil, SkewNone},
		{f(9.99), SkewNone},
		{f(10), SkewWarning},
		{f(19.99), SkewWarning},
		{f(20), SkewError},
		{f(-20), SkewError},
		{f(-10), SkewWarning},
		{f(-9.99), SkewNone},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, SkewSeverity(c.in), "SkewSeverity(%v)", c.in)
	}
}

func TestFormatConfigValue(t *testing.T) {
	cases := []struct {
		name      string
		key       string
		value     string
		sensitive bool
		want      string
	}{
		{"bytes → human", "log.segment.bytes", "1073741824", false, "1.00 GB"},
		{"negative bytes as-is", "log.retention.bytes", "-1", false, "-1"},
		{"non-numeric bytes as-is", "some.bytes", "abc", false, "abc"},
		{"ms suffix", "log.retention.ms", "604800000", false, "604800000 ms"},
		{"sensitive masked", "ssl.keystore.password", "", true, ConfigValueMask},
		{"sensitive masked over value", "ssl.key", "secret", true, ConfigValueMask},
		{"plain passthrough", "compression.type", "producer", false, "producer"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, FormatConfigValue(c.key, c.value, c.sensitive))
		})
	}
}
