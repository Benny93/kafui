package kafui

import (
	"strings"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
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

// TestMainPage_FetchConsumerGroups tests consumer group fetching
func TestMainPage_FetchConsumerGroups(t *testing.T) {
	mainPage := NewMainPage()
	
	tests := []struct {
		name           string
		dataSource     api.KafkaDataSource
		expectedLength int
		expectError    bool
	}{
		{
			name:           "successful fetch",
			dataSource:     &MockKafkaDataSource{},
			expectedLength: 2,
			expectError:    false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize notification text view to capture error messages
			mainPage.NotificationTextView = tview.NewTextView()
			
			result := mainPage.FetchConsumerGroups(tt.dataSource)
			
			if len(result) != tt.expectedLength {
				t.Errorf("FetchConsumerGroups() returned %d groups, want %d", len(result), tt.expectedLength)
			}
			
			// For successful cases, verify the content
			if !tt.expectError && len(result) > 0 {
				for i, group := range result {
					if group.Name == "" {
						t.Errorf("Consumer group %d has empty Name", i)
					}
				}
			}
		})
	}
}

// TestMainPage_FetchContexts tests context fetching
func TestMainPage_FetchContexts(t *testing.T) {
	mainPage := NewMainPage()
	
	tests := []struct {
		name           string
		dataSource     api.KafkaDataSource
		expectedLength int
		expectError    bool
	}{
		{
			name:           "successful fetch",
			dataSource:     &MockKafkaDataSource{},
			expectedLength: 2,
			expectError:    false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize notification text view to capture error messages
			mainPage.NotificationTextView = tview.NewTextView()
			
			result := mainPage.FetchContexts(tt.dataSource)
			
			if len(result) != tt.expectedLength {
				t.Errorf("FetchContexts() returned %d contexts, want %d", len(result), tt.expectedLength)
			}
			
			// For successful cases, verify the content
			if !tt.expectError && len(result) > 0 {
				for i, context := range result {
					if context == "" {
						t.Errorf("Context %d is empty", i)
					}
				}
			}
		})
	}
}

// TestCreateNotificationTextView tests notification text view creation
func TestCreateNotificationTextView(t *testing.T) {
	textView := createNotificationTextView()
	
	if textView == nil {
		t.Fatal("createNotificationTextView() returned nil")
	}
	
	// Test initial state
	text := textView.GetText(false)
	if text != "" {
		t.Errorf("Initial text should be empty, got %q", text)
	}
	
	// Test that border is disabled (we can't easily test this with tview API)
	// The border setting is internal to tview
}

// TestCreateContextInfo tests context info creation
func TestCreateContextInfo(t *testing.T) {
	contextInfo := createContextInfo()
	
	if contextInfo == nil {
		t.Fatal("createContextInfo() returned nil")
	}
	
	// Test initial state
	text := contextInfo.GetText()
	if text != "n/a" {
		t.Errorf("Initial text should be 'n/a', got %q", text)
	}
	
	// Test that field is disabled (we can't easily test this with tview API)
	// The disabled setting is internal to tview
	
	// Test label
	label := contextInfo.GetLabel()
	if label != "Current Context: " {
		t.Errorf("Label should be 'Current Context: ', got %q", label)
	}
}

