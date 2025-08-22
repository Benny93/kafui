package ui

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNaturalSorting(t *testing.T) {
	// Test case 1: Basic natural sorting with numbers
	testNames := []string{
		"topic-10",
		"topic-2",
		"topic-1",
		"topic-20",
		"topic-3",
	}

	expected := []string{
		"topic-1",
		"topic-2",
		"topic-3",
		"topic-10",
		"topic-20",
	}

	sort.Sort(NaturalSort(testNames))

	assert.Equal(t, expected, testNames, "Natural sorting should order numbers correctly")

	// Test case 2: Mixed alphanumeric sorting
	testNames2 := []string{
		"consumer-group-10",
		"consumer-group-2",
		"api-topic-1",
		"api-topic-10",
		"data-stream-5",
		"data-stream-15",
	}

	expected2 := []string{
		"api-topic-1",
		"api-topic-10",
		"consumer-group-2",
		"consumer-group-10",
		"data-stream-5",
		"data-stream-15",
	}

	sort.Sort(NaturalSort(testNames2))

	assert.Equal(t, expected2, testNames2, "Natural sorting should handle mixed prefixes correctly")

	// Test case 3: Numbers with different lengths
	testNames3 := []string{
		"log-100",
		"log-20",
		"log-3",
		"log-1000",
		"log-4",
	}

	expected3 := []string{
		"log-3",
		"log-4",
		"log-20",
		"log-100",
		"log-1000",
	}

	sort.Sort(NaturalSort(testNames3))

	assert.Equal(t, expected3, testNames3, "Natural sorting should handle different number lengths correctly")
}

func TestNaturalLess(t *testing.T) {
	// Test specific comparisons
	assert.True(t, NaturalLess("topic-1", "topic-10"), "topic-1 should come before topic-10")
	assert.True(t, NaturalLess("topic-2", "topic-10"), "topic-2 should come before topic-10")
	assert.False(t, NaturalLess("topic-10", "topic-2"), "topic-10 should come after topic-2")
	assert.True(t, NaturalLess("abc", "abc123"), "abc should come before abc123")
	assert.True(t, NaturalLess("topic-9", "topic-10"), "topic-9 should come before topic-10")
}

func TestHighlightSearchMatches(t *testing.T) {
	// Test search highlighting
	text := "my-topic-name-123"
	searchQuery := "topic"

	highlighted := HighlightSearchMatches(text, searchQuery)

	// Should contain the original text
	assert.Contains(t, highlighted, "my-", "Should contain text before match")
	assert.Contains(t, highlighted, "-name-123", "Should contain text after match")

	// Test empty search query
	noHighlight := HighlightSearchMatches(text, "")
	assert.Equal(t, text, noHighlight, "Empty search should return original text")

	// Test no matches
	noMatch := HighlightSearchMatches(text, "xyz")
	assert.Equal(t, text, noMatch, "No matches should return original text")
}
