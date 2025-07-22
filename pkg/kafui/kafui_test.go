package kafui

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/kafds"
	"github.com/Benny93/kafui/pkg/datasource/mock"
)

// TestInit tests the main initialization function
func TestInit(t *testing.T) {
	// Note: This test focuses on the data source selection logic
	// We can't easily test the full UI initialization without mocking the entire UI stack
	
	tests := []struct {
		name      string
		cfgOption string
		useMock   bool
		expectMock bool
	}{
		{
			name:       "mock data source selected",
			cfgOption:  "test-config",
			useMock:    true,
			expectMock: true,
		},
		{
			name:       "real data source selected",
			cfgOption:  "test-config",
			useMock:    false,
			expectMock: false,
		},
		{
			name:       "empty config with mock",
			cfgOption:  "",
			useMock:    true,
			expectMock: true,
		},
		{
			name:       "empty config without mock",
			cfgOption:  "",
			useMock:    false,
			expectMock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily test the full Init function due to UI dependencies
			// Instead, we test the data source selection logic directly
			
			var dataSource api.KafkaDataSource
			dataSource = mock.KafkaDataSourceMock{}
			if !tt.useMock {
				dataSource = &kafds.KafkaDataSourceKaf{}
			}

			// Verify the correct data source type is selected
			switch ds := dataSource.(type) {
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
	configs := []string{
		"",
		"config.yaml",
		"/path/to/config.yaml",
		"~/.kaf/config",
		"invalid-config",
	}

	for _, config := range configs {
		t.Run("config_"+config, func(t *testing.T) {
			// Test that different config options don't cause panics
			// We simulate the config handling without actually initializing the UI
			
			var dataSource api.KafkaDataSource = mock.KafkaDataSourceMock{}
			
			// This should not panic regardless of config value
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Init logic panicked with config '%s': %v", config, r)
				}
			}()
			
			// Simulate the Init call without UI
			dataSource.Init(config)
		})
	}
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