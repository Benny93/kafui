package keys

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap contains all key bindings for the application organized by context
type KeyMap struct {
	// Global key bindings available everywhere
	Global GlobalKeyMap

	// Main page key bindings
	Main MainKeyMap

	// Topic page key bindings
	Topic TopicKeyMap

	// Detail page key bindings
	Detail DetailKeyMap

	// Resource detail page key bindings
	ResourceDetail ResourceDetailKeyMap

	// Search mode key bindings
	Search SearchKeyMap
}

// GlobalKeyMap contains global key bindings available on all pages
type GlobalKeyMap struct {
	Quit  key.Binding
	Help  key.Binding
	Back  key.Binding
	Search key.Binding
}

// MainKeyMap contains key bindings for the main page
type MainKeyMap struct {
	Select         key.Binding
	SwitchResource key.Binding
	Search         key.Binding
	Help           key.Binding
	Quit           key.Binding
	Back           key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k MainKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Search, k.SwitchResource, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k MainKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Search, k.SwitchResource, k.Select},
		{k.Back, k.Help, k.Quit},
	}
}

// TopicKeyMap contains key bindings for the topic page
type TopicKeyMap struct {
	// Navigation
	Select     key.Binding
	Back       key.Binding
	Search     key.Binding
	Help       key.Binding
	Quit       key.Binding
	
	// Consumption control
	Pause      key.Binding
	Refresh    key.Binding
	Retry      key.Binding
	
	// Display options
	Format     key.Binding
	Headers    key.Binding
	Metadata   key.Binding
	
	// Scrolling
	ScrollUp   key.Binding
	ScrollDown key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	GotoStart  key.Binding
	GotoEnd    key.Binding
	
	// Message operations
	CopyKey    key.Binding
	CopyValue  key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the help.KeyMap interface.
func (k TopicKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Search, k.Pause, k.Select, k.Back, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// help.KeyMap interface.
func (k TopicKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Select, k.Search, k.Pause},              // first column
		{k.Refresh, k.Retry, k.Format},             // second column
		{k.ScrollUp, k.ScrollDown, k.PageUp},       // third column
		{k.Back, k.Help, k.Quit},                   // fourth column
	}
}

// DetailKeyMap contains key bindings for the message detail page
type DetailKeyMap struct {
	Back        key.Binding
	Help        key.Binding
	Quit        key.Binding
	Format      key.Binding
	Headers     key.Binding
	Metadata    key.Binding
	Wrap        key.Binding
	ScrollUp    key.Binding
	ScrollDown  key.Binding
	PageUp      key.Binding
	PageDown    key.Binding
	GotoStart   key.Binding
	GotoEnd     key.Binding
	Copy        key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the help.KeyMap interface.
func (k DetailKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Format, k.Copy, k.Back, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// help.KeyMap interface.
func (k DetailKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Format, k.Headers, k.Metadata},     // first column
		{k.Copy, k.ScrollUp, k.ScrollDown},    // second column
		{k.Back, k.Help, k.Quit},              // third column
	}
}

// ResourceDetailKeyMap contains key bindings for the resource detail page
type ResourceDetailKeyMap struct {
	Back       key.Binding
	Help       key.Binding
	Quit       key.Binding
	ScrollUp   key.Binding
	ScrollDown key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	GotoStart  key.Binding
	GotoEnd    key.Binding
	Copy       key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the help.KeyMap interface.
func (k ResourceDetailKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Copy, k.Back, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// help.KeyMap interface.
func (k ResourceDetailKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.ScrollUp, k.ScrollDown, k.PageUp, k.PageDown},  // first column
		{k.GotoStart, k.GotoEnd, k.Copy},                   // second column
		{k.Back, k.Help, k.Quit},                           // third column
	}
}

// SearchKeyMap contains key bindings for search mode
type SearchKeyMap struct {
	Confirm   key.Binding
	Cancel    key.Binding
	Clear     key.Binding
	Navigate  key.Binding
	TabComplete key.Binding
}

// DefaultKeyMap returns the default key bindings for the application
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Global: DefaultGlobalKeyMap(),
		Main:   DefaultMainKeyMap(),
		Topic:  DefaultTopicKeyMap(),
		Detail: DefaultDetailKeyMap(),
		ResourceDetail: DefaultResourceDetailKeyMap(),
		Search: DefaultSearchKeyMap(),
	}
}

// DefaultGlobalKeyMap returns the default global key bindings
func DefaultGlobalKeyMap() GlobalKeyMap {
	return GlobalKeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?", "ctrl+g"),
			key.WithHelp("?", "help"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "q"),
			key.WithHelp("esc", "back"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
	}
}

