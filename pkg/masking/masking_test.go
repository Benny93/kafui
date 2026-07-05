package masking

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustMasker(t *testing.T, rules []Rule) *Masker {
	t.Helper()
	m, err := New(rules)
	require.NoError(t, err)
	return m
}

func TestApply(t *testing.T) {
	tests := []struct {
		name    string
		rules   []Rule
		content string
		target  Target
		want    string
	}{
		{
			name:    "MASK char-class on whole non-JSON string",
			rules:   []Rule{{Action: ActionMask, Values: true}},
			content: "Ab 1?",
			target:  Value,
			want:    "Xx n-",
		},
		{
			name:    "REPLACE specific nested field",
			rules:   []Rule{{Action: ActionReplace, Values: true, Fields: []string{"ssn"}}},
			content: `{"user":{"ssn":"123-45-6789","name":"Bob"}}`,
			target:  Value,
			want:    `{"user":{"name":"Bob","ssn":"***DATA_MASKED***"}}`,
		},
		{
			name:    "REPLACE whole object masks all inner scalars",
			rules:   []Rule{{Action: ActionReplace, Values: true, Fields: []string{"user"}}},
			content: `{"user":{"name":"Bob","age":30}}`,
			target:  Value,
			want:    `{"user":{"age":"***DATA_MASKED***","name":"***DATA_MASKED***"}}`,
		},
		{
			name:    "REMOVE deletes a JSON field",
			rules:   []Rule{{Action: ActionRemove, Values: true, Fields: []string{"secret"}}},
			content: `{"secret":"x","keep":"y"}`,
			target:  Value,
			want:    `{"keep":"y"}`,
		},
		{
			name:    "REMOVE on non-JSON yields null",
			rules:   []Rule{{Action: ActionRemove, Values: true}},
			content: "some plain text",
			target:  Value,
			want:    "null",
		},
		{
			name:    "MASK specific numeric field only",
			rules:   []Rule{{Action: ActionMask, Values: true, Fields: []string{"age"}}},
			content: `{"age":30,"name":"Bob"}`,
			target:  Value,
			want:    `{"age":"nn","name":"Bob"}`,
		},
		{
			name:    "MASK specific string field only",
			rules:   []Rule{{Action: ActionMask, Values: true, Fields: []string{"name"}}},
			content: `{"age":30,"name":"Bob"}`,
			target:  Value,
			want:    `{"age":30,"name":"Xxx"}`,
		},
		{
			name:    "key-scoped rule does not affect Value target",
			rules:   []Rule{{Action: ActionReplace, Keys: true, Fields: []string{"ssn"}}},
			content: `{"ssn":"123"}`,
			target:  Value,
			want:    `{"ssn":"123"}`,
		},
		{
			name:    "value-scoped rule does not affect Key target",
			rules:   []Rule{{Action: ActionReplace, Values: true, Fields: []string{"ssn"}}},
			content: `{"ssn":"123"}`,
			target:  Key,
			want:    `{"ssn":"123"}`,
		},
		{
			name:    "empty rules leave content unchanged",
			rules:   nil,
			content: `{"a":"b"}`,
			target:  Value,
			want:    `{"a":"b"}`,
		},
		{
			name:    "arrays are traversed",
			rules:   []Rule{{Action: ActionReplace, Values: true, Fields: []string{"items"}}},
			content: `{"items":["a","b","c"]}`,
			target:  Value,
			want:    `{"items":["***DATA_MASKED***","***DATA_MASKED***","***DATA_MASKED***"]}`,
		},
		{
			name:    "regex selector matches field names",
			rules:   []Rule{{Action: ActionRemove, Values: true, FieldRegex: "^secret"}},
			content: `{"secretKey":"x","secretId":"y","keep":"z"}`,
			target:  Value,
			want:    `{"keep":"z"}`,
		},
		{
			name:    "no selector masks all scalars including nested and arrays",
			rules:   []Rule{{Action: ActionReplace, Values: true}},
			content: `{"a":"x","b":{"c":1},"d":["e"]}`,
			target:  Value,
			want:    `{"a":"***DATA_MASKED***","b":{"c":"***DATA_MASKED***"},"d":["***DATA_MASKED***"]}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := mustMasker(t, tc.rules)
			assert.Equal(t, tc.want, m.Apply(tc.content, tc.target))
		})
	}
}

func TestNilMasker(t *testing.T) {
	var m *Masker
	assert.Equal(t, `{"a":"b"}`, m.Apply(`{"a":"b"}`, Value))

	noop, err := New(nil)
	require.NoError(t, err)
	assert.Equal(t, "anything", noop.Apply("anything", Value))
}

func TestParseRules(t *testing.T) {
	lines := []string{
		"MASK scope=value field=ssn,creditCard",
		"REPLACE scope=both",
		"REMOVE scope=key regex=^secret",
		"", // blank lines ignored
	}
	rules, err := ParseRules(lines)
	require.NoError(t, err)
	require.Len(t, rules, 3)

	assert.Equal(t, Rule{Action: ActionMask, Values: true, Fields: []string{"ssn", "creditCard"}}, rules[0])
	assert.Equal(t, Rule{Action: ActionReplace, Keys: true, Values: true}, rules[1])
	assert.Equal(t, Rule{Action: ActionRemove, Keys: true, FieldRegex: "^secret"}, rules[2])
}

func TestParseRulesErrors(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"unknown action", "BOGUS scope=value"},
		{"missing scope", "MASK field=a"},
		{"unknown scope", "MASK scope=nope"},
		{"malformed token", "MASK scope=value foo"},
		{"unknown token key", "MASK scope=value color=red"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseRules([]string{tc.line})
			assert.Error(t, err)
		})
	}
}

func TestRuleValidate(t *testing.T) {
	tests := []struct {
		name    string
		rule    Rule
		wantErr bool
	}{
		{"valid", Rule{Action: ActionMask, Values: true}, false},
		{"unknown action", Rule{Action: "NOPE", Values: true}, true},
		{"neither scope", Rule{Action: ActionMask}, true},
		{"fields and regex", Rule{Action: ActionMask, Values: true, Fields: []string{"a"}, FieldRegex: "b"}, true},
		{"bad regex", Rule{Action: ActionMask, Values: true, FieldRegex: "["}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.rule.Validate()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewValidatesRules(t *testing.T) {
	_, err := New([]Rule{{Action: "NOPE", Values: true}})
	assert.Error(t, err)
}
