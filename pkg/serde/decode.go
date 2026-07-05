package serde

import (
	"encoding/hex"
	"fmt"
	"unicode/utf8"
)

// Auto is the sentinel serde name meaning "auto-detect".
const Auto = "auto"

// fallbackSuffix marks a serde name when the chosen/auto serde failed and a
// fallback (string/hex) was used, so the UI can flag the row (MSG-16).
const fallbackSuffix = " (fallback)"

// Decode renders data using the chosen serde (empty or "auto" = auto-detect).
// When the selected serde fails (or none is found) it falls back to string for
// valid UTF-8, otherwise hex, and marks the returned name with a " (fallback)"
// suffix. Decoding always yields some text, so no error is returned.
func Decode(reg *Registry, chosen string, data []byte) (text, name string, fallback bool) {
	if len(data) == 0 {
		return "null", NameNull, false
	}
	var s Serde
	if chosen == "" || chosen == Auto {
		s = reg.AutoDetect(data)
	} else if got, ok := reg.Get(chosen); ok {
		s = got
	}
	if s != nil {
		if out, err := s.Deserialize(data); err == nil {
			return out, s.Name(), false
		}
	}
	if utf8.Valid(data) {
		return string(data), NameString + fallbackSuffix, true
	}
	return hex.EncodeToString(data), NameHex + fallbackSuffix, true
}

// IsFallback reports whether a serde name (as returned by Decode) denotes a
// fallback rendering.
func IsFallback(name string) bool {
	return len(name) > len(fallbackSuffix) && name[len(name)-len(fallbackSuffix):] == fallbackSuffix
}

// Validate checks that an explicitly chosen serde exists and can decode the
// given sample bytes. Auto/empty always validates. A missing serde yields an
// UnknownSerdeError; an incapable one a descriptive error (MSG-15).
func Validate(reg *Registry, chosen string, sample []byte) error {
	if chosen == "" || chosen == Auto {
		return nil
	}
	s, ok := reg.Get(chosen)
	if !ok {
		return UnknownSerdeError{Name: chosen}
	}
	if _, err := s.Deserialize(sample); err != nil {
		return fmt.Errorf("serde %q cannot decode this payload: %w", chosen, err)
	}
	return nil
}
