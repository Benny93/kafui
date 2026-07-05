package serde

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestPrimitiveSerdes(t *testing.T) {
	var neg int32 = -5
	i32 := make([]byte, 4)
	binary.BigEndian.PutUint32(i32, uint32(neg))
	i64 := make([]byte, 8)
	binary.BigEndian.PutUint64(i64, uint64(1234567890123))

	tests := []struct {
		name  string
		serde Serde
		in    []byte
		want  string
	}{
		{"string", StringSerde{}, []byte("hello"), "hello"},
		{"hex", HexSerde{}, []byte{0xDE, 0xAD}, "dead"},
		{"json", JSONSerde{}, []byte(`{"a":1}`), "{\n  \"a\": 1\n}"},
		{"null", NullSerde{}, nil, "null"},
		{"int", IntSerde(), i32, "-5"},
		{"long", LongSerde(), i64, "1234567890123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.serde.Deserialize(tt.in)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPrimitiveRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		s    Serde
		text string
	}{
		{"string", StringSerde{}, "hello world"},
		{"hex", HexSerde{}, "deadbeef"},
		{"int", IntSerde(), "-42"},
		{"long", LongSerde(), "9000000000"},
		{"float", FloatSerde(), "1.5"},
		{"double", DoubleSerde(), "3.14159"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ser, ok := tt.s.(Serializer)
			require.True(t, ok)
			b, err := ser.Serialize(tt.text)
			require.NoError(t, err)
			got, err := tt.s.Deserialize(b)
			require.NoError(t, err)
			assert.Equal(t, tt.text, got)
		})
	}
}

func TestNumericInvalidLength(t *testing.T) {
	_, err := IntSerde().Deserialize([]byte{0x01, 0x02})
	assert.Error(t, err)
}

func TestRegistryDuplicateRejected(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(StringSerde{}))
	err := r.Register(StringSerde{})
	assert.ErrorAs(t, err, &DuplicateSerdeError{})
}

func TestAutoDetectOrdering(t *testing.T) {
	r, err := BuildRegistry(nil, nil)
	require.NoError(t, err)

	// Confluent-framed bytes win first (schema-registry serde).
	framed := append([]byte{0x00, 0x00, 0x00, 0x00, 0x01}, []byte("x")...)
	assert.Equal(t, NameSchemaRegistry, r.AutoDetect(framed).Name())

	// Valid JSON is detected as json, not plain string.
	assert.Equal(t, NameJSON, r.AutoDetect([]byte(`{"k":1}`)).Name())

	// Empty is null.
	assert.Equal(t, NameNull, r.AutoDetect(nil).Name())

	// Arbitrary text falls through to string.
	assert.Equal(t, NameString, r.AutoDetect([]byte("not json")).Name())
}

func TestSchemaRegistryMagicByte(t *testing.T) {
	// Fake registry decoder: returns fixed JSON regardless of payload.
	dec := func(data []byte) ([]byte, error) {
		id, ok := SchemaID(data)
		require.True(t, ok)
		assert.Equal(t, uint32(7), id)
		return []byte(`{"decoded":true}`), nil
	}
	s := NewSchemaRegistrySerde(dec)

	framed := []byte{0x00, 0x00, 0x00, 0x00, 0x07, 0xAA, 0xBB}
	assert.True(t, s.CanDeserialize(framed))
	out, err := s.Deserialize(framed)
	require.NoError(t, err)
	assert.Equal(t, `{"decoded":true}`, out)

	// Missing magic byte → error (would trigger fallback).
	assert.False(t, s.CanDeserialize([]byte{0x01, 0x02}))
	_, err = s.Deserialize([]byte{0x01, 0x02})
	assert.Error(t, err)
}

func TestDecodeFallbackMarks(t *testing.T) {
	r, err := BuildRegistry(nil, nil)
	require.NoError(t, err)

	// Explicit "int" on wrong-length, non-UTF-8 bytes → hex fallback.
	text, name, fb := Decode(r, NameInt, []byte{0xFF, 0xFE, 0xFD})
	assert.True(t, fb)
	assert.True(t, IsFallback(name))
	assert.Equal(t, "fffefd", text)
	assert.Equal(t, NameHex+" (fallback)", name)

	// Explicit "int" on invalid but valid-UTF-8 bytes → string fallback.
	text, name, fb = Decode(r, NameInt, []byte("hi"))
	assert.True(t, fb)
	assert.Equal(t, "hi", text)
	assert.Equal(t, NameString+" (fallback)", name)

	// Success path is not marked.
	_, name, fb = Decode(r, NameJSON, []byte(`{"a":1}`))
	assert.False(t, fb)
	assert.False(t, IsFallback(name))
}

func TestValidate(t *testing.T) {
	r, err := BuildRegistry(nil, nil)
	require.NoError(t, err)

	assert.NoError(t, Validate(r, Auto, []byte("x")))
	assert.NoError(t, Validate(r, "", []byte("x")))
	assert.ErrorAs(t, Validate(r, "does-not-exist", nil), &UnknownSerdeError{})
	// int serde cannot decode 2 bytes.
	assert.Error(t, Validate(r, NameInt, []byte{0x01, 0x02}))
}

func TestMsgpackDecode(t *testing.T) {
	// {"a":1} as msgpack.
	packed, err := MsgpackSerde{}.Serialize(`{"a":1}`)
	require.NoError(t, err)
	assert.True(t, MsgpackSerde{}.CanDeserialize(packed))
	out, err := MsgpackSerde{}.Deserialize(packed)
	require.NoError(t, err)
	assert.Contains(t, out, `"a"`)
	assert.Contains(t, out, "1")
}

