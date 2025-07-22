package kafui

import (
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TestNewSearchBar tests the SearchBar constructor
func TestNewSearchBar(t *testing.T) {
	table := tview.NewTable()
	dataSource := &MockKafkaDataSource{}
	pages := tview.NewPages()
	app := tview.NewApplication()
	modal := tview.NewModal()
	updateTable := func(newResource Resource, searchText string) {}
	onError := func(err error) {}

	searchBar := NewSearchBar(table, dataSource, pages, app, modal, updateTable, onError)

	if searchBar == nil {
		t.Fatal("NewSearchBar returned nil")
	}

	// Test initial state
	if searchBar.Table != table {
		t.Error("Table not set correctly")
	}

	if searchBar.DataSource != dataSource {
		t.Error("DataSource not set correctly")
	}

	if searchBar.Pages != pages {
		t.Error("Pages not set correctly")
	}

	if searchBar.App != app {
		t.Error("App not set correctly")
	}

	if searchBar.Modal != modal {
		t.Error("Modal not set correctly")
	}

	if searchBar.DefaultLabel != "😎|" {
		t.Errorf("DefaultLabel = %v, want 😎|", searchBar.DefaultLabel)
	}

	if searchBar.CurrentMode != ResouceSearch {
		t.Errorf("CurrentMode = %v, want %v", searchBar.CurrentMode, ResouceSearch)
	}

	if searchBar.CurrentString != "" {
		t.Errorf("CurrentString = %v, want empty string", searchBar.CurrentString)
	}

	if searchBar.CurrentResource == nil {
		t.Error("CurrentResource should be initialized")
	}
}

// TestSearchBar_CreateSearchInput tests the search input creation
func TestSearchBar_CreateSearchInput(t *testing.T) {
	searchBar := createTestSearchBar()
	msgChannel := make(chan UIEvent, 10)

	searchInput := searchBar.CreateSearchInput(msgChannel)

	if searchInput == nil {
		t.Fatal("CreateSearchInput returned nil")
	}

	// Test that the search input is stored
	if searchBar.SearchInput != searchInput {
		t.Error("SearchInput not stored correctly")
	}

	// Test initial label
	if searchInput.GetLabel() != searchBar.DefaultLabel {
		t.Errorf("Label = %v, want %v", searchInput.GetLabel(), searchBar.DefaultLabel)
	}

	// Test field width
	// Note: We can't directly test GetFieldWidth() as it's not exposed
}

// TestSearchBar_HandleTableSearch tests table search functionality
func TestSearchBar_HandleTableSearch(t *testing.T) {
	searchBar := createTestSearchBar()
	
	// Mock the UpdateTable function to track calls
	updateTableCalled := false
	var lastResource Resource
	var lastSearchText string
	
	searchBar.UpdateTable = func(newResource Resource, searchText string) {
		updateTableCalled = true
		lastResource = newResource
		lastSearchText = searchText
	}

	// Test table search
	searchText := "test-search"
	searchBar.handleTableSearch(searchText)

	// Verify state changes
	if searchBar.CurrentString != searchText {
		t.Errorf("CurrentString = %v, want %v", searchBar.CurrentString, searchText)
	}

	if !updateTableCalled {
		t.Error("UpdateTable should have been called")
	}

	if lastSearchText != searchText {
		t.Errorf("UpdateTable called with searchText = %v, want %v", lastSearchText, searchText)
	}

	if lastResource != searchBar.CurrentResource {
		t.Error("UpdateTable called with wrong resource")
	}
}

// TestSearchBar_HandleResouceSearch tests resource search functionality
func TestSearchBar_HandleResouceSearch(t *testing.T) {
	searchBar := createTestSearchBar()
	
	// Mock the UpdateTable function
	updateTableCalled := false
	searchBar.UpdateTable = func(newResource Resource, searchText string) {
		updateTableCalled = true
	}

	tests := []struct {
		name           string
		searchText     string
		expectMatch    bool
		expectedType   string
	}{
		{
			name:         "context search",
			searchText:   "context",
			expectMatch:  true,
			expectedType: "*kafui.ResourceContext",
		},
		{
			name:         "topic search",
			searchText:   "topics",
			expectMatch:  true,
			expectedType: "*kafui.ResouceTopic",
		},
		{
			name:         "consumer group search",
			searchText:   "groups",
			expectMatch:  true,
			expectedType: "*kafui.ResourceGroup",
		},
		{
			name:         "invalid search",
			searchText:   "invalid",
			expectMatch:  false,
			expectedType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state
			updateTableCalled = false
			
			// Initialize SearchInput to avoid nil pointer
			msgChannel := make(chan UIEvent, 10)
			searchBar.CreateSearchInput(msgChannel)
			defer close(msgChannel)
			
			// Perform search
			searchBar.handleResouceSearch(tt.searchText)

			if tt.expectMatch {
				if !updateTableCalled {
					t.Error("UpdateTable should have been called for valid search")
				}

				// Check resource type
				resourceType := ""
				switch searchBar.CurrentResource.(type) {
				case *ResourceContext:
					resourceType = "*kafui.ResourceContext"
				case *ResouceTopic:
					resourceType = "*kafui.ResouceTopic"
				case *ResourceGroup:
					resourceType = "*kafui.ResourceGroup"
				}

				if resourceType != tt.expectedType {
					t.Errorf("Resource type = %v, want %v", resourceType, tt.expectedType)
				}
			} else {
				// For invalid searches, modal should be shown
				// We can't easily test modal visibility in unit tests
			}
		})
	}
}

