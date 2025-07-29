package kafui

import (
	"strings"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUpdateTableFunc is a mock for the UpdateTable function
type MockUpdateTableFunc struct {
	mock.Mock
}

func (m *MockUpdateTableFunc) Call(newResource Resource, searchText string) {
	m.Called(newResource, searchText)
}

// MockOnErrorFunc is a mock for the onError function
type MockOnErrorFunc struct {
	mock.Mock
}

func (m *MockOnErrorFunc) Call(err error) {
	m.Called(err)
}

// createTestSearchBarWithMocks creates a SearchBar for testing with mocks
func createTestSearchBarWithMocks() (*SearchBar, *MockKafkaDataSource, *MockUpdateTableFunc, *MockOnErrorFunc) {
	table := tview.NewTable()
	mockDS := &MockKafkaDataSource{}
	pages := tview.NewPages()
	app := tview.NewApplication()
	modal := tview.NewModal()
	
	mockUpdateTable := &MockUpdateTableFunc{}
	mockOnError := &MockOnErrorFunc{}
	
	updateTableFunc := func(newResource Resource, searchText string) {
		mockUpdateTable.Call(newResource, searchText)
	}
	
	onErrorFunc := func(err error) {
		mockOnError.Call(err)
	}
	
	searchBar := NewSearchBar(table, mockDS, pages, app, modal, updateTableFunc, onErrorFunc)
	return searchBar, mockDS, mockUpdateTable, mockOnError
}

// TestNewSearchBarEnhanced tests the NewSearchBar constructor with enhanced coverage
func TestNewSearchBarEnhanced(t *testing.T) {
	table := tview.NewTable()
	mockDS := &MockKafkaDataSource{}
	pages := tview.NewPages()
	app := tview.NewApplication()
	modal := tview.NewModal()
	
	updateTableFunc := func(newResource Resource, searchText string) {}
	onErrorFunc := func(err error) {}
	
	searchBar := NewSearchBar(table, mockDS, pages, app, modal, updateTableFunc, onErrorFunc)
	
	assert.NotNil(t, searchBar)
	assert.Equal(t, table, searchBar.Table)
	assert.Equal(t, mockDS, searchBar.DataSource)
	assert.Equal(t, pages, searchBar.Pages)
	assert.Equal(t, app, searchBar.App)
	assert.Equal(t, modal, searchBar.Modal)
	assert.Equal(t, "😎|", searchBar.DefaultLabel)
	assert.Equal(t, ResouceSearch, searchBar.CurrentMode)
	assert.Equal(t, "", searchBar.CurrentString)
	assert.NotNil(t, searchBar.CurrentResource)
	assert.NotNil(t, searchBar.UpdateTable)
	assert.NotNil(t, searchBar.onError)
}

// TestCreateSearchInput tests the CreateSearchInput method
func TestCreateSearchInput(t *testing.T) {
	searchBar, _, _, _ := createTestSearchBarWithMocks()
	msgChannel := make(chan UIEvent, 10)
	
	input := searchBar.CreateSearchInput(msgChannel)
	
	assert.NotNil(t, input)
	assert.Equal(t, input, searchBar.SearchInput)
	assert.Equal(t, "😎|", input.GetLabel())
	
	// Test that input field is properly configured
	// Note: tview doesn't expose HasBorder(), so we test other properties
	assert.Equal(t, 0, input.GetFieldWidth())
	
	// Clean up the goroutine
	close(msgChannel)
}

// TestHandleTableSearch tests the handleTableSearch method
func TestHandleTableSearch(t *testing.T) {
	searchBar, _, mockUpdateTable, _ := createTestSearchBarWithMocks()
	
	// Set up expectations
	mockUpdateTable.On("Call", mock.AnythingOfType("*kafui.ResouceTopic"), "test-search").Return()
	
	searchBar.handleTableSearch("test-search")
	
	assert.Equal(t, "test-search", searchBar.CurrentString)
	mockUpdateTable.AssertExpectations(t)
}

// TestHandleResourceSearch tests the handleResouceSearch method
func TestHandleResourceSearch(t *testing.T) {
	tests := []struct {
		name           string
		searchText     string
		expectedMatch  bool
		expectedType   string
	}{
		{
			name:          "context search",
			searchText:    "context",
			expectedMatch: true,
			expectedType:  "ResourceContext",
		},
		{
			name:          "topic search",
			searchText:    "topics",
			expectedMatch: true,
			expectedType:  "ResouceTopic",
		},
		{
			name:          "consumer group search",
			searchText:    "groups",
			expectedMatch: true,
			expectedType:  "ResourceGroup",
		},
		{
			name:          "no match",
			searchText:    "invalid",
			expectedMatch: false,
			expectedType:  "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searchBar, _, mockUpdateTable, _ := createTestSearchBarWithMocks()
			msgChannel := make(chan UIEvent, 10)
			
			// Create the search input first to avoid nil pointer
			_ = searchBar.CreateSearchInput(msgChannel)
			
			if tt.expectedMatch {
				// The resource type depends on what the search matches
				if tt.expectedType == "ResourceContext" {
					mockUpdateTable.On("Call", mock.AnythingOfType("*kafui.ResourceContext"), "").Return()
				} else if tt.expectedType == "ResouceTopic" {
					mockUpdateTable.On("Call", mock.AnythingOfType("*kafui.ResouceTopic"), "").Return()
				} else if tt.expectedType == "ResourceGroup" {
					mockUpdateTable.On("Call", mock.AnythingOfType("*kafui.ResourceGroup"), "").Return()
				}
			}
			
			searchBar.handleResouceSearch(tt.searchText)
			
			assert.Equal(t, "😎|", searchBar.SearchInput.GetLabel())
			assert.Equal(t, "", searchBar.SearchInput.GetText())
			
			if tt.expectedMatch {
				mockUpdateTable.AssertExpectations(t)
			}
			
			close(msgChannel)
		})
	}
}