// TestMainPage_UpdateTable_EdgeCases tests edge cases for table updates
func TestMainPage_UpdateTable_EdgeCases(t *testing.T) {
	mainPage := NewMainPage()
	table := tview.NewTable()
	dataSource := &MockKafkaDataSource{}
	
	tests := []struct {
		name         string
		setupFunc    func()
		expectPanic  bool
	}{
		{
			name: "nil current resource",
			setupFunc: func() {
				mainPage.CurrentResource = nil
				mainPage.SearchBar = nil
			},
			expectPanic: false,
		},
		{
			name: "nil search bar",
			setupFunc: func() {
				mockResource := NewResouceTopic(dataSource, func(err error) {}, func() {})
				var resource Resource = mockResource
				mainPage.CurrentResource = &resource
				mainPage.SearchBar = nil
			},
			expectPanic: false,
		},
		{
			name: "valid setup with search string",
			setupFunc: func() {
				mockResource := NewResouceTopic(dataSource, func(err error) {}, func() {})
				var resource Resource = mockResource
				mainPage.CurrentResource = &resource
				searchBar := NewSearchBar(table, dataSource, tview.NewPages(), tview.NewApplication(), 
					tview.NewModal(), func(Resource, string) {}, func(error) {})
				searchBar.CurrentString = "test-search"
				mainPage.SearchBar = searchBar
			},
			expectPanic: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()
			
			defer func() {
				if r := recover(); r != nil {
					if !tt.expectPanic {
						t.Errorf("UpdateTable panicked unexpectedly: %v", r)
					}
				} else if tt.expectPanic {
					t.Error("UpdateTable should have panicked but didn't")
				}
			}()
			
			mainPage.UpdateTable(table, dataSource)
		})
	}
}

// TestMainPage_ShowNotification_EdgeCases tests edge cases for notifications
func TestMainPage_ShowNotification_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		setupFunc    func(*MainPage)
		expectPanic  bool
	}{
		{
			name:    "nil notification text view",
			message: "Test message",
			setupFunc: func(mp *MainPage) {
				mp.NotificationTextView = nil
			},
			expectPanic: false,
		},
		{
			name:    "empty message",
			message: "",
			setupFunc: func(mp *MainPage) {
				mp.NotificationTextView = tview.NewTextView()
			},
			expectPanic: false,
		},
		{
			name:    "long message",
			message: strings.Repeat("This is a very long message. ", 100),
			setupFunc: func(mp *MainPage) {
				mp.NotificationTextView = tview.NewTextView()
			},
			expectPanic: false,
		},
		{
			name:    "message with special characters",
			message: "Error: Connection failed! @#$%^&*(){}[]|\\:;\"'<>,.?/~`",
			setupFunc: func(mp *MainPage) {
				mp.NotificationTextView = tview.NewTextView()
			},
			expectPanic: false,
		},
		{
			name:    "unicode message",
			message: "错误: 连接失败! 🚨💥",
			setupFunc: func(mp *MainPage) {
				mp.NotificationTextView = tview.NewTextView()
			},
			expectPanic: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mainPage := NewMainPage()
			tt.setupFunc(mainPage)
			
			defer func() {
				if r := recover(); r != nil {
					if !tt.expectPanic {
						t.Errorf("ShowNotification panicked unexpectedly: %v", r)
					}
				} else if tt.expectPanic {
					t.Error("ShowNotification should have panicked but didn't")
				}
			}()
			
			mainPage.ShowNotification(tt.message)
			
			// Give some time for goroutine to start
			time.Sleep(5 * time.Millisecond)
		})
	}
}

