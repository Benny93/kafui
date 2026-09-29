package kafds

import (
	"fmt"
	"strings"
	"testing"

	"github.com/linkedin/goavro/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goavro renders a record by ranging over a Go map, so its field order changes
// from message to message. After sortJSONKeys every rendering is identical.
func TestSortJSONKeysMakesAvroOrderStable(t *testing.T) {
	var fields []string
	native := map[string]interface{}{}
	for i := 0; i < 12; i++ {
		name := fmt.Sprintf("f%02d", i)
		fields = append(fields, fmt.Sprintf(`{"name":%q,"type":"long"}`, name))
		native[name] = int64(9007199254740993) + int64(i) // beyond float64 precision
	}
	native["nested"] = map[string]interface{}{"b": "x", "a": "<&>"}
	schema := `{"type":"record","name":"R","fields":[` + strings.Join(fields, ",") +
		`,{"name":"nested","type":{"type":"record","name":"N","fields":[{"name":"b","type":"string"},{"name":"a","type":"string"}]}}]}`
	codec, err := goavro.NewCodec(schema)
	require.NoError(t, err)

	seen := map[string]bool{}
	var sorted string
	for i := 0; i < 20; i++ {
		text, err := codec.TextualFromNative(nil, native)
		require.NoError(t, err)
		seen[string(text)] = true
		out := string(sortJSONKeys(text))
		if sorted == "" {
			sorted = out
		}
		assert.Equal(t, sorted, out, "every rendering sorts the same")
	}
	require.Greater(t, len(seen), 1, "goavro's order should vary, or this test proves nothing")

	assert.True(t, strings.HasPrefix(sorted, `{"f00":9007199254740993,"f01":`), sorted)
	assert.Contains(t, sorted, `"nested":{"a":"<&>","b":"x"}`, "nested records sort too; no HTML escaping")
}

func TestSortJSONKeysLeavesNonJSONAlone(t *testing.T) {
	for _, in := range []string{"plain text", "", `{"broken":`, "42"} {
		assert.Equal(t, in, string(sortJSONKeys([]byte(in))))
	}
}
