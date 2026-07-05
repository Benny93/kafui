package broker

import "github.com/charmbracelet/bubbles/key"

// pageKeys are the broker detail page key bindings.
type pageKeys struct {
	NextTab key.Binding
	Expand  key.Binding
	Edit    key.Binding
	Move    key.Binding
	Retry   key.Binding
	Search  key.Binding
	Back    key.Binding
}

func defaultKeys() pageKeys {
	return pageKeys{
		NextTab: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
		Expand:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "expand")),
		Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit config")),
		Move:    key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "move replica")),
		Retry:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "retry")),
		Search:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Back:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}
