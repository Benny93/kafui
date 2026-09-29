package kafds

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockConsumer implements ConsumerInterface for testing
type MockConsumer struct {
	mock.Mock
}

func (m *MockConsumer) CreateConsumerGroupFromClient(group string, client sarama.Client) (sarama.ConsumerGroup, error) {
	args := m.Called(group, client)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(sarama.ConsumerGroup), args.Error(1)
}

type MockSaramaClient struct {
	mock.Mock
}

// Implement all sarama.Client interface methods
func (m *MockSaramaClient) Config() *sarama.Config {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*sarama.Config)
}

func (m *MockSaramaClient) Controller() (*sarama.Broker, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sarama.Broker), args.Error(1)
}

func (m *MockSaramaClient) RefreshController() (*sarama.Broker, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sarama.Broker), args.Error(1)
}

func (m *MockSaramaClient) Brokers() []*sarama.Broker {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]*sarama.Broker)
}

func (m *MockSaramaClient) Broker(brokerID int32) (*sarama.Broker, error) {
	args := m.Called(brokerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sarama.Broker), args.Error(1)
}

func (m *MockSaramaClient) Topics() ([]string, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockSaramaClient) Partitions(topic string) ([]int32, error) {
	args := m.Called(topic)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]int32), args.Error(1)
}

func (m *MockSaramaClient) WritablePartitions(topic string) ([]int32, error) {
	args := m.Called(topic)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]int32), args.Error(1)
}

func (m *MockSaramaClient) Leader(topic string, partitionID int32) (*sarama.Broker, error) {
	args := m.Called(topic, partitionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sarama.Broker), args.Error(1)
}

func (m *MockSaramaClient) LeaderAndEpoch(topic string, partitionID int32) (*sarama.Broker, int32, error) {
	args := m.Called(topic, partitionID)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).(*sarama.Broker), int32(args.Int(1)), args.Error(2)
}

func (m *MockSaramaClient) Replicas(topic string, partitionID int32) ([]int32, error) {
	args := m.Called(topic, partitionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]int32), args.Error(1)
}

func (m *MockSaramaClient) InSyncReplicas(topic string, partitionID int32) ([]int32, error) {
	args := m.Called(topic, partitionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]int32), args.Error(1)
}

func (m *MockSaramaClient) OfflineReplicas(topic string, partitionID int32) ([]int32, error) {
	args := m.Called(topic, partitionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]int32), args.Error(1)
}

func (m *MockSaramaClient) RefreshBrokers(addrs []string) error {
	args := m.Called(addrs)
	return args.Error(0)
}

func (m *MockSaramaClient) RefreshMetadata(topics ...string) error {
	args := m.Called(topics)
	return args.Error(0)
}

func (m *MockSaramaClient) GetOffset(topic string, partitionID int32, time int64) (int64, error) {
	args := m.Called(topic, partitionID, time)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSaramaClient) Coordinator(consumerGroup string) (*sarama.Broker, error) {
	args := m.Called(consumerGroup)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sarama.Broker), args.Error(1)
}

func (m *MockSaramaClient) RefreshCoordinator(consumerGroup string) error {
	args := m.Called(consumerGroup)
	return args.Error(0)
}

func (m *MockSaramaClient) TransactionCoordinator(transactionID string) (*sarama.Broker, error) {
	args := m.Called(transactionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sarama.Broker), args.Error(1)
}

func (m *MockSaramaClient) RefreshTransactionCoordinator(transactionID string) error {
	args := m.Called(transactionID)
	return args.Error(0)
}

func (m *MockSaramaClient) InitProducerID() (*sarama.InitProducerIDResponse, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sarama.InitProducerIDResponse), args.Error(1)
}

func (m *MockSaramaClient) LeastLoadedBroker() *sarama.Broker {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*sarama.Broker)
}

func (m *MockSaramaClient) PartitionNotReadable(topic string, partition int32) bool {
	args := m.Called(topic, partition)
	return args.Bool(0)
}

func (m *MockSaramaClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSaramaClient) Closed() bool {
	args := m.Called()
	return args.Bool(0)
}

// MockConfigProvider implements ConfigProviderInterface for testing
type MockConfigProvider struct {
	mock.Mock
}

func (m *MockConfigProvider) GetConsumerConfig() (*sarama.Config, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sarama.Config), args.Error(1)
}

