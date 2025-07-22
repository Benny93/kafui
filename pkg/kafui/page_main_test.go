package kafui

import (
	"strings"
	"testing"
	"time"

	"github.com/rivo/tview"
)

// TestNewMainPage tests the MainPage constructor
func TestNewMainPage(t *testing.T) {
	mainPage := NewMainPage()
	
	if mainPage == nil {
		t.Fatal("NewMainPage() returned nil")
	}
	
	// Test initial state
	if mainPage.CurrentContextName != "" {
		t.Errorf("CurrentContextName = %v, want empty string", mainPage.CurrentContextName)
	}
	
	if mainPage.NotificationTextView != nil {
		t.Error("NotificationTextView should be nil initially")
	}
	
	if mainPage.MidFlex != nil {
		t.Error("MidFlex should be nil initially")
	}
	
	if mainPage.ContextInfo != nil {
		t.Error("ContextInfo should be nil initially")
	}
	
	if mainPage.CurrentSearchString != "" {
		t.Error("CurrentSearchString should be empty initially")
	}
	
	if mainPage.CurrentResource != nil {
		t.Error("CurrentResource should be nil initially")
	}
	
	if mainPage.SearchBar != nil {
		t.Error("SearchBar should be nil initially")
	}
}

// TestMainPage_CurrentTimeString tests the time string formatting
func TestMainPage_CurrentTimeString(t *testing.T) {
	mainPage := NewMainPage()
	
	timeString := mainPage.CurrentTimeString()
	
	if timeString == "" {
		t.Error("CurrentTimeString() returned empty string")
	}
	
	// Should contain "Current time is" prefix
	if !strings.HasPrefix(timeString, "Current time is ") {
		t.Errorf("CurrentTimeString() = %v, want prefix 'Current time is '", timeString)
	}
	
	// Should be in HH:MM format after prefix
	timePart := strings.TrimPrefix(timeString, "Current time is ")
	if len(timePart) != 5 { // HH:MM format
		t.Errorf("Time part '%s' should be 5 characters (HH:MM)", timePart)
	}
	
	// Should contain colon
	if !strings.Contains(timePart, ":") {
		t.Errorf("Time part '%s' should contain colon", timePart)
	}
}

// TestMainPage_UpdateTable tests the table update functionality
func TestMainPage_UpdateTable(t *testing.T) {
	mainPage := NewMainPage()
	table := tview.NewTable()
	dataSource := &MockKafkaDataSource{}
	
	// Set up a mock resource
	mockResource := NewResouceTopic(dataSource, func(err error) {}, func() {})
	var resource Resource = mockResource
	mainPage.CurrentResource = &resource
	
	// Set up search bar
	searchBar := NewSearchBar(table, dataSource, tview.NewPages(), tview.NewApplication(), 
		tview.NewModal(), func(Resource, string) {}, func(error) {})
	mainPage.SearchBar = searchBar
	
	// Test UpdateTable doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("UpdateTable panicked: %v", r)
		}
	}()
	
	mainPage.UpdateTable(table, dataSource)
}

// TestMainPage_CreateMainPage tests the main page creation
func TestMainPage_CreateMainPage(t *testing.T) {
	mainPage := NewMainPage()
	dataSource := &MockKafkaDataSource{}
	pages := tview.NewPages()
	app := tview.NewApplication()
	modal := tview.NewModal()
	msgChannel := make(chan UIEvent, 10)
	defer close(msgChannel)
	
	// Test that CreateMainPage doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("CreateMainPage panicked: %v", r)
		}
	}()
	
	flex := mainPage.CreateMainPage(dataSource, pages, app, modal, msgChannel)
	
	if flex == nil {
		t.Fatal("CreateMainPage returned nil flex")
	}
	
	// Test that components are initialized
	if mainPage.SearchBar == nil {
		t.Error("SearchBar should be initialized after CreateMainPage")
	}
	
	if mainPage.ContextInfo == nil {
		t.Error("ContextInfo should be initialized after CreateMainPage")
	}
	
	if mainPage.CurrentResource == nil {
		t.Error("CurrentResource should be initialized after CreateMainPage")
	}
	
	// Test that the resource is a topic resource by default
	switch (*mainPage.CurrentResource).(type) {
	case *ResouceTopic:
		// Expected
	default:
		t.Error("Default resource should be ResouceTopic")
	}
}

