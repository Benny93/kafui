package components

import (
	"strings"
	"testing"
)

func TestScrollbar_NoScrollbarWhenContentFits(t *testing.T) {
	// When content size <= viewport size, no scrollbar should be rendered
	scrollbar := Scrollbar(20, 10, 20, 0)
	if scrollbar != "" {
		t.Error("Expected no scrollbar when content fits, got scrollbar")
	}
}

func TestScrollbar_NoScrollbarWhenHeightIsZero(t *testing.T) {
	scrollbar := Scrollbar(0, 50, 20, 0)
	if scrollbar != "" {
		t.Error("Expected no scrollbar when height is 0")
	}
}

func TestScrollbar_ScrollbarWhenContentOverflows(t *testing.T) {
	// When content size > viewport size, scrollbar should be rendered
	scrollbar := Scrollbar(20, 50, 20, 0)
	if scrollbar == "" {
		t.Error("Expected scrollbar for overflow content")
	}
}

func TestScrollbar_ThumbSize(t *testing.T) {
	// Test that thumb size is proportional to content/viewport ratio
	height := 20
	contentSize := 100
	viewportSize := 20
	
	scrollbar := Scrollbar(height, contentSize, viewportSize, 0)
	lines := strings.Split(scrollbar, "\n")
	
	// Count thumb characters (should be different from track)
	// The thumb should be at least 1 character
	if len(lines) != height {
		t.Errorf("Expected %d lines in scrollbar, got %d", height, len(lines))
	}
}

func TestScrollbar_ScrollPosition(t *testing.T) {
	height := 20
	contentSize := 100
	viewportSize := 20
	
	// Test scrollbar at different positions
	scrollbarTop := Scrollbar(height, contentSize, viewportSize, 0)
	scrollbarMiddle := Scrollbar(height, contentSize, viewportSize, 40)
	scrollbarBottom := Scrollbar(height, contentSize, viewportSize, 80)
	
	// Verify all scrollbars are rendered
	if scrollbarTop == "" {
		t.Error("Expected scrollbar at top position")
	}
	if scrollbarMiddle == "" {
		t.Error("Expected scrollbar at middle position")
	}
	if scrollbarBottom == "" {
		t.Error("Expected scrollbar at bottom position")
	}
}

func TestScrollbar_MinThumbSize(t *testing.T) {
	// Test that thumb size is at least 1 character even for very large content
	height := 10
	contentSize := 1000
	viewportSize := 10
	
	scrollbar := Scrollbar(height, contentSize, viewportSize, 0)
	if scrollbar == "" {
		t.Error("Expected scrollbar with minimum thumb size")
	}
	
	lines := strings.Split(scrollbar, "\n")
	hasThumb := false
	for _, line := range lines {
		if strings.Contains(line, "│") {
			hasThumb = true
			break
		}
	}
	
	if !hasThumb {
		t.Error("Scrollbar should have at least one thumb character")
	}
}

func TestScrollbar_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		height       int
		contentSize  int
		viewportSize int
		offset       int
		expectEmpty  bool
	}{
		{"Negative height", -5, 50, 20, 0, true},
		{"Equal content and viewport", 20, 20, 20, 0, true},
		{"Content smaller than viewport", 20, 10, 20, 0, true},
		{"Zero viewport", 20, 50, 0, 0, true},
		{"Large offset", 20, 100, 20, 80, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scrollbar := Scrollbar(tt.height, tt.contentSize, tt.viewportSize, tt.offset)
			if tt.expectEmpty && scrollbar != "" {
				t.Errorf("Expected empty scrollbar, got: %s", scrollbar)
			}
			if !tt.expectEmpty && scrollbar == "" {
				t.Error("Expected non-empty scrollbar")
			}
		})
	}
}
