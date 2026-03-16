package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
)

type keyMap struct {
	Quit    key.Binding
	Tab     key.Binding
	BackTab key.Binding
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Back    key.Binding
	Search  key.Binding
	Help    key.Binding
	Review  key.Binding
}

// Key bindings use ctrl+ modifiers to avoid conflicts with terminal
// capability responses (OSC 11 contains "/", "r", "g", "b", "?", digits;
// DECRPM contains "[", "?", digits, "$", "y").
var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next tab"),
	),
	BackTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("S-tab", "prev tab"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "expand"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Search: key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("C-f", "search"),
	),
	Help: key.NewBinding(
		key.WithKeys("ctrl+h"),
		key.WithHelp("C-h", "help"),
	),
	Review: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("C-r", "review"),
	),
}

// ShortHelp implements help.KeyMap.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Up, k.Down, k.Enter, k.Back, k.Search, k.Quit}
}

// FullHelp implements help.KeyMap.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab, k.BackTab},
		{k.Up, k.Down, k.Enter, k.Back},
		{k.Search, k.Review, k.Help, k.Quit},
	}
}

// Compile-time check.
var _ help.KeyMap = keyMap{}
