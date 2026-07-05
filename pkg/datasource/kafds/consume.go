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
	"time"

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
	// Backward compatibility global variables
	offsetFlag      string
	groupFlag       string
	groupCommitFlag bool
	outputFormat    = OutputFormatDefault
	raw             bool
	follow          bool
	tail            int32
	schemaCache     *avro.SchemaCache
	keyfmt          *prettyjson.Formatter
	protoType       string
	keyProtoType    string
	flagPartitions  []int32
	limitMessagesFlag int64
	reg             *proto.DescriptorRegistry
	handler         api.MessageHandlerFunc // Global handler for backward compatibility
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

// ConsumeConfig holds all configuration for consuming messages
type ConsumeConfig struct {
	OffsetFlag        string
	GroupFlag         string
	GroupCommitFlag   bool
	OutputFormat      OutputFormat
	Raw               bool
	Follow            bool
	Tail              int32
	SchemaCache       *avro.SchemaCache
	Keyfmt            *prettyjson.Formatter
	ProtoType         string
	KeyProtoType      string
	FlagPartitions    []int32
	LimitMessagesFlag int64
	Reg               *proto.DescriptorRegistry
	DecodeMsgPack     bool

	// Typed seek model (MSG-1..4). When Seek is set it drives per-partition
	// offset resolution instead of OffsetFlag.
	Seek          api.SeekMode
	SeekOffset    *int64
	SeekTimestamp *time.Time

	// Resource controls (MSG-10). TailRatePerSec throttles follow delivery;
	// MaxBytesPerSec throttles browse fetches. Zero disables each.
	TailRatePerSec int
	MaxBytesPerSec int

	// OnEvent, when set, receives browse phase/statistics events (MSG-7).
	OnEvent func(api.BrowseEvent)
}

// DefaultConsumeConfig returns a default configuration
func DefaultConsumeConfig() *ConsumeConfig {
	return &ConsumeConfig{
		OutputFormat:      OutputFormatDefault,
		OffsetFlag:        "oldest",
		FlagPartitions:    []int32{},
		LimitMessagesFlag: 0,
	}
}

func DoConsume(ctx context.Context, topic string, consumeFlags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) {
	DoConsumeWithDeps(ctx, topic, consumeFlags, handleMessage, onError, configProviderInstance, consumerInstance, messageProcessorInstance)
}

func DoConsumeWithDeps(ctx context.Context, topic string, consumeFlags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any), configProvider ConfigProviderInterface, consumer ConsumerInterface, processor MessageProcessorInterface) {
	config := DefaultConsumeConfig()
	DoConsumeWithConfig(ctx, topic, consumeFlags, handleMessage, onError, configProvider, consumer, processor, config)
}

func DoConsumeWithConfig(ctx context.Context, topic string, consumeFlags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any), configProvider ConfigProviderInterface, consumer ConsumerInterface, processor MessageProcessorInterface, config *ConsumeConfig) {
	var offset int64
	cfg, err := configProvider.GetConsumerConfig()
	if err != nil {
		onError(err)
		return
	}
	client, err := configProvider.GetClientFromConfig(cfg)
	if err != nil {
		onError(err)
		return
	}

	// Update config from flags
	config.OffsetFlag = consumeFlags.OffsetFlag
	if config.OffsetFlag == "" {
		config.OffsetFlag = "oldest" // Default fallback
	}
	config.Follow = consumeFlags.Follow
	config.Tail = consumeFlags.Tail
	config.GroupFlag = consumeFlags.GroupFlag
	config.LimitMessagesFlag = consumeFlags.LimitMessages
	config.Seek = consumeFlags.Seek
	config.SeekOffset = consumeFlags.SeekOffset
	config.SeekTimestamp = consumeFlags.SeekTimestamp
	if len(consumeFlags.Partitions) > 0 {
		config.FlagPartitions = consumeFlags.Partitions
	}
	if config.Seek == api.SeekLive {
		config.Follow = true
	}

	// Validate the typed seek model before doing any work (MSG-1).
	if err := consumeFlags.Validate(); err != nil {
		onError(err)
		return
	}

	// Allow deprecated flag to override when outputFormat is not specified, or default.
	if config.OutputFormat == OutputFormatDefault && config.Raw {
		config.OutputFormat = OutputFormatRaw
	}

	// Initialize the Avro schema cache from the active cluster config so that
	// Avro-encoded message values and keys are decoded to JSON automatically.
	// getSchemaCache() returns nil (not an error) when no registry URL is set,
	// in which case avroDecodeWithCache passes the raw bytes through unchanged.
	if config.SchemaCache == nil {
		if sc, err := getSchemaCache(); err == nil {
			config.SchemaCache = sc
		}
	}

	switch config.OffsetFlag {
	case "oldest":
		offset = sarama.OffsetOldest
		cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	case "newest", "latest":
		offset = sarama.OffsetNewest
		cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	default:
		o, err := strconv.ParseInt(config.OffsetFlag, 10, 64)
		if err != nil {
			onError(err)
			return
		}
		offset = o
	}

	if config.GroupFlag != "" {
		withConsumerGroupWithDeps(ctx, client, topic, config.GroupFlag, consumer, processor, config, handleMessage)
	} else {
		withoutConsumerGroupWithDeps(ctx, client, topic, offset, onError, consumer, processor, config, handleMessage)
	}
}

