package resource_detail

import (
	"testing"

	"github.com/stretchr/testify/assert"
	tea "github.com/charmbracelet/bubbletea"
)

// MockResourceItem is a mock implementation for testing
type MockResourceItem struct {
	id      string
	values  []string
	details map[string]string
}

func (m *MockResourceItem) GetID() string {
	return m.id
}

func (m *MockResourceItem) GetValues() []string {
	return m.values
}

func (m *MockResourceItem) GetDetails() map[string]string {
	return m.details
}

func TestNewModel(t *testing.T) {
	// Create mock resource item
	mockItem := &MockResourceItem{
		id:      "test-resource",
		values:  []string{"test-resource", "active", "2"},
		details: map[string]string{
			"Name":   "test-resource",
			"Status": "active",
			"Count":  "2",
		},
	}
	
	// Create new model
	model := NewModel(mockItem, "consumer-group")
	
	// Verify model is properly initialized
	assert.NotNil(t, model)
	assert.Equal(t, mockItem, model.resourceItem)
	assert.Equal(t, "consumer-group", model.resourceType)
	assert.NotNil(t, model.handlers)
	assert.NotNil(t, model.keys)
	assert.NotNil(t, model.view)
	assert.Nil(t, model.error)
}

func TestModelImplementsPageInterface(t *testing.T) {
	// Create mock resource item
	mockItem := &MockResourceItem{
		id:      "test-resource",
		values:  []string{"test-resource", "active"},
		details: map[string]string{
			"Name":   "test-resource",
			"Status": "active",
		},
	}
	
	// Create new model
	model := NewModel(mockItem, "topic")
	
	// Test that model implements the Page interface methods
	assert.Equal(t, "resource_detail", model.GetID())
	
	// Test Init returns nil (no initialization needed)
	cmd := model.Init()
	assert.Nil(t, cmd)
	
	// Test SetDimensions
	model.SetDimensions(80, 24)
	assert.Equal(t, 80, model.dimensions.Width)
	assert.Equal(t, 24, model.dimensions.Height)
	
	// Test View returns a string (basic test)
	model.SetDimensions(80, 24) // Ensure dimensions are set
	view := model.View()
	assert.IsType(t, "", view)
	assert.Contains(t, view, "test-resource")
	assert.Contains(t, view, "TOPIC") // Resource type should be uppercase
}

func TestGetResourceDetails(t *testing.T) {
	// Test with valid resource item
	mockItem := &MockResourceItem{
		id:      "test-topic",
		values:  []string{"test-topic", "3", "2", "100"},
		details: map[string]string{
			"Name":        "test-topic",
			"Partitions":  "3",
			"Replication": "2",
			"Messages":    "100",
		},
	}
	
	model := NewModel(mockItem, "topic")
	details := model.GetResourceDetails()
	
	assert.Equal(t, mockItem.details, details)
	assert.Equal(t, "test-topic", details["Name"])
	assert.Equal(t, "3", details["Partitions"])
	assert.Equal(t, "2", details["Replication"])
	assert.Equal(t, "100", details["Messages"])
	
	// Test with nil resource item
	modelWithNil := &Model{resourceItem: nil}
	details = modelWithNil.GetResourceDetails()
	assert.Equal(t, map[string]string{"Error": "No resource item"}, details)
}

func TestGetResourceValues(t *testing.T) {
	// Test with valid resource item
	mockItem := &MockResourceItem{
		id:     "test-group",
		values: []string{"test-group", "Stable", "3"},
	}
	
	model := NewModel(mockItem, "consumer-group")
	values := model.GetResourceValues()
	
	assert.Equal(t, mockItem.values, values)
	assert.Equal(t, []string{"test-group", "Stable", "3"}, values)
	
	// Test with nil resource item
	modelWithNil := &Model{resourceItem: nil}
	values = modelWithNil.GetResourceValues()
	assert.Equal(t, []string{"No resource"}, values)
}

