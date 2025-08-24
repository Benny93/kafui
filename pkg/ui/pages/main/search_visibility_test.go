package mainpage

import (
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestSearchBarVisibility(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	// Create new model
	model := NewModel(mockDS)

	// Set dimensions
	model.SetDimensions(100, 30)

	// Initially should not be in search mode
	assert.False(t, model.searchMode)
	
	// Update the view to reflect the initial state
	view := NewView()
	view.SetDimensions(100, 30)
	view.updateComponents(model)
	
	// Check that main content is configured to not show search bar
	config := model.mainContent.GetConfig()
	assert.False(t, config.ShowSearch)

	// Enable search mode
	model.searchMode = true

	// Update the view to reflect the change
	view.updateComponents(model)

	// Check that main content is now configured to show search bar
	config = model.mainContent.GetConfig()
	assert.True(t, config.ShowSearch)

	// Disable search mode
	model.searchMode = false

	// Update the view to reflect the change
	view.updateComponents(model)

	// Check that main content is now configured to not show search bar
	config = model.mainContent.GetConfig()
	assert.False(t, config.ShowSearch)
}

func TestSearchModeKeyHandling(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	// Create new model
	model := NewModel(mockDS)

	// Enable search mode
	model.searchMode = true

	// Test that Enter key confirms search/resource switch
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)

	// Should stay in the same model
	assert.IsType(t, &Model{}, updatedModel)

	// Should exit search mode
	updatedMainModel := updatedModel.(*Model)
	assert.False(t, updatedMainModel.searchMode)
}