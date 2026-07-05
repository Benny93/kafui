package serde

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// NameRawProtobuf is the schemaless wire-format protobuf serde name.
const NameRawProtobuf = "protobuf-raw"

// RawProtobufSerde performs a best-effort, schemaless dump of a protobuf
// wire-format payload: it walks the tag/field structure and renders it as JSON
// keyed by field number. No message type is required.
type RawProtobufSerde struct{}

func (RawProtobufSerde) Name() string { return NameRawProtobuf }

func (RawProtobufSerde) CanDeserialize(d []byte) bool {
	if len(d) == 0 {
		return false
	}
	_, err := walkProtobuf(d)
	return err == nil
}

func (RawProtobufSerde) Deserialize(d []byte) (string, error) {
	fields, err := walkProtobuf(d)
	if err != nil {
		return "", err
	}
	out, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// walkProtobuf parses the wire format into a field-number-keyed map. Nested
// length-delimited fields that are themselves valid messages are decoded
// recursively; otherwise they are rendered as a UTF-8 string or hex.
func walkProtobuf(d []byte) (map[string]any, error) {
	out := make(map[string]any)
	for len(d) > 0 {
		num, typ, n := protowire.ConsumeTag(d)
		if n < 0 {
			return nil, fmt.Errorf("invalid protobuf tag")
		}
		d = d[n:]
		key := "field_" + strconv.Itoa(int(num))
		var val any
		switch typ {
		case protowire.VarintType:
			v, m := protowire.ConsumeVarint(d)
			if m < 0 {
				return nil, fmt.Errorf("invalid varint")
			}
			d, val = d[m:], v
		case protowire.Fixed32Type:
			v, m := protowire.ConsumeFixed32(d)
			if m < 0 {
				return nil, fmt.Errorf("invalid fixed32")
			}
			d, val = d[m:], v
		case protowire.Fixed64Type:
			v, m := protowire.ConsumeFixed64(d)
			if m < 0 {
				return nil, fmt.Errorf("invalid fixed64")
			}
			d, val = d[m:], v
		case protowire.BytesType:
			v, m := protowire.ConsumeBytes(d)
			if m < 0 {
				return nil, fmt.Errorf("invalid length-delimited field")
			}
			d = d[m:]
			if nested, err := walkProtobuf(v); err == nil && len(nested) > 0 {
				val = nested
			} else if s := string(v); isPrintable(s) {
				val = s
			} else {
				val = fmt.Sprintf("0x%x", v)
			}
		default:
			return nil, fmt.Errorf("unsupported wire type %d", typ)
		}
		appendField(out, key, val)
	}
	return out, nil
}

func appendField(m map[string]any, key string, val any) {
	if existing, ok := m[key]; ok {
		if arr, ok := existing.([]any); ok {
			m[key] = append(arr, val)
		} else {
			m[key] = []any{existing, val}
		}
		return
	}
	m[key] = val
}

func isPrintable(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r == '�' || (r < 0x20 && r != '\n' && r != '\t' && r != '\r') {
			return false
		}
	}
	return true
}

// DescriptorProtobufSerde decodes protobuf payloads against a message type
// loaded from a compiled FileDescriptorSet (`protoc --descriptor_set_out`).
type DescriptorProtobufSerde struct {
	name    string
	msgType protoreflect.MessageType
}

// NewDescriptorProtobufSerde loads descPath (a serialized FileDescriptorSet)
// and resolves messageName (fully-qualified, e.g. "pkg.MyMessage").
func NewDescriptorProtobufSerde(name, descPath, messageName string) (*DescriptorProtobufSerde, error) {
	raw, err := os.ReadFile(descPath)
	if err != nil {
		return nil, fmt.Errorf("read descriptor set %q: %w", descPath, err)
	}
	var fds descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(raw, &fds); err != nil {
		return nil, fmt.Errorf("parse descriptor set %q: %w", descPath, err)
	}
	files, err := protodesc.NewFiles(&fds)
	if err != nil {
		return nil, fmt.Errorf("build descriptor files: %w", err)
	}
	desc, err := files.FindDescriptorByName(protoreflect.FullName(messageName))
	if err != nil {
		return nil, fmt.Errorf("message %q not found in descriptor set: %w", messageName, err)
	}
	md, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("%q is not a message type", messageName)
	}
	return &DescriptorProtobufSerde{name: name, msgType: dynamicpb.NewMessageType(md)}, nil
}

func (s *DescriptorProtobufSerde) Name() string { return s.name }

func (s *DescriptorProtobufSerde) CanDeserialize(d []byte) bool {
	_, err := s.decode(d)
	return err == nil
}

func (s *DescriptorProtobufSerde) Deserialize(d []byte) (string, error) {
	msg, err := s.decode(d)
	if err != nil {
		return "", err
	}
	out, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(msg.Interface())
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (s *DescriptorProtobufSerde) decode(d []byte) (protoreflect.Message, error) {
	msg := s.msgType.New()
	if err := proto.Unmarshal(d, msg.Interface()); err != nil {
		return nil, err
	}
	return msg, nil
}
