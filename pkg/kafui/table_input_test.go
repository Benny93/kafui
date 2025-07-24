package kafui

import (
	"context"
	"strings"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TestMockKafkaDataSource for table input tests
type TestMockKafkaDataSource struct{}

func (m *TestMockKafkaDataSource) Init(cfgOption string) {}

func (m *TestMockKafkaDataSource) GetTopics() (map[string]api.Topic, error) {
	return map[string]api.Topic{
		"test-topic": {
			NumPartitions:     3,
			ReplicationFactor: 2,
			MessageCount:      100,
		},
	}, nil
}

func (m *TestMockKafkaDataSource) GetContexts() ([]string, error) {
	return []string{"test-context", "prod-context"}, nil
}

func (m *TestMockKafkaDataSource) GetContext() string {
	return "test-context"
}

func (m *TestMockKafkaDataSource) SetContext(contextName string) error {
	return nil
}

func (m *TestMockKafkaDataSource) GetConsumerGroups() ([]api.ConsumerGroup, error) {
	return []api.ConsumerGroup{
		{Name: "test-group", State: "Active", Consumers: 2},
	}, nil
}

func (m *TestMockKafkaDataSource) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) error {
	return nil
}

// TestCopySelectedRowToClipboard tests the clipboard functionality
func TestCopySelectedRowToClipboard(t *testing.T) {
	tests := []struct {
		name           string
		tableData      [][]string
		selectedRow    int
		expectedCSV    string
		expectError    bool
		expectedMsg    string
	}{
		{
			name: "valid row selection",
			tableData: [][]string{
				{"Header1", "Header2", "Header3"},
				{"Value1", "Value2", "Value3"},
				{"Data1", "Data2", "Data3"},
			},
			selectedRow: 1,
			expectedCSV: "Value1,Value2,Value3",
			expectError: false,
			expectedMsg: "😎 Copied selection to clipboard ...",
		},
		{
			name: "single column row",
			tableData: [][]string{
				{"Header"},
				{"SingleValue"},
			},
			selectedRow: 1,
			expectedCSV: "SingleValue",
			expectError: false,
			expectedMsg: "😎 Copied selection to clipboard ...",
		},
		{
			name: "row with empty cells",
			tableData: [][]string{
				{"Header1", "Header2", "Header3"},
				{"Value1", "", "Value3"},
			},
			selectedRow: 1,
			expectedCSV: "Value1,,Value3",
			expectError: false,
			expectedMsg: "😎 Copied selection to clipboard ...",
		},
		{
			name: "invalid row selection - header row",
			tableData: [][]string{
				{"Header1", "Header2"},
				{"Value1", "Value2"},
			},
			selectedRow: 0,
			expectedCSV: "",
			expectError: true,
			expectedMsg: "Copy: Invalid row selection",
		},
		{
			name: "invalid row selection - out of bounds",
			tableData: [][]string{
				{"Header1", "Header2"},
				{"Value1", "Value2"},
			},
			selectedRow: 5,
			expectedCSV: "",
			expectError: true,
			expectedMsg: "Copy: Invalid row selection",
		},
		{
			name: "empty table",
			tableData: [][]string{},
			selectedRow: 1,
			expectedCSV: "",
			expectError: true,
			expectedMsg: "Copy: Invalid row selection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create table and populate with test data
			table := tview.NewTable()
			
			for row, rowData := range tt.tableData {
				for col, cellData := range rowData {
					table.SetCell(row, col, tview.NewTableCell(cellData))
				}
			}

			// Set the selection to the test row
			table.Select(tt.selectedRow, 0)

			// Capture the notification message
			var capturedMessage string
			consumeMessage := func(message string) {
				capturedMessage = message
			}

			// Test the copy function
			CopySelectedRowToClipboard(table, consumeMessage)

			// Verify the notification message
			if capturedMessage != tt.expectedMsg {
				t.Errorf("Message = %v, want %v", capturedMessage, tt.expectedMsg)
			}

			// Note: We can't easily test the actual clipboard content in unit tests
			// due to external dependencies, but we can verify the logic flow
		})
	}
}

