package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up             key.Binding
	Down           key.Binding
	Open           key.Binding
	FilterRepo     key.Binding
	FilterStatus   key.Binding
	FilterWorkflow key.Binding
	ClearFilter    key.Binding
	Quit           key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
	),
	Open: key.NewBinding(
		key.WithKeys("enter"),
	),
	FilterRepo: key.NewBinding(
		key.WithKeys("1"),
	),
	FilterStatus: key.NewBinding(
		key.WithKeys("2"),
	),
	FilterWorkflow: key.NewBinding(
		key.WithKeys("3"),
	),
	ClearFilter: key.NewBinding(
		key.WithKeys("c"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
	),
}
