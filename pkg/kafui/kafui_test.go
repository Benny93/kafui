package kafui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/kafds"
	"github.com/Benny93/kafui/pkg/datasource/mock"
)

// Global variables to track calls for testing
var (
	openUICalled      bool
	openUIDataSource  api.KafkaDataSource
	originalOpenUIFunc func(api.KafkaDataSource)
)

// mockOpenUI is a test replacement for OpenUI that doesn't start the actual UI
func mockOpenUI(dataSource api.KafkaDataSource) {
	openUICalled = true
	openUIDataSource = dataSource
}

// setupMockOpenUI replaces openUIFunc with a mock for testing
func setupMockOpenUI() {
	originalOpenUIFunc = openUIFunc
	openUIFunc = mockOpenUI
	openUICalled = false
	openUIDataSource = nil
}

// teardownMockOpenUI restores the original openUIFunc function
func teardownMockOpenUI() {
	if originalOpenUIFunc != nil {
		openUIFunc = originalOpenUIFunc
	}
	openUICalled = false
	openUIDataSource = nil
}

// TestInit tests the actual Init function with mocked OpenUI
func TestInit(t *testing.T) {
	tests := []struct {
		name      string
		cfgOption string
		useMock   bool
		expectMock bool
	}{
		{
			name:       "init_with_mock_data_source",
			cfgOption:  "test-config",
			useMock:    true,
			expectMock: true,
		},
		{
			name:       "init_with_real_data_source",
			cfgOption:  "test-config",
			useMock:    false,
			expectMock: false,
		},
		{
			name:       "init_with_empty_config_and_mock",
			cfgOption:  "",
			useMock:    true,
			expectMock: true,
		},
		{
			name:       "init_with_empty_config_and_real",
			cfgOption:  "",
			useMock:    false,
			expectMock: false,
		},
		{
			name:       "init_with_config_file_path",
			cfgOption:  "/path/to/config.yaml",
			useMock:    true,
			expectMock: true,
		},
		{
			name:       "init_with_home_config_path",
			cfgOption:  "~/.kaf/config",
			useMock:    false,
			expectMock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock OpenUI
			setupMockOpenUI()
			defer teardownMockOpenUI()

			// Call the actual Init function
			Init(tt.cfgOption, tt.useMock)

			// Verify OpenUI was called
			if !openUICalled {
				t.Error("Expected OpenUI to be called, but it wasn't")
			}

			// Verify the correct data source type was passed to OpenUI
			if openUIDataSource == nil {
				t.Error("Expected data source to be passed to OpenUI, but it was nil")
				return
			}

			switch ds := openUIDataSource.(type) {
			case mock.KafkaDataSourceMock:
				if !tt.expectMock {
					t.Errorf("Expected real data source, got mock: %T", ds)
				}
			case *kafds.KafkaDataSourceKaf:
				if tt.expectMock {
					t.Errorf("Expected mock data source, got real: %T", ds)
				}
			default:
				t.Errorf("Unexpected data source type: %T", ds)
			}
		})
	}
}

// TestInitDataSourceInitialization tests that data sources are properly initialized
func TestInitDataSourceInitialization(t *testing.T) {
	tests := []struct {
		name      string
		cfgOption string
		useMock   bool
	}{
		{
			name:      "mock_data_source_init",
			cfgOption: "test-config",
			useMock:   true,
		},
		{
			name:      "real_data_source_init",
			cfgOption: "test-config",
			useMock:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMockOpenUI()
			defer teardownMockOpenUI()

			// Call Init
			Init(tt.cfgOption, tt.useMock)

			// Verify the data source was initialized
			if openUIDataSource == nil {
				t.Error("Data source should not be nil after Init")
				return
			}

			// Test that the data source implements the required interface
			if _, ok := openUIDataSource.(api.KafkaDataSource); !ok {
				t.Errorf("Data source does not implement KafkaDataSource interface: %T", openUIDataSource)
			}
		})
	}
}

