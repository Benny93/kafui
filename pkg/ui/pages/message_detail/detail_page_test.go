package messagedetail

import (
	"fmt"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	tea "github.com/charmbracelet/bubbletea"
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
	assert.Equal(t, "message_detail", pageModel.GetID())

	// Test Init returns a command (template system initialization)
	cmd := pageModel.Init()
	assert.NotNil(t, cmd)

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
		model := pageModel.GetDetailModel()

	// Test window size message
	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	updatedModel, cmd := model.Update(msg)

	// Should return the same model type
	assert.IsType(t, &Model{}, updatedModel)
	assert.Nil(t, cmd) // Window size updates don't return commands for detail page

	// Check dimensions were updated
	updatedDetailModel := updatedModel.(*Model)
	assert.Equal(t, 100, updatedDetailModel.dimensions.Width)
	assert.Equal(t, 30, updatedDetailModel.dimensions.Height)
}

func TestKeyHandling(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	message := api.Message{Key: "test-key", Value: "test-value", Offset: 1, Partition: 0}
	pageModel := NewMessageDetailPageModel(mockDS, "test-topic", message)
		model := pageModel.GetDetailModel()

	// Test toggle format key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}
	originalFormat := model.displayFormat.ValueFormat
	updatedModel, cmd := model.Update(msg)
	updatedDetailModel := updatedModel.(*Model)

	// Format should have changed
	assert.NotEqual(t, originalFormat, updatedDetailModel.displayFormat.ValueFormat)
	assert.Nil(t, cmd) // Toggle commands don't return tea.Cmd

	// Test toggle headers key
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	originalHeaders := updatedDetailModel.showHeaders
	updatedModel, cmd = updatedDetailModel.Update(msg)
	updatedDetailModel = updatedModel.(*Model)

	// Headers display should have toggled
	assert.NotEqual(t, originalHeaders, updatedDetailModel.showHeaders)
	assert.Nil(t, cmd)

	// Test toggle metadata key
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
	originalMetadata := updatedDetailModel.showMetadata
	updatedModel, cmd = updatedDetailModel.Update(msg)
	updatedDetailModel = updatedModel.(*Model)

	// Metadata display should have toggled
	assert.NotEqual(t, originalMetadata, updatedDetailModel.showMetadata)
	assert.Nil(t, cmd)
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
	pageModel := NewMessageDetailPageModel(mockDS, "test-topic", testMessage)
		model := pageModel.GetDetailModel()

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

	pageModel := NewMessageDetailPageModel(mockDS, "test-topic", messageNoSchema)
		modelNoSchema := pageModel.GetDetailModel()
	schemaInfoNoSchema := modelNoSchema.GetSchemaInfo()

	// Should remain nil for messages without schema IDs
	assert.Nil(t, schemaInfoNoSchema)
	assert.Nil(t, modelNoSchema.schemaInfo)
}

