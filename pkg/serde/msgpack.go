package serde

import (
	"encoding/json"

	"github.com/vmihailenco/msgpack/v5"
)

// NameMsgpack is the MessagePack serde name.
const NameMsgpack = "msgpack"

// MsgpackSerde decodes MessagePack payloads to JSON. It wraps the decode path
// previously hardwired in kafds.handleMessageWithConfig.
type MsgpackSerde struct{}

func (MsgpackSerde) Name() string { return NameMsgpack }

func (MsgpackSerde) CanDeserialize(d []byte) bool {
	if len(d) == 0 {
		return false
	}
	var v any
	return msgpack.Unmarshal(d, &v) == nil
}

func (MsgpackSerde) Deserialize(d []byte) (string, error) {
	var v any
	if err := msgpack.Unmarshal(d, &v); err != nil {
		return "", err
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (MsgpackSerde) Serialize(text string) ([]byte, error) {
	var v any
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		return nil, err
	}
	return msgpack.Marshal(v)
}
