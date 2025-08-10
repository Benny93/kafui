package kafds

import (
	"github.com/IBM/sarama"
	"github.com/birdayz/kaf/pkg/config"
)

// ClusterAdminInterface wraps the methods we actually use from sarama.ClusterAdmin
type ClusterAdminInterface interface {
	ListTopics() (map[string]sarama.TopicDetail, error)
	ListConsumerGroups() (map[string]string, error)
	DescribeConsumerGroups(groups []string) ([]*sarama.GroupDescription, error)
	Close() error
}

// KafkaClientFactory interface for creating Kafka clients
type KafkaClientFactory interface {
	CreateClusterAdmin(brokers []string, config *sarama.Config) (ClusterAdminInterface, error)
	CreateClient(brokers []string, config *sarama.Config) (sarama.Client, error)
}

// ConfigManager interface for configuration operations
type ConfigManager interface {
	ReadConfig(configFile string) (config.Config, error)
	GetActiveCluster(cfg config.Config) *config.Cluster
}

// DefaultKafkaClientFactory implements KafkaClientFactory using real Sarama clients
type DefaultKafkaClientFactory struct{}

func (f *DefaultKafkaClientFactory) CreateClusterAdmin(brokers []string, config *sarama.Config) (ClusterAdminInterface, error) {
	return sarama.NewClusterAdmin(brokers, config)
}

func (f *DefaultKafkaClientFactory) CreateClient(brokers []string, config *sarama.Config) (sarama.Client, error) {
	return sarama.NewClient(brokers, config)
}

// DefaultConfigManager implements ConfigManager using real config operations
type DefaultConfigManager struct{}

func (m *DefaultConfigManager) ReadConfig(configFile string) (config.Config, error) {
	return config.ReadConfig(configFile)
}

func (m *DefaultConfigManager) GetActiveCluster(cfg config.Config) *config.Cluster {
	return cfg.ActiveCluster()
}

// Global instances that can be replaced for testing
var (
	kafkaClientFactory KafkaClientFactory = &DefaultKafkaClientFactory{}
	configManager      ConfigManager      = &DefaultConfigManager{}
)