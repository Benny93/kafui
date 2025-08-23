package shared

import (
	"github.com/Benny93/kafui/pkg/api"
	"github.com/charmbracelet/bubbles/table"
)

// Common interface definitions used across UI components

// ResourceItem represents a displayable resource item
type ResourceItem interface {
	GetID() string
	GetValues() []string
	GetDetails() map[string]string
}

// FilterableItem represents an item that can be filtered
type FilterableItem interface {
	FilterValue() string
}

// SelectableItem represents an item that can be selected
type SelectableItem interface {
	GetID() string
	IsSelected() bool
	SetSelected(selected bool)
}

// HighlightableItem represents an item that can have highlighted content
type HighlightableItem interface {
	GetDisplayText() string
	SetHighlight(query string)
	ClearHighlight()
}

// Common data structures

// PageState represents the current state of a page
type PageState struct {
	Loading    bool
	Error      error
	FilterMode bool
	SearchMode bool
	EditMode   bool
}

// TableConfig represents table configuration
type TableConfig struct {
	Columns     []table.Column
	Styles      table.Styles
	Width       int
	Height      int
	ShowHeader  bool
	Selectable  bool
}

// SearchConfig represents search configuration
type SearchConfig struct {
	CaseSensitive bool
	UseRegex      bool
	UseFuzzy      bool
	HighlightMatches bool
}

// ViewDimensions represents view dimensions and layout information
type ViewDimensions struct {
	Width         int
	Height        int
	ContentWidth  int
	ContentHeight int
	SidebarWidth  int
	FooterHeight  int
	HeaderHeight  int
}

// StatusInfo represents status information for display
type StatusInfo struct {
	Message    string
	Type       StatusType
	Timestamp  string
	Persistent bool
}

// StatusType represents different types of status messages
type StatusType int

const (
	StatusTypeInfo StatusType = iota
	StatusTypeSuccess
	StatusTypeWarning
	StatusTypeError
)

// Navigation represents navigation state and history
type Navigation struct {
	Current  string
	Previous []string
	CanGoBack bool
}

// ResourceListItem wraps a resource item for list display
type ResourceListItem struct {
	ResourceItem ResourceItem
	Selected     bool
	Highlighted  bool
	SearchQuery  string
}

func (r ResourceListItem) FilterValue() string {
	return r.ResourceItem.GetID()
}

func (r ResourceListItem) GetID() string {
	return r.ResourceItem.GetID()
}

func (r ResourceListItem) IsSelected() bool {
	return r.Selected
}

func (r *ResourceListItem) SetSelected(selected bool) {
	r.Selected = selected
}

// TopicListItem represents a topic item for list display
type TopicListItem struct {
	Name         string
	Topic        api.Topic
	Selected     bool
	Highlighted  bool
	SearchQuery  string
}

func (t TopicListItem) FilterValue() string {
	return t.Name
}

func (t TopicListItem) GetID() string {
	return t.Name
}

func (t TopicListItem) IsSelected() bool {
	return t.Selected
}

func (t *TopicListItem) SetSelected(selected bool) {
	t.Selected = selected
}

// ConsumerGroupListItem represents a consumer group item for list display
type ConsumerGroupListItem struct {
	GroupID      string
	Group        api.ConsumerGroup
	Selected     bool
	Highlighted  bool
	SearchQuery  string
}

func (c ConsumerGroupListItem) FilterValue() string {
	return c.GroupID
}

func (c ConsumerGroupListItem) GetID() string {
	return c.GroupID
}

func (c ConsumerGroupListItem) IsSelected() bool {
	return c.Selected
}

func (c *ConsumerGroupListItem) SetSelected(selected bool) {
	c.Selected = selected
}

// MessageListItem represents a message item for display
type MessageListItem struct {
	Message      api.Message
	Index        int
	Selected     bool
	Highlighted  bool
	SearchQuery  string
}

func (m MessageListItem) FilterValue() string {
	// Filter by message content or key
	if m.Message.Key != "" {
		return m.Message.Key
	}
	return m.Message.Value
}

func (m MessageListItem) GetID() string {
	return m.Message.Key
}

func (m MessageListItem) IsSelected() bool {
	return m.Selected
}

func (m *MessageListItem) SetSelected(selected bool) {
	m.Selected = selected
}

// Common constants
const (
	DefaultTableHeight = 20
	DefaultSidebarWidth = 35
	DefaultFooterHeight = 3
	DefaultHeaderHeight = 3
	MinContentWidth = 50
	MinContentHeight = 10
)

// Common error types
type UIError struct {
	Type    string
	Message string
	Cause   error
}

func (e UIError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func NewUIError(errorType, message string, cause error) UIError {
	return UIError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}

// Common error types
const (
	ErrorTypeDataLoad    = "data_load"
	ErrorTypeValidation  = "validation"
	ErrorTypeNavigation  = "navigation"
	ErrorTypeRender      = "render"
	ErrorTypeConfiguration = "configuration"
)

// Helper functions for common operations
func CalculateContentDimensions(totalWidth, totalHeight, sidebarWidth, footerHeight, headerHeight int) ViewDimensions {
	contentWidth := totalWidth - sidebarWidth
	if contentWidth < MinContentWidth {
		contentWidth = MinContentWidth
	}

	contentHeight := totalHeight - footerHeight - headerHeight
	if contentHeight < MinContentHeight {
		contentHeight = MinContentHeight
	}

	return ViewDimensions{
		Width:         totalWidth,
		Height:        totalHeight,
		ContentWidth:  contentWidth,
		ContentHeight: contentHeight,
		SidebarWidth:  sidebarWidth,
		FooterHeight:  footerHeight,
		HeaderHeight:  headerHeight,
	}
}

// IsValidDimensions checks if the provided dimensions are valid for UI rendering
func IsValidDimensions(width, height int) bool {
	return width >= MinContentWidth && height >= MinContentHeight
}