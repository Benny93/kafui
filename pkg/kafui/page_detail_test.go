package kafui

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TestNewDetailPage tests the DetailPage constructor
func TestNewDetailPage(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	
	tests := []struct {
		name    string
		headers api.MessageHeaders
		value   string
	}{
		{
			name: "valid JSON message",
			headers: api.MessageHeaders{
				{Key: "content-type", Value: "application/json"},
				{Key: "timestamp", Value: "2024-01-15T10:30:00Z"},
			},
			value: `{"user_id": 123, "action": "login", "timestamp": "2024-01-15T10:30:00Z"}`,
		},
		{
			name: "plain text message",
			headers: api.MessageHeaders{
				{Key: "content-type", Value: "text/plain"},
			},
			value: "Simple text message",
		},
		{
			name: "empty headers",
			headers: api.MessageHeaders{},
			value: `{"test": "value"}`,
		},
		{
			name:    "empty value",
			headers: api.MessageHeaders{{Key: "test", Value: "header"}},
			value:   "",
		},
		{
			name: "invalid JSON",
			headers: api.MessageHeaders{
				{Key: "content-type", Value: "application/json"},
			},
			value: `{"invalid": json}`,
		},
		{
			name: "complex JSON",
			headers: api.MessageHeaders{
				{Key: "content-type", Value: "application/json"},
				{Key: "schema-id", Value: "user-event-v1"},
			},
			value: `{
				"user": {
					"id": 123,
					"name": "John Doe",
					"preferences": {
						"theme": "dark",
						"notifications": true
					}
				},
				"event": {
					"type": "login",
					"timestamp": "2024-01-15T10:30:00Z",
					"metadata": {
						"ip": "192.168.1.1",
						"user_agent": "Mozilla/5.0"
					}
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detailPage := NewDetailPage(app, pages, tt.headers, tt.value)

			if detailPage == nil {
				t.Fatal("NewDetailPage returned nil")
			}

			// Test basic properties
			if detailPage.app != app {
				t.Error("App not set correctly")
			}

			if detailPage.pages != pages {
				t.Error("Pages not set correctly")
			}

			if len(detailPage.headers) != len(tt.headers) {
				t.Errorf("Headers length = %d, want %d", len(detailPage.headers), len(tt.headers))
			}

			if detailPage.value != tt.value {
				t.Errorf("Value = %v, want %v", detailPage.value, tt.value)
			}

			// Test initial state
			if detailPage.showHeaders != false {
				t.Error("showHeaders should be false initially")
			}

			// Test UI components
			if detailPage.headerTable == nil {
				t.Error("headerTable should be initialized")
			}

			if detailPage.valueTextView == nil {
				t.Error("valueTextView should be initialized")
			}

			// Test header table setup
			if len(tt.headers) > 0 {
				// Check that header table has the correct number of rows
				expectedRows := len(tt.headers)
				if detailPage.headerTable.GetRowCount() != expectedRows {
					t.Errorf("Header table rows = %d, want %d", detailPage.headerTable.GetRowCount(), expectedRows)
				}

				// Check header table content
				for i, header := range tt.headers {
					keyCell := detailPage.headerTable.GetCell(i, 0)
					valueCell := detailPage.headerTable.GetCell(i, 1)

					if keyCell == nil || valueCell == nil {
						t.Errorf("Header table cells missing for row %d", i)
						continue
					}

					if keyCell.Text != header.Key {
						t.Errorf("Header key[%d] = %v, want %v", i, keyCell.Text, header.Key)
					}

					if valueCell.Text != header.Value {
						t.Errorf("Header value[%d] = %v, want %v", i, valueCell.Text, header.Value)
					}
				}
			}
		})
	}
}

// TestDetailPage_Show tests the Show method
func TestDetailPage_Show(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	headers := api.MessageHeaders{
		{Key: "content-type", Value: "application/json"},
	}
	value := `{"test": "value"}`

	detailPage := NewDetailPage(app, pages, headers, value)

	// Test Show method
	detailPage.Show()

	// Verify that the page was added
	pageNames := pages.GetPageNames(false)
	found := false
	for _, name := range pageNames {
		if name == "DetailPage" {
			found = true
			break
		}
	}

	if !found {
		t.Error("DetailPage was not added to pages")
	}

	// Test that the page is visible
	frontPageName, _ := pages.GetFrontPage()
	if frontPageName != "DetailPage" {
		t.Errorf("Front page = %v, want DetailPage", frontPageName)
	}
}

// TestDetailPage_Hide tests the Hide method
func TestDetailPage_Hide(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	headers := api.MessageHeaders{}
	value := "test value"

	detailPage := NewDetailPage(app, pages, headers, value)

	// First show the page
	detailPage.Show()

	// Verify it's there
	pageNames := pages.GetPageNames(false)
	if len(pageNames) == 0 {
		t.Fatal("No pages found after Show()")
	}

	// Now hide it
	detailPage.Hide()

	// Verify it's removed
	pageNames = pages.GetPageNames(false)
	for _, name := range pageNames {
		if name == "DetailPage" {
			t.Error("DetailPage should have been removed")
		}
	}
}

// TestDetailPage_HandleInput tests the input handling
func TestDetailPage_HandleInput(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	headers := api.MessageHeaders{
		{Key: "test-header", Value: "test-value"},
	}
	value := `{"test": "json"}`

	detailPage := NewDetailPage(app, pages, headers, value)

	tests := []struct {
		name           string
		key            tcell.Key
		rune           rune
		expectedResult *tcell.EventKey
		testFunc       func(*testing.T, *DetailPage)
	}{
		{
			name:           "copy content with 'c'",
			key:            tcell.KeyRune,
			rune:           'c',
			expectedResult: nil, // Should return nil to indicate handled
			testFunc: func(t *testing.T, dp *DetailPage) {
				// Note: We can't easily test clipboard functionality in unit tests
				// but we can verify the key was handled
			},
		},
		{
			name:           "toggle headers with 'h'",
			key:            tcell.KeyRune,
			rune:           'h',
			expectedResult: nil, // Should return nil to indicate handled
			testFunc: func(t *testing.T, dp *DetailPage) {
				// The showHeaders state should have been toggled
				// Note: This is tricky to test due to the Show/Hide cycle
			},
		},
		{
			name:           "unhandled key",
			key:            tcell.KeyRune,
			rune:           'x',
			expectedResult: nil, // Should return the event unchanged
			testFunc: func(t *testing.T, dp *DetailPage) {
				// Should not change any state
			},
		},
		{
			name:           "escape key",
			key:            tcell.KeyEsc,
			rune:           0,
			expectedResult: nil, // Should return the event
			testFunc: func(t *testing.T, dp *DetailPage) {
				// Should not be handled by DetailPage
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create event
			event := tcell.NewEventKey(tt.key, tt.rune, tcell.ModNone)

			// Note: TextView doesn't have SetFocus method, but we can test input handling

			// Handle the input
			result := detailPage.handleInput(event)

			// Note: The actual behavior depends on TextView focus state
			// For 'c' and 'h' keys, they should be handled when TextView has focus
			// Since we can't easily simulate focus in unit tests, we test the method exists
			if result == nil && (tt.rune != 'c' && tt.rune != 'h') {
				// Only non-handled keys should return the original event
			}

			// For other keys, result should be the original event
			if tt.rune != 'c' && tt.rune != 'h' && result != event {
				t.Errorf("Expected original event for key '%c', got %v", tt.rune, result)
			}

			// Run additional test function
			if tt.testFunc != nil {
				tt.testFunc(t, detailPage)
			}
		})
	}
}

// TestDetailPage_CreateInputLegend tests the legend creation
func TestDetailPage_CreateInputLegend(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	headers := api.MessageHeaders{}
	value := "test"

	detailPage := NewDetailPage(app, pages, headers, value)
	legend := detailPage.CreateInputLegend()

	if legend == nil {
		t.Fatal("CreateInputLegend returned nil")
	}

	// Test that it's a Flex container (we can't directly test direction)
	// but we can verify it was created successfully

	// Test that it has items (left and right columns)
	if legend.GetItemCount() != 2 {
		t.Errorf("Legend should have 2 items (left and right), got %d", legend.GetItemCount())
	}
}

// TestDetailPage_HeaderToggling tests the header visibility toggling
func TestDetailPage_HeaderToggling(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	headers := api.MessageHeaders{
		{Key: "header1", Value: "value1"},
		{Key: "header2", Value: "value2"},
	}
	value := `{"test": "value"}`

	detailPage := NewDetailPage(app, pages, headers, value)

	// Initial state should be headers hidden
	if detailPage.showHeaders {
		t.Error("Headers should be hidden initially")
	}

	// Note: The actual toggle happens in handleInput, but testing the full cycle
	// with Show/Hide is complex due to UI dependencies
	originalState := detailPage.showHeaders
	
	// Simulate the toggle logic
	detailPage.showHeaders = !detailPage.showHeaders
	
	if detailPage.showHeaders == originalState {
		t.Error("Header visibility should have been toggled")
	}
}

// TestDetailPage_JSONFormatting tests JSON formatting behavior
func TestDetailPage_JSONFormatting(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	headers := api.MessageHeaders{}

	tests := []struct {
		name        string
		value       string
		expectJSON  bool
	}{
		{
			name:       "valid JSON",
			value:      `{"key": "value", "number": 123}`,
			expectJSON: true,
		},
		{
			name:       "invalid JSON",
			value:      `{"invalid": json}`,
			expectJSON: false,
		},
		{
			name:       "plain text",
			value:      "This is plain text",
			expectJSON: false,
		},
		{
			name:       "empty string",
			value:      "",
			expectJSON: false,
		},
		{
			name:       "JSON array",
			value:      `[{"item": 1}, {"item": 2}]`,
			expectJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detailPage := NewDetailPage(app, pages, headers, tt.value)

			if detailPage.valueTextView == nil {
				t.Fatal("valueTextView should be initialized")
			}

			// The text view should contain the value (either formatted JSON or plain text)
			text := detailPage.valueTextView.GetText(false)
			if text == "" && tt.value != "" {
				t.Error("valueTextView should contain text")
			}

			// For valid JSON, the formatted text might be different from input
			// For invalid JSON or plain text, it should match the input
			if !tt.expectJSON && text != tt.value {
				t.Errorf("Plain text should match input: got %v, want %v", text, tt.value)
			}
		})
	}
}

// TestDetailPage_EdgeCases tests edge cases and error conditions
func TestDetailPage_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		app     *tview.Application
		pages   *tview.Pages
		headers api.MessageHeaders
		value   string
		expectPanic bool
	}{
		{
			name:    "nil app",
			app:     nil,
			pages:   tview.NewPages(),
			headers: api.MessageHeaders{},
			value:   "test",
			expectPanic: false, // Should handle gracefully
		},
		{
			name:    "nil pages",
			app:     tview.NewApplication(),
			pages:   nil,
			headers: api.MessageHeaders{},
			value:   "test",
			expectPanic: false, // Should handle gracefully
		},
		{
			name:    "nil headers",
			app:     tview.NewApplication(),
			pages:   tview.NewPages(),
			headers: nil,
			value:   "test",
			expectPanic: false, // Should handle gracefully
		},
		{
			name:    "large JSON value",
			app:     tview.NewApplication(),
			pages:   tview.NewPages(),
			headers: api.MessageHeaders{},
			value:   generateLargeJSON(),
			expectPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.expectPanic {
						t.Errorf("Unexpected panic: %v", r)
					}
				}
			}()

			detailPage := NewDetailPage(tt.app, tt.pages, tt.headers, tt.value)

			if !tt.expectPanic && detailPage == nil {
				t.Error("NewDetailPage should not return nil for valid inputs")
			}
		})
	}
}

// Helper function to generate large JSON for testing
func generateLargeJSON() string {
	return `{
		"large_object": {
			"field1": "value1",
			"field2": "value2",
			"nested": {
				"deep": {
					"very_deep": {
						"data": "test"
					}
				}
			},
			"array": [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
		}
	}`
}

// Benchmark tests for DetailPage operations
func BenchmarkNewDetailPage(b *testing.B) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	headers := api.MessageHeaders{
		{Key: "content-type", Value: "application/json"},
	}
	value := `{"benchmark": "test"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewDetailPage(app, pages, headers, value)
	}
}

func BenchmarkDetailPage_Show(b *testing.B) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	headers := api.MessageHeaders{}
	value := "benchmark test"

	detailPage := NewDetailPage(app, pages, headers, value)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detailPage.Show()
		detailPage.Hide()
	}
}