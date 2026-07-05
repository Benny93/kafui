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
	ListAcls(filter sarama.AclFilter) ([]sarama.ResourceAcls, error)
	// CreateACLs creates one or more ACL bindings.
	CreateACLs(resourceACLs []*sarama.ResourceAcls) error
	// DeleteACL deletes ACLs matching the filter, returning the bindings that
	// were removed (empty when nothing matched).
	DeleteACL(filter sarama.AclFilter, validateOnly bool) ([]sarama.MatchingAcl, error)
	// DescribeClientQuotas returns the client quotas matching the components.
	DescribeClientQuotas(components []sarama.QuotaFilterComponent, strict bool) ([]sarama.DescribeClientQuotasEntry, error)
	// AlterClientQuotas applies a single set/remove op to the entity's quotas.
	AlterClientQuotas(entity []sarama.QuotaEntityComponent, op sarama.ClientQuotasOp, validateOnly bool) error
	// DescribeCluster returns the online brokers and the active controller ID.
	DescribeCluster() (brokers []*sarama.Broker, controllerID int32, err error)
	// DescribeConfig returns the config entries for a resource (e.g. a broker).
	DescribeConfig(resource sarama.ConfigResource) ([]sarama.ConfigEntry, error)
	// IncrementalAlterConfig incrementally updates config entries, preserving
	// other dynamic configs. sarama v1.45.1's ClusterAdmin implements this
	// natively, so no AlterConfig fallback is required.
	IncrementalAlterConfig(resourceType sarama.ConfigResourceType, name string, entries map[string]sarama.IncrementalAlterConfigsEntry, validateOnly bool) error
	// DescribeLogDirs returns log-directory metadata for the given broker IDs.
	DescribeLogDirs(brokers []int32) (map[int32][]sarama.DescribeLogDirsResponseDirMetadata, error)
	// DescribeTopics returns full partition metadata for the given topics.
	DescribeTopics(topics []string) ([]*sarama.TopicMetadata, error)
	// ListConsumerGroupOffsets returns the committed offsets of a group. Pass a
	// nil topicPartitions map to fetch all committed offsets for the group.
	ListConsumerGroupOffsets(group string, topicPartitions map[string][]int32) (*sarama.OffsetFetchResponse, error)
	// DeleteConsumerGroup deletes a consumer group.
	DeleteConsumerGroup(group string) error
	// DeleteConsumerGroupOffset deletes the committed offset of a single
	// group/topic/partition.
	DeleteConsumerGroupOffset(group string, topic string, partition int32) error

	// --- Topic administration (TP-1). All pass-throughs on sarama.ClusterAdmin. ---
	CreateTopic(topic string, detail *sarama.TopicDetail, validateOnly bool) error
	DeleteTopic(topic string) error
	CreatePartitions(topic string, count int32, assignment [][]int32, validateOnly bool) error
	DeleteRecords(topic string, partitionOffsets map[int32]int64) error
	AlterPartitionReassignments(topic string, assignment [][]int32) error

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
