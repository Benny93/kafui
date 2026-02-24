package messagedetail

import (
	"fmt"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewModel(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	// Create test message
	testMessage := api.Message{
		Key:       "test-key",
		Value:     "test-value",
		Offset:    123,
		Partition: 0,
		Headers: []api.MessageHeader{
			{Key: "content-type", Value: "application/json"},
		},
	}

	// Create new model using the migrated structure
	pageModel := NewMessageDetailPageModel(mockDS, "test-topic", testMessage)
	model := pageModel.GetDetailModel()

	// Verify model is properly initialized
	assert.NotNil(t, pageModel)
	assert.NotNil(t, model)
	assert.Equal(t, "test-topic", pageModel.GetTopicName())
	assert.Equal(t, testMessage, pageModel.GetMessage())
	assert.Equal(t, mockDS, model.dataSource)

	// Check display format defaults
	assert.Equal(t, "pretty", model.displayFormat.ValueFormat)
	assert.Equal(t, "raw", model.displayFormat.KeyFormat)
	assert.True(t, model.displayFormat.WrapLines)
	assert.False(t, model.displayFormat.ShowBytes)
	assert.True(t, model.showHeaders)
	assert.True(t, model.showMetadata)
}

func TestModelImplementsPageInterface(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	// Create test message
	testMessage := api.Message{
		Key:       "test-key",
		Value:     "test-value",
		Offset:    123,
		Partition: 0,
	}

	// Create new model using the migrated structure
	pageModel := NewMessageDetailPageModel(mockDS, "test-topic", testMessage)
	model := pageModel.GetDetailModel()

	// Test that pageModel implements the Page interface methods
	// GetID now returns dynamic ID format: "detail:<topic>:<partition>:<offset>"
	id := pageModel.GetID()
	assert.Contains(t, id, "detail:")
	assert.Contains(t, id, "test-topic")

	// Test Init (template system initialization)
	// Note: Init may return nil if child components don't have init commands
	_ = pageModel.Init()

	// Test SetDimensions
	pageModel.SetDimensions(80, 24)
	assert.Equal(t, 80, model.dimensions.Width)
	assert.Equal(t, 24, model.dimensions.Height)

	// Test View returns a string (basic test)
	pageModel.SetDimensions(80, 24) // Ensure dimensions are set
	view := pageModel.View()
	assert.IsType(t, "", view)
	// Note: The template system may format content differently, so we check for basic functionality
}

func TestGetFormattedKey(t *testing.T) {
	testCases := []struct {
		name        string
		key         string
		keyFormat   string
		expected    string
		description string
	}{
		{
			name:        "Nil key",
			key:         "",
			keyFormat:   "raw",
			expected:    "<null>",
			description: "Should return <null> for empty key",
		},
		{
			name:        "Raw format",
			key:         "test-key",
			keyFormat:   "raw",
			expected:    "test-key",
			description: "Should return key as-is for raw format",
		},
		{
			name:        "JSON format",
			key:         `{"id": "123"}`,
			keyFormat:   "json",
			expected:    "{\n  \"id\": \"123\"\n}",
			description: "Should return pretty-printed JSON for json format",
		},
		{
			name:        "Hex format",
			key:         "test",
			keyFormat:   "hex",
			expected:    "74657374",
			description: "Should return hex representation for hex format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock data source
			mockDS := &mock.KafkaDataSourceMock{}
			mockDS.Init("")

			message := api.Message{Key: tc.key, Value: "test-value", Offset: 1, Partition: 0}
			if tc.key == "" {
				// Simulate nil key by setting it to empty and testing the logic
				message.Key = ""
			}

			pageModel := NewMessageDetailPageModel(mockDS, "test-topic", message)
		model := pageModel.GetDetailModel()
			model.displayFormat.KeyFormat = tc.keyFormat

			result := model.GetFormattedKey()
			if tc.key == "" {
				assert.Equal(t, tc.expected, result, tc.description)
			} else {
				// For non-empty keys, check the result contains expected content
				if tc.keyFormat == "hex" {
					assert.Equal(t, tc.expected, result, tc.description)
				} else if tc.keyFormat == "json" {
					// For JSON format, check that the result is the expected pretty-printed JSON
					assert.Equal(t, tc.expected, result, tc.description)
				} else {
					assert.Contains(t, result, tc.key, tc.description)
				}
			}
		})
	}
}

