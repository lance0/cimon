package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the key bindings for the TUI
type KeyMap struct {
	Quit         key.Binding
	Refresh      key.Binding
	Watch        key.Binding
	Open         key.Binding
	Up           key.Binding
	Down         key.Binding
	Enter        key.Binding
	Logs         key.Binding
	Search       key.Binding
	NextMatch    key.Binding
	PrevMatch    key.Binding
	NextRun      key.Binding
	PrevRun      key.Binding
	BranchSelect key.Binding
	Filter       key.Binding
	Help         key.Binding
	Workflow     key.Binding
	Artifacts    key.Binding

	// v0.6 Log keys
	LogFilter     key.Binding
	LogSave       key.Binding
	LogHighlight  key.Binding
	LogCompare    key.Binding
	LogMulti      key.Binding
	LogViewToggle key.Binding

	// General UI keys
	Escape key.Binding
	Space  key.Binding
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
		NextRun: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("l/→", "next run"),
		),
		PrevRun: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("h/←", "prev run"),
		),
		BranchSelect: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "select branch"),
		),
		Filter: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "filter status"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Workflow: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "view workflow"),
		),
		Artifacts: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "download artifacts"),
		),

		// v0.6 Log keys
		LogFilter: key.NewBinding(
			key.WithKeys("F"),
			key.WithHelp("F", "filter logs"),
		),
		LogSave: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "save logs"),
		),
		LogHighlight: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "toggle syntax"),
		),
		LogCompare: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "compare runs"),
		),
		LogMulti: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "multi-job"),
		),
		LogViewToggle: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "split/combined"),
		),

		// General UI keys
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Space: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle"),
		),
	}
}