// TestMainPage_UpdateMidFlexTitle_EdgeCases tests edge cases for title updates
func TestMainPage_UpdateMidFlexTitle_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(*MainPage)
		resource     string
		amount       int
		expectPanic  bool
	}{
		{
			name: "nil MidFlex",
			setupFunc: func(mp *MainPage) {
				mp.MidFlex = nil
			},
			resource:    "Topics",
			amount:      5,
			expectPanic: false,
		},
		{
			name: "valid MidFlex",
			setupFunc: func(mp *MainPage) {
				mp.MidFlex = tview.NewFlex()
			},
			resource:    "Topics",
			amount:      5,
			expectPanic: false,
		},
		{
			name: "empty resource name",
			setupFunc: func(mp *MainPage) {
				mp.MidFlex = tview.NewFlex()
			},
			resource:    "",
			amount:      0,
			expectPanic: false,
		},
		{
			name: "negative amount",
			setupFunc: func(mp *MainPage) {
				mp.MidFlex = tview.NewFlex()
			},
			resource:    "Topics",
			amount:      -1,
			expectPanic: false,
		},
		{
			name: "large amount",
			setupFunc: func(mp *MainPage) {
				mp.MidFlex = tview.NewFlex()
			},
			resource:    "Topics",
			amount:      1000000,
			expectPanic: false,
		},
		{
			name: "special characters in resource name",
			setupFunc: func(mp *MainPage) {
				mp.MidFlex = tview.NewFlex()
			},
			resource:    "Topics@#$%^&*()",
			amount:      5,
			expectPanic: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mainPage := NewMainPage()
			tt.setupFunc(mainPage)
			
			defer func() {
				if r := recover(); r != nil {
					if !tt.expectPanic {
						t.Errorf("UpdateMidFlexTitle panicked unexpectedly: %v", r)
					}
				} else if tt.expectPanic {
					t.Error("UpdateMidFlexTitle should have panicked but didn't")
				}
			}()
			
			mainPage.UpdateMidFlexTitle(tt.resource, tt.amount)
		})
	}
}

// TestMainPage_UpdateTableRoutine_EdgeCases tests edge cases for the update routine
func TestMainPage_UpdateTableRoutine_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() (*MainPage, *tview.Application, *tview.Table, *tview.TextView, api.KafkaDataSource)
		expectExit  bool
	}{
		{
			name: "nil app",
			setupFunc: func() (*MainPage, *tview.Application, *tview.Table, *tview.TextView, api.KafkaDataSource) {
				mainPage := NewMainPage()
				return mainPage, nil, tview.NewTable(), tview.NewTextView(), &MockKafkaDataSource{}
			},
			expectExit: true,
		},
		{
			name: "nil table",
			setupFunc: func() (*MainPage, *tview.Application, *tview.Table, *tview.TextView, api.KafkaDataSource) {
				mainPage := NewMainPage()
				app := tview.NewApplication()
				return mainPage, app, nil, tview.NewTextView(), &MockKafkaDataSource{}
			},
			expectExit: false,
		},
		{
			name: "nil timer view",
			setupFunc: func() (*MainPage, *tview.Application, *tview.Table, *tview.TextView, api.KafkaDataSource) {
				mainPage := NewMainPage()
				app := tview.NewApplication()
				return mainPage, app, tview.NewTable(), nil, &MockKafkaDataSource{}
			},
			expectExit: false,
		},
		{
			name: "nil data source",
			setupFunc: func() (*MainPage, *tview.Application, *tview.Table, *tview.TextView, api.KafkaDataSource) {
				mainPage := NewMainPage()
				app := tview.NewApplication()
				return mainPage, app, tview.NewTable(), tview.NewTextView(), nil
			},
			expectExit: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mainPage, app, table, timerView, dataSource := tt.setupFunc()
			
			// Set up required components for non-nil cases
			if app != nil {
				mockResource := NewResouceTopic(&MockKafkaDataSource{}, func(err error) {}, func() {})
				var resource Resource = mockResource
				mainPage.CurrentResource = &resource
				if table != nil {
					searchBar := NewSearchBar(table, &MockKafkaDataSource{}, tview.NewPages(), app, 
						tview.NewModal(), func(Resource, string) {}, func(error) {})
					mainPage.SearchBar = searchBar
				}
			}
			
			// Test that the routine handles edge cases gracefully
			done := make(chan bool, 1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("UpdateTableRoutine panicked: %v", r)
					}
					done <- true
				}()
				mainPage.UpdateTableRoutine(app, table, timerView, dataSource)
			}()
			
			// Wait a short time for the routine to process
			select {
			case <-done:
				if !tt.expectExit {
					t.Log("Routine exited as expected")
				}
			case <-time.After(50 * time.Millisecond):
				if tt.expectExit {
					t.Error("Routine should have exited immediately but didn't")
				}
			}
		})
	}
}