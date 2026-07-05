// Package serde is kafui's pluggable message serialization/deserialization
// framework. It formalizes the previously ad-hoc Avro/Protobuf/MessagePack
// decode paths into a Registry of named Serdes with auto-detection and a
// UI-visible fallback.
//
// # Extension point (MSG-20)
//
// The Registry is the extension point. There is no plugin system (Go's
// `plugin` package requires identical toolchain/version and is Linux/macOS
// only, so it is not worth the complexity here). To add a custom serde,
// implement the Serde interface and register it in code — see
// BuildRegistry in config.go, which is the single place built-in and
// configured serdes are wired up. A custom serde added there is
// indistinguishable from a built-in one to the rest of the app.
package serde

import "fmt"

// Serde decodes (and optionally encodes) the raw bytes of a Kafka message key
// or value into a human-readable string. Implementations must be safe for
// concurrent use.
type Serde interface {
	// Name is the unique identifier used for lookup and shown in the UI.
	Name() string
	// CanDeserialize reports whether this serde is a plausible decoder for the
	// given bytes. It is used for auto-detection and must not panic on
	// arbitrary input.
	CanDeserialize(data []byte) bool
	// Deserialize renders the bytes as text, or returns an error when the bytes
	// are not valid for this serde (which triggers fallback).
	Deserialize(data []byte) (string, error)
}

// Serializer is optionally implemented by serdes that support producing. It is
// separate from Serde so that read-only serdes (e.g. schema-registry decode,
// internal-topic decoders) need not implement it.
type Serializer interface {
	// Serialize encodes the given text back into wire bytes.
	Serialize(text string) ([]byte, error)
}

// DuplicateSerdeError is returned when registering a serde whose name is
// already taken.
type DuplicateSerdeError struct {
	Name string
}

func (e DuplicateSerdeError) Error() string {
	return fmt.Sprintf("serde %q is already registered", e.Name)
}

// UnknownSerdeError is returned when an explicitly chosen serde name is not
// registered.
type UnknownSerdeError struct {
	Name string
}

func (e UnknownSerdeError) Error() string {
	return fmt.Sprintf("unknown serde %q", e.Name)
}
