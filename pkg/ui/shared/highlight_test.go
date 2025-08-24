package shared

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestSearchHighlightConfig(t *testing.T) {
	config := SearchHighlightConfig()
	
	// Should have no background color
	assert.Equal(t, "", config.BackgroundColor)
	
	// Should have foreground color
	assert.Equal(t, "205", config.ForegroundColor)
	
	// Should be bold
	assert.True(t, config.Bold)
}

func TestHighlightSearchMatches(t *testing.T) {
	// Test basic highlighting
	result := HighlightSearchMatches("hello world", "world")
	assert.Contains(t, result, "world") // Should contain the highlighted text
	
	// Test case insensitive matching
	result = HighlightSearchMatches("Hello World", "world")
	assert.Contains(t, result, "World") // Should highlight "World"
	
	// Test no match
	result = HighlightSearchMatches("hello world", "xyz")
	assert.Equal(t, "hello world", result) // Should return original text
	
	// Test empty query
	result = HighlightSearchMatches("hello world", "")
	assert.Equal(t, "hello world", result) // Should return original text
}