type g struct{}

func (g *g) Setup(s sarama.ConsumerGroupSession) error {
	return nil
}

func (g *g) Cleanup(s sarama.ConsumerGroupSession) error {
	return nil
}

type consumerGroupHandler struct {
	config  *ConsumeConfig
	handler api.MessageHandlerFunc
}

func (g *consumerGroupHandler) Setup(s sarama.ConsumerGroupSession) error {
	return nil
}

func (g *consumerGroupHandler) Cleanup(s sarama.ConsumerGroupSession) error {
	return nil
}

func (g *consumerGroupHandler) ConsumeClaim(s sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	mu := sync.Mutex{} // Synchronizes stderr and stdout.
	for msg := range claim.Messages() {
		handleMessageWithConfig(msg, &mu, g.config, g.handler)
		if g.config.GroupCommitFlag {
			s.MarkMessage(msg, "")
		}
	}
	return nil
}

func withConsumerGroup(ctx context.Context, client sarama.Client, topic, group string) error {
	config := DefaultConsumeConfig()
	return withConsumerGroupWithDeps(ctx, client, topic, group, consumerInstance, messageProcessorInstance, config, nil)
}

func withConsumerGroupWithDeps(ctx context.Context, client sarama.Client, topic, group string, consumer ConsumerInterface, processor MessageProcessorInterface, config *ConsumeConfig, handler api.MessageHandlerFunc) error {
	cg, err := consumer.CreateConsumerGroupFromClient(group, client)
	if err != nil {
		return fmt.Errorf("Failed to create consumer group: %v", err)
	}

	groupHandler := &consumerGroupHandler{
		config:  config,
		handler: handler,
	}
	
	err = cg.Consume(ctx, []string{topic}, groupHandler)
	if err != nil {
		return fmt.Errorf("Error on consume: %v", err)
	}
	return nil
}

func withoutConsumerGroup(ctx context.Context, client sarama.Client, topic string, offset int64, onError func(err any)) {
	config := DefaultConsumeConfig()
	withoutConsumerGroupWithDeps(ctx, client, topic, offset, onError, consumerInstance, messageProcessorInstance, config, nil)
}

func withoutConsumerGroupWithDeps(ctx context.Context, client sarama.Client, topic string, offset int64, onError func(err any), consumer ConsumerInterface, processor MessageProcessorInterface, config *ConsumeConfig, handler api.MessageHandlerFunc) {
	if client == nil {
		onError(fmt.Sprintf("Unable to create consumer from client: client is nil\n"))
		return
	}
	saramaConsumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		onError(fmt.Sprintf("Unable to create consumer from client: %v\n", err))
		return
	}

	availablePartitions, err := saramaConsumer.Partitions(topic)
	if err != nil {
		onError(fmt.Sprintf("Unable to get partitions: %v\n", err))
		return
	}

	var partitions []int32
	if len(config.FlagPartitions) == 0 {
		partitions = availablePartitions
	} else {
		// Validate requested partitions against topic metadata (MSG-4).
		avail := make(map[int32]bool, len(availablePartitions))
		for _, p := range availablePartitions {
			avail[p] = true
		}
		for _, p := range config.FlagPartitions {
			if !avail[p] {
				onError(api.NewPartitionError("partition does not exist", topic, p))
				return
			}
		}
		partitions = config.FlagPartitions
	}

	emitEvent(config, api.BrowseEvent{Phase: api.PhaseCreatingConsumer, Description: "creating consumer"})

	// Per-follow-loop rate limiter shared across partitions (MSG-10). In live
	// mode delivery is capped so it can't overwhelm the terminal.
	tailRate := config.TailRatePerSec
	if config.Follow && tailRate == 0 {
		tailRate = api.DefaultTailRate
	}
	limiter := api.NewRateLimiter(tailRate)

	wg := sync.WaitGroup{}
	mu := sync.Mutex{} // Synchronizes stderr and stdout.
	for _, partition := range partitions {
		wg.Add(1)

		go func(partition int32, legacyOffset int64) {
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
				return
			}

			// Skip empty partitions when browsing (MSG-4).
			if !config.Follow && offsets.newest == offsets.oldest {
				return
			}

			start, stop, backward, err := resolvePartitionSeek(client, topic, partition, offsets, config, legacyOffset)
			if err != nil {
				onError(err)
				return
			}

			pc, err := saramaConsumer.ConsumePartition(topic, partition, start)
			if err != nil {
				onError(fmt.Errorf("Unable to consume partition: %v %v %v %v\n", topic, partition, start, err))
				return
			}

			var count int64 = 0

			// In non-follow mode, reset this timer on every received message.
			// If no message arrives within the window we've likely hit the end of
			// readable messages (remaining offsets are control/transaction markers
			// that Sarama filters out but that advance the high-water mark).
			const idleTimeout = 300 * time.Millisecond
			idleTimer := func() <-chan time.Time {
				if config.Follow {
					return nil // no idle timeout in follow mode
				}
				return time.After(idleTimeout)
			}
			idle := idleTimer()

			for {
				select {
				case <-ctx.Done():
					return
				case <-idle:
					// No readable message received within the idle window; the
					// remaining offsets are control messages — we're done.
					return
				case msg := <-pc.Messages():
					// Backward window: stop once we pass the resolved end offset.
					if backward && msg.Offset >= stop {
						return
					}
					if config.Follow {
						if err := limiter.Wait(ctx); err != nil {
							return
						}
					}
					handleMessageWithConfig(msg, &mu, config, handler)
					count++
					if config.LimitMessagesFlag > 0 && count >= config.LimitMessagesFlag {
						return
					}
					if !config.Follow && !backward && msg.Offset+1 >= pc.HighWaterMarkOffset() {
						return
					}
					// Reset idle timer after each real message.
					idle = idleTimer()
				}
			}
		}(partition, offset)
	}
	emitEvent(config, api.BrowseEvent{Phase: api.PhasePolling, Description: "polling partitions"})
	wg.Wait()
	emitEvent(config, api.BrowseEvent{Phase: api.PhaseDone, Description: "done", Done: true})
}