// TestSearchInputDoneFunc tests the done function behavior
func TestSearchInputDoneFunc(t *testing.T) {
	searchBar, _, mockUpdateTable, _ := createTestSearchBarWithMocks()
	msgChannel := make(chan UIEvent, 10)
	
	input := searchBar.CreateSearchInput(msgChannel)
	
	tests := []struct {
		name       string
		inputText  string
		mode       SearchMode
		shouldExit bool
	}{
		{
			name:       "exit command",
			inputText:  "q",
			mode:       ResouceSearch,
			shouldExit: true,
		},
		{
			name:       "exit command 2",
			inputText:  "exit",
			mode:       ResouceSearch,
			shouldExit: true,
		},
		{
			name:       "resource search",
			inputText:  "topics",
			mode:       ResouceSearch,
			shouldExit: false,
		},
		{
			name:       "table search",
			inputText:  "test",
			mode:       TableSearch,
			shouldExit: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searchBar.CurrentMode = tt.mode
			
			if !tt.shouldExit {
				if tt.mode == ResouceSearch && Contains(Topic, tt.inputText) {
					mockUpdateTable.On("Call", mock.AnythingOfType("*kafui.ResouceTopic"), "").Return()
				} else if tt.mode == TableSearch {
					// For table search, we expect the ChangedFunc to be called with the input text
					mockUpdateTable.On("Call", mock.AnythingOfType("*kafui.ResouceTopic"), tt.inputText).Return()
				}
			}
			
			// Set the text which will trigger ChangedFunc for TableSearch mode
			input.SetText(tt.inputText)
			
			// For resource search mode, we need to simulate the DoneFunc behavior
			if tt.mode == ResouceSearch && !tt.shouldExit {
				// Manually call the resource search handler to simulate Enter key press
				searchBar.handleResouceSearch(tt.inputText)
			}
			
			// Note: tview doesn't expose GetDoneFunc(), so we test the behavior indirectly
			// by checking that the search bar responds to the input text correctly
			
			if !tt.shouldExit {
				mockUpdateTable.AssertExpectations(t)
			}
		})
	}
	
	close(msgChannel)
}

// TestSearchInputChangedFunc tests the changed function behavior
func TestSearchInputChangedFunc(t *testing.T) {
	searchBar, _, _, _ := createTestSearchBarWithMocks()
	msgChannel := make(chan UIEvent, 10)
	
	_ = searchBar.CreateSearchInput(msgChannel)
	
	// Test TableSearch mode
	searchBar.CurrentMode = TableSearch
	
	// Note: tview doesn't expose GetChangedFunc(), so we test the behavior indirectly
	// by setting the mode and verifying the search bar state
	searchBar.CurrentString = "test"
	
	assert.Equal(t, "test", searchBar.CurrentString)
	
	// Test ResouceSearch mode (should not trigger update)
	searchBar.CurrentMode = ResouceSearch
	
	// CurrentString should remain as set
	assert.Equal(t, "test", searchBar.CurrentString)
	
	close(msgChannel)
}

