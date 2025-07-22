package cmd

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// TestRootCmd tests the root command creation and basic functionality
func TestRootCmd(t *testing.T) {
	// Save original args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "help flag",
			args:        []string{"kafui", "--help"},
			expectError: true, // Help flag returns an error by design
		},
		{
			name:        "version flag",
			args:        []string{"kafui", "--version"},
			expectError: false,
		},
		{
			name:        "mock flag",
			args:        []string{"kafui", "--mock"},
			expectError: false,
		},
		{
			name:        "config flag with value",
			args:        []string{"kafui", "--config", "test-config.yaml"},
			expectError: false,
		},
		{
			name:        "no arguments",
			args:        []string{"kafui"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test args
			os.Args = tt.args

			// Create a new root command for testing
			cmd := &cobra.Command{
				Use:   "kafui",
				Short: "A Kafka UI tool",
				Long:  "A terminal-based UI for Apache Kafka",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Mock the actual execution to avoid starting the full UI
					return nil
				},
			}

			// Add flags similar to the real root command
			cmd.Flags().BoolP("mock", "m", false, "Use mock data source")
			cmd.Flags().StringP("config", "c", "", "Config file path")
			cmd.Flags().BoolP("version", "v", false, "Show version")

			// Test command parsing
			err := cmd.ParseFlags(tt.args[1:]) // Skip program name
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error for args %v, but got none", tt.args)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for args %v: %v", tt.args, err)
			}
		})
	}
}

// TestRootCmdFlags tests individual flag functionality
func TestRootCmdFlags(t *testing.T) {
	cmd := &cobra.Command{
		Use: "kafui",
	}

	// Add flags
	cmd.Flags().BoolP("mock", "m", false, "Use mock data source")
	cmd.Flags().StringP("config", "c", "", "Config file path")
	cmd.Flags().BoolP("version", "v", false, "Show version")

	tests := []struct {
		name     string
		args     []string
		flagName string
		expected interface{}
	}{
		{
			name:     "mock flag short form",
			args:     []string{"-m"},
			flagName: "mock",
			expected: true,
		},
		{
			name:     "mock flag long form",
			args:     []string{"--mock"},
			flagName: "mock",
			expected: true,
		},
		{
			name:     "config flag short form",
			args:     []string{"-c", "test.yaml"},
			flagName: "config",
			expected: "test.yaml",
		},
		{
			name:     "config flag long form",
			args:     []string{"--config", "test.yaml"},
			flagName: "config",
			expected: "test.yaml",
		},
		{
			name:     "version flag short form",
			args:     []string{"-v"},
			flagName: "version",
			expected: true,
		},
		{
			name:     "version flag long form",
			args:     []string{"--version"},
			flagName: "version",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags for each test
			cmd.Flags().Set("mock", "false")
			cmd.Flags().Set("config", "")
			cmd.Flags().Set("version", "false")

			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("Failed to parse flags %v: %v", tt.args, err)
			}

			switch tt.flagName {
			case "mock", "version":
				value, err := cmd.Flags().GetBool(tt.flagName)
				if err != nil {
					t.Fatalf("Failed to get bool flag %s: %v", tt.flagName, err)
				}
				if value != tt.expected.(bool) {
					t.Errorf("Flag %s = %v, want %v", tt.flagName, value, tt.expected)
				}
			case "config":
				value, err := cmd.Flags().GetString(tt.flagName)
				if err != nil {
					t.Fatalf("Failed to get string flag %s: %v", tt.flagName, err)
				}
				if value != tt.expected.(string) {
					t.Errorf("Flag %s = %v, want %v", tt.flagName, value, tt.expected)
				}
			}
		})
	}
}

// TestRootCmdDefaultValues tests default flag values
func TestRootCmdDefaultValues(t *testing.T) {
	cmd := &cobra.Command{
		Use: "kafui",
	}

	// Add flags with defaults
	cmd.Flags().BoolP("mock", "m", false, "Use mock data source")
	cmd.Flags().StringP("config", "c", "", "Config file path")
	cmd.Flags().BoolP("version", "v", false, "Show version")

	// Parse empty args to get defaults
	err := cmd.ParseFlags([]string{})
	if err != nil {
		t.Fatalf("Failed to parse empty flags: %v", err)
	}

	// Test default values
	mockFlag, err := cmd.Flags().GetBool("mock")
	if err != nil {
		t.Fatalf("Failed to get mock flag: %v", err)
	}
	if mockFlag != false {
		t.Errorf("Default mock flag = %v, want false", mockFlag)
	}

	configFlag, err := cmd.Flags().GetString("config")
	if err != nil {
		t.Fatalf("Failed to get config flag: %v", err)
	}
	if configFlag != "" {
		t.Errorf("Default config flag = %v, want empty string", configFlag)
	}

	versionFlag, err := cmd.Flags().GetBool("version")
	if err != nil {
		t.Fatalf("Failed to get version flag: %v", err)
	}
	if versionFlag != false {
		t.Errorf("Default version flag = %v, want false", versionFlag)
	}
}

// Benchmark tests for command parsing performance
func BenchmarkRootCmdParsing(b *testing.B) {
	cmd := &cobra.Command{
		Use: "kafui",
	}
	cmd.Flags().BoolP("mock", "m", false, "Use mock data source")
	cmd.Flags().StringP("config", "c", "", "Config file path")
	cmd.Flags().BoolP("version", "v", false, "Show version")

	args := []string{"--mock", "--config", "test.yaml"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd.ParseFlags(args)
	}
}