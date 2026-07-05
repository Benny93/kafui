package serde

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
)

// Internal-topic serde names. These decode the well-known binary formats Kafka
// uses for its internal topics. They are read-only (no Serialize).
const (
	NameConsumerOffsetsKey   = "consumer-offsets-key"
	NameConsumerOffsetsValue = "consumer-offsets-value"
)

// ponytail: __transaction_state and MirrorMaker2 internal topics
// (heartbeats/checkpoints/offset-syncs) use their own binary schemas. They are
// deferred — only __consumer_offsets (the common case) is decoded here.

// binReader is a minimal big-endian reader for the Kafka on-wire format.
type binReader struct {
	b   []byte
	pos int
	err error
}

func (r *binReader) int16() int16 {
	if r.err != nil || r.pos+2 > len(r.b) {
		r.err = fmt.Errorf("short read (int16)")
		return 0
	}
	v := int16(binary.BigEndian.Uint16(r.b[r.pos:]))
	r.pos += 2
	return v
}

func (r *binReader) int32() int32 {
	if r.err != nil || r.pos+4 > len(r.b) {
		r.err = fmt.Errorf("short read (int32)")
		return 0
	}
	v := int32(binary.BigEndian.Uint32(r.b[r.pos:]))
	r.pos += 4
	return v
}

func (r *binReader) int64() int64 {
	if r.err != nil || r.pos+8 > len(r.b) {
		r.err = fmt.Errorf("short read (int64)")
		return 0
	}
	v := int64(binary.BigEndian.Uint64(r.b[r.pos:]))
	r.pos += 8
	return v
}

// string reads an int16-length-prefixed string (-1 length = null).
func (r *binReader) string() string {
	n := int(r.int16())
	if r.err != nil {
		return ""
	}
	if n < 0 {
		return ""
	}
	if r.pos+n > len(r.b) {
		r.err = fmt.Errorf("short read (string len %d)", n)
		return ""
	}
	s := string(r.b[r.pos : r.pos+n])
	r.pos += n
	return s
}

func toJSON(v any) (string, error) {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// ConsumerOffsetsKeySerde decodes __consumer_offsets record keys.
type ConsumerOffsetsKeySerde struct{}

func (ConsumerOffsetsKeySerde) Name() string { return NameConsumerOffsetsKey }

func (s ConsumerOffsetsKeySerde) CanDeserialize(d []byte) bool {
	_, err := s.decode(d)
	return err == nil
}

func (ConsumerOffsetsKeySerde) decode(d []byte) (map[string]any, error) {
	r := &binReader{b: d}
	version := r.int16()
	switch version {
	case 0, 1: // OffsetCommitKey
		group := r.string()
		topic := r.string()
		partition := r.int32()
		if r.err != nil {
			return nil, r.err
		}
		return map[string]any{
			"type": "offset-commit", "version": version,
			"group": group, "topic": topic, "partition": partition,
		}, nil
	case 2: // GroupMetadataKey
		group := r.string()
		if r.err != nil {
			return nil, r.err
		}
		return map[string]any{"type": "group-metadata", "version": version, "group": group}, nil
	default:
		return nil, fmt.Errorf("unknown __consumer_offsets key version %d", version)
	}
}

func (s ConsumerOffsetsKeySerde) Deserialize(d []byte) (string, error) {
	m, err := s.decode(d)
	if err != nil {
		return "", err
	}
	return toJSON(m)
}

// ConsumerOffsetsValueSerde decodes __consumer_offsets offset-commit values.
type ConsumerOffsetsValueSerde struct{}

func (ConsumerOffsetsValueSerde) Name() string { return NameConsumerOffsetsValue }

func (s ConsumerOffsetsValueSerde) CanDeserialize(d []byte) bool {
	_, err := s.decode(d)
	return err == nil
}

func (ConsumerOffsetsValueSerde) decode(d []byte) (map[string]any, error) {
	r := &binReader{b: d}
	version := r.int16()
	if version < 0 || version > 3 {
		// Group-metadata values (written under a version-2 key) use a different,
		// involved schema (member assignments) — ponytail: deferred.
		return nil, fmt.Errorf("unsupported __consumer_offsets value version %d", version)
	}
	out := map[string]any{"version": version}
	out["offset"] = r.int64()
	if version >= 3 {
		out["leaderEpoch"] = r.int32()
	}
	out["metadata"] = r.string()
	out["commitTimestamp"] = r.int64()
	if version == 1 {
		out["expireTimestamp"] = r.int64()
	}
	if r.err != nil {
		return nil, r.err
	}
	return out, nil
}

func (s ConsumerOffsetsValueSerde) Deserialize(d []byte) (string, error) {
	m, err := s.decode(d)
	if err != nil {
		return "", err
	}
	return toJSON(m)
}
