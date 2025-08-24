package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONContentViewBasics(t *testing.T) {
	// Test basic creation
	config := JSONContentConfig{
		Width:           50,
		Height:          20,
		Title:           "Test Content",
		Content:         `{"name": "test", "value": 123}`,
		DisplayFormat:   "json",
		ShowLineNumbers: true,
		Focused:         false,
	}

	viewer := NewJSONContentView(config)
	assert.NotNil(t, viewer)
	assert.NotNil(t, viewer.Viewport())
}

func TestContentFormatting(t *testing.T) {
	// Test content formatting
	content := `{"name": "test", "value": 123}`
	formatted := formatContent(content, "json")
	assert.Contains(t, formatted, "{")
	assert.Contains(t, formatted, "\"name\"")
	assert.Contains(t, formatted, "\"test\"")

	// Test line numbering
	numbered := addLineNumbers("line1\nline2\nline3")
	assert.Contains(t, numbered, "   1 line1")
	assert.Contains(t, numbered, "   2 line2")
	assert.Contains(t, numbered, "   3 line3")

	// Test null content
	nullNumbered := addLineNumbers("<null>")
	assert.Equal(t, "<null>", nullNumbered)

	// Test hex formatting
	hexContent := formatContent("test", "hex")
	assert.Equal(t, "74657374", hexContent)

	// Test raw formatting
	rawContent := formatContent("test content", "raw")
	assert.Equal(t, "test content", rawContent)
}