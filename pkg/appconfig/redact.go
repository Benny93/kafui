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
	if providerRef.MatchString(strings.TrimSpace(value)) {
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
