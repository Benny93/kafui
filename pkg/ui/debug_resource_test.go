package ui

import (
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestDebugResourceSwitching(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}

	// Create main page model
	mainModel := NewMainPage(mockDS)
	mainModel.width = 120
	mainModel.height = 40

	// Create UI model
	uiModel := Model{
		dataSource:  mockDS,
		currentPage: mainPage,
		mainPage:    &mainModel,
		width:       120,
		height:      40,
	}

	t.Logf("Initial resource type: %v", uiModel.mainPage.currentResource.GetType())
	t.Logf("Initial allRows count: %d", len(uiModel.mainPage.allRows))

	// Test resource switching step by step

	// Step 1: Press ':'
	updatedModel, _ := uiModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	uiModel = updatedModel.(Model)
	t.Logf("After pressing ':', searchMode: %v, resourceMode: %v",
		uiModel.mainPage.searchMode, uiModel.mainPage.searchBar.IsResourceMode())

	// Step 2: Type "consumer-groups"
	for _, char := range "consumer-groups" {
		updatedModel, _ = uiModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		uiModel = updatedModel.(Model)
	}
	t.Logf("After typing 'consumer-groups', search value: '%s'", uiModel.mainPage.searchBar.Value())

	// Step 3: Press enter
	updatedModel, cmd := uiModel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	uiModel = updatedModel.(Model)
	
	// Execute the command to trigger the resource switch message
	if cmd != nil {
		msg := cmd()
		if msg != nil {
			updatedModel, _ = uiModel.Update(msg)
			uiModel = updatedModel.(Model)
		}
	}

	t.Logf("After pressing enter:")
	t.Logf("  Current page: %v", uiModel.currentPage)
	t.Logf("  Resource type: %v", uiModel.mainPage.currentResource.GetType())
	t.Logf("  SearchMode: %v", uiModel.mainPage.searchMode)
	t.Logf("  AllRows count: %d", len(uiModel.mainPage.allRows))

	// Debug: Check if consumer groups data was loaded
	items, err := uiModel.mainPage.currentResource.GetData()
	t.Logf("Consumer groups data:")
	t.Logf("  Error: %v", err)
	t.Logf("  Items count: %d", len(items))
	if len(items) > 0 {
		for i, item := range items {
			t.Logf("  Item %d: ID=%s, Values=%v", i, item.GetID(), item.GetValues())
		}
	}

	// Verify resource switching worked
	assert.Equal(t, mainPage, uiModel.currentPage, "Should be on main page")
	assert.Equal(t, ConsumerGroupResourceType, uiModel.mainPage.currentResource.GetType(), "Should be showing consumer groups")
}