func (m *MockConfigProvider) GetClientFromConfig(config *sarama.Config) (sarama.Client, error) {
	args := m.Called(config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(sarama.Client), args.Error(1)
}

// Use the existing MockClient from consume_test.go instead of creating a new one

// MockConsumerGroup for testing
type MockConsumerGroup struct {
	mock.Mock
}

func (m *MockConsumerGroup) Consume(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
	args := m.Called(ctx, topics, handler)
	return args.Error(0)
}

func (m *MockConsumerGroup) Errors() <-chan error {
	args := m.Called()
	return args.Get(0).(<-chan error)
}

func (m *MockConsumerGroup) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockConsumerGroup) Pause(partitions map[string][]int32) {
	m.Called(partitions)
}

func (m *MockConsumerGroup) Resume(partitions map[string][]int32) {
	m.Called(partitions)
}

func (m *MockConsumerGroup) PauseAll() {
	m.Called()
}

func (m *MockConsumerGroup) ResumeAll() {
	m.Called()
}

// Test functions

func TestDoConsumeWithDeps_ConfigError(t *testing.T) {
	mockConfigProvider := &MockConfigProvider{}
	mockConsumer := &MockConsumer{}

	mockConfigProvider.On("GetConsumerConfig").Return(nil, errors.New("config error"))

	var errorCalled bool
	var errorMsg interface{}
	onError := func(err interface{}) {
		errorCalled = true
		errorMsg = err
	}

	ctx := context.Background()
	flags := api.ConsumeFlags{Follow: true, Tail: 10}
	handler := func(msg api.Message) {}

	DoConsumeWithDeps(ctx, "test-topic", flags, handler, onError, mockConfigProvider, mockConsumer)

	assert.True(t, errorCalled)
	assert.Contains(t, errorMsg.(error).Error(), "config error")
	mockConfigProvider.AssertExpectations(t)
}

func TestDoConsumeWithDeps_ClientError(t *testing.T) {
	mockConfigProvider := &MockConfigProvider{}
	mockConsumer := &MockConsumer{}

	config := sarama.NewConfig()
	mockConfigProvider.On("GetConsumerConfig").Return(config, nil)
	mockConfigProvider.On("GetClientFromConfig", config).Return(nil, errors.New("client error"))

	var errorCalled bool
	var errorMsg interface{}
	onError := func(err interface{}) {
		errorCalled = true
		errorMsg = err
	}

	ctx := context.Background()
	flags := api.ConsumeFlags{Follow: true, Tail: 10}
	handler := func(msg api.Message) {}

	DoConsumeWithDeps(ctx, "test-topic", flags, handler, onError, mockConfigProvider, mockConsumer)

	assert.True(t, errorCalled)
	assert.Contains(t, errorMsg.(error).Error(), "client error")
	mockConfigProvider.AssertExpectations(t)
}

func TestDoConsumeWithDeps_OffsetParsing(t *testing.T) {
	mockConfigProvider := &MockConfigProvider{}
	mockConsumer := &MockConsumer{}
	mockClient := &MockSaramaClient{}

	config := sarama.NewConfig()
	mockConfigProvider.On("GetConsumerConfig").Return(config, nil)
	mockConfigProvider.On("GetClientFromConfig", config).Return(mockClient, nil)
	mockClient.On("Close").Return(nil)

	var errorCalled bool
	var errorMsg interface{}
	onError := func(err interface{}) {
		errorCalled = true
		errorMsg = err
	}

	ctx := context.Background()
	flags := api.ConsumeFlags{
		Follow:     true,
		Tail:       10,
		OffsetFlag: "invalid-offset",
	}
	handler := func(msg api.Message) {}

	DoConsumeWithDeps(ctx, "test-topic", flags, handler, onError, mockConfigProvider, mockConsumer)

	assert.True(t, errorCalled)
	assert.Contains(t, errorMsg.(error).Error(), "invalid syntax")
	mockConfigProvider.AssertExpectations(t)
	// The client is closed even when consumption bails out early.
	mockClient.AssertCalled(t, "Close")
}

// brokerConfigProvider builds real sarama clients against a mock broker and
// remembers the last one so a test can check it was closed.
type brokerConfigProvider struct {
	addr   string
	client sarama.Client
}

func (p *brokerConfigProvider) GetConsumerConfig() (*sarama.Config, error) {
	cfg := sarama.NewConfig()
	cfg.Metadata.Retry.Max = 0
	return cfg, nil
}

func (p *brokerConfigProvider) GetClientFromConfig(cfg *sarama.Config) (sarama.Client, error) {
	c, err := sarama.NewClient([]string{p.addr}, cfg)
	p.client = c
	return c, err
}