func TestRawProtobufDecode(t *testing.T) {
	// field 1 (varint) = 42; field 2 (bytes) = "Bob".
	data := []byte{0x08, 0x2A, 0x12, 0x03, 'B', 'o', 'b'}
	assert.True(t, RawProtobufSerde{}.CanDeserialize(data))
	out, err := RawProtobufSerde{}.Deserialize(data)
	require.NoError(t, err)
	assert.Contains(t, out, "field_1")
	assert.Contains(t, out, "42")
	assert.Contains(t, out, "Bob")
}

func TestDescriptorProtobufDecode(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
		Package: proto.String("test"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: proto.String("Person"),
			Field: []*descriptorpb.FieldDescriptorProto{
				{Name: proto.String("id"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()},
				{Name: proto.String("name"), Number: proto.Int32(2), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()},
			},
		}},
	}
	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{fdp}}
	raw, err := proto.Marshal(fds)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "test.desc")
	require.NoError(t, os.WriteFile(path, raw, 0o600))

	s, err := NewDescriptorProtobufSerde("person", path, "test.Person")
	require.NoError(t, err)

	data := []byte{0x08, 0x2A, 0x12, 0x03, 'B', 'o', 'b'}
	out, err := s.Deserialize(data)
	require.NoError(t, err)
	assert.Contains(t, out, "Bob")
	assert.Contains(t, out, "42")

	// Missing message type fails to build.
	_, err = NewDescriptorProtobufSerde("x", path, "test.Missing")
	assert.Error(t, err)
}

func TestBuildRegistryDescriptorFromConfig(t *testing.T) {
	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{
		Name:    proto.String("t.proto"),
		Package: proto.String("t"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name:  proto.String("M"),
			Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("id"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}},
		}},
	}}}
	raw, err := proto.Marshal(fds)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "t.desc")
	require.NoError(t, os.WriteFile(path, raw, 0o600))

	r, err := BuildRegistry(nil, []SerdeConfig{{Name: "myproto", DescriptorPath: path, MessageType: "t.M"}})
	require.NoError(t, err)
	_, ok := r.Get("myproto")
	assert.True(t, ok)

	// A bad descriptor path fails the build.
	_, err = BuildRegistry(nil, []SerdeConfig{{Name: "bad", DescriptorPath: "/nope", MessageType: "x"}})
	assert.Error(t, err)
}

func TestListSerdesContents(t *testing.T) {
	r, err := BuildRegistry(nil, nil)
	require.NoError(t, err)
	names := r.Names()
	for _, want := range []string{
		NameString, NameHex, NameJSON, NameNull, NameInt, NameLong, NameFloat, NameDouble,
		NameMsgpack, NameRawProtobuf, NameSchemaRegistry, NameConsumerOffsetsKey, NameConsumerOffsetsValue,
	} {
		assert.Contains(t, names, want)
	}
	// Names are sorted.
	assert.True(t, sortedStrings(names))
}

func sortedStrings(s []string) bool {
	for i := 1; i < len(s); i++ {
		if s[i-1] > s[i] {
			return false
		}
	}
	return true
}

func TestSelectSerdePerCluster(t *testing.T) {
	configs := []SerdeConfig{
		{Name: "myproto", TopicPattern: `^orders\..*`, Target: "value"},
		{Name: NameHex, Target: "key"},
	}
	// value on matching topic → myproto.
	assert.Equal(t, "myproto", SelectSerde(configs, "orders.created", false))
	// key on any topic → hex.
	assert.Equal(t, NameHex, SelectSerde(configs, "orders.created", true))
	// value on non-matching topic → no binding.
	assert.Equal(t, "", SelectSerde(configs, "users", false))
}

func TestConsumerOffsetsSerdes(t *testing.T) {
	// key: version 1, group "g1", topic "t1", partition 3.
	key := &binBuilder{}
	key.int16(1)
	key.string("g1")
	key.string("t1")
	key.int32(3)
	out, err := ConsumerOffsetsKeySerde{}.Deserialize(key.bytes())
	require.NoError(t, err)
	assert.Contains(t, out, `"group": "g1"`)
	assert.Contains(t, out, `"topic": "t1"`)
	assert.Contains(t, out, `"partition": 3`)
	assert.True(t, ConsumerOffsetsKeySerde{}.CanDeserialize(key.bytes()))

	// value: version 1, offset 100, metadata "m", commitTs 5, expireTs 9.
	val := &binBuilder{}
	val.int16(1)
	val.int64(100)
	val.string("m")
	val.int64(5)
	val.int64(9)
	vout, err := ConsumerOffsetsValueSerde{}.Deserialize(val.bytes())
	require.NoError(t, err)
	assert.Contains(t, vout, `"offset": 100`)
	assert.Contains(t, vout, `"metadata": "m"`)
	assert.Contains(t, vout, `"expireTimestamp": 9`)
}

// binBuilder writes the big-endian, int16-length-prefixed Kafka wire format.
type binBuilder struct{ b []byte }

func (w *binBuilder) int16(v int16) {
	w.b = append(w.b, byte(v>>8), byte(v))
}
func (w *binBuilder) int32(v int32) {
	w.b = append(w.b, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}
func (w *binBuilder) int64(v int64) {
	for i := 7; i >= 0; i-- {
		w.b = append(w.b, byte(v>>(8*i)))
	}
}
func (w *binBuilder) string(s string) {
	w.int16(int16(len(s)))
	w.b = append(w.b, []byte(s)...)
}
func (w *binBuilder) bytes() []byte { return w.b }

func TestJSONSerdeSerialize(t *testing.T) {
	_, err := JSONSerde{}.Serialize("not json")
	assert.Error(t, err)
	b, err := JSONSerde{}.Serialize(`{"a":1}`)
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(b), "a"))
}
