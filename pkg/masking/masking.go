// Package masking implements data-masking rules applied at display time to
// already-rendered message key/value strings. It never mutates raw bytes.
package masking

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// maskedLiteral is the placeholder used by ActionReplace.
const maskedLiteral = "***DATA_MASKED***"

// Action is what a rule does to the fields it affects.
type Action string

const (
	ActionMask    Action = "MASK"    // class-wise char replacement
	ActionReplace Action = "REPLACE" // replace affected scalars with ***DATA_MASKED***
	ActionRemove  Action = "REMOVE"  // delete affected fields
)

// Target selects whether a rendering is a message key or value.
type Target int

const (
	Key Target = iota
	Value
)

// Rule is one masking rule.
type Rule struct {
	Action     Action
	Keys       bool     // rule applies to message keys
	Values     bool     // rule applies to message values
	Fields     []string // explicit JSON field names to affect (XOR with FieldRegex)
	FieldRegex string   // regex matching JSON field names (XOR with Fields); neither set => all fields
}

// Validate reports config errors.
func (r Rule) Validate() error {
	switch r.Action {
	case ActionMask, ActionReplace, ActionRemove:
	default:
		return fmt.Errorf("unknown action %q", r.Action)
	}
	if !r.Keys && !r.Values {
		return fmt.Errorf("rule must set at least one of Keys or Values")
	}
	if len(r.Fields) > 0 && r.FieldRegex != "" {
		return fmt.Errorf("Fields and FieldRegex are mutually exclusive")
	}
	if r.FieldRegex != "" {
		if _, err := regexp.Compile(r.FieldRegex); err != nil {
			return fmt.Errorf("invalid FieldRegex %q: %w", r.FieldRegex, err)
		}
	}
	return nil
}

// applies reports whether the rule covers the given target.
func (r Rule) applies(t Target) bool {
	return (t == Key && r.Keys) || (t == Value && r.Values)
}

// hasSelectors reports whether the rule limits itself to specific fields.
func (r Rule) hasSelectors() bool {
	return len(r.Fields) > 0 || r.FieldRegex != ""
}

// ParseRules parses DSL lines from config into Rules.
// DSL per line: "ACTION scope=key|value|both [field=a,b,c | regex=REGEX]".
func ParseRules(lines []string) ([]Rule, error) {
	var rules []Rule
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		tokens := strings.Fields(line)
		r := Rule{Action: Action(tokens[0])}
		for _, tok := range tokens[1:] {
			kv := strings.SplitN(tok, "=", 2)
			if len(kv) != 2 {
				return nil, fmt.Errorf("line %d: malformed token %q", i+1, tok)
			}
			key, val := kv[0], kv[1]
			switch key {
			case "scope":
				switch val {
				case "key":
					r.Keys = true
				case "value":
					r.Values = true
				case "both":
					r.Keys, r.Values = true, true
				default:
					return nil, fmt.Errorf("line %d: unknown scope %q", i+1, val)
				}
			case "field":
				r.Fields = strings.Split(val, ",")
			case "regex":
				r.FieldRegex = val
			default:
				return nil, fmt.Errorf("line %d: unknown token key %q", i+1, key)
			}
		}
		if err := r.Validate(); err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		rules = append(rules, r)
	}
	return rules, nil
}

type compiledRule struct {
	rule Rule
	re   *regexp.Regexp // nil unless FieldRegex is set
}

// Masker applies a set of validated rules to rendered strings.
type Masker struct {
	rules []compiledRule
}

// New builds a Masker from validated rules. New(nil) returns a no-op masker.
func New(rules []Rule) (*Masker, error) {
	m := &Masker{}
	for _, r := range rules {
		if err := r.Validate(); err != nil {
			return nil, err
		}
		cr := compiledRule{rule: r}
		if r.FieldRegex != "" {
			cr.re = regexp.MustCompile(r.FieldRegex) // already validated
		}
		m.rules = append(m.rules, cr)
	}
	return m, nil
}

// Apply returns the masked rendering of content for the given target.
func (m *Masker) Apply(content string, t Target) string {
	if m == nil || len(m.rules) == 0 {
		return content
	}
	var applicable []compiledRule
	for _, cr := range m.rules {
		if cr.rule.applies(t) {
			applicable = append(applicable, cr)
		}
	}
	if len(applicable) == 0 {
		return content
	}

	if json.Valid([]byte(content)) {
		var v interface{}
		dec := json.NewDecoder(strings.NewReader(content))
		dec.UseNumber()
		if err := dec.Decode(&v); err == nil {
			for _, cr := range applicable {
				v = applyRuleJSON(v, cr)
			}
			if out, err := json.Marshal(v); err == nil {
				return string(out)
			}
			return content
		}
	}

	// Non-JSON: apply the first applicable rule to the whole string.
	switch applicable[0].rule.Action {
	case ActionMask:
		return charMask(content)
	case ActionReplace:
		return maskedLiteral
	case ActionRemove:
		return "null"
	default:
		return content
	}
}

// applyRuleJSON applies one rule to a decoded JSON value, returning the result.
func applyRuleJSON(v interface{}, cr compiledRule) interface{} {
	if !cr.rule.hasSelectors() {
		if cr.rule.Action == ActionRemove {
			return removeAll(v)
		}
		return transformScalars(v, cr.rule.Action)
	}
	return applyWithSelectors(v, cr)
}

// applyWithSelectors walks v looking for keys the rule selects.
func applyWithSelectors(v interface{}, cr compiledRule) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		for k, val := range t {
			if selectorMatches(k, cr) {
				if cr.rule.Action == ActionRemove {
					delete(t, k)
				} else {
					t[k] = transformScalars(val, cr.rule.Action)
				}
			} else {
				t[k] = applyWithSelectors(val, cr)
			}
		}
		return t
	case []interface{}:
		for i := range t {
			t[i] = applyWithSelectors(t[i], cr)
		}
		return t
	default:
		return v
	}
}

func selectorMatches(key string, cr compiledRule) bool {
	if len(cr.rule.Fields) > 0 {
		for _, f := range cr.rule.Fields {
			if f == key {
				return true
			}
		}
		return false
	}
	if cr.re != nil {
		return cr.re.MatchString(key)
	}
	return false
}

// transformScalars applies MASK/REPLACE to every scalar in the subtree.
func transformScalars(v interface{}, action Action) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		for k, val := range t {
			t[k] = transformScalars(val, action)
		}
		return t
	case []interface{}:
		for i := range t {
			t[i] = transformScalars(t[i], action)
		}
		return t
	case string:
		return maskScalar(t, action)
	case json.Number:
		return maskScalar(t.String(), action)
	default: // bool, nil
		return v
	}
}

func maskScalar(s string, action Action) interface{} {
	if action == ActionMask {
		return charMask(s)
	}
	return maskedLiteral // ActionReplace
}

// removeAll deletes all fields (used for a selector-less REMOVE rule).
func removeAll(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		return map[string]interface{}{}
	case []interface{}:
		for i := range t {
			t[i] = removeAll(t[i])
		}
		return t
	default:
		return nil
	}
}

// charMask replaces characters class-wise, preserving whitespace separators.
func charMask(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == ' ' || r == '\n' || r == '\r' || r == '\t':
			b.WriteRune(r)
		case unicode.IsLetter(r) && unicode.IsUpper(r):
			b.WriteByte('X')
		case unicode.IsLetter(r) && unicode.IsLower(r):
			b.WriteByte('x')
		case unicode.IsDigit(r):
			b.WriteByte('n')
		default:
			b.WriteByte('-')
		}
	}
	return b.String()
}
