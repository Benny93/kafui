package connector

import "github.com/charmbracelet/bubbles/key"

// pageKeys are the connector detail page key bindings.
type pageKeys struct {
	NextTab      key.Binding
	Expand       key.Binding
	Pause        key.Binding
	Resume       key.Binding
	Stop         key.Binding
	Restart      key.Binding
	Delete       key.Binding
	ResetOffsets key.Binding
	RestartTask  key.Binding
	RestartAll   key.Binding
	RestartFail  key.Binding
	Edit         key.Binding
	Save         key.Binding
	Retry        key.Binding
	Back         key.Binding
}

func defaultKeys() pageKeys {
	return pageKeys{
		NextTab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
		Expand:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "expand/open")),
		Pause:        key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pause")),
		Resume:       key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "resume")),
		Stop:         key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "stop")),
		Restart:      key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "restart connector")),
		Delete:       key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "delete")),
		ResetOffsets: key.NewBinding(key.WithKeys("z"), key.WithHelp("z", "reset offsets")),
		RestartTask:  key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "restart task")),
		RestartAll:   key.NewBinding(key.WithKeys("T"), key.WithHelp("T", "restart all tasks")),
		RestartFail:  key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "restart failed tasks")),
		Edit:         key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit config")),
		Save:         key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),
		Retry:        key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "retry")),
		Back:         key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}