// Regression (CONC-1/DS-2): a completed fetch must close the client it
// created instead of leaking it with its broker connections.
func TestDoConsumeWithConfig_ClosesClientAfterFetch(t *testing.T) {
	broker := sarama.NewMockBroker(t, 1)
	defer broker.Close()
	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(broker.Addr(), broker.BrokerID()).
			SetLeader("t", 0, broker.BrokerID()),
		"OffsetRequest": sarama.NewMockOffsetResponse(t).
			SetOffset("t", 0, sarama.OffsetOldest, 0).
			SetOffset("t", 0, sarama.OffsetNewest, 2),
		"FetchRequest": sarama.NewMockFetchResponse(t, 1).
			SetMessage("t", 0, 0, sarama.StringEncoder("a")).
			SetMessage("t", 0, 1, sarama.StringEncoder("b")).
			SetHighWaterMark("t", 0, 2),
	})

	provider := &brokerConfigProvider{addr: broker.Addr()}
	var (
		mu   sync.Mutex
		got  []string
		errs []any
	)
	handle := func(m api.Message) {
		mu.Lock()
		got = append(got, m.Value)
		mu.Unlock()
	}
	onError := func(err any) {
		mu.Lock()
		errs = append(errs, err)
		mu.Unlock()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	DoConsumeWithConfig(ctx, "t", api.ConsumeFlags{OffsetFlag: "oldest"}, handle, onError,
		provider, &MockConsumer{}, DefaultConsumeConfig())

	mu.Lock()
	defer mu.Unlock()
	assert.Empty(t, errs)
	assert.Equal(t, []string{"a", "b"}, got)
	require.NotNil(t, provider.client)
	assert.True(t, provider.client.Closed(), "client must be closed once the fetch completes")
}

// Regression (DS-11): the consumer-group path reports errors, rejoins after a
// rebalance until ctx is done, and always closes the group.
func TestWithConsumerGroupWithDeps(t *testing.T) {
	t.Run("consume error is returned and group closed", func(t *testing.T) {
		cg := &MockConsumerGroup{}
		cg.On("Consume", mock.Anything, []string{"t"}, mock.Anything).Return(errors.New("boom"))
		cg.On("Close").Return(nil)
		consumer := &MockConsumer{}
		consumer.On("CreateConsumerGroupFromClient", "g", mock.Anything).Return(cg, nil)

		err := withConsumerGroupWithDeps(context.Background(), &MockClient{}, "t", "g", consumer, DefaultConsumeConfig(), nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "boom")
		cg.AssertCalled(t, "Close")
	})

	t.Run("rejoins after rebalance until ctx is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls := 0
		cg := &MockConsumerGroup{}
		cg.On("Consume", mock.Anything, []string{"t"}, mock.Anything).Return(nil).Run(func(mock.Arguments) {
			calls++
			if calls == 2 {
				cancel()
			}
		})
		cg.On("Close").Return(nil)
		consumer := &MockConsumer{}
		consumer.On("CreateConsumerGroupFromClient", "g", mock.Anything).Return(cg, nil)

		err := withConsumerGroupWithDeps(ctx, &MockClient{}, "t", "g", consumer, DefaultConsumeConfig(), nil)

		require.NoError(t, err)
		assert.Equal(t, 2, calls)
		cg.AssertCalled(t, "Close")
	})

	t.Run("DoConsume forwards group errors to onError", func(t *testing.T) {
		provider := &MockConfigProvider{}
		client := &MockSaramaClient{}
		cfg := sarama.NewConfig()
		provider.On("GetConsumerConfig").Return(cfg, nil)
		provider.On("GetClientFromConfig", cfg).Return(client, nil)
		client.On("Close").Return(nil)
		consumer := &MockConsumer{}
		consumer.On("CreateConsumerGroupFromClient", "g", client).Return(nil, errors.New("no group"))

		var got any
		DoConsumeWithDeps(context.Background(), "t", api.ConsumeFlags{GroupFlag: "g"}, func(api.Message) {},
			func(err any) { got = err }, provider, consumer)

		require.NotNil(t, got)
		assert.Contains(t, got.(error).Error(), "no group")
		client.AssertCalled(t, "Close")
	})
}

