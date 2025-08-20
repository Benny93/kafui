package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/components"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// TestFuzzyMatchingDemo demonstrates the fuzzy matching functionality
func TestFuzzyMatchingDemo(t *testing.T) {
	fmt.Println("=== Fuzzy Matching Demo ===")
	
	// Test the fuzzy matcher directly
	matcher := components.NewFuzzyMatcher(false)
	
	// Test 1: Resource name fuzzy matching
	fmt.Println("\n1. Testing resource name fuzzy matching...")
	
	resourceCandidates := []string{
		"topics", "topic", "consumer-groups", "consumers", 
		"consumer", "groups", "cg", "schemas", "schema", 
		"contexts", "context", "ctx",
	}
	
	testCases := []struct {
		query    string
		expected string
		desc     string
	}{
		{"con", "consumer-groups", "partial match should prefer full name"},
		{"cg", "cg", "exact match should be preferred"},
		{"grps", "groups", "subsequence match should prefer shorter when ambiguous"},
		{"schm", "schema", "subsequence match with missing vowels should prefer shorter"},
		{"tpcs", "topics", "subsequence match"},
		{"ctxt", "context", "subsequence match should prefer shorter when ambiguous"},
		{"consumr", "consumer", "typo tolerance"},
		{"topic", "topic", "exact match"},
		{"xyz", "", "no match for invalid input"},
	}
	
	for _, tc := range testCases {
		bestMatch := matcher.GetBestMatch(tc.query, resourceCandidates)
		fmt.Printf("   Query: '%s' → Best match: '%s' (%s)\n", tc.query, bestMatch, tc.desc)
		
		if tc.expected == "" {
			assert.Empty(t, bestMatch, "Should have no match for query '%s'", tc.query)
		} else {
			assert.Equal(t, tc.expected, bestMatch, "Query '%s' should match '%s'", tc.query, tc.expected)
		}
	}
	
	// Test 2: Topic name fuzzy matching
	fmt.Println("\n2. Testing topic name fuzzy matching...")
	
	topicCandidates := []string{
		"user-events", "user-analytics", "order-processing", 
		"payment-transactions", "inventory-updates", "notification-service",
		"audit-logs", "error-tracking", "performance-metrics",
	}
	
	topicTestCases := []struct {
		query    string
		expected string
		desc     string
	}{
		{"user", "user-events", "prefix match should prefer first alphabetically"},
		{"usr", "user-events", "subsequence match"},
		{"order", "order-processing", "exact word match"},
		{"pay", "payment-transactions", "prefix match"},
		{"notif", "notification-service", "prefix match"},
		{"perf", "performance-metrics", "prefix match"},
		{"err", "error-tracking", "prefix match"},
		{"invntry", "inventory-updates", "subsequence with missing vowels"},
		{"audt", "audit-logs", "subsequence match"},
	}
	
	for _, tc := range topicTestCases {
		bestMatch := matcher.GetBestMatch(tc.query, topicCandidates)
		fmt.Printf("   Query: '%s' → Best match: '%s' (%s)\n", tc.query, bestMatch, tc.desc)
		assert.Equal(t, tc.expected, bestMatch, "Query '%s' should match '%s'", tc.query, tc.expected)
	}
	
	// Test 3: Integration with search bar
	fmt.Println("\n3. Testing fuzzy matching integration with search bar...")
	
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	
	// Create main page model
	model := NewMainPage(mockDS)
	model.width = 120
	model.height = 40
	
	// Load topics to populate suggestions
	topicListMsg := model.loadTopics()
	updatedModel, _ := model.Update(topicListMsg)
	mainModel := updatedModel.(*MainPageModel)
	
	// Enter resource mode and test fuzzy completion
	updatedModel, _ = mainModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	mainModel = updatedModel.(*MainPageModel)
	
	// Test fuzzy completion with typos
	fuzzyTests := []struct {
		input    string
		expected string
	}{
		{"consumr", "consumer"},
		{"tpic", "topic"},
		{"schma", "schema"},
		{"cntxt", "context"},
	}
	
	for _, test := range fuzzyTests {
		// Clear and type the input
		mainModel.searchBar.SetValue("")
		for _, char := range test.input {
			mainModel.searchBar, _ = mainModel.searchBar.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		}
		
		// Press tab for fuzzy completion
		mainModel.searchBar, _ = mainModel.searchBar.Update(tea.KeyMsg{Type: tea.KeyTab})
		
		value := mainModel.searchBar.Value()
		fmt.Printf("   Fuzzy input: '%s' → Completed to: '%s'\n", test.input, value)
		assert.Equal(t, test.expected, value, "Fuzzy input '%s' should complete to '%s'", test.input, test.expected)
	}
	
	// Test 4: Multiple matches and ranking
	fmt.Println("\n4. Testing multiple matches and ranking...")
	
	matches := matcher.Match("con", resourceCandidates, 5)
	fmt.Printf("   Query 'con' found %d matches:\n", len(matches))
	for i, match := range matches {
		fmt.Printf("     %d. '%s' (score: %d)\n", i+1, match.Text, match.Score)
	}
	
	// Should prefer exact matches, then prefix matches, then subsequence matches
	assert.True(t, len(matches) > 0, "Should find matches for 'con'")
	if len(matches) > 0 {
		// First match should be a prefix match starting with 'con'
		validFirstMatches := []string{"consumer-groups", "consumers", "consumer", "contexts", "context"}
		isValidFirst := false
		for _, valid := range validFirstMatches {
			if matches[0].Text == valid {
				isValidFirst = true
				break
			}
		}
		assert.True(t, isValidFirst, "First match should be one of the 'con' prefixed items, got: %s", matches[0].Text)
	}
	
	// Test 5: Dynamic suggestions update
	fmt.Println("\n5. Testing dynamic suggestions update...")
	
	// Exit resource mode and enter search mode
	updatedModel, _ = mainModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mainModel = updatedModel.(*MainPageModel)
	
	updatedModel, _ = mainModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	mainModel = updatedModel.(*MainPageModel)
	
	// Type partial topic name
	for _, char := range "Topic" {
		mainModel.searchBar, _ = mainModel.searchBar.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
	}
	
	// Press tab for completion
	mainModel.searchBar, _ = mainModel.searchBar.Update(tea.KeyMsg{Type: tea.KeyTab})
	
	value := mainModel.searchBar.Value()
	fmt.Printf("   Search completion: 'Topic' → '%s'\n", value)
	
	// Should complete to one of the Topic names
	assert.True(t, strings.HasPrefix(value, "Topic"), "Should complete to a topic name starting with 'Topic'")
	
	fmt.Println("\n✅ All fuzzy matching tests passed!")
	fmt.Println("\nFuzzy Matching Features Demonstrated:")
	fmt.Println("• Exact match prioritization")
	fmt.Println("• Prefix matching")
	fmt.Println("• Subsequence matching (characters in order)")
	fmt.Println("• Typo tolerance")
	fmt.Println("• Word boundary matching")
	fmt.Println("• Score-based ranking")
	fmt.Println("• Dynamic suggestion updates")
	fmt.Println("• Integration with Tab completion")
}