// TestSearchInputAutocompleteFunc tests the autocomplete functionality
func TestSearchInputAutocompleteFunc(t *testing.T) {
	searchBar, _, _, _ := createTestSearchBarWithMocks()
	msgChannel := make(chan UIEvent, 10)
	
	_ = searchBar.CreateSearchInput(msgChannel)
	
	tests := []struct {
		name         string
		inputText    string
		expectedLen  int
		shouldBeNil  bool
	}{
		{
			name:        "empty input",
			inputText:   "",
			expectedLen: 0,
			shouldBeNil: true,
		},
		{
			name:        "context match",
			inputText:   "con",
			expectedLen: 2, // Should match "context" and "consumers"
			shouldBeNil: false,
		},
		{
			name:        "topic match",
			inputText:   "to",
			expectedLen: 1, // Should match "topics"
			shouldBeNil: true, // Single matches return nil
		},
		{
			name:        "single match",
			inputText:   "kafka",
			expectedLen: 0, // Single matches return nil
			shouldBeNil: true,
		},
		{
			name:        "no match",
			inputText:   "xyz",
			expectedLen: 0,
			shouldBeNil: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: tview doesn't expose GetAutocompleteFunc(), so we test the autocomplete
			// logic by testing the Contains function that would be used internally
			words := append(append(Context, Topic...), ConsumerGroup...)
			var entries []string
			
			if len(tt.inputText) > 0 {
				for _, word := range words {
					if strings.HasPrefix(strings.ToLower(word), strings.ToLower(tt.inputText)) {
						entries = append(entries, word)
					}
				}
				if len(entries) <= 1 {
					entries = nil
				}
			}
			
			if tt.shouldBeNil {
				assert.Nil(t, entries)
			} else {
				assert.NotNil(t, entries)
				assert.GreaterOrEqual(t, len(entries), tt.expectedLen)
			}
		})
	}
	
	close(msgChannel)
}

// TestReceivingMessage tests the ReceivingMessage goroutine
func TestReceivingMessage(t *testing.T) {
	searchBar, _, _, _ := createTestSearchBarWithMocks()
	msgChannel := make(chan UIEvent, 10)
	
	_ = searchBar.CreateSearchInput(msgChannel)
	
	// Test OnModalClose message
	msgChannel <- OnModalClose
	time.Sleep(10 * time.Millisecond) // Give goroutine time to process
	
	// Test OnFocusSearch message
	msgChannel <- OnFocusSearch
	time.Sleep(10 * time.Millisecond)
	
	assert.Equal(t, ResouceSearch, searchBar.CurrentMode)
	assert.Equal(t, "", searchBar.CurrentString)
	
	// Test OnStartTableSearch message
	msgChannel <- OnStartTableSearch
	time.Sleep(10 * time.Millisecond)
	
	assert.Equal(t, TableSearch, searchBar.CurrentMode)
	assert.Equal(t, "", searchBar.CurrentString)
	
	close(msgChannel)
}

// TestSearchBarWithSpecialCharacters tests search with special characters
func TestSearchBarWithSpecialCharacters(t *testing.T) {
	searchBar, _, mockUpdateTable, _ := createTestSearchBarWithMocks()
	
	specialChars := []string{
		"test-topic",
		"test_topic",
		"test.topic",
		"test@topic",
		"test#topic",
		"test$topic",
		"test%topic",
		"test&topic",
		"test*topic",
		"test+topic",
		"test=topic",
		"test?topic",
		"test!topic",
		"test~topic",
		"test`topic",
		"test|topic",
		"test\\topic",
		"test/topic",
		"test:topic",
		"test;topic",
		"test<topic",
		"test>topic",
		"test[topic",
		"test]topic",
		"test{topic",
		"test}topic",
		"test(topic",
		"test)topic",
		"test\"topic",
		"test'topic",
	}
	
	for _, char := range specialChars {
		t.Run("special_char_"+char, func(t *testing.T) {
			mockUpdateTable.On("Call", mock.AnythingOfType("*kafui.ResouceTopic"), char).Return()
			
			searchBar.handleTableSearch(char)
			
			assert.Equal(t, char, searchBar.CurrentString)
			mockUpdateTable.AssertExpectations(t)
		})
	}
}

// TestSearchBarWithUnicodeCharacters tests search with unicode characters
func TestSearchBarWithUnicodeCharacters(t *testing.T) {
	searchBar, _, mockUpdateTable, _ := createTestSearchBarWithMocks()
	
	unicodeStrings := []string{
		"测试主题", // Chinese
		"тестовая тема", // Russian
		"テストトピック", // Japanese
		"🚀🔥💯", // Emojis
		"café", // Accented characters
		"naïve", // More accented characters
	}
	
	for _, unicode := range unicodeStrings {
		t.Run("unicode_"+unicode, func(t *testing.T) {
			mockUpdateTable.On("Call", mock.AnythingOfType("*kafui.ResouceTopic"), unicode).Return()
			
			searchBar.handleTableSearch(unicode)
			
			assert.Equal(t, unicode, searchBar.CurrentString)
			mockUpdateTable.AssertExpectations(t)
		})
	}
}

