package kafds

import (
	"github.com/IBM/sarama"
)

// ConsumerInterface creates consumer groups; replaceable for testing.
type ConsumerInterface interface {
	CreateConsumerGroupFromClient(group string, client sarama.Client) (sarama.ConsumerGroup, error)
}

// ConfigProviderInterface provides configuration for consumers
type ConfigProviderInterface interface {
	GetConsumerConfig() (*sarama.Config, error)
	GetClientFromConfig(config *sarama.Config) (sarama.Client, error)
}

// DefaultConsumer implements ConsumerInterface using real Sarama
type DefaultConsumer struct{}

func (c *DefaultConsumer) CreateConsumerGroupFromClient(group string, client sarama.Client) (sarama.ConsumerGroup, error) {
	return sarama.NewConsumerGroupFromClient(group, client)
}

// DefaultConfigProvider implements ConfigProviderInterface
type DefaultConfigProvider struct{}

func (cp *DefaultConfigProvider) GetConsumerConfig() (*sarama.Config, error) {
	return getConfig()
}

func (cp *DefaultConfigProvider) GetClientFromConfig(config *sarama.Config) (sarama.Client, error) {
	return getClientFromConfig(config)
}

// Global instances that can be replaced for testing
var (
	consumerInstance       ConsumerInterface       = &DefaultConsumer{}
	configProviderInstance ConfigProviderInterface = &DefaultConfigProvider{}
)
