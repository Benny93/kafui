package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/components"
	"github.com/charmbracelet/bubbles/table"
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
				"CURRENT RESOURCE",
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
				"CURRENT RESOURCE",
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
				mockRows := []table.Row{
					{"test-topic-1", "topic", "3", "1", "100 messages"},
					{"test-topic-2", "topic", "5", "2", "200 messages"},
				}
				model.resourcesTable.SetRows(mockRows)
				model.allRows = mockRows
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

			// Update sidebar with proper configuration
			model.sidebar.UpdateConfig(components.SidebarConfig{
				Context:         "test-context",
				CurrentResource: components.ResourceType(resourceType),
				ShowResources:   true,
				ShowShortcuts:   false,
			})

			// Render resource buttons using the sidebar component
			buttons := model.sidebar.RenderResourceButtons()

			// Use docStyle to render and print
			doc := strings.Builder{}
			doc.WriteString(buttons)
			fmt.Println(docStyle.Render(doc.String()))

			// Verify all resource types are present (without F-keys)
			assert.Contains(t, buttons, "Topics")
			assert.Contains(t, buttons, "Consumer Groups")
			assert.Contains(t, buttons, "Schemas")
			assert.Contains(t, buttons, "Contexts")
			assert.Contains(t, buttons, "Use : to switch")
		})
	}
}

func TestMainPageModelRenderShortcuts(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}

	// Create main page model
	model := NewMainPage(mockDS)

	// Configure sidebar to show shortcuts
	model.sidebar.UpdateConfig(components.SidebarConfig{
		Context:       "test-context",
		ShowResources: false,
		ShowShortcuts: true,
	})

	// Render shortcuts using the sidebar component
	shortcuts := model.sidebar.RenderShortcuts()

	// Use docStyle to render and print
	doc := strings.Builder{}
	doc.WriteString(shortcuts)
	fmt.Println(docStyle.Render(doc.String()))

	// Verify expected shortcuts are present
	expectedShortcuts := []string{
		"↑/↓   Navigate items",
		"Enter   Select item",
		"/       Search items",
		":       Switch resource",
		"Esc     Cancel/clear",
		"q       Quit",
	}

	for _, expected := range expectedShortcuts {
		assert.Contains(t, shortcuts, expected, "Expected shortcut '%s' not found", expected)
	}
}

func TestMainPageModelRenderFooter(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		searchMode  bool
		selectedRow []string
		expected    []string
	}{
		{
			name:        "Normal footer",
			width:       120,
			searchMode:  false,
			selectedRow: []string{"test-topic", "topic", "3", "1", "Test topic"},
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

			// Set up mock table rows
			mockRows := []table.Row{
				{"test-topic-1", "topic", "3", "1", "100 messages"},
				{"test-topic-2", "topic", "5", "2", "200 messages"},
			}
			model.resourcesTable.SetRows(mockRows)
			model.allRows = mockRows

			// Select a row if provided
			if tt.selectedRow != nil {
				model.resourcesTable.GotoTop()
			}

			// Update footer configuration
			selectedItem := "None"
			if tt.selectedRow != nil && len(tt.selectedRow) > 0 {
				selectedItem = tt.selectedRow[0]
			}

			model.footer.UpdateConfig(components.FooterConfig{
				Width:         tt.width,
				SearchMode:    tt.searchMode,
				SelectedItem:  selectedItem,
				TotalItems:    len(mockRows),
				StatusMessage: "Test status",
				LastUpdate:    time.Now(),
				Spinner:       model.spinner,
			})

			// Render footer using the footer component
			footer := model.footer.Render()

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
			name:     "Window size message",
			msg:      tea.WindowSizeMsg{Width: 100, Height: 30},
			expected: "window size updated",
		},
		{
			name:     "Search key press",
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}},
			expected: "search mode activated",
		},
		{
			name:     "Escape key press",
			msg:      tea.KeyMsg{Type: tea.KeyEsc},
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

	// Add mock rows for table and corresponding items for search functionality
	mockRows := []table.Row{
		{"user-events", "topic", "3", "1", "User events topic"},
		{"order-processing", "topic", "5", "2", "Order processing topic"},
		{"user-analytics", "topic", "2", "1", "User analytics topic"},
	}
	model.resourcesTable.SetRows(mockRows)
	model.allRows = mockRows

	// Add corresponding items to allItems (this is what search functionality uses)
	mockTopic := api.Topic{
		ReplicationFactor: 1,
		ReplicaAssignment: map[int32][]int32{},
		NumPartitions:     1,
		ConfigEntries:     make(map[string]*string),
	}
	model.allItems = []interface{}{
		topicItem{name: "user-events", topic: mockTopic},
		topicItem{name: "order-processing", topic: mockTopic},
		topicItem{name: "user-analytics", topic: mockTopic},
	}

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
	// The search should either show filtered results or "No items found"
	searchApplied := strings.Contains(rendered, "Showing") || strings.Contains(rendered, "No items found for:")
	assert.True(t, searchApplied, "Should show search results status")

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
	assert.NotNil(t, model.resourcesTable, "Resources table should be initialized")
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
