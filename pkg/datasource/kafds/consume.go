package kafds

import (
	"bytes"
	"context"
	"encoding/binary"
	_ "encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
	"github.com/birdayz/kaf/pkg/avro"
	"github.com/birdayz/kaf/pkg/proto"
	"github.com/golang/protobuf/jsonpb"
	prettyjson "github.com/hokaccha/go-prettyjson"
	"github.com/spf13/cobra"
	"github.com/vmihailenco/msgpack/v5"
)

var (
	offsetFlag      string
	groupFlag       string
	groupCommitFlag bool
	outputFormat    = OutputFormatDefault
	// Deprecated: Use outputFormat instead.
	raw         bool
	follow      bool
	tail        int32
	schemaCache *avro.SchemaCache
	keyfmt      *prettyjson.Formatter

	protoType    string
	keyProtoType string

	flagPartitions []int32

	limitMessagesFlag int64

	reg *proto.DescriptorRegistry
)

type offsets struct {
	newest int64
	oldest int64
}

func getOffsets(client sarama.Client, topic string, partition int32) (*offsets, error) {
	newest, err := client.GetOffset(topic, partition, sarama.OffsetNewest)
	if err != nil {
		return nil, err
	}

	oldest, err := client.GetOffset(topic, partition, sarama.OffsetOldest)
	if err != nil {
		return nil, err
	}

	return &offsets{
		newest: newest,
		oldest: oldest,
	}, nil
}

var handler api.MessageHandlerFunc // todo remove global var

func DoConsume(ctx context.Context, topic string, consumeFlags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) {
	var offset int64
	cfg, err := getConfig()
	if err != nil {
		onError(err)
	}
	client, err := getClientFromConfig(cfg)
	if err != nil {
		onError(err)
	}
	handler = handleMessage
	// Allow deprecated flag to override when outputFormat is not specified, or default.
	if outputFormat == OutputFormatDefault && raw {
		outputFormat = OutputFormatRaw
	}
	offsetFlag = "oldest" // TODO as parameter
	follow = consumeFlags.Follow
	tail = consumeFlags.Tail

	switch offsetFlag {
	case "oldest":
		offset = sarama.OffsetOldest
		cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	case "newest":
		offset = sarama.OffsetNewest
		cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	default:
		o, err := strconv.ParseInt(offsetFlag, 10, 64)
		if err != nil {
			onError(err)
		}
		offset = o
	}

	if groupFlag != "" {
		withConsumerGroup(ctx, client, topic, groupFlag)
	} else {
		withoutConsumerGroup(ctx, client, topic, offset, onError)
	}

}

type g struct{}

func (g *g) Setup(s sarama.ConsumerGroupSession) error {
	return nil
}

func (g *g) Cleanup(s sarama.ConsumerGroupSession) error {
	return nil
}

func (g *g) ConsumeClaim(s sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {

	mu := sync.Mutex{} // Synchronizes stderr and stdout.
	for msg := range claim.Messages() {
		handleMessage(msg, &mu)
		if groupCommitFlag {
			s.MarkMessage(msg, "")
		}
	}
	return nil
}

func withConsumerGroup(ctx context.Context, client sarama.Client, topic, group string) error {
	cg, err := sarama.NewConsumerGroupFromClient(group, client)
	if err != nil {
		return fmt.Errorf("Failed to create consumer group: %v", err)
	}

	err = cg.Consume(ctx, []string{topic}, &g{})
	if err != nil {
		return fmt.Errorf("Error on consume: %v", err)
	}
	return nil
}

func withoutConsumerGroup(ctx context.Context, client sarama.Client, topic string, offset int64, onError func(err any)) {
	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		onError(fmt.Sprintf("Unable to create consumer from client: %v\n", err))
		return
	}

	var partitions []int32
	if len(flagPartitions) == 0 {
		partitions, err = consumer.Partitions(topic)
		if err != nil {
			onError(fmt.Sprintf("Unable to get partitions: %v\n", err))
			return
		}
	} else {
		partitions = flagPartitions
	}

	wg := sync.WaitGroup{}
	mu := sync.Mutex{} // Synchronizes stderr and stdout.
	for _, partition := range partitions {
		wg.Add(1)

		go func(partition int32, offset int64) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					onError(r)
					return
				}
			}()

			offsets, err := getOffsets(client, topic, partition)
			if err != nil {
				onError(fmt.Errorf("Failed to get %s offsets for partition %d: %w", topic, partition, err))
			}

			if tail != 0 {
				offset = offsets.newest - int64(tail)
				if offset < offsets.oldest {
					offset = offsets.oldest
				}
			}

			// Already at end of partition, return early
			if !follow && offsets.newest == offsets.oldest {
				return
			}

			pc, err := consumer.ConsumePartition(topic, partition, offset)
			if err != nil {
				onError(fmt.Errorf("Unable to consume partition: %v %v %v %v\n", topic, partition, offset, err))
			}

			var count int64 = 0
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-pc.Messages():
					handleMessage(msg, &mu)
					count++
					if limitMessagesFlag > 0 && count >= limitMessagesFlag {
						return
					}
					if !follow && msg.Offset+1 >= pc.HighWaterMarkOffset() {
						return
					}
				}
			}
		}(partition, offset)
	}
	wg.Wait()
}