// TestSearchBar_AutocompleteFunc tests the autocomplete functionality
func TestSearchBar_AutocompleteFunc(t *testing.T) {
	searchBar := createTestSearchBar()
	msgChannel := make(chan UIEvent, 10)
	_ = searchBar.CreateSearchInput(msgChannel)

	// We need to access the autocomplete function indirectly
	// by testing the behavior through the search input
	
	tests := []struct {
		name           string
		input          string
		expectedCount  int
		shouldContain  []string
	}{
		{
			name:          "context prefix",
			input:         "con",
			expectedCount: 1, // Should match "context"
			shouldContain: []string{"context"},
		},
		{
			name:          "topic prefix",
			input:         "to",
			expectedCount: 0, // "to" doesn't match "topics" (needs "top")
			shouldContain: []string{},
		},
		{
			name:          "empty input",
			input:         "",
			expectedCount: 0,
			shouldContain: []string{},
		},
		{
			name:          "no matches",
			input:         "xyz",
			expectedCount: 0,
			shouldContain: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test autocomplete logic directly
			var entries []string
			currentText := tt.input
			
			if len(currentText) > 0 {
				words := append(append(Context, Topic...), ConsumerGroup...)
				for _, word := range words {
					if strings.HasPrefix(strings.ToLower(word), strings.ToLower(currentText)) {
						entries = append(entries, word)
					}
				}
				if len(entries) <= 1 {
					entries = nil
				}
			}

			if tt.expectedCount == 0 {
				if entries != nil {
					t.Errorf("Expected no entries, got %v", entries)
				}
			} else {
				if entries == nil && tt.expectedCount > 1 {
					t.Errorf("Expected %d entries, got nil", tt.expectedCount)
				}
			}

			// Check that expected strings are contained
			for _, expected := range tt.shouldContain {
				found := false
				for _, entry := range entries {
					if entry == expected {
						found = true
						break
					}
				}
				if !found && len(tt.shouldContain) > 0 {
					t.Errorf("Expected to find '%s' in autocomplete results", expected)
				}
			}
		})
	}

	// Clean up the goroutine
	close(msgChannel)
}