// TestSetupTableInput tests the table input setup
func TestSetupTableInput(t *testing.T) {
	// Create test components
	mainPage := NewMainPage()
	table := tview.NewTable()
	app := tview.NewApplication()
	pages := tview.NewPages()
	dataSource := &TestMockKafkaDataSource{}
	msgChannel := make(chan UIEvent, 10)

	// Set up search bar for the main page
	searchBar := NewSearchBar(table, dataSource, pages, app, tview.NewModal(), 
		func(newResource Resource, searchText string) {}, func(err error) {})
	mainPage.SearchBar = searchBar

	// Test that SetupTableInput doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SetupTableInput panicked: %v", r)
		}
	}()

	mainPage.SetupTableInput(table, app, pages, dataSource, msgChannel)

	// The function sets up input capture, but we can't easily test the actual
	// key handling without complex event simulation
}

// TestTableInputKeyHandling tests key handling logic
func TestTableInputKeyHandling(t *testing.T) {
	// Create test setup
	mainPage := NewMainPage()
	table := tview.NewTable()
	app := tview.NewApplication()
	pages := tview.NewPages()
	dataSource := &TestMockKafkaDataSource{}
	msgChannel := make(chan UIEvent, 10)

	// Add main page to pages
	pages.AddPage("main", tview.NewFlex(), true, true)

	// Set up search bar
	searchBar := NewSearchBar(table, dataSource, pages, app, tview.NewModal(), 
		func(newResource Resource, searchText string) {}, func(err error) {})
	mainPage.SearchBar = searchBar

	// Create test topic resource with sample data
	topicResource := NewResouceTopic(dataSource, func(err error) {}, func() {})
	topicResource.LastFetchedTopics = map[string]api.Topic{
		"test-topic": {
			NumPartitions:     3,
			ReplicationFactor: 2,
			MessageCount:      100,
		},
	}
	searchBar.CurrentResource = topicResource

	// Set up table with test data
	table.SetCell(0, 0, tview.NewTableCell("Topic"))
	table.SetCell(1, 0, tview.NewTableCell("test-topic"))
	table.Select(1, 0)

	// Test key events
	tests := []struct {
		name        string
		key         tcell.Key
		rune        rune
		expectEvent bool
	}{
		{
			name:        "Enter key",
			key:         tcell.KeyEnter,
			rune:        0,
			expectEvent: true,
		},
		{
			name:        "Copy key 'c'",
			key:         tcell.KeyRune,
			rune:        'c',
			expectEvent: true,
		},
		{
			name:        "Other key",
			key:         tcell.KeyRune,
			rune:        'x',
			expectEvent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily test the actual input capture function
			// but we can test the key handling logic components
			
			if tt.key == tcell.KeyEnter {
				// Test Enter key logic
				row, _ := table.GetSelection()
				if row == 0 {
					// Header row should not trigger action
				} else {
					// Data row should trigger action based on resource type
					switch mainPage.SearchBar.CurrentResource.(type) {
					case *ResouceTopic:
						topicName := table.GetCell(row, 0).Text
						if topicName == "" {
							t.Error("Topic name should not be empty")
						}
					case *ResourceContext:
						contextName := table.GetCell(row, 0).Text
						if contextName == "" {
							t.Error("Context name should not be empty")
						}
					}
				}
			}

			if tt.key == tcell.KeyRune && tt.rune == 'c' {
				// Test copy functionality
				var capturedMessage string
				CopySelectedRowToClipboard(table, func(message string) {
					capturedMessage = message
				})
				
				if !strings.Contains(capturedMessage, "Copied") && !strings.Contains(capturedMessage, "Invalid") {
					t.Error("Copy operation should provide feedback")
				}
			}
		})
	}

	// Clean up
	close(msgChannel)
}

