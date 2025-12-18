package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lance0/cimon/internal/git"
	"github.com/spf13/pflag"
)

// ErrHelp is returned when --help is requested
var ErrHelp = pflag.ErrHelp

// Config holds all runtime configuration for cimon
type Config struct {
	Owner    string
	Repo     string
	Branch   string
	Watch    bool
	Poll     time.Duration
	NoColor  bool
	Plain    bool
	Version  bool
}

// Default values
const (
	DefaultPollInterval = 5 * time.Second
)

var (
	// ErrNoRepo is returned when repo cannot be determined
	ErrNoRepo = errors.New("could not determine repository")

	// ErrNoBranch is returned when branch cannot be determined
	ErrNoBranch = errors.New("could not determine branch")
)

// Parse parses command-line flags and resolves configuration.
// It auto-detects repo and branch from git if not specified.
func Parse(args []string) (*Config, error) {
	cfg := &Config{}

	fs := pflag.NewFlagSet("cimon", pflag.ContinueOnError)

	var repoFlag string
	fs.StringVarP(&repoFlag, "repo", "r", "", "Repository in owner/name format")
	fs.StringVarP(&cfg.Branch, "branch", "b", "", "Branch name")
	fs.BoolVarP(&cfg.Watch, "watch", "w", false, "Watch mode - poll until completion")
	fs.DurationVarP(&cfg.Poll, "poll", "p", DefaultPollInterval, "Poll interval for watch mode")
	fs.BoolVar(&cfg.NoColor, "no-color", false, "Disable color output")
	fs.BoolVar(&cfg.Plain, "plain", false, "Plain text output (no TUI)")
	fs.BoolVarP(&cfg.Version, "version", "v", false, "Show version")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Handle --repo flag
	if repoFlag != "" {
		parts := strings.SplitN(repoFlag, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid repo format %q: expected owner/name", repoFlag)
		}
		cfg.Owner = parts[0]
		cfg.Repo = parts[1]
	}

	return cfg, nil
}

// Resolve fills in missing Owner, Repo, and Branch from git.
// Should be called after Parse.
func (c *Config) Resolve() error {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get working directory: %w", err)
	}

	// Resolve repo if not specified
	if c.Owner == "" || c.Repo == "" {
		info, err := git.GetRepoInfo(cwd)
		if err != nil {
			return fmt.Errorf("%w: %v\nRun inside a git repo or pass --repo owner/name", ErrNoRepo, err)
		}
		c.Owner = info.Owner
		c.Repo = info.Repo
	}

	// Resolve branch if not specified
	if c.Branch == "" {
		branch, err := git.GetBranch(cwd)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrNoBranch, err)
		}
		c.Branch = branch
	}

	return nil
}

// RepoSlug returns the owner/repo format
func (c *Config) RepoSlug() string {
	return c.Owner + "/" + c.Repo
}
