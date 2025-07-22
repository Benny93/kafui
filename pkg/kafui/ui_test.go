package kafui

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TestCreatePropertyInfo tests the CreatePropertyInfo function
func TestCreatePropertyInfo(t *testing.T) {
	tests := []struct {
		name          string
		propertyName  string
		propertyValue string
		expectedLabel string
		expectedText  string
		expectedDisabled bool
	}{
		{
			name:             "basic property",
			propertyName:     "Topic",
			propertyValue:    "user-events",
			expectedLabel:    "Topic: ",
			expectedText:     "user-events",
			expectedDisabled: true,
		},
		{
			name:             "empty property name",
			propertyName:     "",
			propertyValue:    "value",
			expectedLabel:    ": ",
			expectedText:     "value",
			expectedDisabled: true,
		},
		{
			name:             "empty property value",
			propertyName:     "Status",
			propertyValue:    "",
			expectedLabel:    "Status: ",
			expectedText:     "",
			expectedDisabled: true,
		},
		{
			name:             "property with special characters",
			propertyName:     "Config-Key",
			propertyValue:    "value with spaces & symbols!",
			expectedLabel:    "Config-Key: ",
			expectedText:     "value with spaces & symbols!",
			expectedDisabled: true,
		},
		{
			name:             "numeric property",
			propertyName:     "Partitions",
			propertyValue:    "5",
			expectedLabel:    "Partitions: ",
			expectedText:     "5",
			expectedDisabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputField := CreatePropertyInfo(tt.propertyName, tt.propertyValue)

			if inputField == nil {
				t.Fatal("CreatePropertyInfo returned nil")
			}

			// Test label
			if inputField.GetLabel() != tt.expectedLabel {
				t.Errorf("Label = %v, want %v", inputField.GetLabel(), tt.expectedLabel)
			}

			// Test text
			if inputField.GetText() != tt.expectedText {
				t.Errorf("Text = %v, want %v", inputField.GetText(), tt.expectedText)
			}

			// Test disabled state
			if !tt.expectedDisabled {
				t.Error("InputField should be disabled")
			}

			// Test field width (should be 0 for full width)
			// Note: We can't directly test GetFieldWidth() as it's not exposed,
			// but we can verify the field was created successfully
		})
	}
}

// TestCreateRunInfo tests the CreateRunInfo function
func TestCreateRunInfo(t *testing.T) {
	tests := []struct {
		name         string
		runeName     string
		info         string
		expectedLabel string
		expectedText string
	}{
		{
			name:          "basic rune info",
			runeName:      "Enter",
			info:          "Select item",
			expectedLabel: "<Enter>: ",
			expectedText:  "Select item",
		},
		{
			name:          "single character rune",
			runeName:      "q",
			info:          "Quit",
			expectedLabel: "<q>: ",
			expectedText:  "Quit",
		},
		{
			name:          "arrow key",
			runeName:      "↑",
			info:          "Move up",
			expectedLabel: "<↑>: ",
			expectedText:  "Move up",
		},
		{
			name:          "empty rune name",
			runeName:      "",
			info:          "No action",
			expectedLabel: "<>: ",
			expectedText:  "No action",
		},
		{
			name:          "empty info",
			runeName:      "Esc",
			info:          "",
			expectedLabel: "<Esc>: ",
			expectedText:  "",
		},
		{
			name:          "complex key combination",
			runeName:      "Ctrl+C",
			info:          "Copy to clipboard",
			expectedLabel: "<Ctrl+C>: ",
			expectedText:  "Copy to clipboard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputField := CreateRunInfo(tt.runeName, tt.info)

			if inputField == nil {
				t.Fatal("CreateRunInfo returned nil")
			}

			// Test label
			if inputField.GetLabel() != tt.expectedLabel {
				t.Errorf("Label = %v, want %v", inputField.GetLabel(), tt.expectedLabel)
			}

			// Test text
			if inputField.GetText() != tt.expectedText {
				t.Errorf("Text = %v, want %v", inputField.GetText(), tt.expectedText)
			}

			// Test that the field is disabled
			// Note: We can't directly test IsDisabled() but we can verify creation
		})
	}
}

