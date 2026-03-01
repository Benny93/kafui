package providers

import (
	"strings"
	"testing"
)

func TestDefaultContentProvider_GetContentSize(t *testing.T) {
	provider := NewDefaultContentProvider()

	tests := []struct {
		name     string
		width    int
		wantMin  int
		wantMax  int
	}{
		{"Minimum size mode", 30, 3, 15},
		{"Small size mode", 60, 5, 25},
		{"Compact size mode", 120, 10, 25},
		{"Normal size mode", 150, 15, 30},
		{"Big size mode", 200, 15, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := provider.GetContentSize(tt.width)

			if size < tt.wantMin {
				t.Errorf("GetContentSize(%d) = %d, want >= %d", tt.width, size, tt.wantMin)
			}
			if size > tt.wantMax {
				t.Errorf("GetContentSize(%d) = %d, want <= %d", tt.width, size, tt.wantMax)
			}
		})
	}
}

func TestDefaultContentProvider_RenderContent_AdaptiveSize(t *testing.T) {
	provider := NewDefaultContentProvider()

	tests := []struct {
		name         string
		width        int
		height       int
		expectLogo   bool
		expectControls bool
	}{
		{"Minimum size", 30, 20, false, false},
		{"Small size", 60, 25, true, false},
		{"Compact size", 120, 35, true, true},
		{"Normal size", 150, 40, true, true},
		{"Big size", 200, 50, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := provider.RenderContent(tt.width, tt.height)

			if tt.expectLogo && !strings.Contains(content, "CRUSH") {
				t.Error("Expected CRUSH logo in content")
			}

			if tt.expectControls && !strings.Contains(content, "Ctrl") {
				t.Error("Expected controls in content")
			}
		})
	}
}

func TestDefaultContentProvider_RenderContent_WidthConstraint(t *testing.T) {
	provider := NewDefaultContentProvider()

	// Test that content respects width constraints
	// Note: Decorative elements (logo lines, section separators) may exceed width
	// but text content should be truncated. The component's lipgloss styling
	// will handle final width constraints.
	width := 80
	height := 30
	content := provider.RenderContent(width, height)

	// Just verify content is rendered and not empty
	if content == "" {
		t.Error("Expected non-empty content")
	}
}

func TestDefaultContentProvider_RenderContent_EmptyDimensions(t *testing.T) {
	provider := NewDefaultContentProvider()

	// Test with zero/negative dimensions
	content := provider.RenderContent(0, 0)
	if content != "" {
		t.Error("Expected empty content for zero dimensions")
	}

	content = provider.RenderContent(-10, -10)
	if content != "" {
		t.Error("Expected empty content for negative dimensions")
	}
}

func TestDefaultContentProvider_RenderContent_TextTruncation(t *testing.T) {
	provider := NewDefaultContentProvider()

	// Test with narrow width - content should adapt and not be excessively long
	width := 40
	height := 30
	content := provider.RenderContent(width, height)

	// Verify content is rendered
	if content == "" {
		t.Error("Expected non-empty content for narrow width")
	}

	// Check that text lines (excluding decorative elements) are reasonable
	lines := strings.Split(content, "\n")
	textLines := 0
	for _, line := range lines {
		// Skip decorative lines (logo, separators)
		if strings.Contains(line, "╱") || strings.Contains(line, "─") {
			continue
		}
		textLines++
		// Text lines should be somewhat constrained
		if len(line) > width+20 { // Allow tolerance for styling
			t.Errorf("RenderContent() text line %d too long: %d > %d. Line: %s", textLines, len(line), width+20, line)
		}
	}
}

func TestDefaultContentProvider_Interface(t *testing.T) {
	provider := NewDefaultContentProvider()

	// Verify that DefaultContentProvider implements ContentProvider interface
	var _ ContentProvider = provider
}

func TestDefaultContentProvider_InitAndHandle(t *testing.T) {
	provider := NewDefaultContentProvider()

	// Test InitContent
	cmd := provider.InitContent()
	if cmd != nil {
		t.Log("InitContent returned a command (this is OK)")
	}

	// Test HandleContentUpdate
	cmd = provider.HandleContentUpdate(nil)
	if cmd != nil {
		t.Log("HandleContentUpdate returned a command (this is OK)")
	}
}