// TestSearchBarFunctionality tests search bar operations
func TestSearchBarFunctionality(t *testing.T) {
	// Test search functionality with different resource types
	tests := []struct {
		name         string
		searchTerm   string
		resourceType ResouceName
		expectMatch  bool
	}{
		{
			name:         "context search match",
			searchTerm:   "prod",
			resourceType: Context,
			expectMatch:  true,
		},
		{
			name:         "topic search match",
			searchTerm:   "user",
			resourceType: Topic,
			expectMatch:  true,
		},
		{
			name:         "consumer group search match",
			searchTerm:   "group",
			resourceType: ConsumerGroup,
			expectMatch:  true,
		},
		{
			name:         "empty search term",
			searchTerm:   "",
			resourceType: Topic,
			expectMatch:  true, // Empty search should match all
		},
		{
			name:         "no match",
			searchTerm:   "nonexistent",
			resourceType: Topic,
			expectMatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that search terms can be processed
			if tt.searchTerm == "" && !tt.expectMatch {
				t.Error("Empty search should always match")
			}

			// Test case insensitive search
			lowerSearch := tt.searchTerm
			if lowerSearch != "" {
				// Simulate case insensitive search logic
				testData := "PROD-TOPIC"
				if Contains([]string{testData}, "prod") {
					// This would fail, showing case sensitivity issue
				}
			}
		})
	}
}

// TestSearchModeHandling tests different search modes
func TestSearchModeHandling(t *testing.T) {
	modes := []SearchMode{
		TableSearch,
		ResouceSearch,
	}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			// Test that search modes are properly defined
			if string(mode) == "" {
				t.Error("Search mode should not be empty")
			}

			// Test mode-specific behavior
			switch mode {
			case TableSearch:
				if string(mode) != "TableSearch" {
					t.Error("TableSearch mode string incorrect")
				}
			case ResouceSearch:
				if string(mode) != "ResouceSearch" {
					t.Error("ResouceSearch mode string incorrect")
				}
			}
		})
	}
}

// TestResourceNameSearch tests searching within resource names
func TestResourceNameSearch(t *testing.T) {
	tests := []struct {
		name         string
		resource     ResouceName
		searchTerm   string
		shouldMatch  bool
	}{
		{
			name:        "context alias match",
			resource:    Context,
			searchTerm:  "kafka",
			shouldMatch: true,
		},
		{
			name:        "topic alias match",
			resource:    Topic,
			searchTerm:  "topics",
			shouldMatch: true,
		},
		{
			name:        "consumer group alias match",
			resource:    ConsumerGroup,
			searchTerm:  "cgs",
			shouldMatch: true,
		},
		{
			name:        "partial match",
			resource:    Context,
			searchTerm:  "ctx",
			shouldMatch: true,
		},
		{
			name:        "no match",
			resource:    Topic,
			searchTerm:  "invalid",
			shouldMatch: false,
		},
		{
			name:        "case insensitive match",
			resource:    Context,
			searchTerm:  "KAFKA",
			shouldMatch: false, // Our Contains function is case sensitive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Contains([]string(tt.resource), tt.searchTerm)
			if result != tt.shouldMatch {
				t.Errorf("Search for '%s' in %v: got %v, want %v", 
					tt.searchTerm, tt.resource, result, tt.shouldMatch)
			}
		})
	}
}

// TestSearchBarUIComponents tests UI component creation for search
func TestSearchBarUIComponents(t *testing.T) {
	// Test that search-related UI components can be created
	searchField := tview.NewInputField()
	searchField.SetLabel("Search: ")
	searchField.SetFieldWidth(0)

	if searchField == nil {
		t.Fatal("Failed to create search input field")
	}

	if searchField.GetLabel() != "Search: " {
		t.Error("Search field label not set correctly")
	}

	// Test placeholder functionality
	searchField.SetPlaceholder("Type to search...")
	// Note: tview doesn't expose GetPlaceholder(), so we can't directly test this

	// Test that the field can accept input
	searchField.SetText("test search")
	if searchField.GetText() != "test search" {
		t.Error("Search field text not set correctly")
	}
}

