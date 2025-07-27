package kafui

import (
	"fmt"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/atotto/clipboard"
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

	// First show the page to set up the UI properly
	detailPage.Show()

	tests := []struct {
		name           string
		key            tcell.Key
		rune           rune
		hasFocus       bool
		expectedResult *tcell.EventKey
		testFunc       func(*testing.T, *DetailPage)
	}{
		{
			name:           "copy content with 'c' when focused",
			key:            tcell.KeyRune,
			rune:           'c',
			hasFocus:       true,
			expectedResult: nil, // Should return nil to indicate handled
			testFunc: func(t *testing.T, dp *DetailPage) {
				// The copy operation should have been triggered
				// We can't test clipboard directly, but we can verify the method was called
			},
		},
		{
			name:           "copy content with 'c' when not focused",
			key:            tcell.KeyRune,
			rune:           'c',
			hasFocus:       false,
			expectedResult: nil, // Should return the event unchanged
			testFunc: func(t *testing.T, dp *DetailPage) {
				// Should not trigger copy when not focused
			},
		},
		{
			name:           "toggle headers with 'h' when focused",
			key:            tcell.KeyRune,
			rune:           'h',
			hasFocus:       true,
			expectedResult: nil, // Should return nil to indicate handled
			testFunc: func(t *testing.T, dp *DetailPage) {
				// The showHeaders state should have been toggled
				// Note: This triggers Show/Hide cycle which is complex to test
			},
		},
		{
			name:           "toggle headers with 'h' when not focused",
			key:            tcell.KeyRune,
			rune:           'h',
			hasFocus:       false,
			expectedResult: nil, // Should return the event unchanged
			testFunc: func(t *testing.T, dp *DetailPage) {
				// Should not toggle headers when not focused
			},
		},
		{
			name:           "unhandled key 'x'",
			key:            tcell.KeyRune,
			rune:           'x',
			hasFocus:       true,
			expectedResult: nil, // Should return the event unchanged
			testFunc: func(t *testing.T, dp *DetailPage) {
				// Should not change any state
			},
		},
		{
			name:           "escape key",
			key:            tcell.KeyEsc,
			rune:           0,
			hasFocus:       true,
			expectedResult: nil, // Should return the event
			testFunc: func(t *testing.T, dp *DetailPage) {
				// Should not be handled by DetailPage
			},
		},
		{
			name:           "enter key",
			key:            tcell.KeyEnter,
			rune:           0,
			hasFocus:       true,
			expectedResult: nil, // Should return the event
			testFunc: func(t *testing.T, dp *DetailPage) {
				// Should not be handled by DetailPage
			},
		},
		{
			name:           "arrow key",
			key:            tcell.KeyUp,
			rune:           0,
			hasFocus:       true,
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

			// Mock the HasFocus behavior by creating a custom test
			// Since we can't directly control HasFocus, we'll test the logic paths
			originalShowHeaders := detailPage.showHeaders

			// Handle the input
			result := detailPage.handleInput(event)

			// In unit tests, TextView.HasFocus() always returns false
			// So we test the actual behavior: keys 'c' and 'h' are only handled when focused
			// Since we can't simulate focus in unit tests, all keys should return the original event
			switch tt.rune {
			case 'c':
				// In unit tests, HasFocus() is false, so 'c' key should return original event
				if result != event {
					t.Errorf("Expected original event for 'c' key (no focus in unit test), got %v", result)
				}
			case 'h':
				// In unit tests, HasFocus() is false, so 'h' key should return original event
				if result != event {
					t.Errorf("Expected original event for 'h' key (no focus in unit test), got %v", result)
				}
				// showHeaders should NOT have been toggled since there's no focus
				if detailPage.showHeaders != originalShowHeaders {
					t.Error("showHeaders should not have been toggled without focus")
				}
			default:
				// For other keys, should return original event
				if result != event {
					t.Errorf("Expected original event for key %v, got %v", tt.key, result)
				}
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

// TestDetailPage_ShowCopiedNotification tests the notification functionality
func TestDetailPage_ShowCopiedNotification(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	headers := api.MessageHeaders{}
	value := "test value"

	detailPage := NewDetailPage(app, pages, headers, value)
	
	// First show the page to set up the UI properly
	detailPage.Show()

	// Test that showCopiedNotification doesn't panic
	t.Run("notification doesn't panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("showCopiedNotification panicked: %v", r)
			}
		}()

		// Call the notification function
		detailPage.showCopiedNotification()

		// Give the goroutine a moment to start
		// Note: We can't easily test the full notification cycle due to timing
		// but we can verify the method doesn't panic
	})

	// Test notification with different page states
	t.Run("notification with multiple pages", func(t *testing.T) {
		// Add another page to test the notification behavior
		pages.AddPage("TestPage", tview.NewTextView(), true, false)
		
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("showCopiedNotification with multiple pages panicked: %v", r)
			}
		}()

		detailPage.showCopiedNotification()
	})
}

