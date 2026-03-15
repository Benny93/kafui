package core

// UIState represents the high-level state of the application
type UIState uint8

const (
	// StateNormal indicates normal operation mode
	StateNormal UIState = iota
	// StateHelp indicates help overlay is shown
	StateHelp
	// StateSearch indicates search mode is active
	StateSearch
	// StateModal indicates a modal dialog is open
	StateModal
)

// String returns a human-readable representation of the UI state
func (s UIState) String() string {
	switch s {
	case StateNormal:
		return "normal"
	case StateHelp:
		return "help"
	case StateSearch:
		return "search"
	case StateModal:
		return "modal"
	default:
		return "unknown"
	}
}

// FocusState represents which component currently has focus
type FocusState uint8

const (
	// FocusNone indicates no component has focus
	FocusNone FocusState = iota
	// FocusMain indicates the main content area has focus
	FocusMain
	// FocusSidebar indicates the sidebar has focus
	FocusSidebar
	// FocusSearch indicates the search input has focus
	FocusSearch
	// FocusFooter indicates the footer has focus
	FocusFooter
)

// String returns a human-readable representation of the focus state
func (f FocusState) String() string {
	switch f {
	case FocusNone:
		return "none"
	case FocusMain:
		return "main"
	case FocusSidebar:
		return "sidebar"
	case FocusSearch:
		return "search"
	case FocusFooter:
		return "footer"
	default:
		return "unknown"
	}
}

// LoadingState represents the loading state of a component
type LoadingState uint8

const (
	// LoadingIdle indicates no loading is in progress
	LoadingIdle LoadingState = iota
	// LoadingInitial indicates initial data load
	LoadingInitial
	// LoadingRefresh indicates data refresh
	LoadingRefresh
	// LoadingMore indicates loading additional data
	LoadingMore
)

// ConnectionState represents the connection state to Kafka
type ConnectionState uint8

const (
	// ConnectionUnknown indicates connection state is unknown
	ConnectionUnknown ConnectionState = iota
	// ConnectionConnected indicates active connection
	ConnectionConnected
	// ConnectionDisconnected indicates no connection
	ConnectionDisconnected
	// ConnectionReconnecting indicates reconnection in progress
	ConnectionReconnecting
)

// String returns a human-readable representation of the connection state
func (c ConnectionState) String() string {
	switch c {
	case ConnectionUnknown:
		return "unknown"
	case ConnectionConnected:
		return "connected"
	case ConnectionDisconnected:
		return "disconnected"
	case ConnectionReconnecting:
		return "reconnecting"
	default:
		return "unknown"
	}
}