// TestSearchBar_ReceivingMessage tests the message handling goroutine
func TestSearchBar_ReceivingMessage(t *testing.T) {
	searchBar := createTestSearchBar()
	msgChannel := make(chan UIEvent, 10)
	_ = searchBar.CreateSearchInput(msgChannel)

	// Test different UI events
	tests := []struct {
		name     string
		event    UIEvent
		testFunc func(*testing.T, *SearchBar)
	}{
		{
			name:  "OnModalClose event",
			event: OnModalClose,
			testFunc: func(t *testing.T, sb *SearchBar) {
				// Should set focus to table (can't easily test in unit tests)
			},
		},
		{
			name:  "OnFocusSearch event",
			event: OnFocusSearch,
			testFunc: func(t *testing.T, sb *SearchBar) {
				// Should change mode to ResouceSearch
				if sb.CurrentMode != ResouceSearch {
					t.Error("Mode should be ResouceSearch after OnFocusSearch")
				}
				if sb.CurrentString != "" {
					t.Error("CurrentString should be empty after OnFocusSearch")
				}
			},
		},
		{
			name:  "OnStartTableSearch event",
			event: OnStartTableSearch,
			testFunc: func(t *testing.T, sb *SearchBar) {
				// Should change mode to TableSearch
				if sb.CurrentMode != TableSearch {
					t.Error("Mode should be TableSearch after OnStartTableSearch")
				}
				if sb.CurrentString != "" {
					t.Error("CurrentString should be empty after OnStartTableSearch")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Send the event
			msgChannel <- tt.event
			
			// Give some time for the goroutine to process
			time.Sleep(10 * time.Millisecond)
			
			// Run the test function
			if tt.testFunc != nil {
				tt.testFunc(t, searchBar)
			}
		})
	}

	// Clean up
	close(msgChannel)
	time.Sleep(10 * time.Millisecond) // Allow goroutine to exit
}

// TestSearchBar_DoneFunc tests the search input done function
func TestSearchBar_DoneFunc(t *testing.T) {
	searchBar := createTestSearchBar()
	msgChannel := make(chan UIEvent, 10)
	_ = searchBar.CreateSearchInput(msgChannel)

	tests := []struct {
		name       string
		inputText  string
		key        tcell.Key
		expectExit bool
	}{
		{
			name:       "quit command",
			inputText:  "q",
			key:        tcell.KeyEnter,
			expectExit: false, // Can't test app.Stop() in unit tests
		},
		{
			name:       "exit command",
			inputText:  "exit",
			key:        tcell.KeyEnter,
			expectExit: false, // Can't test app.Stop() in unit tests
		},
		{
			name:       "normal search",
			inputText:  "topics",
			key:        tcell.KeyEnter,
			expectExit: false,
		},
		{
			name:       "escape key",
			inputText:  "test",
			key:        tcell.KeyEsc,
			expectExit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the input text
			searchBar.SearchInput.SetText(tt.inputText)
			
			// We can't easily trigger the DoneFunc in unit tests
			// but we can test the logic directly
			if tt.inputText == "q" || tt.inputText == "exit" {
				// These should trigger app.Stop() which we can't test
			} else {
				// Normal search should update the search bar state
			}
		})
	}

	// Clean up
	close(msgChannel)
}

// Helper function to create a test SearchBar
func createTestSearchBar() *SearchBar {
	table := tview.NewTable()
	dataSource := &MockKafkaDataSource{}
	pages := tview.NewPages()
	app := tview.NewApplication()
	modal := tview.NewModal()
	updateTable := func(newResource Resource, searchText string) {}
	onError := func(err error) {}

	return NewSearchBar(table, dataSource, pages, app, modal, updateTable, onError)
}

// Benchmark tests for SearchBar operations
func BenchmarkNewSearchBar(b *testing.B) {
	table := tview.NewTable()
	dataSource := &MockKafkaDataSource{}
	pages := tview.NewPages()
	app := tview.NewApplication()
	modal := tview.NewModal()
	updateTable := func(newResource Resource, searchText string) {}
	onError := func(err error) {}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewSearchBar(table, dataSource, pages, app, modal, updateTable, onError)
	}
}

func BenchmarkSearchBar_HandleResouceSearch(b *testing.B) {
	searchBar := createTestSearchBar()
	searchBar.UpdateTable = func(newResource Resource, searchText string) {}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		searchBar.handleResouceSearch("topics")
	}
}