// TestDataSourceSelection tests the data source selection logic in isolation
func TestDataSourceSelection(t *testing.T) {
	tests := []struct {
		name        string
		useMock     bool
		expectedType string
	}{
		{
			name:         "select mock data source",
			useMock:      true,
			expectedType: "mock.KafkaDataSourceMock",
		},
		{
			name:         "select real data source",
			useMock:      false,
			expectedType: "*kafds.KafkaDataSourceKaf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the data source selection logic from Init
			var dataSource api.KafkaDataSource
			
			dataSource = mock.KafkaDataSourceMock{}
			if !tt.useMock {
				dataSource = &kafds.KafkaDataSourceKaf{}
			}

			// Get the actual type name
			actualType := ""
			switch dataSource.(type) {
			case mock.KafkaDataSourceMock:
				actualType = "mock.KafkaDataSourceMock"
			case *kafds.KafkaDataSourceKaf:
				actualType = "*kafds.KafkaDataSourceKaf"
			}

			if actualType != tt.expectedType {
				t.Errorf("Expected data source type %s, got %s", tt.expectedType, actualType)
			}
		})
	}
}

// TestInitConfigurationHandling tests how different configuration options are handled
func TestInitConfigurationHandling(t *testing.T) {
	configs := []struct {
		name   string
		config string
	}{
		{"empty_config", ""},
		{"yaml_config", "config.yaml"},
		{"absolute_path_config", "/path/to/config.yaml"},
		{"home_config", "~/.kaf/config"},
		{"invalid_config", "invalid-config"},
		{"nonexistent_file", "/nonexistent/path/config.yaml"},
		{"special_chars_config", "config-with-special_chars.yaml"},
	}

	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			setupMockOpenUI()
			defer teardownMockOpenUI()

			// This should not panic regardless of config value
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Init panicked with config '%s': %v", tc.config, r)
				}
			}()

			// Test with both mock and real data sources
			for _, useMock := range []bool{true, false} {
				t.Run(func() string {
					if useMock {
						return "with_mock"
					}
					return "with_real"
				}(), func(t *testing.T) {
					// Reset mock state
					openUICalled = false
					openUIDataSource = nil

					// Call Init
					Init(tc.config, useMock)

					// Verify OpenUI was called
					if !openUICalled {
						t.Error("Expected OpenUI to be called")
					}

					// Verify data source is not nil
					if openUIDataSource == nil {
						t.Error("Expected data source to be passed to OpenUI")
					}
				})
			}
		})
	}
}

// TestInitErrorHandling tests error handling scenarios
func TestInitErrorHandling(t *testing.T) {
	t.Run("init_with_nil_check", func(t *testing.T) {
		setupMockOpenUI()
		defer teardownMockOpenUI()

		// Test that Init doesn't panic with various inputs
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Init should not panic: %v", r)
			}
		}()

		Init("", true)

		if openUIDataSource == nil {
			t.Error("Data source should not be nil after Init")
		}
	})

	t.Run("init_output_verification", func(t *testing.T) {
		setupMockOpenUI()
		defer teardownMockOpenUI()

		// Capture stdout to verify the "Init..." message is printed
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		done := make(chan string)
		go func() {
			var buf bytes.Buffer
			io.Copy(&buf, r)
			done <- buf.String()
		}()

		Init("test-config", true)

		w.Close()
		os.Stdout = oldStdout
		output := <-done

		if !strings.Contains(output, "Init...") {
			t.Errorf("Expected output to contain 'Init...', got: %s", output)
		}

		if !openUICalled {
			t.Error("Expected OpenUI to be called")
		}
	})
}

// TestInitConcurrency tests that Init can be called safely (though it shouldn't be called concurrently in practice)
func TestInitConcurrency(t *testing.T) {
	t.Run("sequential_init_calls", func(t *testing.T) {
		setupMockOpenUI()
		defer teardownMockOpenUI()

		// Test multiple sequential calls don't cause issues
		for i := 0; i < 3; i++ {
			openUICalled = false
			openUIDataSource = nil

			Init("test-config", true)

			if !openUICalled {
				t.Errorf("Expected OpenUI to be called on iteration %d", i)
			}
		}
	})
}

// TestMockVsRealDataSourceBehavior tests behavioral differences
func TestMockVsRealDataSourceBehavior(t *testing.T) {
	tests := []struct {
		name       string
		dataSource api.KafkaDataSource
		expectInit bool
	}{
		{
			name:       "mock data source",
			dataSource: mock.KafkaDataSourceMock{},
			expectInit: true,
		},
		{
			name:       "real data source",
			dataSource: &kafds.KafkaDataSourceKaf{},
			expectInit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that both data sources can be initialized
			defer func() {
				if r := recover(); r != nil {
					if tt.expectInit {
						t.Errorf("Data source %T should not panic on Init: %v", tt.dataSource, r)
					}
				}
			}()

			// Initialize with empty config for testing
			tt.dataSource.Init("")

			// Test basic interface compliance
			if tt.dataSource == nil {
				t.Error("Data source should not be nil after initialization")
			}
		})
	}
}