func handleMessage(msg *sarama.ConsumerMessage, mu *sync.Mutex) {
	var stderr bytes.Buffer

	var dataToDisplay []byte
	var keyToDisplay []byte
	var err error

	if protoType != "" {
		dataToDisplay, err = protoDecode(reg, msg.Value, protoType)
		if err != nil {
			fmt.Fprintf(&stderr, "failed to decode proto. falling back to binary outputla. Error: %v\n", err)
		}
	} else {
		dataToDisplay, err = avroDecode(msg.Value)
		if err != nil {
			fmt.Fprintf(&stderr, "could not decode Avro data: %v\n", err)
		}
	}

	if keyProtoType != "" {
		keyToDisplay, err = protoDecode(reg, msg.Key, keyProtoType)
		if err != nil {
			fmt.Fprintf(&stderr, "failed to decode proto key. falling back to binary outputla. Error: %v\n", err)
		}
	} else {
		keyToDisplay, err = avroDecode(msg.Key)
		if err != nil {
			fmt.Fprintf(&stderr, "could not decode Avro data: %v\n", err)
		}
	}

	if decodeMsgPack {
		var obj interface{}
		err = msgpack.Unmarshal(msg.Value, &obj)
		if err != nil {
			fmt.Fprintf(&stderr, "could not decode msgpack data: %v\n", err)
		}

		dataToDisplay, err = json.Marshal(obj)
		if err != nil {
			fmt.Fprintf(&stderr, "could not decode msgpack data: %v\n", err)
		}
	}

	//dataToDisplay = formatMessage(msg, dataToDisplay, keyToDisplay, &stderr)

	//mu.Lock()
	//stderr.WriteTo(errWriter)
	//_, _ = colorableOut.Write(dataToDisplay)
	//fmt.Fprintln(outWriter)
	keySchema := getSchemaIdIfPresent(msg.Key)
	valueSchema := getSchemaIdIfPresent(msg.Value)
	headers := make([]api.MessageHeader, 0)
	for _, saramaHeader := range msg.Headers {
		header := api.MessageHeader{
			Key:   string(saramaHeader.Key),
			Value: string(saramaHeader.Value),
		}
		headers = append(headers, header)
	}

	newMessage := api.Message{

		Key:           string(keyToDisplay),
		Headers:       headers,
		Value:         string(dataToDisplay),
		Offset:        msg.Offset,
		Partition:     msg.Partition,
		KeySchemaID:   keySchema,
		ValueSchemaID: valueSchema,
	}
	handler(newMessage)
	//mu.Unlock()
}

func getSchemaIdIfPresent(b []byte) string {
	// Ensure avro header is present with the magic start-byte.
	if len(b) < 5 || b[0] != 0x00 {
		// The message does not contain Avro-encoded data
		return ""
	}

	// Schema ID is stored in the 4 bytes following the magic byte.
	schemaID := binary.BigEndian.Uint32(b[1:5])
	return fmt.Sprint(int(schemaID))
}

