package kafds

import (
	"errors"
	"github.com/IBM/sarama"
	"github.com/birdayz/kaf/pkg/config"
)

// MockKafkaClientFactory for testing
type MockKafkaClientFactory struct {
	ShouldFailClusterAdmin bool
	ShouldFailClient       bool
	MockClusterAdmin       ClusterAdminInterface
	MockClient             sarama.Client
}

func (m *MockKafkaClientFactory) CreateClusterAdmin(brokers []string, config *sarama.Config) (ClusterAdminInterface, error) {
	if m.ShouldFailClusterAdmin {
		return nil, errors.New("mock cluster admin creation failed")
	}
	return m.MockClusterAdmin, nil
}

func (m *MockKafkaClientFactory) CreateClient(brokers []string, config *sarama.Config) (sarama.Client, error) {
	if m.ShouldFailClient {
		return nil, errors.New("mock client creation failed")
	}
	return m.MockClient, nil
}

// MockConfigManager for testing
type MockConfigManager struct {
	ShouldFailReadConfig bool
	MockConfig           config.Config
	MockActiveCluster    *config.Cluster
}

func (m *MockConfigManager) ReadConfig(configFile string) (config.Config, error) {
	if m.ShouldFailReadConfig {
		return config.Config{}, errors.New("mock config read failed")
	}
	return m.MockConfig, nil
}

func (m *MockConfigManager) GetActiveCluster(cfg config.Config) *config.Cluster {
	return m.MockActiveCluster
}

// MockClusterAdmin for testing - implements ClusterAdminInterface
type MockClusterAdmin struct {
	ShouldFailListTopics         bool
	ShouldFailListConsumerGroups bool
	ShouldFailDescribeGroups     bool
	MockTopics                   map[string]sarama.TopicDetail
	MockConsumerGroups           map[string]string
	MockGroupDescriptions        []*sarama.GroupDescription
}

func (m *MockClusterAdmin) ListTopics() (map[string]sarama.TopicDetail, error) {
	if m.ShouldFailListTopics {
		return nil, errors.New("mock list topics failed")
	}
	return m.MockTopics, nil
}

func (m *MockClusterAdmin) ListConsumerGroups() (map[string]string, error) {
	if m.ShouldFailListConsumerGroups {
		return nil, errors.New("mock list consumer groups failed")
	}
	return m.MockConsumerGroups, nil
}

func (m *MockClusterAdmin) DescribeConsumerGroups(groups []string) ([]*sarama.GroupDescription, error) {
	if m.ShouldFailDescribeGroups {
		return nil, errors.New("mock describe consumer groups failed")
	}
	return m.MockGroupDescriptions, nil
}

func (m *MockClusterAdmin) Close() error {
	return nil
}