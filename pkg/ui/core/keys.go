package core

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap defines global key bindings that work across all pages
type GlobalKeyMap struct {
	Help                    key.Binding
	Quit                    key.Binding
	Back                    key.Binding
	NextPage                key.Binding
	PrevPage                key.Binding
	ToggleTheme             key.Binding
	DebugScreenshot         key.Binding
	DebugScreenshotRedacted key.Binding
}

// DefaultGlobalKeys provides default global key bindings
var DefaultGlobalKeys = GlobalKeyMap{
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("q/ctrl+c", "quit"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	NextPage: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next page"),
	),
	PrevPage: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev page"),
	),
	ToggleTheme: key.NewBinding(
		key.WithKeys("T"),
		key.WithHelp("T", "toggle theme"),
	),
	DebugScreenshot: key.NewBinding(
		key.WithKeys("f3"),
		key.WithHelp("F3", "save screenshot"),
	),
	DebugScreenshotRedacted: key.NewBinding(
		key.WithKeys("shift+f3"),
		key.WithHelp("Shift+F3", "save redacted screenshot"),
	),
}

// GetAllBindings returns all global key bindings as a slice
func (g GlobalKeyMap) GetAllBindings() []key.Binding {
	return []key.Binding{
		g.Help,
		g.Quit,
		g.Back,
		g.NextPage,
		g.PrevPage,
		g.ToggleTheme,
		g.DebugScreenshot,
		g.DebugScreenshotRedacted,
	}
}