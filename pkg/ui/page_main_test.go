package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestMainPageModelView(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		height   int
		expected []string
	}{
		{
			name:   "Normal size rendering",
			width:  120,
			height: 40,
			expected: []string{
				"Kafui - Kafka UI",
				"CONTEXT",
				"RESOURCES",
				"SHORTCUTS",
				"Selected:",
				"Last update:",
			},
		},
		{
			name:   "Small size rendering",
			width:  80,
			height: 24,
			expected: []string{
				"Kafui - Kafka UI",
				"CONTEXT",
				"RESOURCES",
			},
		},
		{
			name:   "Zero size handling",
			width:  0,
			height: 0,
			expected: []string{
				"Loading...",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock data source
			mockDS := &mock.KafkaDataSourceMock{}
			
			// Create main page model
			model := NewMainPage(mockDS)
			model.width = tt.width
			model.height = tt.height
			model.lastUpdate = time.Now()
			
			// Add some mock items to avoid empty state
			if tt.width > 0 && tt.height > 0 {
				mockItems := []list.Item{
					topicItem{name: "test-topic-1", topic: api.Topic{NumPartitions: 3, ReplicationFactor: 1}},
					topicItem{name: "test-topic-2", topic: api.Topic{NumPartitions: 5, ReplicationFactor: 2}},
				}
				model.topicList.SetItems(mockItems)
				model.allItems = mockItems
			}
			
			// Render the view
			rendered := model.View()
			
			// Use docStyle to render and print (as requested)
			doc := strings.Builder{}
			doc.WriteString(rendered)
			fmt.Println(docStyle.Render(doc.String()))
			
			// Verify expected content is present
			for _, expected := range tt.expected {
				assert.Contains(t, rendered, expected, "Expected content '%s' not found in rendered output", expected)
			}
		})
	}
}

func TestMainPageModelRenderResourceButtons(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	
	// Create main page model
	model := NewMainPage(mockDS)
	model.width = 120
	model.height = 40
	
	// Test different resource types
	resourceTypes := []ResourceType{
		TopicResourceType,
		ConsumerGroupResourceType,
		SchemaResourceType,
		ContextResourceType,
	}
	
	for _, resourceType := range resourceTypes {
		t.Run(fmt.Sprintf("Resource_%s", resourceType), func(t *testing.T) {
			// Switch to the resource type
			model.switchToResource(resourceType)
			
			// Render resource buttons
			buttons := model.renderResourceButtons()
			
			// Use docStyle to render and print
			doc := strings.Builder{}
			doc.WriteString(buttons)
			fmt.Println(docStyle.Render(doc.String()))
			
			// Verify all resource types are present
			assert.Contains(t, buttons, "F1 Topics")
			assert.Contains(t, buttons, "F2 Consumer Groups")
			assert.Contains(t, buttons, "F3 Schemas")
			assert.Contains(t, buttons, "F4 Contexts")
		})
	}
}

func TestMainPageModelRenderShortcuts(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	
	// Create main page model
	model := NewMainPage(mockDS)
	
	// Render shortcuts
	shortcuts := model.renderShortcuts()
	
	// Use docStyle to render and print
	doc := strings.Builder{}
	doc.WriteString(shortcuts)
	fmt.Println(docStyle.Render(doc.String()))
	
	// Verify expected shortcuts are present
	expectedShortcuts := []string{
		"↑/↓   Navigate items",
		"Enter   Select item",
		"/       Search",
		"Esc     Cancel search",
		"q       Quit",
	}
	
	for _, expected := range expectedShortcuts {
		assert.Contains(t, shortcuts, expected, "Expected shortcut '%s' not found", expected)
	}
}

func TestMainPageModelRenderFooter(t *testing.T) {
	tests := []struct {
		name         string
		width        int
		searchMode   bool
		selectedItem list.Item
		expected     []string
	}{
		{
			name:       "Normal footer",
			width:      120,
			searchMode: false,
			selectedItem: topicItem{
				name:  "test-topic",
				topic: api.Topic{NumPartitions: 3, ReplicationFactor: 1},
			},
			expected: []string{
				"Selected: test-topic",
				"items total",
				"Last update:",
			},
		},
		{
			name:       "Search mode footer",
			width:      120,
			searchMode: true,
			expected: []string{
				"Type to search",
				"Enter: confirm",
				"Esc: cancel",
			},
		},
		{
			name:       "Narrow width footer",
			width:      50,
			searchMode: false,
			expected: []string{
				"Last update:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock data source
			mockDS := &mock.KafkaDataSourceMock{}
			
			// Create main page model
			model := NewMainPage(mockDS)
			model.width = tt.width
			model.height = 40
			model.searchMode = tt.searchMode
			model.lastUpdate = time.Now()
			model.statusMessage = "Test status"
			
			// Set up mock items
			mockItems := []list.Item{
				topicItem{name: "test-topic-1", topic: api.Topic{NumPartitions: 3, ReplicationFactor: 1}},
				topicItem{name: "test-topic-2", topic: api.Topic{NumPartitions: 5, ReplicationFactor: 2}},
			}
			model.topicList.SetItems(mockItems)
			model.allItems = mockItems
			
			// Select an item if provided
			if tt.selectedItem != nil {
				model.topicList.Select(0)
			}
			
			// Render footer
			footer := model.renderFooter()
			
			// Use docStyle to render and print
			doc := strings.Builder{}
			doc.WriteString(footer)
			fmt.Println(docStyle.Render(doc.String()))
			
			// Verify expected content is present
			for _, expected := range tt.expected {
				assert.Contains(t, footer, expected, "Expected content '%s' not found in footer", expected)
			}
		})
	}
}