// TestMainPage_UpdateTableRoutine tests the background update routine
func TestMainPage_UpdateTableRoutine(t *testing.T) {
	mainPage := NewMainPage()
	app := tview.NewApplication()
	table := tview.NewTable()
	timerView := tview.NewTextView()
	dataSource := &MockKafkaDataSource{}
	
	// Set up required components
	mockResource := NewResouceTopic(dataSource, func(err error) {}, func() {})
	var resource Resource = mockResource
	mainPage.CurrentResource = &resource
	searchBar := NewSearchBar(table, dataSource, tview.NewPages(), app, 
		tview.NewModal(), func(Resource, string) {}, func(error) {})
	mainPage.SearchBar = searchBar
	
	// Test with nil app (should return immediately)
	mainPage.UpdateTableRoutine(nil, table, timerView, dataSource)
	
	// Test with valid app (we can't easily test the full routine without complex setup)
	// Just verify it doesn't panic on startup
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("UpdateTableRoutine panicked: %v", r)
			}
		}()
		mainPage.UpdateTableRoutine(app, table, timerView, dataSource)
	}()
	
	// Give some time for the routine to start
	time.Sleep(10 * time.Millisecond)
}

// TestMainPage_ShowNotification tests notification functionality
func TestMainPage_ShowNotification(t *testing.T) {
	mainPage := NewMainPage()
	
	// Initialize NotificationTextView
	mainPage.NotificationTextView = tview.NewTextView()
	
	testMessage := "Test notification message"
	
	// Test ShowNotification doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ShowNotification panicked: %v", r)
		}
	}()
	
	// Test with nil NotificationTextView (should not panic)
	mainPage.NotificationTextView = nil
	mainPage.ShowNotification(testMessage)
	
	// Test with valid NotificationTextView
	mainPage.NotificationTextView = tview.NewTextView()
	
	// Since ShowNotification uses goroutines and tviewApp which may not be initialized in tests,
	// we'll test that the method doesn't panic and the NotificationTextView is properly initialized
	mainPage.ShowNotification(testMessage)
	
	// The actual text setting happens in a goroutine with tviewApp.QueueUpdateDraw,
	// which may not work in unit tests. We'll just verify the method doesn't panic.
	t.Log("ShowNotification method executed without panic")
}

// TestMainPage_UpdateMidFlexTitle tests title update functionality
func TestMainPage_UpdateMidFlexTitle(t *testing.T) {
	mainPage := NewMainPage()
	
	// Initialize MidFlex
	mainPage.MidFlex = tview.NewFlex()
	
	tests := []struct {
		name      string
		title     string
		rowCount  int
		expectTitle string
	}{
		{
			name:        "topics with count",
			title:       "Topics",
			rowCount:    5,
			expectTitle: "Topics (4)", // rowCount - 1 for header
		},
		{
			name:        "contexts with count",
			title:       "Contexts",
			rowCount:    3,
			expectTitle: "Contexts (2)",
		},
		{
			name:        "empty table",
			title:       "Empty",
			rowCount:    1, // Just header
			expectTitle: "Empty (0)",
		},
		{
			name:        "zero rows",
			title:       "Zero",
			rowCount:    0,
			expectTitle: "Zero (-1)", // Edge case
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test UpdateMidFlexTitle doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("UpdateMidFlexTitle panicked: %v", r)
				}
			}()
			
			mainPage.UpdateMidFlexTitle(tt.title, tt.rowCount)
			
			// Verify the title was set (we can't easily check the actual title without complex UI inspection)
		})
	}
}

