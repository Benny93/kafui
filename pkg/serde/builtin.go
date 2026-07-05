package serde

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Built-in serde names.
const (
	NameString = "string"
	NameHex    = "hex"    // also serves as the bytes representation
	NameJSON   = "json"   // pretty-printed JSON
	NameNull   = "null"   // empty / absent payloads
	NameInt    = "int"    // 4-byte big-endian signed
	NameLong   = "long"   // 8-byte big-endian signed
	NameFloat  = "float"  // 4-byte big-endian IEEE-754
	NameDouble = "double" // 8-byte big-endian IEEE-754
)

// StringSerde renders bytes as UTF-8 text. It is the universal fallback.
type StringSerde struct{}

func (StringSerde) Name() string                 { return NameString }
func (StringSerde) CanDeserialize(_ []byte) bool  { return true }
func (StringSerde) Deserialize(d []byte) (string, error) {
	if !utf8.Valid(d) {
		return "", fmt.Errorf("not valid UTF-8")
	}
	return string(d), nil
}
func (StringSerde) Serialize(text string) ([]byte, error) { return []byte(text), nil }

// HexSerde renders bytes as a hex string. It never fails and accepts any input,
// so it is the binary fallback.
type HexSerde struct{}

func (HexSerde) Name() string                  { return NameHex }
func (HexSerde) CanDeserialize(_ []byte) bool   { return true }
func (HexSerde) Deserialize(d []byte) (string, error) {
	return hex.EncodeToString(d), nil
}
func (HexSerde) Serialize(text string) ([]byte, error) {
	return hex.DecodeString(strings.TrimSpace(text))
}

// JSONSerde pretty-prints JSON payloads. CanDeserialize only claims valid JSON
// so auto-detection prefers it over plain string.
type JSONSerde struct{}

func (JSONSerde) Name() string { return NameJSON }
func (JSONSerde) CanDeserialize(d []byte) bool {
	return json.Valid(bytes.TrimSpace(d))
}
func (JSONSerde) Deserialize(d []byte) (string, error) {
	var v any
	if err := json.Unmarshal(d, &v); err != nil {
		return "", err
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}
func (JSONSerde) Serialize(text string) ([]byte, error) {
	if !json.Valid([]byte(text)) {
		return nil, fmt.Errorf("not valid JSON")
	}
	return []byte(text), nil
}

// NullSerde renders empty/nil payloads.
type NullSerde struct{}

func (NullSerde) Name() string                { return NameNull }
func (NullSerde) CanDeserialize(d []byte) bool { return len(d) == 0 }
func (NullSerde) Deserialize(d []byte) (string, error) {
	if len(d) != 0 {
		return "", fmt.Errorf("not empty")
	}
	return "null", nil
}

// numeric serde: fixed-width big-endian integer/float.
type numericSerde struct {
	name  string
	size  int
	float bool
}

func (s numericSerde) Name() string { return s.name }
func (s numericSerde) CanDeserialize(d []byte) bool { return len(d) == s.size }
func (s numericSerde) Deserialize(d []byte) (string, error) {
	if len(d) != s.size {
		return "", fmt.Errorf("%s requires %d bytes, got %d", s.name, s.size, len(d))
	}
	switch {
	case s.float && s.size == 4:
		return strconv.FormatFloat(float64(math.Float32frombits(binary.BigEndian.Uint32(d))), 'g', -1, 32), nil
	case s.float && s.size == 8:
		return strconv.FormatFloat(math.Float64frombits(binary.BigEndian.Uint64(d)), 'g', -1, 64), nil
	case s.size == 4:
		return strconv.FormatInt(int64(int32(binary.BigEndian.Uint32(d))), 10), nil
	default: // size 8
		return strconv.FormatInt(int64(binary.BigEndian.Uint64(d)), 10), nil
	}
}
func (s numericSerde) Serialize(text string) ([]byte, error) {
	buf := make([]byte, s.size)
	switch {
	case s.float && s.size == 4:
		f, err := strconv.ParseFloat(strings.TrimSpace(text), 32)
		if err != nil {
			return nil, err
		}
		binary.BigEndian.PutUint32(buf, math.Float32bits(float32(f)))
	case s.float && s.size == 8:
		f, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil {
			return nil, err
		}
		binary.BigEndian.PutUint64(buf, math.Float64bits(f))
	case s.size == 4:
		n, err := strconv.ParseInt(strings.TrimSpace(text), 10, 32)
		if err != nil {
			return nil, err
		}
		binary.BigEndian.PutUint32(buf, uint32(int32(n)))
	default:
		n, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		if err != nil {
			return nil, err
		}
		binary.BigEndian.PutUint64(buf, uint64(n))
	}
	return buf, nil
}

// IntSerde, LongSerde, FloatSerde, DoubleSerde are the big-endian numeric serdes.
func IntSerde() Serde    { return numericSerde{name: NameInt, size: 4} }
func LongSerde() Serde   { return numericSerde{name: NameLong, size: 8} }
func FloatSerde() Serde  { return numericSerde{name: NameFloat, size: 4, float: true} }
func DoubleSerde() Serde { return numericSerde{name: NameDouble, size: 8, float: true} }