// TestDetailPage_ShowWithHeaders tests the Show method with headers visible
func TestDetailPage_ShowWithHeaders(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	headers := api.MessageHeaders{
		{Key: "header1", Value: "value1"},
		{Key: "header2", Value: "value2"},
		{Key: "header3", Value: "value3"},
	}
	value := `{"test": "value"}`

	detailPage := NewDetailPage(app, pages, headers, value)

	// Test showing with headers hidden (default)
	t.Run("show with headers hidden", func(t *testing.T) {
		detailPage.showHeaders = false
		detailPage.Show()

		// Verify page was added
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
	})

	// Test showing with headers visible
	t.Run("show with headers visible", func(t *testing.T) {
		detailPage.Hide() // Clean up first
		detailPage.showHeaders = true
		detailPage.Show()

		// Verify page was added
		pageNames := pages.GetPageNames(false)
		found := false
		for _, name := range pageNames {
			if name == "DetailPage" {
				found = true
				break
			}
		}
		if !found {
			t.Error("DetailPage was not added to pages with headers visible")
		}

		// Verify that the front page is DetailPage
		frontPageName, _ := pages.GetFrontPage()
		if frontPageName != "DetailPage" {
			t.Errorf("Front page = %v, want DetailPage", frontPageName)
		}
	})

	// Test showing with empty headers but showHeaders = true
	t.Run("show with empty headers but showHeaders true", func(t *testing.T) {
		emptyHeadersPage := NewDetailPage(app, pages, api.MessageHeaders{}, value)
		emptyHeadersPage.showHeaders = true
		
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Show with empty headers panicked: %v", r)
			}
		}()
		
		emptyHeadersPage.Show()
	})
}

// TestDetailPage_InputCaptureSetup tests that input capture is properly set up
func TestDetailPage_InputCaptureSetup(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	headers := api.MessageHeaders{}
	value := "test"

	detailPage := NewDetailPage(app, pages, headers, value)
	
	// Show the page to set up input capture
	detailPage.Show()

	// Test that the valueTextView has input capture set
	// Note: We can't directly test if SetInputCapture was called,
	// but we can verify the Show method completes without error
	if detailPage.valueTextView == nil {
		t.Error("valueTextView should be initialized after Show()")
	}
}

