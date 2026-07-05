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
	ReadConfigCallCount  int
}

func (m *MockConfigManager) ReadConfig(configFile string) (config.Config, error) {
	m.ReadConfigCallCount++
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

	// Broker-management fields (BR-2..BR-7).
	MockBrokers               []*sarama.Broker
	MockControllerID          int32
	ShouldFailDescribeCluster bool
	MockConfigEntries         []sarama.ConfigEntry
	ShouldFailDescribeConfig  bool
	MockLogDirs               map[int32][]sarama.DescribeLogDirsResponseDirMetadata
	ShouldFailDescribeLogDirs bool
	MockTopicMetadata         []*sarama.TopicMetadata
	ShouldFailDescribeTopics  bool
	// AlterConfigErr, when set, is returned from IncrementalAlterConfig.
	AlterConfigErr error
	// IncrementalAlterConfigCalls records (name, key, value) of each SET.
	IncrementalAlterConfigCalls []AlterConfigCall

	// Consumer-group offset/mutation fields (CG-2..CG-8).
	MockGroupOffsets              *sarama.OffsetFetchResponse
	ShouldFailListGroupOffsets    bool
	DeleteConsumerGroupErr        error
	DeleteConsumerGroupCalls      []string
	DeleteConsumerGroupOffsetErr  error
	DeleteConsumerGroupOffsetCall []DeleteOffsetCall

	// Topic-administration fields (TP-1..TP-11).
	CreateTopicErr         error
	CreateTopicCalls       []CreateTopicCall
	DeleteTopicErr         error
	DeleteTopicCalls       []string
	CreatePartitionsErr    error
	CreatePartitionsCalls  []CreatePartitionsCall
	DeleteRecordsErr       error
	DeleteRecordsCalls     []DeleteRecordsCall
	AlterReassignmentErr   error
	AlterReassignmentCalls []ReassignmentCall

	// ACL write fields (AQ-5).
	MockAcls         []sarama.ResourceAcls
	ListAclsErr      error
	CreateACLsErr    error
	CreateACLsCalls  [][]*sarama.ResourceAcls
	DeleteACLErr     error
	DeleteACLCalls   []sarama.AclFilter
	MockMatchingAcls []sarama.MatchingAcl

	// Client-quota fields (AQ-11).
	MockQuotas             []sarama.DescribeClientQuotasEntry
	DescribeQuotasErr      error
	DescribeQuotasCalls    []DescribeQuotasCall
	AlterQuotasErr         error
	AlterClientQuotasCalls []AlterQuotaCall
}

// DescribeQuotasCall captures the arguments of a DescribeClientQuotas call.
type DescribeQuotasCall struct {
	Components []sarama.QuotaFilterComponent
	Strict     bool
}

// AlterQuotaCall captures the arguments of an AlterClientQuotas call.
type AlterQuotaCall struct {
	Entity []sarama.QuotaEntityComponent
	Op     sarama.ClientQuotasOp
}

// CreateTopicCall captures the arguments of a CreateTopic invocation.
type CreateTopicCall struct {
	Topic        string
	Detail       *sarama.TopicDetail
	ValidateOnly bool
}

// CreatePartitionsCall captures the arguments of a CreatePartitions invocation.
type CreatePartitionsCall struct {
	Topic      string
	Count      int32
	Assignment [][]int32
}

// DeleteRecordsCall captures the arguments of a DeleteRecords invocation.
type DeleteRecordsCall struct {
	Topic            string
	PartitionOffsets map[int32]int64
}

// ReassignmentCall captures the arguments of an AlterPartitionReassignments call.
type ReassignmentCall struct {
	Topic      string
	Assignment [][]int32
}

// DeleteOffsetCall captures the arguments of a DeleteConsumerGroupOffset call.
type DeleteOffsetCall struct {
	Group     string
	Topic     string
	Partition int32
}