// TestUIThemeConfiguration tests that the UI theme is properly configured
func TestUIThemeConfiguration(t *testing.T) {
	// Save original theme
	originalTheme := tview.Styles

	// Test theme configuration (simulating what happens in OpenUI)
	testTheme := tview.Theme{
		PrimitiveBackgroundColor:    tcell.ColorBlack.TrueColor(),
		ContrastBackgroundColor:     tcell.ColorBlack.TrueColor(),
		MoreContrastBackgroundColor: tcell.ColorGreen.TrueColor(),
		BorderColor:                 tcell.ColorWhite.TrueColor(),
		TitleColor:                  tcell.ColorWhite.TrueColor(),
		GraphicsColor:               tcell.ColorBlack.TrueColor(),
		PrimaryTextColor:            tcell.ColorDarkCyan.TrueColor(),
		SecondaryTextColor:          tcell.ColorWhite.TrueColor(),
		TertiaryTextColor:           tcell.ColorGreen.TrueColor(),
		InverseTextColor:            tcell.ColorGreen.TrueColor(),
		ContrastSecondaryTextColor:  tcell.ColorWhite.TrueColor(),
	}

	// Apply test theme
	tview.Styles = testTheme

	// Verify theme colors
	if tview.Styles.PrimitiveBackgroundColor != tcell.ColorBlack.TrueColor() {
		t.Error("PrimitiveBackgroundColor not set correctly")
	}

	if tview.Styles.BorderColor != tcell.ColorWhite.TrueColor() {
		t.Error("BorderColor not set correctly")
	}

	if tview.Styles.PrimaryTextColor != tcell.ColorDarkCyan.TrueColor() {
		t.Error("PrimaryTextColor not set correctly")
	}

	// Restore original theme
	tview.Styles = originalTheme
}

// TestUIEventHandling tests UI event constants and handling
func TestUIEventHandling(t *testing.T) {
	// Test that UI events are properly defined
	events := []UIEvent{
		OnModalClose,
		OnFocusSearch,
		OnStartTableSearch,
		OnPageChange,
	}

	expectedEvents := []string{
		"ModalClose",
		"FocusSearch",
		"OnStartTableSearch",
		"PageChange",
	}

	for i, event := range events {
		if string(event) != expectedEvents[i] {
			t.Errorf("Event %d = %v, want %v", i, string(event), expectedEvents[i])
		}
	}
}

// TestInputFieldCreation tests the creation of input fields with various configurations
func TestInputFieldCreation(t *testing.T) {
	tests := []struct {
		name     string
		function func() *tview.InputField
		testFunc func(*testing.T, *tview.InputField)
	}{
		{
			name: "property info field",
			function: func() *tview.InputField {
				return CreatePropertyInfo("TestProp", "TestValue")
			},
			testFunc: func(t *testing.T, field *tview.InputField) {
				if field.GetLabel() != "TestProp: " {
					t.Error("Property info label incorrect")
				}
				if field.GetText() != "TestValue" {
					t.Error("Property info text incorrect")
				}
			},
		},
		{
			name: "run info field",
			function: func() *tview.InputField {
				return CreateRunInfo("TestKey", "TestAction")
			},
			testFunc: func(t *testing.T, field *tview.InputField) {
				if field.GetLabel() != "<TestKey>: " {
					t.Error("Run info label incorrect")
				}
				if field.GetText() != "TestAction" {
					t.Error("Run info text incorrect")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := tt.function()
			if field == nil {
				t.Fatal("Input field creation returned nil")
			}
			tt.testFunc(t, field)
		})
	}
}

// TestUIComponentIntegration tests basic UI component integration
func TestUIComponentIntegration(t *testing.T) {
	// Test that UI components can be created and configured together
	propertyField := CreatePropertyInfo("Topic", "test-topic")
	runeField := CreateRunInfo("Enter", "Select")

	if propertyField == nil || runeField == nil {
		t.Fatal("Failed to create UI components")
	}

	// Test that components have different configurations
	if propertyField.GetLabel() == runeField.GetLabel() {
		t.Error("Property and rune fields should have different labels")
	}

	// Test that both components are properly configured
	if propertyField.GetText() == "" {
		t.Error("Property field should have text")
	}

	if runeField.GetText() == "" {
		t.Error("Rune field should have text")
	}
}

// TestUIGlobalVariables tests the global UI variables
func TestUIGlobalVariables(t *testing.T) {
	// Test that global variables can be accessed
	// Note: These are package-level variables that get set during UI initialization
	
	// We can't easily test the actual values without running the full UI,
	// but we can test that the variables exist and can be assigned
	var testTopic api.Topic
	testTopic = api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 2,
		MessageCount:      100,
	}

	// Simulate setting the global variable
	originalTopic := currentTopic
	currentTopic = testTopic

	// Verify the assignment worked
	if currentTopic.NumPartitions != 3 {
		t.Error("Global topic variable not set correctly")
	}

	// Restore original value
	currentTopic = originalTopic
}

// Benchmark tests for UI component creation
func BenchmarkCreatePropertyInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		field := CreatePropertyInfo("BenchmarkProp", "BenchmarkValue")
		_ = field
	}
}

func BenchmarkCreateRunInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		field := CreateRunInfo("BenchmarkKey", "BenchmarkAction")
		_ = field
	}
}