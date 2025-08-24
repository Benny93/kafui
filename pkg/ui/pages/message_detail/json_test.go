package messagedetail

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/stretchr/testify/assert"
)

func TestJSONPrettyPrinting(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	// Create test message with JSON content
	testMessage := api.Message{
		Key:       `{"id": "test-key"}`,
		Value:     `{"name": "test", "age": 30, "active": true, "data": null}`,
		Offset:    123,
		Partition: 0,
	}

	// Create new model
	model := NewModel(mockDS, "test-topic", testMessage)

	// Test JSON key formatting
	model.displayFormat.KeyFormat = "json"
	formattedKey := model.GetFormattedKey()
	assert.Contains(t, formattedKey, "{")
	assert.Contains(t, formattedKey, "\"id\"")
	assert.Contains(t, formattedKey, "\"test-key\"")
	assert.Contains(t, formattedKey, "\n") // Should have newlines for pretty printing

	// Test JSON value formatting
	model.displayFormat.ValueFormat = "json"
	formattedValue := model.GetFormattedValue()
	assert.Contains(t, formattedValue, "{")
	assert.Contains(t, formattedValue, "\"name\"")
	assert.Contains(t, formattedValue, "\"test\"")
	assert.Contains(t, formattedValue, "\n") // Should have newlines for pretty printing

	// Test pretty format (with syntax highlighting)
	model.displayFormat.ValueFormat = "pretty"
	prettyValue := model.GetFormattedValue()
	assert.Contains(t, prettyValue, "{")
	assert.Contains(t, prettyValue, "\"name\"")
	assert.Contains(t, prettyValue, "\"test\"")
	assert.Contains(t, prettyValue, "\n") // Should have newlines for pretty printing
}

func TestNonJSONContent(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	// Create test message with non-JSON content
	testMessage := api.Message{
		Key:       "simple-key",
		Value:     "simple-value",
		Offset:    123,
		Partition: 0,
	}

	// Create new model
	model := NewModel(mockDS, "test-topic", testMessage)

	// Test that non-JSON content is returned as-is
	model.displayFormat.KeyFormat = "json"
	formattedKey := model.GetFormattedKey()
	assert.Equal(t, "simple-key", formattedKey) // Should return original content

	model.displayFormat.ValueFormat = "json"
	formattedValue := model.GetFormattedValue()
	assert.Equal(t, "simple-value", formattedValue) // Should return original content
}