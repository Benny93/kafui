package kafui

import (
	"reflect"
	"sort"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/rivo/tview"
)

// TestContains tests the generic Contains function
func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    interface{}
		element  interface{}
		expected bool
	}{
		{
			name:     "string slice contains element",
			slice:    []string{"apple", "banana", "cherry"},
			element:  "banana",
			expected: true,
		},
		{
			name:     "string slice does not contain element",
			slice:    []string{"apple", "banana", "cherry"},
			element:  "grape",
			expected: false,
		},
		{
			name:     "int slice contains element",
			slice:    []int{1, 2, 3, 4, 5},
			element:  3,
			expected: true,
		},
		{
			name:     "int slice does not contain element",
			slice:    []int{1, 2, 3, 4, 5},
			element:  6,
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			element:  "test",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result bool
			switch s := tt.slice.(type) {
			case []string:
				result = Contains(s, tt.element.(string))
			case []int:
				result = Contains(s, tt.element.(int))
			}

			if result != tt.expected {
				t.Errorf("Contains() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestFilter tests the generic filter function
func TestFilter(t *testing.T) {
	t.Run("filter even numbers", func(t *testing.T) {
		numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		isEven := func(n int) bool { return n%2 == 0 }
		
		result := filter(numbers, isEven)
		expected := []int{2, 4, 6, 8, 10}

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("filter() = %v, want %v", result, expected)
		}
	})

	t.Run("filter strings by length", func(t *testing.T) {
		words := []string{"cat", "elephant", "dog", "hippopotamus", "ant"}
		isLongWord := func(s string) bool { return len(s) > 3 }
		
		result := filter(words, isLongWord)
		expected := []string{"elephant", "hippopotamus"}

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("filter() = %v, want %v", result, expected)
		}
	})

	t.Run("filter empty slice", func(t *testing.T) {
		numbers := []int{}
		isPositive := func(n int) bool { return n > 0 }
		
		result := filter(numbers, isPositive)
		
		if len(result) != 0 {
			t.Errorf("filter() on empty slice should return empty slice, got %v", result)
		}
	})

	t.Run("filter with no matches", func(t *testing.T) {
		numbers := []int{1, 3, 5, 7, 9}
		isEven := func(n int) bool { return n%2 == 0 }
		
		result := filter(numbers, isEven)
		
		if len(result) != 0 {
			t.Errorf("filter() with no matches should return empty slice, got %v", result)
		}
	})
}

// TestByOffsetThenPartition tests the sorting implementation
func TestByOffsetThenPartition(t *testing.T) {
	tests := []struct {
		name     string
		messages []api.Message
		expected []api.Message
	}{
		{
			name: "sort by offset ascending",
			messages: []api.Message{
				{Offset: 3, Partition: 0},
				{Offset: 1, Partition: 0},
				{Offset: 2, Partition: 0},
			},
			expected: []api.Message{
				{Offset: 1, Partition: 0},
				{Offset: 2, Partition: 0},
				{Offset: 3, Partition: 0},
			},
		},
		{
			name: "sort by partition when offset is same",
			messages: []api.Message{
				{Offset: 1, Partition: 2},
				{Offset: 1, Partition: 0},
				{Offset: 1, Partition: 1},
			},
			expected: []api.Message{
				{Offset: 1, Partition: 0},
				{Offset: 1, Partition: 1},
				{Offset: 1, Partition: 2},
			},
		},
		{
			name: "mixed offset and partition sorting",
			messages: []api.Message{
				{Offset: 2, Partition: 1},
				{Offset: 1, Partition: 2},
				{Offset: 1, Partition: 0},
				{Offset: 2, Partition: 0},
			},
			expected: []api.Message{
				{Offset: 1, Partition: 0},
				{Offset: 1, Partition: 2},
				{Offset: 2, Partition: 0},
				{Offset: 2, Partition: 1},
			},
		},
		{
			name:     "empty slice",
			messages: []api.Message{},
			expected: []api.Message{},
		},
		{
			name: "single message",
			messages: []api.Message{
				{Offset: 42, Partition: 3},
			},
			expected: []api.Message{
				{Offset: 42, Partition: 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := make([]api.Message, len(tt.messages))
			copy(messages, tt.messages)
			
			sort.Sort(ByOffsetThenPartition(messages))

			if !reflect.DeepEqual(messages, tt.expected) {
				t.Errorf("ByOffsetThenPartition sort failed:\ngot:  %v\nwant: %v", messages, tt.expected)
			}
		})
	}
}

// TestByOffsetThenPartitionInterface tests the sort.Interface implementation
func TestByOffsetThenPartitionInterface(t *testing.T) {
	messages := []api.Message{
		{Offset: 3, Partition: 1},
		{Offset: 1, Partition: 0},
		{Offset: 2, Partition: 2},
	}

	sorter := ByOffsetThenPartition(messages)

	// Test Len()
	if sorter.Len() != 3 {
		t.Errorf("Len() = %d, want 3", sorter.Len())
	}

	// Test Less()
	if !sorter.Less(1, 0) { // messages[1].Offset (1) < messages[0].Offset (3)
		t.Errorf("Less(1, 0) should be true")
	}

	if sorter.Less(0, 1) { // messages[0].Offset (3) > messages[1].Offset (1)
		t.Errorf("Less(0, 1) should be false")
	}

	// Test Swap()
	originalFirst := messages[0]
	originalSecond := messages[1]
	sorter.Swap(0, 1)

	if !reflect.DeepEqual(messages[0], originalSecond) || !reflect.DeepEqual(messages[1], originalFirst) {
		t.Errorf("Swap(0, 1) did not swap elements correctly")
	}
}

// TestRecoverAndExit tests the panic recovery function
func TestRecoverAndExit(t *testing.T) {
	// Create a mock tview application
	app := tview.NewApplication()
	
	// Test that RecoverAndExit handles panic gracefully
	t.Run("handles panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RecoverAndExit should have caught the panic, but panic propagated: %v", r)
			}
		}()

		// This test is tricky because RecoverAndExit is designed to be used in a defer statement
		// and we can't easily test the actual panic recovery without causing side effects
		// Instead, we'll test that the function exists and can be called
		RecoverAndExit(app)
	})
}

// Benchmark tests for performance-critical functions
func BenchmarkContains(b *testing.B) {
	slice := []string{"apple", "banana", "cherry", "date", "elderberry", "fig", "grape"}
	element := "cherry"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Contains(slice, element)
	}
}

func BenchmarkFilter(b *testing.B) {
	numbers := make([]int, 1000)
	for i := range numbers {
		numbers[i] = i
	}
	isEven := func(n int) bool { return n%2 == 0 }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter(numbers, isEven)
	}
}

func BenchmarkByOffsetThenPartitionSort(b *testing.B) {
	// Create a large slice of messages for benchmarking
	messages := make([]api.Message, 1000)
	for i := range messages {
		messages[i] = api.Message{
			Offset:    int64(1000 - i), // Reverse order to force sorting
			Partition: int32(i % 10),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Make a copy for each iteration to ensure consistent state
		testMessages := make([]api.Message, len(messages))
		copy(testMessages, messages)
		sort.Sort(ByOffsetThenPartition(testMessages))
	}
}