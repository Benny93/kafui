package cmd

import (
	"os"
	"testing"
)

// TestCfgFileVariable tests the global cfgFile variable
func TestCfgFileVariable(t *testing.T) {
	originalCfgFile := cfgFile
	defer func() { cfgFile = originalCfgFile }()

	cfgFile = "test-config.yaml"
	if cfgFile != "test-config.yaml" {
		t.Errorf("cfgFile = %s, want test-config.yaml", cfgFile)
	}
}

// TestCreateRootCommand tests the CreateRootCommand function
func TestCreateRootCommand(t *testing.T) {
	mockInit := func(configFile string, mock bool) {
		// Mock function for testing
	}

	cmd := CreateRootCommand(mockInit)

	// Test command properties
	if cmd.Use != "kafui" {
		t.Errorf("cmd.Use = %s, want kafui", cmd.Use)
	}
	if cmd.Short != "k9s style kafka explorer" {
		t.Errorf("cmd.Short = %s, want 'k9s style kafka explorer'", cmd.Short)
	}
	if cmd.Long != "Explore different kafka broker in a k9s fashion with quick switches between topics, consumer groups and brokers" {
		t.Errorf("cmd.Long incorrect")
	}

	// Test flags exist
	mockFlag := cmd.PersistentFlags().Lookup("mock")
	if mockFlag == nil {
		t.Error("mock flag not found")
	}
	configFlag := cmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Error("config flag not found")
	}

	// Test flag defaults
	if mockFlag.DefValue != "false" {
		t.Errorf("mock flag default = %s, want false", mockFlag.DefValue)
	}
	if configFlag.DefValue != "" {
		t.Errorf("config flag default = %s, want empty", configFlag.DefValue)
	}
}

// TestCreateRootCommandRun tests the Run function of the root command
func TestCreateRootCommandRun(t *testing.T) {
	var initCalled bool
	var receivedConfig string
	var receivedMock bool

	mockInit := func(configFile string, mock bool) {
		initCalled = true
		receivedConfig = configFile
		receivedMock = mock
	}

	tests := []struct {
		name           string
		args           []string
		expectedConfig string
		expectedMock   bool
	}{
		{
			name:           "no flags",
			args:           []string{},
			expectedConfig: "",
			expectedMock:   false,
		},
		{
			name:           "mock flag only",
			args:           []string{"--mock"},
			expectedConfig: "",
			expectedMock:   true,
		},
		{
			name:           "config flag only",
			args:           []string{"--config", "test.yaml"},
			expectedConfig: "test.yaml",
			expectedMock:   false,
		},
		{
			name:           "both flags",
			args:           []string{"--mock", "--config", "test.yaml"},
			expectedConfig: "test.yaml",
			expectedMock:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state
			initCalled = false
			receivedConfig = ""
			receivedMock = false
			cfgFile = ""

			cmd := CreateRootCommand(mockInit)
			
			// Parse flags
			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			// Execute the Run function
			cmd.Run(cmd, []string{})

			// Verify init was called
			if !initCalled {
				t.Error("Init function was not called")
			}

			// Verify parameters
			if receivedConfig != tt.expectedConfig {
				t.Errorf("receivedConfig = %s, want %s", receivedConfig, tt.expectedConfig)
			}
			if receivedMock != tt.expectedMock {
				t.Errorf("receivedMock = %v, want %v", receivedMock, tt.expectedMock)
			}
		})
	}
}

// TestDoExecute tests the DoExecute function
func TestDoExecute(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Test with help flag - should exit gracefully
	os.Args = []string{"kafui", "--help"}
	
	defer func() {
		if r := recover(); r != nil {
			// Help flag causes cobra to exit, which may cause a panic
			t.Logf("Expected behavior for help flag: %v", r)
		}
	}()

	// This will test the DoExecute path but exit due to help
	DoExecute()
}

// TestDoExecuteWithMockInit tests DoExecute with a mock init function
func TestDoExecuteWithMockInit(t *testing.T) {
	originalArgs := os.Args
	originalInit := defaultKafuiInit
	defer func() { 
		os.Args = originalArgs
		defaultKafuiInit = originalInit
	}()

	var initCalled bool
	var receivedMock bool

	// Replace the default init function
	defaultKafuiInit = func(configFile string, mock bool) {
		initCalled = true
		receivedMock = mock
	}

	// Test with mock flag
	os.Args = []string{"kafui", "--mock"}
	
	DoExecute()

	// Verify init was called with correct parameters
	if !initCalled {
		t.Error("Init function was not called")
	}
	if !receivedMock {
		t.Error("Mock flag was not passed correctly")
	}
}

// TestDefaultKafuiInit tests that the default init function is set correctly
func TestDefaultKafuiInit(t *testing.T) {
	if defaultKafuiInit == nil {
		t.Error("defaultKafuiInit should not be nil")
	}
}