func TestGetResourceID(t *testing.T) {
	// Test with valid resource item
	mockItem := &MockResourceItem{
		id: "test-schema",
	}
	
	model := NewModel(mockItem, "schema")
	id := model.GetResourceID()
	
	assert.Equal(t, "test-schema", id)
	
	// Test with nil resource item
	modelWithNil := &Model{resourceItem: nil}
	id = modelWithNil.GetResourceID()
	assert.Equal(t, "Unknown", id)
}

func TestWindowSizeUpdate(t *testing.T) {
	mockItem := &MockResourceItem{
		id:     "test-resource",
		values: []string{"test-resource", "active"},
	}
	
	model := NewModel(mockItem, "context")
	
	// Test window size message
	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	updatedModel, cmd := model.Update(msg)
	
	// Should return the same model type
	assert.IsType(t, &Model{}, updatedModel)
	assert.Nil(t, cmd) // Window size updates don't return commands
	
	// Check dimensions were updated
	updatedResourceModel := updatedModel.(*Model)
	assert.Equal(t, 100, updatedResourceModel.dimensions.Width)
	assert.Equal(t, 30, updatedResourceModel.dimensions.Height)
}

func TestKeyHandling(t *testing.T) {
	mockItem := &MockResourceItem{
		id:     "test-resource",
		values: []string{"test-resource", "active"},
	}
	
	model := NewModel(mockItem, "context")
	
	// Test back key (should trigger page change)
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, cmd := model.Update(msg)
	
	// Should return the same model type
	assert.IsType(t, &Model{}, updatedModel)
	assert.NotNil(t, cmd) // Back navigation should return a command
	
	// Test quit key
	msg = tea.KeyMsg{Type: tea.KeyCtrlC}
	updatedModel, cmd = model.Update(msg)
	
	// Should return the same model type
	assert.IsType(t, &Model{}, updatedModel)
	assert.NotNil(t, cmd) // Quit should return tea.Quit command
}

func TestViewRendering(t *testing.T) {
	testCases := []struct {
		name         string
		resourceType string
		resourceItem *MockResourceItem
		expectedText []string
	}{
		{
			name:         "Topic resource",
			resourceType: "topic",
			resourceItem: &MockResourceItem{
				id: "my-topic",
				details: map[string]string{
					"Name":        "my-topic",
					"Partitions":  "5",
					"Replication": "3",
				},
			},
			expectedText: []string{"Resource Details - TOPIC", "ID: my-topic", "Name: my-topic", "Partitions: 5"},
		},
		{
			name:         "Consumer Group resource",
			resourceType: "consumer-group",
			resourceItem: &MockResourceItem{
				id: "my-group",
				details: map[string]string{
					"Name":      "my-group",
					"State":     "Stable",
					"Consumers": "2",
				},
			},
			expectedText: []string{"Resource Details - CONSUMER-GROUP", "ID: my-group", "State: Stable", "Consumers: 2"},
		},
		{
			name:         "Zero dimensions",
			resourceType: "topic",
			resourceItem: &MockResourceItem{id: "test"},
			expectedText: []string{"Loading resource details..."},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			model := NewModel(tc.resourceItem, tc.resourceType)
			
			if tc.name != "Zero dimensions" {
				model.SetDimensions(80, 24) // Set proper dimensions
			}
			
			view := model.View()
			
			for _, expected := range tc.expectedText {
				assert.Contains(t, view, expected, "Expected text '%s' not found in view", expected)
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	// Test model with error state
	model := &Model{
		resourceItem: nil,
		resourceType: "test",
		error:        assert.AnError,
	}
	model.handlers = NewHandlers(model)
	model.keys = NewKeys()
	model.view = NewView()
	
	// Test that methods handle nil resource gracefully
	assert.Equal(t, "Unknown", model.GetResourceID())
	assert.Equal(t, []string{"No resource"}, model.GetResourceValues())
	assert.Equal(t, map[string]string{"Error": "No resource item"}, model.GetResourceDetails())
	
	// Test view with zero dimensions
	view := model.View()
	assert.Contains(t, view, "Loading resource details...")
}