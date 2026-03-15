package core

import (
	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/layout"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
)

// Common provides shared context and dependencies across all UI components.
// This pattern ensures consistent dependency injection and makes testing easier.
type Common struct {
	// DataSource is the Kafka data source
	DataSource api.KafkaDataSource

	// Styles contains all application styles
	Styles *stylesPkg.Styles

	// Layout contains the current layout configuration
	Layout *layout.Layout

	// LayoutConfig contains layout configuration options
	LayoutConfig *layout.LayoutConfig

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

	// ScreenshotDir is the directory for debug screenshots (defaults to temp dir)
	ScreenshotDir string
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
	layoutConfig := layout.DefaultLayoutConfig()
	return &Common{
		DataSource:   dataSource,
		Styles:       stylesPkg.DefaultStyles(),
		Layout:       layout.CalculateLayout(80, 24, layoutConfig), // Default size
		LayoutConfig: layoutConfig,
		Config:       DefaultUIConfig(),
	}
}

// NewCommonWithConfig creates a new Common context with custom configuration
func NewCommonWithConfig(dataSource api.KafkaDataSource, config *UIConfig) *Common {
	layoutConfig := layout.DefaultLayoutConfig()
	return &Common{
		DataSource:   dataSource,
		Styles:       stylesPkg.DefaultStyles(),
		Layout:       layout.CalculateLayout(80, 24, layoutConfig), // Default size
		LayoutConfig: layoutConfig,
		Config:       config,
	}
}

// UpdateLayout recalculates the layout based on new dimensions
func (c *Common) UpdateLayout(width, height int) {
	c.Layout = layout.CalculateLayout(width, height, c.LayoutConfig)
}

// GetLayout returns the current layout, calculating it if necessary
func (c *Common) GetLayout(width, height int) *layout.Layout {
	if c.Layout == nil || c.Layout.Width != width || c.Layout.Height != height {
		c.UpdateLayout(width, height)
	}
	return c.Layout
}
