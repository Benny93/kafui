package core

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap defines global key bindings that work across all pages
type GlobalKeyMap struct {
	Help     key.Binding
	Quit     key.Binding
	Back     key.Binding
	NextPage key.Binding
	PrevPage key.Binding
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
}

// GetAllBindings returns all global key bindings as a slice
func (g GlobalKeyMap) GetAllBindings() []key.Binding {
	return []key.Binding{
		g.Help,
		g.Quit,
		g.Back,
		g.NextPage,
		g.PrevPage,
	}
}