// emitEvent forwards a browse event when a callback is configured (MSG-7).
func emitEvent(config *ConsumeConfig, ev api.BrowseEvent) {
	if config != nil && config.OnEvent != nil {
		config.OnEvent(ev)
	}
}

// resolvePartitionSeek computes the start offset (inclusive), the stop offset
// (exclusive; only meaningful when backward is true), and whether the read runs
// backward, for a single partition given the configured seek model (MSG-2/3/4).
// legacyStart is the offset derived from the deprecated OffsetFlag and is used
// when no typed seek mode is set.
func resolvePartitionSeek(client sarama.Client, topic string, partition int32, offs *offsets, config *ConsumeConfig, legacyStart int64) (start, stop int64, backward bool, err error) {
	window := config.LimitMessagesFlag
	if window <= 0 {
		window = int64(api.DefaultPageSize)
	}

	switch config.Seek {
	case api.SeekFromOffset:
		return clampSeekOffset(*config.SeekOffset, offs), offs.newest, false, nil

	case api.SeekToOffset:
		end := clampSeekOffset(*config.SeekOffset, offs)
		start = end - window
		if start < offs.oldest {
			start = offs.oldest
		}
		return start, end + 1, true, nil // inclusive of the target offset

	case api.SeekFromTimestamp:
		o, e := client.GetOffset(topic, partition, config.SeekTimestamp.UnixMilli())
		if e != nil {
			return 0, 0, false, fmt.Errorf("resolve timestamp offset for partition %d: %w", partition, e)
		}
		if o < 0 { // no message at/after T — nothing new to read; sit at the end
			o = offs.newest
		}
		return clampSeekOffset(o, offs), offs.newest, false, nil

	case api.SeekToTimestamp:
		o, e := client.GetOffset(topic, partition, config.SeekTimestamp.UnixMilli())
		if e != nil {
			return 0, 0, false, fmt.Errorf("resolve timestamp offset for partition %d: %w", partition, e)
		}
		if o < 0 { // no match — read backward from the end
			o = offs.newest
		}
		end := clampSeekOffset(o, offs)
		start = end - window
		if start < offs.oldest {
			start = offs.oldest
		}
		return start, end, true, nil // exclusive: messages strictly before T

	default:
		// Legacy behaviour: newest with a tail window, else the OffsetFlag offset.
		if config.Tail != 0 {
			start = offs.newest - int64(config.Tail)
			if start < offs.oldest {
				start = offs.oldest
			}
			return start, offs.newest, false, nil
		}
		return legacyStart, offs.newest, false, nil
	}
}

// clampSeekOffset constrains a requested offset to the partition's [oldest, newest] range.
func clampSeekOffset(o int64, offs *offsets) int64 {
	if o < offs.oldest {
		return offs.oldest
	}
	if o > offs.newest {
		return offs.newest
	}
	return o
}

