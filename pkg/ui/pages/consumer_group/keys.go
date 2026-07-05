package consumergroup

import "github.com/charmbracelet/bubbles/key"

// pageKeys are the consumer-group detail page key bindings.
type pageKeys struct {
	Expand      key.Binding
	Filter      key.Binding
	Sort        key.Binding
	Refresh     key.Binding
	AutoRefresh key.Binding
	GotoTopic   key.Binding
	Delete      key.Binding
	DeleteOff   key.Binding
	Reset       key.Binding
	Export      key.Binding
	Retry       key.Binding
	Back        key.Binding
}

func defaultKeys() pageKeys {
	return pageKeys{
		Expand:      key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("enter", "expand")),
		Filter:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter topics")),
		Sort:        key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
		Refresh:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		AutoRefresh: key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "auto-refresh")),
		GotoTopic:   key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "go to topic")),
		Delete:      key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "delete group")),
		DeleteOff:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete offsets")),
		Reset:       key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "reset offsets")),
		Export:      key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "export CSV")),
		Retry:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "retry")),
		Back:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}