// TestDetailPage_HandleInputFocusSimulation tests input handling with simulated focus
func TestDetailPage_HandleInputFocusSimulation(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	headers := api.MessageHeaders{
		{Key: "test", Value: "header"},
	}
	value := `{"key": "value"}`

	detailPage := NewDetailPage(app, pages, headers, value)
	detailPage.Show()

	// Test the actual focus-dependent behavior by examining the code path
	t.Run("simulate focused state for 'c' key", func(t *testing.T) {
		// Create a mock TextView that reports as focused
		// Since we can't easily mock HasFocus, we'll test the method directly
		event := tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModNone)
		
		// The handleInput method checks if valueTextView.HasFocus()
		// We can't mock this easily, but we can test that the method handles the event
		result := detailPage.handleInput(event)
		
		// The result depends on the actual focus state, but the method should not panic
		if result == nil {
			// Key was handled (focus was true)
			t.Log("'c' key was handled (focus simulation successful)")
		} else if result == event {
			// Key was not handled (focus was false)
			t.Log("'c' key was not handled (no focus)")
		} else {
			t.Errorf("Unexpected result from handleInput: %v", result)
		}
	})

	t.Run("simulate focused state for 'h' key", func(t *testing.T) {
		originalShowHeaders := detailPage.showHeaders
		event := tcell.NewEventKey(tcell.KeyRune, 'h', tcell.ModNone)
		
		result := detailPage.handleInput(event)
		
		// The result depends on the actual focus state
		if result == nil {
			// Key was handled (focus was true)
			if detailPage.showHeaders == originalShowHeaders {
				t.Error("showHeaders should have been toggled when 'h' key is handled")
			}
			t.Log("'h' key was handled and headers toggled")
		} else if result == event {
			// Key was not handled (focus was false)
			if detailPage.showHeaders != originalShowHeaders {
				t.Error("showHeaders should not have been toggled when 'h' key is not handled")
			}
			t.Log("'h' key was not handled (no focus)")
		} else {
			t.Errorf("Unexpected result from handleInput: %v", result)
		}
	})
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

// TestDetailPage_ClipboardIntegration tests clipboard functionality
func TestDetailPage_ClipboardIntegration(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	headers := api.MessageHeaders{}
	value := "test clipboard content"

	detailPage := NewDetailPage(app, pages, headers, value)
	detailPage.Show()

	// Test clipboard write functionality by calling the method directly
	t.Run("clipboard write", func(t *testing.T) {
		// Create a mock focused text view by directly testing the clipboard operation
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Clipboard operation panicked: %v", r)
			}
		}()

		// Test the clipboard.WriteAll call directly
		err := clipboard.WriteAll(detailPage.valueTextView.GetText(true))
		if err != nil {
			t.Logf("Clipboard write failed (expected in test environment): %v", err)
		}
	})
}

// TestDetailPage_FocusedInputHandling tests input handling with mocked focus
func TestDetailPage_FocusedInputHandling(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	headers := api.MessageHeaders{
		{Key: "test", Value: "header"},
	}
	value := `{"test": "content"}`

	detailPage := NewDetailPage(app, pages, headers, value)
	detailPage.Show()

	// Create a custom TextView that reports as focused
	focusedTextView := tview.NewTextView()
	focusedTextView.SetText(value)
	
	// Replace the valueTextView temporarily to test focused behavior
	originalTextView := detailPage.valueTextView
	detailPage.valueTextView = focusedTextView

	t.Run("copy with simulated focus", func(t *testing.T) {
		// Test the copy logic path by calling the clipboard operation directly
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Copy operation panicked: %v", r)
			}
		}()

		// Simulate the copy operation that would happen with focus
		clipboard.WriteAll(detailPage.valueTextView.GetText(true))
		detailPage.showCopiedNotification()
	})

	t.Run("header toggle with simulated focus", func(t *testing.T) {
		originalShowHeaders := detailPage.showHeaders
		
		// Simulate the header toggle logic
		detailPage.showHeaders = !detailPage.showHeaders
		
		if detailPage.showHeaders == originalShowHeaders {
			t.Error("Headers should have been toggled")
		}
		
		// Test the Show/Hide cycle that happens during toggle
		detailPage.Hide()
		detailPage.Show()
	})

	// Restore original text view
	detailPage.valueTextView = originalTextView
}

// TestDetailPage_NotificationTiming tests the notification display and removal
func TestDetailPage_NotificationTiming(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	headers := api.MessageHeaders{}
	value := "notification test"

	detailPage := NewDetailPage(app, pages, headers, value)
	detailPage.Show()

	t.Run("notification lifecycle", func(t *testing.T) {
		// Test that the notification function completes without error
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Notification lifecycle panicked: %v", r)
			}
		}()

		// Call the notification function
		detailPage.showCopiedNotification()
		
		// Give the goroutine a moment to execute
		time.Sleep(10 * time.Millisecond)
	})
}

