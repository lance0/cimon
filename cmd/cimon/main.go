package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lance0/cimon/internal/config"
	"github.com/lance0/cimon/internal/gh"
	"github.com/lance0/cimon/internal/tui"
)

// Build variables (set by goreleaser)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(run())
}

func run() int {
	// Parse CLI flags
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		if err == config.ErrHelp {
			return 0
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Handle --version
	if cfg.Version {
		fmt.Printf("cimon %s (%s) built %s\n", version, commit, date)
		return 0
	}

	// Resolve repo and branch from git
	if err := cfg.Resolve(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Create GitHub client
	client, err := gh.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Create and run TUI
	model := tui.NewModel(cfg, client)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return 2
	}

	// Return exit code based on run status
	if m, ok := finalModel.(tui.Model); ok {
		return m.ExitCode()
	}

	return 0
}