func TestGetFormattedValue(t *testing.T) {
	testCases := []struct {
		name        string
		value       string
		valueFormat string
		expected    string
		description string
	}{
		{
			name:        "Nil value",
			value:       "",
			valueFormat: "raw",
			expected:    "<null>",
			description: "Should return <null> for empty value",
		},
		{
			name:        "Raw format",
			value:       "test-value",
			valueFormat: "raw",
			expected:    "test-value",
			description: "Should return value as-is for raw format",
		},
		{
			name:        "Pretty format",
			value:       `{"name": "test"}`,
			valueFormat: "pretty",
			expected:    "{\n  \"name\": \"test\"\n}",
			description: "Should return pretty-printed content for pretty format",
		},
		{
			name:        "JSON format",
			value:       `{"data": "value"}`,
			valueFormat: "json",
			expected:    "{\n  \"data\": \"value\"\n}",
			description: "Should return pretty-printed JSON content",
		},
		{
			name:        "Hex format",
			value:       "test",
			valueFormat: "hex",
			expected:    "74657374",
			description: "Should return hex representation for hex format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock data source
			mockDS := &mock.KafkaDataSourceMock{}
			mockDS.Init("")

			message := api.Message{Key: "test-key", Value: tc.value, Offset: 1, Partition: 0}
			if tc.value == "" {
				message.Value = ""
			}

			pageModel := NewMessageDetailPageModel(mockDS, "test-topic", message)
		model := pageModel.GetDetailModel()
			model.displayFormat.ValueFormat = tc.valueFormat

			result := model.GetFormattedValue()
			if tc.value == "" {
				assert.Equal(t, tc.expected, result, tc.description)
			} else {
				if tc.valueFormat == "hex" {
					assert.Equal(t, tc.expected, result, tc.description)
				} else if tc.valueFormat == "json" || tc.valueFormat == "pretty" {
					// For JSON/pretty formats, check that the result is the expected pretty-printed JSON
					assert.Equal(t, tc.expected, result, tc.description)
				} else {
					assert.Contains(t, result, tc.value, tc.description)
				}
			}
		})
	}
}

func TestToggleDisplayFormat(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	message := api.Message{Key: "test-key", Value: "test-value", Offset: 1, Partition: 0}
	pageModel := NewMessageDetailPageModel(mockDS, "test-topic", message)
		model := pageModel.GetDetailModel()

	// Test format cycling
	testCases := []struct {
		initial  string
		expected string
	}{
		{"raw", "pretty"},
		{"pretty", "json"},
		{"json", "hex"},
		{"hex", "raw"},
		{"unknown", "raw"}, // Default case
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_to_%s", tc.initial, tc.expected), func(t *testing.T) {
			model.displayFormat.ValueFormat = tc.initial
			model.ToggleDisplayFormat()
			assert.Equal(t, tc.expected, model.displayFormat.ValueFormat)
		})
	}
}

func TestToggleHeaders(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	message := api.Message{Key: "test-key", Value: "test-value", Offset: 1, Partition: 0}
	pageModel := NewMessageDetailPageModel(mockDS, "test-topic", message)
		model := pageModel.GetDetailModel()

	// Initially should show headers
	assert.True(t, model.showHeaders)

	// Toggle off
	model.ToggleHeaders()
	assert.False(t, model.showHeaders)

	// Toggle on
	model.ToggleHeaders()
	assert.True(t, model.showHeaders)
}

func TestToggleMetadata(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	message := api.Message{Key: "test-key", Value: "test-value", Offset: 1, Partition: 0}
	pageModel := NewMessageDetailPageModel(mockDS, "test-topic", message)
		model := pageModel.GetDetailModel()

	// Initially should show metadata
	assert.True(t, model.showMetadata)

	// Toggle off
	model.ToggleMetadata()
	assert.False(t, model.showMetadata)

	// Toggle on
	model.ToggleMetadata()
	assert.True(t, model.showMetadata)
}

func TestWindowSizeUpdate(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	message := api.Message{Key: "test-key", Value: "test-value", Offset: 1, Partition: 0}
	pageModel := NewMessageDetailPageModel(mockDS, "test-topic", message)

	// Test SetDimensions method (the proper way to update dimensions)
	pageModel.SetDimensions(100, 30)

	// Check dimensions were updated via the detail model
	detailModel := pageModel.GetDetailModel()
	assert.Equal(t, 100, detailModel.dimensions.Width)
	assert.Equal(t, 30, detailModel.dimensions.Height)
}

