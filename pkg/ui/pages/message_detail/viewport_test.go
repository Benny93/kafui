package messagedetail

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/stretchr/testify/assert"
)

func TestViewportFunctionality(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	// Create test message with multi-line content
	testMessage := api.Message{
		Key:       "test-key",
		Value:     "line1\nline2\nline3\nline4\nline5",
		Offset:    123,
		Partition: 0,
	}

	// Create new model
	pageModel := NewMessageDetailPageModel(mockDS, "test-topic", testMessage)

	// Verify model is initialized
	assert.NotNil(t, pageModel)

	// Check initial focus via detail model
	detailModel := pageModel.GetDetailModel()
	assert.Equal(t, "value", detailModel.focusedViewport)

	// Test switching focus
	detailModel.SwitchFocus()
	assert.Equal(t, "key", detailModel.focusedViewport)

	detailModel.SwitchFocus()
	assert.Equal(t, "value", detailModel.focusedViewport)

	// Test SetDimensions
	pageModel.SetDimensions(100, 50)

	// Check dimensions were updated
	updatedDetailModel := pageModel.GetDetailModel()
	assert.Equal(t, 100, updatedDetailModel.dimensions.Width)
	assert.Equal(t, 50, updatedDetailModel.dimensions.Height)
}

func TestAddLineNumbers(t *testing.T) {
	// Test single line
	result := addLineNumbers("hello")
	assert.Equal(t, "   1 hello", result)

	// Test multi-line
	content := "line1\nline2\nline3"
	result = addLineNumbers(content)
	expected := "   1 line1\n   2 line2\n   3 line3"
	assert.Equal(t, expected, result)

	// Test null content
	result = addLineNumbers("<null>")
	assert.Equal(t, "<null>", result)
}