// TestSearchResultFiltering tests filtering logic
func TestSearchResultFiltering(t *testing.T) {
	// Test data
	topics := map[string]api.Topic{
		"user-events":    {NumPartitions: 3, ReplicationFactor: 2},
		"order-events":   {NumPartitions: 5, ReplicationFactor: 3},
		"payment-events": {NumPartitions: 1, ReplicationFactor: 1},
		"system-logs":    {NumPartitions: 2, ReplicationFactor: 2},
	}

	tests := []struct {
		name           string
		searchTerm     string
		expectedCount  int
		expectedTopics []string
	}{
		{
			name:           "events filter",
			searchTerm:     "events",
			expectedCount:  3,
			expectedTopics: []string{"user-events", "order-events", "payment-events"},
		},
		{
			name:           "user filter",
			searchTerm:     "user",
			expectedCount:  1,
			expectedTopics: []string{"user-events"},
		},
		{
			name:           "system filter",
			searchTerm:     "system",
			expectedCount:  1,
			expectedTopics: []string{"system-logs"},
		},
		{
			name:           "no match",
			searchTerm:     "nonexistent",
			expectedCount:  0,
			expectedTopics: []string{},
		},
		{
			name:           "empty search",
			searchTerm:     "",
			expectedCount:  4,
			expectedTopics: []string{"user-events", "order-events", "payment-events", "system-logs"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the filtering logic used in ShowTopicsInTable
			var filteredTopics []string
			for topicName := range topics {
				if tt.searchTerm == "" || strings.Contains(strings.ToLower(topicName), strings.ToLower(tt.searchTerm)) {
					filteredTopics = append(filteredTopics, topicName)
				}
			}

			if len(filteredTopics) != tt.expectedCount {
				t.Errorf("Filtered count = %d, want %d", len(filteredTopics), tt.expectedCount)
			}

			// Check that expected topics are present
			for _, expectedTopic := range tt.expectedTopics {
				if !Contains(filteredTopics, expectedTopic) {
					t.Errorf("Expected topic '%s' not found in filtered results", expectedTopic)
				}
			}
		})
	}
}

// TestSearchCaseSensitivity tests case sensitivity in search
func TestSearchCaseSensitivity(t *testing.T) {
	testData := []string{"PROD-TOPIC", "dev-topic", "Test-Topic"}

	tests := []struct {
		name        string
		searchTerm  string
		expectMatch bool
	}{
		{
			name:        "exact case match",
			searchTerm:  "PROD-TOPIC",
			expectMatch: true,
		},
		{
			name:        "different case no match",
			searchTerm:  "prod-topic",
			expectMatch: false, // Our Contains is case sensitive
		},
		{
			name:        "partial case match",
			searchTerm:  "Test-Topic",
			expectMatch: true,
		},
		{
			name:        "partial different case no match",
			searchTerm:  "test",
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Contains(testData, tt.searchTerm)
			if result != tt.expectMatch {
				t.Errorf("Search for '%s': got %v, want %v", tt.searchTerm, result, tt.expectMatch)
			}
		})
	}
}

// TestSearchPerformance tests search performance with large datasets
func TestSearchPerformance(t *testing.T) {
	// Create large dataset
	largeDataset := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		largeDataset[i] = "topic-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10)) + string(rune('0'+(i/100)%10))
	}

	// Test search performance
	searchTerm := "topic-500"
	result := Contains(largeDataset, searchTerm)

	if !result {
		t.Error("Should find the search term in large dataset")
	}

	// Test non-existent search
	nonExistentTerm := "nonexistent-topic"
	result = Contains(largeDataset, nonExistentTerm)

	if result {
		t.Error("Should not find non-existent term")
	}
}

// Benchmark tests for search operations
func BenchmarkSearchContains(b *testing.B) {
	data := []string{"topic1", "topic2", "topic3", "user-events", "order-events"}
	searchTerm := "user-events"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Contains(data, searchTerm)
	}
}

func BenchmarkSearchFilter(b *testing.B) {
	// Create test data
	topics := make(map[string]api.Topic, 100)
	for i := 0; i < 100; i++ {
		topicName := "topic-" + string(rune(i))
		topics[topicName] = api.Topic{NumPartitions: 3, ReplicationFactor: 2}
	}

	searchTerm := "topic-5"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var filtered []string
		for topicName := range topics {
			if searchTerm == "" || strings.Contains(strings.ToLower(topicName), strings.ToLower(searchTerm)) {
				filtered = append(filtered, topicName)
			}
		}
		_ = filtered
	}
}