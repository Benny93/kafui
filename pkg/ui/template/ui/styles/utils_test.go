package styles

import (
	"testing"
)

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		availableWidth int
		ellipsis       string
		expectedLen    int // Maximum expected length
	}{
		{"Short text", "Hello", 10, "…", 5},
		{"Long text", "This is a very long text that should be truncated", 10, "…", 10},
		{"Empty text", "", 10, "…", 0},
		{"Zero width", "Hello", 0, "…", 0},
		{"Custom ellipsis", "Hello World", 8, "...", 8},
		{"Empty ellipsis", "Hello World", 10, "", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateText(tt.text, tt.availableWidth, tt.ellipsis)
			resultLen := len([]rune(result))
			if resultLen > tt.expectedLen {
				t.Errorf("TruncateText() result length = %d, want <= %d. Got: %s", resultLen, tt.expectedLen, result)
			}
		})
	}
}

func TestTruncateWithEllipsis(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		availableWidth int
		shouldTruncate bool
	}{
		{"Short text", "Hello", 20, false},
		{"Long text", "This is a very long text that should be truncated", 10, true},
		{"Empty text", "", 10, false},
		{"Exact fit", "Hello", 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateWithEllipsis(tt.text, tt.availableWidth)
			
			// Check that result contains ellipsis if truncated
			if tt.shouldTruncate && len(tt.text) > tt.availableWidth {
				// Result should be truncated and contain ellipsis
				if len([]rune(result)) > tt.availableWidth {
					t.Errorf("TruncateWithEllipsis() result too long: %d > %d", len([]rune(result)), tt.availableWidth)
				}
			}
		})
	}
}

func TestTruncateText_UnicodeCharacters(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		availableWidth int
	}{
		{"Emoji", "Hello 🌍 World", 10},
		{"Multi-byte", "こんにちは", 5},
		{"Mixed", "Test テスト 123", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateText(tt.text, tt.availableWidth, "…")
			// Just ensure it doesn't panic and returns something reasonable
			if len([]rune(result)) > tt.availableWidth {
				t.Errorf("TruncateText() with unicode: result length = %d, want <= %d", len([]rune(result)), tt.availableWidth)
			}
		})
	}
}

func TestTruncateText_EllipsisPlacement(t *testing.T) {
	longText := "This is a very long text that definitely needs truncation"
	width := 20
	
	result := TruncateText(longText, width, "…")
	
	// Result should not exceed width
	if len([]rune(result)) > width {
		t.Errorf("Result exceeds available width: %d > %d", len([]rune(result)), width)
	}
	
	// Result should contain ellipsis if truncated
	if len([]rune(longText)) > width && !containsRune(result, '…') {
		t.Error("Expected ellipsis in truncated text")
	}
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

func TestTruncateText_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		availableWidth int
		ellipsis       string
	}{
		{"Negative width", "Hello", -5, "…"},
		{"Very large width", "Hello", 1000, "…"},
		{"Ellipsis longer than width", "Hello World", 2, "…"},
		{"Single character width", "Hello", 1, "…"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := TruncateText(tt.text, tt.availableWidth, tt.ellipsis)
			_ = result
		})
	}
}