// AlterConfigCall captures the arguments of an IncrementalAlterConfig invocation.
type AlterConfigCall struct {
	Name  string
	Key   string
	Value string
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

func (m *MockClusterAdmin) ListAcls(filter sarama.AclFilter) ([]sarama.ResourceAcls, error) {
	if m.ListAclsErr != nil {
		return nil, m.ListAclsErr
	}
	return m.MockAcls, nil
}

func (m *MockClusterAdmin) CreateACLs(resourceACLs []*sarama.ResourceAcls) error {
	m.CreateACLsCalls = append(m.CreateACLsCalls, resourceACLs)
	return m.CreateACLsErr
}

func (m *MockClusterAdmin) DeleteACL(filter sarama.AclFilter, validateOnly bool) ([]sarama.MatchingAcl, error) {
	m.DeleteACLCalls = append(m.DeleteACLCalls, filter)
	if m.DeleteACLErr != nil {
		return nil, m.DeleteACLErr
	}
	return m.MockMatchingAcls, nil
}

func (m *MockClusterAdmin) DescribeClientQuotas(components []sarama.QuotaFilterComponent, strict bool) ([]sarama.DescribeClientQuotasEntry, error) {
	m.DescribeQuotasCalls = append(m.DescribeQuotasCalls, DescribeQuotasCall{Components: components, Strict: strict})
	if m.DescribeQuotasErr != nil {
		return nil, m.DescribeQuotasErr
	}
	return m.MockQuotas, nil
}

func (m *MockClusterAdmin) AlterClientQuotas(entity []sarama.QuotaEntityComponent, op sarama.ClientQuotasOp, validateOnly bool) error {
	m.AlterClientQuotasCalls = append(m.AlterClientQuotasCalls, AlterQuotaCall{Entity: entity, Op: op})
	return m.AlterQuotasErr
}

func (m *MockClusterAdmin) DescribeCluster() ([]*sarama.Broker, int32, error) {
	if m.ShouldFailDescribeCluster {
		return nil, 0, errors.New("mock describe cluster failed")
	}
	return m.MockBrokers, m.MockControllerID, nil
}

func (m *MockClusterAdmin) DescribeConfig(resource sarama.ConfigResource) ([]sarama.ConfigEntry, error) {
	if m.ShouldFailDescribeConfig {
		return nil, errors.New("mock describe config failed")
	}
	return m.MockConfigEntries, nil
}

func (m *MockClusterAdmin) IncrementalAlterConfig(resourceType sarama.ConfigResourceType, name string, entries map[string]sarama.IncrementalAlterConfigsEntry, validateOnly bool) error {
	for key, entry := range entries {
		value := ""
		if entry.Value != nil {
			value = *entry.Value
		}
		m.IncrementalAlterConfigCalls = append(m.IncrementalAlterConfigCalls, AlterConfigCall{Name: name, Key: key, Value: value})
	}
	return m.AlterConfigErr
}

func (m *MockClusterAdmin) DescribeLogDirs(brokers []int32) (map[int32][]sarama.DescribeLogDirsResponseDirMetadata, error) {
	if m.ShouldFailDescribeLogDirs {
		return nil, errors.New("mock describe log dirs failed")
	}
	return m.MockLogDirs, nil
}

func (m *MockClusterAdmin) DescribeTopics(topics []string) ([]*sarama.TopicMetadata, error) {
	if m.ShouldFailDescribeTopics {
		return nil, errors.New("mock describe topics failed")
	}
	return m.MockTopicMetadata, nil
}

func (m *MockClusterAdmin) ListConsumerGroupOffsets(group string, topicPartitions map[string][]int32) (*sarama.OffsetFetchResponse, error) {
	if m.ShouldFailListGroupOffsets {
		return nil, errors.New("mock list consumer group offsets failed")
	}
	if m.MockGroupOffsets != nil {
		return m.MockGroupOffsets, nil
	}
	return &sarama.OffsetFetchResponse{Blocks: map[string]map[int32]*sarama.OffsetFetchResponseBlock{}}, nil
}

func (m *MockClusterAdmin) DeleteConsumerGroup(group string) error {
	m.DeleteConsumerGroupCalls = append(m.DeleteConsumerGroupCalls, group)
	return m.DeleteConsumerGroupErr
}

func (m *MockClusterAdmin) DeleteConsumerGroupOffset(group string, topic string, partition int32) error {
	m.DeleteConsumerGroupOffsetCall = append(m.DeleteConsumerGroupOffsetCall, DeleteOffsetCall{Group: group, Topic: topic, Partition: partition})
	return m.DeleteConsumerGroupOffsetErr
}

func (m *MockClusterAdmin) CreateTopic(topic string, detail *sarama.TopicDetail, validateOnly bool) error {
	m.CreateTopicCalls = append(m.CreateTopicCalls, CreateTopicCall{Topic: topic, Detail: detail, ValidateOnly: validateOnly})
	return m.CreateTopicErr
}

func (m *MockClusterAdmin) DeleteTopic(topic string) error {
	m.DeleteTopicCalls = append(m.DeleteTopicCalls, topic)
	return m.DeleteTopicErr
}

func (m *MockClusterAdmin) CreatePartitions(topic string, count int32, assignment [][]int32, validateOnly bool) error {
	m.CreatePartitionsCalls = append(m.CreatePartitionsCalls, CreatePartitionsCall{Topic: topic, Count: count, Assignment: assignment})
	return m.CreatePartitionsErr
}

func (m *MockClusterAdmin) DeleteRecords(topic string, partitionOffsets map[int32]int64) error {
	m.DeleteRecordsCalls = append(m.DeleteRecordsCalls, DeleteRecordsCall{Topic: topic, PartitionOffsets: partitionOffsets})
	return m.DeleteRecordsErr
}

func (m *MockClusterAdmin) AlterPartitionReassignments(topic string, assignment [][]int32) error {
	m.AlterReassignmentCalls = append(m.AlterReassignmentCalls, ReassignmentCall{Topic: topic, Assignment: assignment})
	return m.AlterReassignmentErr
}

func (m *MockClusterAdmin) Close() error {
	return nil
}
