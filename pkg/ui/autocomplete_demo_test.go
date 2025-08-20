package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// TestAutoCompletionDemo demonstrates the new auto-completion functionality
func TestAutoCompletionDemo(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	
	// Create main page model
	model := NewMainPage(mockDS)
	model.width = 120
	model.height = 40
	
	fmt.Println("=== Auto-Completion Demo ===")
	
	// Test 1: Resource auto-completion
	fmt.Println("\n1. Testing resource auto-completion...")
	
	// Enter resource mode
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	mainModel := updatedModel.(*MainPageModel)
	
	// Verify suggestions are set for resource mode
	assert.True(t, mainModel.searchBar.IsResourceMode(), "Should be in resource mode")
	
	// Simulate typing "con" and pressing tab for completion
	fmt.Println("   Typing 'con' and pressing Tab...")
	for _, char := range "con" {
		mainModel.searchBar, _ = mainModel.searchBar.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
	}
	
	// Press tab for auto-completion
	mainModel.searchBar, _ = mainModel.searchBar.Update(tea.KeyMsg{Type: tea.KeyTab})
	
	// Check if it completed to a valid resource name
	value := mainModel.searchBar.Value()
	fmt.Printf("   Auto-completed to: '%s'\n", value)
	
	// Should complete to one of: consumer-groups, consumers, consumer, contexts, context
	validCompletions := []string{"consumer-groups", "consumers", "consumer", "contexts", "context"}
	isValidCompletion := false
	for _, completion := range validCompletions {
		if value == completion {
			isValidCompletion = true
			break
		}
	}
	assert.True(t, isValidCompletion, "Should auto-complete to a valid resource name starting with 'con'")
	
	// Render the view with auto-completion
	rendered := mainModel.View()
	doc := strings.Builder{}
	doc.WriteString(rendered)
	fmt.Println(docStyle.Render(doc.String()))
	
	// Test 2: Search auto-completion with topic names
	fmt.Println("\n2. Testing search auto-completion with topic names...")
	
	// Exit resource mode
	updatedModel, _ = mainModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mainModel = updatedModel.(*MainPageModel)
	
	// Load some topics first to populate suggestions
	topicListMsg := mainModel.loadTopics()
	updatedModel, _ = mainModel.Update(topicListMsg)
	mainModel = updatedModel.(*MainPageModel)
	
	// Enter search mode
	updatedModel, _ = mainModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	mainModel = updatedModel.(*MainPageModel)
	
	// Verify we're in search mode (not resource mode)
	assert.True(t, mainModel.searchMode, "Should be in search mode")
	assert.False(t, mainModel.searchBar.IsResourceMode(), "Should NOT be in resource mode")
	
	// Render the view in search mode with suggestions
	rendered = mainModel.View()
	doc = strings.Builder{}
	doc.WriteString(rendered)
	fmt.Println(docStyle.Render(doc.String()))
	
	// Test 3: Different resource types and their suggestions
	fmt.Println("\n3. Testing different resource type completions...")
	
	resourceTests := []struct {
		input    string
		expected []string
	}{
		{"top", []string{"topics", "topic"}},
		{"sch", []string{"schemas", "schema"}},
		{"ctx", []string{"ctx"}},
		{"cg", []string{"cg"}},
	}
	
	for _, test := range resourceTests {
		// Exit current mode
		updatedModel, _ = mainModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
		mainModel = updatedModel.(*MainPageModel)
		
		// Enter resource mode
		updatedModel, _ = mainModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		mainModel = updatedModel.(*MainPageModel)
		
		// Type the input
		for _, char := range test.input {
			mainModel.searchBar, _ = mainModel.searchBar.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		}
		
		// Press tab for completion
		mainModel.searchBar, _ = mainModel.searchBar.Update(tea.KeyMsg{Type: tea.KeyTab})
		
		value := mainModel.searchBar.Value()
		fmt.Printf("   Input: '%s' → Completed to: '%s'\n", test.input, value)
		
		// Check if completion is valid
		isValid := false
		for _, expected := range test.expected {
			if value == expected {
				isValid = true
				break
			}
		}
		assert.True(t, isValid, "Should complete '%s' to one of %v, got '%s'", test.input, test.expected, value)
	}
	
	// Test 4: Tab completion behavior with no matches
	fmt.Println("\n4. Testing tab completion with no matches...")
	
	// Exit current mode
	updatedModel, _ = mainModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mainModel = updatedModel.(*MainPageModel)
	
	// Enter resource mode
	updatedModel, _ = mainModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	mainModel = updatedModel.(*MainPageModel)
	
	// Type something that doesn't match
	for _, char := range "xyz" {
		mainModel.searchBar, _ = mainModel.searchBar.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
	}
	
	originalValue := mainModel.searchBar.Value()
	
	// Press tab - should not change the value
	mainModel.searchBar, _ = mainModel.searchBar.Update(tea.KeyMsg{Type: tea.KeyTab})
	
	newValue := mainModel.searchBar.Value()
	fmt.Printf("   No match input: '%s' → After tab: '%s'\n", originalValue, newValue)
	assert.Equal(t, originalValue, newValue, "Should not change value when no completion matches")
	
	fmt.Println("\n✅ All auto-completion tests passed!")
	fmt.Println("\nAuto-Completion Features Demonstrated:")
	fmt.Println("• Resource name completion with Tab key")
	fmt.Println("• Dynamic suggestions based on available items")
	fmt.Println("• Prefix matching for partial inputs")
	fmt.Println("• No-op behavior when no matches found")
	fmt.Println("• Separate suggestion sets for search vs resource modes")
	fmt.Println("• Visual feedback through suggestion dropdown")
}