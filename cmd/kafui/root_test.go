package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/Benny93/kafui/pkg/ui"
	"github.com/stretchr/testify/assert"
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
	mockInit := func(opts ui.InitOptions) {
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

	mockInit := func(opts ui.InitOptions) {
		initCalled = true
		receivedConfig = opts.ConfigFile
		receivedMock = opts.Mock
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

// TestOverrideFlagsReachInit verifies broker/schema-registry/cluster overrides
// are parsed and threaded into InitOptions.
func TestOverrideFlagsReachInit(t *testing.T) {
	var got ui.InitOptions
	mockInit := func(opts ui.InitOptions) { got = opts }

	brokersFlag = nil
	schemaRegistryURL = ""
	clusterFlag = ""

	cmd := CreateRootCommand(mockInit)
	if err := cmd.ParseFlags([]string{"--brokers", "a:9092,b:9092", "--schema-registry", "http://sr:8081", "--cluster", "prod"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	cmd.Run(cmd, []string{})

	if len(got.Brokers) != 2 || got.Brokers[0] != "a:9092" {
		t.Errorf("brokers = %v, want [a:9092 b:9092]", got.Brokers)
	}
	if got.SchemaRegistry != "http://sr:8081" {
		t.Errorf("schema-registry = %s", got.SchemaRegistry)
	}
	if got.Cluster != "prod" {
		t.Errorf("cluster = %s", got.Cluster)
	}
}

// TestReadOnlyFlagReachesInit verifies the global --read-only flag is parsed and
// threaded into InitOptions (AA-4).
func TestReadOnlyFlagReachesInit(t *testing.T) {
	var got ui.InitOptions
	mockInit := func(opts ui.InitOptions) { got = opts }

	readOnlyFlag = false
	cmd := CreateRootCommand(mockInit)
	if f := cmd.PersistentFlags().Lookup("read-only"); f == nil {
		t.Fatal("read-only flag not found")
	}
	if err := cmd.ParseFlags([]string{"--read-only"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	cmd.Run(cmd, []string{})

	if !got.ReadOnly {
		t.Errorf("InitOptions.ReadOnly = %v, want true", got.ReadOnly)
	}
}

// TestMetricsListenFlagReachesInit verifies the --metrics-listen flag is parsed
// and threaded into InitOptions (MM-16).
func TestMetricsListenFlagReachesInit(t *testing.T) {
	var got ui.InitOptions
	mockInit := func(opts ui.InitOptions) { got = opts }

	metricsListenFlag = ""
	cmd := CreateRootCommand(mockInit)
	if f := cmd.PersistentFlags().Lookup("metrics-listen"); f == nil {
		t.Fatal("metrics-listen flag not found")
	}
	if err := cmd.ParseFlags([]string{"--metrics-listen", ":9090"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	cmd.Run(cmd, []string{})

	if got.MetricsListen != ":9090" {
		t.Errorf("InitOptions.MetricsListen = %q, want %q", got.MetricsListen, ":9090")
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
	defaultKafuiInit = func(opts ui.InitOptions) {
		initCalled = true
		receivedMock = opts.Mock
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

// TestRootCommandSilencesErrorsAndUsage guards against bug #2 regressing: a
// subcommand error must not make cobra print its own "Error:"/usage on top of
// DoExecute's single message, and must not panic.
func TestRootCommandSilencesErrorsAndUsage(t *testing.T) {
	cmd := CreateRootCommand(func(ui.InitOptions) {})
	assert.True(t, cmd.SilenceErrors, "SilenceErrors should be set so cobra doesn't double-print")
	assert.True(t, cmd.SilenceUsage, "SilenceUsage should be set so a runtime failure doesn't dump usage")

	cmd.SetArgs([]string{"get", "brokers", "--format", "json"})
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	err := cmd.Execute()
	assert.Error(t, err, "unsupported format should surface as an error, not a panic")
}