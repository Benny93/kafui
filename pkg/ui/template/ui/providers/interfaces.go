package providers

import tea "github.com/charmbracelet/bubbletea"

// ContentProvider defines the interface for providing main content
type ContentProvider interface {
	// RenderContent returns the content to display in the main content area
	// width is already capped for readability, height accounts for borders
	RenderContent(width, height int) string

	// HandleContentUpdate allows the provider to handle messages and return commands
	HandleContentUpdate(msg tea.Msg) tea.Cmd

	// InitContent initializes the content provider
	InitContent() tea.Cmd

	// GetContentSize returns the actual content size (number of lines) for scrollbar calculation
	GetContentSize(width int) int

	// IsInputMode returns true when the provider has an active text input (e.g. search bar).
	// While in input mode, app-level hotkeys must not be intercepted so that every
	// keystroke reaches the input field unmodified.
	IsInputMode() bool
}

// PageActionsProvider is an optional interface a ContentProvider can implement
// to advertise its context actions (e.g. "d delete • e edit") in the header/
// breadcrumb bar's page-actions slot (UI-11). Feature plans populate this so the
// available page-level operations are consistently surfaced across pages.
type PageActionsProvider interface {
	// PageActions returns a short, pre-rendered actions string, or "" for none.
	PageActions() string
}

// SidebarSection defines the interface for sidebar sections
type SidebarSection interface {
	// GetTitle returns the section title
	GetTitle() string
	
	// RenderItems returns the items to display in this section
	RenderItems(maxItems, width int) []SidebarItem
	
	// HandleSectionUpdate allows the section to handle messages and return commands
	HandleSectionUpdate(msg tea.Msg) tea.Cmd
	
	// InitSection initializes the section
	InitSection() tea.Cmd
	
	// RefreshSection refreshes the section data
	RefreshSection() tea.Cmd
}

// SidebarItem represents an item within a sidebar section
type SidebarItem struct {
	// Icon is the status icon (●, ○, ×, ⚠, ✓, etc.)
	Icon string

	// Text is the main text content
	Text string

	// Value is optional secondary text (size, percentage, etc.)
	Value string

	// Status determines the color styling ("success", "error", "warning", "info", "muted")
	Status string

	// ZoneID is an optional bubblezone ID. When set the sidebar renderer wraps
	// the rendered line with zone.Mark so that mouse clicks can be detected.
	ZoneID string
}

// HeaderDataProvider defines the interface for providing header data
type HeaderDataProvider interface {
	// GetBrandName returns the brand name to display
	GetBrandName() string
	
	// GetAppName returns the application name to display
	GetAppName() string
	
	// GetStatusData returns status information for the header
	GetStatusData() map[string]interface{}
	
	// HandleHeaderUpdate allows the provider to handle messages and return commands
	HandleHeaderUpdate(msg tea.Msg) tea.Cmd
	
	// InitHeader initializes the header provider
	InitHeader() tea.Cmd
}

// AppConfig holds the configuration for the reusable app
type AppConfig struct {
	// ContentProvider provides the main content
	ContentProvider ContentProvider
	
	// HeaderDataProvider provides header data
	HeaderDataProvider HeaderDataProvider
	
	// SidebarSections is a list of sections to display in the sidebar
	SidebarSections []SidebarSection
	
	// ShowSidebarByDefault determines if sidebar is shown initially
	ShowSidebarByDefault bool
	
	// CompactModeBreakpoints allows customizing when compact mode activates
	CompactModeWidthBreakpoint  int
	CompactModeHeightBreakpoint int
}