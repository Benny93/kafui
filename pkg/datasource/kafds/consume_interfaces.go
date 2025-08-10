package kafds

import (
	"github.com/IBM/sarama"
	"github.com/Benny93/kafui/pkg/api"
)

// ConsumerInterface wraps the consumer functionality for testing
type ConsumerInterface interface {
	GetOffsets(client sarama.Client, topic string, partition int32) (*offsets, error)
	CreateConsumerFromClient(client sarama.Client, topic string, partition int32) (sarama.PartitionConsumer, error)
	CreateConsumerGroupFromClient(group string, client sarama.Client) (sarama.ConsumerGroup, error)
}

// MessageProcessorInterface handles message processing and formatting
type MessageProcessorInterface interface {
	ProcessMessage(msg *sarama.ConsumerMessage, handler api.MessageHandlerFunc) error
	FormatKey(key []byte) []byte
	FormatValue(value []byte) []byte
	DecodeAvro(data []byte) ([]byte, error)
	DecodeProto(data []byte, protoType string) ([]byte, error)
}

// ConfigProviderInterface provides configuration for consumers
type ConfigProviderInterface interface {
	GetConsumerConfig() (*sarama.Config, error)
	GetClientFromConfig(config *sarama.Config) (sarama.Client, error)
}

// DefaultConsumer implements ConsumerInterface using real Sarama
type DefaultConsumer struct{}

func (c *DefaultConsumer) GetOffsets(client sarama.Client, topic string, partition int32) (*offsets, error) {
	return getOffsets(client, topic, partition)
}

func (c *DefaultConsumer) CreateConsumerFromClient(client sarama.Client, topic string, partition int32) (sarama.PartitionConsumer, error) {
	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		return nil, err
	}
	return consumer.ConsumePartition(topic, partition, sarama.OffsetOldest)
}

func (c *DefaultConsumer) CreateConsumerGroupFromClient(group string, client sarama.Client) (sarama.ConsumerGroup, error) {
	return sarama.NewConsumerGroupFromClient(group, client)
}

// DefaultMessageProcessor implements MessageProcessorInterface
type DefaultMessageProcessor struct{}

func (p *DefaultMessageProcessor) ProcessMessage(msg *sarama.ConsumerMessage, handler api.MessageHandlerFunc) error {
	apiMsg := api.Message{
		Key:       string(msg.Key),
		Value:     string(msg.Value),
		Offset:    msg.Offset,
		Partition: msg.Partition,
		Headers:   convertHeaders(msg.Headers),
	}
	handler(apiMsg)
	return nil
}

func (p *DefaultMessageProcessor) FormatKey(key []byte) []byte {
	return formatKey(key)
}

func (p *DefaultMessageProcessor) FormatValue(value []byte) []byte {
	return formatValue(value)
}

func (p *DefaultMessageProcessor) DecodeAvro(data []byte) ([]byte, error) {
	return avroDecode(data)
}

func (p *DefaultMessageProcessor) DecodeProto(data []byte, protoType string) ([]byte, error) {
	if reg != nil {
		return protoDecode(reg, data, protoType)
	}
	return data, nil
}

// DefaultConfigProvider implements ConfigProviderInterface
type DefaultConfigProvider struct{}

func (cp *DefaultConfigProvider) GetConsumerConfig() (*sarama.Config, error) {
	return getConfig()
}

func (cp *DefaultConfigProvider) GetClientFromConfig(config *sarama.Config) (sarama.Client, error) {
	return getClientFromConfig(config)
}

// Helper function to convert Sarama headers to API headers
func convertHeaders(saramaHeaders []*sarama.RecordHeader) []api.MessageHeader {
	headers := make([]api.MessageHeader, len(saramaHeaders))
	for i, h := range saramaHeaders {
		headers[i] = api.MessageHeader{
			Key:   string(h.Key),
			Value: string(h.Value),
		}
	}
	return headers
}

// Global instances that can be replaced for testing
var (
	consumerInstance       ConsumerInterface       = &DefaultConsumer{}
	messageProcessorInstance MessageProcessorInterface = &DefaultMessageProcessor{}
	configProviderInstance ConfigProviderInterface   = &DefaultConfigProvider{}
)