//go:build !debug

// Package debug provides debugging utilities for Kafui.
// This package is only included in debug builds.
// This is a stub implementation for production builds.
package debug

import (
	"fmt"
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

// Capture is a stub implementation that returns an error in production builds
func Capture(view string, options CaptureOptions) (string, error) {
	return "", fmt.Errorf("debug screenshot feature is only available in debug builds (build with -tags debug)")
}

// CaptureWithFormat is a stub implementation for production builds
func CaptureWithFormat(view string, options CaptureOptions) (string, error) {
	return "", fmt.Errorf("debug screenshot feature is only available in debug builds (build with -tags debug)")
}

// StripANSICodes removes ANSI escape codes from a string
func StripANSICodes(input string) string {
	return input
}