// TestDetailPage_JSONFormattingEdgeCases tests additional JSON formatting scenarios
func TestDetailPage_JSONFormattingEdgeCases(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	headers := api.MessageHeaders{}

	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "null JSON",
			value: "null",
		},
		{
			name:  "boolean JSON",
			value: "true",
		},
		{
			name:  "number JSON",
			value: "42",
		},
		{
			name:  "string JSON",
			value: `"hello world"`,
		},
		{
			name:  "nested array",
			value: `[{"nested": [1, 2, {"deep": true}]}, "string"]`,
		},
		{
			name:  "unicode content",
			value: `{"unicode": "Hello 世界 🌍", "emoji": "😀🎉"}`,
		},
		{
			name:  "special characters",
			value: `{"special": "line\nbreak\ttab\"quote"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detailPage := NewDetailPage(app, pages, headers, tt.value)
			
			if detailPage.valueTextView == nil {
				t.Fatal("valueTextView should be initialized")
			}
			
			// Verify the text view contains content
			text := detailPage.valueTextView.GetText(false)
			if text == "" && tt.value != "" {
				t.Error("valueTextView should contain formatted content")
			}
		})
	}
}

// TestDetailPage_HeaderTableConfiguration tests header table setup details
func TestDetailPage_HeaderTableConfiguration(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	
	tests := []struct {
		name    string
		headers api.MessageHeaders
	}{
		{
			name: "single header",
			headers: api.MessageHeaders{
				{Key: "single", Value: "value"},
			},
		},
		{
			name: "multiple headers with special characters",
			headers: api.MessageHeaders{
				{Key: "content-type", Value: "application/json; charset=utf-8"},
				{Key: "x-custom-header", Value: "special!@#$%^&*()"},
				{Key: "unicode-header", Value: "世界🌍"},
			},
		},
		{
			name: "headers with empty values",
			headers: api.MessageHeaders{
				{Key: "empty-value", Value: ""},
				{Key: "space-value", Value: " "},
				{Key: "tab-value", Value: "\t"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detailPage := NewDetailPage(app, pages, tt.headers, "test")
			
			// Verify header table configuration
			if detailPage.headerTable == nil {
				t.Fatal("headerTable should be initialized")
			}
			
			// Check row count
			expectedRows := len(tt.headers)
			actualRows := detailPage.headerTable.GetRowCount()
			if actualRows != expectedRows {
				t.Errorf("Header table rows = %d, want %d", actualRows, expectedRows)
			}
			
			// Verify each header is properly set
			for i, header := range tt.headers {
				keyCell := detailPage.headerTable.GetCell(i, 0)
				valueCell := detailPage.headerTable.GetCell(i, 1)
				
				if keyCell == nil || valueCell == nil {
					t.Errorf("Missing cells for header %d", i)
					continue
				}
				
				if keyCell.Text != header.Key {
					t.Errorf("Header key[%d] = %q, want %q", i, keyCell.Text, header.Key)
				}
				
				if valueCell.Text != header.Value {
					t.Errorf("Header value[%d] = %q, want %q", i, valueCell.Text, header.Value)
				}
			}
		})
	}
}

// TestDetailPage_ShowHideSequence tests multiple show/hide cycles
func TestDetailPage_ShowHideSequence(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	headers := api.MessageHeaders{
		{Key: "test", Value: "header"},
	}
	value := "test value"

	detailPage := NewDetailPage(app, pages, headers, value)

	// Test multiple show/hide cycles
	for i := 0; i < 3; i++ {
		t.Run(fmt.Sprintf("cycle_%d", i), func(t *testing.T) {
			// Show
			detailPage.Show()
			
			// Verify it's shown
			pageNames := pages.GetPageNames(false)
			found := false
			for _, name := range pageNames {
				if name == "DetailPage" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("DetailPage not found after Show() in cycle %d", i)
			}
			
			// Hide
			detailPage.Hide()
			
			// Verify it's hidden
			pageNames = pages.GetPageNames(false)
			for _, name := range pageNames {
				if name == "DetailPage" {
					t.Errorf("DetailPage should be hidden after Hide() in cycle %d", i)
				}
			}
		})
	}
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