// DefaultMainKeyMap returns the default main page key bindings
func DefaultMainKeyMap() MainKeyMap {
	return MainKeyMap{
		Select: key.NewBinding(
			key.WithKeys("enter", "l", "right"),
			key.WithHelp("enter", "select"),
		),
		SwitchResource: key.NewBinding(
			key.WithKeys(":", "t"),
			key.WithHelp(":", "switch resource"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
	}
}

// DefaultTopicKeyMap returns the default topic page key bindings
func DefaultTopicKeyMap() TopicKeyMap {
	return TopicKeyMap{
		// Navigation
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "view message"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		// Consumption control
		Pause: key.NewBinding(
			key.WithKeys("p", " "),
			key.WithHelp("p", "pause/resume"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "refresh messages"),
		),
		Retry: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "retry connection"),
		),
		// Display options
		Format: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "format"),
		),
		Headers: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "toggle headers"),
		),
		Metadata: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "toggle metadata"),
		),
		// Scrolling
		ScrollUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓", "scroll down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "b"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", " "),
			key.WithHelp("pgdn", "page down"),
		),
		GotoStart: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g", "go to start"),
		),
		GotoEnd: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G", "go to end"),
		),
		// Message operations
		CopyKey: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy message key"),
		),
		CopyValue: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "copy message value"),
		),
	}
}

// DefaultDetailKeyMap returns the default detail page key bindings
func DefaultDetailKeyMap() DetailKeyMap {
	return DetailKeyMap{
		Back: key.NewBinding(
			key.WithKeys("esc", "q"),
			key.WithHelp("esc", "back"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		Format: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "format"),
		),
		Headers: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "toggle headers"),
		),
		Metadata: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "toggle metadata"),
		),
		Wrap: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "toggle wrap"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓", "scroll down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "b"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", " "),
			key.WithHelp("pgdn", "page down"),
		),
		GotoStart: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g", "go to start"),
		),
		GotoEnd: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G", "go to end"),
		),
		Copy: key.NewBinding(
			key.WithKeys("c", "y"),
			key.WithHelp("c", "copy"),
		),
	}
}

// DefaultResourceDetailKeyMap returns the default resource detail page key bindings
func DefaultResourceDetailKeyMap() ResourceDetailKeyMap {
	return ResourceDetailKeyMap{
		Back: key.NewBinding(
			key.WithKeys("esc", "q"),
			key.WithHelp("esc", "back"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓", "scroll down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "b"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", " "),
			key.WithHelp("pgdn", "page down"),
		),
		GotoStart: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g", "go to start"),
		),
		GotoEnd: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G", "go to end"),
		),
		Copy: key.NewBinding(
			key.WithKeys("c", "y"),
			key.WithHelp("c", "copy"),
		),
	}
}

// DefaultSearchKeyMap returns the default search mode key bindings
func DefaultSearchKeyMap() SearchKeyMap {
	return SearchKeyMap{
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc", "ctrl+c"),
			key.WithHelp("esc", "cancel"),
		),
		Clear: key.NewBinding(
			key.WithKeys("ctrl+u", "ctrl+k"),
			key.WithHelp("ctrl+u", "clear"),
		),
		Navigate: key.NewBinding(
			key.WithKeys("up", "down", "ctrl+n", "ctrl+p"),
			key.WithHelp("↑/↓", "navigate"),
		),
		TabComplete: key.NewBinding(
			key.WithKeys("tab", "shift+tab"),
			key.WithHelp("tab", "complete"),
		),
	}
}

// GetShortHelp returns key bindings for the mini help view
func (km KeyMap) GetShortHelp() []key.Binding {
	return []key.Binding{
		km.Global.Help,
		km.Global.Back,
		km.Global.Quit,
	}
}

// GetFullHelp returns key bindings for the expanded help view
func (km KeyMap) GetFullHelp() [][]key.Binding {
	return [][]key.Binding{
		{km.Global.Help, km.Global.Back, km.Global.Quit},
		{km.Global.Search},
	}
}

// GetMainPageHelp returns help key bindings for the main page
func (km KeyMap) GetMainPageHelp() []key.Binding {
	return []key.Binding{
		km.Main.Search,
		km.Main.SwitchResource,
		km.Main.Select,
		km.Main.Back,
		km.Main.Help,
		km.Main.Quit,
	}
}

// GetTopicPageHelp returns help key bindings for the topic page
func (km KeyMap) GetTopicPageHelp() []key.Binding {
	return []key.Binding{
		km.Topic.Select,
		km.Topic.Search,
		km.Topic.Pause,
		km.Topic.Back,
		km.Topic.Help,
		km.Topic.Quit,
	}
}

// GetDetailPageHelp returns help key bindings for the detail page
func (km KeyMap) GetDetailPageHelp() []key.Binding {
	return []key.Binding{
		km.Detail.Format,
		km.Detail.Headers,
		km.Detail.Copy,
		km.Detail.Back,
		km.Detail.Help,
		km.Detail.Quit,
	}
}
