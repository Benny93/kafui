package serde

import (
	"encoding/binary"
	"fmt"
)

// NameSchemaRegistry is the Confluent-wire-format schema-registry serde name.
// It covers Avro, JSON Schema and Protobuf payloads: all three use the same
// framing (magic byte 0x00 + 4-byte big-endian schema id), and the concrete
// decode is delegated to the injected DecodeFunc (which reuses kafds' existing
// schema-registry client / Avro cache).
const NameSchemaRegistry = "schema-registry"

// DecodeFunc decodes a full Confluent-framed payload (including the magic byte
// and schema id) into human-readable bytes (typically JSON). kafds supplies the
// Avro-cache-backed implementation; a nil decoder makes the serde report an
// explanatory error so decoding falls back.
type DecodeFunc func(data []byte) ([]byte, error)

// SchemaRegistrySerde decodes payloads framed with the Confluent wire format.
type SchemaRegistrySerde struct {
	decode DecodeFunc
}

// NewSchemaRegistrySerde builds the serde with the given decoder. A nil decoder
// is allowed (CanDeserialize still recognises the framing, but Deserialize
// reports that no registry is configured, triggering fallback).
func NewSchemaRegistrySerde(decode DecodeFunc) *SchemaRegistrySerde {
	return &SchemaRegistrySerde{decode: decode}
}

func (s *SchemaRegistrySerde) Name() string { return NameSchemaRegistry }

// CanDeserialize reports whether data carries the Confluent magic-byte framing.
func (s *SchemaRegistrySerde) CanDeserialize(data []byte) bool {
	return len(data) >= 5 && data[0] == 0x00
}

// SchemaID extracts the schema id from a Confluent-framed payload.
func SchemaID(data []byte) (uint32, bool) {
	if len(data) < 5 || data[0] != 0x00 {
		return 0, false
	}
	return binary.BigEndian.Uint32(data[1:5]), true
}

func (s *SchemaRegistrySerde) Deserialize(data []byte) (string, error) {
	if !s.CanDeserialize(data) {
		return "", fmt.Errorf("payload is not Confluent-framed (missing magic byte)")
	}
	if s.decode == nil {
		return "", fmt.Errorf("no schema registry configured")
	}
	out, err := s.decode(data)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