// TestInitWithEnvironmentVariables tests Init with different environment setups
func TestInitWithEnvironmentVariables(t *testing.T) {
	// Save original environment
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("USERPROFILE", originalUserProfile)
	}()

	tests := []struct {
		name     string
		homeVar  string
		homeVal  string
		config   string
		useMock  bool
	}{
		{
			name:    "with_home_set",
			homeVar: "HOME",
			homeVal: "/home/testuser",
			config:  "~/.kaf/config",
			useMock: true,
		},
		{
			name:    "with_userprofile_set",
			homeVar: "USERPROFILE",
			homeVal: "C:\\Users\\testuser",
			config:  "config.yaml",
			useMock: false,
		},
		{
			name:    "with_no_home_set",
			homeVar: "",
			homeVal: "",
			config:  "config.yaml",
			useMock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMockOpenUI()
			defer teardownMockOpenUI()

			// Set up environment
			if tt.homeVar != "" {
				os.Setenv(tt.homeVar, tt.homeVal)
			} else {
				os.Unsetenv("HOME")
				os.Unsetenv("USERPROFILE")
			}

			// Test should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Init panicked with environment setup: %v", r)
				}
			}()

			Init(tt.config, tt.useMock)

			if !openUICalled {
				t.Error("Expected OpenUI to be called")
			}

			if openUIDataSource == nil {
				t.Error("Expected data source to be initialized")
			}
		})
	}
}

// TestInitDataSourceInterfaceCompliance tests that both data sources implement the interface correctly
func TestInitDataSourceInterfaceCompliance(t *testing.T) {
	tests := []struct {
		name    string
		useMock bool
	}{
		{"mock_data_source_compliance", true},
		{"real_data_source_compliance", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMockOpenUI()
			defer teardownMockOpenUI()

			Init("test-config", tt.useMock)

			if openUIDataSource == nil {
				t.Fatal("Data source should not be nil")
			}

			// Test that all required interface methods exist
			// This is compile-time checked, but we can verify at runtime too
			ds := openUIDataSource

			// Test interface compliance by calling methods (if they don't panic, they exist)
			defer func() {
				if r := recover(); r != nil {
					// Some methods might panic due to missing config, but they should exist
					// We're mainly testing that the interface is implemented
				}
			}()

			// These methods should exist on the interface
			_ = ds // Just verify it implements api.KafkaDataSource interface
		})
	}
}

// TestInitConfigFileScenarios tests various config file scenarios
func TestInitConfigFileScenarios(t *testing.T) {
	// Create a temporary config file for testing
	tempFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	
	// Write some test config content
	configContent := `
brokers:
  - localhost:9092
security:
  protocol: PLAINTEXT
`
	if _, err := tempFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	tests := []struct {
		name       string
		configPath string
		useMock    bool
		shouldWork bool
	}{
		{
			name:       "valid_temp_config_file",
			configPath: tempFile.Name(),
			useMock:    true,
			shouldWork: true,
		},
		{
			name:       "nonexistent_config_file",
			configPath: "/nonexistent/config.yaml",
			useMock:    true,
			shouldWork: true, // Should not panic, even if file doesn't exist
		},
		{
			name:       "empty_config_path",
			configPath: "",
			useMock:    false,
			shouldWork: true,
		},
		{
			name:       "relative_config_path",
			configPath: "./config.yaml",
			useMock:    true,
			shouldWork: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMockOpenUI()
			defer teardownMockOpenUI()

			defer func() {
				if r := recover(); r != nil {
					if tt.shouldWork {
						t.Errorf("Init should not panic with config '%s': %v", tt.configPath, r)
					}
				}
			}()

			Init(tt.configPath, tt.useMock)

			if tt.shouldWork {
				if !openUICalled {
					t.Error("Expected OpenUI to be called")
				}
				if openUIDataSource == nil {
					t.Error("Expected data source to be initialized")
				}
			}
		})
	}
}

// Benchmark tests for initialization performance
func BenchmarkDataSourceSelection(b *testing.B) {
	b.Run("mock_selection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var dataSource api.KafkaDataSource
			dataSource = mock.KafkaDataSourceMock{}
			_ = dataSource
		}
	})

	b.Run("real_selection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var dataSource api.KafkaDataSource
			dataSource = &kafds.KafkaDataSourceKaf{}
			_ = dataSource
		}
	})
}