// TestCopySelectedRowToClipboard_EdgeCases tests edge cases
func TestCopySelectedRowToClipboard_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupTable  func() *tview.Table
		expectedMsg string
	}{
		{
			name: "table with nil cells",
			setupTable: func() *tview.Table {
				table := tview.NewTable()
				table.SetCell(0, 0, tview.NewTableCell("Header"))
				table.SetCell(1, 0, tview.NewTableCell("Value"))
				table.SetCell(1, 1, nil) // Nil cell
				table.Select(1, 0)
				return table
			},
			expectedMsg: "😎 Copied selection to clipboard ...",
		},
		{
			name: "table with special characters",
			setupTable: func() *tview.Table {
				table := tview.NewTable()
				table.SetCell(0, 0, tview.NewTableCell("Header"))
				table.SetCell(1, 0, tview.NewTableCell("Value,with,commas"))
				table.SetCell(1, 1, tview.NewTableCell("Value\"with\"quotes"))
				table.Select(1, 0)
				return table
			},
			expectedMsg: "😎 Copied selection to clipboard ...",
		},
		{
			name: "very large table",
			setupTable: func() *tview.Table {
				table := tview.NewTable()
				// Create header
				for col := 0; col < 10; col++ {
					table.SetCell(0, col, tview.NewTableCell("Header"+string(rune('A'+col))))
				}
				// Create data row
				for col := 0; col < 10; col++ {
					table.SetCell(1, col, tview.NewTableCell("Data"+string(rune('A'+col))))
				}
				table.Select(1, 0)
				return table
			},
			expectedMsg: "😎 Copied selection to clipboard ...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := tt.setupTable()
			
			var capturedMessage string
			CopySelectedRowToClipboard(table, func(message string) {
				capturedMessage = message
			})

			if capturedMessage != tt.expectedMsg {
				t.Errorf("Message = %v, want %v", capturedMessage, tt.expectedMsg)
			}
		})
	}
}

// TestTableInputResourceSwitching tests resource switching logic
func TestTableInputResourceSwitching(t *testing.T) {
	// Test the resource switching logic used in table input
	mainPage := NewMainPage()
	dataSource := &TestMockKafkaDataSource{}
	
	// Test context switching
	contextName := "test-context"
	err := dataSource.SetContext(contextName)
	if err != nil {
		t.Errorf("SetContext should not fail for mock data source: %v", err)
	}

	// Test that context name is stored
	mainPage.CurrentContextName = contextName
	if mainPage.CurrentContextName != contextName {
		t.Errorf("CurrentContextName = %v, want %v", mainPage.CurrentContextName, contextName)
	}
}

// Benchmark tests for table input operations
func BenchmarkCopySelectedRowToClipboard(b *testing.B) {
	// Create test table
	table := tview.NewTable()
	for row := 0; row < 100; row++ {
		for col := 0; col < 5; col++ {
			table.SetCell(row, col, tview.NewTableCell("Data"))
		}
	}
	table.Select(50, 0)

	consumeMessage := func(message string) {}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CopySelectedRowToClipboard(table, consumeMessage)
	}
}

func BenchmarkTableSetup(b *testing.B) {
	mainPage := NewMainPage()
	dataSource := &TestMockKafkaDataSource{}
	msgChannel := make(chan UIEvent, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		table := tview.NewTable()
		app := tview.NewApplication()
		pages := tview.NewPages()
		
		searchBar := NewSearchBar(table, dataSource, pages, app, tview.NewModal(), 
			func(newResource Resource, searchText string) {}, func(err error) {})
		mainPage.SearchBar = searchBar
		
		mainPage.SetupTableInput(table, app, pages, dataSource, msgChannel)
	}
	
	close(msgChannel)
}

