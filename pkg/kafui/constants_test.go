package kafui

import (
	"reflect"
	"testing"
)

// TestUIEvent tests the UIEvent type
func TestUIEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    UIEvent
		expected string
	}{
		{
			name:     "OnModalClose event",
			event:    OnModalClose,
			expected: "ModalClose",
		},
		{
			name:     "OnFocusSearch event",
			event:    OnFocusSearch,
			expected: "FocusSearch",
		},
		{
			name:     "OnStartTableSearch event",
			event:    OnStartTableSearch,
			expected: "OnStartTableSearch",
		},
		{
			name:     "OnPageChange event",
			event:    OnPageChange,
			expected: "PageChange",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.event) != tt.expected {
				t.Errorf("UIEvent %s = %v, want %v", tt.name, string(tt.event), tt.expected)
			}
		})
	}
}

// TestResouceName tests the ResouceName type and predefined resource names
func TestResouceName(t *testing.T) {
	tests := []struct {
		name     string
		resource ResouceName
		expected []string
	}{
		{
			name:     "Context resource names",
			resource: Context,
			expected: []string{"context", "ctx", "kafka", "broker"},
		},
		{
			name:     "Topic resource names",
			resource: Topic,
			expected: []string{"topics", "ts"},
		},
		{
			name:     "ConsumerGroup resource names",
			resource: ConsumerGroup,
			expected: []string{"consumergroups", "groups", "consumers", "cgs"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual([]string(tt.resource), tt.expected) {
				t.Errorf("ResouceName %s = %v, want %v", tt.name, []string(tt.resource), tt.expected)
			}
		})
	}
}

// TestResourceNameContains tests if resource names contain expected aliases
func TestResourceNameContains(t *testing.T) {
	tests := []struct {
		name     string
		resource ResouceName
		alias    string
		expected bool
	}{
		{
			name:     "Context contains 'ctx'",
			resource: Context,
			alias:    "ctx",
			expected: true,
		},
		{
			name:     "Context contains 'kafka'",
			resource: Context,
			alias:    "kafka",
			expected: true,
		},
		{
			name:     "Context does not contain 'invalid'",
			resource: Context,
			alias:    "invalid",
			expected: false,
		},
		{
			name:     "Topic contains 'topics'",
			resource: Topic,
			alias:    "topics",
			expected: true,
		},
		{
			name:     "Topic contains 'ts'",
			resource: Topic,
			alias:    "ts",
			expected: true,
		},
		{
			name:     "Topic does not contain 'topic'",
			resource: Topic,
			alias:    "topic",
			expected: false,
		},
		{
			name:     "ConsumerGroup contains 'groups'",
			resource: ConsumerGroup,
			alias:    "groups",
			expected: true,
		},
		{
			name:     "ConsumerGroup contains 'cgs'",
			resource: ConsumerGroup,
			alias:    "cgs",
			expected: true,
		},
		{
			name:     "ConsumerGroup does not contain 'group'",
			resource: ConsumerGroup,
			alias:    "group",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Contains([]string(tt.resource), tt.alias)
			if result != tt.expected {
				t.Errorf("ResouceName.Contains(%s) = %v, want %v", tt.alias, result, tt.expected)
			}
		})
	}
}

// TestSearchMode tests the SearchMode type
func TestSearchMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     SearchMode
		expected string
	}{
		{
			name:     "TableSearch mode",
			mode:     TableSearch,
			expected: "TableSearch",
		},
		{
			name:     "ResouceSearch mode",
			mode:     ResouceSearch,
			expected: "ResouceSearch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.mode) != tt.expected {
				t.Errorf("SearchMode %s = %v, want %v", tt.name, string(tt.mode), tt.expected)
			}
		})
	}
}

// TestResourceNameLength tests that resource names have expected lengths
func TestResourceNameLength(t *testing.T) {
	tests := []struct {
		name           string
		resource       ResouceName
		expectedLength int
	}{
		{
			name:           "Context has 4 aliases",
			resource:       Context,
			expectedLength: 4,
		},
		{
			name:           "Topic has 2 aliases",
			resource:       Topic,
			expectedLength: 2,
		},
		{
			name:           "ConsumerGroup has 4 aliases",
			resource:       ConsumerGroup,
			expectedLength: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.resource) != tt.expectedLength {
				t.Errorf("ResouceName %s length = %d, want %d", tt.name, len(tt.resource), tt.expectedLength)
			}
		})
	}
}

// TestResourceNameUniqueness tests that all aliases within a resource are unique
func TestResourceNameUniqueness(t *testing.T) {
	resources := map[string]ResouceName{
		"Context":       Context,
		"Topic":         Topic,
		"ConsumerGroup": ConsumerGroup,
	}

	for name, resource := range resources {
		t.Run(name+" aliases are unique", func(t *testing.T) {
			seen := make(map[string]bool)
			for _, alias := range resource {
				if seen[alias] {
					t.Errorf("Duplicate alias '%s' found in %s resource", alias, name)
				}
				seen[alias] = true
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkUIEventString(b *testing.B) {
	event := OnModalClose
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = string(event)
	}
}

func BenchmarkResourceNameContains(b *testing.B) {
	resource := Context
	alias := "kafka"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Contains([]string(resource), alias)
	}
}

func BenchmarkSearchModeString(b *testing.B) {
	mode := TableSearch
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = string(mode)
	}
}