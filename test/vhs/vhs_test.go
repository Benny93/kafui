package vhs_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestVHS_TopicNavigation runs the VHS integration test for topic navigation
// This test requires:
// 1. VHS installed: go install github.com/charmbracelet/vhs@latest
// 2. A running Kafka instance or mock data
// 3. Terminal access (not available in CI without proper setup)
func TestVHS_TopicNavigation(t *testing.T) {
	// Skip in CI environment or when VHS is not available
	if os.Getenv("CI") != "" {
		t.Skip("Skipping VHS test in CI environment")
	}

	// Check if VHS is installed
	vhsPath, err := exec.LookPath("vhs")
	if err != nil {
		t.Skip("VHS not installed. Install with: go install github.com/charmbracelet/vhs@latest")
	}

	t.Logf("Using VHS from: %s", vhsPath)

	// Get the project root directory
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	tapesDir := filepath.Join(projectRoot, "test", "vhs", "tapes")
	outputDir := filepath.Join(projectRoot, "test", "vhs", "output")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Change to project root directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}

	// Run VHS on the topic navigation tape
	tapeFile := filepath.Join(tapesDir, "topic_navigation_mock.tape")
	outputFile := filepath.Join(outputDir, "topic_navigation.gif")

	// Check if tape file exists
	if _, err := os.Stat(tapeFile); os.IsNotExist(err) {
		t.Skipf("Tape file not found: %s", tapeFile)
	}

	t.Logf("Running VHS tape: %s", tapeFile)
	t.Logf("Output will be saved to: %s", outputFile)

	// Run VHS with timeout
	cmd := exec.Command(vhsPath, tapeFile, "--output", outputFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up environment
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
	)

	// Run with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("VHS test failed: %v", err)
		}
		t.Log("VHS test completed successfully")

		// Verify output file was created
		if _, err := os.Stat(outputFile); err == nil {
			t.Logf("GIF output created: %s", outputFile)
		}

	case <-time.After(5 * time.Minute):
		t.Fatal("VHS test timed out after 5 minutes")
	}
}

// TestVHS_TopicNavigationWithRealData runs the VHS test with real Kafka data
// This requires a running Kafka instance with test.users topic
func TestVHS_TopicNavigationWithRealData(t *testing.T) {
	// Skip by default - requires real Kafka instance
	t.Skip("Skipping test with real data. Requires running Kafka with test.users topic")

	// Check if VHS is installed
	vhsPath, err := exec.LookPath("vhs")
	if err != nil {
		t.Skip("VHS not installed")
	}

	// Get the project root directory
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	tapesDir := filepath.Join(projectRoot, "test", "vhs", "tapes")
	outputDir := filepath.Join(projectRoot, "test", "vhs", "output")

	// Create output directory
	os.MkdirAll(outputDir, 0755)

	// Change to project root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(projectRoot)

	// Run VHS on the real data tape
	tapeFile := filepath.Join(tapesDir, "topic_navigation.tape")
	outputFile := filepath.Join(outputDir, "topic_navigation_real.gif")

	cmd := exec.Command(vhsPath, tapeFile, "--output", outputFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	if err := cmd.Run(); err != nil {
		t.Fatalf("VHS test with real data failed: %v", err)
	}

	t.Logf("GIF output created: %s", outputFile)
}

// TestVHS_ValidateTapes validates that all VHS tape files exist and are readable
func TestVHS_ValidateTapes(t *testing.T) {
	// Get the project root directory
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	tapesDir := filepath.Join(projectRoot, "test", "vhs", "tapes")

	// Find all .tape files
	tapeFiles, err := filepath.Glob(filepath.Join(tapesDir, "*.tape"))
	if err != nil {
		t.Fatalf("Failed to glob tape files: %v", err)
	}

	if len(tapeFiles) == 0 {
		t.Skip("No tape files found")
	}

	for _, tapeFile := range tapeFiles {
		t.Run(filepath.Base(tapeFile), func(t *testing.T) {
			// Check file exists and is readable
			content, err := os.ReadFile(tapeFile)
			if err != nil {
				t.Errorf("Failed to read tape file %s: %v", filepath.Base(tapeFile), err)
				return
			}

			// Basic validation: file should not be empty
			if len(content) == 0 {
				t.Error("Tape file is empty")
				return
			}

			// Check for required VHS commands
			contentStr := string(content)
			hasContent := false
			
			// Look for common VHS commands
			commands := []string{"Type", "Enter", "Sleep", "Down", "Up", "Left", "Right", "Escape", "Set "}
			for _, cmd := range commands {
				if strings.Contains(contentStr, cmd) {
					hasContent = true
					break
				}
			}

			if !hasContent {
				t.Error("Tape file doesn't contain any recognizable VHS commands")
				return
			}

			t.Logf("Tape %s validated successfully (%d bytes)", filepath.Base(tapeFile), len(content))
		})
	}
}

// TestVHS_GenerateReadme generates a README for the VHS tests
func TestVHS_GenerateReadme(t *testing.T) {
	// This is a helper test to generate documentation
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	readmePath := filepath.Join(projectRoot, "test", "vhs", "README.md")

	readmeContent := `# VHS Integration Tests for Kafui

This directory contains VHS integration tests for the Kafui Kafka UI application.

## What is VHS?

VHS is a tool for creating terminal GIFs and testing terminal applications.
It allows you to write test scenarios in a simple tape format and replay them
to verify application behavior.

See: https://github.com/charmbracelet/vhs

## Prerequisites

1. Install VHS:
   go install github.com/charmbracelet/vhs@latest

2. Docker (optional, for running Kafka locally)

## Quick Start

Run topic navigation test with mock data:
   go test ./test/vhs/... -run TestVHS_TopicNavigation -v

## Test Scenarios

### topic_navigation_mock.tape

Tests basic topic navigation using mock data.
Duration: ~30 seconds

### topic_navigation.tape

Tests topic navigation with real Kafka data.
Duration: ~45 seconds
Requirements: Running Kafka instance on localhost:9092

## Running Tests

All VHS Tests:
   go test ./test/vhs/... -v

Individual Tests:
   go test ./test/vhs/... -run TestVHS_TopicNavigation -v
   go test ./test/vhs/... -run TestVHS_ValidateTapes -v

Using VHS Directly:
   vhs test/vhs/tapes/topic_navigation_mock.tape
   vhs --validate test/vhs/tapes/topic_navigation_mock.tape

## Keyboard Shortcuts Tested

Navigation:
- Up/Down arrows - Navigate lists
- j/k - Vim-style navigation  
- Home/End - Jump to top/bottom
- PageUp/PageDown - Page navigation

Actions:
- Enter - Select/open item
- Escape - Go back
- q - Quit application
- r - Refresh data

## Troubleshooting

VHS not installed:
   go install github.com/charmbracelet/vhs@latest

Kafka connection failed:
   docker-compose -f test/docker/docker-compose.yml up -d

## Resources

- VHS Documentation: https://github.com/charmbracelet/vhs
- VHS Examples: https://github.com/charmbracelet/vhs/tree/main/examples
`

	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}

	t.Logf("README generated at: %s", readmePath)
}

// Helper function to check if Kafka is available
func isKafkaAvailable(t *testing.T) bool {
	t.Helper()

	// Try to connect to Kafka
	cmd := exec.Command("nc", "-z", "localhost", "9092")
	if err := cmd.Run(); err != nil {
		return false
	}

	// Check if test.users topic exists
	// This would require kaf or similar tool
	return true
}
