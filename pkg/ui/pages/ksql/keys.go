package ksql

import "github.com/charmbracelet/bubbles/key"

// overviewKeys are the ksqlDB overview page bindings.
type overviewKeys struct {
	NextTab key.Binding
	Sort    key.Binding
	Query   key.Binding
	Seed    key.Binding
	Retry   key.Binding
	Back    key.Binding
}

func defaultOverviewKeys() overviewKeys {
	return overviewKeys{
		NextTab: key.NewBinding(key.WithKeys("tab", "left", "right"), key.WithHelp("tab", "switch tab")),
		Sort:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
		Query:   key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "query editor")),
		Seed:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "query selected")),
		Retry:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "retry")),
		Back:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

// queryKeys are the ksqlDB query editor page bindings.
//
// Terminals cannot distinguish Cmd+Enter and even Ctrl+Enter is unreliable
// (kitty-protocol dependent), so execute is bound to Ctrl+Enter with a
// documented Ctrl+X fallback that every terminal delivers.
type queryKeys struct {
	Execute   key.Binding
	Clear     key.Binding
	Abort     key.Binding
	ClearRes  key.Binding
	FocusNext key.Binding
	AddProp   key.Binding
	DelProp   key.Binding
	Back      key.Binding
}

func defaultQueryKeys() queryKeys {
	return queryKeys{
		Execute:   key.NewBinding(key.WithKeys("ctrl+@", "ctrl+x"), key.WithHelp("ctrl+x", "execute")),
		Clear:     key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "clear editor")),
		Abort:     key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "abort/back")),
		ClearRes:  key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "clear results")),
		FocusNext: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
		AddProp:   key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "add property")),
		DelProp:   key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "delete property")),
		Back:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}