func handleMessage(msg *sarama.ConsumerMessage, mu *sync.Mutex) {
	// Backward compatibility function - uses global variables
	config := &ConsumeConfig{
		ProtoType:     protoType,
		KeyProtoType:  keyProtoType,
		DecodeMsgPack: decodeMsgPack,
		Reg:           reg,
		SchemaCache:   schemaCache,
	}
	handleMessageWithConfig(msg, mu, config, handler)
}

func handleMessageWithConfig(msg *sarama.ConsumerMessage, mu *sync.Mutex, config *ConsumeConfig, handler api.MessageHandlerFunc) {
	var stderr bytes.Buffer

	// Default to raw bytes; proto/msgpack decode inline, Avro deferred to DecodeMessage.
	dataToDisplay := msg.Value
	keyToDisplay := msg.Key
	var err error

	if config.ProtoType != "" {
		if decoded, decErr := protoDecode(config.Reg, msg.Value, config.ProtoType); decErr == nil {
			dataToDisplay = decoded
		} else {
			fmt.Fprintf(&stderr, "failed to decode proto. falling back to binary output. Error: %v\n", decErr)
		}
	}

	if config.KeyProtoType != "" {
		if decoded, decErr := protoDecode(config.Reg, msg.Key, config.KeyProtoType); decErr == nil {
			keyToDisplay = decoded
		} else {
			fmt.Fprintf(&stderr, "failed to decode proto key. falling back to binary output. Error: %v\n", decErr)
		}
	}

	if config.DecodeMsgPack {
		var obj interface{}
		err = msgpack.Unmarshal(msg.Value, &obj)
		if err != nil {
			fmt.Fprintf(&stderr, "could not decode msgpack data: %v\n", err)
		} else {
			dataToDisplay, err = json.Marshal(obj)
			if err != nil {
				fmt.Fprintf(&stderr, "could not decode msgpack data: %v\n", err)
			}
		}
	}

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

	// For Avro-encoded messages, store raw bytes and defer decoding to DecodeMessage.
	// For proto/msgpack/plain messages, store the already-decoded string directly.
	var keyStr, valueStr string
	var rawKey, rawValue []byte

	if keySchema != "" {
		rawKey = make([]byte, len(msg.Key))
		copy(rawKey, msg.Key)
	} else {
		keyStr = string(keyToDisplay)
	}

	if valueSchema != "" {
		rawValue = make([]byte, len(msg.Value))
		copy(rawValue, msg.Value)
	} else {
		valueStr = string(dataToDisplay)
	}

	// Per-message metadata (MSG-5). Distinguish null (nil) from empty (len 0).
	var keySize, valueSize *int
	keyNull := msg.Key == nil
	valueNull := msg.Value == nil
	if !keyNull {
		n := len(msg.Key)
		keySize = &n
	}
	if !valueNull {
		n := len(msg.Value)
		valueSize = &n
	}
	headersSize := 0
	for _, h := range msg.Headers {
		headersSize += len(h.Key) + len(h.Value)
	}
	tsType := api.TimestampTypeNone
	if !msg.BlockTimestamp.IsZero() {
		tsType = api.TimestampTypeLogAppend
	} else if !msg.Timestamp.IsZero() {
		tsType = api.TimestampTypeCreate
	}

	newMessage := api.Message{
		Key:           keyStr,
		Value:         valueStr,
		RawKey:        rawKey,
		RawValue:      rawValue,
		Headers:       headers,
		Offset:        msg.Offset,
		Partition:     msg.Partition,
		KeySchemaID:   keySchema,
		ValueSchemaID: valueSchema,
		Timestamp:     msg.Timestamp,
		TimestampType: tsType,
		KeySize:       keySize,
		ValueSize:     valueSize,
		HeadersSize:   headersSize,
		KeyNull:       keyNull,
		ValueNull:     valueNull,
		KeySerde:      serdeName(keySchema, config.KeyProtoType, config),
		ValueSerde:    serdeName(valueSchema, config.ProtoType, config),
	}

	if handler != nil {
		handler(newMessage)
	}
}

// serdeName reports the decoder currently used for a key or value. It is a
// placeholder until the serde framework (MSG-11..) replaces the hardwired paths.
func serdeName(schemaID, protoType string, config *ConsumeConfig) string {
	switch {
	case schemaID != "":
		return "avro"
	case protoType != "":
		return "protobuf"
	case config.DecodeMsgPack:
		return "msgpack"
	default:
		return "string"
	}
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
	if reg == nil {
		return b, nil
	}
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

func avroDecodeWithCache(b []byte, cache *avro.SchemaCache) ([]byte, error) {
	if cache != nil {
		return cache.DecodeMessage(b)
	}
	return b, nil
}

func formatKey(key []byte) []byte {
	if keyfmt != nil {
		if b, err := keyfmt.Format(key); err == nil {
			return b
		}
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
