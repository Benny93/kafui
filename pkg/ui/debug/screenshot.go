//go:build debug

// Package debug provides debugging utilities for Kafui.
// This package is only included in debug builds.
package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// ScreenshotFormat represents the output format for screenshots
type ScreenshotFormat int

const (
	// FormatPlainText outputs plain text without ANSI codes
	FormatPlainText ScreenshotFormat = iota
	// FormatANSI outputs text with ANSI color codes preserved
	FormatANSI
)

// CaptureOptions holds options for screenshot capture
type CaptureOptions struct {
	// Format is the output format (plain text or ANSI)
	Format ScreenshotFormat

	// Redact enables sensitive data redaction
	Redact bool

	// OutputDir is the directory to save screenshots (defaults to temp dir)
	OutputDir string

	// Version is the application version
	Version string

	// CurrentPage is the current page name
	CurrentPage string

	// PageContext provides additional context about the current page
	PageContext string

	// TerminalWidth is the terminal width
	TerminalWidth int

	// TerminalHeight is the terminal height
	TerminalHeight int
}

// Capture captures the current TUI screen content to a file
func Capture(view string, options CaptureOptions) (string, error) {
	// Determine output directory
	outputDir := options.OutputDir
	if outputDir == "" {
		outputDir = os.TempDir()
	}

	// Ensure directory exists
	if err := os.MkdirAll(outputDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create screenshot directory: %w", err)
	}

	// Generate filename
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("kafui-screenshot-%s.txt", timestamp)
	filepath := filepath.Join(outputDir, filename)

	// Build content
	var content strings.Builder

	// Add metadata header
	content.WriteString(buildMetadataHeader(options))

	// Add view content
	if options.Redact {
		content.WriteString(redactSensitiveData(view))
	} else {
		content.WriteString(view)
	}

	// Write to file with restricted permissions
	if err := os.WriteFile(filepath, []byte(content.String()), 0600); err != nil {
		return "", fmt.Errorf("failed to write screenshot: %w", err)
	}

	return filepath, nil
}

// buildMetadataHeader creates the metadata header for screenshots
func buildMetadataHeader(options CaptureOptions) string {
	var sb strings.Builder

	sb.WriteString("# Kafui Debug Screenshot\n")
	sb.WriteString(fmt.Sprintf("# Timestamp: %s\n", time.Now().Format("2006-01-02 15:04:05 UTC")))
	sb.WriteString(fmt.Sprintf("# Version: %s\n", getVersion(options.Version)))
	sb.WriteString(fmt.Sprintf("# Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH))
	sb.WriteString(fmt.Sprintf("# Go Version: %s\n", runtime.Version()))
	sb.WriteString(fmt.Sprintf("# Terminal: %dx%d\n", options.TerminalWidth, options.TerminalHeight))
	sb.WriteString(fmt.Sprintf("# Current Page: %s\n", options.CurrentPage))

	if options.PageContext != "" {
		sb.WriteString(fmt.Sprintf("# Page Context: %s\n", options.PageContext))
	}

	if options.Redact {
		sb.WriteString("# Redaction: ENABLED (sensitive data masked)\n")
	}

	sb.WriteString("# " + strings.Repeat("=", 78) + "\n")
	sb.WriteString("\n")

	return sb.String()
}

// getVersion returns the version string, defaulting to "dev" if not specified
func getVersion(version string) string {
	if version == "" {
		return "dev"
	}
	return version
}

// redactSensitiveData replaces sensitive information with placeholders
func redactSensitiveData(input string) string {
	result := input

	// Redact topic names (alphanumeric with dashes/underscores)
	topicRegex := regexp.MustCompile(`\b[a-z][a-z0-9_-]*\b`)
	result = topicRegex.ReplaceAllStringFunc(result, func(s string) string {
		// Skip common words
		if isCommonWord(s) {
			return s
		}
		return fmt.Sprintf("TOPIC-%s", maskString(s))
	})

	// Redact consumer group IDs
	groupRegex := regexp.MustCompile(`consumer-group[-_]?\d+|group[-_]?\d+`)
	result = groupRegex.ReplaceAllStringFunc(result, func(s string) string {
		return "GROUP-MASKED"
	})

	// Redact message content (quoted strings that look like data)
	messageRegex := regexp.MustCompile(`"[^"]{20,}"`)
	result = messageRegex.ReplaceAllStringFunc(result, func(s string) string {
		return `"MSG-MASKED"`
	})

	// Redact IP addresses
	ipRegex := regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	result = ipRegex.ReplaceAllStringFunc(result, func(s string) string {
		return "IP-MASKED"
	})

	// Redact UUIDs
	uuidRegex := regexp.MustCompile(`\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	result = uuidRegex.ReplaceAllStringFunc(result, func(s string) string {
		return "UUID-MASKED"
	})

	return result
}

// isCommonWord checks if a string is a common word that shouldn't be redacted
func isCommonWord(s string) bool {
	commonWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"up": true, "about": true, "into": true, "through": true,
		"topic": true, "topics": true, "group": true, "groups": true,
		"consumer": true, "consumers": true, "partition": true, "partitions": true,
		"offset": true, "offsets": true, "message": true, "messages": true,
		"key": true, "value": true, "header": true, "headers": true,
		"schema": true, "schemas": true, "context": true, "contexts": true,
		"connected": true, "disconnected": true, "loading": true, "error": true,
		"help": true, "quit": true, "back": true, "search": true,
		"pause": true, "resume": true, "refresh": true, "retry": true,
		"format": true, "metadata": true, "copy": true, "select": true,
		"enter": true, "esc": true, "ctrl": true, "shift": true,
		"page": true, "start": true, "end": true, "down": true,
		"left": true, "right": true, "home": true, "pgup": true, "pgdn": true,
	}
	return commonWords[strings.ToLower(s)]
}

// maskString creates a masked version of a string
func maskString(s string) string {
	if len(s) <= 4 {
		return "XXXX"
	}
	return fmt.Sprintf("%sXXXX", s[:min(4, len(s))])
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// StripANSICodes removes ANSI escape codes from a string
func StripANSICodes(input string) string {
	// ANSI escape code regex
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiRegex.ReplaceAllString(input, "")
}

// CaptureWithFormat captures a screenshot in the specified format
func CaptureWithFormat(view string, options CaptureOptions) (string, error) {
	// Strip ANSI codes if plain text format requested
	viewContent := view
	if options.Format == FormatPlainText {
		viewContent = StripANSICodes(view)
	}

	// Update options with processed view
	return Capture(viewContent, options)
}
