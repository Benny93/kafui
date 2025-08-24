package messagedetail

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	tea "github.com/charmbracelet/bubbletea"
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
	model := NewModel(mockDS, "test-topic", testMessage)

	// Verify viewports are initialized
	assert.NotNil(t, model.keyViewport)
	assert.NotNil(t, model.valueViewport)

	// Check initial focus
	assert.Equal(t, "value", model.focusedViewport)

	// Test switching focus
	model.SwitchFocus()
	assert.Equal(t, "key", model.focusedViewport)

	model.SwitchFocus()
	assert.Equal(t, "value", model.focusedViewport)

	// Test window size update
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updatedModel, cmd := model.Update(msg)

	assert.IsType(t, &Model{}, updatedModel)
	assert.Nil(t, cmd)

	updatedDetailModel := updatedModel.(*Model)
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