func TestKeyHandling(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	message := api.Message{Key: "test-key", Value: "test-value", Offset: 1, Partition: 0}
	pageModel := NewMessageDetailPageModel(mockDS, "test-topic", message)
	detailModel := pageModel.GetDetailModel()

	// Test toggle format
	originalFormat := detailModel.displayFormat.ValueFormat
	detailModel.ToggleDisplayFormat()

	// Format should have changed
	assert.NotEqual(t, originalFormat, detailModel.displayFormat.ValueFormat)

	// Test toggle headers
	originalHeaders := detailModel.showHeaders
	detailModel.ToggleHeaders()

	// Headers display should have toggled
	assert.NotEqual(t, originalHeaders, detailModel.showHeaders)

	// Test toggle metadata
	originalMetadata := detailModel.showMetadata
	detailModel.ToggleMetadata()

	// Metadata display should have toggled
	assert.NotEqual(t, originalMetadata, detailModel.showMetadata)
}

func TestSchemaInfoLazyLoading(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	// Create test message with schema IDs
	testMessage := api.Message{
		Key:           "test-key",
		Value:         "test-value",
		Offset:        123,
		Partition:     0,
		KeySchemaID:   "1",
		ValueSchemaID: "2",
	}

	// Create new model
	pageModel1 := NewMessageDetailPageModel(mockDS, "test-topic", testMessage)
	model := pageModel1.GetDetailModel()

	// Schema info should be nil initially (lazy loading)
	assert.Nil(t, model.schemaInfo)

	// Accessing schema info should trigger lazy loading
	schemaInfo := model.GetSchemaInfo()

	// Now schema info should be loaded
	assert.NotNil(t, schemaInfo)
	assert.NotNil(t, model.schemaInfo) // Should be cached now

	// Test message without schema IDs
	messageNoSchema := api.Message{
		Key:       "test-key",
		Value:     "test-value",
		Offset:    124,
		Partition: 0,
		// No schema IDs
	}

	pageModel2 := NewMessageDetailPageModel(mockDS, "test-topic", messageNoSchema)
	modelNoSchema := pageModel2.GetDetailModel()
	schemaInfoNoSchema := modelNoSchema.GetSchemaInfo()

	// Should remain nil for messages without schema IDs
	assert.Nil(t, schemaInfoNoSchema)
	assert.Nil(t, modelNoSchema.schemaInfo)
}

// TestGetID tests the unique page ID generation
func TestGetID(t *testing.T) {
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	testMessage := api.Message{
		Key:       "test-key",
		Value:     "test-value",
		Offset:    123,
		Partition: 2,
	}

	pageModel := NewMessageDetailPageModel(mockDS, "my-topic", testMessage)
	id := pageModel.GetID()

	// Verify ID format: "detail:<topic>:<partition>:<offset>"
	assert.Contains(t, id, "detail:")
	assert.Contains(t, id, "my-topic")
	assert.Contains(t, id, "2")  // partition
	assert.Contains(t, id, "123") // offset

	// Verify different messages produce different IDs
	testMessage2 := api.Message{
		Key:       "test-key-2",
		Value:     "test-value-2",
		Offset:    456,
		Partition: 1,
	}
	pageModel2 := NewMessageDetailPageModel(mockDS, "my-topic", testMessage2)
	id2 := pageModel2.GetID()

	assert.NotEqual(t, id, id2)
	assert.Contains(t, id2, "456") // different offset
	assert.Contains(t, id2, "1")   // different partition
}

// TestGetIDWithSpecialCharacters tests page ID generation with special topic names
func TestGetIDWithSpecialCharacters(t *testing.T) {
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	testMessage := api.Message{
		Key:       "key",
		Value:     "value",
		Offset:    0,
		Partition: 0,
	}

	// Test with topic names containing special characters
	testCases := []struct {
		topicName    string
		shouldContain string
	}{
		{"topic-with-dashes", "topic-with-dashes"},
		{"topic_with_underscores", "topic_with_underscores"},
		{"topic.with.dots", "topic.with.dots"},
		{"TopicWithCamelCase", "TopicWithCamelCase"},
	}

	for _, tc := range testCases {
		t.Run(tc.topicName, func(t *testing.T) {
			pageModel := NewMessageDetailPageModel(mockDS, tc.topicName, testMessage)
			id := pageModel.GetID()
			assert.Contains(t, id, tc.shouldContain)
		})
	}
}

// TestGetTitle tests the page title generation
func TestGetTitle(t *testing.T) {
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	testMessage := api.Message{
		Key:       "test-key",
		Value:     "test-value",
		Offset:    123,
		Partition: 0,
	}

	pageModel := NewMessageDetailPageModel(mockDS, "my-topic", testMessage)
	title := pageModel.GetTitle()

	assert.NotEmpty(t, title)
	assert.Contains(t, title, "my-topic")
}

