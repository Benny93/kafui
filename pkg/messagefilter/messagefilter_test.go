package messagefilter

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
)

func msg() api.Message {
	return api.Message{
		Key:       "order-123",
		Value:     "Hello World",
		Offset:    42,
		Partition: 3,
		Headers: []api.MessageHeader{
			{Key: "source", Value: "web"},
			{Key: "trace-id", Value: "abc"},
		},
	}
}

func TestEval(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		want    bool
		evalErr bool
	}{
		// operators on string fields
		{"key equals", `key == "order-123"`, true, false},
		{"key equals case sensitive miss", `key == "ORDER-123"`, false, false},
		{"key not equals", `key != "x"`, true, false},
		{"value contains ci", `value contains "hello"`, true, false},
		{"value contains ci upper", `value contains "WORLD"`, true, false},
		{"value contains miss", `value contains "bye"`, false, false},
		{"value lexicographic gt", `value > "A"`, true, false},
		{"value lexicographic lt false", `value < "A"`, false, false},

		// numeric fields
		{"partition eq", `partition == 3`, true, false},
		{"partition eq miss", `partition == 4`, false, false},
		{"partition gt", `partition > 1`, true, false},
		{"partition lt false", `partition < 1`, false, false},
		{"offset ge", `offset >= 42`, true, false},
		{"offset le", `offset <= 42`, true, false},
		{"offset ne", `offset != 0`, true, false},
		{"offset gt false", `offset > 100`, false, false},
		{"partition contains decimal form", `partition contains "3"`, true, false},

		// non-numeric operand for numeric op => eval error
		{"partition eq non-numeric", `partition == "abc"`, false, true},
		{"offset gt non-numeric", `offset > "xyz"`, false, true},

		// headers
		{"header match", `header.source == "web"`, true, false},
		{"headers alias match", `headers.source == "web"`, true, false},
		{"header contains", `header.trace-id contains "AB"`, true, false},
		{"header missing empty", `header.missing == ""`, true, false},
		{"header missing miss", `header.missing == "x"`, false, false},

		// AND / OR precedence: AND binds tighter than OR.
		// false AND true OR true  =>  (false AND true) OR true => true
		{"and or precedence", `partition == 99 AND partition == 3 OR offset == 42`, true, false},
		// true AND false OR false => false
		{"and or precedence false", `partition == 3 AND partition == 99 OR offset == 0`, false, false},
		{"and both true", `key == "order-123" AND offset == 42`, true, false},
		{"and one false", `key == "order-123" AND offset == 0`, false, false},
		{"or symbolic", `offset == 0 || partition == 3`, true, false},
		{"and symbolic", `offset == 42 && partition == 3`, true, false},
		{"lowercase and/or", `offset == 42 and partition == 3 or key == "x"`, true, false},

		// parentheses grouping: true AND (false OR true) => true
		{"parens group", `key == "order-123" AND (partition == 99 OR offset == 42)`, true, false},
		// (true OR false) AND false => false
		{"parens group false", `(partition == 3 OR offset == 0) AND offset == 0`, false, false},
		{"nested parens", `((offset == 42))`, true, false},

		// a clear miss
		{"clear miss", `key == "nope"`, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f, err := Compile(tc.expr)
			assert.NoError(t, err, "compile should succeed")
			got, evalErr := f.Eval(msg())
			if tc.evalErr {
				assert.Error(t, evalErr)
				return
			}
			assert.NoError(t, evalErr)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCompileErrors(t *testing.T) {
	cases := []struct {
		name string
		expr string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"unknown field", `bogus == "x"`},
		{"unknown operator", `key ~ "x"`},
		{"unbalanced parens", `(key == "x"`},
		{"missing operand", `key ==`},
		{"missing operator", `key`},
		{"bare boolean not supported", `key`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile(tc.expr)
			assert.Error(t, err)
		})
	}
}

func TestExpr(t *testing.T) {
	f, err := Compile(`key == "x"`)
	assert.NoError(t, err)
	assert.Equal(t, `key == "x"`, f.Expr())
}

func TestID(t *testing.T) {
	a, _ := Compile(`key == "x"`)
	b, _ := Compile(`key == "x"`)
	c, _ := Compile(`key == "y"`)

	assert.Len(t, a.ID(), 8)
	assert.Equal(t, a.ID(), b.ID(), "same expression => same id")
	assert.NotEqual(t, a.ID(), c.ID(), "different expression => different id")
}

func TestTest(t *testing.T) {
	samples := []api.Message{
		{Partition: 1},
		{Partition: 3},
		{Partition: 3},
	}
	results, compileErr, evalErr := Test(`partition == 3`, samples)
	assert.NoError(t, compileErr)
	assert.NoError(t, evalErr)
	assert.Equal(t, []bool{false, true, true}, results)

	// compile error path
	results, compileErr, evalErr = Test(``, samples)
	assert.Error(t, compileErr)
	assert.Nil(t, results)
	assert.NoError(t, evalErr)

	// eval error path
	_, compileErr, evalErr = Test(`partition == "abc"`, samples)
	assert.NoError(t, compileErr)
	assert.Error(t, evalErr)
}