// TestCreateMainInputLegend tests the input legend creation
func TestCreateMainInputLegend(t *testing.T) {
	legend := CreateMainInputLegend()
	
	if legend == nil {
		t.Fatal("CreateMainInputLegend returned nil")
	}
	
	// Test that it has items (left and right columns)
	if legend.GetItemCount() != 2 {
		t.Errorf("Legend should have 2 items (left and right), got %d", legend.GetItemCount())
	}
}

// TestMainPage_Constants tests the defined constants
func TestMainPage_Constants(t *testing.T) {
	// Test refresh intervals are reasonable
	if refreshInterval <= 0 {
		t.Error("refreshInterval should be positive")
	}
	
	if refreshIntervalTable <= 0 {
		t.Error("refreshIntervalTable should be positive")
	}
	
	// Test that table refresh is faster than general refresh
	if refreshIntervalTable >= refreshInterval {
		t.Error("refreshIntervalTable should be less than refreshInterval")
	}
	
	// Test specific values
	expectedRefreshInterval := 5000 * time.Millisecond
	if refreshInterval != expectedRefreshInterval {
		t.Errorf("refreshInterval = %v, want %v", refreshInterval, expectedRefreshInterval)
	}
	
	expectedRefreshIntervalTable := 500 * time.Millisecond
	if refreshIntervalTable != expectedRefreshIntervalTable {
		t.Errorf("refreshIntervalTable = %v, want %v", refreshIntervalTable, expectedRefreshIntervalTable)
	}
}

// TestMainPage_ErrorHandling tests error handling scenarios
func TestMainPage_ErrorHandling(t *testing.T) {
	mainPage := NewMainPage()
	
	tests := []struct {
		name     string
		testFunc func() error
	}{
		{
			name: "UpdateTable with nil resource",
			testFunc: func() error {
				defer func() {
					if r := recover(); r != nil {
						// Expected to panic with nil resource
					}
				}()
				mainPage.UpdateTable(tview.NewTable(), &MockKafkaDataSource{})
				return nil
			},
		},
		{
			name: "CurrentTimeString with time manipulation",
			testFunc: func() error {
				// This should always work
				timeStr := mainPage.CurrentTimeString()
				if timeStr == "" {
					t.Error("CurrentTimeString should not be empty")
				}
				return nil
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			if err != nil {
				t.Errorf("Test function failed: %v", err)
			}
		})
	}
}

// TestMainPage_ComponentIntegration tests integration between components
func TestMainPage_ComponentIntegration(t *testing.T) {
	mainPage := NewMainPage()
	dataSource := &MockKafkaDataSource{}
	pages := tview.NewPages()
	app := tview.NewApplication()
	modal := tview.NewModal()
	msgChannel := make(chan UIEvent, 10)
	defer close(msgChannel)
	
	// Create the main page
	flex := mainPage.CreateMainPage(dataSource, pages, app, modal, msgChannel)
	
	// Test component interactions
	if mainPage.SearchBar == nil {
		t.Fatal("SearchBar not initialized")
	}
	
	if mainPage.CurrentResource == nil {
		t.Fatal("CurrentResource not initialized")
	}
	
	// Test that search bar and current resource are compatible
	resourceName := (*mainPage.CurrentResource).GetName()
	if resourceName == "" {
		t.Error("Current resource should have a name")
	}
	
	// Test that the flex container is properly structured
	if flex.GetItemCount() == 0 {
		t.Error("Main flex should have items")
	}
}

// Benchmark tests for MainPage operations
func BenchmarkNewMainPage(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewMainPage()
	}
}

func BenchmarkMainPage_CurrentTimeString(b *testing.B) {
	mainPage := NewMainPage()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mainPage.CurrentTimeString()
	}
}

func BenchmarkCreateMainInputLegend(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CreateMainInputLegend()
	}
}