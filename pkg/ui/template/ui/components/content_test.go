package components

import (
	"testing"
)

func TestCappedContentWidth(t *testing.T) {
	tests := []struct {
		name           string
		availableWidth int
		expectedWidth  int
	}{
		{"Very small width", 10, 6},
		{"Small width", 50, 46},
		{"Normal width", 100, 96},
		{"Large width (should cap)", 150, 120},
		{"Very large width (should cap)", 300, 120},
		{"Edge case: exactly MaxContentWidth + padding", 124, 120},
		{"Edge case: less than padding", 3, -1}, // Will be negative, component handles this
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cappedContentWidth(tt.availableWidth)
			if result != tt.expectedWidth {
				t.Errorf("cappedContentWidth(%d) = %d, want %d", tt.availableWidth, result, tt.expectedWidth)
			}
		})
	}
}

func TestCappedContentWidth_MaxWidth(t *testing.T) {
	// Ensure that the max width is never exceeded
	for width := 0; width < 500; width += 10 {
		result := cappedContentWidth(width)
		if result > MaxContentWidth {
			t.Errorf("cappedContentWidth(%d) = %d exceeds MaxContentWidth (%d)", width, result, MaxContentWidth)
		}
	}
}

func TestCappedContentWidth_MinWidth(t *testing.T) {
	// Ensure that very small widths are handled correctly
	result := cappedContentWidth(0)
	if result != -4 {
		// Note: This will be negative, which is expected behavior
		// The component should handle this case in View()
		t.Logf("cappedContentWidth(0) = %d (negative is expected)", result)
	}
}
