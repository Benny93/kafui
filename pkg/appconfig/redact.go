package appconfig

import (
	"regexp"
	"strings"
)

const redactPlaceholder = "**********"

// defaultRedactPatterns are matched case-insensitively against config keys.
var defaultRedactPatterns = []string{
	"password", "secret", "token", "key", "credentials", "passphrase",
	"sasl.jaas.config", "ssl.*password", "basic.auth.user.info",
	"aws.access", "aws.secret", "aws.session",
}

// providerRef matches externalized secret references like ${env:MY_VAR} or
// ${file:/path:key}; these are passed through unmasked.
var providerRef = regexp.MustCompile(`^\$\{[^:]+:.*\}$`)

// RedactPlaceholder returns the marker Redactor substitutes for a secret value.
// Exported so display surfaces outside the config view — such as a masked form
// field standing in for a stored credential — show the same marker rather than
// inventing their own.
func RedactPlaceholder() string { return redactPlaceholder }

// IsProviderRef reports whether value is an externalized secret reference such
// as ${env:MY_VAR} or ${file:/path:key}. Such a value is a pointer, not secret
// material: it is safe to display and must round-trip verbatim through an edit
// form instead of being masked or treated as a stored credential. This is the
// same check Redact applies before masking, exposed for those other surfaces.
func IsProviderRef(value string) bool {
	return providerRef.MatchString(strings.TrimSpace(value))
}

// Redactor masks secret values in displayed configuration.
type Redactor struct {
	enabled  bool
	patterns []*regexp.Regexp
}

// NewRedactor builds a Redactor from settings. When s.Patterns is set it fully
// replaces the defaults; a glob-ish "ssl.*password" is treated as a substring
// regex where "*" means ".*".
func NewRedactor(s RedactionSettings) *Redactor {
	raw := defaultRedactPatterns
	if len(s.Patterns) > 0 {
		raw = s.Patterns
	}
	compiled := make([]*regexp.Regexp, 0, len(raw))
	for _, p := range raw {
		// Escape everything, then re-enable "*" as a wildcard.
		esc := regexp.QuoteMeta(strings.ToLower(p))
		esc = strings.ReplaceAll(esc, `\*`, `.*`)
		if re, err := regexp.Compile(esc); err == nil {
			compiled = append(compiled, re)
		}
	}
	return &Redactor{enabled: s.Enabled, patterns: compiled}
}

// Redact returns the value masked when key matches a secret pattern.
// Externalized ${provider:...} references pass through unmasked.
func (r *Redactor) Redact(key, value string) string {
	if !r.enabled || value == "" {
		return value
	}
	if IsProviderRef(value) {
		return value
	}
	lk := strings.ToLower(key)
	for _, re := range r.patterns {
		if re.MatchString(lk) {
			return redactPlaceholder
		}
	}
	return value
}
