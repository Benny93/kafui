package core

import (
	"github.com/Benny93/kafui/pkg/api"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
)

// Common provides shared context and dependencies across all UI components.
// This pattern ensures consistent dependency injection and makes testing easier.
type Common struct {
	// DataSource is the Kafka data source
	DataSource api.KafkaDataSource

	// Styles contains all application styles
	Styles *stylesPkg.Styles

	// Config contains UI configuration
	Config *UIConfig
}

// UIConfig contains UI-specific configuration
type UIConfig struct {
	// ShowSidebar indicates whether sidebar should be shown by default
	ShowSidebar bool

	// CompactMode indicates whether compact layout mode is enabled
	CompactMode bool

	// Theme name (e.g., "dark", "light")
	Theme string
}

// DefaultUIConfig returns the default UI configuration
func DefaultUIConfig() *UIConfig {
	return &UIConfig{
		ShowSidebar: true,
		CompactMode: false,
		Theme:       "dark",
	}
}

// NewCommon creates a new Common context with the given data source.
// It initializes default styles and configuration.
func NewCommon(dataSource api.KafkaDataSource) *Common {
	return &Common{
		DataSource: dataSource,
		Styles:     stylesPkg.DefaultStyles(),
		Config:     DefaultUIConfig(),
	}
}

// NewCommonWithConfig creates a new Common context with custom configuration
func NewCommonWithConfig(dataSource api.KafkaDataSource, config *UIConfig) *Common {
	return &Common{
		DataSource: dataSource,
		Styles:     stylesPkg.DefaultStyles(),
		Config:     config,
	}
}
