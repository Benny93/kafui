package kafds

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
	"github.com/birdayz/kaf/pkg/avro"
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
	Follow            bool
	Tail              int32
	FlagPartitions    []int32
	LimitMessagesFlag int64

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
		OffsetFlag:        "oldest",
		FlagPartitions:    []int32{},
		LimitMessagesFlag: 0,
	}
}

func DoConsume(ctx context.Context, topic string, consumeFlags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) {
	DoConsumeWithDeps(ctx, topic, consumeFlags, handleMessage, onError, configProviderInstance, consumerInstance)
}

func DoConsumeWithDeps(ctx context.Context, topic string, consumeFlags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any), configProvider ConfigProviderInterface, consumer ConsumerInterface) {
	config := DefaultConsumeConfig()
	DoConsumeWithConfig(ctx, topic, consumeFlags, handleMessage, onError, configProvider, consumer, config)
}

func DoConsumeWithConfig(ctx context.Context, topic string, consumeFlags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any), configProvider ConfigProviderInterface, consumer ConsumerInterface, config *ConsumeConfig) {
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
	// This call owns the client: both consume paths below return only once
	// they are done with it, so closing here also drops its broker connections.
	if client != nil {
		defer client.Close()
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

	// Avro keys/values are kept as raw bytes here and decoded lazily by
	// DecodeMessage, which uses the shared schema cache.

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
		if err := withConsumerGroupWithDeps(ctx, client, topic, config.GroupFlag, consumer, config, handleMessage); err != nil {
			onError(err)
		}
	} else {
		withoutConsumerGroupWithDeps(ctx, client, topic, offset, onError, config, handleMessage)
	}
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

func withConsumerGroupWithDeps(ctx context.Context, client sarama.Client, topic, group string, consumer ConsumerInterface, config *ConsumeConfig, handler api.MessageHandlerFunc) error {
	cg, err := consumer.CreateConsumerGroupFromClient(group, client)
	if err != nil {
		return fmt.Errorf("Failed to create consumer group: %w", err)
	}
	defer cg.Close()

	groupHandler := &consumerGroupHandler{
		config:  config,
		handler: handler,
	}

	// Consume returns after every rebalance, so rejoin until the caller is done.
	for ctx.Err() == nil {
		if err := cg.Consume(ctx, []string{topic}, groupHandler); err != nil {
			if errors.Is(err, sarama.ErrClosedConsumerGroup) || ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("Error on consume: %w", err)
		}
	}
	return nil
}

func withoutConsumerGroupWithDeps(ctx context.Context, client sarama.Client, topic string, offset int64, onError func(err any), config *ConsumeConfig, handler api.MessageHandlerFunc) {
	if client == nil {
		onError(fmt.Sprintf("Unable to create consumer from client: client is nil\n"))
		return
	}
	saramaConsumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		onError(fmt.Sprintf("Unable to create consumer from client: %v\n", err))
		return
	}
	// Runs after wg.Wait below, once every partition goroutine has returned.
	defer saramaConsumer.Close()

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
			// Stop the partition's background fetch loop once we're done with
			// it; otherwise it keeps pulling data into its buffer indefinitely.
			defer pc.AsyncClose()

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
		// newest is one past the last record, so the last readable target is newest-1.
		if end == offs.newest && end > offs.oldest {
			end--
		}
		// [start, end] holds exactly `window` offsets, so the per-partition
		// message limit is reached on the target itself rather than just before it.
		start = end - window + 1
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

func handleMessageWithConfig(msg *sarama.ConsumerMessage, mu *sync.Mutex, config *ConsumeConfig, handler api.MessageHandlerFunc) {
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
	// Other messages are stored as plain strings.
	var keyStr, valueStr string
	var rawKey, rawValue []byte

	if keySchema != "" {
		rawKey = make([]byte, len(msg.Key))
		copy(rawKey, msg.Key)
	} else {
		keyStr = string(msg.Key)
	}

	if valueSchema != "" {
		rawValue = make([]byte, len(msg.Value))
		copy(rawValue, msg.Value)
	} else {
		valueStr = string(msg.Value)
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
		KeySerde:      serdeName(keySchema),
		ValueSerde:    serdeName(valueSchema),
	}

	if handler != nil {
		handler(newMessage)
	}
}

// serdeName reports the decoder used for a key or value at consume time:
// Avro when a schema ID is present, otherwise plain string. DecodeMessage
// refines this with the serde framework.
func serdeName(schemaID string) string {
	if schemaID != "" {
		return "avro"
	}
	return "string"
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

func avroDecodeWithCache(b []byte, cache *avro.SchemaCache) ([]byte, error) {
	if cache != nil {
		decoded, err := cache.DecodeMessage(b)
		if err != nil {
			return decoded, err
		}
		return sortJSONKeys(decoded), nil
	}
	return b, nil
}

// sortJSONKeys re-renders a JSON object or array with every object's keys in
// sorted order. goavro renders a record by ranging over a Go map, so without
// this the same fields come out in a different order in every message, which
// makes two similar messages hard to compare. Numbers are kept verbatim (a
// long beyond float64 precision stays exact) and nothing is HTML-escaped.
// Input that is not a JSON object or array is returned unchanged.
func sortJSONKeys(b []byte) []byte {
	trimmed := bytes.TrimSpace(b)
	if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
		return b
	}
	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.UseNumber()
	var v interface{}
	if err := dec.Decode(&v); err != nil || dec.More() {
		return b
	}
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil { // encoding/json sorts map keys
		return b
	}
	return bytes.TrimSuffix(out.Bytes(), []byte("\n"))
}
