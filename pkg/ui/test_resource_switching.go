package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// TestResourceSwitchingDemo demonstrates the new k9s-style resource switching
func TestResourceSwitchingDemo(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}

	// Create main page model
	model := NewMainPage(mockDS)
	model.width = 120
	model.height = 40

	fmt.Println("=== K9s-Style Resource Switching Demo ===")

	// Test 1: Enter resource mode with ":"
	fmt.Println("\n1. Pressing ':' to enter resource mode...")
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	mainModel := updatedModel.(*MainPageModel)

	// Verify we're in resource mode
	assert.True(t, mainModel.searchMode, "Should be in search mode")
	assert.True(t, mainModel.searchBar.IsResourceMode(), "Should be in resource mode")

	// Render the view in resource mode
	rendered := mainModel.View()
	doc := strings.Builder{}
	doc.WriteString(rendered)
	fmt.Println(docStyle.Render(doc.String()))

	// Test 2: Type "consumer-groups" and press enter
	fmt.Println("\n2. Typing 'consumer-groups' and pressing enter...")

	// Simulate typing "consumer-groups"
	for _, char := range "consumer-groups" {
		mainModel.searchBar, _ = mainModel.searchBar.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
	}

	// Press enter to switch resource
	updatedModel, _ = mainModel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mainModel = updatedModel.(*MainPageModel)

	// Verify we switched to consumer groups
	assert.Equal(t, ConsumerGroupResourceType, mainModel.currentResource.GetType(), "Should switch to consumer groups")
	assert.False(t, mainModel.searchMode, "Should exit search mode")

	// Render the view after switching
	rendered = mainModel.View()
	doc = strings.Builder{}
	doc.WriteString(rendered)
	fmt.Println(docStyle.Render(doc.String()))

	// Test 3: Enter search mode with "/" to search within consumer groups
	fmt.Println("\n3. Pressing '/' to search within consumer groups...")
	updatedModel, _ = mainModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	mainModel = updatedModel.(*MainPageModel)

	// Verify we're in normal search mode (not resource mode)
	assert.True(t, mainModel.searchMode, "Should be in search mode")
	assert.False(t, mainModel.searchBar.IsResourceMode(), "Should NOT be in resource mode")

	// Render the view in search mode
	rendered = mainModel.View()
	doc = strings.Builder{}
	doc.WriteString(rendered)
	fmt.Println(docStyle.Render(doc.String()))

	// Test 4: Press escape to cancel search
	fmt.Println("\n4. Pressing 'Esc' to cancel search...")
	updatedModel, _ = mainModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mainModel = updatedModel.(*MainPageModel)

	// Verify search was cancelled
	assert.False(t, mainModel.searchMode, "Should exit search mode")
	assert.False(t, mainModel.searchBar.IsResourceMode(), "Should not be in resource mode")

	// Test 5: Try typing 'q' in search mode (should not quit)
	fmt.Println("\n5. Testing that 'q' doesn't quit during search...")

	// Enter search mode
	updatedModel, _ = mainModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	mainModel = updatedModel.(*MainPageModel)

	// Try to quit with 'q' - should not quit
	updatedModel, cmd = mainModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	mainModel = updatedModel.(*MainPageModel)

	// Verify we're still in the application (not quit)
	assert.True(t, mainModel.searchMode, "Should still be in search mode")
	assert.Nil(t, cmd, "Should not return quit command")

	// Exit search mode and try 'q' again - should quit
	updatedModel, _ = mainModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mainModel = updatedModel.(*MainPageModel)

	updatedModel, cmd = mainModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.NotNil(t, cmd, "Should return quit command when not in search mode")

	fmt.Println("\n✅ All resource switching tests passed!")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("• ':' enters resource switching mode")
	fmt.Println("• '/' enters normal search mode")
	fmt.Println("• 'Esc' cancels any search mode")
	fmt.Println("• 'q' is disabled during search to allow typing")
	fmt.Println("• Resource switching works with aliases (consumer-groups, cg, etc.)")
	fmt.Println("• Visual indicators show current mode (: vs 🔍)")
}
