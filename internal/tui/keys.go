package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the key bindings for the TUI
type KeyMap struct {
	Quit      key.Binding
	Refresh   key.Binding
	Watch     key.Binding
	Open      key.Binding
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	Logs      key.Binding
	Search    key.Binding
	NextMatch key.Binding
	PrevMatch key.Binding
	Help      key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Watch: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "watch"),
		),
		Open: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Logs: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "view logs"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		NextMatch: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next match"),
		),
		PrevMatch: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "prev match"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// ShortHelp returns the short help text (shown in footer)
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Refresh, k.Watch, k.Open, k.Quit}
}

// ShortHelpWithEnter returns help text including Enter key for job selection
func (k KeyMap) ShortHelpWithEnter() []key.Binding {
	return []key.Binding{k.Refresh, k.Watch, k.Open, k.Enter, k.Quit}
}

// ShortHelpWithLogs returns help text including enter and logs keys
func (k KeyMap) ShortHelpWithLogs() []key.Binding {
	return []key.Binding{k.Refresh, k.Watch, k.Open, k.Enter, k.Logs, k.Quit}
}

// FullHelp returns the full help text
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Refresh, k.Watch, k.Open},
		{k.Quit, k.Help},
	}
}
