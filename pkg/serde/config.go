package serde

import (
	"fmt"
	"regexp"
)

// SerdeConfig is a per-cluster serde binding. For topics whose name matches
// TopicPattern (a regex; empty = all topics), the named serde is applied to the
// key and/or value. When DescriptorPath is set the binding also *defines* a
// descriptor-file Protobuf serde (registered under Name) rather than merely
// referencing a built-in.
type SerdeConfig struct {
	Name           string `yaml:"name"`           // registered serde name to apply / define
	TopicPattern   string `yaml:"topicPattern"`   // regex; empty = all topics
	Target         string `yaml:"target"`         // "key" | "value" | "both" (default both)
	DescriptorPath string `yaml:"descriptorPath"` // FileDescriptorSet path (descriptor protobuf)
	MessageType    string `yaml:"messageType"`    // fully-qualified message name
}

func (c SerdeConfig) matchesTarget(isKey bool) bool {
	switch c.Target {
	case "key":
		return isKey
	case "value":
		return !isKey
	default: // "", "both"
		return true
	}
}

// SelectSerde returns the name of the first configured serde bound to the given
// topic/target, or "" when none matches (the caller then auto-detects).
func SelectSerde(configs []SerdeConfig, topic string, isKey bool) string {
	for _, c := range configs {
		if c.Name == "" || !c.matchesTarget(isKey) {
			continue
		}
		if c.TopicPattern != "" {
			re, err := regexp.Compile(c.TopicPattern)
			if err != nil || !re.MatchString(topic) {
				continue
			}
		}
		return c.Name
	}
	return ""
}

// BuildRegistry assembles the standard registry: the schema-registry serde
// (using the given decoder), then configured descriptor-protobuf serdes, then
// the primitive/format built-ins. Auto-detection order (MSG-15) is
// schema-registry → configured → JSON → string. Numeric/hex/msgpack/raw-proto
// and internal-topic serdes are selectable by name only (they would falsely
// match arbitrary bytes during auto-detection). Duplicate configured names fail
// (MSG-11/17).
func BuildRegistry(decode DecodeFunc, configs []SerdeConfig) (*Registry, error) {
	r := NewRegistry()

	// Confluent-framed schema-registry payloads are unambiguous — detect first.
	if err := r.RegisterAuto(NewSchemaRegistrySerde(decode)); err != nil {
		return nil, err
	}

	// Configured descriptor-file protobuf serdes, prioritised in auto-detect.
	for _, c := range configs {
		if c.DescriptorPath == "" {
			continue
		}
		s, err := NewDescriptorProtobufSerde(c.Name, c.DescriptorPath, c.MessageType)
		if err != nil {
			return nil, fmt.Errorf("serde %q: %w", c.Name, err)
		}
		if err := r.RegisterAuto(s); err != nil {
			return nil, err
		}
	}

	// Name-only serdes.
	for _, s := range []Serde{
		HexSerde{}, IntSerde(), LongSerde(), FloatSerde(), DoubleSerde(),
		MsgpackSerde{}, RawProtobufSerde{},
		ConsumerOffsetsKeySerde{}, ConsumerOffsetsValueSerde{},
	} {
		if err := r.Register(s); err != nil {
			return nil, err
		}
	}

	// Auto-detected primitives, most-specific first.
	for _, s := range []Serde{NullSerde{}, JSONSerde{}, StringSerde{}} {
		if err := r.RegisterAuto(s); err != nil {
			return nil, err
		}
	}
	return r, nil
}