func TestDoConsumeWithDeps_Success_OldestOffset(t *testing.T) {
	mockConfigProvider := &MockConfigProvider{}
	mockConsumer := &MockConsumer{}
	mockClient := &MockSaramaClient{}

	config := sarama.NewConfig()
	mockConfigProvider.On("GetConsumerConfig").Return(config, nil)
	mockConfigProvider.On("GetClientFromConfig", config).Return(mockClient, nil)
	
	// Add mock expectations for methods that will be called
	mockClient.On("Closed").Return(false)
	mockClient.On("Close").Return(nil)
	mockClient.On("Config").Return(config)
	// Mock GetOffset calls for getting newest and oldest offsets
	mockClient.On("GetOffset", "test-topic", int32(0), int64(-1)).Return(int64(100), nil)  // newest
	mockClient.On("GetOffset", "test-topic", int32(0), int64(-2)).Return(int64(0), nil)   // oldest
	// Mock LeaderAndEpoch for partition consumer creation - return nil broker to avoid actual consumer creation
	mockClient.On("LeaderAndEpoch", "test-topic", int32(0)).Return((*sarama.Broker)(nil), int32(0), nil)

	var errorCalled bool
	var errorMsg interface{}
	onError := func(err interface{}) {
		errorCalled = true
		errorMsg = err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	flags := api.ConsumeFlags{
		Follow:     false,
		Tail:       10,
		OffsetFlag: "oldest",
	}
	handler := func(msg api.Message) {}

	// Add the missing Partitions expectation for the first test
	mockClient.On("Partitions", "test-topic").Return([]int32{0}, nil)

	// This will call withoutConsumerGroupWithDeps since groupFlag is empty
	DoConsumeWithDeps(ctx, "test-topic", flags, handler, onError, mockConfigProvider, mockConsumer)

	if errorCalled {
		t.Logf("Error occurred: %v", errorMsg)
	}
	// Since we're mocking and the consumer creation will fail, we expect an error
	assert.True(t, errorCalled)
	mockConfigProvider.AssertExpectations(t)
}

func TestDoConsumeWithDeps_Success_NewestOffset(t *testing.T) {
	mockConfigProvider := &MockConfigProvider{}
	mockConsumer := &MockConsumer{}
	mockClient := &MockSaramaClient{}

	config := sarama.NewConfig()
	mockConfigProvider.On("GetConsumerConfig").Return(config, nil)
	mockConfigProvider.On("GetClientFromConfig", config).Return(mockClient, nil)
	
	// Add mock expectations for methods that will be called
	mockClient.On("Closed").Return(false)
	mockClient.On("Close").Return(nil)
	mockClient.On("Config").Return(config)
	mockClient.On("Partitions", "test-topic").Return([]int32{0}, nil)
	// Mock GetOffset calls for getting newest and oldest offsets
	mockClient.On("GetOffset", "test-topic", int32(0), int64(-1)).Return(int64(100), nil)  // newest
	mockClient.On("GetOffset", "test-topic", int32(0), int64(-2)).Return(int64(0), nil)   // oldest
	// Mock LeaderAndEpoch for partition consumer creation - return nil broker to avoid actual consumer creation
	mockClient.On("LeaderAndEpoch", "test-topic", int32(0)).Return((*sarama.Broker)(nil), int32(0), nil)

	var errorCalled bool
	var errorMsg interface{}
	onError := func(err interface{}) {
		errorCalled = true
		errorMsg = err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	flags := api.ConsumeFlags{
		Follow:     false,
		Tail:       10,
		OffsetFlag: "newest",
	}
	handler := func(msg api.Message) {}

	DoConsumeWithDeps(ctx, "test-topic", flags, handler, onError, mockConfigProvider, mockConsumer)

	if errorCalled {
		t.Logf("Error occurred: %v", errorMsg)
	}
	// Since we're mocking and the consumer creation will fail, we expect an error
	assert.True(t, errorCalled)
	assert.Equal(t, sarama.OffsetNewest, config.Consumer.Offsets.Initial)
	mockConfigProvider.AssertExpectations(t)
}

func TestWithoutConsumerGroupWithDeps_NilClient(t *testing.T) {
	var errorCalled bool
	var errorMsg interface{}
	onError := func(err interface{}) {
		errorCalled = true
		errorMsg = err
	}

	ctx := context.Background()
	config := DefaultConsumeConfig()
	withoutConsumerGroupWithDeps(ctx, nil, "test-topic", sarama.OffsetOldest, onError, config, nil)

	assert.True(t, errorCalled)
	assert.Contains(t, errorMsg.(string), "client is nil")
}
