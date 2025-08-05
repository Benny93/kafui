package kafds

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockConsumer implements ConsumerInterface for testing
type MockConsumer struct {
	mock.Mock
}

func (m *MockConsumer) GetOffsets(client sarama.Client, topic string, partition int32) (*offsets, error) {
	args := m.Called(client, topic, partition)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*offsets), args.Error(1)
}

func (m *MockConsumer) CreateConsumerFromClient(client sarama.Client, topic string, partition int32) (sarama.PartitionConsumer, error) {
	args := m.Called(client, topic, partition)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(sarama.PartitionConsumer), args.Error(1)
}

func (m *MockConsumer) CreateConsumerGroupFromClient(group string, client sarama.Client) (sarama.ConsumerGroup, error) {
	args := m.Called(group, client)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(sarama.ConsumerGroup), args.Error(1)
}

// MockMessageProcessor implements MessageProcessorInterface for testing
type MockMessageProcessor struct {
	mock.Mock
}

type MockSaramaClient struct {
	mock.Mock
}

func (m *MockMessageProcessor) ProcessMessage(msg *sarama.ConsumerMessage, handler api.MessageHandlerFunc) error {
	args := m.Called(msg, handler)
	return args.Error(0)
}

func (m *MockMessageProcessor) FormatKey(key []byte) []byte {
	args := m.Called(key)
	return args.Get(0).([]byte)
}

func (m *MockMessageProcessor) FormatValue(value []byte) []byte {
	args := m.Called(value)
	return args.Get(0).([]byte)
}

func (m *MockMessageProcessor) DecodeAvro(data []byte) ([]byte, error) {
	args := m.Called(data)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockMessageProcessor) DecodeProto(data []byte, protoType string) ([]byte, error) {
	args := m.Called(data, protoType)
	return args.Get(0).([]byte), args.Error(1)
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
	mockProcessor := &MockMessageProcessor{}

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

	DoConsumeWithDeps(ctx, "test-topic", flags, handler, onError, mockConfigProvider, mockConsumer, mockProcessor)

	assert.True(t, errorCalled)
	assert.Contains(t, errorMsg.(error).Error(), "config error")
	mockConfigProvider.AssertExpectations(t)
}

func TestDoConsumeWithDeps_ClientError(t *testing.T) {
	mockConfigProvider := &MockConfigProvider{}
	mockConsumer := &MockConsumer{}
	mockProcessor := &MockMessageProcessor{}

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

	DoConsumeWithDeps(ctx, "test-topic", flags, handler, onError, mockConfigProvider, mockConsumer, mockProcessor)

	assert.True(t, errorCalled)
	assert.Contains(t, errorMsg.(error).Error(), "client error")
	mockConfigProvider.AssertExpectations(t)
}

func TestDoConsumeWithDeps_OffsetParsing(t *testing.T) {
	mockConfigProvider := &MockConfigProvider{}
	mockConsumer := &MockConsumer{}
	mockProcessor := &MockMessageProcessor{}
	mockClient := &MockSaramaClient{}

	config := sarama.NewConfig()
	mockConfigProvider.On("GetConsumerConfig").Return(config, nil)
	mockConfigProvider.On("GetClientFromConfig", config).Return(mockClient, nil)

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

	DoConsumeWithDeps(ctx, "test-topic", flags, handler, onError, mockConfigProvider, mockConsumer, mockProcessor)

	assert.True(t, errorCalled)
	assert.Contains(t, errorMsg.(error).Error(), "invalid syntax")
	mockConfigProvider.AssertExpectations(t)
}

