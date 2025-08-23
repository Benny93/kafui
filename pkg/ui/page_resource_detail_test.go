package ui

import (
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestResourceDetailPageNavigation(t *testing.T) {
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

	// Test 1: Switch to consumer groups resource
	t.Run("SwitchToConsumerGroups", func(t *testing.T) {
		// Switch to consumer groups
		updatedModel, _ := uiModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		uiModel = updatedModel.(Model)

		// Type "consumer-groups"
		for _, char := range "consumer-groups" {
			updatedModel, _ = uiModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
			uiModel = updatedModel.(Model)
		}

		// Press enter to switch
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

		// Verify we're on main page showing consumer groups
		assert.Equal(t, mainPage, uiModel.currentPage, "Should be on main page")
		assert.Equal(t, ConsumerGroupResourceType, uiModel.mainPage.currentResource.GetType(), "Should be showing consumer groups")
	})

	// Test 2: Navigate to resource detail page
	t.Run("NavigateToResourceDetail", func(t *testing.T) {
		// Ensure we have some items and one is selected
		if len(uiModel.mainPage.allRows) > 0 {
			// Make sure we're not in search mode
			uiModel.mainPage.searchMode = false
			uiModel.mainPage.searchBar.Blur()
			
			// Ensure table has the correct rows and position cursor
			uiModel.mainPage.resourcesTable.SetRows(uiModel.mainPage.allRows)
			uiModel.mainPage.resourcesTable.GotoTop()

			// Check what getSelectedResourceItem returns
			selectedItem := uiModel.mainPage.getSelectedResourceItem()
			
			// Only proceed if we have a selected item
			if selectedItem != nil {
				// Press enter to navigate to resource detail
				updatedModel, cmd := uiModel.Update(tea.KeyMsg{Type: tea.KeyEnter})
				uiModel = updatedModel.(Model)
				
				// Execute any commands that might be returned
				if cmd != nil {
					msg := cmd()
					if msg != nil {
						updatedModel, _ = uiModel.Update(msg)
						uiModel = updatedModel.(Model)
					}
				}

				// Verify we're on resource detail page
				assert.Equal(t, resourceDetailPage, uiModel.currentPage, "Should be on resource detail page")
				assert.NotNil(t, uiModel.resourceDetailPage, "Resource detail page should be initialized")
			} else {
				t.Skip("No selected item available for navigation test")
			}
		}
	})

	// Test 3: Navigate back from resource detail page
	t.Run("NavigateBackFromResourceDetail", func(t *testing.T) {
		if uiModel.currentPage == resourceDetailPage {
			// Press escape to go back
			updatedModel, _ := uiModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
			uiModel = updatedModel.(Model)

			// Verify we're back on main page
			assert.Equal(t, mainPage, uiModel.currentPage, "Should be back on main page")
			assert.Nil(t, uiModel.resourceDetailPage, "Resource detail page should be cleaned up")
		}
	})

	// Test 4: Test resource detail page rendering
	t.Run("ResourceDetailPageRendering", func(t *testing.T) {
		if len(uiModel.mainPage.allRows) > 0 {
			// Get first resource row
			selectedRow := uiModel.mainPage.allRows[0]
			if len(selectedRow) > 0 {
				// Create a minimal resource item for testing
				resourceItem := &minimalResourceItem{id: selectedRow[0]}
				// Create resource detail page
				rdp := NewResourceDetailPage(resourceItem, ConsumerGroupResourceType)
				rdp.width = 120
				rdp.height = 40

				// Test rendering
				view := rdp.View()
				assert.NotEmpty(t, view, "Resource detail page should render content")
				assert.Contains(t, view, "Consumer Group Information", "Should contain consumer group header")
			}
		}
	})
}

func TestResourceDetailPageKeyHandling(t *testing.T) {
	// Create a mock resource item
	mockDS := &mock.KafkaDataSourceMock{}
	mainModel := NewMainPage(mockDS)

	if len(mainModel.allRows) > 0 {
		selectedRow := mainModel.allRows[0]
		if len(selectedRow) > 0 {
			// Create a minimal resource item for testing
			resourceItem := &minimalResourceItem{id: selectedRow[0]}
			// Create resource detail page
			rdp := NewResourceDetailPage(resourceItem, ConsumerGroupResourceType)
			rdp.width = 120
			rdp.height = 40

			// Test key handling
			t.Run("EscapeKey", func(t *testing.T) {
				updatedModel, cmd := rdp.Update(tea.KeyMsg{Type: tea.KeyEsc})
				assert.NotNil(t, updatedModel, "Should return updated model")
				assert.NotNil(t, cmd, "Should return command for page change")
			})

			t.Run("ScrollKeys", func(t *testing.T) {
				// Test scroll down
				updatedModel, _ := rdp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
				rdp = updatedModel.(ResourceDetailPageModel)

				// Test scroll up
				updatedModel, _ = rdp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
				rdp = updatedModel.(ResourceDetailPageModel)

				assert.NotNil(t, updatedModel, "Should handle scroll keys")
			})

			t.Run("WrapToggle", func(t *testing.T) {
				originalWrapped := rdp.wrapped
				updatedModel, _ := rdp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
				rdp = updatedModel.(ResourceDetailPageModel)

				assert.NotEqual(t, originalWrapped, rdp.wrapped, "Should toggle wrap mode")
			})
		}
	}
}
