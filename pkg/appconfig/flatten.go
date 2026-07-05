package appconfig

import "fmt"

// Flatten turns a nested map into dot-separated keys, e.g.
// {"a": {"b": "c"}} becomes {"a.b": "c"}. Non-map leaves are stringified.
func Flatten(m map[string]any) map[string]string {
	out := map[string]string{}
	flattenInto("", m, out)
	return out
}

func flattenInto(prefix string, m map[string]any, out map[string]string) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch child := v.(type) {
		case map[string]any:
			flattenInto(key, child, out)
		case map[any]any: // yaml.v2-style maps, just in case
			conv := make(map[string]any, len(child))
			for ck, cv := range child {
				conv[fmt.Sprintf("%v", ck)] = cv
			}
			flattenInto(key, conv, out)
		default:
			out[key] = fmt.Sprintf("%v", v)
		}
	}
}