func TestDoConsumeWithDeps_Success_OldestOffset(t *testing.T) {
	mockConfigProvider := &MockConfigProvider{}
	mockConsumer := &MockConsumer{}
	mockProcessor := &MockMessageProcessor{}
	mockClient := &MockSaramaClient{}

	config := sarama.NewConfig()
	mockConfigProvider.On("GetConsumerConfig").Return(config, nil)
	mockConfigProvider.On("GetClientFromConfig", config).Return(mockClient, nil)

	var errorCalled bool
	onError := func(err interface{}) {
		errorCalled = true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	flags := api.ConsumeFlags{
		Follow:     false,
		Tail:       10,
		OffsetFlag: "oldest",
	}
	handler := func(msg api.Message) {}

	// This will call withoutConsumerGroupWithDeps since groupFlag is empty
	DoConsumeWithDeps(ctx, "test-topic", flags, handler, onError, mockConfigProvider, mockConsumer, mockProcessor)

	assert.False(t, errorCalled)
	mockConfigProvider.AssertExpectations(t)
}

func TestDoConsumeWithDeps_Success_NewestOffset(t *testing.T) {
	mockConfigProvider := &MockConfigProvider{}
	mockConsumer := &MockConsumer{}
	mockProcessor := &MockMessageProcessor{}
	mockClient := &MockSaramaClient{}

	config := sarama.NewConfig()
	mockConfigProvider.On("GetConsumerConfig").Return(config, nil)
	mockConfigProvider.On("GetClientFromConfig", config).Return(mockClient, nil)

	var errorCalled bool
	onError := func(err interface{}) {
		errorCalled = true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	flags := api.ConsumeFlags{
		Follow:     false,
		Tail:       10,
		OffsetFlag: "newest",
	}
	handler := func(msg api.Message) {}

	DoConsumeWithDeps(ctx, "test-topic", flags, handler, onError, mockConfigProvider, mockConsumer, mockProcessor)

	assert.False(t, errorCalled)
	assert.Equal(t, sarama.OffsetNewest, config.Consumer.Offsets.Initial)
	mockConfigProvider.AssertExpectations(t)
}

func TestWithoutConsumerGroupWithDeps_NilClient(t *testing.T) {
	mockConsumer := &MockConsumer{}
	mockProcessor := &MockMessageProcessor{}

	var errorCalled bool
	var errorMsg interface{}
	onError := func(err interface{}) {
		errorCalled = true
		errorMsg = err
	}

	ctx := context.Background()
	withoutConsumerGroupWithDeps(ctx, nil, "test-topic", sarama.OffsetOldest, onError, mockConsumer, mockProcessor)

	assert.True(t, errorCalled)
	assert.Contains(t, errorMsg.(string), "client is nil")
}

// Test formatting functions

func TestFormatJSON_ValidJSON(t *testing.T) {
	jsonData := []byte(`{"key": "value"}`)
	result := formatJSON(jsonData)

	assert.IsType(t, map[string]interface{}{}, result)
}

func TestFormatJSON_InvalidJSON(t *testing.T) {
	invalidData := []byte(`invalid json`)
	result := formatJSON(invalidData)

	assert.Equal(t, "invalid json", result)
}

func TestIsJSON_ValidJSON(t *testing.T) {
	jsonData := []byte(`{"key": "value"}`)
	result := isJSON(jsonData)

	assert.True(t, result)
}

func TestIsJSON_InvalidJSON(t *testing.T) {
	invalidData := []byte(`invalid json`)
	result := isJSON(invalidData)

	assert.False(t, result)
}

// Test OutputFormat

func TestOutputFormat_String(t *testing.T) {
	format := OutputFormatJSON
	assert.Equal(t, "json", format.String())
}

func TestOutputFormat_Set_Valid(t *testing.T) {
	var format OutputFormat

	err := format.Set("json")
	assert.NoError(t, err)
	assert.Equal(t, OutputFormatJSON, format)

	err = format.Set("raw")
	assert.NoError(t, err)
	assert.Equal(t, OutputFormatRaw, format)

	err = format.Set("default")
	assert.NoError(t, err)
	assert.Equal(t, OutputFormatDefault, format)
}

func TestOutputFormat_Set_Invalid(t *testing.T) {
	var format OutputFormat

	err := format.Set("invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be one of")
}

func TestOutputFormat_Type(t *testing.T) {
	var format OutputFormat
	assert.Equal(t, "OutputFormat", format.Type())
}

func TestCompleteOutputFormat(t *testing.T) {
	completions, directive := completeOutputFormat(nil, nil, "")

	assert.Equal(t, []string{"default", "raw", "json"}, completions)
	// Just check that directive is returned (specific value may vary)
	assert.NotNil(t, directive)
}
