//go:build debug

package debug

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCapture_PlainText(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	view := "Hello World"
	options := CaptureOptions{
		Format:          FormatPlainText,
		Redact:          false,
		OutputDir:       tempDir,
		Version:         "test-v1.0.0",
		CurrentPage:     "test_page",
		PageContext:     "test=context",
		TerminalWidth:   80,
		TerminalHeight:  24,
	}

	filepath, err := Capture(view, options)
	assert.NoError(t, err)
	assert.NotEmpty(t, filepath)

	// Verify file exists
	content, err := os.ReadFile(filepath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "Hello World")
	assert.Contains(t, string(content), "test-v1.0.0")
	assert.Contains(t, string(content), "test_page")

	// Verify file permissions
	info, err := os.Stat(filepath)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestCapture_ANSI(t *testing.T) {
	tempDir := t.TempDir()

	view := "\x1b[31mRed Text\x1b[0m"
	options := CaptureOptions{
		Format:    FormatANSI,
		OutputDir: tempDir,
	}

	filepath, err := Capture(view, options)
	assert.NoError(t, err)

	content, err := os.ReadFile(filepath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "\x1b[31m")
}

func TestCapture_Redacted(t *testing.T) {
	tempDir := t.TempDir()

	view := "Topic: my-sensitive-topic-123\nMessage: \"secret data here\""
	options := CaptureOptions{
		Format:    FormatPlainText,
		Redact:    true,
		OutputDir: tempDir,
	}

	filepath, err := Capture(view, options)
	assert.NoError(t, err)

	content, err := os.ReadFile(filepath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "Redaction: ENABLED")
	assert.NotContains(t, string(content), "my-sensitive-topic-123")
	assert.Contains(t, string(content), "TOPIC-")
}

func TestCapture_DefaultTempDir(t *testing.T) {
	view := "Test"
	options := CaptureOptions{
		Format: FormatPlainText,
	}

	filepath, err := Capture(view, options)
	assert.NoError(t, err)
	assert.NotEmpty(t, filepath)

	// Verify file is in temp directory
	tempDir := os.TempDir()
	assert.True(t, strings.HasPrefix(filepath, tempDir))

	// Cleanup
	os.Remove(filepath)
}

func TestCapture_MetadataHeader(t *testing.T) {
	tempDir := t.TempDir()

	view := "Content"
	options := CaptureOptions{
		Format:          FormatPlainText,
		OutputDir:       tempDir,
		Version:         "v2.0.0",
		CurrentPage:     "main",
		PageContext:     "cluster=prod",
		TerminalWidth:   120,
		TerminalHeight:  40,
	}

	filepath, err := Capture(view, options)
	assert.NoError(t, err)

	content, err := os.ReadFile(filepath)
	assert.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "# Kafui Debug Screenshot")
	assert.Contains(t, contentStr, "Version: v2.0.0")
	assert.Contains(t, contentStr, "Current Page: main")
	assert.Contains(t, contentStr, "Page Context: cluster=prod")
	assert.Contains(t, contentStr, "120x40")
}

func TestStripANSICodes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no codes",
			input:    "Plain text",
			expected: "Plain text",
		},
		{
			name:     "single code",
			input:    "\x1b[31mRed\x1b[0m",
			expected: "Red",
		},
		{
			name:     "multiple codes",
			input:    "\x1b[1m\x1b[32mGreen Bold\x1b[0m",
			expected: "Green Bold",
		},
		{
			name:     "complex formatting",
			input:    "\x1b[38;5;196mRed\x1b[0m \x1b[38;5;46mGreen\x1b[0m",
			expected: "Red Green",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripANSICodes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactSensitiveData(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		check    string // String to check is present (not exact match)
	}{
		{
			name:     "topic name",
			input:    "Topic: my-topic-123",
			check:    "TOPIC-",
		},
		{
			name:     "consumer group",
			input:    "Consumer group-456",
			check:    "TOPIC-", // Will be redacted as topic-like word
		},
		{
			name:     "IP address",
			input:    "Server: 192.168.1.100",
			check:    "IP-MASKED",
		},
		{
			name:     "UUID",
			input:    "ID: 550e8400-e29b-41d4-a716-446655440000",
			check:    "TOPIC-", // Will be partially redacted
		},
		{
			name:     "common words preserved",
			input:    "The topic is connected",
			check:    "The topic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactSensitiveData(tt.input)
			assert.Contains(t, result, tt.check)
		})
	}
}

func TestIsCommonWord(t *testing.T) {
	commonWords := []string{"the", "topic", "topics", "connected", "help", "quit"}
	for _, word := range commonWords {
		assert.True(t, isCommonWord(word), "%s should be common", word)
	}

	uncommonWords := []string{"my-topic", "secret-data", "xyz123"}
	for _, word := range uncommonWords {
		assert.False(t, isCommonWord(word), "%s should not be common", word)
	}
}

func TestMaskString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abc", "XXXX"},
		{"abcd", "XXXX"},
		{"abcde", "abcdXXXX"},
		{"my-topic", "my-tXXXX"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := maskString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCapture_FilePermissions(t *testing.T) {
	tempDir := t.TempDir()

	view := "Test content"
	options := CaptureOptions{
		Format:    FormatPlainText,
		OutputDir: tempDir,
	}

	filepath, err := Capture(view, options)
	assert.NoError(t, err)

	// Check file permissions
	info, err := os.Stat(filepath)
	assert.NoError(t, err)

	// Should be owner read/write only (0600)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestCapture_FilenameFormat(t *testing.T) {
	tempDir := t.TempDir()

	view := "Test"
	options := CaptureOptions{
		Format:    FormatPlainText,
		OutputDir: tempDir,
	}

	screenshotPath, err := Capture(view, options)
	assert.NoError(t, err)

	// Verify filename format
	filename := filepath.Base(screenshotPath)
	assert.True(t, strings.HasPrefix(filename, "kafui-screenshot-"))
	assert.True(t, strings.HasSuffix(filename, ".txt"))
}

func TestCaptureWithFormat(t *testing.T) {
	tempDir := t.TempDir()

	view := "\x1b[31mColored\x1b[0m"

	// Test plain text format
	options := CaptureOptions{
		Format:    FormatPlainText,
		OutputDir: tempDir,
	}

	filepath, err := CaptureWithFormat(view, options)
	assert.NoError(t, err)

	content, err := os.ReadFile(filepath)
	assert.NoError(t, err)
	assert.NotContains(t, string(content), "\x1b[31m")

	// Test ANSI format
	options.Format = FormatANSI
	filepath2, err := CaptureWithFormat(view, options)
	assert.NoError(t, err)

	content2, err := os.ReadFile(filepath2)
	assert.NoError(t, err)
	assert.Contains(t, string(content2), "\x1b[31m")
}

func TestCapture_NestedDirectory(t *testing.T) {
	tempDir := t.TempDir()
	nestedDir := filepath.Join(tempDir, "subdir", "nested")

	view := "Test"
	options := CaptureOptions{
		Format:    FormatPlainText,
		OutputDir: nestedDir,
	}

	filepath, err := Capture(view, options)
	assert.NoError(t, err)
	assert.NotEmpty(t, filepath)

	// Verify file exists in nested directory
	_, err = os.Stat(filepath)
	assert.NoError(t, err)
}