func formatMessage(msg *sarama.ConsumerMessage, rawMessage []byte, keyToDisplay []byte, stderr *bytes.Buffer) []byte {
	switch outputFormat {
	case OutputFormatRaw:
		return rawMessage
	case OutputFormatJSON:
		jsonMessage := make(map[string]interface{})

		jsonMessage["partition"] = msg.Partition
		jsonMessage["offset"] = msg.Offset
		jsonMessage["timestamp"] = msg.Timestamp

		if len(msg.Headers) > 0 {
			jsonMessage["headers"] = msg.Headers
		}

		jsonMessage["key"] = formatJSON(keyToDisplay)
		jsonMessage["payload"] = formatJSON(rawMessage)

		jsonToDisplay, err := json.Marshal(jsonMessage)
		if err != nil {
			fmt.Fprintf(stderr, "could not decode JSON data: %v", err)
		}

		return jsonToDisplay
	case OutputFormatDefault:
		fallthrough
	default:
		if isJSON(rawMessage) {
			rawMessage = formatValue(rawMessage)
		}

		if isJSON(keyToDisplay) {
			keyToDisplay = formatKey(keyToDisplay)
		}

		//w := tabwriter.NewWriter(stderr, tabwriterMinWidth, tabwriterWidth, tabwriterPadding, tabwriterPadChar, tabwriterFlags)
		constructedMsg := ""
		if len(msg.Headers) > 0 {
			//fmt.Fprintf(w, "Headers:\n")
			constructedMsg += "Headers:\n"
		}

		for _, hdr := range msg.Headers {
			var hdrValue string
			// Try to detect azure eventhub-specific encoding
			if len(hdr.Value) > 0 {
				switch hdr.Value[0] {
				case 161:
					hdrValue = string(hdr.Value[2 : 2+hdr.Value[1]])
				case 131:
					hdrValue = strconv.FormatUint(binary.BigEndian.Uint64(hdr.Value[1:9]), 10)
				default:
					hdrValue = string(hdr.Value)
				}
			}

			//fmt.Fprintf(w, "\tKey: %v\tValue: %v\n", string(hdr.Key), hdrValue)
			constructedMsg += fmt.Sprintf("\tKey: %v\tValue: %v\n", string(hdr.Key), hdrValue)

		}

		if msg.Key != nil && len(msg.Key) > 0 {
			//fmt.Fprintf(w, "Key:\t%v\n", string(keyToDisplay))
			constructedMsg += fmt.Sprintf("Key:\t%v\n", string(keyToDisplay))
		}
		//fmt.Fprintf(w, "Partition:\t%v\nOffset:\t%v\nTimestamp:\t%v\n", msg.Partition, msg.Offset, msg.Timestamp)
		//w.Flush()
		constructedMsg += fmt.Sprintf("Partition:\t%v\nOffset:\t%v\nTimestamp:\t%v\n", msg.Partition, msg.Offset, msg.Timestamp)
		constructedMsg += string(rawMessage)
		return []byte(constructedMsg)
	}
}

// proto to JSON
func protoDecode(reg *proto.DescriptorRegistry, b []byte, _type string) ([]byte, error) {
	dynamicMessage := reg.MessageForType(_type)
	if dynamicMessage == nil {
		return b, nil
	}

	err := dynamicMessage.Unmarshal(b)
	if err != nil {
		return nil, err
	}

	var m jsonpb.Marshaler
	var w bytes.Buffer

	err = m.Marshal(&w, dynamicMessage)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil

}

func avroDecode(b []byte) ([]byte, error) {
	if schemaCache != nil {
		return schemaCache.DecodeMessage(b)
	}
	return b, nil
}

func formatKey(key []byte) []byte {
	if b, err := keyfmt.Format(key); err == nil {
		return b
	}
	return key

}

func formatValue(key []byte) []byte {
	if b, err := prettyjson.Format(key); err == nil {
		return b
	}
	return key
}

func formatJSON(data []byte) interface{} {
	var i interface{}
	if err := json.Unmarshal(data, &i); err != nil {
		return string(data)
	}

	return i
}

func isJSON(data []byte) bool {
	var i interface{}
	if err := json.Unmarshal(data, &i); err == nil {
		return true
	}
	return false
}

type OutputFormat string

const (
	OutputFormatDefault OutputFormat = "default"
	OutputFormatRaw     OutputFormat = "raw"
	OutputFormatJSON    OutputFormat = "json"
)

func (e *OutputFormat) String() string {
	return string(*e)
}

func (e *OutputFormat) Set(v string) error {
	switch v {
	case "default", "raw", "json":
		*e = OutputFormat(v)
		return nil
	default:
		return fmt.Errorf("must be one of: default, raw, json")
	}
}

func (e *OutputFormat) Type() string {
	return "OutputFormat"
}

func completeOutputFormat(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"default", "raw", "json"}, cobra.ShellCompDirectiveNoFileComp
}