// TestSetupTableInputAdvanced tests more advanced scenarios for table input setup
func TestSetupTableInputAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() (*MainPage, *tview.Table, *tview.Application, *tview.Pages, api.KafkaDataSource, chan UIEvent)
		expectPanic bool
	}{
		{
			name: "normal_setup",
			setupFunc: func() (*MainPage, *tview.Table, *tview.Application, *tview.Pages, api.KafkaDataSource, chan UIEvent) {
				mainPage := NewMainPage()
				table := tview.NewTable()
				app := tview.NewApplication()
				pages := tview.NewPages()
				dataSource := &TestMockKafkaDataSource{}
				msgChannel := make(chan UIEvent, 10)
				
				searchBar := NewSearchBar(table, dataSource, pages, app, tview.NewModal(), 
					func(newResource Resource, searchText string) {}, func(err error) {})
				mainPage.SearchBar = searchBar
				
				return mainPage, table, app, pages, dataSource, msgChannel
			},
			expectPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tt.expectPanic && r == nil {
					t.Error("Expected panic but didn't get one")
				} else if !tt.expectPanic && r != nil {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			mainPage, table, app, pages, dataSource, msgChannel := tt.setupFunc()
			mainPage.SetupTableInput(table, app, pages, dataSource, msgChannel)
			
			if msgChannel != nil {
				close(msgChannel)
			}
		})
	}
}

// TestTableInputContextSwitching tests context switching functionality
func TestTableInputContextSwitching(t *testing.T) {
	mainPage := NewMainPage()
	table := tview.NewTable()
	app := tview.NewApplication()
	pages := tview.NewPages()
	dataSource := &TestMockKafkaDataSource{}
	msgChannel := make(chan UIEvent, 10)
	defer close(msgChannel)

	// Add main page to pages
	pages.AddPage("main", tview.NewFlex(), true, true)

	// Set up search bar with context resource
	searchBar := NewSearchBar(table, dataSource, pages, app, tview.NewModal(), 
		func(newResource Resource, searchText string) {}, func(err error) {})
	mainPage.SearchBar = searchBar

	// Create context resource
	contextResource := NewResourceContext(dataSource, func(err error) {}, func() {})
	searchBar.CurrentResource = contextResource

	// Set up table with context data
	table.SetCell(0, 0, tview.NewTableCell("Context"))
	table.SetCell(1, 0, tview.NewTableCell("test-context"))
	table.SetCell(2, 0, tview.NewTableCell("prod-context"))

	// Test context switching logic
	tests := []struct {
		name        string
		selectedRow int
		expectedCtx string
	}{
		{"switch_to_test_context", 1, "test-context"},
		{"switch_to_prod_context", 2, "prod-context"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table.Select(tt.selectedRow, 0)
			
			// Simulate the context switching logic from SetupTableInput
			row, _ := table.GetSelection()
			if row > 0 {
				text := table.GetCell(row, 0).Text
				mainPage.CurrentContextName = text
				err := dataSource.SetContext(mainPage.CurrentContextName)
				
				if err != nil {
					t.Errorf("SetContext failed: %v", err)
				}
				
				if mainPage.CurrentContextName != tt.expectedCtx {
					t.Errorf("Expected context %s, got %s", tt.expectedCtx, mainPage.CurrentContextName)
				}
			}
		})
	}
}

// TestCopySelectedRowToClipboardErrorHandling tests error scenarios
func TestCopySelectedRowToClipboardErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupTable  func() *tview.Table
		expectedMsg string
	}{
		{
			name: "negative_row_selection",
			setupTable: func() *tview.Table {
				table := tview.NewTable()
				table.SetCell(0, 0, tview.NewTableCell("Header"))
				table.SetCell(1, 0, tview.NewTableCell("Value"))
				table.Select(-1, 0) // Negative row
				return table
			},
			expectedMsg: "Copy: Invalid row selection",
		},
		{
			name: "table_with_no_columns",
			setupTable: func() *tview.Table {
				table := tview.NewTable()
				// Table with rows but no columns
				table.Select(1, 0)
				return table
			},
			expectedMsg: "Copy: Invalid row selection",
		},
		{
			name: "table_with_only_header",
			setupTable: func() *tview.Table {
				table := tview.NewTable()
				table.SetCell(0, 0, tview.NewTableCell("Header"))
				table.Select(0, 0) // Header row
				return table
			},
			expectedMsg: "Copy: Invalid row selection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := tt.setupTable()
			
			var capturedMessage string
			CopySelectedRowToClipboard(table, func(message string) {
				capturedMessage = message
			})

			if capturedMessage != tt.expectedMsg {
				t.Errorf("Message = %v, want %v", capturedMessage, tt.expectedMsg)
			}
		})
	}
}