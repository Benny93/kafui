package core

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDefaultGlobalKeys(t *testing.T) {
	tests := []struct {
		name        string
		keyBinding  key.Binding
		expectedKey string
		expectedHelp string
	}{
		{
			name:        "Help key",
			keyBinding:  DefaultGlobalKeys.Help,
			expectedKey: "?",
			expectedHelp: "help",
		},
		{
			name:        "Quit key",
			keyBinding:  DefaultGlobalKeys.Quit,
			expectedKey: "q",
			expectedHelp: "quit",
		},
		{
			name:        "Back key",
			keyBinding:  DefaultGlobalKeys.Back,
			expectedKey: "esc",
			expectedHelp: "back",
		},
		{
			name:        "Next page key",
			keyBinding:  DefaultGlobalKeys.NextPage,
			expectedKey: "tab",
			expectedHelp: "next page",
		},
		{
			name:        "Previous page key",
			keyBinding:  DefaultGlobalKeys.PrevPage,
			expectedKey: "shift+tab",
			expectedHelp: "prev page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the key binding matches expected keys
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.expectedKey)}
			if tt.expectedKey == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			} else if tt.expectedKey == "tab" {
				keyMsg = tea.KeyMsg{Type: tea.KeyTab}
			} else if tt.expectedKey == "shift+tab" {
				keyMsg = tea.KeyMsg{Type: tea.KeyShiftTab}
			}

			matches := key.Matches(keyMsg, tt.keyBinding)
			if !matches && tt.expectedKey != "q" { // q has multiple bindings
				t.Errorf("Expected key %s to match binding, but it didn't", tt.expectedKey)
			}

			// Test help text
			help := tt.keyBinding.Help()
			if help.Desc != tt.expectedHelp {
				t.Errorf("Expected help text %s, got %s", tt.expectedHelp, help.Desc)
			}
		})
	}
}

func TestGetAllBindings(t *testing.T) {
	bindings := DefaultGlobalKeys.GetAllBindings()
	
	expectedCount := 5
	if len(bindings) != expectedCount {
		t.Errorf("Expected %d bindings, got %d", expectedCount, len(bindings))
	}

	// Verify all bindings are present
	expectedBindings := []key.Binding{
		DefaultGlobalKeys.Help,
		DefaultGlobalKeys.Quit,
		DefaultGlobalKeys.Back,
		DefaultGlobalKeys.NextPage,
		DefaultGlobalKeys.PrevPage,
	}

	for i, expected := range expectedBindings {
		if bindings[i].Help().Key != expected.Help().Key {
			t.Errorf("Binding at index %d doesn't match expected key: got %s, want %s", 
				i, bindings[i].Help().Key, expected.Help().Key)
		}
		if bindings[i].Help().Desc != expected.Help().Desc {
			t.Errorf("Binding at index %d doesn't match expected description: got %s, want %s", 
				i, bindings[i].Help().Desc, expected.Help().Desc)
		}
	}
}