func TestMainPageModelUpdate(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	
	// Create main page model
	model := NewMainPage(mockDS)
	model.width = 120
	model.height = 40
	
	tests := []struct {
		name     string
		msg      tea.Msg
		expected string
	}{
		{
			name: "Window size message",
			msg:  tea.WindowSizeMsg{Width: 100, Height: 30},
			expected: "window size updated",
		},
		{
			name: "Search key press",
			msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}},
			expected: "search mode activated",
		},
		{
			name: "Escape key press",
			msg:  tea.KeyMsg{Type: tea.KeyEsc},
			expected: "search cancelled or normal mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Update the model
			updatedModel, cmd := model.Update(tt.msg)
			
			// Verify the model was updated
			assert.NotNil(t, updatedModel)
			
			// Render the updated view directly from the interface
			rendered := updatedModel.View()
			
			// Use docStyle to render and print
			doc := strings.Builder{}
			doc.WriteString(rendered)
			fmt.Println(docStyle.Render(doc.String()))
			
			// Verify the update had some effect (basic check)
			assert.NotEmpty(t, rendered, "Rendered view should not be empty")
			
			// Check if command was returned when expected
			switch tt.msg.(type) {
			case tea.WindowSizeMsg:
				assert.Nil(t, cmd, "Window size update should not return command")
			case tea.KeyMsg:
				// Some key presses might return commands
				// This is acceptable
			}
		})
	}
}

func TestMainPageModelSearchFunctionality(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	
	// Create main page model
	model := NewMainPage(mockDS)
	model.width = 120
	model.height = 40
	
	// Add mock items
	mockItems := []list.Item{
		topicItem{name: "user-events", topic: api.Topic{NumPartitions: 3, ReplicationFactor: 1}},
		topicItem{name: "order-processing", topic: api.Topic{NumPartitions: 5, ReplicationFactor: 2}},
		topicItem{name: "user-analytics", topic: api.Topic{NumPartitions: 2, ReplicationFactor: 1}},
	}
	model.topicList.SetItems(mockItems)
	model.allItems = mockItems
	
	// Test search functionality
	searchQuery := "user"
	updatedModel, _ := model.Update(searchTopicsMsg(searchQuery))
	
	// Render the view after search
	rendered := updatedModel.View()
	
	// Use docStyle to render and print
	doc := strings.Builder{}
	doc.WriteString(rendered)
	fmt.Println(docStyle.Render(doc.String()))
	
	// Verify search results - check if the search was applied
	// The search might filter the display, so let's check for search-related content
	assert.Contains(t, rendered, "Showing", "Should show filtered results status")
	
	// Verify the view contains search-related content
	assert.NotEmpty(t, rendered, "Rendered view should not be empty after search")
}

func TestMainPageModelResourceSwitching(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	
	// Create main page model
	model := NewMainPage(mockDS)
	model.width = 120
	model.height = 40
	
	// Test switching between different resource types
	resourceTypes := []ResourceType{
		TopicResourceType,
		ConsumerGroupResourceType,
		SchemaResourceType,
		ContextResourceType,
	}
	
	for _, resourceType := range resourceTypes {
		t.Run(fmt.Sprintf("Switch_to_%s", resourceType), func(t *testing.T) {
			// Switch to resource type
			model.switchToResource(resourceType)
			
			// Render the view
			rendered := model.View()
			
			// Use docStyle to render and print
			doc := strings.Builder{}
			doc.WriteString(rendered)
			fmt.Println(docStyle.Render(doc.String()))
			
			// Verify the resource type is reflected in the view
			assert.Contains(t, rendered, strings.ToUpper(resourceType.String()), 
				"View should show current resource type")
			
			// Verify the current resource was updated
			assert.Equal(t, resourceType, model.currentResource.GetType(), 
				"Current resource should be updated")
		})
	}
}

func TestMainPageModelInitialization(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	
	// Create main page model
	model := NewMainPage(mockDS)
	
	// Verify initial state
	assert.NotNil(t, model.dataSource, "Data source should be set")
	assert.NotNil(t, model.topicList, "Topic list should be initialized")
	assert.NotNil(t, model.searchBar, "Search bar should be initialized")
	assert.NotNil(t, model.spinner, "Spinner should be initialized")
	assert.NotNil(t, model.resourceManager, "Resource manager should be initialized")
	assert.NotNil(t, model.currentResource, "Current resource should be set")
	assert.False(t, model.searchMode, "Search mode should be false initially")
	assert.False(t, model.loading, "Loading should be false initially")
	
	// Test initialization command
	cmd := model.Init()
	assert.NotNil(t, cmd, "Init should return a command")
	
	// Render initial view
	model.width = 120
	model.height = 40
	rendered := model.View()
	
	// Use docStyle to render and print
	doc := strings.Builder{}
	doc.WriteString(rendered)
	fmt.Println(docStyle.Render(doc.String()))
	
	// Verify initial rendering works
	assert.NotEmpty(t, rendered, "Initial view should not be empty")
	assert.Contains(t, rendered, "Kafui - Kafka UI", "Should show application title")
}